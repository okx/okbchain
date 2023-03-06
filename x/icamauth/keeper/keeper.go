package keeper

import (
	"fmt"

	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	storetypes "github.com/okx/okbchain/libs/cosmos-sdk/store/types"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	capabilitykeeper "github.com/okx/okbchain/libs/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/okx/okbchain/libs/cosmos-sdk/x/capability/types"
	icacontrollerkeeper "github.com/okx/okbchain/libs/ibc-go/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/okx/okbchain/libs/tendermint/libs/log"
	"github.com/okx/okbchain/x/icamauth/types"
)

type Keeper struct {
	cdc *codec.CodecProxy

	storeKey storetypes.StoreKey

	scopedKeeper        capabilitykeeper.ScopedKeeper
	icaControllerKeeper icacontrollerkeeper.Keeper
}

func NewKeeper(cdc *codec.CodecProxy, storeKey storetypes.StoreKey, iaKeeper icacontrollerkeeper.Keeper, scopedKeeper capabilitykeeper.ScopedKeeper) Keeper {
	return Keeper{
		cdc:                 cdc,
		storeKey:            storeKey,
		scopedKeeper:        scopedKeeper,
		icaControllerKeeper: iaKeeper,
	}
}

// Logger returns the application logger, scoped to the associated module
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// ClaimCapability claims the channel capability passed via the OnOpenChanInit callback
func (k *Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}
