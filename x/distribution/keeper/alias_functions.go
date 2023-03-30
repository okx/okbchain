package keeper

import (
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/supply/exported"

	"github.com/okx/okbchain/x/distribution/types"
)

// GetDistributionAccount returns the distribution ModuleAccount
func (k Keeper) GetDistributionAccount(ctx sdk.Context) exported.ModuleAccountI {
	return k.supplyKeeper.GetModuleAccount(ctx, types.ModuleName)
}

// GetExtraFeeAccount returns the extra fee collector ModuleAccount
func (k Keeper) GetExtraFeeAccount(ctx sdk.Context) exported.ModuleAccountI {
	k.supplyKeeper.RegisterPerAddr(auth.ExtraFeeCollectorName, nil)
	return k.supplyKeeper.GetModuleAccount(ctx, auth.ExtraFeeCollectorName)
}
