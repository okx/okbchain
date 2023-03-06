package simulation

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/okx/okbchain/libs/cosmos-sdk/store"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/params"
	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	tmlog "github.com/okx/okbchain/libs/tendermint/libs/log"
	dbm "github.com/okx/okbchain/libs/tm-db"
	"github.com/okx/okbchain/x/evm"
	evmtypes "github.com/okx/okbchain/x/evm/types"
	"github.com/okx/okbchain/x/evm/watcher"
)

type EvmFactory struct {
	ChainId        string
	WrappedQuerier *watcher.Querier
	storeKey       *sdk.KVStoreKey
	paramsKey      *sdk.KVStoreKey
	paramsTKey     *sdk.TransientStoreKey
	cms            sdk.CommitMultiStore
	storePool      sync.Pool
}

func NewEvmFactory(chainId string, q *watcher.Querier) EvmFactory {
	ef := EvmFactory{ChainId: chainId, WrappedQuerier: q, storeKey: sdk.NewKVStoreKey(evm.StoreKey)}
	ef.cms, ef.paramsKey, ef.paramsTKey = initCommitMultiStore(ef.storeKey)
	ef.storePool = sync.Pool{
		New: func() interface{} {
			return ef.cms.CacheMultiStore()
		},
	}
	return ef
}

func initCommitMultiStore(storeKey *sdk.KVStoreKey) (sdk.CommitMultiStore, *sdk.KVStoreKey, *sdk.TransientStoreKey) {
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db)
	authKey := sdk.NewKVStoreKey(auth.StoreKey)
	paramsKey := sdk.NewKVStoreKey(params.StoreKey)
	paramsTKey := sdk.NewTransientStoreKey(params.TStoreKey)
	cms.MountStoreWithDB(authKey, sdk.StoreTypeIAVL, db)
	cms.MountStoreWithDB(paramsKey, sdk.StoreTypeIAVL, db)
	cms.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	cms.MountStoreWithDB(paramsTKey, sdk.StoreTypeTransient, db)
	cms.LoadLatestVersion()
	return cms, paramsKey, paramsTKey
}

func (ef *EvmFactory) PutBackStorePool(multiStore sdk.CacheMultiStore) {
	multiStore.Clear()
	ef.storePool.Put(multiStore)
}

func (ef EvmFactory) BuildSimulator(qoc QueryOnChainProxy) *EvmSimulator {
	keeper := ef.makeEvmKeeper(qoc)
	if !watcher.IsWatcherEnabled() {
		return nil
	}
	timestamp := time.Now()

	latest, _ := ef.WrappedQuerier.GetLatestBlockNumber()
	hash, e := ef.WrappedQuerier.GetBlockHashByNumber(latest)
	if e != nil {
		hash = common.HexToHash("0x000000000000000000000000000000")
	}

	block, e := ef.WrappedQuerier.GetBlockByHash(hash, false)

	if e == nil {
		timestamp = time.Unix(int64(block.Timestamp), 0)
	}
	req := abci.RequestBeginBlock{
		Header: abci.Header{
			ChainID: ef.ChainId,
			LastBlockId: abci.BlockID{
				Hash: hash.Bytes(),
			},
			Height: int64(latest),
			Time:   timestamp,
		},
		Hash: hash.Bytes(),
	}

	multiStore := ef.storePool.Get().(sdk.CacheMultiStore)
	ctx := ef.makeContext(multiStore, req.Header)

	keeper.BeginBlock(ctx, req)

	return &EvmSimulator{
		handler: evm.NewHandler(keeper),
		ctx:     ctx,
	}
}

type EvmSimulator struct {
	handler sdk.Handler
	ctx     sdk.Context
}

// DoCall call simulate tx. we pass the sender by args to reduce address convert
func (es *EvmSimulator) DoCall(msg *evmtypes.MsgEthereumTx, sender string, overridesBytes []byte, callBack func(sdk.CacheMultiStore)) (*sdk.SimulationResponse, error) {
	defer callBack(es.ctx.MultiStore().(sdk.CacheMultiStore))
	es.ctx.SetFrom(sender)
	if overridesBytes != nil {
		es.ctx.SetOverrideBytes(overridesBytes)
	}
	r, err := es.handler(es.ctx, msg)
	if err != nil {
		return nil, err
	}
	return &sdk.SimulationResponse{
		GasInfo: sdk.GasInfo{
			GasWanted: es.ctx.GasMeter().Limit(),
			GasUsed:   es.ctx.GasMeter().GasConsumed(),
		},
		Result: r,
	}, nil
}

func (ef EvmFactory) makeEvmKeeper(qoc QueryOnChainProxy) *evm.Keeper {
	return evm.NewSimulateKeeper(qoc.GetCodec(), ef.storeKey, NewSubspaceProxy(ef.paramsKey, ef.paramsTKey), NewAccountKeeperProxy(qoc), SupplyKeeperProxy{}, NewBankKeeperProxy(), StakingKeeperProxy{}, NewInternalDba(qoc), tmlog.NewNopLogger())
}

func (ef EvmFactory) makeContext(multiStore sdk.CacheMultiStore, header abci.Header) sdk.Context {
	ctx := sdk.NewContext(multiStore, header, true, tmlog.NewNopLogger())
	ctx.SetGasMeter(sdk.NewInfiniteGasMeter())
	return ctx
}
