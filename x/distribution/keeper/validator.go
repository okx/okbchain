package keeper

import (
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"

	"github.com/okx/brczero/x/staking/exported"
)

// initialize rewards for a new validator
func (k Keeper) initializeValidator(ctx sdk.Context, val exported.ValidatorI) {
	k.initializeValidatorDistrProposal(ctx, val)
	return
}
