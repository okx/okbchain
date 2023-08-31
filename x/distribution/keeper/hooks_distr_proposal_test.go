package keeper

import (
	"testing"
	"time"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/x/staking"

	"github.com/okx/brczero/x/distribution/types"
	"github.com/stretchr/testify/require"
)

func TestHooksBeforeDelegationSharesModified(t *testing.T) {
	communityTax := sdk.NewDecWithPrec(2, 2)
	ctx, _, _, dk, sk, _, _ := CreateTestInputAdvanced(t, false, 1000, communityTax)

	dk.SetDistributionType(ctx, types.DistributionTypeOnChain)

	// create validator
	DoCreateValidator(t, ctx, sk, valOpAddr1, valConsPk1)
	//change val commission
	newRate, _ := sdk.NewDecFromStr("0.5")
	ctx.SetBlockTime(time.Now().UTC().Add(48 * time.Hour))
	DoEditValidator(t, ctx, sk, valOpAddr1, newRate)
	hook := dk.Hooks()

	// test BeforeDelegationSharesModified
	DoDeposit(t, ctx, sk, delAddr1, sdk.NewCoin(sk.BondDenom(ctx), sdk.NewInt(100)))
	require.Equal(t, uint64(1), dk.GetValidatorHistoricalReferenceCount(ctx))
	valOpAddrs := []sdk.ValAddress{valOpAddr1}
	DoAddShares(t, ctx, sk, delAddr1, valOpAddrs)

	hook.BeforeDelegationSharesModified(ctx, delAddr1, valOpAddrs)
	//will delete it
	require.False(t, dk.HasDelegatorStartingInfo(ctx, valOpAddr1, delAddr1))

}

func TestHooksAfterValidatorRemoved(t *testing.T) {
	communityTax := sdk.NewDecWithPrec(2, 2)
	ctx, ak, _, dk, sk, _, supplyKeeper := CreateTestInputAdvanced(t, false, 1000, communityTax)
	dk.SetDistributionType(ctx, types.DistributionTypeOnChain)

	// create validator
	DoCreateValidator(t, ctx, sk, valOpAddr1, valConsPk1)
	//change val commission
	newRate, _ := sdk.NewDecFromStr("0.5")
	ctx.SetBlockTime(time.Now().UTC().Add(48 * time.Hour))
	DoEditValidator(t, ctx, sk, valOpAddr1, newRate)

	// end block to bond validator
	staking.EndBlocker(ctx, sk)

	// next block
	ctx.SetBlockHeight(ctx.BlockHeight() + 1)

	hook := dk.Hooks()

	// test AfterValidatorCreated
	hook.AfterValidatorCreated(ctx, valOpAddr1)
	require.True(t, dk.GetValidatorAccumulatedCommission(ctx, valOpAddr1).IsZero())

	// test AfterValidatorRemoved
	acc := ak.GetAccount(ctx, supplyKeeper.GetModuleAddress(types.ModuleName))
	err := acc.SetCoins(NewTestSysCoins(123, 2))
	require.NoError(t, err)
	ak.SetAccount(ctx, acc)
	dk.SetValidatorAccumulatedCommission(ctx, valOpAddr1, NewTestSysCoins(123, 2))
	dk.SetValidatorOutstandingRewards(ctx, valOpAddr1, NewTestSysCoins(123, 2))
	hook.AfterValidatorRemoved(ctx, nil, valOpAddr1)
	require.True(t, ctx.KVStore(dk.storeKey).Get(valOpAddr1) == nil)

	// test to promote the coverage
	hook.AfterValidatorDestroyed(ctx, valConsAddr1, valOpAddr1)
	hook.BeforeValidatorModified(ctx, valOpAddr1)
	hook.AfterValidatorBonded(ctx, valConsAddr1, valOpAddr1)
	hook.AfterValidatorBeginUnbonding(ctx, valConsAddr1, valOpAddr1)
	hook.BeforeDelegationRemoved(ctx, valAccAddr1, valOpAddr1)
}
