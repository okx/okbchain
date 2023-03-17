//go:build rocksdb
// +build rocksdb

package types

import (
	"runtime"
	"sync"

	"github.com/cosmos/gorocksdb"
	"github.com/ethereum/go-ethereum/ethdb"
	tmdb "github.com/okx/okbchain/libs/tm-db"
)

var _ ethdb.Batch = (*WrapRocksDBBatch)(nil)

var batchPool = &sync.Pool{
	New: func() interface{} {
		return (*tmdb.RocksDBBatch)(nil)
	},
}

type WrapRocksDBBatch struct {
	batch     *tmdb.RocksDBBatch
	valueSize int
}

func NewWrapRocksDBBatch(db *tmdb.RocksDB) *WrapRocksDBBatch {
	batch := batchPool.Get().(*tmdb.RocksDBBatch)
	if batch == nil {
		batch = tmdb.NewRocksDBBatch(db)
	} else {
		batch.Reset()
	}

	//batch := tmdb.NewRocksDBBatch(db)
	runtime.SetFinalizer(batch, func(b *tmdb.RocksDBBatch) {
		if b == nil {
			return
		}
		batchPool.Put(b)
		runtime.SetFinalizer(b, nil)
	})
	return &WrapRocksDBBatch{
		batch: batch,
	}
}

func (wrb *WrapRocksDBBatch) Put(key []byte, value []byte) error {
	wrb.batch.Set(key, value)
	wrb.valueSize += len(key) + len(value)
	return nil
}

func (wrb *WrapRocksDBBatch) Delete(key []byte) error {
	wrb.batch.Delete(key)
	wrb.valueSize += len(key)
	return nil
}

func (wrb *WrapRocksDBBatch) ValueSize() int {
	return wrb.batch.Size()
	//return wrb.valueSize
}

func (wrb *WrapRocksDBBatch) Write() error {
	return wrb.batch.WriteWithoutClose()
}

// Replay replays the batch contents.
func (wrb *WrapRocksDBBatch) Replay(w ethdb.KeyValueWriter) error {
	iter := wrb.batch.NewIterator()
	for iter.Next() {
		switch rcd := iter.Record(); rcd.Type {
		case gorocksdb.WriteBatchValueRecord:
			if err := w.Put(rcd.Key, rcd.Value); err != nil {
				return err
			}
		case gorocksdb.WriteBatchDeletionRecord:
			if err := w.Delete(rcd.Key); err != nil {
				return err
			}
		}
	}
	return nil
}

func (wrb *WrapRocksDBBatch) Reset() {
	wrb.batch.Reset()
}
