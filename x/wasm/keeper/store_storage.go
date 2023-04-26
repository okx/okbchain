package keeper

import (
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	"github.com/okx/okbchain/libs/cosmos-sdk/types"
	"sync"
)

var (
	gStoreStorageReader     *storeStorageReader
	gStoreStorageReaderOnce sync.Once
)

// storeStorage is a adapter to mpt. it will handle new node.
type storeStorage struct {
	types.KVStore
}

type storeStorageReader struct {
	newStorageCache map[string][]byte
	rwLock          sync.RWMutex
}

func wrapReadKVStore(store types.KVStore) types.KVStore {
	return &storeStorage{
		KVStore: store,
	}
}

func buildKey(addr ethcmn.Address, realKey []byte) string {
	return addr.String() + string(realKey)
}

func (s *storeStorage) Get(key []byte) []byte {
	if addr, realKey, newStorageKey := mpt.IsNewStorageKey(key); newStorageKey {
		return StoreStorageReaderInst().get(addr, realKey)
	}
	return s.KVStore.Get(key)
}

func (s *storeStorage) Set(key, value []byte) {
	s.KVStore.Set(key, value)
	if addr, realKey, newStorageKey := mpt.IsNewStorageKey(key); newStorageKey {
		StoreStorageReaderInst().set(addr, realKey, value)
	}
}

func (s *storeStorage) Delete(key []byte) {
	s.KVStore.Delete(key)
	if addr, realKey, newStorageKey := mpt.IsNewStorageKey(key); newStorageKey {
		StoreStorageReaderInst().delete(addr, realKey)
	}
}

func StoreStorageReaderInst() *storeStorageReader {
	gStoreStorageReaderOnce.Do(
		func() {
			gStoreStorageReader = &storeStorageReader{}
		},
	)

	return gStoreStorageReader
}

func (sr *storeStorageReader) Clear() {
	sr.newStorageCache = nil
}

func (sr *storeStorageReader) set(addr ethcmn.Address, realKey []byte, value []byte) {
	sr.rwLock.Lock()
	if sr.newStorageCache == nil {
		sr.newStorageCache = make(map[string][]byte)
	}
	sr.newStorageCache[buildKey(addr, realKey)] = value
	sr.rwLock.Unlock()
}

func (sr *storeStorageReader) get(addr ethcmn.Address, realKey []byte) []byte {
	sr.rwLock.RLock()
	defer sr.rwLock.RUnlock()
	if sr.newStorageCache != nil {
		return sr.newStorageCache[buildKey(addr, realKey)]
	}

	return nil
}

func (sr *storeStorageReader) delete(addr ethcmn.Address, realKey []byte) {
	sr.rwLock.Lock()
	if sr.newStorageCache != nil {
		delete(sr.newStorageCache, buildKey(addr, realKey))
	}
	sr.rwLock.Unlock()
}
