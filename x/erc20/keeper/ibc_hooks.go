package keeper

import (
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	trensferTypes "github.com/okx/okbchain/libs/ibc-go/modules/apps/transfer/types"
	"github.com/okx/okbchain/x/erc20/types"
	"github.com/okx/okbchain/x/evm/watcher"
)

var (
	_ trensferTypes.TransferHooks = IBCTransferHooks{}
)

type IBCTransferHooks struct {
	Keeper
}

func NewIBCTransferHooks(k Keeper) IBCTransferHooks {
	return IBCTransferHooks{k}
}

func (iths IBCTransferHooks) AfterSendTransfer(
	ctx sdk.Context,
	sourcePort, sourceChannel string,
	token sdk.SysCoin,
	sender sdk.AccAddress,
	receiver string,
	isSource bool) error {
	iths.Logger(ctx).Info(
		"trigger ibc transfer hook",
		"hook", "AfterSendTransfer",
		"sourcePort", sourcePort,
		"sourceChannel", sourceChannel,
		"token", token.String(),
		"sender", sender.String(),
		"receiver", receiver,
		"isSource", isSource)
	return nil
}

func (iths IBCTransferHooks) AfterRecvTransfer(
	ctx sdk.Context,
	destPort, destChannel string,
	token sdk.SysCoin,
	receiver string,
	isSource bool) error {
	iths.Logger(ctx).Info(
		"trigger ibc transfer hook",
		"hook", "AfterRecvTransfer",
		"destPort", destPort,
		"destChannel", destChannel,
		"token", token.String(),
		"receiver", receiver,
		"isSource", isSource)
	// only after minting vouchers on this chain
	if watcher.IsWatcherEnabled() {
		ctx.SetWatcher(watcher.NewTxWatcher())
	}

	var err error
	if !isSource {
		// the native coin come from other chain with ibc
		if err = iths.Keeper.OnMintVouchers(ctx, sdk.NewCoins(token), receiver); err == types.ErrNoContractNotAuto {
			err = nil
		}
	} else if token.Denom != sdk.DefaultBondDenom {
		// the native coin come from this chain,
		err = iths.Keeper.OnUnescrowNatives(ctx, sdk.NewCoins(token), receiver)
	}

	if watcher.IsWatcherEnabled() && err == nil {
		ctx.GetWatcher().Finalize()
	}
	return err
}

func (iths IBCTransferHooks) AfterRefundTransfer(
	ctx sdk.Context,
	sourcePort, sourceChannel string,
	token sdk.SysCoin,
	sender string,
	isSource bool) error {
	iths.Logger(ctx).Info(
		"trigger ibc transfer hook",
		"hook", "AfterRefundTransfer",
		"sourcePort", sourcePort,
		"sourceChannel", sourceChannel,
		"token", token.String(),
		"sender", sender,
		"isSource", isSource)
	// only after minting vouchers on this chain
	if watcher.IsWatcherEnabled() {
		ctx.SetWatcher(watcher.NewTxWatcher())
	}

	var err error
	if !isSource {
		// the native coin come from other chain with ibc
		if err = iths.Keeper.OnMintVouchers(ctx, sdk.NewCoins(token), sender); err == types.ErrNoContractNotAuto {
			err = nil
		}
	} else if token.Denom != sdk.DefaultBondDenom {
		// the native coin come from this chain,
		err = iths.Keeper.OnUnescrowNatives(ctx, sdk.NewCoins(token), sender)
	}

	if watcher.IsWatcherEnabled() && err == nil {
		ctx.GetWatcher().Finalize()
	}
	return err
}
