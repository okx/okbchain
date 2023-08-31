package proxy

import (
	"github.com/okx/brczero/libs/cosmos-sdk/baseapp"
	"github.com/okx/brczero/libs/cosmos-sdk/store/types"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	clientcontext "github.com/okx/brczero/libs/cosmos-sdk/client/context"
	"github.com/okx/brczero/libs/cosmos-sdk/store"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/libs/cosmos-sdk/x/params"
	abci "github.com/okx/brczero/libs/tendermint/abci/types"
	tmlog "github.com/okx/brczero/libs/tendermint/libs/log"
	dbm "github.com/okx/brczero/libs/tm-db"
	evmwatcher "github.com/okx/brczero/x/evm/watcher"
)

var clientCtx clientcontext.CLIContext

func SetCliContext(ctx clientcontext.CLIContext) {
	clientCtx = ctx
}

func MakeContext(storeKey, storageStoreKey sdk.StoreKey) sdk.Context {
	initCommitMultiStore(storeKey, storageStoreKey)
	header := getHeader()
	cms := getCommitMultiStore()

	ctx := sdk.NewContext(cms, header, true, tmlog.NewNopLogger())
	ctx.SetGasMeter(sdk.NewGasMeter(baseapp.SimulationGasLimit))
	return ctx
}

var (
	qOnce      sync.Once
	evmQuerier *evmwatcher.Querier
)

func getHeader() abci.Header {
	qOnce.Do(func() {
		evmQuerier = evmwatcher.NewQuerier()
	})
	timestamp := time.Now()
	latest, _ := evmQuerier.GetLatestBlockNumber()
	hash, e := evmQuerier.GetBlockHashByNumber(latest)
	if e != nil {
		hash = common.HexToHash("0x000000000000000000000000000000")
	}

	block, e := evmQuerier.GetBlockByHash(hash, false)
	if e == nil {
		timestamp = time.Unix(int64(block.Timestamp), 0)
	}

	header := abci.Header{
		LastBlockId: abci.BlockID{
			Hash: hash.Bytes(),
		},
		Height: int64(latest),
		Time:   timestamp,
	}
	return header
}

var (
	cmsOnce           sync.Once
	gCommitMultiStore types.CommitMultiStore
)

func initCommitMultiStore(storeKey, storageStoreKey sdk.StoreKey) sdk.CommitMultiStore {
	cmsOnce.Do(func() {
		db := dbm.NewMemDB()
		cms := store.NewCommitMultiStore(db)
		// authKey := sdk.NewKVStoreKey(auth.StoreKey)
		paramsKey := sdk.NewKVStoreKey(params.StoreKey)
		paramsTKey := sdk.NewTransientStoreKey(params.TStoreKey)
		cms.MountStoreWithDB(storageStoreKey, sdk.StoreTypeMPT, db)
		cms.MountStoreWithDB(paramsKey, sdk.StoreTypeIAVL, db)
		cms.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
		cms.MountStoreWithDB(paramsTKey, sdk.StoreTypeTransient, db)

		err := cms.LoadLatestVersion()
		if err != nil {
			panic(err)
		}
		gCommitMultiStore = cms
	})

	return gCommitMultiStore
}

var storePool = &sync.Pool{
	New: func() interface{} {
		return gCommitMultiStore.CacheMultiStore()
	},
}

func getCommitMultiStore() sdk.CacheMultiStore {
	multiStore := storePool.Get().(sdk.CacheMultiStore)
	multiStore.Clear()

	return multiStore
}

func PutBackStorePool(cms sdk.CacheMultiStore) {
	cms.Clear()
	storePool.Put(cms)
}
