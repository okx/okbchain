package module

import (
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	clictx "github.com/okx/okbchain/libs/cosmos-sdk/client/context"
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	codectypes "github.com/okx/okbchain/libs/cosmos-sdk/codec/types"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

// AppModuleBasic is the standard form for basic non-dependant elements of an application module.
type AppModuleBasicAdapter interface {
	AppModuleBasic
	RegisterInterfaces(codectypes.InterfaceRegistry)
	// client functionality
	RegisterGRPCGatewayRoutes(clictx.CLIContext, *runtime.ServeMux)
	GetTxCmdV2(cdc *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command
	GetQueryCmdV2(cdc *codec.CodecProxy, reg codectypes.InterfaceRegistry) *cobra.Command

	RegisterRouterForGRPC(cliCtx clictx.CLIContext, r *mux.Router)
}

// AppModuleGenesis is the standard form for an application module genesis functions
type AppModuleGenesisAdapter interface {
	AppModuleGenesis
	AppModuleBasicAdapter
}

// AppModule is the standard form for an application module
type AppModuleAdapter interface {
	AppModule
	AppModuleGenesisAdapter
	// registers
	RegisterInvariants(sdk.InvariantRegistry)
	// RegisterServices allows a module to register services
	RegisterServices(Configurator)
}
