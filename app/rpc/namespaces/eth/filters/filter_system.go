package filters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/okx/okbchain/libs/cosmos-sdk/server"
	tmquery "github.com/okx/okbchain/libs/tendermint/libs/pubsub/query"
	rpcclient "github.com/okx/okbchain/libs/tendermint/rpc/client"
	coretypes "github.com/okx/okbchain/libs/tendermint/rpc/core/types"
	tmtypes "github.com/okx/okbchain/libs/tendermint/types"
	"github.com/spf13/viper"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rpc"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"

	rpctypes "github.com/okx/okbchain/app/rpc/types"
	evmtypes "github.com/okx/okbchain/x/evm/types"
)

var (
	txEvents        = tmtypes.QueryForEvent(tmtypes.EventTx).String()
	pendingtxEvents = tmtypes.QueryForEvent(tmtypes.EventPendingTx).String()
	txsEvents       = tmtypes.QueryForEvent(tmtypes.EventTxs).String()
	evmEvents       = tmquery.MustParse(fmt.Sprintf("%s='%s' AND %s.%s='%s'", tmtypes.EventTypeKey, tmtypes.EventTx, sdk.EventTypeMessage, sdk.AttributeKeyModule, evmtypes.ModuleName)).String()
	headerEvents    = tmtypes.QueryForEvent(tmtypes.EventNewBlockHeader).String()
	blockTimeEvents = tmtypes.QueryForEvent(tmtypes.EventBlockTime).String()

	rmPendingTxEvents = tmtypes.QueryForEvent(tmtypes.EventRmPendingTx).String()
)

// EventSystem creates subscriptions, processes events and broadcasts them to the
// subscription which match the subscription criteria using the Tendermint's RPC client.
type EventSystem struct {
	ctx    context.Context
	client rpcclient.Client

	// channel length when subscribing
	channelLength int

	// light client mode
	lightMode bool

	index    filterIndex
	indexMux *sync.RWMutex

	// Channels
	install   chan *Subscription // install filter for event notification
	uninstall chan *Subscription // remove filter for event notification
}

// NewEventSystem creates a new manager that listens for event on the given mux,
// parses and filters them. It uses the all map to retrieve filter changes. The
// work loop holds its own index that is used to forward events to filters.
//
// The returned manager has a loop that needs to be stopped with the Stop function
// or by stopping the given mux.
func NewEventSystem(client rpcclient.Client) *EventSystem {
	index := make(filterIndex)
	for i := filters.UnknownSubscription; i < filters.LastIndexSubscription; i++ {
		index[i] = make(map[rpc.ID]*Subscription)
	}

	es := &EventSystem{
		ctx:           context.Background(),
		client:        client,
		channelLength: viper.GetInt(server.FlagWsSubChannelLength),
		lightMode:     false,
		index:         index,
		indexMux:      new(sync.RWMutex),
		install:       make(chan *Subscription),
		uninstall:     make(chan *Subscription),
	}

	go es.eventLoop()
	return es
}

// WithContext sets a new context to the EventSystem. This is required to set a timeout context when
// a new filter is intantiated.
func (es *EventSystem) WithContext(ctx context.Context) {
	es.ctx = ctx
}

// subscribe performs a new event subscription to a given Tendermint event.
// The subscription creates a unidirectional receive event channel to receive the ResultEvent. By
// default, the subscription timeouts (i.e is canceled) after 5 minutes. This function returns an
// error if the subscription fails (eg: if the identifier is already subscribed) or if the filter
// type is invalid.
func (es *EventSystem) subscribe(sub *Subscription) (*Subscription, context.CancelFunc, error) {
	var (
		err      error
		cancelFn context.CancelFunc
		eventCh  <-chan coretypes.ResultEvent
	)

	es.ctx, cancelFn = context.WithTimeout(context.Background(), deadline)

	switch sub.typ {
	case filters.PendingTransactionsSubscription:
		eventCh, err = es.client.Subscribe(es.ctx, string(sub.id), sub.event, es.channelLength)
	case filters.PendingLogsSubscription, filters.MinedAndPendingLogsSubscription:
		eventCh, err = es.client.Subscribe(es.ctx, string(sub.id), sub.event, es.channelLength)
	case filters.LogsSubscription:
		eventCh, err = es.client.Subscribe(es.ctx, string(sub.id), sub.event, es.channelLength)
	case filters.BlocksSubscription:
		eventCh, err = es.client.Subscribe(es.ctx, string(sub.id), sub.event)
	default:
		err = fmt.Errorf("invalid filter subscription type %d", sub.typ)
	}

	if err != nil {
		sub.err <- err
		return nil, cancelFn, err
	}

	// wrap events in a go routine to prevent blocking
	go func() {
		es.install <- sub
		<-sub.installed
	}()

	sub.eventCh = eventCh
	return sub, cancelFn, nil
}

func (es *EventSystem) SubscribeLogsBatch(crit filters.FilterCriteria) (*Subscription, context.CancelFunc, error) {
	return es.subLogs(crit, txsEvents)
}

// SubscribeLogs creates a subscription that will write all logs matching the
// given criteria to the given logs channel. Default value for the from and to
// block is "latest". If the fromBlock > toBlock an error is returned.
func (es *EventSystem) SubscribeLogs(crit filters.FilterCriteria) (*Subscription, context.CancelFunc, error) {
	return es.subLogs(crit, evmEvents)
}

func (es *EventSystem) subLogs(crit filters.FilterCriteria, event string) (*Subscription, context.CancelFunc, error) {
	var from, to rpc.BlockNumber
	if crit.FromBlock == nil {
		from = rpc.LatestBlockNumber
	} else {
		from = rpc.BlockNumber(crit.FromBlock.Int64())
	}
	if crit.ToBlock == nil {
		to = rpc.LatestBlockNumber
	} else {
		to = rpc.BlockNumber(crit.ToBlock.Int64())
	}

	switch {
	// only interested in pending logs
	case from == rpc.PendingBlockNumber && to == rpc.PendingBlockNumber:
		return es.subscribeLogs(crit, filters.PendingLogsSubscription, event)

	// only interested in new mined logs, mined logs within a specific block range, or
	// logs from a specific block number to new mined blocks
	case (from == rpc.LatestBlockNumber && to == rpc.LatestBlockNumber),
		(from >= 0 && to >= 0 && to >= from):
		return es.subscribeLogs(crit, filters.LogsSubscription, event)

	// interested in mined logs from a specific block number, new logs and pending logs
	case from >= rpc.LatestBlockNumber && (to == rpc.PendingBlockNumber || to == rpc.LatestBlockNumber):
		return es.subscribeLogs(crit, filters.MinedAndPendingLogsSubscription, event)

	default:
		return nil, nil, fmt.Errorf("invalid from and to block combination: from > to (%d > %d)", from, to)
	}
}

// subscribeLogs creates a subscription that will write all logs matching the
// given criteria to the given logs channel.
func (es *EventSystem) subscribeLogs(crit filters.FilterCriteria, t filters.Type, e string) (*Subscription, context.CancelFunc, error) {
	sub := &Subscription{
		id:        rpc.NewID(),
		typ:       t,
		event:     e,
		logsCrit:  crit,
		created:   time.Now().UTC(),
		logs:      make(chan []*ethtypes.Log),
		installed: make(chan struct{}, 1),
		err:       make(chan error, 1),
	}
	return es.subscribe(sub)
}

// SubscribeNewHeads subscribes to new block headers events.
func (es EventSystem) SubscribeNewHeads() (*Subscription, context.CancelFunc, error) {
	sub := &Subscription{
		id:        rpc.NewID(),
		typ:       filters.BlocksSubscription,
		event:     headerEvents,
		created:   time.Now().UTC(),
		headers:   make(chan *ethtypes.Header),
		installed: make(chan struct{}, 1),
		err:       make(chan error, 1),
	}
	return es.subscribe(sub)
}

// SubscribePendingTxs subscribes to new pending transactions events from the mempool.
func (es EventSystem) SubscribePendingTxs() (*Subscription, context.CancelFunc, error) {
	sub := &Subscription{
		id:        rpc.NewID(),
		typ:       filters.PendingTransactionsSubscription,
		event:     pendingtxEvents,
		created:   time.Now().UTC(),
		hashes:    make(chan []common.Hash),
		installed: make(chan struct{}, 1),
		err:       make(chan error, 1),
	}
	return es.subscribe(sub)
}

// SubscribeBlockTime subscribes to the latest block time events
func (es EventSystem) SubscribeBlockTime() (*Subscription, context.CancelFunc, error) {
	sub := &Subscription{
		id:        rpc.NewID(),
		typ:       filters.BlocksSubscription,
		event:     blockTimeEvents,
		created:   time.Now().UTC(),
		installed: make(chan struct{}, 1),
		err:       make(chan error, 1),
	}
	return es.subscribe(sub)
}

// SubscribeRmPendingTx subscribes to the rm pending txs events
func (es EventSystem) SubscribeRmPendingTx() (*Subscription, context.CancelFunc, error) {
	sub := &Subscription{
		id:        rpc.NewID(),
		typ:       filters.LogsSubscription,
		event:     rmPendingTxEvents,
		created:   time.Now().UTC(),
		installed: make(chan struct{}, 1),
		err:       make(chan error, 1),
	}
	return es.subscribe(sub)
}

type filterIndex map[filters.Type]map[rpc.ID]*Subscription

func (es *EventSystem) handleLogs(ev coretypes.ResultEvent) {
	data, _ := ev.Data.(tmtypes.EventDataTx)
	resultData, err := evmtypes.DecodeResultData(data.TxResult.Result.Data)
	if err != nil {
		return
	}

	if len(resultData.Logs) == 0 {
		return
	}
	for _, f := range es.index[filters.LogsSubscription] {
		matchedLogs := FilterLogs(resultData.Logs, f.logsCrit.FromBlock, f.logsCrit.ToBlock, f.logsCrit.Addresses, f.logsCrit.Topics)
		if len(matchedLogs) > 0 {
			f.logs <- matchedLogs
		}
	}
}

func (es *EventSystem) handleTxsEvent(ev coretypes.ResultEvent) {
	data, _ := ev.Data.(tmtypes.EventDataTx)
	for _, f := range es.index[filters.PendingTransactionsSubscription] {
		f.hashes <- []common.Hash{common.BytesToHash(data.Tx.Hash())}
	}
}

func (es *EventSystem) handleChainEvent(ev coretypes.ResultEvent) {
	data, _ := ev.Data.(tmtypes.EventDataNewBlockHeader)
	for _, f := range es.index[filters.BlocksSubscription] {
		f.headers <- rpctypes.EthHeaderFromTendermint(data.Header)
	}
	// TODO: light client
}

// eventLoop (un)installs filters and processes mux events.
func (es *EventSystem) eventLoop() {
	for {
		select {
		case f := <-es.install:
			es.indexMux.Lock()
			if f.typ == filters.MinedAndPendingLogsSubscription {
				// the type are logs and pending logs subscriptions
				es.index[filters.LogsSubscription][f.id] = f
				es.index[filters.PendingLogsSubscription][f.id] = f
			} else {
				es.index[f.typ][f.id] = f
			}
			es.indexMux.Unlock()
			close(f.installed)

		case f := <-es.uninstall:
			es.indexMux.Lock()
			if f.typ == filters.MinedAndPendingLogsSubscription {
				// the type are logs and pending logs subscriptions
				delete(es.index[filters.LogsSubscription], f.id)
				delete(es.index[filters.PendingLogsSubscription], f.id)
			} else {
				delete(es.index[f.typ], f.id)
			}
			es.indexMux.Unlock()
			close(f.err)
		}
	}
	// }()
}
