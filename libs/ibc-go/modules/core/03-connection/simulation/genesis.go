package simulation

import (
	"math/rand"

	"github.com/okx/brczero/libs/cosmos-sdk/x/simulation"
	"github.com/okx/brczero/libs/ibc-go/modules/core/03-connection/types"
)

// GenConnectionGenesis returns the default connection genesis state.
func GenConnectionGenesis(_ *rand.Rand, _ []simulation.Account) types.GenesisState {
	return types.DefaultGenesisState()
}
