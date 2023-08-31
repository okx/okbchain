package cli

import (
	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	interfacetypes "github.com/okx/brczero/libs/cosmos-sdk/codec/types"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the query commands for the ICA host submodule
func GetQueryCmd(cdc *codec.CodecProxy, reg interfacetypes.InterfaceRegistry) *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        "host",
		Short:                      "interchain-accounts host subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	queryCmd.AddCommand(
		GetCmdParams(cdc, reg),
		GetCmdPacketEvents(cdc, reg),
	)

	return queryCmd
}
