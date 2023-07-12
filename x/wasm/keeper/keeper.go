package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/gogo/protobuf/proto"

	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/prefix"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	"github.com/okx/okbchain/libs/cosmos-sdk/types/innertx"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
	"github.com/okx/okbchain/libs/tendermint/libs/log"
	paramtypes "github.com/okx/okbchain/x/params"
	"github.com/okx/okbchain/x/wasm/ioutils"
	"github.com/okx/okbchain/x/wasm/types"
	"github.com/okx/okbchain/x/wasm/watcher"
)

// contractMemoryLimit is the memory limit of each contract execution (in MiB)
// constant value so all nodes run with the same limit.
const ContractMemoryLimit = 32
const SupportedFeatures = "iterator,staking,stargate"

type contextKey int

const (
	// private type creates an interface key for Context that cannot be accessed by any other package
	contextKeyQueryStackSize contextKey = iota
)

// Option is an extension point to instantiate keeper with non default values
type Option interface {
	apply(*Keeper)
}

// WasmVMQueryHandler is an extension point for custom query handler implementations
type WasmVMQueryHandler interface {
	// HandleQuery executes the requested query
	HandleQuery(ctx sdk.Context, caller sdk.WasmAddress, request wasmvmtypes.QueryRequest) ([]byte, error)
}

type CoinTransferrer interface {
	// TransferCoins sends the coin amounts from the source to the destination with rules applied.
	TransferCoins(ctx sdk.Context, fromAddr sdk.WasmAddress, toAddr sdk.WasmAddress, amt sdk.Coins) error
}

// WasmVMResponseHandler is an extension point to handles the response data returned by a contract call.
type WasmVMResponseHandler interface {
	// Handle processes the data returned by a contract invocation.
	Handle(
		ctx sdk.Context,
		contractAddr sdk.WasmAddress,
		ibcPort string,
		messages []wasmvmtypes.SubMsg,
		origRspData []byte,
	) ([]byte, error)
}

// Keeper will have a reference to Wasmer with it's own data directory.
type Keeper struct {
	storeKey              sdk.StoreKey
	storageStoreKey       sdk.StoreKey // wasm contract storage will store into mpt store
	cdc                   *codec.CodecProxy
	accountKeeper         types.AccountKeeper
	bank                  CoinTransferrer
	portKeeper            types.PortKeeper
	capabilityKeeper      types.CapabilityKeeper
	wasmVM                types.WasmerEngine
	wasmVMQueryHandler    WasmVMQueryHandler
	wasmVMResponseHandler WasmVMResponseHandler
	messenger             Messenger
	innertxKeeper         innertx.InnerTxKeeper

	// queryGasLimit is the max wasmvm gas that can be spent on executing a query with a contract
	queryGasLimit     uint64
	paramSpace        types.Subspace
	gasRegister       GasRegister
	maxQueryStackSize uint32
	ada               types.DBAdapter
}

type defaultAdapter struct{}

func (d defaultAdapter) NewStore(ctx sdk.Context, storeKey sdk.StoreKey, pre []byte) sdk.KVStore {
	store := ctx.KVStore(storeKey)
	store = watcher.WrapWriteKVStore(store)
	if len(pre) != 0 {
		store = prefix.NewStore(store, pre)
	}

	return store
}

// NewKeeper creates a new contract Keeper instance
// If customEncoders is non-nil, we can use this to override some of the message handler, especially custom
func NewKeeper(
	cdc *codec.CodecProxy,
	storeKey sdk.StoreKey,
	storageKey sdk.StoreKey,
	paramSpace paramtypes.Subspace,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	//distKeeper types.DistributionKeeper,
	channelKeeper types.ChannelKeeper,
	portKeeper types.PortKeeper,
	capabilityKeeper types.CapabilityKeeper,
	portSource types.ICS20TransferPortSource,
	router MessageRouter,
	queryRouter GRPCQueryRouter,
	homeDir string,
	wasmConfig types.WasmConfig,
	supportedFeatures string,
	opts ...Option,
) Keeper {
	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}
	watcher.SetWatchDataManager()
	wasmStorageKey = storeKey
	wasmMptStorageKey = storageKey
	*wasmAccountKeeper = accountKeeper
	*WasmbankKeeper = bankKeeper
	k := newKeeper(cdc, storeKey, storageKey, paramSpace, accountKeeper, bankKeeper, channelKeeper, portKeeper, capabilityKeeper, portSource, router, queryRouter, homeDir, wasmConfig, supportedFeatures, defaultAdapter{}, opts...)
	*wasmGasRegister = k.gasRegister
	accountKeeper.SetObserverKeeper(k)

	return k
}

var (
	nilwasmGasRegister = GasRegister(nil)
	nilAccountKeeper   = types.AccountKeeper(nil)
	nilBankKeeper      = types.BankKeeper(nil)
	wasmStorageKey     = sdk.StoreKey(sdk.NewKVStoreKey("wasm")) // need reset by NewKeeper
	wasmMptStorageKey  = sdk.StoreKey(sdk.NewKVStoreKey("mpt"))  //need reset by NewKeeper
	wasmAccountKeeper  = &nilAccountKeeper                       //need reset by NewKeeper
	WasmbankKeeper     = &nilBankKeeper
	wasmGasRegister    = &nilwasmGasRegister
)

func NewSimulateKeeper(
	cdc *codec.CodecProxy,
	paramSpace types.Subspace,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	channelKeeper types.ChannelKeeper,
	portKeeper types.PortKeeper,
	capabilityKeeper types.CapabilityKeeper,
	portSource types.ICS20TransferPortSource,
	router MessageRouter,
	queryRouter GRPCQueryRouter,
	homeDir string,
	wasmConfig types.WasmConfig,
	supportedFeatures string,
	opts ...Option,
) Keeper {
	k := newKeeper(cdc, wasmStorageKey, wasmMptStorageKey, paramSpace, *wasmAccountKeeper, *WasmbankKeeper, channelKeeper, portKeeper, capabilityKeeper, portSource, router, queryRouter, homeDir, wasmConfig, supportedFeatures, watcher.Adapter{}, opts...)
	k.gasRegister = *wasmGasRegister
	return k
}

func newKeeper(cdc *codec.CodecProxy,
	storeKey sdk.StoreKey,
	storageStoreKey sdk.StoreKey,
	paramSpace types.Subspace,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	channelKeeper types.ChannelKeeper,
	portKeeper types.PortKeeper,
	capabilityKeeper types.CapabilityKeeper,
	portSource types.ICS20TransferPortSource,
	router MessageRouter,
	queryRouter GRPCQueryRouter,
	homeDir string,
	wasmConfig types.WasmConfig,
	supportedFeatures string,
	ada types.DBAdapter,
	opts ...Option,
) Keeper {
	wasmer, err := wasmvm.NewVM(filepath.Join(homeDir, "wasm"), supportedFeatures, ContractMemoryLimit, wasmConfig.ContractDebugMode, wasmConfig.MemoryCacheSize)
	if err != nil {
		panic(err)
	}

	keeper := &Keeper{
		storeKey:          storeKey,
		storageStoreKey:   storageStoreKey,
		cdc:               cdc,
		wasmVM:            wasmer,
		accountKeeper:     accountKeeper,
		bank:              NewBankCoinTransferrer(bankKeeper),
		portKeeper:        portKeeper,
		capabilityKeeper:  capabilityKeeper,
		messenger:         NewDefaultMessageHandler(router, channelKeeper, capabilityKeeper, cdc.GetProtocMarshal(), portSource),
		queryGasLimit:     wasmConfig.SmartQueryGasLimit,
		paramSpace:        paramSpace,
		gasRegister:       NewDefaultWasmGasRegister(),
		ada:               ada,
		maxQueryStackSize: types.DefaultMaxQueryStackSize,
	}
	keeper.wasmVMQueryHandler = DefaultQueryPlugins(bankKeeper, channelKeeper, queryRouter, keeper)
	for _, o := range opts {
		o.apply(keeper)
	}
	// not updateable, yet
	keeper.wasmVMResponseHandler = NewDefaultWasmVMContractResponseHandler(NewMessageDispatcher(keeper.messenger, keeper))
	return *keeper
}

func (k Keeper) GetStoreKey() sdk.StoreKey {
	return k.storeKey
}

func (k Keeper) GetStorageStoreKey() sdk.StoreKey {
	return k.storageStoreKey
}

func (k Keeper) IsContractMethodBlocked(ctx sdk.Context, contractAddr, method string) bool {
	blockedMethods := k.GetContractMethodBlockedList(ctx, contractAddr)
	return blockedMethods.IsMethodBlocked(method)
}

func (k Keeper) GetContractMethodBlockedList(ctx sdk.Context, contractAddr string) *types.ContractMethods {
	return k.getContractMethodBlockedList(ctx, contractAddr)
}

func (k Keeper) getAllBlockedList(ctx sdk.Context) []*types.ContractMethods {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	it := sdk.KVStorePrefixIterator(store, types.GetContractMethodBlockedListPrefix(""))
	defer it.Close()

	var cms []*types.ContractMethods
	for ; it.Valid(); it.Next() {
		var method types.ContractMethods
		err := proto.Unmarshal(it.Value(), &method)
		if err != nil {
			panic(err)
		}
		cms = append(cms, &method)
	}
	return cms
}

func (k Keeper) getContractMethodBlockedList(ctx sdk.Context, contractAddr string) *types.ContractMethods {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	key := types.GetContractMethodBlockedListPrefix(contractAddr)
	data := store.Get(key)
	var blockedMethods types.ContractMethods
	err := proto.Unmarshal(data, &blockedMethods)
	if err != nil {
		panic(err)
	}
	return &blockedMethods
}

func (k Keeper) updateContractMethodBlockedList(ctx sdk.Context, blockedMethods *types.ContractMethods, isDelete bool) error {
	oldBlockedMethods := k.getContractMethodBlockedList(ctx, blockedMethods.GetContractAddr())
	if isDelete {
		oldBlockedMethods.DeleteMethods(blockedMethods.Methods)
	} else {
		oldBlockedMethods.AddMethods(blockedMethods.Methods)
	}
	data, err := proto.Marshal(oldBlockedMethods)
	if err != nil {
		return err
	}
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	key := types.GetContractMethodBlockedListPrefix(blockedMethods.ContractAddr)
	store.Set(key, data)
	GetWasmParamsCache().SetNeedBlockedUpdate()
	return nil
}

func (k Keeper) updateUploadAccessConfig(ctx sdk.Context, config types.AccessConfig) {
	params := k.GetParams(ctx)
	params.CodeUploadAccess = config
	k.SetParams(ctx, params)
}

func (k Keeper) getUploadAccessConfig(ctx sdk.Context) types.AccessConfig {
	//var a types.AccessConfig
	//k.paramSpace.Get(ctx, types.ParamStoreKeyUploadAccess, &a)
	//return a
	return k.GetParams(ctx).CodeUploadAccess
}

func (k Keeper) getInstantiateAccessConfig(ctx sdk.Context) types.AccessType {
	//var a types.AccessType
	//k.paramSpace.Get(ctx, types.ParamStoreKeyInstantiateAccess, &a)
	//return a
	return k.GetParams(ctx).InstantiateDefaultPermission
}

// GetParams returns the total set of wasm parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	var params types.Params
	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

func (k Keeper) InvokeExtraProposal(ctx sdk.Context, action string, extra string) error {
	switch action {
	case types.ActionModifyGasFactor:
		return k.modifyGasFactor(ctx, extra)
	}

	return nil
}

// UpdateGasRegister warning, only use it in beginblock
func (k *Keeper) UpdateGasRegister(ctx sdk.Context) {
	if ctx.IsCheckTx() || ctx.IsTraceTx() {
		return
	}

	gasFactor := k.GetGasFactor(ctx)
	if gasFactor != k.gasRegister.GetGasMultiplier() {
		k.gasRegister.UpdateGasMultiplier(gasFactor)
	}
	return
}

func (k *Keeper) modifyGasFactor(ctx sdk.Context, extra string) error {
	result, err := types.NewActionModifyGasFactor(extra)
	if err != nil {
		return err
	}

	value := result.MulInt64(int64(BaseGasMultiplier)).TruncateInt64()
	if value <= 0 {
		return types.ErrCodeInvalidGasFactor
	}
	k.SetGasFactor(ctx, uint64(value))
	return nil
}

// get the gas factor
func (k Keeper) GetGasFactor(ctx sdk.Context) uint64 {
	store := k.ada.NewStore(ctx, k.storeKey, nil)

	if !store.Has(types.KeyGasFactorPrefix) {
		return DefaultGasMultiplier
	}

	b := store.Get(types.KeyGasFactorPrefix)
	if b != nil {
		return sdk.BigEndianToUint64(b)
	}
	return DefaultGasMultiplier
}

// set the gas factor
func (k Keeper) SetGasFactor(ctx sdk.Context, factor uint64) {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	store.Set(types.KeyGasFactorPrefix, sdk.Uint64ToBigEndian(factor))
}

func (k Keeper) SetParams(ctx sdk.Context, ps types.Params) {
	watcher.SetParams(ps)
	k.paramSpace.SetParamSet(ctx, &ps)
	GetWasmParamsCache().SetNeedParamsUpdate()
}

func (k Keeper) OnAccountUpdated(acc exported.Account) {
	watcher.DeleteAccount(sdk.AccToAWasmddress(acc.GetAddress()))
}

func (k Keeper) create(ctx sdk.Context, creator sdk.WasmAddress, wasmCode []byte, instantiateAccess *types.AccessConfig, authZ AuthorizationPolicy) (codeID uint64, err error) {
	if creator == nil {
		return 0, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "cannot be nil")
	}

	if !authZ.CanCreateCode(k.getUploadAccessConfig(ctx), creator) {
		return 0, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, types.GenerateUnauthorizeError(k.GetParams(ctx).CodeUploadAccess.Permission))
	}
	// figure out proper instantiate access
	defaultAccessConfig := k.getInstantiateAccessConfig(ctx).With(creator)
	if instantiateAccess == nil {
		instantiateAccess = &defaultAccessConfig
	} else if !instantiateAccess.IsSubset(defaultAccessConfig) {
		// we enforce this must be subset of default upload access
		return 0, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "instantiate access must be subset of default upload access")
	}

	wasmCode, err = ioutils.Uncompress(wasmCode, uint64(types.MaxWasmSize))
	if err != nil {
		return 0, sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}
	ctx.GasMeter().ConsumeGas(k.gasRegister.CompileCosts(len(wasmCode)), "Compiling WASM Bytecode")

	checksum, err := k.wasmVM.Create(wasmCode)
	if err != nil {
		return 0, sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}
	report, err := k.wasmVM.AnalyzeCode(checksum)
	if err != nil {
		return 0, sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}
	codeID = k.autoIncrementID(ctx, types.KeyLastCodeID)
	k.Logger(ctx).Debug("storing new contract", "features", report.RequiredFeatures, "code_id", codeID)
	result, err := types.ConvertAccessConfig(*instantiateAccess)
	if err != nil {
		return 0, sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}
	codeInfo := types.NewCodeInfo(checksum, creator, result)
	k.storeCodeInfo(ctx, codeID, codeInfo)

	evt := sdk.NewEvent(
		types.EventTypeStoreCode,
		sdk.NewAttribute(types.AttributeKeyCodeID, strconv.FormatUint(codeID, 10)),
	)
	for _, f := range strings.Split(report.RequiredFeatures, ",") {
		evt.AppendAttributes(sdk.NewAttribute(types.AttributeKeyFeature, strings.TrimSpace(f)))
	}
	ctx.EventManager().EmitEvent(evt)

	return codeID, nil
}

func (k Keeper) storeCodeInfo(ctx sdk.Context, codeID uint64, codeInfo types.CodeInfo) {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	// 0x01 | codeID (uint64) -> ContractInfo
	store.Set(types.GetCodeKey(codeID), k.cdc.GetProtocMarshal().MustMarshal(&codeInfo))
}

func (k Keeper) importCode(ctx sdk.Context, codeID uint64, codeInfo types.CodeInfo, wasmCode []byte) error {
	wasmCode, err := ioutils.Uncompress(wasmCode, uint64(types.MaxWasmSize))
	if err != nil {
		return sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}
	newCodeHash, err := k.wasmVM.Create(wasmCode)
	if err != nil {
		return sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}
	if !bytes.Equal(codeInfo.CodeHash, newCodeHash) {
		return sdkerrors.Wrap(types.ErrInvalid, "code hashes not same")
	}

	store := k.ada.NewStore(ctx, k.storeKey, nil)

	key := types.GetCodeKey(codeID)
	if store.Has(key) {
		return sdkerrors.Wrapf(types.ErrDuplicate, "duplicate code: %d", codeID)
	}
	// 0x01 | codeID (uint64) -> ContractInfo
	store.Set(key, k.cdc.GetProtocMarshal().MustMarshal(&codeInfo))
	return nil
}

func (k Keeper) instantiate(ctx sdk.Context, codeID uint64, creator, admin sdk.WasmAddress, initMsg []byte, label string, deposit sdk.Coins, authZ AuthorizationPolicy) (sdk.WasmAddress, []byte, error) {
	//defer telemetry.MeasureSince(time.Now(), "wasm", "contract", "instantiate")
	// This method does not support parallel execution.
	if ctx.ParaMsg() != nil {
		ctx.ParaMsg().InvalidExecute = true
	}
	fmt.Printf("-------instantiate -0--%d\n", ctx.GasMeter().GasConsumed())
	instanceCosts := k.gasRegister.NewContractInstanceCosts(k.IsPinnedCode(ctx, codeID), len(initMsg))
	ctx.GasMeter().ConsumeGas(instanceCosts, "Loading CosmWasm module: instantiate")
	fmt.Printf("-------instantiate -1--%d\n", ctx.GasMeter().GasConsumed())
	// create contract address
	contractAddress := k.generateContractAddress(ctx, codeID)
	existingAcct := k.accountKeeper.GetAccount(ctx, sdk.WasmToAccAddress(contractAddress))
	if existingAcct != nil {
		return nil, nil, sdkerrors.Wrap(types.ErrAccountExists, existingAcct.GetAddress().String())
	}
	fmt.Printf("-------instantiate -2--%d\n", ctx.GasMeter().GasConsumed())

	// deposit initial contract funds
	if !deposit.IsZero() {
		if err := k.bank.TransferCoins(ctx, creator, contractAddress, deposit); err != nil {
			return nil, nil, err
		}
	} else {
		// create an empty account (so we don't have issues later)
		// TODO: can we remove this?
		contractAccount := k.accountKeeper.NewAccountWithAddress(ctx, sdk.WasmToAccAddress(contractAddress))
		k.accountKeeper.SetAccount(ctx, contractAccount)
	}
	fmt.Printf("-------instantiate -3--%d\n", ctx.GasMeter().GasConsumed())

	// get contact info
	codeInfo := k.GetCodeInfo(ctx, codeID)
	if codeInfo == nil {
		return nil, nil, sdkerrors.Wrap(types.ErrNotFound, "code")
	}
	fmt.Printf("-------instantiate -4--%d\n", ctx.GasMeter().GasConsumed())
	if !authZ.CanInstantiateContract(codeInfo.InstantiateConfig, creator) {
		return nil, nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "can not instantiate")
	}
	fmt.Printf("-------instantiate -5--%d\n", ctx.GasMeter().GasConsumed())
	// prepare params for contract instantiate call
	env := types.NewEnv(ctx, contractAddress)
	adapters := sdk.CoinsToCoinAdapters(deposit)
	info := types.NewInfo(creator, adapters)

	// create prefixed data store
	// 0x00 | BuildContractAddress (sdk.WasmAddress) | stateRoot
	prefixStore := k.getStorageStore(ctx, contractAddress)
	prefixStoreAdapter := types.NewStoreAdapter(prefixStore)
	fmt.Printf("-------instantiate -6--%d\n", ctx.GasMeter().GasConsumed())
	// prepare querier
	querier := k.newQueryHandler(ctx, contractAddress)

	// instantiate wasm contract
	gas := k.runtimeGasForContract(ctx)
	fmt.Printf("-------instantiate -7--%d\n", ctx.GasMeter().GasConsumed())
	res, gasUsed, err := k.wasmVM.Instantiate(codeInfo.CodeHash, env, info, initMsg, prefixStoreAdapter, cosmwasmAPI, querier, k.gasMeter(ctx), gas, costJSONDeserialization)
	fmt.Printf("-------instantiate -8--%d, gasUsed:%d\n", ctx.GasMeter().GasConsumed(), gasUsed)
	k.consumeRuntimeGas(ctx, gasUsed)
	if !ctx.IsCheckTx() && k.innertxKeeper != nil {
		k.innertxKeeper.UpdateWasmInnerTx(ctx.TxBytes(), ctx.BlockHeight(), innertx.CosmosDepth, creator, contractAddress, innertx.CosmosCallType, types.InstantiateInnertxName, sdk.Coins{}, err, k.gasRegister.FromWasmVMGas(gasUsed), string(initMsg))
	}
	if err != nil {
		return nil, nil, sdkerrors.Wrap(types.ErrInstantiateFailed, err.Error())
	}
	fmt.Printf("-------instantiate -9--%d\n", ctx.GasMeter().GasConsumed())
	// persist instance first
	createdAt := types.NewAbsoluteTxPosition(ctx)
	contractInfo := types.NewContractInfo(codeID, creator, admin, label, createdAt)
	fmt.Printf("-------instantiate -10--%d\n", ctx.GasMeter().GasConsumed())
	// check for IBC flag
	report, err := k.wasmVM.AnalyzeCode(codeInfo.CodeHash)
	if err != nil {
		return nil, nil, sdkerrors.Wrap(types.ErrInstantiateFailed, err.Error())
	}
	if report.HasIBCEntryPoints {
		// register IBC port
		ibcPort, err := k.ensureIbcPort(ctx, contractAddress)
		if err != nil {
			return nil, nil, err
		}
		contractInfo.IBCPortID = ibcPort
	}

	// store contract before dispatch so that contract could be called back
	historyEntry := contractInfo.InitialHistory(initMsg)
	k.addToContractCodeSecondaryIndex(ctx, contractAddress, historyEntry)
	k.appendToContractHistory(ctx, contractAddress, historyEntry)
	k.storeContractInfo(ctx, contractAddress, &contractInfo)
	fmt.Printf("-------instantiate -11--%d\n", ctx.GasMeter().GasConsumed())

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeInstantiate,
		sdk.NewAttribute(types.AttributeKeyContractAddr, contractAddress.String()),
		sdk.NewAttribute(types.AttributeKeyCodeID, strconv.FormatUint(codeID, 10)),
	))
	fmt.Printf("-------instantiate -12--%d\n", ctx.GasMeter().GasConsumed())
	data, err := k.handleContractResponse(ctx, contractAddress, contractInfo.IBCPortID, res.Messages, res.Attributes, res.Data, res.Events)
	if err != nil {
		return nil, nil, sdkerrors.Wrap(err, "dispatch")
	}
	fmt.Printf("-------instantiate -13--%d\n", ctx.GasMeter().GasConsumed())
	return contractAddress, data, nil
}

// Execute executes the contract instance
func (k Keeper) execute(ctx sdk.Context, contractAddress sdk.WasmAddress, caller sdk.WasmAddress, msg []byte, coins sdk.Coins) ([]byte, error) {
	//defer telemetry.MeasureSince(time.Now(), "wasm", "contract", "execute")
	contractInfo, codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddress)
	if err != nil {
		return nil, err
	}

	executeCosts := k.gasRegister.InstantiateContractCosts(k.IsPinnedCode(ctx, contractInfo.CodeID), len(msg))
	ctx.GasMeter().ConsumeGas(executeCosts, "Loading CosmWasm module: execute")

	// add more funds
	if !coins.IsZero() {
		if err := k.bank.TransferCoins(ctx, caller, contractAddress, coins); err != nil {
			return nil, err
		}
	}

	env := types.NewEnv(ctx, contractAddress)
	adapters := sdk.CoinsToCoinAdapters(coins)
	info := types.NewInfo(caller, adapters)

	// prepare querier
	querier := k.newQueryHandler(ctx, contractAddress)
	gas := k.runtimeGasForContract(ctx)
	if k.GetParams(ctx).UseContractBlockedList {
		var methodsMap map[string]interface{}
		err = json.Unmarshal(msg, &methodsMap)
		if err != nil {
			return nil, err
		}
		for method := range methodsMap {
			if k.IsContractMethodBlocked(ctx, contractAddress.String(), method) {
				return nil, sdkerrors.Wrap(types.ErrExecuteFailed, fmt.Sprintf("%s method of contract %s is not allowed", contractAddress.String(), method))
			}
		}
	}

	res, gasUsed, execErr := k.wasmVM.Execute(codeInfo.CodeHash, env, info, msg, prefixStore, cosmwasmAPI, querier, k.gasMeter(ctx), gas, costJSONDeserialization)
	k.consumeRuntimeGas(ctx, gasUsed)
	if !ctx.IsCheckTx() && k.innertxKeeper != nil {
		k.innertxKeeper.UpdateWasmInnerTx(ctx.TxBytes(), ctx.BlockHeight(), innertx.CosmosDepth, caller, contractAddress, innertx.CosmosCallType, types.ExecuteInnertxName, coins, err, k.gasRegister.FromWasmVMGas(gasUsed), string(msg))
	}
	if execErr != nil {
		return nil, sdkerrors.Wrap(types.ErrExecuteFailed, execErr.Error())
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeExecute,
		sdk.NewAttribute(types.AttributeKeyContractAddr, contractAddress.String()),
	))

	data, err := k.handleContractResponse(ctx, contractAddress, contractInfo.IBCPortID, res.Messages, res.Attributes, res.Data, res.Events)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "dispatch")
	}

	return data, nil
}

func (k Keeper) migrate(ctx sdk.Context, contractAddress sdk.WasmAddress, caller sdk.WasmAddress, newCodeID uint64, msg []byte, authZ AuthorizationPolicy) ([]byte, error) {
	//defer telemetry.MeasureSince(time.Now(), "wasm", "contract", "migrate")
	// This method does not support parallel execution.
	if ctx.ParaMsg() != nil {
		ctx.ParaMsg().InvalidExecute = true
	}
	migrateSetupCosts := k.gasRegister.InstantiateContractCosts(k.IsPinnedCode(ctx, newCodeID), len(msg))
	ctx.GasMeter().ConsumeGas(migrateSetupCosts, "Loading CosmWasm module: migrate")

	contractInfo := k.GetContractInfo(ctx, contractAddress)
	if contractInfo == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "unknown contract")
	}
	if !authZ.CanModifyContract(contractInfo.AdminAddr(), caller) {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "can not migrate")
	}

	newCodeInfo := k.GetCodeInfo(ctx, newCodeID)
	if newCodeInfo == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "unknown code")
	}

	// check for IBC flag
	switch report, err := k.wasmVM.AnalyzeCode(newCodeInfo.CodeHash); {
	case err != nil:
		return nil, sdkerrors.Wrap(types.ErrMigrationFailed, err.Error())
	case !report.HasIBCEntryPoints && contractInfo.IBCPortID != "":
		// prevent update to non ibc contract
		return nil, sdkerrors.Wrap(types.ErrMigrationFailed, "requires ibc callbacks")
	case report.HasIBCEntryPoints && contractInfo.IBCPortID == "":
		// add ibc port
		ibcPort, err := k.ensureIbcPort(ctx, contractAddress)
		if err != nil {
			return nil, err
		}
		contractInfo.IBCPortID = ibcPort
	}

	env := types.NewEnv(ctx, contractAddress)

	// prepare querier
	querier := k.newQueryHandler(ctx, contractAddress)

	prefixStore := k.getStorageStore(ctx, contractAddress)

	prefixAdapater := types.NewStoreAdapter(prefixStore)

	gas := k.runtimeGasForContract(ctx)
	res, gasUsed, err := k.wasmVM.Migrate(newCodeInfo.CodeHash, env, msg, &prefixAdapater, cosmwasmAPI, &querier, k.gasMeter(ctx), gas, costJSONDeserialization)
	k.consumeRuntimeGas(ctx, gasUsed)
	if !ctx.IsCheckTx() && k.innertxKeeper != nil {
		k.innertxKeeper.UpdateWasmInnerTx(ctx.TxBytes(), ctx.BlockHeight(), innertx.CosmosDepth, caller, contractAddress, innertx.CosmosCallType, types.MigrateInnertxName, sdk.Coins{}, err, k.gasRegister.FromWasmVMGas(gasUsed), string(msg))
	}
	if err != nil {
		return nil, sdkerrors.Wrap(types.ErrMigrationFailed, err.Error())
	}

	// delete old secondary index entry
	k.removeFromContractCodeSecondaryIndex(ctx, contractAddress, k.getLastContractHistoryEntry(ctx, contractAddress))
	// persist migration updates
	historyEntry := contractInfo.AddMigration(ctx, newCodeID, msg)
	k.appendToContractHistory(ctx, contractAddress, historyEntry)
	k.addToContractCodeSecondaryIndex(ctx, contractAddress, historyEntry)
	k.storeContractInfo(ctx, contractAddress, contractInfo)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeMigrate,
		sdk.NewAttribute(types.AttributeKeyCodeID, strconv.FormatUint(newCodeID, 10)),
		sdk.NewAttribute(types.AttributeKeyContractAddr, contractAddress.String()),
	))

	data, err := k.handleContractResponse(ctx, contractAddress, contractInfo.IBCPortID, res.Messages, res.Attributes, res.Data, res.Events)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "dispatch")
	}

	return data, nil
}

// Sudo allows priviledged access to a contract. This can never be called by an external tx, but only by
// another native Go module directly, or on-chain governance (if sudo proposals are enabled). Thus, the keeper doesn't
// place any access controls on it, that is the responsibility or the app developer (who passes the wasm.Keeper in app.go)
func (k Keeper) Sudo(ctx sdk.Context, contractAddress sdk.WasmAddress, msg []byte) ([]byte, error) {
	//defer telemetry.MeasureSince(time.Now(), "wasm", "contract", "sudo")
	contractInfo, codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddress)
	if err != nil {
		return nil, err
	}

	sudoSetupCosts := k.gasRegister.InstantiateContractCosts(k.IsPinnedCode(ctx, contractInfo.CodeID), len(msg))
	ctx.GasMeter().ConsumeGas(sudoSetupCosts, "Loading CosmWasm module: sudo")

	env := types.NewEnv(ctx, contractAddress)

	// prepare querier
	querier := k.newQueryHandler(ctx, contractAddress)
	gas := k.runtimeGasForContract(ctx)
	res, gasUsed, execErr := k.wasmVM.Sudo(codeInfo.CodeHash, env, msg, prefixStore, cosmwasmAPI, querier, k.gasMeter(ctx), gas, costJSONDeserialization)
	k.consumeRuntimeGas(ctx, gasUsed)
	if execErr != nil {
		return nil, sdkerrors.Wrap(types.ErrExecuteFailed, execErr.Error())
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSudo,
		sdk.NewAttribute(types.AttributeKeyContractAddr, contractAddress.String()),
	))

	data, err := k.handleContractResponse(ctx, contractAddress, contractInfo.IBCPortID, res.Messages, res.Attributes, res.Data, res.Events)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "dispatch")
	}

	return data, nil
}

// reply is only called from keeper internal functions (dispatchSubmessages) after processing the submessage
func (k Keeper) reply(ctx sdk.Context, contractAddress sdk.WasmAddress, reply wasmvmtypes.Reply) ([]byte, error) {
	contractInfo, codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddress)
	if err != nil {
		return nil, err
	}

	// always consider this pinned
	replyCosts := k.gasRegister.ReplyCosts(true, reply)
	ctx.GasMeter().ConsumeGas(replyCosts, "Loading CosmWasm module: reply")

	env := types.NewEnv(ctx, contractAddress)

	// prepare querier
	querier := k.newQueryHandler(ctx, contractAddress)
	gas := k.runtimeGasForContract(ctx)
	res, gasUsed, execErr := k.wasmVM.Reply(codeInfo.CodeHash, env, reply, prefixStore, cosmwasmAPI, querier, k.gasMeter(ctx), gas, costJSONDeserialization)
	k.consumeRuntimeGas(ctx, gasUsed)
	if execErr != nil {
		return nil, sdkerrors.Wrap(types.ErrExecuteFailed, execErr.Error())
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeReply,
		sdk.NewAttribute(types.AttributeKeyContractAddr, contractAddress.String()),
	))

	data, err := k.handleContractResponse(ctx, contractAddress, contractInfo.IBCPortID, res.Messages, res.Attributes, res.Data, res.Events)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "dispatch")
	}

	return data, nil
}

// addToContractCodeSecondaryIndex adds element to the index for contracts-by-codeid queries
func (k Keeper) addToContractCodeSecondaryIndex(ctx sdk.Context, contractAddress sdk.WasmAddress, entry types.ContractCodeHistoryEntry) {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	store.Set(types.GetContractByCreatedSecondaryIndexKey(contractAddress, entry), []byte{})
}

// removeFromContractCodeSecondaryIndex removes element to the index for contracts-by-codeid queries
func (k Keeper) removeFromContractCodeSecondaryIndex(ctx sdk.Context, contractAddress sdk.WasmAddress, entry types.ContractCodeHistoryEntry) {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	store.Delete(types.GetContractByCreatedSecondaryIndexKey(contractAddress, entry))
}

// IterateContractsByCode iterates over all contracts with given codeID ASC on code update time.
func (k Keeper) IterateContractsByCode(ctx sdk.Context, codeID uint64, cb func(address sdk.WasmAddress) bool) {
	prefixStore := k.ada.NewStore(ctx, k.storeKey, types.GetContractByCodeIDSecondaryIndexPrefix(codeID))
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if cb(key[types.AbsoluteTxPositionLen:]) {
			return
		}
	}
}

func (k Keeper) setContractAdmin(ctx sdk.Context, contractAddress, caller, newAdmin sdk.WasmAddress, authZ AuthorizationPolicy) (err error) {
	// This method does not support parallel execution.
	if ctx.ParaMsg() != nil {
		ctx.ParaMsg().InvalidExecute = true
	}
	gas := ctx.GasMeter().GasConsumed()
	defer func() {
		if !ctx.IsCheckTx() && k.innertxKeeper != nil {
			k.innertxKeeper.UpdateWasmInnerTx(ctx.TxBytes(), ctx.BlockHeight(), innertx.CosmosDepth, caller, contractAddress, innertx.CosmosCallType, types.SetContractAdminInnertxName, sdk.Coins{}, err, ctx.GasMeter().GasConsumed()-gas, "")
		}
	}()
	contractInfo := k.GetContractInfo(ctx, contractAddress)
	if contractInfo == nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "unknown contract")
	}
	if !authZ.CanModifyContract(contractInfo.AdminAddr(), caller) {
		return sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "can not modify contract")
	}
	contractInfo.Admin = newAdmin.String()
	k.storeContractInfo(ctx, contractAddress, contractInfo)
	return nil
}

func (k Keeper) appendToContractHistory(ctx sdk.Context, contractAddr sdk.WasmAddress, newEntries ...types.ContractCodeHistoryEntry) {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	// find last element position
	var pos uint64
	prefixStore := prefix.NewStore(store, types.GetContractCodeHistoryElementPrefix(contractAddr))
	iter := prefixStore.ReverseIterator(nil, nil)
	defer iter.Close()

	if iter.Valid() {
		pos = sdk.BigEndianToUint64(iter.Key())
	}
	// then store with incrementing position
	for _, e := range newEntries {
		pos++
		key := types.GetContractCodeHistoryElementKey(contractAddr, pos)
		store.Set(key, k.cdc.GetProtocMarshal().MustMarshal(&e)) //nolint:gosec
	}
}

func (k Keeper) GetContractHistory(ctx sdk.Context, contractAddr sdk.WasmAddress) []types.ContractCodeHistoryEntry {
	store := k.ada.NewStore(ctx, k.storeKey, types.GetContractCodeHistoryElementPrefix(contractAddr))

	r := make([]types.ContractCodeHistoryEntry, 0)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var e types.ContractCodeHistoryEntry
		k.cdc.GetProtocMarshal().MustUnmarshal(iter.Value(), &e)
		r = append(r, e)
	}
	return r
}

// getLastContractHistoryEntry returns the last element from history. To be used internally only as it panics when none exists
func (k Keeper) getLastContractHistoryEntry(ctx sdk.Context, contractAddr sdk.WasmAddress) types.ContractCodeHistoryEntry {
	store := k.ada.NewStore(ctx, k.storeKey, types.GetContractCodeHistoryElementPrefix(contractAddr))
	iter := store.ReverseIterator(nil, nil)
	defer iter.Close()

	var r types.ContractCodeHistoryEntry
	if !iter.Valid() {
		// all contracts have a history
		panic(fmt.Sprintf("no history for %s", contractAddr.String()))
	}
	k.cdc.GetProtocMarshal().MustUnmarshal(iter.Value(), &r)
	return r
}

// QuerySmart queries the smart contract itself.
func (k Keeper) QuerySmart(ctx sdk.Context, contractAddr sdk.WasmAddress, req []byte) ([]byte, error) {
	//defer telemetry.MeasureSince(time.Now(), "wasm", "contract", "query-smart")

	// checks and increase query stack size
	ctx, err := checkAndIncreaseQueryStackSize(ctx, k.maxQueryStackSize)
	if err != nil {
		return nil, err
	}

	contractInfo, codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return nil, err
	}

	smartQuerySetupCosts := k.gasRegister.InstantiateContractCosts(k.IsPinnedCode(ctx, contractInfo.CodeID), len(req))
	ctx.GasMeter().ConsumeGas(smartQuerySetupCosts, "Loading CosmWasm module: query")

	// prepare querier
	querier := k.newQueryHandler(ctx, contractAddr)

	env := types.NewEnv(ctx, contractAddr)
	queryResult, gasUsed, qErr := k.wasmVM.Query(codeInfo.CodeHash, env, req, prefixStore, cosmwasmAPI, querier, k.gasMeter(ctx), k.runtimeGasForContract(ctx), costJSONDeserialization)
	k.consumeRuntimeGas(ctx, gasUsed)
	if qErr != nil {
		return nil, sdkerrors.Wrap(types.ErrQueryFailed, qErr.Error())
	}
	return queryResult, nil
}

func checkAndIncreaseQueryStackSize(ctx sdk.Context, maxQueryStackSize uint32) (sdk.Context, error) {
	var queryStackSize uint32

	// read current value
	if size := ctx.Context().Value(contextKeyQueryStackSize); size != nil {
		queryStackSize = size.(uint32)
	} else {
		queryStackSize = 0
	}

	// increase
	queryStackSize++

	// did we go too far?
	if queryStackSize > maxQueryStackSize {
		return ctx, types.ErrExceedMaxQueryStackSize
	}

	// set updated stack size
	contextCtx := context.WithValue(ctx.Context(), contextKeyQueryStackSize, queryStackSize)
	ctx.SetContext(contextCtx)

	return ctx, nil
}

// QueryRaw returns the contract's state for give key. Returns `nil` when key is `nil`.
func (k Keeper) QueryRaw(ctx sdk.Context, contractAddress sdk.WasmAddress, key []byte) []byte {
	//defer telemetry.MeasureSince(time.Now(), "wasm", "contract", "query-raw")
	if key == nil {
		return nil
	}
	prefixStore := k.getStorageStore(ctx, contractAddress)

	return prefixStore.Get(key)
}

func (k Keeper) contractInstance(ctx sdk.Context, contractAddress sdk.WasmAddress) (types.ContractInfo, types.CodeInfo, types.StoreAdapter, error) {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	contractBz := store.Get(types.GetContractAddressKey(contractAddress))
	if contractBz == nil {
		return types.ContractInfo{}, types.CodeInfo{}, types.StoreAdapter{}, sdkerrors.Wrap(types.ErrNotFound, "contract")
	}
	var contractInfo types.ContractInfo
	k.cdc.GetProtocMarshal().MustUnmarshal(contractBz, &contractInfo)

	codeInfoBz := store.Get(types.GetCodeKey(contractInfo.CodeID))
	if codeInfoBz == nil {
		return contractInfo, types.CodeInfo{}, types.StoreAdapter{}, sdkerrors.Wrap(types.ErrNotFound, "code info")
	}
	var codeInfo types.CodeInfo
	k.cdc.GetProtocMarshal().MustUnmarshal(codeInfoBz, &codeInfo)
	prefixStore := k.getStorageStore(ctx, contractAddress)

	return contractInfo, codeInfo, types.NewStoreAdapter(prefixStore), nil
}

func (k Keeper) GetContractInfo(ctx sdk.Context, contractAddress sdk.WasmAddress) *types.ContractInfo {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	var contract types.ContractInfo
	contractBz := store.Get(types.GetContractAddressKey(contractAddress))
	if contractBz == nil {
		return nil
	}
	k.cdc.GetProtocMarshal().MustUnmarshal(contractBz, &contract)
	return &contract
}

func (k Keeper) HasContractInfo(ctx sdk.Context, contractAddress sdk.WasmAddress) bool {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	return store.Has(types.GetContractAddressKey(contractAddress))
}

// storeContractInfo persists the ContractInfo. No secondary index updated here.
func (k Keeper) storeContractInfo(ctx sdk.Context, contractAddress sdk.WasmAddress, contract *types.ContractInfo) {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	store.Set(types.GetContractAddressKey(contractAddress), k.cdc.GetProtocMarshal().MustMarshal(contract))
}

func (k Keeper) IterateContractInfo(ctx sdk.Context, cb func(sdk.WasmAddress, types.ContractInfo) bool) {
	prefixStore := k.ada.NewStore(ctx, k.storeKey, types.ContractKeyPrefix)
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var contract types.ContractInfo
		k.cdc.GetProtocMarshal().MustUnmarshal(iter.Value(), &contract)
		// cb returns true to stop early
		if cb(iter.Key(), contract) {
			break
		}
	}
}

// IterateContractState iterates through all elements of the key value store for the given contract address and passes
// them to the provided callback function. The callback method can return true to abort early.
func (k Keeper) IterateContractState(ctx sdk.Context, contractAddress sdk.WasmAddress, cb func(key, value []byte) bool) {
	prefixStore := k.getStorageStore(ctx, contractAddress)

	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		if cb(iter.Key(), iter.Value()) {
			break
		}
	}
}

func (k Keeper) importContractState(ctx sdk.Context, contractAddress sdk.WasmAddress, models []types.Model) error {
	prefixStore := k.getStorageStore(ctx, contractAddress)
	for _, model := range models {
		if model.Value == nil {
			model.Value = []byte{}
		}
		if prefixStore.Has(model.Key) {
			return sdkerrors.Wrapf(types.ErrDuplicate, "duplicate key: %x", model.Key)
		}
		prefixStore.Set(model.Key, model.Value)
	}
	return nil
}

func (k Keeper) GetCodeInfo(ctx sdk.Context, codeID uint64) *types.CodeInfo {

	store := k.ada.NewStore(ctx, k.storeKey, nil)
	var codeInfo types.CodeInfo
	codeInfoBz := store.Get(types.GetCodeKey(codeID))
	if codeInfoBz == nil {
		return nil
	}
	k.cdc.GetProtocMarshal().MustUnmarshal(codeInfoBz, &codeInfo)
	return &codeInfo
}

func (k Keeper) containsCodeInfo(ctx sdk.Context, codeID uint64) bool {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	return store.Has(types.GetCodeKey(codeID))
}

func (k Keeper) IterateCodeInfos(ctx sdk.Context, cb func(uint64, types.CodeInfo) bool) {
	prefixStore := k.ada.NewStore(ctx, k.storeKey, types.CodeKeyPrefix)

	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var c types.CodeInfo
		k.cdc.GetProtocMarshal().MustUnmarshal(iter.Value(), &c)
		// cb returns true to stop early
		if cb(binary.BigEndian.Uint64(iter.Key()), c) {
			return
		}
	}
}

func (k Keeper) GetByteCode(ctx sdk.Context, codeID uint64) ([]byte, error) {
	store := k.ada.NewStore(ctx, k.storeKey, nil)

	var codeInfo types.CodeInfo
	codeInfoBz := store.Get(types.GetCodeKey(codeID))
	if codeInfoBz == nil {
		return nil, nil
	}
	k.cdc.GetProtocMarshal().MustUnmarshal(codeInfoBz, &codeInfo)
	return k.wasmVM.GetCode(codeInfo.CodeHash)
}

// PinCode pins the wasm contract in wasmvm cache
func (k Keeper) pinCode(ctx sdk.Context, codeID uint64) error {
	codeInfo := k.GetCodeInfo(ctx, codeID)
	if codeInfo == nil {
		return sdkerrors.Wrap(types.ErrNotFound, "code info")
	}

	if err := k.wasmVM.Pin(codeInfo.CodeHash); err != nil {
		return sdkerrors.Wrap(types.ErrPinContractFailed, err.Error())
	}
	store := k.ada.NewStore(ctx, k.storeKey, nil)

	// store 1 byte to not run into `nil` debugging issues
	store.Set(types.GetPinnedCodeIndexPrefix(codeID), []byte{1})

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypePinCode,
		sdk.NewAttribute(types.AttributeKeyCodeID, strconv.FormatUint(codeID, 10)),
	))
	return nil
}

// UnpinCode removes the wasm contract from wasmvm cache
func (k Keeper) unpinCode(ctx sdk.Context, codeID uint64) error {
	codeInfo := k.GetCodeInfo(ctx, codeID)
	if codeInfo == nil {
		return sdkerrors.Wrap(types.ErrNotFound, "code info")
	}
	if err := k.wasmVM.Unpin(codeInfo.CodeHash); err != nil {
		return sdkerrors.Wrap(types.ErrUnpinContractFailed, err.Error())
	}

	store := k.ada.NewStore(ctx, k.storeKey, nil)

	store.Delete(types.GetPinnedCodeIndexPrefix(codeID))

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUnpinCode,
		sdk.NewAttribute(types.AttributeKeyCodeID, strconv.FormatUint(codeID, 10)),
	))
	return nil
}

// IsPinnedCode returns true when codeID is pinned in wasmvm cache
func (k Keeper) IsPinnedCode(ctx sdk.Context, codeID uint64) bool {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	return store.Has(types.GetPinnedCodeIndexPrefix(codeID))
}

// InitializePinnedCodes updates wasmvm to pin to cache all contracts marked as pinned
func (k Keeper) InitializePinnedCodes(ctx sdk.Context) error {
	store := k.ada.NewStore(ctx, k.storeKey, types.PinnedCodeIndexPrefix)

	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		codeInfo := k.GetCodeInfo(ctx, types.ParsePinnedCodeIndex(iter.Key()))
		if codeInfo == nil {
			return sdkerrors.Wrap(types.ErrNotFound, "code info")
		}
		if err := k.wasmVM.Pin(codeInfo.CodeHash); err != nil {
			return sdkerrors.Wrap(types.ErrPinContractFailed, err.Error())
		}
	}
	return nil
}

// setContractInfoExtension updates the extension point data that is stored with the contract info
func (k Keeper) setContractInfoExtension(ctx sdk.Context, contractAddr sdk.WasmAddress, ext types.ContractInfoExtension) error {
	info := k.GetContractInfo(ctx, contractAddr)
	if info == nil {
		return sdkerrors.Wrap(types.ErrNotFound, "contract info")
	}
	if err := info.SetExtension(ext); err != nil {
		return err
	}
	k.storeContractInfo(ctx, contractAddr, info)
	return nil
}

// setAccessConfig updates the access config of a code id.
func (k Keeper) setAccessConfig(ctx sdk.Context, codeID uint64, config types.AccessConfig) error {
	info := k.GetCodeInfo(ctx, codeID)
	if info == nil {
		return sdkerrors.Wrap(types.ErrNotFound, "code info")
	}
	info.InstantiateConfig = config
	k.storeCodeInfo(ctx, codeID, *info)
	return nil
}

// handleContractResponse processes the contract response data by emitting events and sending sub-/messages.
func (k *Keeper) handleContractResponse(
	ctx sdk.Context,
	contractAddr sdk.WasmAddress,
	ibcPort string,
	msgs []wasmvmtypes.SubMsg,
	attrs []wasmvmtypes.EventAttribute,
	data []byte,
	evts wasmvmtypes.Events,
) ([]byte, error) {
	attributeGasCost := k.gasRegister.EventCosts(attrs, evts)
	ctx.GasMeter().ConsumeGas(attributeGasCost, "Custom contract event attributes")
	// emit all events from this contract itself
	if len(attrs) != 0 {
		wasmEvents, err := newWasmModuleEvent(attrs, contractAddr)
		if err != nil {
			return nil, err
		}
		ctx.EventManager().EmitEvents(wasmEvents)
	}
	if len(evts) > 0 {
		customEvents, err := newCustomEvents(evts, contractAddr)
		if err != nil {
			return nil, err
		}
		ctx.EventManager().EmitEvents(customEvents)
	}
	return k.wasmVMResponseHandler.Handle(ctx, contractAddr, ibcPort, msgs, data)
}

func (k Keeper) runtimeGasForContract(ctx sdk.Context) uint64 {
	meter := ctx.GasMeter()
	if meter.IsOutOfGas() {
		return 0
	}
	if meter.Limit() == 0 { // infinite gas meter with limit=0 and not out of gas
		return math.MaxUint64
	}
	return k.gasRegister.ToWasmVMGas(meter.Limit() - meter.GasConsumedToLimit())
}

func (k Keeper) consumeRuntimeGas(ctx sdk.Context, gas uint64) {
	consumed := k.gasRegister.FromWasmVMGas(gas)
	ctx.GasMeter().ConsumeGas(consumed, "wasm contract")
	// throw OutOfGas error if we ran out (got exactly to zero due to better limit enforcing)
	if ctx.GasMeter().IsOutOfGas() {
		panic(sdk.ErrorOutOfGas{Descriptor: "Wasmer function execution"})
	}
}

// generates a contract address from codeID + instanceID
func (k Keeper) generateContractAddress(ctx sdk.Context, codeID uint64) sdk.WasmAddress {
	instanceID := k.autoIncrementID(ctx, types.KeyLastInstanceID)
	return BuildContractAddress(codeID, instanceID)
}

// BuildContractAddress builds an sdk account address for a contract.
func BuildContractAddress(codeID, instanceID uint64) sdk.WasmAddress {
	contractID := make([]byte, 16)
	binary.BigEndian.PutUint64(contractID[:8], codeID)
	binary.BigEndian.PutUint64(contractID[8:], instanceID)
	return types.Module(types.ModuleName, contractID)[types.ContractIndex:]
}

func (k Keeper) autoIncrementID(ctx sdk.Context, lastIDKey []byte) uint64 {
	store := k.ada.NewStore(ctx, k.storeKey, nil)

	bz := store.Get(lastIDKey)
	id := uint64(1)
	if bz != nil {
		id = binary.BigEndian.Uint64(bz)
	}
	bz = sdk.Uint64ToBigEndian(id + 1)
	store.Set(lastIDKey, bz)
	return id
}

// PeekAutoIncrementID reads the current value without incrementing it.
func (k Keeper) PeekAutoIncrementID(ctx sdk.Context, lastIDKey []byte) uint64 {
	store := k.ada.NewStore(ctx, k.storeKey, nil)
	bz := store.Get(lastIDKey)
	id := uint64(1)
	if bz != nil {
		id = binary.BigEndian.Uint64(bz)
	}
	return id
}

func (k Keeper) importAutoIncrementID(ctx sdk.Context, lastIDKey []byte, val uint64) error {
	store := k.ada.NewStore(ctx, k.storeKey, nil)

	if store.Has(lastIDKey) {
		return sdkerrors.Wrapf(types.ErrDuplicate, "autoincrement id: %s", string(lastIDKey))
	}
	bz := sdk.Uint64ToBigEndian(val)
	store.Set(lastIDKey, bz)
	return nil
}

func (k Keeper) importContract(ctx sdk.Context, contractAddr sdk.WasmAddress, c *types.ContractInfo, state []types.Model) error {
	if !k.containsCodeInfo(ctx, c.CodeID) {
		return sdkerrors.Wrapf(types.ErrNotFound, "code id: %d", c.CodeID)
	}
	if k.HasContractInfo(ctx, contractAddr) {
		return sdkerrors.Wrapf(types.ErrDuplicate, "contract: %s", contractAddr)
	}

	historyEntry := c.ResetFromGenesis(ctx)
	k.appendToContractHistory(ctx, contractAddr, historyEntry)
	k.storeContractInfo(ctx, contractAddr, c)
	k.addToContractCodeSecondaryIndex(ctx, contractAddr, historyEntry)
	return k.importContractState(ctx, contractAddr, state)
}

func (k Keeper) newQueryHandler(ctx sdk.Context, contractAddress sdk.WasmAddress) QueryHandler {
	return NewQueryHandler(ctx, k.wasmVMQueryHandler, contractAddress, k.gasRegister)
}

// MultipliedGasMeter wraps the GasMeter from context and multiplies all reads by out defined multiplier
type MultipliedGasMeter struct {
	originalMeter sdk.GasMeter
	GasRegister   GasRegister
}

func NewMultipliedGasMeter(originalMeter sdk.GasMeter, gr GasRegister) MultipliedGasMeter {
	return MultipliedGasMeter{originalMeter: originalMeter, GasRegister: gr}
}

var _ wasmvm.GasMeter = MultipliedGasMeter{}

func (m MultipliedGasMeter) GasConsumed() sdk.Gas {
	return m.GasRegister.ToWasmVMGas(m.originalMeter.GasConsumed())
}

func (k Keeper) gasMeter(ctx sdk.Context) MultipliedGasMeter {
	return NewMultipliedGasMeter(ctx.GasMeter(), k.gasRegister)
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return moduleLogger(ctx)
}

func (k *Keeper) Cleanup() {
	k.wasmVM.Cleanup()
}

func (k *Keeper) SetInnerTxKeeper(innertxKeeper innertx.InnerTxKeeper) {
	k.innertxKeeper = innertxKeeper
}

func moduleLogger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// Querier creates a new grpc querier instance
func Querier(k *Keeper) *grpcQuerier {
	return NewGrpcQuerier(*k.cdc, k.storeKey, k.storageStoreKey, k, k.queryGasLimit)
}

// QueryGasLimit returns the gas limit for smart queries.
func (k Keeper) QueryGasLimit() sdk.Gas {
	return k.queryGasLimit
}

// BankCoinTransferrer replicates the cosmos-sdk behaviour as in
// https://github.com/okx/okbchain/blob/v0.1.0/libs/cosmos-sdk/x/bank/internal/keeperadapter/msg_server.go#L24
type BankCoinTransferrer struct {
	keeper types.BankKeeper
}

func NewBankCoinTransferrer(keeper types.BankKeeper) BankCoinTransferrer {
	return BankCoinTransferrer{
		keeper: keeper,
	}
}

// TransferCoins transfers coins from source to destination account when coin send was enabled for them and the recipient
// is not in the blocked address list.
func (c BankCoinTransferrer) TransferCoins(parentCtx sdk.Context, fromAddr sdk.WasmAddress, toAddr sdk.WasmAddress, amount sdk.Coins) error {
	em := sdk.NewEventManager()
	ctx := *parentCtx.SetEventManager(em)
	if err := c.keeper.IsSendEnabledCoins(ctx, amount...); err != nil {
		return err
	}
	if c.keeper.BlockedAddr(sdk.WasmToAccAddress(toAddr)) {
		return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive funds", toAddr.String())
	}

	sdkerr := c.keeper.SendCoins(ctx, sdk.WasmToAccAddress(fromAddr), sdk.WasmToAccAddress(toAddr), amount)
	if sdkerr != nil {
		return sdkerr
	}
	for _, e := range em.Events() {
		if e.Type == sdk.EventTypeMessage { // skip messages as we talk to the keeper directly
			continue
		}
		parentCtx.EventManager().EmitEvent(e)
	}
	return nil
}

type msgDispatcher interface {
	DispatchSubmessages(ctx sdk.Context, contractAddr sdk.WasmAddress, ibcPort string, msgs []wasmvmtypes.SubMsg) ([]byte, error)
}

// DefaultWasmVMContractResponseHandler default implementation that first dispatches submessage then normal messages.
// The Submessage execution may include an success/failure response handling by the contract that can overwrite the
// original
type DefaultWasmVMContractResponseHandler struct {
	md msgDispatcher
}

func NewDefaultWasmVMContractResponseHandler(md msgDispatcher) *DefaultWasmVMContractResponseHandler {
	return &DefaultWasmVMContractResponseHandler{md: md}
}

// Handle processes the data returned by a contract invocation.
func (h DefaultWasmVMContractResponseHandler) Handle(ctx sdk.Context, contractAddr sdk.WasmAddress, ibcPort string, messages []wasmvmtypes.SubMsg, origRspData []byte) ([]byte, error) {
	result := origRspData
	switch rsp, err := h.md.DispatchSubmessages(ctx, contractAddr, ibcPort, messages); {
	case err != nil:
		return nil, sdkerrors.Wrap(err, "submessages")
	case rsp != nil:
		result = rsp
	}
	return result, nil
}
