package types

import (
	"container/list"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/cosmos/gorocksdb"
	"github.com/ethereum/go-ethereum/ethdb"
	tmdb "github.com/okx/okbchain/libs/tm-db"
)

type BatchCache struct {
	batchList  *list.List
	batchCache map[int64]*list.Element
	//batchInUse map[int64]bool
	maxSize int

	lock sync.Mutex
}

func NewBatchCache(maxSize int) *BatchCache {
	return &BatchCache{
		batchList:  list.New(),
		batchCache: make(map[int64]*list.Element, maxSize),
		//batchInUse: make(map[int64]bool),
		maxSize: maxSize,
	}
}

func (bc *BatchCache) PushBack(batch *WrapRocksDBBatch) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	ele := bc.batchList.PushBack(batch)
	bc.batchCache[batch.GetID()] = ele
}

func (bc *BatchCache) TryPopFront() *WrapRocksDBBatch {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	if bc.batchList.Len() > bc.maxSize {
		deathEle := bc.batchList.Front()
		deathBatch := deathEle.Value.(*WrapRocksDBBatch)
		//if bc.batchInUse[deathBatch.GetID()] {
		//	return nil
		//}
		bc.batchList.Remove(deathEle)

		delete(bc.batchCache, deathBatch.GetID())

		return deathBatch
	}

	return nil
}

func (bc *BatchCache) MoveToBack(id int64) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	if ele, ok := bc.batchCache[id]; ok {
		bc.batchList.MoveToBack(ele)
	}
	//bc.batchInUse[id] = false
}

var (
	gBatchCache *BatchCache
	batchIdSeed int64

	initRocksdbBatchOnce sync.Once
)

func InstanceBatchCache() *BatchCache {
	initRocksdbBatchOnce.Do(func() {
		gBatchCache = NewBatchCache(int(TrieRocksdbBatchSize))
		fmt.Println("TrieRocksdbBatchSize", TrieRocksdbBatchSize)
	})

	return gBatchCache
}

var _ ethdb.Batch = (*WrapRocksDBBatch)(nil)

type WrapRocksDBBatch struct {
	*tmdb.RocksDBBatch
	id int64
}

func NewWrapRocksDBBatch(db *tmdb.RocksDB) *WrapRocksDBBatch {
	sed := atomic.LoadInt64(&batchIdSeed)
	batch := &WrapRocksDBBatch{tmdb.NewRocksDBBatch(db), sed}
	atomic.AddInt64(&batchIdSeed, 1)

	batchCache := InstanceBatchCache()
	batchCache.PushBack(batch)
	if deathBatch := batchCache.TryPopFront(); deathBatch != nil {
		deathBatch.Close()
	}

	return batch
}

func (wrsdbb *WrapRocksDBBatch) Put(key []byte, value []byte) error {

	wrsdbb.Set(key, value)
	InstanceBatchCache().MoveToBack(wrsdbb.GetID())
	return nil
}

func (wrsdbb *WrapRocksDBBatch) Delete(key []byte) error {

	wrsdbb.RocksDBBatch.Delete(key)
	InstanceBatchCache().MoveToBack(wrsdbb.GetID())
	return nil
}

func (wrsdbb *WrapRocksDBBatch) ValueSize() int {
	return wrsdbb.Size()
}

func (wrsdbb *WrapRocksDBBatch) Write() error {

	err := wrsdbb.RocksDBBatch.WriteWithoutClose()
	InstanceBatchCache().MoveToBack(wrsdbb.GetID())
	return err

}

// Replay replays the batch contents.
func (wrsdbb *WrapRocksDBBatch) Replay(w ethdb.KeyValueWriter) error {
	rp := &replayer{writer: w}

	itr := wrsdbb.NewIterator()
	for itr.Next() {
		rcd := itr.Record()

		switch rcd.Type {
		case gorocksdb.WriteBatchValueRecord:
			rp.Put(rcd.Key, rcd.Value)
		case gorocksdb.WriteBatchDeletionRecord:
			rp.Delete(rcd.Key)
		}
	}
	InstanceBatchCache().MoveToBack(wrsdbb.GetID())
	return nil
}

func (wrsdbb *WrapRocksDBBatch) GetID() int64 {
	return wrsdbb.id
}
