package websockets

import (
	"fmt"
	"sync"

	"github.com/okx/brczero/libs/tendermint/libs/log"
	coretypes "github.com/okx/brczero/libs/tendermint/rpc/core/types"
	tmtypes "github.com/okx/brczero/libs/tendermint/types"
	"github.com/okx/brczero/x/evm/watcher"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/okx/brczero/libs/cosmos-sdk/client/context"

	rpcfilters "github.com/okx/brczero/app/rpc/namespaces/eth/filters"
	rpctypes "github.com/okx/brczero/app/rpc/types"
	evmtypes "github.com/okx/brczero/x/evm/types"
)

// PubSubAPI is the eth_ prefixed set of APIs in the Web3 JSON-RPC spec
type PubSubAPI struct {
	clientCtx context.CLIContext
	events    *rpcfilters.EventSystem
	filtersMu *sync.RWMutex
	filters   map[rpc.ID]*wsSubscription
	logger    log.Logger
}

// NewAPI creates an instance of the ethereum PubSub API.
func NewAPI(clientCtx context.CLIContext, log log.Logger) *PubSubAPI {
	return &PubSubAPI{
		clientCtx: clientCtx,
		events:    rpcfilters.NewEventSystem(clientCtx.Client),
		filtersMu: new(sync.RWMutex),
		filters:   make(map[rpc.ID]*wsSubscription),
		logger:    log.With("module", "websocket-client"),
	}
}

func (api *PubSubAPI) subscribe(conn *wsConn, params []interface{}) (rpc.ID, error) {
	method, ok := params[0].(string)
	if !ok {
		return "0", fmt.Errorf("invalid parameters")
	}

	switch method {
	case "newHeads":
		// TODO: handle extra params
		return api.subscribeNewHeads(conn)
	case "logs":
		var p interface{}
		if len(params) > 1 {
			p = params[1]
		}

		return api.subscribeLogs(conn, p)
	case "newPendingTransactions":
		var isDetail, ok bool
		if len(params) > 1 {
			isDetail, ok = params[1].(bool)
			if !ok {
				return "0", fmt.Errorf("invalid parameters")
			}
		}
		return api.subscribePendingTransactions(conn, isDetail)
	case "syncing":
		return api.subscribeSyncing(conn)
	case "blockTime":
		return api.subscribeLatestBlockTime(conn)

	default:
		return "0", fmt.Errorf("unsupported method %s", method)
	}
}

func (api *PubSubAPI) unsubscribe(id rpc.ID) bool {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	if api.filters[id] == nil {
		api.logger.Debug("client doesn't exist in filters", "ID", id)
		return false
	}
	if api.filters[id].sub != nil {
		api.filters[id].sub.Unsubscribe(api.events)
	}
	close(api.filters[id].unsubscribed)
	delete(api.filters, id)
	api.logger.Debug("close client channel & delete client from filters", "ID", id)
	return true
}

func (api *PubSubAPI) subscribeNewHeads(conn *wsConn) (rpc.ID, error) {
	sub, _, err := api.events.SubscribeNewHeads()
	if err != nil {
		return "", fmt.Errorf("error creating block filter: %s", err.Error())
	}

	unsubscribed := make(chan struct{})
	api.filtersMu.Lock()
	api.filters[sub.ID()] = &wsSubscription{
		sub:          sub,
		conn:         conn,
		unsubscribed: unsubscribed,
	}
	api.filtersMu.Unlock()

	go func(headersCh <-chan coretypes.ResultEvent, errCh <-chan error) {
		for {
			select {
			case event := <-headersCh:
				data, ok := event.Data.(tmtypes.EventDataNewBlockHeader)
				if !ok {
					api.logger.Error(fmt.Sprintf("invalid data type %T, expected EventDataTx", event.Data), "ID", sub.ID())
					continue
				}
				headerWithBlockHash, err := rpctypes.EthHeaderWithBlockHashFromTendermint(&data.Header)
				if err != nil {
					api.logger.Error("failed to get header with block hash", "error", err)
					continue
				}

				api.filtersMu.RLock()
				if f, found := api.filters[sub.ID()]; found {
					// write to ws conn
					res := &SubscriptionNotification{
						Jsonrpc: "2.0",
						Method:  "eth_subscription",
						Params: &SubscriptionResult{
							Subscription: sub.ID(),
							Result:       headerWithBlockHash,
						},
					}

					err = f.conn.WriteJSON(res)
					if err != nil {
						api.logger.Error("failed to write header", "ID", sub.ID(), "blockNumber", headerWithBlockHash.Number, "error", err)
					} else {
						api.logger.Debug("successfully write header", "ID", sub.ID(), "blockNumber", headerWithBlockHash.Number)
					}
				}
				api.filtersMu.RUnlock()

				if err != nil {
					api.unsubscribe(sub.ID())
				}
			case err := <-errCh:
				if err != nil {
					api.unsubscribe(sub.ID())
					api.logger.Error("websocket recv error, close the conn", "ID", sub.ID(), "error", err)
				}
				return
			case <-unsubscribed:
				api.logger.Debug("NewHeads channel is closed", "ID", sub.ID())
				return
			}
		}
	}(sub.Event(), sub.Err())

	return sub.ID(), nil
}

func (api *PubSubAPI) subscribeLogs(conn *wsConn, extra interface{}) (rpc.ID, error) {
	crit := filters.FilterCriteria{}
	bytx := false // batch logs push by tx

	if extra != nil {
		params, ok := extra.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("invalid criteria")
		}

		if params["address"] != nil {
			address, ok := params["address"].(string)
			addresses, sok := params["address"].([]interface{})
			if !ok && !sok {
				return "", fmt.Errorf("invalid address; must be address or array of addresses")
			}

			if ok {
				if !common.IsHexAddress(address) {
					return "", fmt.Errorf("invalid address")
				}
				crit.Addresses = []common.Address{common.HexToAddress(address)}
			} else if sok {
				crit.Addresses = []common.Address{}
				for _, addr := range addresses {
					address, ok := addr.(string)
					if !ok || !common.IsHexAddress(address) {
						return "", fmt.Errorf("invalid address")
					}

					crit.Addresses = append(crit.Addresses, common.HexToAddress(address))
				}
			}
		}

		if params["topics"] != nil {
			topics, ok := params["topics"].([]interface{})
			if !ok {
				return "", fmt.Errorf("invalid topics")
			}

			topicFilterLists, err := resolveTopicList(topics)
			if err != nil {
				return "", fmt.Errorf("invalid topics")
			}
			crit.Topics = topicFilterLists
		}

		if params["bytx"] != nil {
			b, ok := params["bytx"].(bool)
			if !ok {
				return "", fmt.Errorf("invalid batch; must be true or false")
			}
			bytx = b
		}
	}

	sub, _, err := api.events.SubscribeLogsBatch(crit)
	if err != nil {
		return rpc.ID(""), err
	}

	unsubscribed := make(chan struct{})
	api.filtersMu.Lock()
	api.filters[sub.ID()] = &wsSubscription{
		sub:          sub,
		conn:         conn,
		unsubscribed: unsubscribed,
	}
	api.filtersMu.Unlock()

	go func(ch <-chan coretypes.ResultEvent, errCh <-chan error) {
		quit := false
		for {
			select {
			case event := <-ch:
				go func(event coretypes.ResultEvent) {
					//batch receive txResult
					txs, ok := event.Data.(tmtypes.EventDataTxs)
					if !ok {
						api.logger.Error(fmt.Sprintf("invalid event data %T, expected EventDataTxs", event.Data))
						return
					}

					for _, txResult := range txs.Results {
						if quit {
							return
						}

						//check evm type event
						if !evmtypes.IsEvmEvent(txResult) {
							continue
						}

						//decode txResult data
						var resultData evmtypes.ResultData
						resultData, err = evmtypes.DecodeResultData(txResult.Data)
						if err != nil {
							api.logger.Error("failed to decode result data", "error", err)
							return
						}

						//filter logs
						logs := rpcfilters.FilterLogs(resultData.Logs, crit.FromBlock, crit.ToBlock, crit.Addresses, crit.Topics)
						if len(logs) == 0 {
							continue
						}

						//write log to client by each tx
						api.filtersMu.RLock()
						if f, found := api.filters[sub.ID()]; found {
							// write to ws conn
							res := &SubscriptionNotification{
								Jsonrpc: "2.0",
								Method:  "eth_subscription",
								Params: &SubscriptionResult{
									Subscription: sub.ID(),
								},
							}
							if bytx {
								res.Params.Result = logs
								err = f.conn.WriteJSON(res)
								if err != nil {
									api.logger.Error("failed to batch write logs", "ID", sub.ID(), "height", logs[0].BlockNumber, "txHash", logs[0].TxHash, "error", err)
								}
								api.logger.Info("successfully batch write logs ", "ID", sub.ID(), "height", logs[0].BlockNumber, "txHash", logs[0].TxHash)
							} else {
								for _, singleLog := range logs {
									res.Params.Result = singleLog
									err = f.conn.WriteJSON(res)
									if err != nil {
										api.logger.Error("failed to write log", "ID", sub.ID(), "height", singleLog.BlockNumber, "txHash", singleLog.TxHash, "error", err)
										break
									}
									api.logger.Info("successfully write log", "ID", sub.ID(), "height", singleLog.BlockNumber, "txHash", singleLog.TxHash)
								}
							}
						}
						api.filtersMu.RUnlock()

						if err != nil {
							//unsubscribe and quit current routine
							api.unsubscribe(sub.ID())
							return
						}
					}
				}(event)
			case err := <-errCh:
				quit = true
				if err != nil {
					api.unsubscribe(sub.ID())
					api.logger.Error("websocket recv error, close the conn", "ID", sub.ID(), "error", err)
				}
				return
			case <-unsubscribed:
				quit = true
				api.logger.Debug("Logs channel is closed", "ID", sub.ID())
				return
			}
		}
	}(sub.Event(), sub.Err())

	return sub.ID(), nil
}

func resolveTopicList(params []interface{}) ([][]common.Hash, error) {
	topicFilterLists := make([][]common.Hash, len(params))
	for i, param := range params { // eg: ["0xddf252......f523b3ef", null, ["0x000000......32fea9e4", "0x000000......ab14dc5d"]]
		if param == nil {
			// 1.1 if the topic is null
			topicFilterLists[i] = nil
		} else {
			// 2.1 judge if the param is the type of string or not
			topicStr, ok := param.(string)
			// 2.1 judge if the param is the type of string slice or not
			topicSlices, sok := param.([]interface{})
			if !ok && !sok {
				// if both judgement are false, return invalid topics
				return topicFilterLists, fmt.Errorf("invalid topics")
			}

			if ok {
				// 2.2 This is string
				// 2.3 judge the topic is a valid hex hash or not
				if !IsHexHash(topicStr) {
					return topicFilterLists, fmt.Errorf("invalid topics")
				}
				// 2.4 add this topic to topic-hash-lists
				topicHash := common.HexToHash(topicStr)
				topicFilterLists[i] = []common.Hash{topicHash}
			} else if sok {
				// 2.2 This is slice of string
				topicHashes := make([]common.Hash, len(topicSlices))
				for n, topicStr := range topicSlices {
					//2.3 judge every topic
					topicHash, ok := topicStr.(string)
					if !ok || !IsHexHash(topicHash) {
						return topicFilterLists, fmt.Errorf("invalid topics")
					}
					topicHashes[n] = common.HexToHash(topicHash)
				}
				// 2.4 add this topic slice to topic-hash-lists
				topicFilterLists[i] = topicHashes
			}
		}
	}
	return topicFilterLists, nil
}

func IsHexHash(s string) bool {
	if has0xPrefix(s) {
		s = s[2:]
	}
	return len(s) == 2*common.HashLength && isHex(s)
}

// has0xPrefix validates str begins with '0x' or '0X'.
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

// isHexCharacter returns bool of c being a valid hexadecimal.
func isHexCharacter(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// isHex validates whether each byte is valid hexadecimal string.
func isHex(str string) bool {
	if len(str)%2 != 0 {
		return false
	}
	for _, c := range []byte(str) {
		if !isHexCharacter(c) {
			return false
		}
	}
	return true
}

func (api *PubSubAPI) subscribePendingTransactions(conn *wsConn, isDetail bool) (rpc.ID, error) {
	sub, _, err := api.events.SubscribePendingTxs()
	if err != nil {
		return "", fmt.Errorf("error creating block filter: %s", err.Error())
	}

	unsubscribed := make(chan struct{})
	api.filtersMu.Lock()
	api.filters[sub.ID()] = &wsSubscription{
		sub:          sub,
		conn:         conn,
		unsubscribed: unsubscribed,
	}
	api.filtersMu.Unlock()

	go func(txsCh <-chan coretypes.ResultEvent, errCh <-chan error) {
		for {
			select {
			case ev := <-txsCh:
				data, ok := ev.Data.(tmtypes.EventDataTx)
				if !ok {
					api.logger.Error(fmt.Sprintf("invalid data type %T, expected EventDataTx", ev.Data), "ID", sub.ID())
					continue
				}
				txHash := common.BytesToHash(data.Tx.Hash())
				var res interface{} = txHash
				if isDetail {
					ethTx, err := rpctypes.RawTxToEthTx(api.clientCtx, data.Tx, data.Height)
					if err != nil {
						api.logger.Error("failed to decode raw tx to eth tx", "hash", txHash.String(), "error", err)
						continue
					}

					tx, err := watcher.NewTransaction(ethTx, txHash, common.Hash{}, uint64(data.Height), uint64(data.Index))
					if err != nil {
						api.logger.Error("failed to new transaction", "hash", txHash.String(), "error", err)
						continue
					}
					res = tx
				}
				api.filtersMu.RLock()
				if f, found := api.filters[sub.ID()]; found {
					// write to ws conn
					res := &SubscriptionNotification{
						Jsonrpc: "2.0",
						Method:  "eth_subscription",
						Params: &SubscriptionResult{
							Subscription: sub.ID(),
							Result:       res,
						},
					}

					err = f.conn.WriteJSON(res)
					if err != nil {
						api.logger.Error("failed to write pending tx", "ID", sub.ID(), "error", err)
					} else {
						api.logger.Info("successfully write pending tx", "ID", sub.ID(), "txHash", txHash)
					}
				}
				api.filtersMu.RUnlock()

				if err != nil {
					api.unsubscribe(sub.ID())
				}
			case err := <-errCh:
				if err != nil {
					api.unsubscribe(sub.ID())
					api.logger.Error("websocket recv error, close the conn", "ID", sub.ID(), "error", err)
				}
				return
			case <-unsubscribed:
				api.logger.Debug("PendingTransactions channel is closed", "ID", sub.ID())
				return
			}
		}
	}(sub.Event(), sub.Err())

	return sub.ID(), nil
}

func (api *PubSubAPI) subscribeSyncing(conn *wsConn) (rpc.ID, error) {
	sub, _, err := api.events.SubscribeNewHeads()
	if err != nil {
		return "", fmt.Errorf("error creating block filter: %s", err.Error())
	}

	unsubscribed := make(chan struct{})
	api.filtersMu.Lock()
	api.filters[sub.ID()] = &wsSubscription{
		sub:          sub,
		conn:         conn,
		unsubscribed: unsubscribed,
	}
	api.filtersMu.Unlock()

	status, err := api.clientCtx.Client.Status()
	if err != nil {
		return "", fmt.Errorf("error get sync status: %s", err.Error())
	}
	startingBlock := hexutil.Uint64(status.SyncInfo.EarliestBlockHeight)
	highestBlock := hexutil.Uint64(0)

	var result interface{}

	go func(headersCh <-chan coretypes.ResultEvent, errCh <-chan error) {
		for {
			select {
			case <-headersCh:

				newStatus, err := api.clientCtx.Client.Status()
				if err != nil {
					api.logger.Error(fmt.Sprintf("error get sync status: %s", err.Error()))
					continue
				}

				if !newStatus.SyncInfo.CatchingUp {
					result = false
				} else {
					result = map[string]interface{}{
						"startingBlock": startingBlock,
						"currentBlock":  hexutil.Uint64(newStatus.SyncInfo.LatestBlockHeight),
						"highestBlock":  highestBlock,
					}
				}

				api.filtersMu.RLock()
				if f, found := api.filters[sub.ID()]; found {
					// write to ws conn
					res := &SubscriptionNotification{
						Jsonrpc: "2.0",
						Method:  "eth_subscription",
						Params: &SubscriptionResult{
							Subscription: sub.ID(),
							Result:       result,
						},
					}

					err = f.conn.WriteJSON(res)
					if err != nil {
						api.logger.Error("failed to write syncing status", "ID", sub.ID(), "error", err)
					} else {
						api.logger.Debug("successfully write syncing status", "ID", sub.ID())
					}
				}
				api.filtersMu.RUnlock()

				if err != nil {
					api.unsubscribe(sub.ID())
				}

			case err := <-errCh:
				if err != nil {
					api.unsubscribe(sub.ID())
					api.logger.Error("websocket recv error, close the conn", "ID", sub.ID(), "error", err)
				}
				return
			case <-unsubscribed:
				api.logger.Debug("Syncing channel is closed", "ID", sub.ID())
				return
			}
		}
	}(sub.Event(), sub.Err())

	return sub.ID(), nil
}

func (api *PubSubAPI) subscribeLatestBlockTime(conn *wsConn) (rpc.ID, error) {
	sub, _, err := api.events.SubscribeBlockTime()
	if err != nil {
		return "", fmt.Errorf("error creating block filter: %s", err.Error())
	}

	unsubscribed := make(chan struct{})
	api.filtersMu.Lock()
	api.filters[sub.ID()] = &wsSubscription{
		sub:          sub,
		conn:         conn,
		unsubscribed: unsubscribed,
	}
	api.filtersMu.Unlock()

	go func(txsCh <-chan coretypes.ResultEvent, errCh <-chan error) {
		for {
			select {
			case ev := <-txsCh:
				result, ok := ev.Data.(tmtypes.EventDataBlockTime)
				if !ok {
					api.logger.Error(fmt.Sprintf("invalid data type %T, expected EventDataTx", ev.Data), "ID", sub.ID())
					continue
				}

				api.filtersMu.RLock()
				if f, found := api.filters[sub.ID()]; found {
					// write to ws conn
					res := &SubscriptionNotification{
						Jsonrpc: "2.0",
						Method:  "eth_subscription",
						Params: &SubscriptionResult{
							Subscription: sub.ID(),
							Result:       result,
						},
					}

					err = f.conn.WriteJSON(res)
					if err != nil {
						api.logger.Error("failed to write latest blocktime", "ID", sub.ID(), "error", err)
					} else {
						api.logger.Debug("successfully write latest blocktime", "ID", sub.ID(), "data", result)
					}
				}
				api.filtersMu.RUnlock()

				if err != nil {
					api.unsubscribe(sub.ID())
				}
			case err := <-errCh:
				if err != nil {
					api.unsubscribe(sub.ID())
					api.logger.Error("websocket recv error, close the conn", "ID", sub.ID(), "error", err)
				}
				return
			case <-unsubscribed:
				api.logger.Debug("BlockTime channel is closed", "ID", sub.ID())
				return
			}
		}
	}(sub.Event(), sub.Err())

	return sub.ID(), nil
}
