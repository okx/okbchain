package ante

import (
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/brczero/libs/cosmos-sdk/types/errors"
	"github.com/okx/brczero/libs/cosmos-sdk/x/auth"
	evmtypes "github.com/okx/brczero/x/evm/types"
)

// AccountVerificationDecorator validates an account balance checks
type AccountVerificationDecorator struct {
	ak        auth.AccountKeeper
	evmKeeper EVMKeeper
}

// NewAccountVerificationDecorator creates a new AccountVerificationDecorator
func NewAccountVerificationDecorator(ak auth.AccountKeeper, ek EVMKeeper) AccountVerificationDecorator {
	return AccountVerificationDecorator{
		ak:        ak,
		evmKeeper: ek,
	}
}

// AnteHandle validates the signature and returns sender address
func (avd AccountVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !ctx.IsCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}

	msgEthTx, ok := tx.(*evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	address := msgEthTx.AccountAddress()
	if address.Empty() {
		panic("sender address cannot be empty")
	}

	acc := avd.ak.GetAccount(ctx, address)
	if acc == nil {
		acc = avd.ak.NewAccountWithAddress(ctx, address)
		avd.ak.SetAccount(ctx, acc)
	}

	// on InitChain make sure account number == 0
	if ctx.BlockHeight() == 0 && acc.GetAccountNumber() != 0 {
		return ctx, sdkerrors.Wrapf(
			sdkerrors.ErrInvalidSequence,
			"invalid account number for height zero (got %d)", acc.GetAccountNumber(),
		)
	}

	evmDenom := sdk.DefaultBondDenom

	// validate sender has enough funds to pay for gas cost
	balance := acc.GetCoins().AmountOf(evmDenom)
	if balance.BigInt().Cmp(msgEthTx.Cost()) < 0 {
		return ctx, sdkerrors.Wrapf(
			sdkerrors.ErrInsufficientFunds,
			"sender balance < tx gas cost (%s%s < %s%s)", balance.String(), evmDenom, sdk.NewDecFromBigIntWithPrec(msgEthTx.Cost(), sdk.Precision).String(), evmDenom,
		)
	}

	return next(ctx, tx, simulate)
}
