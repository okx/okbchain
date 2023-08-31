package null

import (
	"context"
	"errors"

	"github.com/okx/brczero/libs/tendermint/libs/pubsub/query"
	"github.com/okx/brczero/libs/tendermint/state/txindex"
	"github.com/okx/brczero/libs/tendermint/types"
)

var _ txindex.TxIndexer = (*TxIndex)(nil)

// TxIndex acts as a /dev/null.
type TxIndex struct{}

// Get on a TxIndex is disabled and panics when invoked.
func (txi *TxIndex) Get(hash []byte) (*types.TxResult, error) {
	return nil, errors.New(`indexing is disabled (set 'tx_index = "kv"' in config)`)
}

// AddBatch is a noop and always returns nil.
func (txi *TxIndex) AddBatch(batch *txindex.Batch) error {
	return nil
}

// Index is a noop and always returns nil.
func (txi *TxIndex) Index(result *types.TxResult) error {
	return nil
}

func (txi *TxIndex) Search(ctx context.Context, q *query.Query) ([]*types.TxResult, error) {
	return []*types.TxResult{}, nil
}
