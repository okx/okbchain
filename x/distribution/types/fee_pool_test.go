package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
)

func TestValidateGenesis(t *testing.T) {

	fp := InitialFeePool()
	require.Nil(t, fp.ValidateGenesis())

	fp2 := FeePool{CommunityPool: sdk.SysCoins{{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDec(-1)}}}
	require.NotNil(t, fp2.ValidateGenesis())
}
