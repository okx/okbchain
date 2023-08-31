package token

import (
	"encoding/json"

	"github.com/gorilla/mux"
	"github.com/okx/brczero/libs/cosmos-sdk/client/context"
	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	"github.com/okx/brczero/libs/cosmos-sdk/types/module"
	"github.com/spf13/cobra"

	"github.com/okx/brczero/x/token/client/cli"
	"github.com/okx/brczero/x/token/client/rest"
	tokenTypes "github.com/okx/brczero/x/token/types"
)

var (
	_ module.AppModuleBasic = AppModuleBasic{}
)

// nolint
type AppModuleBasic struct{}

// nolint
func (AppModuleBasic) Name() string {
	return tokenTypes.ModuleName
}

// nolint
func (AppModuleBasic) RegisterCodec(cdc *codec.Codec) {
	RegisterCodec(cdc)
}

// nolint
func (AppModuleBasic) DefaultGenesis() json.RawMessage {
	return tokenTypes.ModuleCdc.MustMarshalJSON(defaultGenesisState())
}

// validateGenesis module validate genesis from json raw message
func (AppModuleBasic) ValidateGenesis(bz json.RawMessage) error {
	var data GenesisState
	err := tokenTypes.ModuleCdc.UnmarshalJSON(bz, &data)
	if err != nil {
		return err
	}
	return validateGenesis(data)
}

// RegisterRESTRoutes register rest routes
func (AppModuleBasic) RegisterRESTRoutes(ctx context.CLIContext, rtr *mux.Router) {
	rest.RegisterRoutes(ctx, rtr, ModuleName)
}

// GetTxCmd gets the root tx command of this module
func (AppModuleBasic) GetTxCmd(cdc *codec.Codec) *cobra.Command {
	return cli.GetTxCmd(tokenTypes.StoreKey, cdc)
}

// GetQueryCmd gets the root query command of this module
func (AppModuleBasic) GetQueryCmd(cdc *codec.Codec) *cobra.Command {
	return cli.GetQueryCmd(tokenTypes.StoreKey, cdc)
}
