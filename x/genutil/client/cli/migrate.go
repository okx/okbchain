package cli

import (
	"fmt"
	"time"

	"github.com/okx/brczero/libs/tendermint/types"
	"github.com/spf13/cobra"

	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	"github.com/okx/brczero/libs/cosmos-sdk/server"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/libs/cosmos-sdk/version"
	extypes "github.com/okx/brczero/libs/cosmos-sdk/x/genutil"
	v018 "github.com/okx/brczero/x/genutil/client/legacy/v0_18"
)

var migrationMap = extypes.MigrationMap{
	"v0.18": v018.Migrate,
}

const (
	flagGenesisTime = "genesis-time"
	flagChainId     = "chain-id"
)

// MigrateGenesisCmd returns the cobra command to migrate
func MigrateGenesisCmd(_ *server.Context, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [target-version] [genesis-file]",
		Short: "Migrate genesis to a specified target version",
		Long: fmt.Sprintf(`Migrate the source genesis into the target version and print to STDOUT.

Example:
$ %s migrate v0.11 /path/to/genesis.json --chain-id=okbchain --genesis-time=2019-04-22T17:00:00Z
`, version.ServerName),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			importGenesis := args[1]

			genDoc, err := types.GenesisDocFromFile(importGenesis)
			if err != nil {
				return err
			}

			var initialState extypes.AppMap
			cdc.MustUnmarshalJSON(genDoc.AppState, &initialState)

			if migrationMap[target] == nil {
				return fmt.Errorf("unknown migration function version: %s", target)
			}

			newGenState := migrationMap[target](initialState)
			genDoc.AppState = cdc.MustMarshalJSON(newGenState)

			genesisTime := cmd.Flag(flagGenesisTime).Value.String()
			if genesisTime != "" {
				var t time.Time

				err := t.UnmarshalText([]byte(genesisTime))
				if err != nil {
					return err
				}

				genDoc.GenesisTime = t
			}

			chainId := cmd.Flag(flagChainId).Value.String()
			if chainId != "" {
				genDoc.ChainID = chainId
			}

			out, err := cdc.MarshalJSONIndent(genDoc, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println(string(sdk.MustSortJSON(out)))
			return nil
		},
	}

	cmd.Flags().String(flagGenesisTime, "", "Override genesis_time with this flag")
	cmd.Flags().String(flagChainId, "", "Override chain_id with this flag")

	return cmd
}
