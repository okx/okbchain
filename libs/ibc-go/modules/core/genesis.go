package ibc

import (
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	client "github.com/okx/okbchain/libs/ibc-go/modules/core/02-client"
	connection "github.com/okx/okbchain/libs/ibc-go/modules/core/03-connection"
	channel "github.com/okx/okbchain/libs/ibc-go/modules/core/04-channel"
	"github.com/okx/okbchain/libs/ibc-go/modules/core/keeper"
	"github.com/okx/okbchain/libs/ibc-go/modules/core/types"
)

// InitGenesis initializes the ibc state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, createLocalhost bool, gs *types.GenesisState) {
	client.InitGenesis(ctx, k.ClientKeeper, gs.ClientGenesis)
	connection.InitGenesis(ctx, k.ConnectionKeeper, gs.ConnectionGenesis)
	channel.InitGenesis(ctx, k.ChannelKeeper, gs.ChannelGenesis)

	k.SetParams(ctx, gs.Params)
}

// ExportGenesis returns the ibc exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		ClientGenesis:     client.ExportGenesis(ctx, k.ClientKeeper),
		ConnectionGenesis: connection.ExportGenesis(ctx, k.ConnectionKeeper),
		ChannelGenesis:    channel.ExportGenesis(ctx, k.ChannelKeeper),
		Params:            k.GetParams(ctx),
	}
}
