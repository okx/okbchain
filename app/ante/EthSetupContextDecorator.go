package ante

import (
	"fmt"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/brczero/libs/cosmos-sdk/types/errors"
)

// EthSetupContextDecorator sets the infinite GasMeter in the Context and wraps
// the next AnteHandler with a defer clause to recover from any downstream
// OutOfGas panics in the AnteHandler chain to return an error with information
// on gas provided and gas used.
// CONTRACT: Must be first decorator in the chain
// CONTRACT: Tx must implement GasTx interface
type EthSetupContextDecorator struct{}

// NewEthSetupContextDecorator creates a new EthSetupContextDecorator
func NewEthSetupContextDecorator() EthSetupContextDecorator {
	return EthSetupContextDecorator{}
}

// AnteHandle sets the infinite gas meter to done to ignore costs in AnteHandler checks.
// This is undone at the EthGasConsumeDecorator, where the context is set with the
// ethereum tx GasLimit.
func (escd EthSetupContextDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	pinAnte(ctx.AnteTracer(), "EthSetupContextDecorator")

	// Decorator will catch an OutOfGasPanic caused in the next antehandler
	// AnteHandlers must have their own defer/recover in order for the BaseApp
	// to know how much gas was used! This is because the GasMeter is created in
	// the AnteHandler, but if it panics the context won't be set properly in
	// runTx's recover call.
	defer func() {
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				log := fmt.Sprintf(
					"out of gas in location: %v; gasLimit: %d, gasUsed: %d",
					rType.Descriptor, tx.GetGas(), ctx.GasMeter().GasConsumed(),
				)
				err = sdkerrors.Wrap(sdkerrors.ErrOutOfGas, log)
			default:
				panic(r)
			}
		}
	}()
	return next(ctx, tx, simulate)
}
