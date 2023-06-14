package cli

import (
	"fmt"
	"github.com/okx/okbchain/libs/cosmos-sdk/client/flags"
	"strings"

	"github.com/okx/okbchain/x/params/types"

	"github.com/okx/okbchain/libs/cosmos-sdk/client/context"
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(queryRoute string, cdc *codec.Codec) *cobra.Command {
	queryCmd := &cobra.Command{
		Use:   "params",
		Short: "Querying commands for the params module",
	}

	queryCmd.AddCommand(flags.GetCommands(
		GetCmdQueryParams(queryRoute, cdc),
		GetCmdQueryUpgrade(queryRoute, cdc),
		GetCmdQueryGasConfig(queryRoute, cdc),
	)...)

	return queryCmd
}

// GetCmdQueryParams implements the query params command.
func GetCmdQueryParams(queryRoute string, cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "params",
		Short: "Query parameters of params",
		Long: strings.TrimSpace(`Query parameters of params:

$ okbchaincli query params params
`),
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			route := fmt.Sprintf("custom/%s/%s", queryRoute, types.QueryParams)
			bz, _, err := cliCtx.QueryWithData(route, nil)
			if err != nil {
				return err
			}

			var params types.Params
			cdc.MustUnmarshalJSON(bz, &params)
			return cliCtx.PrintOutput(params)
		},
	}
}

// GetCmdQueryParams implements the query params command.
func GetCmdQueryGasConfig(queryRoute string, cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "gasconfig",
		Short: "Query parameters of gasconfig",
		Long: strings.TrimSpace(`Query parameters of gasconfig:
$ exchaincli query params gasconfig
`),
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			route := fmt.Sprintf("custom/%s/%s", queryRoute, types.QueryGasConfig)
			bz, _, err := cliCtx.QueryWithData(route, nil)
			if err != nil {
				return err
			}

			var params types.GasConfig
			cdc.MustUnmarshalJSON(bz, &params)
			return cliCtx.PrintOutput(params.GasConfig)
		},
	}
}
