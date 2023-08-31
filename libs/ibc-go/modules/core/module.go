package ibc

import (
	"context"
	"encoding/json"
	"fmt"

	ibcclient "github.com/okx/brczero/libs/ibc-go/modules/core/02-client"

	"math/rand"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	clientCtx "github.com/okx/brczero/libs/cosmos-sdk/client/context"
	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	codectypes "github.com/okx/brczero/libs/cosmos-sdk/codec/types"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/libs/cosmos-sdk/types/module"
	simulation2 "github.com/okx/brczero/libs/cosmos-sdk/x/simulation"
	clienttypes "github.com/okx/brczero/libs/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/okx/brczero/libs/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/okx/brczero/libs/ibc-go/modules/core/04-channel/types"
	host "github.com/okx/brczero/libs/ibc-go/modules/core/24-host"
	"github.com/okx/brczero/libs/ibc-go/modules/core/client/cli"
	"github.com/okx/brczero/libs/ibc-go/modules/core/keeper"
	"github.com/okx/brczero/libs/ibc-go/modules/core/simulation"
	"github.com/okx/brczero/libs/ibc-go/modules/core/types"
	abci "github.com/okx/brczero/libs/tendermint/abci/types"
	"github.com/spf13/cobra"
)

var (
	_ module.AppModuleAdapter      = AppModule{}
	_ module.AppModuleBasicAdapter = AppModuleBasic{}
	_ module.AppModuleSimulation   = AppModule{}
)

// AppModuleBasic defines the basic application module used by the ibc module.
type AppModuleBasic struct{}

var _ module.AppModuleBasic = AppModuleBasic{}

// Name returns the ibc module's name.
func (AppModuleBasic) Name() string {
	return host.ModuleName
}

// DefaultGenesis returns default genesis state as raw bytes for the ibc
// module.
func (AppModuleBasic) DefaultGenesis() json.RawMessage {
	return ModuleCdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the mint module.
func (AppModuleBasic) ValidateGenesis(bz json.RawMessage) error {
	var gs types.GenesisState
	if err := ModuleCdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", host.ModuleName, err)
	}

	return gs.Validate()
}

// RegisterRESTRoutes does nothing. IBC does not support legacy REST routes.
func (AppModuleBasic) RegisterRESTRoutes(ctx clientCtx.CLIContext, rtr *mux.Router) {}

func (AppModuleBasic) RegisterCodec(cdc *codec.Codec) {
	types.RegisterCodec(cdc)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the ibc module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(ctx clientCtx.CLIContext, mux *runtime.ServeMux) {
	clienttypes.RegisterQueryHandlerClient(context.Background(), mux, clienttypes.NewQueryClient(ctx))
	connectiontypes.RegisterQueryHandlerClient(context.Background(), mux, connectiontypes.NewQueryClient(ctx))
	channeltypes.RegisterQueryHandlerClient(context.Background(), mux, channeltypes.NewQueryClient(ctx))
}

// GetTxCmd returns the root tx command for the ibc module.
func (AppModuleBasic) GetTxCmd(cdc *codec.Codec) *cobra.Command {
	return nil
}

func (am AppModuleBasic) GetTxCmdV2(cdc *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	return cli.GetTxCmd(cdc, reg)
}

// GetQueryCmd returns no root query command for the ibc module.
func (AppModuleBasic) GetQueryCmd(cdc *codec.Codec) *cobra.Command {
	return nil
}

func (AppModuleBasic) GetQueryCmdV2(cdc *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command {
	return cli.GetQueryCmd(cdc, reg)
}

// RegisterInterfaces registers module concrete types into protobuf Any.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// AppModule implements an application module for the ibc module.
type AppModule struct {
	AppModuleBasic
	keeper *keeper.FacadedKeeper

	// create localhost by default
	createLocalhost bool
}

// NewAppModule creates a new AppModule object
func NewAppModule(k *keeper.FacadedKeeper) AppModule {
	ret := AppModule{
		keeper: k,
	}
	return ret
}

func (a AppModule) NewQuerierHandler() sdk.Querier {
	return nil
}

func (a AppModule) NewHandler() sdk.Handler {
	return NewHandler(*a.keeper)
}

// Name returns the ibc module's name.
func (AppModule) Name() string {
	return host.ModuleName
}

// RegisterInvariants registers the ibc module invariants.
func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {}

// Route returns the message routing key for the ibc module.
func (am AppModule) Route() string {
	return host.RouterKey
}

// QuerierRoute returns the ibc module's querier route name.
func (AppModule) QuerierRoute() string {
	return host.QuerierRoute
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	clienttypes.RegisterMsgServer(cfg.MsgServer(), am.keeper)
	connectiontypes.RegisterMsgServer(cfg.MsgServer(), am.keeper)
	channeltypes.RegisterMsgServer(cfg.MsgServer(), am.keeper)
	types.RegisterQueryService(cfg.QueryServer(), am.keeper.V2Keeper)
}

// InitGenesis performs genesis initialization for the ibc module. It returns
// no validator updates.
//func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONMarshaler, bz json.RawMessage) []abci.ValidatorUpdate {
func (am AppModule) InitGenesis(ctx sdk.Context, data json.RawMessage) []abci.ValidatorUpdate {
	return am.initGenesis(ctx, data)
}

func (am AppModule) initGenesis(ctx sdk.Context, data json.RawMessage) []abci.ValidatorUpdate {
	var gs types.GenesisState
	err := ModuleCdc.UnmarshalJSON(data, &gs)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal %s genesis state: %s", host.ModuleName, err))
	}
	InitGenesis(ctx, *am.keeper.V2Keeper, am.createLocalhost, &gs)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the exported genesis state as raw bytes for the ibc
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context) json.RawMessage {
	return am.exportGenesis(ctx)
}

func (am AppModule) exportGenesis(ctx sdk.Context) json.RawMessage {
	return ModuleCdc.MustMarshalJSON(ExportGenesis(ctx, *am.keeper.V2Keeper))
}

func lazyGenesis() json.RawMessage {
	ret := DefaultGenesisState()
	return ModuleCdc.MustMarshalJSON(&ret)
}

// BeginBlock returns the begin blocker for the ibc module.
func (am AppModule) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {
	ibcclient.BeginBlocker(ctx, am.keeper.V2Keeper.ClientKeeper)
}

// EndBlock returns the end blocker for the ibc module. It returns no validator
// updates.
func (am AppModule) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

func (am AppModuleBasic) RegisterRouterForGRPC(cliCtx clientCtx.CLIContext, r *mux.Router) {

}

// ____________________________________________________________________________

// AppModuleSimulation functions

// GenerateGenesisState creates a randomized GenState of the ibc module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenState(simState)
}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simulation2.WeightedProposalContent {
	return nil
}

// RandomizedParams returns nil since IBC doesn't register parameter changes.
func (AppModule) RandomizedParams(_ *rand.Rand) []simulation2.ParamChange {
	return nil
}

// RegisterStoreDecoder registers a decoder for ibc module's types
func (am AppModule) RegisterStoreDecoder(sdr sdk.StoreDecoderRegistry) {
	sdr[host.StoreKey] = simulation.NewDecodeStore(*am.keeper.V2Keeper)
}

// WeightedOperations returns the all the ibc module operations with their respective weights.
func (am AppModule) WeightedOperations(_ module.SimulationState) []simulation2.WeightedOperation {
	return nil
}
