package ante

import (
	ethermint "github.com/okx/brczero/app/types"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/brczero/libs/cosmos-sdk/types/errors"
	evmtypes "github.com/okx/brczero/x/evm/types"
)

// EthSigVerificationDecorator validates an ethereum signature
type EthSigVerificationDecorator struct{}

// NewEthSigVerificationDecorator creates a new EthSigVerificationDecorator
func NewEthSigVerificationDecorator() EthSigVerificationDecorator {
	return EthSigVerificationDecorator{}
}

// AnteHandle validates the signature and returns sender address
func (esvd EthSigVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// simulate means 'eth_call' or 'eth_estimateGas', when it means 'eth_estimateGas' we can not 'VerifySig'.so skip here
	if simulate {
		return next(ctx, tx, simulate)
	}
	pinAnte(ctx.AnteTracer(), "EthSigVerificationDecorator")

	msgEthTx, ok := tx.(*evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	// parse the chainID from a string to a base-10 integer
	chainIDEpoch, err := ethermint.ParseChainID(ctx.ChainID())
	if err != nil {
		return ctx, err
	}

	// validate sender/signature and cache the address
	err = msgEthTx.VerifySig(chainIDEpoch, ctx.BlockHeight())
	if err != nil {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "signature verification failed: %s", err.Error())
	}

	// NOTE: when signature verification succeeds, a non-empty signer address can be
	// retrieved from the transaction on the next AnteDecorators.
	return next(ctx, msgEthTx, simulate)
}
