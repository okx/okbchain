//go:build rocksdb
// +build rocksdb

package types

import (
	"github.com/ethereum/go-ethereum/ethdb"
	tmdb "github.com/okx/brczero/libs/tm-db"
)

var _ ethdb.Iterator = (*WrapRocksDBIterator)(nil)

type WrapRocksDBIterator struct {
	*tmdb.RocksDBIterator
	key, value []byte
}

func NewWrapRocksDBIterator(db *tmdb.RocksDB, start, end []byte) *WrapRocksDBIterator {
	itr, _ := db.Iterator(start, end)
	return &WrapRocksDBIterator{itr.(*tmdb.RocksDBIterator), nil, nil}
}

func (wrsdi *WrapRocksDBIterator) Key() []byte {
	return wrsdi.key
}

func (wrsdi *WrapRocksDBIterator) Value() []byte {
	return wrsdi.value
}

func (wrsdi *WrapRocksDBIterator) Next() bool {
	if !wrsdi.Valid() {
		return false
	}
	if wrsdi.Valid() {
		k, v := wrsdi.RocksDBIterator.Key(), wrsdi.RocksDBIterator.Value()
		wrsdi.key, wrsdi.value = k, v
	}
	if wrsdi.Error() != nil {
		return false
	}
	wrsdi.RocksDBIterator.Next()
	return true
}

func (wrsdi *WrapRocksDBIterator) Error() error {
	return wrsdi.RocksDBIterator.Error()
}

// Release releases associated resources. Release should always succeed and can
// be called multiple times without causing error.
func (wrsdi *WrapRocksDBIterator) Release() {
	if wrsdi.RocksDBIterator != nil {
		wrsdi.RocksDBIterator.Close()
		wrsdi.RocksDBIterator = nil
	}
}
