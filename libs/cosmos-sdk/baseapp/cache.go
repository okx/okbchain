package baseapp

import (
	"sync"

	"github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/tendermint/go-amino"
)

type blockDataCache struct {
	txLock sync.RWMutex
	txs    map[string]types.Tx
}

func NewBlockDataCache() *blockDataCache {
	return &blockDataCache{
		txs: make(map[string]types.Tx),
	}
}

func (cache *blockDataCache) SetTx(txRaw []byte, tx types.Tx) {
	if cache == nil {
		return
	}
	cache.txLock.Lock()
	// txRaw should be immutable, so no need to copy it
	cache.txs[amino.BytesToStr(txRaw)] = tx
	cache.txLock.Unlock()
}

func (cache *blockDataCache) GetTx(txRaw []byte) (tx types.Tx, ok bool) {
	if cache == nil {
		return
	}
	cache.txLock.RLock()
	tx, ok = cache.txs[amino.BytesToStr(txRaw)]
	cache.txLock.RUnlock()
	return
}

func (cache *blockDataCache) Clear() {
	if cache == nil {
		return
	}
	cache.txLock.Lock()
	for k := range cache.txs {
		delete(cache.txs, k)
	}
	cache.txLock.Unlock()
}
