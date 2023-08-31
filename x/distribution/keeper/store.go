package keeper

import (
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"

	"github.com/okx/brczero/x/distribution/types"
)

// GetDelegatorWithdrawAddr returns the delegator withdraw address, defaulting to the delegator address
func (k Keeper) GetDelegatorWithdrawAddr(ctx sdk.Context, delAddr sdk.AccAddress) sdk.AccAddress {
	store := ctx.KVStore(k.storeKey)
	b := store.Get(types.GetDelegatorWithdrawAddrKey(delAddr))
	if b == nil {
		return delAddr
	}
	return sdk.AccAddress(b)
}

// SetDelegatorWithdrawAddr sets the delegator withdraw address
func (k Keeper) SetDelegatorWithdrawAddr(ctx sdk.Context, delAddr, withdrawAddr sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetDelegatorWithdrawAddrKey(delAddr), withdrawAddr.Bytes())
}

// IterateDelegatorWithdrawAddrs iterates over delegator withdraw addrs
func (k Keeper) IterateDelegatorWithdrawAddrs(ctx sdk.Context,
	handler func(del sdk.AccAddress, addr sdk.AccAddress) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iter := sdk.KVStorePrefixIterator(store, types.DelegatorWithdrawAddrPrefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		addr := sdk.AccAddress(iter.Value())
		del := types.GetDelegatorWithdrawInfoAddress(iter.Key())
		if handler(del, addr) {
			break
		}
	}
}

// GetFeePool returns the global fee pool distribution info
func (k Keeper) GetFeePool(ctx sdk.Context) (feePool types.FeePool) {
	store := ctx.KVStore(k.storeKey)
	b := store.Get(types.FeePoolKey)
	if b == nil {
		panic("Stored fee pool should not have been nil")
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &feePool)
	return
}

// SetFeePool sets the global fee pool distribution info
func (k Keeper) SetFeePool(ctx sdk.Context, feePool types.FeePool) {
	store := ctx.KVStore(k.storeKey)
	b := k.cdc.MustMarshalBinaryLengthPrefixed(feePool)
	store.Set(types.FeePoolKey, b)
}

// GetFeePoolCommunityCoins returns the community coins
func (k Keeper) GetFeePoolCommunityCoins(ctx sdk.Context) sdk.SysCoins {
	return k.GetFeePool(ctx).CommunityPool
}

// GetPreviousProposerConsAddr returns the proposer public key for this block
func (k Keeper) GetPreviousProposerConsAddr(ctx sdk.Context) (consAddr sdk.ConsAddress) {
	store := ctx.KVStore(k.storeKey)
	b := store.Get(types.ProposerKey)
	if b == nil {
		panic("Previous proposer not set")
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &consAddr)
	return consAddr
}

// SetPreviousProposerConsAddr sets the proposer public key for this block
func (k Keeper) SetPreviousProposerConsAddr(ctx sdk.Context, consAddr sdk.ConsAddress) {
	store := ctx.KVStore(k.storeKey)
	b := k.cdc.MustMarshalBinaryLengthPrefixed(consAddr)
	store.Set(types.ProposerKey, b)
}

// GetValidatorAccumulatedCommission returns accumulated commission for a validator
func (k Keeper) GetValidatorAccumulatedCommission(ctx sdk.Context, val sdk.ValAddress) (
	commission types.ValidatorAccumulatedCommission) {

	store := ctx.KVStore(k.storeKey)
	b := store.Get(types.GetValidatorAccumulatedCommissionKey(val))
	if b == nil {
		return types.ValidatorAccumulatedCommission{}
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &commission)
	return commission
}

// SetValidatorAccumulatedCommission sets accumulated commission for a validator
func (k Keeper) SetValidatorAccumulatedCommission(ctx sdk.Context, val sdk.ValAddress,
	commission types.ValidatorAccumulatedCommission) {

	var bz []byte
	store := ctx.KVStore(k.storeKey)
	if commission.IsZero() {
		bz = k.cdc.MustMarshalBinaryLengthPrefixed(types.InitialValidatorAccumulatedCommission())
	} else {
		bz = k.cdc.MustMarshalBinaryLengthPrefixed(commission)
	}
	store.Set(types.GetValidatorAccumulatedCommissionKey(val), bz)
}

// deleteValidatorAccumulatedCommission deletes accumulated commission for a validator
func (k Keeper) deleteValidatorAccumulatedCommission(ctx sdk.Context, val sdk.ValAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.GetValidatorAccumulatedCommissionKey(val))
}

// IterateValidatorAccumulatedCommissions iterates over accumulated commissions
func (k Keeper) IterateValidatorAccumulatedCommissions(ctx sdk.Context,
	handler func(val sdk.ValAddress, commission types.ValidatorAccumulatedCommission) (stop bool)) {

	store := ctx.KVStore(k.storeKey)
	iter := sdk.KVStorePrefixIterator(store, types.ValidatorAccumulatedCommissionPrefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var commission types.ValidatorAccumulatedCommission
		k.cdc.MustUnmarshalBinaryLengthPrefixed(iter.Value(), &commission)
		addr := types.GetValidatorAccumulatedCommissionAddress(iter.Key())
		if handler(addr, commission) {
			break
		}
	}
}
