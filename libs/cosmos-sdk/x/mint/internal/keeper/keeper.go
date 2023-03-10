package keeper

import (
	"fmt"

	"github.com/okx/okbchain/libs/tendermint/libs/log"

	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/mint/internal/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/params"
)

// Keeper of the mint store
type Keeper struct {
	cdc              *codec.Codec
	storeKey         sdk.StoreKey
	paramSpace       params.Subspace
	sk               types.StakingKeeper
	supplyKeeper     types.SupplyKeeper
	feeCollectorName string

	originalMintedPerBlock sdk.Dec
	govKeeper              types.GovKeeper
}

// NewKeeper creates a new mint Keeper instance
func NewKeeper(
	cdc *codec.Codec, key sdk.StoreKey, paramSpace params.Subspace,
	sk types.StakingKeeper, supplyKeeper types.SupplyKeeper, feeCollectorName string,
) Keeper {

	// ensure mint module account is set
	if addr := supplyKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the mint module account has not been set")
	}

	return Keeper{
		cdc:                    cdc,
		storeKey:               key,
		paramSpace:             paramSpace.WithKeyTable(types.ParamKeyTable()),
		sk:                     sk,
		supplyKeeper:           supplyKeeper,
		feeCollectorName:       feeCollectorName,
		originalMintedPerBlock: types.DefaultOriginalMintedPerBlock(),
	}
}

//______________________________________________________________________

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// get the minter
func (k Keeper) GetMinter(ctx sdk.Context) (minter types.Minter) {
	store := ctx.KVStore(k.storeKey)
	b := store.Get(types.MinterKey)
	if b == nil {
		panic("stored minter should not have been nil")
	}

	k.cdc.MustUnmarshalBinaryLengthPrefixed(b, &minter)
	return
}

// set the minter
func (k Keeper) SetMinter(ctx sdk.Context, minter types.MinterCustom) {
	store := ctx.KVStore(k.storeKey)
	b := k.cdc.MustMarshalBinaryLengthPrefixed(minter)
	store.Set(types.MinterKey, b)
}

//______________________________________________________________________

// GetParams returns the total set of minting parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the total set of minting parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}

//______________________________________________________________________

// StakingTokenSupply implements an alias call to the underlying staking keeper's
// StakingTokenSupply to be used in BeginBlocker.
func (k Keeper) StakingTokenSupply(ctx sdk.Context) sdk.Dec {
	return k.sk.StakingTokenSupply(ctx)
}

// BondedRatio implements an alias call to the underlying staking keeper's
// BondedRatio to be used in BeginBlocker.
func (k Keeper) BondedRatio(ctx sdk.Context) sdk.Dec {
	return k.sk.BondedRatio(ctx)
}

// MintCoins implements an alias call to the underlying supply keeper's
// MintCoins to be used in BeginBlocker.
func (k Keeper) MintCoins(ctx sdk.Context, newCoins sdk.Coins) error {
	if newCoins.Empty() {
		// skip as no coins need to be minted
		return nil
	}

	return k.supplyKeeper.MintCoins(ctx, types.ModuleName, newCoins)
}

// AddCollectedFees implements an alias call to the underlying supply keeper's
// AddCollectedFees to be used in BeginBlocker.
func (k Keeper) AddCollectedFees(ctx sdk.Context, fees sdk.Coins) error {
	remain, err := k.AllocateTokenToTreasure(ctx, fees)
	if err != nil {
		return err
	}
	return k.supplyKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, k.feeCollectorName, remain)
}

// SetGovKeeper sets keeper of gov
func (k *Keeper) SetGovKeeper(gk types.GovKeeper) {
	k.govKeeper = gk
}

func (k *Keeper) InvokeExtraProposal(ctx sdk.Context, action string, extra string) error {
	switch action {
	case types.ActionNextBlockUpdate:
		return k.handleNextBlockUpdate(ctx, extra)
	case types.ActionMintedPerBlock:
		return k.handleMintedPerBlock(ctx, extra)
	}

	return nil
}

func (k *Keeper) handleNextBlockUpdate(ctx sdk.Context, extra string) error {
	param, err := types.NewNextBlockUpdate(extra)
	if err != nil {
		return err
	}

	if param.BlockNum <= uint64(ctx.BlockHeight()) {
		return types.ErrNextBlockUpdateTooLate
	}

	minter := k.GetMinterCustom(ctx)
	minter.NextBlockToUpdate = param.BlockNum
	k.SetMinterCustom(ctx, minter)
	return nil
}

func (k *Keeper) handleMintedPerBlock(ctx sdk.Context, extra string) error {
	param, err := types.NewMintedPerBlockParams(extra)
	if err != nil {
		return err
	}

	if param.Coins.AmountOf(sdk.DefaultBondDenom).IsZero() {
		return types.ErrExtendProposalParams("coin is zero")
	}

	if param.Coins.AmountOf(sdk.DefaultBondDenom).IsNegative() {
		return types.ErrExtendProposalParams("coin is negative")
	}

	minter := k.GetMinterCustom(ctx)
	minter.MintedPerBlock = param.Coins
	k.SetMinterCustom(ctx, minter)
	return nil
}
