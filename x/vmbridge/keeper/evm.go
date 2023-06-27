package keeper

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	ethermint "github.com/okx/okbchain/app/types"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	tmtypes "github.com/okx/okbchain/libs/tendermint/types"
	erc20types "github.com/okx/okbchain/x/erc20/types"
	evmtypes "github.com/okx/okbchain/x/evm/types"
	"github.com/okx/okbchain/x/evm/watcher"
	"github.com/okx/okbchain/x/vmbridge/types"
	"math/big"
)

// event __SendToWasmEventName(string wasmAddr,string recipient, string amount)
type SendToWasmEventHandler struct {
	Keeper
}

func NewSendToWasmEventHandler(k Keeper) *SendToWasmEventHandler {
	return &SendToWasmEventHandler{k}
}

// EventID Return the id of the log signature it handles
func (h SendToWasmEventHandler) EventID() common.Hash {
	return types.SendToWasmEvent.ID
}

// Handle Process the log
func (h SendToWasmEventHandler) Handle(ctx sdk.Context, contract common.Address, data []byte) error {
	if !tmtypes.HigherThanEarth(ctx.BlockHeight()) {
		errMsg := fmt.Sprintf("vmbridger not supprt at height %d", ctx.BlockHeight())
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
	}

	params := h.wasmKeeper.GetParams(ctx)
	if !params.VmbridgeEnable {
		return types.ErrVMBridgeEnable
	}

	logger := h.Keeper.Logger()
	unpacked, err := types.SendToWasmEvent.Inputs.Unpack(data)
	if err != nil {
		// log and ignore
		logger.Error("log signature matches but failed to decode", "error", err)
		return nil
	}

	caller := sdk.AccAddress(contract.Bytes())
	wasmAddr := unpacked[0].(string)
	recipient := unpacked[1].(string)
	amount := sdk.NewIntFromBigInt(unpacked[2].(*big.Int))

	return h.Keeper.SendToWasm(ctx, caller, wasmAddr, recipient, amount)
}

// event __OKBCCallToWasm(string wasmAddr,uint256 value, string calldata)
type CallToWasmEventHandler struct {
	Keeper
}

func NewCallToWasmEventHandler(k Keeper) *CallToWasmEventHandler {
	return &CallToWasmEventHandler{k}
}

// EventID Return the id of the log signature it handles
func (h CallToWasmEventHandler) EventID() common.Hash {
	return types.CallToWasmEvent.ID
}

// Handle Process the log
func (h CallToWasmEventHandler) Handle(ctx sdk.Context, contract common.Address, data []byte) error {
	if !tmtypes.HigherThanEarth(ctx.BlockHeight()) {
		errMsg := fmt.Sprintf("vmbridge not supprt at height %d", ctx.BlockHeight())
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
	}

	params := h.wasmKeeper.GetParams(ctx)
	if !params.VmbridgeEnable {
		return types.ErrVMBridgeEnable
	}

	logger := h.Keeper.Logger()
	unpacked, err := types.CallToWasmEvent.Inputs.Unpack(data)
	if err != nil {
		// log and ignore
		logger.Error("log signature matches but failed to decode", "error", err)
		return nil
	}

	caller := sdk.AccAddress(contract.Bytes())
	wasmAddr := unpacked[0].(string)
	value := sdk.NewIntFromBigInt(unpacked[1].(*big.Int))
	calldata := unpacked[2].(string)

	buff, err := hex.DecodeString(calldata)
	if err != nil {
		return err
	}
	_, err = h.Keeper.CallToWasm(ctx, caller, wasmAddr, value, string(buff))
	return err
}

// wasm call evm for erc20 exchange cw20,
func (k Keeper) SendToEvm(ctx sdk.Context, caller, contract string, recipient string, amount sdk.Int) (success bool, err error) {
	if !sdk.IsETHAddress(recipient) {
		return false, types.ErrIsNotETHAddr
	}

	if !sdk.IsETHAddress(contract) {
		return false, types.ErrIsNotETHAddr
	}

	contractAccAddr, err := sdk.AccAddressFromBech32(contract)
	if err != nil {
		return false, err
	}
	conrtractAddr := common.BytesToAddress(contractAccAddr.Bytes())

	recipientAccAddr, err := sdk.AccAddressFromBech32(recipient)
	if err != nil {
		return false, err
	}
	recipientAddr := common.BytesToAddress(recipientAccAddr.Bytes())
	input, err := types.GetMintERC20Input(caller, recipientAddr, amount.BigInt())
	if err != nil {
		return false, err
	}
	// k.CallEvm will call evm, so we must enable evm watch db with follow code
	if watcher.IsWatcherEnabled() {
		ctx.SetWatcher(watcher.NewTxWatcher())
	}
	_, result, err := k.CallEvm(ctx, erc20types.IbcEvmModuleETHAddr, &conrtractAddr, big.NewInt(0), input)
	if err != nil {
		return false, err
	}
	success, err = types.GetMintERC20Output(result.Ret)
	if watcher.IsWatcherEnabled() && err == nil {
		ctx.GetWatcher().Finalize()
	}
	return success, err
}

// wasm call evm
func (k Keeper) CallToEvm(ctx sdk.Context, caller, contract string, calldata string, value sdk.Int) (response string, err error) {

	if !sdk.IsETHAddress(contract) {
		return types.ErrIsNotETHAddr.Error(), types.ErrIsNotETHAddr
	}

	contractAccAddr, err := sdk.AccAddressFromBech32(contract)
	if err != nil {
		return err.Error(), err
	}
	conrtractAddr := common.BytesToAddress(contractAccAddr.Bytes())
	callerAddr, err := sdk.WasmAddressFromBech32(caller)
	if err != nil {
		return err.Error(), err
	}
	// k.CallEvm will call evm, so we must enable evm watch db with follow code
	if watcher.IsWatcherEnabled() {
		ctx.SetWatcher(watcher.NewTxWatcher())
	}

	realCall, err := hex.DecodeString(calldata)
	if err != nil {
		return err.Error(), err
	}
	_, result, err := k.CallEvm(ctx, common.BytesToAddress(callerAddr.Bytes()), &conrtractAddr, value.BigInt(), realCall)
	if err != nil {
		return err.Error(), err
	}
	if watcher.IsWatcherEnabled() && err == nil {
		ctx.GetWatcher().Finalize()
	}
	return string(result.Ret), nil
}

// callEvm execute an evm message from native module
func (k Keeper) CallEvm(ctx sdk.Context, callerAddr common.Address, to *common.Address, value *big.Int, data []byte) (*evmtypes.ExecutionResult, *evmtypes.ResultData, error) {

	config, found := k.evmKeeper.GetChainConfig(ctx)
	if !found {
		return nil, nil, types.ErrChainConfigNotFound
	}

	chainIDEpoch, err := ethermint.ParseChainID(ctx.ChainID())
	if err != nil {
		return nil, nil, err
	}

	acc := k.accountKeeper.GetAccount(ctx, callerAddr.Bytes())
	if acc == nil {
		acc = k.accountKeeper.NewAccountWithAddress(ctx, callerAddr.Bytes())
	}
	nonce := acc.GetSequence()
	txHash := tmtypes.Tx(ctx.TxBytes()).Hash()
	ethTxHash := common.BytesToHash(txHash)

	gasLimit := ctx.GasMeter().Limit()
	if gasLimit == sdk.NewInfiniteGasMeter().Limit() {
		gasLimit = k.evmKeeper.GetParams(ctx).MaxGasLimitPerTx
	}

	st := evmtypes.StateTransition{
		AccountNonce: nonce,
		Price:        big.NewInt(0),
		GasLimit:     gasLimit,
		Recipient:    to,
		Amount:       value,
		Payload:      data,
		Csdb:         evmtypes.CreateEmptyCommitStateDB(k.evmKeeper.GenerateCSDBParams(), ctx),
		ChainID:      chainIDEpoch,
		TxHash:       &ethTxHash,
		Sender:       callerAddr,
		Simulate:     ctx.IsCheckTx(),
		TraceTx:      false,
		TraceTxLog:   false,
	}
	st.Csdb.PrepareBlockTx(ethTxHash, k.evmKeeper.GetBlockHash(), 0)

	st.SetCallToCM(k.evmKeeper.GetCallToCM())
	addVMBridgeInnertx(ctx, k.evmKeeper, callerAddr.String(), to, VMBRIDGE_START_INNERTX, value)
	executionResult, resultData, err, innertxs, contracts := st.TransitionDb(ctx, config)
	addVMBridgeInnertx(ctx, k.evmKeeper, callerAddr.String(), to, VMBRIDGE_END_INNERTX, value)
	if !ctx.IsCheckTx() && !ctx.IsTraceTx() {
		if innertxs != nil {
			k.evmKeeper.AddInnerTx(ethTxHash.Hex(), innertxs)
		}
		if contracts != nil {
			k.evmKeeper.AddContract(contracts)
		}
	}
	attributes := make([]sdk.Attribute, 0)
	if err != nil {
		attribute := sdk.NewAttribute(types.AttributeResult, err.Error())
		attributes = append(attributes, attribute)
	} else {
		buff, err := json.Marshal(resultData)
		if err != nil {
			attribute := sdk.NewAttribute(types.AttributeResult, err.Error())
			attributes = append(attributes, attribute)
		} else {
			attribute := sdk.NewAttribute(types.AttributeResult, string(buff))
			attributes = append(attributes, attribute)
		}

	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeWasmCallEvm,
			attributes...,
		),
	)
	if err != nil {
		return nil, nil, err
	}

	st.Csdb.Commit(false) // write code to db

	temp := k.accountKeeper.GetAccount(ctx, callerAddr.Bytes())
	if temp == nil {
		if err := acc.SetCoins(sdk.Coins{}); err != nil {
			return nil, nil, err
		}
		temp = acc
	}
	if err := temp.SetSequence(nonce + 1); err != nil {
		return nil, nil, err
	}
	k.accountKeeper.SetAccount(ctx, temp)

	return executionResult, resultData, err
}
