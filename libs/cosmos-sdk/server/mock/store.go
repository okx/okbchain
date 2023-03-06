package mock

import (
	"io"

	store "github.com/okx/okbchain/libs/cosmos-sdk/store/types"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/iavl"
	"github.com/okx/okbchain/libs/tendermint/libs/log"
	dbm "github.com/okx/okbchain/libs/tm-db"
)

var _ sdk.MultiStore = multiStore{}
var _ sdk.CommitMultiStore = multiStore{}

type multiStore struct {
	kv map[sdk.StoreKey]kvStore
}

func (ms multiStore) AppendVersionFilters(filters []store.VersionFilter) {
	panic("not implemented")
}

func (ms multiStore) CleanVersionFilters() {
	panic("not implemented")
}

func (ms multiStore) CacheMultiStore() sdk.CacheMultiStore {
	panic("not implemented")
}

func (ms multiStore) CacheMultiStoreWithVersion(_ int64) (sdk.CacheMultiStore, error) {
	panic("not implemented")
}

func (ms multiStore) AppendCommitFilters(filters []store.StoreFilter) {
	panic("not implemented")
}

func (ms multiStore) CleanCommitFilters() {
	panic("not implemented")
}

func (ms multiStore) AppendPruneFilters(filters []store.StoreFilter) {
	panic("not implemented")
}

func (ms multiStore) CleanPruneFilters() {
	panic("not implemented")
}

func (ms multiStore) CacheWrap() sdk.CacheWrap {
	panic("not implemented")
}

func (ms multiStore) CacheWrapWithTrace(_ io.Writer, _ sdk.TraceContext) sdk.CacheWrap {
	panic("not implemented")
}

func (ms multiStore) TracingEnabled() bool {
	panic("not implemented")
}

func (ms multiStore) SetTracingContext(tc sdk.TraceContext) sdk.MultiStore {
	panic("not implemented")
}

func (ms multiStore) SetTracer(w io.Writer) sdk.MultiStore {
	panic("not implemented")
}

func (ms multiStore) CommitterCommit(*iavl.TreeDelta) (store.CommitID, *iavl.TreeDelta) {
	panic("not implemented")
}

func (ms multiStore) LastCommitID() sdk.CommitID {
	panic("not implemented")
}

func (ms multiStore) LastCommitVersion() int64 {
	panic("not implemented")
}

func (ms multiStore) SetPruning(opts sdk.PruningOptions) {
	panic("not implemented")
}

func (ms multiStore) GetCommitKVStore(key sdk.StoreKey) sdk.CommitKVStore {
	panic("not implemented")
}

func (ms multiStore) GetCommitStore(key sdk.StoreKey) sdk.CommitStore {
	panic("not implemented")
}

func (ms multiStore) MountStoreWithDB(key sdk.StoreKey, typ sdk.StoreType, db dbm.DB) {
	ms.kv[key] = kvStore{store: make(map[string][]byte)}
}

func (ms multiStore) LoadLatestVersion() error {
	return nil
}

func (ms multiStore) LoadLatestVersionAndUpgrade(upgrades *store.StoreUpgrades) error {
	return nil
}

func (ms multiStore) LoadVersionAndUpgrade(ver int64, upgrades *store.StoreUpgrades) error {
	panic("not implemented")
}

func (ms multiStore) LoadVersion(ver int64) error {
	panic("not implemented")
}

func (ms multiStore) GetKVStore(key sdk.StoreKey) sdk.KVStore {
	return ms.kv[key]
}

func (ms multiStore) GetStore(key sdk.StoreKey) sdk.Store {
	panic("not implemented")
}

func (ms multiStore) GetStoreType() sdk.StoreType {
	panic("not implemented")
}

func (ms multiStore) SetInterBlockCache(_ sdk.MultiStorePersistentCache) {
	panic("not implemented")
}

func (ms multiStore) GetDBReadTime() int {
	return 0
}

func (ms multiStore) GetDBWriteCount() int {
	return 0
}

func (ms multiStore) GetDBReadCount() int {
	return 0
}
func (ms multiStore) GetNodeReadCount() int {
	return 0
}

func (ms multiStore) ResetCount() {
}

func (ms multiStore) GetFlatKVReadTime() int {
	return 0
}

func (ms multiStore) GetFlatKVWriteTime() int {
	return 0
}

func (ms multiStore) GetFlatKVReadCount() int {
	return 0
}

func (ms multiStore) GetFlatKVWriteCount() int {
	return 0
}

func (ms multiStore) SetUpgradeVersion(int64) {}

var _ sdk.KVStore = kvStore{}

type kvStore struct {
	store map[string][]byte
}

func (kv kvStore) CacheWrap() sdk.CacheWrap {
	panic("not implemented")
}

func (kv kvStore) CacheWrapWithTrace(w io.Writer, tc sdk.TraceContext) sdk.CacheWrap {
	panic("not implemented")
}

func (kv kvStore) GetStoreType() sdk.StoreType {
	panic("not implemented")
}

func (kv kvStore) Get(key []byte) []byte {
	v, ok := kv.store[string(key)]
	if !ok {
		return nil
	}
	return v
}

func (kv kvStore) Has(key []byte) bool {
	_, ok := kv.store[string(key)]
	return ok
}

func (kv kvStore) Set(key, value []byte) {
	kv.store[string(key)] = value
}

func (kv kvStore) Delete(key []byte) {
	delete(kv.store, string(key))
}

func (kv kvStore) Prefix(prefix []byte) sdk.KVStore {
	panic("not implemented")
}

func (kv kvStore) Gas(meter sdk.GasMeter, config sdk.GasConfig) sdk.KVStore {
	panic("not implmeneted")
}

func (kv kvStore) Iterator(start, end []byte) sdk.Iterator {
	panic("not implemented")
}

func (kv kvStore) ReverseIterator(start, end []byte) sdk.Iterator {
	panic("not implemented")
}

func (kv kvStore) SubspaceIterator(prefix []byte) sdk.Iterator {
	panic("not implemented")
}

func (kv kvStore) ReverseSubspaceIterator(prefix []byte) sdk.Iterator {
	panic("not implemented")
}

func NewCommitMultiStore() sdk.CommitMultiStore {
	return multiStore{kv: make(map[sdk.StoreKey]kvStore)}
}

func (ms multiStore) StopStore() {
	panic("not implemented")
}

func (ms multiStore) SetLogger(log log.Logger) {
	panic("not implemented")
}

func (ms multiStore) GetCommitVersion() (int64, error) {
	panic("not implemented")
}

func (ms multiStore) CommitterCommitMap(inputDeltaMap iavl.TreeDeltaMap) (sdk.CommitID, iavl.TreeDeltaMap) {
	panic("not implemented")
}
