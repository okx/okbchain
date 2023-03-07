package types

import (
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
)

// MultiStakingHooks combines multiple staking hooks, all hook functions are run in array sequence
// TODO:need to trim the staking hooks as the okbchain design
type MultiStakingHooks []StakingHooks

// NewMultiStakingHooks creates a new object of MultiStakingHooks
func NewMultiStakingHooks(hooks ...StakingHooks) MultiStakingHooks {
	return hooks
}

// AfterValidatorCreated handles the hooks after the validator created
func (h MultiStakingHooks) AfterValidatorCreated(ctx sdk.Context, valAddr sdk.ValAddress) {
	for i := range h {
		h[i].AfterValidatorCreated(ctx, valAddr)
	}
}

// BeforeValidatorModified handles the hooks before the validator modified
func (h MultiStakingHooks) BeforeValidatorModified(ctx sdk.Context, valAddr sdk.ValAddress) {
	for i := range h {
		h[i].BeforeValidatorModified(ctx, valAddr)
	}
}

// AfterValidatorRemoved handles the hooks after the validator was removed
func (h MultiStakingHooks) AfterValidatorRemoved(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) {
	for i := range h {
		h[i].AfterValidatorRemoved(ctx, consAddr, valAddr)
	}
}

// AfterValidatorBonded handles the hooks after the validator was bonded
func (h MultiStakingHooks) AfterValidatorBonded(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) {
	for i := range h {
		h[i].AfterValidatorBonded(ctx, consAddr, valAddr)
	}
}

// AfterValidatorBeginUnbonding handles the hooks after the validator began the unbonding
func (h MultiStakingHooks) AfterValidatorBeginUnbonding(ctx sdk.Context, consAddr sdk.ConsAddress,
	valAddr sdk.ValAddress) {
	for i := range h {
		h[i].AfterValidatorBeginUnbonding(ctx, consAddr, valAddr)
	}
}

// AfterValidatorDestroyed handles the hooks after the validator was destroyed by tx
func (h MultiStakingHooks) AfterValidatorDestroyed(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) {
	for i := range h {
		h[i].AfterValidatorDestroyed(ctx, consAddr, valAddr)
	}
}
