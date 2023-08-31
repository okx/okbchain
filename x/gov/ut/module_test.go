package ut

import (
	"testing"

	"github.com/okx/brczero/x/gov"

	chaincodec "github.com/okx/brczero/app/codec"
	"github.com/okx/brczero/libs/cosmos-sdk/client/context"
	cliLcd "github.com/okx/brczero/libs/cosmos-sdk/client/lcd"
	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	interfacetypes "github.com/okx/brczero/libs/cosmos-sdk/codec/types"
	"github.com/okx/brczero/libs/cosmos-sdk/types/module"
	ibctransfer "github.com/okx/brczero/libs/ibc-go/modules/apps/transfer"
	ibc "github.com/okx/brczero/libs/ibc-go/modules/core"
	abci "github.com/okx/brczero/libs/tendermint/abci/types"
	"github.com/okx/brczero/x/gov/client"
	"github.com/okx/brczero/x/gov/client/rest"
	"github.com/okx/brczero/x/gov/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestAppModule_BeginBlock(t *testing.T) {

}

func getCmdSubmitProposal(proxy *codec.CodecProxy, reg interfacetypes.InterfaceRegistry) *cobra.Command {
	return &cobra.Command{}
}

func proposalRESTHandler(cliCtx context.CLIContext) rest.ProposalRESTHandler {
	return rest.ProposalRESTHandler{}
}

func TestNewAppModuleBasic(t *testing.T) {
	ctx, _, gk, _, crisisKeeper := CreateTestInput(t, false, 1000)

	moduleBasic := gov.NewAppModuleBasic(client.ProposalHandler{
		CLIHandler:  getCmdSubmitProposal,
		RESTHandler: proposalRESTHandler,
	})

	require.Equal(t, types.ModuleName, moduleBasic.Name())

	cdc := codec.New()
	ModuleBasics := module.NewBasicManager(
		ibc.AppModuleBasic{},
		ibctransfer.AppModuleBasic{},
	)
	interfaceReg := chaincodec.MakeIBC(ModuleBasics)
	protoCodec := codec.NewProtoCodec(interfaceReg)
	codecProxy := codec.NewCodecProxy(protoCodec, cdc)

	moduleBasic.RegisterCodec(cdc)
	bz, err := cdc.MarshalBinaryBare(types.MsgSubmitProposal{})
	require.NotNil(t, bz)
	require.Nil(t, err)

	jsonMsg := moduleBasic.DefaultGenesis()
	err = moduleBasic.ValidateGenesis(jsonMsg)
	require.Nil(t, err)
	err = moduleBasic.ValidateGenesis(jsonMsg[:len(jsonMsg)-1])
	require.NotNil(t, err)

	rs := cliLcd.NewRestServer(codecProxy, interfaceReg, nil)
	moduleBasic.RegisterRESTRoutes(rs.CliCtx, rs.Mux)

	// todo: check diff after GetTxCmd
	moduleBasic.GetTxCmd(cdc)

	// todo: check diff after GetQueryCmd
	moduleBasic.GetQueryCmd(cdc)

	appModule := gov.NewAppModule(gk, gk.SupplyKeeper())
	require.Equal(t, types.ModuleName, appModule.Name())

	// todo: check diff after RegisterInvariants
	appModule.RegisterInvariants(&crisisKeeper)

	require.Equal(t, types.RouterKey, appModule.Route())

	require.IsType(t, gov.NewHandler(gk), appModule.NewHandler())

	require.Equal(t, types.QuerierRoute, appModule.QuerierRoute())

	require.IsType(t, gov.NewQuerier(gk), appModule.NewQuerierHandler())

	require.Equal(t, []abci.ValidatorUpdate{}, appModule.InitGenesis(ctx, jsonMsg))

	require.Equal(t, jsonMsg, appModule.ExportGenesis(ctx))

	appModule.BeginBlock(ctx, abci.RequestBeginBlock{})

	appModule.EndBlock(ctx, abci.RequestEndBlock{})
}
