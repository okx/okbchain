package cachemulti

import (
	"fmt"
	dbm "github.com/okx/okbchain/libs/tm-db"
	"io"
	"sync"

	"github.com/okx/okbchain/libs/cosmos-sdk/store/cachekv"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/dbadapter"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/types"
)

//----------------------------------------
// Store

// Store holds many cache-wrapped stores.
// Implements MultiStore.
// NOTE: a Store (and MultiStores in general) should never expose the
// keys for the substores.
type Store struct {
	db     types.CacheKVStore
	stores map[types.StoreKey]types.CacheWrap
	keys   map[string]types.StoreKey

	traceWriter  io.Writer
	traceContext types.TraceContext
}

var _ types.CacheMultiStore = Store{}

// NewFromKVStore creates a new Store object from a mapping of store keys to
// CacheWrapper objects and a KVStore as the database. Each CacheWrapper store
// is cache-wrapped.
func NewFromKVStore(
	store types.KVStore, stores map[types.StoreKey]types.CacheWrapper,
	keys map[string]types.StoreKey, traceWriter io.Writer, traceContext types.TraceContext,
) Store {
	cms := Store{
		db:           cachekv.NewStore(store),
		stores:       make(map[types.StoreKey]types.CacheWrap, len(stores)),
		keys:         keys,
		traceWriter:  traceWriter,
		traceContext: traceContext,
	}

	for key, store := range stores {
		if cms.TracingEnabled() {
			cms.stores[key] = store.CacheWrapWithTrace(cms.traceWriter, cms.traceContext)
		} else {
			cms.stores[key] = store.CacheWrap()
		}
	}

	return cms
}

// newFromKVStore creates a new Store object from a mapping of store keys to
// CacheWrap objects and a KVStore as the database. Each CacheWrapper store
// is cache-wrapped.
func newFromKVStore(
	store types.KVStore, stores map[types.StoreKey]types.CacheWrap,
	keys map[string]types.StoreKey, traceWriter io.Writer, traceContext types.TraceContext,
) Store {
	cms := Store{
		db:           cachekv.NewStore(store),
		stores:       make(map[types.StoreKey]types.CacheWrap, len(stores)),
		keys:         keys,
		traceWriter:  traceWriter,
		traceContext: traceContext,
	}

	for key, store := range stores {
		if cms.TracingEnabled() {
			cms.stores[key] = store.CacheWrapWithTrace(cms.traceWriter, cms.traceContext)
		} else {
			cms.stores[key] = store.CacheWrap()
		}
	}

	return cms
}

// NewStore creates a new Store object from a mapping of store keys to
// CacheWrapper objects. Each CacheWrapper store is cache-wrapped.
func NewStore(
	db dbm.DB, stores map[types.StoreKey]types.CacheWrapper, keys map[string]types.StoreKey,
	traceWriter io.Writer, traceContext types.TraceContext,
) Store {

	return NewFromKVStore(dbadapter.Store{DB: db}, stores, keys, traceWriter, traceContext)
}

func newCacheMultiStoreFromCMS(cms Store) Store {
	return newFromKVStore(cms.db, cms.stores, nil, cms.traceWriter, cms.traceContext)
}

func (cms Store) Reset(ms types.MultiStore) bool {
	switch rms := ms.(type) {
	case Store:
		cms.reset(rms)
		return true
	default:
		return false
	}
}

var keysPool = &sync.Pool{
	New: func() interface{} {
		return make(map[types.StoreKey]struct{})
	},
}

func (cms Store) reset(ms Store) {
	cms.db.(*cachekv.Store).Reset(ms.db)
	cms.traceWriter = ms.traceWriter
	cms.traceContext = ms.traceContext
	cms.keys = ms.keys

	keysMap := keysPool.Get().(map[types.StoreKey]struct{})
	defer keysPool.Put(keysMap)

	for k := range keysMap {
		delete(keysMap, k)
	}
	for k := range ms.stores {
		keysMap[k] = struct{}{}
	}

	for k := range keysMap {
		msstore := ms.stores[k]
		if store, ok := cms.stores[k]; ok {
			if cstore, ok := store.(*cachekv.Store); ok {
				msKvstore, ok := msstore.(types.KVStore)
				if ok {
					cstore.Reset(msKvstore)
				} else {
					cms.stores[k] = msstore.CacheWrap()
				}
			} else {
				cms.stores[k] = msstore.CacheWrap()
			}
		} else {
			cms.stores[k] = msstore.CacheWrap()
		}
	}

	for k := range cms.stores {
		if _, ok := keysMap[k]; !ok {
			delete(cms.stores, k)
		}
	}
}

// SetTracer sets the tracer for the MultiStore that the underlying
// stores will utilize to trace operations. A MultiStore is returned.
func (cms Store) SetTracer(w io.Writer) types.MultiStore {
	cms.traceWriter = w
	return cms
}

// SetTracingContext updates the tracing context for the MultiStore by merging
// the given context with the existing context by key. Any existing keys will
// be overwritten. It is implied that the caller should update the context when
// necessary between tracing operations. It returns a modified MultiStore.
func (cms Store) SetTracingContext(tc types.TraceContext) types.MultiStore {
	if cms.traceContext != nil {
		for k, v := range tc {
			cms.traceContext[k] = v
		}
	} else {
		cms.traceContext = tc
	}

	return cms
}

// TracingEnabled returns if tracing is enabled for the MultiStore.
func (cms Store) TracingEnabled() bool {
	return cms.traceWriter != nil
}

// GetStoreType returns the type of the store.
func (cms Store) GetStoreType() types.StoreType {
	return types.StoreTypeMulti
}

// Write calls Write on each underlying store.
func (cms Store) Write() {
	cms.db.Write()
	for _, store := range cms.stores {
		store.Write()
	}
}

func (cms Store) IteratorCache(isdirty bool, cb func(key string, value []byte, isDirty bool, isDelete bool, storeKey types.StoreKey) bool, sKey types.StoreKey) bool {
	for key, store := range cms.stores {
		if !store.IteratorCache(isdirty, cb, key) {
			return false
		}
	}
	return true
}

func (cms Store) GetRWSet(mp types.MsRWSet) {
	for key, store := range cms.stores {
		if _, ok := mp[key]; !ok {
			mp[key] = types.NewCacheKvRWSet()
		}
		store.(*cachekv.Store).CopyRWSet(mp[key])
	}
}

// Implements CacheWrapper.
func (cms Store) CacheWrap() types.CacheWrap {
	return cms.CacheMultiStore().(types.CacheWrap)
}

// CacheWrapWithTrace implements the CacheWrapper interface.
func (cms Store) CacheWrapWithTrace(_ io.Writer, _ types.TraceContext) types.CacheWrap {
	return cms.CacheWrap()
}

// Implements MultiStore.
func (cms Store) CacheMultiStore() types.CacheMultiStore {
	return newCacheMultiStoreFromCMS(cms)
}

// CacheMultiStoreWithVersion implements the MultiStore interface. It will panic
// as an already cached multi-store cannot load previous versions.
//
// TODO: The store implementation can possibly be modified to support this as it
// seems safe to load previous versions (heights).
func (cms Store) CacheMultiStoreWithVersion(_ int64) (types.CacheMultiStore, error) {
	panic("cannot cache-wrap cached multi-store with a version")
}

// GetStore returns an underlying Store by key.
func (cms Store) GetStore(key types.StoreKey) types.Store {
	s := cms.stores[key]
	if key == nil || s == nil {
		panic(fmt.Sprintf("kv store with key %v has not been registered in stores", key))
	}
	return s.(types.Store)
}

// GetKVStore returns an underlying KVStore by key.
func (cms Store) GetKVStore(key types.StoreKey) types.KVStore {
	store := cms.stores[key]
	if key == nil || store == nil {
		panic(fmt.Sprintf("kv store with key %v has not been registered in stores", key))
	}

	return store.(types.KVStore)
}

func (cms Store) Clear() {
	cms.db.Clear()
	for _, store := range cms.stores {
		store.Clear()
	}
}

func (cms Store) DisableCacheReadList() {
	for _, store := range cms.stores {
		store.DisableCacheReadList()
	}
}
