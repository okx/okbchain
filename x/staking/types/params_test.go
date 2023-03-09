package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParams(t *testing.T) {
	p1 := DefaultDposParams()
	p2 := DefaultDposParams()

	ok := p1.Equal(p2)
	require.True(t, ok)

	p2.UnbondingTime = 60 * 60 * 24 * 2

	ok = p1.Equal(p2)
	require.False(t, ok)

	// validate
	p2 = p1
	p2.MaxValidators = 0
	require.Error(t, p2.Validate())

	p2 = p1
	p2.Epoch = 0
	require.Error(t, p2.Validate())

	p2 = p1
	p2.MaxValsToAddShares = 0
	require.Error(t, p2.Validate())

}
