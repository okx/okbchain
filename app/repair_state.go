package app

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/okx/okbchain/app/config"
	"github.com/okx/okbchain/app/utils/appstatus"
	"github.com/okx/okbchain/libs/cosmos-sdk/server"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/flatkv"
	mpttypes "github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/rootmulti"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/iavl"
	"github.com/okx/okbchain/libs/system"
	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	cfg "github.com/okx/okbchain/libs/tendermint/config"
	"github.com/okx/okbchain/libs/tendermint/global"
	tmlog "github.com/okx/okbchain/libs/tendermint/libs/log"
	"github.com/okx/okbchain/libs/tendermint/mock"
	"github.com/okx/okbchain/libs/tendermint/node"
	"github.com/okx/okbchain/libs/tendermint/proxy"
	sm "github.com/okx/okbchain/libs/tendermint/state"
	blockindex "github.com/okx/okbchain/libs/tendermint/state/indexer"
	blockindexer "github.com/okx/okbchain/libs/tendermint/state/indexer/block/kv"
	bloxkindexnull "github.com/okx/okbchain/libs/tendermint/state/indexer/block/null"
	"github.com/okx/okbchain/libs/tendermint/state/txindex"
	"github.com/okx/okbchain/libs/tendermint/state/txindex/kv"
	"github.com/okx/okbchain/libs/tendermint/state/txindex/null"
	"github.com/okx/okbchain/libs/tendermint/store"
	"github.com/okx/okbchain/libs/tendermint/types"
	dbm "github.com/okx/okbchain/libs/tm-db"
	evmtypes "github.com/okx/okbchain/x/evm/types"
	"github.com/spf13/viper"
)

const (
	applicationDB = "application"
	blockStoreDB  = "blockstore"
	stateDB       = "state"
	txIndexDB     = "tx_index"
	blockIndexDb  = "block_index"

	FlagStartHeight       string = "start-height"
	FlagEnableRepairState string = "enable-repair-state"
)

type repairApp struct {
	db dbm.DB
	*OKBChainApp
}

func (app *repairApp) getLatestVersion() int64 {
	rs := rootmulti.NewStore(app.db)
	return rs.GetLatestVersion()
}

func repairStateOnStart(ctx *server.Context) {
	// set flag
	orgIgnoreSmbCheck := sm.IgnoreSmbCheck
	orgIgnoreVersionCheck := iavl.GetIgnoreVersionCheck()
	orgEnableFlatKV := viper.GetBool(flatkv.FlagEnable)
	iavl.EnableAsyncCommit = false
	mpttypes.EnableAsyncCommit = false
	viper.Set(flatkv.FlagEnable, false)
	iavl.SetEnableFastStorage(appstatus.IsFastStorageStrategy())
	iavl.SetForceReadIavl(true)

	// repair state
	RepairState(ctx, true)

	//set original flag
	iavl.SetForceReadIavl(false)
	sm.SetIgnoreSmbCheck(orgIgnoreSmbCheck)
	iavl.SetIgnoreVersionCheck(orgIgnoreVersionCheck)

	treeEnableAc := viper.GetBool(system.FlagTreeEnableAsyncCommit)
	iavl.EnableAsyncCommit = treeEnableAc
	mpttypes.EnableAsyncCommit = treeEnableAc

	viper.Set(flatkv.FlagEnable, orgEnableFlatKV)
	// load latest block height
}

func RepairState(ctx *server.Context, onStart bool) {
	sm.SetIgnoreSmbCheck(true)
	iavl.SetIgnoreVersionCheck(true)
	global.SetRepairState(true)
	defer global.SetRepairState(false)

	// load latest block height
	dataDir := filepath.Join(ctx.Config.RootDir, "data")
	latestBlockHeight := latestBlockHeight(dataDir)
	startBlockHeight := types.GetStartBlockHeight()
	if latestBlockHeight <= startBlockHeight+2 {
		log.Println(fmt.Sprintf("There is no need to repair data. The latest block height is %d, start block height is %d", latestBlockHeight, startBlockHeight))
		return
	}

	config.RegisterDynamicConfig(ctx.Logger.With("module", "config"))
	// create proxy app
	proxyApp, repairApp, err := createRepairApp(ctx)
	panicError(err)
	defer repairApp.Close()

	// get async commit version
	commitVersion, err := repairApp.GetCommitVersion()
	log.Println(fmt.Sprintf("repair state latestBlockHeight = %d \t commitVersion = %d", latestBlockHeight, commitVersion))
	panicError(err)

	if onStart && commitVersion == latestBlockHeight {
		log.Println("no need to repair state on start")
		return
	}

	// load state
	stateStoreDB, err := sdk.NewDB(stateDB, dataDir)
	panicError(err)
	defer func() {
		err := stateStoreDB.Close()
		panicError(err)
	}()
	genesisDocProvider := node.DefaultGenesisDocProviderFunc(ctx.Config)
	state, _, err := node.LoadStateFromDBOrGenesisDocProvider(stateStoreDB, genesisDocProvider)
	panicError(err)

	// load start version
	startVersion := viper.GetInt64(FlagStartHeight)
	if startVersion == 0 {
		if onStart {
			startVersion = commitVersion
		} else {
			stateOkHeight := latestBlockHeight - 2 // case: state machine broken
			if commitVersion <= stateOkHeight {
				startVersion = commitVersion
			} else {
				startVersion = stateOkHeight
			}
		}
	}
	if startVersion <= 0 {
		panic("height too low, please restart from height 0 with genesis file")
	}
	log.Println(fmt.Sprintf("repair state at version = %d", startVersion))

	err = repairApp.LoadStartVersion(startVersion)
	panicError(err)

	repairApp.InitUpgrade(repairApp.BaseApp.NewContext(true, abci.Header{}))

	rawTrieDirtyDisabledFlag := viper.GetBool(mpttypes.FlagTrieDirtyDisabled)
	mpttypes.TrieDirtyDisabled = true

	// repair data by apply the latest two blocks
	doRepair(ctx, state, stateStoreDB, proxyApp, startVersion, latestBlockHeight, dataDir)

	mpttypes.TrieDirtyDisabled = rawTrieDirtyDisabledFlag
}
func createRepairApp(ctx *server.Context) (proxy.AppConns, *repairApp, error) {
	rootDir := ctx.Config.RootDir
	dataDir := filepath.Join(rootDir, "data")
	db, err := sdk.NewDB(applicationDB, dataDir)
	panicError(err)
	repairApp := newRepairApp(ctx.Logger, db, nil)

	clientCreator := proxy.NewLocalClientCreator(repairApp)
	// Create the proxyApp and establish connections to the ABCI app (consensus, mempool, query).
	proxyApp, err := createAndStartProxyAppConns(clientCreator)
	return proxyApp, repairApp, err
}

func newRepairApp(logger tmlog.Logger, db dbm.DB, traceStore io.Writer) *repairApp {
	return &repairApp{db, NewOKBChainApp(
		logger,
		db,
		traceStore,
		false,
		map[int64]bool{},
		0,
	)}
}

func doRepair(ctx *server.Context, state sm.State, stateStoreDB dbm.DB,
	proxyApp proxy.AppConns, startHeight, latestHeight int64, dataDir string) {
	stateCopy := state.Copy()

	ctx.Logger.Debug("stateCopy", "state", fmt.Sprintf("%+v", stateCopy))
	// construct state for repair
	state = constructStartState(state, stateStoreDB, startHeight)
	ctx.Logger.Debug("constructStartState", "state", fmt.Sprintf("%+v", state))
	// repair state
	eventBus := types.NewEventBus()
	txStore, blockIndexStore, txindexServer, err := startEventBusAndIndexerService(ctx.Config, eventBus, ctx.Logger)
	panicError(err)
	blockExec := sm.NewBlockExecutor(stateStoreDB, ctx.Logger, proxyApp.Consensus(), mock.Mempool{}, sm.MockEvidencePool{})
	blockExec.SetEventBus(eventBus)
	// Save state synchronously during repair state
	blockExec.SetIsAsyncSaveDB(false)
	defer func() {
		// stop sequence is important to avoid data missing: blockExecutor->eventBus->txIndexer
		// keep the same sequence as node.go:OnStop
		blockExec.Stop()

		if eventBus != nil && eventBus.IsRunning() {
			eventBus.Stop()
			eventBus.Wait()
		}
		if txindexServer != nil && txindexServer.IsRunning() {
			txindexServer.Stop()
			txindexServer.Wait()
		}
		if txStore != nil {
			err := txStore.Close()
			panicError(err)
		}
		if blockIndexStore != nil {
			err := blockIndexStore.Close()
			panicError(err)
		}
	}()

	global.SetGlobalHeight(startHeight + 1)
	for height := startHeight + 1; height <= latestHeight; height++ {
		repairBlock, repairBlockMeta := loadBlock(height, dataDir)
		state, _, err = blockExec.ApplyBlockWithTrace(state, repairBlockMeta.BlockID, repairBlock)
		panicError(err)
		ctx.Logger.Debug("repairedState", "state", fmt.Sprintf("%+v", state))
		res, err := proxyApp.Query().InfoSync(proxy.RequestInfo)
		panicError(err)
		repairedBlockHeight := res.LastBlockHeight
		repairedAppHash := res.LastBlockAppHash
		log.Println("Repaired block height", repairedBlockHeight)
		log.Println("Repaired app hash", fmt.Sprintf("%X", repairedAppHash))
	}
}

func startEventBusAndIndexerService(config *cfg.Config, eventBus *types.EventBus, logger tmlog.Logger) (txStore dbm.DB, blockIndexStore dbm.DB, indexerService *txindex.IndexerService, err error) {
	eventBus.SetLogger(logger.With("module", "events"))
	if err := eventBus.Start(); err != nil {
		return nil, nil, nil, err
	}
	// Transaction indexing
	var txIndexer txindex.TxIndexer
	var blockIndexer blockindex.BlockIndexer
	switch config.TxIndex.Indexer {
	case "kv":
		txStore, err = sdk.NewDB(txIndexDB, filepath.Join(config.RootDir, "data"))
		if err != nil {
			return nil, nil, nil, err
		}
		blockIndexStore, err = sdk.NewDB(blockIndexDb, filepath.Join(config.RootDir, "data"))
		if err != nil {
			return nil, nil, nil, err
		}
		switch {
		case config.TxIndex.IndexKeys != "":
			txIndexer = kv.NewTxIndex(txStore, kv.IndexEvents(splitAndTrimEmpty(config.TxIndex.IndexKeys, ",", " ")))
		case config.TxIndex.IndexAllKeys:
			txIndexer = kv.NewTxIndex(txStore, kv.IndexAllEvents())
		default:
			txIndexer = kv.NewTxIndex(txStore)
		}
		blockIndexer = blockindexer.New(dbm.NewPrefixDB(blockIndexStore, []byte("block_events")))
	default:
		txIndexer = &null.TxIndex{}
		blockIndexer = &bloxkindexnull.BlockerIndexer{}
	}

	indexerService = txindex.NewIndexerService(txIndexer, blockIndexer, eventBus)
	indexerService.SetLogger(logger.With("module", "txindex"))
	if err := indexerService.Start(); err != nil {
		if eventBus != nil {
			eventBus.Stop()
		}
		if txStore != nil {
			txStore.Close()
		}

		return nil, nil, nil, err
	}
	return txStore, blockIndexStore, indexerService, nil
}

// splitAndTrimEmpty slices s into all subslices separated by sep and returns a
// slice of the string s with all leading and trailing Unicode code points
// contained in cutset removed. If sep is empty, SplitAndTrim splits after each
// UTF-8 sequence. First part is equivalent to strings.SplitN with a count of
// -1.  also filter out empty strings, only return non-empty strings.
func splitAndTrimEmpty(s, sep, cutset string) []string {
	if s == "" {
		return []string{}
	}

	spl := strings.Split(s, sep)
	nonEmptyStrings := make([]string, 0, len(spl))
	for i := 0; i < len(spl); i++ {
		element := strings.Trim(spl[i], cutset)
		if element != "" {
			nonEmptyStrings = append(nonEmptyStrings, element)
		}
	}
	return nonEmptyStrings
}

func constructStartState(state sm.State, stateStoreDB dbm.DB, startHeight int64) sm.State {
	stateCopy := state.Copy()
	validators, lastStoredHeight, err := sm.LoadValidatorsWithStoredHeight(stateStoreDB, startHeight+1)
	lastValidators, err := sm.LoadValidators(stateStoreDB, startHeight)
	if err != nil {
		return stateCopy
	}
	nextValidators, err := sm.LoadValidators(stateStoreDB, startHeight+2)
	if err != nil {
		return stateCopy
	}
	consensusParams, err := sm.LoadConsensusParams(stateStoreDB, startHeight+1)
	if err != nil {
		return stateCopy
	}
	stateCopy.Validators = validators
	stateCopy.LastValidators = lastValidators
	stateCopy.NextValidators = nextValidators
	stateCopy.ConsensusParams = consensusParams
	stateCopy.LastBlockHeight = startHeight
	stateCopy.LastHeightValidatorsChanged = lastStoredHeight
	return stateCopy
}

func loadBlock(height int64, dataDir string) (*types.Block, *types.BlockMeta) {
	storeDB, err := sdk.NewDB(blockStoreDB, dataDir)
	defer storeDB.Close()
	blockStore := store.NewBlockStore(storeDB)
	panicError(err)
	block := blockStore.LoadBlock(height)
	meta := blockStore.LoadBlockMeta(height)
	return block, meta
}

func latestBlockHeight(dataDir string) int64 {
	storeDB, err := sdk.NewDB(blockStoreDB, dataDir)
	panicError(err)
	defer storeDB.Close()
	blockStore := store.NewBlockStore(storeDB)
	return blockStore.Height()
}

// panic if error is not nil
func panicError(err error) {
	if err != nil {
		panic(err)
	}
}

func createAndStartProxyAppConns(clientCreator proxy.ClientCreator) (proxy.AppConns, error) {
	proxyApp := proxy.NewAppConns(clientCreator)
	if err := proxyApp.Start(); err != nil {
		return nil, fmt.Errorf("error starting proxy app connections: %v", err)
	}
	return proxyApp, nil
}

func (app *repairApp) Close() {
	app.StopBaseApp()

	indexer := evmtypes.GetIndexer()
	if indexer != nil {
		for indexer.IsProcessing() {
			time.Sleep(100 * time.Millisecond)
		}
	}
	evmtypes.CloseIndexer()
	err := app.db.Close()
	panicError(err)
}
