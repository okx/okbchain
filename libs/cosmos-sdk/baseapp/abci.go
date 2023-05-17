package baseapp

import (
	"encoding/json"
	"fmt"
	"github.com/okx/okbchain/app/rpc/simulator"
	cosmost "github.com/okx/okbchain/libs/cosmos-sdk/store/types"
	cfg "github.com/okx/okbchain/libs/tendermint/config"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/okx/okbchain/libs/system/trace/persist"
	"github.com/spf13/viper"

	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	"github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	"github.com/okx/okbchain/libs/system/trace"
	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	tmtypes "github.com/okx/okbchain/libs/tendermint/types"
	"github.com/tendermint/go-amino"
)

// InitChain implements the ABCI interface. It runs the initialization logic
// directly on the CommitMultiStore.
func (app *BaseApp) InitChain(req abci.RequestInitChain) (res abci.ResponseInitChain) {
	// stash the consensus params in the cms main store and memoize
	if req.ConsensusParams != nil {
		app.setConsensusParams(req.ConsensusParams)
		app.storeConsensusParams(req.ConsensusParams)
	}

	initHeader := abci.Header{ChainID: req.ChainId, Time: req.Time}

	// initialize the deliver state and check state with a correct header
	app.setDeliverState(initHeader)
	app.setCheckState(initHeader)

	if app.initChainer == nil {
		return
	}

	// add block gas meter for any genesis transactions (allow infinite gas)
	app.deliverState.ctx.SetBlockGasMeter(sdk.NewInfiniteGasMeter())

	res = app.initChainer(app.deliverState.ctx, req)

	// sanity check
	if len(req.Validators) > 0 {
		if len(req.Validators) != len(res.Validators) {
			panic(
				fmt.Errorf(
					"len(RequestInitChain.Validators) != len(GenesisValidators) (%d != %d)",
					len(req.Validators), len(res.Validators),
				),
			)
		}

		sort.Sort(abci.ValidatorUpdates(req.Validators))
		sort.Sort(abci.ValidatorUpdates(res.Validators))

		for i, val := range res.Validators {
			if !val.Equal(req.Validators[i]) {
				panic(fmt.Errorf("genesisValidators[%d] != req.Validators[%d] ", i, i))
			}
		}
	}

	// NOTE: We don't commit, but BeginBlock for block 1 starts from this
	// deliverState.
	return res
}

// Info implements the ABCI interface.
func (app *BaseApp) Info(req abci.RequestInfo) abci.ResponseInfo {
	lastCommitID := app.cms.LastCommitID()

	return abci.ResponseInfo{
		Data:             app.name,
		LastBlockHeight:  lastCommitID.Version,
		LastBlockAppHash: lastCommitID.Hash,
	}
}

// SetOption implements the ABCI interface.
func (app *BaseApp) SetOption(req abci.RequestSetOption) (res abci.ResponseSetOption) {
	// TODO: Implement!
	switch req.Key {
	case "ResetCheckState":
		// reset check state
		app.setCheckState(app.checkState.ctx.BlockHeader())
	default:
		// do nothing
	}
	return
}

// FilterPeerByAddrPort filters peers by address/port.
func (app *BaseApp) FilterPeerByAddrPort(info string) abci.ResponseQuery {
	if app.addrPeerFilter != nil {
		return app.addrPeerFilter(info)
	}
	return abci.ResponseQuery{}
}

// FilterPeerByIDfilters peers by node ID.
func (app *BaseApp) FilterPeerByID(info string) abci.ResponseQuery {
	if app.idPeerFilter != nil {
		return app.idPeerFilter(info)
	}
	return abci.ResponseQuery{}
}

// BeginBlock implements the ABCI application interface.
func (app *BaseApp) BeginBlock(req abci.RequestBeginBlock) (res abci.ResponseBeginBlock) {
	sdk.TxIndex = 0
	app.blockDataCache.Clear()
	app.PutCacheMultiStore(nil)
	if app.cms.TracingEnabled() {
		app.cms.SetTracingContext(sdk.TraceContext(
			map[string]interface{}{"blockHeight": req.Header.Height},
		))
	}

	if err := app.validateHeight(req); err != nil {
		panic(err)
	}

	// Initialize the DeliverTx state. If this is the first block, it should
	// already be initialized in InitChain. Otherwise app.deliverState will be
	// nil, since it is reset on Commit.
	if req.Header.Height > 1+tmtypes.GetStartBlockHeight() {
		if app.deliverState != nil {
			app.logger.Info(
				"deliverState was not reset by BaseApp.Commit due to the previous prerun task being stopped",
				"height", req.Header.Height)
		}
		app.setDeliverState(req.Header)
	} else {
		// In the first block, app.deliverState.ctx will already be initialized
		// by InitChain. Context is now updated with Header information.
		app.deliverState.ctx.
			SetBlockHeader(req.Header).
			SetBlockHeight(req.Header.Height)
	}

	// add block gas meter
	var gasMeter sdk.GasMeter
	if maxGas := app.getMaximumBlockGas(); maxGas > 0 {
		gasMeter = sdk.NewGasMeter(maxGas)
	} else {
		gasMeter = sdk.NewInfiniteGasMeter()
	}

	app.deliverState.ctx.SetBlockGasMeter(gasMeter)

	if app.beginBlocker != nil {
		res = app.beginBlocker(app.deliverState.ctx, req)
	}

	// set the signed validators for addition to context in deliverTx
	app.voteInfos = req.LastCommitInfo.GetVotes()

	app.anteTracer = trace.NewTracer(trace.AnteChainDetail)

	app.feeCollector = sdk.Coins{}
	app.feeChanged = false
	// clean FeeSplitCollector
	app.FeeSplitCollector = make([]*sdk.FeeSplitInfo, 0)

	return res
}

func (app *BaseApp) UpdateFeeCollector(fee sdk.Coins, add bool) {
	if fee.IsZero() {
		return
	}
	app.feeChanged = true
	if add {
		app.feeCollector = app.feeCollector.Add(fee...)
	} else {
		app.feeCollector = app.feeCollector.Sub(fee)
	}
}

func (app *BaseApp) updateFeeCollectorAccount(isEndBlock bool) {
	if app.updateFeeCollectorAccHandler == nil || !app.feeChanged {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic: %v", r)
			app.logger.Error("update fee collector account failed", "err", err)
		}
	}()

	ctx, cache := app.cacheTxContext(app.getContextForTx(runTxModeDeliver, []byte{}), []byte{})
	if isEndBlock {
		// The feesplit is only processed at the endblock
		if err := app.updateFeeCollectorAccHandler(ctx, app.feeCollector, app.FeeSplitCollector); err != nil {
			panic(err)
		}
	} else {
		if err := app.updateFeeCollectorAccHandler(ctx, app.feeCollector, nil); err != nil {
			panic(err)
		}
	}
	cache.Write()
}

// EndBlock implements the ABCI interface.
func (app *BaseApp) EndBlock(req abci.RequestEndBlock) (res abci.ResponseEndBlock) {
	app.updateFeeCollectorAccount(true)

	if app.deliverState.ms.TracingEnabled() {
		app.deliverState.ms = app.deliverState.ms.SetTracingContext(nil).(sdk.CacheMultiStore)
	}

	if app.endBlocker != nil {
		res = app.endBlocker(app.deliverState.ctx, req)
	}
	if app.deliverState.ms != nil && app.deliverState.ctx.BlockHeight() == 949 {
		app.deliverState.ms.IteratorCache(true, func(key string, value []byte, isDirty bool, isDelete bool, storeKey cosmost.StoreKey) bool {
			//fmt.Println("dirtty", hex.EncodeToString([]byte(key)), hex.EncodeToString(value))
			return true
		}, nil)
	}

	return
}

func (app *BaseApp) addCommitTraceInfo() {
	nodeReadCountStr := strconv.Itoa(app.cms.GetNodeReadCount())
	dbReadCountStr := strconv.Itoa(app.cms.GetDBReadCount())
	dbReadTimeStr := strconv.FormatInt(time.Duration(app.cms.GetDBReadTime()).Milliseconds(), 10)
	dbWriteCountStr := strconv.Itoa(app.cms.GetDBWriteCount())

	info := strings.Join([]string{"getnode<", nodeReadCountStr, ">, rdb<", dbReadCountStr, ">, rdbTs<", dbReadTimeStr, "ms>, savenode<", dbWriteCountStr, ">"}, "")

	elapsedInfo := trace.GetElapsedInfo()
	elapsedInfo.AddInfo(trace.Storage, info)

	flatKvReadCountStr := strconv.Itoa(app.cms.GetFlatKVReadCount())
	flatKvReadTimeStr := strconv.FormatInt(time.Duration(app.cms.GetFlatKVReadTime()).Milliseconds(), 10)
	flatKvWriteCountStr := strconv.Itoa(app.cms.GetFlatKVWriteCount())
	flatKvWriteTimeStr := strconv.FormatInt(time.Duration(app.cms.GetFlatKVWriteTime()).Milliseconds(), 10)

	flatInfo := strings.Join([]string{"rflat<", flatKvReadCountStr, ">, rflatTs<", flatKvReadTimeStr, "ms>, wflat<", flatKvWriteCountStr, ">, wflatTs<", flatKvWriteTimeStr, "ms>"}, "")

	elapsedInfo.AddInfo(trace.FlatKV, flatInfo)

	//rtx := float64(atomic.LoadInt64(&app.checkTxNum))
	//wtx := float64(atomic.LoadInt64(&app.wrappedCheckTxNum))

	//elapsedInfo.AddInfo(trace.WtxRatio,
	//	amino.BytesToStr(strconv.AppendFloat(make([]byte, 0, 4), wtx/(wtx+rtx), 'f', 2, 32)),
	//)

	readCache := float64(tmtypes.SignatureCache().ReadCount())
	hitCache := float64(tmtypes.SignatureCache().HitCount())

	elapsedInfo.AddInfo(trace.SigCacheRatio,
		amino.BytesToStr(strconv.AppendFloat(make([]byte, 0, 4), hitCache/readCache, 'f', 2, 32)),
	)

	elapsedInfo.AddInfo(trace.AnteChainDetail, app.anteTracer.FormatRepeatingPins(sdk.AnteTerminatorTag))
}

// Commit implements the ABCI interface. It will commit all state that exists in
// the deliver state's multi-store and includes the resulting commit ID in the
// returned abci.ResponseCommit. Commit will set the check state based on the
// latest header and reset the deliver state. Also, if a non-zero halt height is
// defined in config, Commit will execute a deferred function call to check
// against that height and gracefully halt if it matches the latest committed
// height.
func (app *BaseApp) Commit(req abci.RequestCommit) abci.ResponseCommit {

	persist.GetStatistics().Init(trace.FlushCache, trace.CommitStores, trace.FlushMeta)
	defer func() {
		trace.GetElapsedInfo().AddInfo(trace.PersistDetails, persist.GetStatistics().Format())
	}()
	header := app.deliverState.ctx.BlockHeader()

	if mptStore := app.cms.GetCommitKVStore(sdk.NewKVStoreKey(mpt.StoreKey)); mptStore != nil {
		// notify mptStore to tryUpdateTrie, must call before app.deliverState.ms.Write()
		mpt.GAccTryUpdateTrieChannel <- struct{}{}
		<-mpt.GAccTrieUpdatedChannel
	}

	// Write the DeliverTx state which is cache-wrapped and commit the MultiStore.
	// The write to the DeliverTx state writes all state transitions to the root
	// MultiStore (app.cms) so when Commit() is called is persists those values.
	app.deliverState.ms.Write()

	var input *tmtypes.TreeDelta
	if tmtypes.DownloadDelta && req.DeltaMap != nil {
		var ok bool
		input, ok = req.DeltaMap.(*tmtypes.TreeDelta)
		if !ok {
			panic("use TreeDeltaMap failed")
		}
	}

	commitID, output := app.cms.CommitterCommitMap(input) // CommitterCommitMap

	app.addCommitTraceInfo()

	app.cms.ResetCount()
	app.logger.Debug("Commit synced", "commit", amino.BytesHexStringer(commitID.Hash))

	// Reset the Check state to the latest committed.
	//
	// NOTE: This is safe because Tendermint holds a lock on the mempool for
	// Commit. Use the header from this latest block.
	app.setCheckState(header)

	app.logger.Debug("deliverState reset by BaseApp.Commit", "height", header.Height)
	// empty/reset the deliver state
	app.deliverState = nil

	var halt bool

	switch {
	case app.haltHeight > 0 && uint64(header.Height) >= app.haltHeight:
		halt = true

	case app.haltTime > 0 && header.Time.Unix() >= int64(app.haltTime):
		halt = true
	}

	if halt {
		// Halt the binary and allow Tendermint to receive the ResponseCommit
		// response with the commit ID hash. This will allow the node to successfully
		// restart and process blocks assuming the halt configuration has been
		// reset or moved to a more distant value.
		app.halt()
	}

	return abci.ResponseCommit{
		Data:     commitID.Hash,
		DeltaMap: output,
	}
}

// halt attempts to gracefully shutdown the node via SIGINT and SIGTERM falling
// back on os.Exit if both fail.
func (app *BaseApp) halt() {
	app.logger.Info("halting node per configuration", "height", app.haltHeight, "time", app.haltTime)

	p, err := os.FindProcess(os.Getpid())
	if err == nil {
		// attempt cascading signals in case SIGINT fails (os dependent)
		sigIntErr := p.Signal(syscall.SIGINT)
		sigTermErr := p.Signal(syscall.SIGTERM)
		//Make sure the TrapSignal execute first
		time.Sleep(50 * time.Millisecond)
		if sigIntErr == nil || sigTermErr == nil {
			return
		}
	}

	// Resort to exiting immediately if the process could not be found or killed
	// via SIGINT/SIGTERM signals.
	app.logger.Info("failed to send SIGINT/SIGTERM; exiting...")
	os.Exit(0)
}

// Query implements the ABCI interface. It delegates to CommitMultiStore if it
// implements Queryable.
func (app *BaseApp) Query(req abci.RequestQuery) abci.ResponseQuery {
	ceptor := app.interceptors[req.Path]
	if nil != ceptor {
		// interceptor is like `aop`,it may record the request or rewrite the data in the request
		// it should have funcs like `Begin` `End`,
		// but for now, we will just redirect the path router,so once the request was intercepted(see #makeInterceptors),
		// grpcQueryRouter#Route will return nil
		ceptor.Intercept(&req)
	}
	path := splitPath(req.Path)
	if len(path) == 0 {
		return sdkerrors.QueryResult(sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "no query path provided"))
	}

	if req.Height == 0 {
		req.Height = app.LastBlockHeight()
	}

	if grpcHandler := app.grpcQueryRouter.Route(req.Path); grpcHandler != nil {
		return app.handleQueryGRPC(grpcHandler, req)
	}

	switch path[0] {
	// "/app" prefix for special application queries
	case "app":
		return handleQueryApp(app, path, req)

	case "store":
		return handleQueryStore(app, path, req)

	case "p2p":
		return handleQueryP2P(app, path)

	case "custom":
		return handleQueryCustom(app, path, req)
	default:
		return sdkerrors.QueryResult(sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "unknown query path"))
	}
}

func handleSimulateWithBuffer(app *BaseApp, path []string, height int64, txBytes []byte, overrideBytes []byte) abci.ResponseQuery {
	simRes, shouldAddBuffer, err := handleSimulate(app, path, height, txBytes, overrideBytes)
	if err != nil {
		return sdkerrors.QueryResult(err)
	}
	if shouldAddBuffer {
		buffer := cfg.DynamicConfig.GetGasLimitBuffer()
		gasUsed := simRes.GasUsed
		gasUsed += gasUsed * buffer / 100
		if gasUsed > SimulationGasLimit {
			gasUsed = SimulationGasLimit
		}
		simRes.GasUsed = gasUsed
	}

	return abci.ResponseQuery{
		Codespace: sdkerrors.RootCodespace,
		Height:    height,
		Value:     codec.Cdc.MustMarshalBinaryBare(simRes),
	}

}

func handleSimulate(app *BaseApp, path []string, height int64, txBytes []byte, overrideBytes []byte) (sdk.SimulationResponse, bool, error) {
	// if path contains address, it means 'eth_estimateGas' the sender
	hasExtraPaths := len(path) > 2
	var from string
	if hasExtraPaths {
		if addr, err := sdk.AccAddressFromBech32(path[2]); err == nil {
			if err = sdk.VerifyAddressFormat(addr); err == nil {
				from = path[2]
			}
		}
	}

	var tx sdk.Tx
	var err error
	if mem := GetGlobalMempool(); mem != nil {
		tx, _ = mem.ReapEssentialTx(txBytes).(sdk.Tx)
	}
	if tx == nil {
		tx, err = app.txDecoder(txBytes)
		if err != nil {
			return sdk.SimulationResponse{}, false, sdkerrors.Wrap(err, "failed to decode tx")
		}
	}
	// if path contains mempool, it means to enable MaxGasUsedPerBlock
	// return the actual gasUsed even though simulate tx failed
	isMempoolSim := hasExtraPaths && path[2] == "mempool"
	var shouldAddBuffer bool
	if !isMempoolSim && tx.GetType() != types.EvmTxType {
		shouldAddBuffer = true
	}

	msgs := tx.GetMsgs()

	if enableFastQuery() {
		isPureWasm := true
		for _, msg := range msgs {
			if msg.Route() != "wasm" {
				isPureWasm = false
				break
			}
		}
		if isPureWasm {
			res, err := handleSimulateWasm(height, txBytes, msgs)
			return res, shouldAddBuffer, err
		}
	}
	gInfo, res, err := app.Simulate(txBytes, tx, height, overrideBytes, from)
	if err != nil && !isMempoolSim {
		return sdk.SimulationResponse{}, false, sdkerrors.Wrap(err, "failed to simulate tx")
	}

	return sdk.SimulationResponse{
		GasInfo: gInfo,
		Result:  res,
	}, shouldAddBuffer, nil
}

func handleSimulateWasm(height int64, txBytes []byte, msgs []sdk.Msg) (simRes sdk.SimulationResponse, err error) {
	wasmSimulator := simulator.NewWasmSimulator()
	defer wasmSimulator.Release()
	defer func() {
		if r := recover(); r != nil {
			gasMeter := wasmSimulator.Context().GasMeter()
			simRes = sdk.SimulationResponse{
				GasInfo: sdk.GasInfo{
					GasUsed: gasMeter.GasConsumed(),
				},
			}
		}
	}()

	wasmSimulator.Context().GasMeter().ConsumeGas(73000, "general ante check cost")
	wasmSimulator.Context().GasMeter().ConsumeGas(uint64(10*len(txBytes)), "tx size cost")
	res, err := wasmSimulator.Simulate(msgs)
	if err != nil {
		return sdk.SimulationResponse{}, sdkerrors.Wrap(err, "failed to simulate wasm tx")
	}

	gasMeter := wasmSimulator.Context().GasMeter()
	return sdk.SimulationResponse{
		GasInfo: sdk.GasInfo{
			GasUsed: gasMeter.GasConsumed(),
		},
		Result: res,
	}, nil
}

func handleQueryApp(app *BaseApp, path []string, req abci.RequestQuery) abci.ResponseQuery {
	if len(path) >= 2 {
		switch path[1] {
		case "simulate":
			return handleSimulateWithBuffer(app, path, req.Height, req.Data, nil)

		case "simulateWithOverrides":
			queryBytes := req.Data
			var queryData types.SimulateData
			if err := json.Unmarshal(queryBytes, &queryData); err != nil {
				return sdkerrors.QueryResult(sdkerrors.Wrap(err, "failed to decode simulateOverrideData"))
			}
			return handleSimulateWithBuffer(app, path, req.Height, queryData.TxBytes, queryData.OverridesBytes)

		case "trace":
			var queryParam sdk.QueryTraceTx
			err := json.Unmarshal(req.Data, &queryParam)
			if err != nil {
				return sdkerrors.QueryResult(sdkerrors.Wrap(err, "invalid trace tx params"))
			}
			tmtx, err := GetABCITx(queryParam.TxHash.Bytes())
			if err != nil {
				return sdkerrors.QueryResult(sdkerrors.Wrap(err, "invalid trace tx bytes"))
			}
			tx, err := app.txDecoder(tmtx.Tx, tmtx.Height)
			if err != nil {
				return sdkerrors.QueryResult(sdkerrors.Wrap(err, "failed to decode tx"))
			}
			block, err := GetABCIBlock(tmtx.Height)
			if err != nil {
				return sdkerrors.QueryResult(sdkerrors.Wrap(err, "invalid trace tx block header"))
			}
			res, err := app.TraceTx(queryParam, tx, tmtx.Index, block.Block)
			if err != nil {
				return sdkerrors.QueryResult(sdkerrors.Wrap(err, "failed to trace tx"))
			}
			return abci.ResponseQuery{
				Codespace: sdkerrors.RootCodespace,
				Height:    req.Height,
				Value:     codec.Cdc.MustMarshalBinaryBare(res),
			}

		case "version":
			return abci.ResponseQuery{
				Codespace: sdkerrors.RootCodespace,
				Height:    req.Height,
				Value:     []byte(app.appVersion),
			}

		default:
			return sdkerrors.QueryResult(sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unknown query: %s", path))
		}
	}

	return sdkerrors.QueryResult(
		sdkerrors.Wrap(
			sdkerrors.ErrUnknownRequest,
			"expected second parameter to be either 'simulate' or 'version', neither was present",
		),
	)
}

func handleQueryStore(app *BaseApp, path []string, req abci.RequestQuery) abci.ResponseQuery {
	// "/store" prefix for store queries
	queryable, ok := app.cms.(sdk.Queryable)
	if !ok {
		return sdkerrors.QueryResult(sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "multistore doesn't support queries"))
	}

	req.Path = "/" + strings.Join(path[1:], "/")

	// when a client did not provide a query height, manually inject the latest
	if req.Height == 0 {
		req.Height = app.LastBlockHeight()
	}

	if req.Height <= 1 && req.Prove {
		return sdkerrors.QueryResult(
			sdkerrors.Wrap(
				sdkerrors.ErrInvalidRequest,
				"cannot query with proof when height <= 1; please provide a valid height",
			),
		)
	}

	resp := queryable.Query(req)
	resp.Height = req.Height

	return resp
}

func handleQueryP2P(app *BaseApp, path []string) abci.ResponseQuery {
	// "/p2p" prefix for p2p queries
	if len(path) >= 4 {
		cmd, typ, arg := path[1], path[2], path[3]
		switch cmd {
		case "filter":
			switch typ {
			case "addr":
				return app.FilterPeerByAddrPort(arg)

			case "id":
				return app.FilterPeerByID(arg)
			}

		default:
			return sdkerrors.QueryResult(sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "expected second parameter to be 'filter'"))
		}
	}

	return sdkerrors.QueryResult(
		sdkerrors.Wrap(
			sdkerrors.ErrUnknownRequest, "expected path is p2p filter <addr|id> <parameter>",
		),
	)
}

func handleQueryCustom(app *BaseApp, path []string, req abci.RequestQuery) abci.ResponseQuery {
	// path[0] should be "custom" because "/custom" prefix is required for keeper
	// queries.
	//
	// The QueryRouter routes using path[1]. For example, in the path
	// "custom/gov/proposal", QueryRouter routes using "gov".
	if len(path) < 2 || path[1] == "" {
		return sdkerrors.QueryResult(sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "no route for custom query specified"))
	}

	querier := app.queryRouter.Route(path[1])
	if querier == nil {
		return sdkerrors.QueryResult(sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "no custom querier found for route %s", path[1]))
	}

	// when a client did not provide a query height, manually inject the latest
	if req.Height == 0 {
		req.Height = app.LastBlockHeight()
	}

	if req.Height <= 1 && req.Prove {
		return sdkerrors.QueryResult(
			sdkerrors.Wrap(
				sdkerrors.ErrInvalidRequest,
				"cannot query with proof when height <= 1; please provide a valid height",
			),
		)
	}

	cacheMS, err := app.cms.CacheMultiStoreWithVersion(req.Height)
	if err != nil {
		return sdkerrors.QueryResult(
			sdkerrors.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"failed to load state at height %d; %s (latest height: %d)", req.Height, err, app.LastBlockHeight(),
			),
		)
	}

	// cache wrap the commit-multistore for safety
	ctx := sdk.NewContext(
		cacheMS, app.checkState.ctx.BlockHeader(), true, app.logger,
	)
	ctx.SetMinGasPrices(app.minGasPrices)
	ctx.SetBlockHeight(req.Height)

	// Passes the rest of the path as an argument to the querier.
	//
	// For example, in the path "custom/gov/proposal/test", the gov querier gets
	// []string{"proposal", "test"} as the path.
	resBytes, err := querier(ctx, path[2:], req)
	if err != nil {
		space, code, log := sdkerrors.ABCIInfo(err, false)
		return abci.ResponseQuery{
			Code:      code,
			Codespace: space,
			Log:       log,
			Height:    req.Height,
		}
	}

	return abci.ResponseQuery{
		Height: req.Height,
		Value:  resBytes,
	}
}

// splitPath splits a string path using the delimiter '/'.
//
// e.g. "this/is/funny" becomes []string{"this", "is", "funny"}
func splitPath(requestPath string) (path []string) {
	path = strings.Split(requestPath, "/")

	// first element is empty string
	if len(path) > 0 && path[0] == "" {
		path = path[1:]
	}

	return path
}

var (
	fastQuery bool
	fqOnce    sync.Once
)

func enableFastQuery() bool {
	fqOnce.Do(func() {
		fastQuery = viper.GetBool("fast-query")
	})
	return fastQuery
}
