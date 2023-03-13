//go:build rocksdb
// +build rocksdb

package types

import (
	"github.com/ethereum/go-ethereum/ethdb"
	tmdb "github.com/okx/okbchain/libs/tm-db"
)

var _ ethdb.Batch = (*WrapRocksDBBatch)(nil)

type record struct {
	key   []byte
	value []byte
}

type WrapRocksDBBatch struct {
	db      *tmdb.RocksDB
	records []record
	size    int
}

func NewWrapRocksDBBatch(db *tmdb.RocksDB) *WrapRocksDBBatch {
	return &WrapRocksDBBatch{db: db}
}

func (wrsdbb *WrapRocksDBBatch) Put(key []byte, value []byte) error {
	wrsdbb.records = append(wrsdbb.records, record{
		key:   key,
		value: value,
	})
	wrsdbb.size += len(value)
	return nil
}

func (wrsdbb *WrapRocksDBBatch) Delete(key []byte) error {
	wrsdbb.records = append(wrsdbb.records, record{
		key: key,
	})
	wrsdbb.size += len(key)
	return nil
}

func (wrsdbb *WrapRocksDBBatch) ValueSize() int {
	return wrsdbb.size
}

func (wrsdbb *WrapRocksDBBatch) Write() error {
	batch := tmdb.NewRocksDBBatch(wrsdbb.db)
	for _, rcd := range wrsdbb.records {
		if rcd.value != nil {
			batch.Set(rcd.key, rcd.value)
		} else {
			batch.Delete(rcd.key)
		}
	}

	return batch.Write()
}

// Replay replays the batch contents.
func (wrsdbb *WrapRocksDBBatch) Replay(w ethdb.KeyValueWriter) error {
	var err error
	for _, rcd := range wrsdbb.records {
		if rcd.value != nil {
			err = w.Put(rcd.key, rcd.value)
		} else {
			err = w.Delete(rcd.key)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (wrsdbb *WrapRocksDBBatch) Reset() {
	wrsdbb.records = wrsdbb.records[:0]
}
