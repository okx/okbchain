package filters

import (
	"context"
	"fmt"
	"math/big"

	"github.com/okx/okbchain/app/rpc/backend"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/bloombits"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/filters"
	rpctypes "github.com/okx/okbchain/app/rpc/types"
	tmtypes "github.com/okx/okbchain/libs/tendermint/types"
	"github.com/spf13/viper"
)

const FlagGetLogsHeightSpan = "logs-height-span"

// Filter can be used to retrieve and filter logs.
type Filter struct {
	backend  Backend
	criteria filters.FilterCriteria
	matcher  *bloombits.Matcher
}

// NewBlockFilter creates a new filter which directly inspects the contents of
// a block to figure out whether it is interesting or not.
func NewBlockFilter(backend Backend, criteria filters.FilterCriteria) *Filter {
	// Create a generic filter and convert it into a block filter
	return newFilter(backend, criteria, nil)
}

// NewRangeFilter creates a new filter which uses a bloom filter on blocks to
// figure out whether a particular block is interesting or not.
func NewRangeFilter(backend Backend, begin, end int64, addresses []common.Address, topics [][]common.Hash) *Filter {
	// Flatten the address and topic filter clauses into a single bloombits filter
	// system. Since the bloombits are not positional, nil topics are permitted,
	// which get flattened into a nil byte slice.
	var filtersBz [][][]byte // nolint: prealloc
	if len(addresses) > 0 {
		filter := make([][]byte, len(addresses))
		for i, address := range addresses {
			filter[i] = address.Bytes()
		}
		filtersBz = append(filtersBz, filter)
	}

	for _, topicList := range topics {
		filter := make([][]byte, len(topicList))
		for i, topic := range topicList {
			filter[i] = topic.Bytes()
		}
		filtersBz = append(filtersBz, filter)
	}

	size, _ := backend.BloomStatus()

	// Create a generic filter and convert it into a range filter
	criteria := filters.FilterCriteria{
		FromBlock: big.NewInt(begin),
		ToBlock:   big.NewInt(end),
		Addresses: addresses,
		Topics:    topics,
	}

	return newFilter(backend, criteria, bloombits.NewMatcher(size, filtersBz))
}

// newFilter returns a new Filter
func newFilter(backend Backend, criteria filters.FilterCriteria, matcher *bloombits.Matcher) *Filter {
	return &Filter{
		backend:  backend,
		criteria: criteria,
		matcher:  matcher,
	}
}

// Logs searches the blockchain for matching log entries, returning all from the
// first block that contains matches, updating the start of the filter accordingly.
func (f *Filter) Logs(ctx context.Context) ([]*ethtypes.Log, error) {
	logs := []*ethtypes.Log{}
	var err error

	// If we're doing singleton block filtering, execute and return
	if f.criteria.BlockHash != nil && *f.criteria.BlockHash != (common.Hash{}) {
		header, err := f.backend.HeaderByHash(*f.criteria.BlockHash)
		if err != nil {
			return nil, err
		}
		if header == nil {
			return nil, fmt.Errorf("unknown block header %s", f.criteria.BlockHash.String())
		}
		return f.blockLogs(header)
	}

	// Figure out the limits of the filter range
	header, err := f.backend.HeaderByNumber(rpctypes.LatestBlockNumber)
	if err != nil {
		return nil, err
	}

	if header == nil || header.Number == nil {
		return nil, nil
	}

	head := header.Number.Int64()
	if f.criteria.FromBlock.Int64() == -1 {
		f.criteria.FromBlock = big.NewInt(head)
	}
	if f.criteria.ToBlock.Int64() == -1 {
		f.criteria.ToBlock = big.NewInt(head)
	}
	if f.criteria.ToBlock.Int64() > head {
		f.criteria.ToBlock = big.NewInt(head)
	}
	if f.criteria.FromBlock.Int64() <= tmtypes.GetStartBlockHeight() ||
		f.criteria.ToBlock.Int64() <= tmtypes.GetStartBlockHeight() {
		return nil, fmt.Errorf("from and to block height must greater than %d", tmtypes.GetStartBlockHeight())
	}

	heightSpan := viper.GetInt64(FlagGetLogsHeightSpan)
	if heightSpan == 0 {
		return nil, fmt.Errorf("the node connected does not support logs filter")
	} else if heightSpan > 0 && f.criteria.ToBlock.Int64()-f.criteria.FromBlock.Int64() > heightSpan {
		return nil, fmt.Errorf("the span between fromBlock and toBlock must be less than or equal to %d", heightSpan)
	}

	begin := f.criteria.FromBlock.Uint64()
	end := f.criteria.ToBlock.Uint64()
	size, sections := f.backend.BloomStatus()
	if indexed := sections*size + uint64(tmtypes.GetStartBlockHeight()); indexed > begin {
		// update from block height
		f.criteria.FromBlock.Sub(f.criteria.FromBlock, big.NewInt(tmtypes.GetStartBlockHeight()))
		if indexed > end {
			logs, err = f.indexedLogs(ctx, end-uint64(tmtypes.GetStartBlockHeight()))
		} else {
			logs, err = f.indexedLogs(ctx, indexed-1-uint64(tmtypes.GetStartBlockHeight()))
		}
		if err != nil {
			return logs, err
		}
		// recover from block height
		f.criteria.FromBlock.Add(f.criteria.FromBlock, big.NewInt(tmtypes.GetStartBlockHeight()))
	}
	rest, err := f.unindexedLogs(ctx, end)
	logs = append(logs, rest...)
	return logs, err
}

// blockLogs returns the logs matching the filter criteria within a single block.
func (f *Filter) blockLogs(header *ethtypes.Header) ([]*ethtypes.Log, error) {
	if !bloomFilter(header.Bloom, f.criteria.Addresses, f.criteria.Topics) {
		return []*ethtypes.Log{}, nil
	}
	height := header.Number.Int64()
	logsList, err := f.backend.GetLogs(height)
	if err != nil {
		return []*ethtypes.Log{}, err
	}

	var unfiltered []*ethtypes.Log // nolint: prealloc
	for _, logs := range logsList {
		unfiltered = append(unfiltered, logs...)
	}
	logs := FilterLogs(unfiltered, nil, nil, f.criteria.Addresses, f.criteria.Topics)
	if len(logs) == 0 {
		return []*ethtypes.Log{}, nil
	}
	return logs, nil
}

// checkMatches checks if the receipts belonging to the given header contain any log events that
// match the filter criteria. This function is called when the bloom filter signals a potential match.
func (f *Filter) checkMatches(height int64) (logs []*ethtypes.Log, err error) {
	// Get the logs of the block
	logsList, err := f.backend.GetLogs(height)
	if err != nil {
		return nil, err
	}
	var unfiltered []*ethtypes.Log
	for _, logs := range logsList {
		unfiltered = append(unfiltered, logs...)
	}
	logs = FilterLogs(unfiltered, nil, nil, f.criteria.Addresses, f.criteria.Topics)
	return logs, nil
}

// indexedLogs returns the logs matching the filter criteria based on the bloom
// bits indexed available locally or via the network.
func (f *Filter) indexedLogs(ctx context.Context, end uint64) ([]*ethtypes.Log, error) {
	// Create a matcher session and request servicing from the backend
	matches := make(chan uint64, 64)

	session, err := f.matcher.Start(ctx, f.criteria.FromBlock.Uint64(), end, matches)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	f.backend.ServiceFilter(ctx, session)

	// Iterate over the matches until exhausted or context closed
	var logs []*ethtypes.Log
	logsLimit := f.backend.LogsLimit()
	bigEnd := big.NewInt(int64(end))
	timeCtx, cancel := context.WithTimeout(context.Background(), f.backend.LogsTimeout())
	defer cancel()
	for {
		select {
		case number, ok := <-matches:
			select {
			case <-timeCtx.Done():
				return nil, backend.ErrTimeout
			default:
				number += uint64(tmtypes.GetStartBlockHeight())
				// Abort if all matches have been fulfilled
				if !ok {
					err := session.Error()
					if err == nil {
						f.criteria.FromBlock = bigEnd.Add(bigEnd, big.NewInt(1))
					}
					return logs, err
				}
				f.criteria.FromBlock = big.NewInt(int64(number)).Add(big.NewInt(int64(number)), big.NewInt(1))

				// Retrieve the suggested block and pull any truly matching logs
				found, err := f.checkMatches(int64(number))
				if err != nil {
					return logs, err
				}
				logs = append(logs, found...)
				// eth_getLogs limitation
				if logsLimit > 0 && len(logs) > logsLimit {
					return nil, LimitError(logsLimit)
				}
			}
		case <-ctx.Done():
			return logs, ctx.Err()
		}
	}
}

// unindexedLogs returns the logs matching the filter criteria based on raw block
// iteration and bloom matching.
func (f *Filter) unindexedLogs(ctx context.Context, end uint64) ([]*ethtypes.Log, error) {
	var logs []*ethtypes.Log
	begin := f.criteria.FromBlock.Int64()
	beginPtr := &begin
	defer f.criteria.FromBlock.SetInt64(*beginPtr)
	logsLimit := f.backend.LogsLimit()
	ctx, cancel := context.WithTimeout(ctx, f.backend.LogsTimeout())
	defer cancel()
	for ; begin <= int64(end); begin++ {
		select {
		case <-ctx.Done():
			return nil, backend.ErrTimeout
		default:
			header, err := f.backend.HeaderByNumber(rpctypes.BlockNumber(begin))
			if header == nil || err != nil {
				return logs, err
			}
			found, err := f.blockLogs(header)
			if err != nil {
				return logs, err
			}
			logs = append(logs, found...)
			// eth_getLogs limitation
			if logsLimit > 0 && len(logs) > logsLimit {
				return nil, LimitError(logsLimit)
			}
		}
	}
	return logs, nil
}
