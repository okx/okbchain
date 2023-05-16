package wasm

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	bam "github.com/okx/okbchain/libs/cosmos-sdk/baseapp"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	sdktypes "github.com/okx/okbchain/libs/cosmos-sdk/x/auth/types"
	cfg "github.com/okx/okbchain/libs/tendermint/config"
	"github.com/okx/okbchain/libs/tendermint/libs/kv"
	types2 "github.com/okx/okbchain/libs/tendermint/types"
	"github.com/okx/okbchain/x/wasm/keeper"
	"github.com/okx/okbchain/x/wasm/types"
	"github.com/okx/okbchain/x/wasm/watcher"
)

// NewHandler returns a handler for "wasm" type messages.
func NewHandler(k types.ContractOpsKeeper) sdk.Handler {
	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx.SetEventManager(sdk.NewEventManager())

		if !types2.HigherThanEarth(ctx.BlockHeight()) {
			errMsg := fmt.Sprintf("wasm not support at height %d", ctx.BlockHeight())
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
		}

		var (
			res proto.Message
			err error
		)

		beforeGC := int64(ctx.GasMeter().GasConsumed())

		// update watcher
		defer func() {
			// update watchDB when delivering tx
			if ctx.IsDeliver() || ctx.ParaMsg() != nil {
				watcher.Save(err)
			}

			if err == nil && !ctx.IsCheckTx() {
				updateHGU(ctx, beforeGC, msg)
			}
		}()

		switch msg := msg.(type) {
		case *MsgStoreCode: //nolint:typecheck
			res, err = msgServer.StoreCode(sdk.WrapSDKContext(ctx), msg)
		case *MsgInstantiateContract:
			res, err = msgServer.InstantiateContract(sdk.WrapSDKContext(ctx), msg)
		case *MsgExecuteContract:
			res, err = msgServer.ExecuteContract(sdk.WrapSDKContext(ctx), msg)
		case *MsgMigrateContract:
			res, err = msgServer.MigrateContract(sdk.WrapSDKContext(ctx), msg)
		case *MsgUpdateAdmin:
			res, err = msgServer.UpdateAdmin(sdk.WrapSDKContext(ctx), msg)
		case *MsgClearAdmin:
			res, err = msgServer.ClearAdmin(sdk.WrapSDKContext(ctx), msg)
		default:
			errMsg := fmt.Sprintf("unrecognized wasm message type: %T", msg)
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
		}

		ctx.SetEventManager(filterMessageEvents(ctx))
		return sdk.WrapServiceResult(ctx, res, err)
	}
}

func updateHGU(ctx sdk.Context, beforeGC int64, msg sdk.Msg) {
	if cfg.DynamicConfig.GetMaxGasUsedPerBlock() <= 0 {
		return
	}

	v, ok := msg.(sdktypes.WasmMsgChecker)
	if !ok {
		return
	}

	fnSign, deploySize, err := v.FnSignatureInfo()
	if err != nil || len(fnSign) <= 0 {
		return
	}

	afterGC := int64(ctx.GasMeter().GasConsumed())
	gc := afterGC - beforeGC
	if gc <= 0 {
		ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName)).Error(
			fmt.Sprintf("Update hgu gc error, before :%d, after:%d", beforeGC, afterGC))
		return
	}

	if deploySize > 0 {
		// calculate average gas consume for deploy contract case
		gc = gc / int64(deploySize)
	}

	bam.InstanceOfHistoryGasUsedRecordDB().UpdateGasUsed([]byte(fnSign), gc)
}

// filterMessageEvents returns the same events with all of type == EventTypeMessage removed except
// for wasm message types.
// this is so only our top-level message event comes through
func filterMessageEvents(ctx sdk.Context) *sdk.EventManager {
	m := sdk.NewEventManager()
	for _, e := range ctx.EventManager().Events() {
		if e.Type == sdk.EventTypeMessage &&
			!hasWasmModuleAttribute(e.Attributes) {
			continue
		}
		m.EmitEvent(e)
	}
	return m
}

func hasWasmModuleAttribute(attrs []kv.Pair) bool {
	for _, a := range attrs {
		if sdk.AttributeKeyModule == string(a.Key) &&
			types.ModuleName == string(a.Value) {
			return true
		}
	}
	return false
}
