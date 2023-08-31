package types_test

import (
	"testing"

	"github.com/okx/brczero/libs/ibc-go/modules/apps/27-interchain-accounts/controller/types"
	"github.com/stretchr/testify/require"
)

func TestValidateParams(t *testing.T) {
	require.NoError(t, types.DefaultParams().Validate())
	require.NoError(t, types.NewParams(false).Validate())
}
