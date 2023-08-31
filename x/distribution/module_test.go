package distribution

import (
	"testing"

	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	abci "github.com/okx/brczero/libs/tendermint/abci/types"
	"github.com/okx/brczero/x/distribution/keeper"
	"github.com/okx/brczero/x/distribution/types"
	"github.com/stretchr/testify/require"
)

func TestAppModule(t *testing.T) {
	ctx, _, k, _, supplyKeeper := keeper.CreateTestInputDefault(t, false, 1000)

	module := NewAppModule(k, supplyKeeper)
	require.EqualValues(t, ModuleName, module.AppModuleBasic.Name())
	require.EqualValues(t, ModuleName, module.Name())
	require.EqualValues(t, RouterKey, module.Route())
	require.EqualValues(t, QuerierRoute, module.QuerierRoute())

	cdc := codec.New()
	module.RegisterCodec(cdc)

	msg := module.DefaultGenesis()
	require.Nil(t, module.ValidateGenesis(msg))
	require.NotNil(t, module.ValidateGenesis([]byte{}))
	module.InitGenesis(ctx, msg)
	exportMsg := module.ExportGenesis(ctx)

	var gs GenesisState
	require.NotPanics(t, func() {
		types.ModuleCdc.MustUnmarshalJSON(exportMsg, &gs)
	})

	// for coverage
	module.BeginBlock(ctx, abci.RequestBeginBlock{})
	module.EndBlock(ctx, abci.RequestEndBlock{})
	module.GetQueryCmd(cdc)
	module.GetTxCmd(cdc)
	module.NewQuerierHandler()
	module.NewHandler()
}
