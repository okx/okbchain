package cli

import (
	"encoding/json"
	"fmt"
	"os"

	tmtypes "github.com/okx/brczero/libs/tendermint/types"
	"github.com/spf13/cobra"

	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	"github.com/okx/brczero/libs/cosmos-sdk/server"
	"github.com/okx/brczero/libs/cosmos-sdk/types/module"
)

// Validate genesis command takes
func ValidateGenesisCmd(ctx *server.Context, cdc *codec.Codec, mbm module.BasicManager) *cobra.Command {
	return &cobra.Command{
		Use:   "validate-genesis [file]",
		Args:  cobra.RangeArgs(0, 1),
		Short: "validates the genesis file at the default location or at the location passed as an arg",
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			// Load default if passed no args, otherwise load passed file
			var genesis string
			if len(args) == 0 {
				genesis = ctx.Config.GenesisFile()
			} else {
				genesis = args[0]
			}

			fmt.Fprintf(os.Stderr, "validating genesis file at %s\n", genesis)

			var genDoc *tmtypes.GenesisDoc
			if genDoc, err = tmtypes.GenesisDocFromFile(genesis); err != nil {
				return fmt.Errorf("error loading genesis doc from %s: %s", genesis, err.Error())
			}

			var genState map[string]json.RawMessage
			if err = cdc.UnmarshalJSON(genDoc.AppState, &genState); err != nil {
				return fmt.Errorf("error unmarshalling genesis doc %s: %s", genesis, err.Error())
			}

			if err = mbm.ValidateGenesis(genState); err != nil {
				return fmt.Errorf("error validating genesis file %s: %s", genesis, err.Error())
			}

			// TODO test to make sure initchain doesn't panic

			fmt.Printf("File at %s is a valid genesis file\n", genesis)
			return nil
		},
	}
}
