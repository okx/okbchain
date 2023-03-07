package ante

import (
	"github.com/okx/okbchain/libs/cosmos-sdk/baseapp"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	evmtypes "github.com/okx/okbchain/x/evm/types"
)

// IncrementSenderSequenceDecorator increments the sequence of the signers. The
// main difference with the SDK's IncrementSequenceDecorator is that the MsgEthereumTx
// doesn't implement the SigVerifiableTx interface.
//
// CONTRACT: must be called after msg.VerifySig in order to cache the sender address.
type IncrementSenderSequenceDecorator struct {
	ak auth.AccountKeeper
}

// NewIncrementSenderSequenceDecorator creates a new IncrementSenderSequenceDecorator.
func NewIncrementSenderSequenceDecorator(ak auth.AccountKeeper) IncrementSenderSequenceDecorator {
	return IncrementSenderSequenceDecorator{
		ak: ak,
	}
}

// AnteHandle handles incrementing the sequence of the sender.
func (issd IncrementSenderSequenceDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	pinAnte(ctx.AnteTracer(), "IncrementSenderSequenceDecorator")

	// always incrementing the sequence when ctx is recheckTx mode (when mempool in disableRecheck mode, we will also has force recheck period),
	// when just in check mode:
	// A、when mempool is in enableRecheck mode, we will need to increase the nonce [means will not support tx replace with same nonce].
	// B、when mempool is in disableRecheck mode [now support tx replace with same nonce], we should just return

	// when IsCheckTx() is true, it will means checkTx and recheckTx mode, but IsReCheckTx() is true it must be recheckTx mode
	// if IsTraceMode is true,  sequence must be set.
	if ctx.IsCheckTx() && !ctx.IsReCheckTx() && !baseapp.IsMempoolEnableRecheck() && !ctx.IsTraceTx() {
		return next(ctx, tx, simulate)
	}

	// get and set account must be called with an infinite gas meter in order to prevent
	// additional gas from being deducted.
	gasMeter := ctx.GasMeter()
	ctx.SetGasMeter(sdk.NewInfiniteGasMeter())

	msgEthTx, ok := tx.(*evmtypes.MsgEthereumTx)
	if !ok {
		ctx.SetGasMeter(gasMeter)
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	if ctx.From() != "" {
		msgEthTx.SetFrom(ctx.From())
	}
	// increment sequence of all signers
	for _, addr := range msgEthTx.GetSigners() {
		acc := issd.ak.GetAccount(ctx, addr)
		seq := acc.GetSequence()
		if !baseapp.IsMempoolEnablePendingPool() {
			seq++
		} else if msgEthTx.Data.AccountNonce == seq {
			seq++
		}
		if err := acc.SetSequence(seq); err != nil {
			panic(err)
		}
		issd.ak.SetAccount(ctx, acc)
	}

	// set the original gas meter
	ctx.SetGasMeter(gasMeter)

	return next(ctx, tx, simulate)
}
