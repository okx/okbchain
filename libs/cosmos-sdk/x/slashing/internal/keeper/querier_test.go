package keeper

import (
	"testing"

	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	"github.com/stretchr/testify/require"

	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/slashing/internal/types"
)

func TestNewQuerier(t *testing.T) {
	ctx, _, _, _, keeper := CreateTestInput(t, TestParams())
	querier := NewQuerier(keeper)

	query := abci.RequestQuery{
		Path: "",
		Data: []byte{},
	}

	_, err := querier(ctx, []string{"parameters"}, query)
	require.NoError(t, err)
}

func TestQueryParams(t *testing.T) {
	cdc := codec.New()
	ctx, _, _, _, keeper := CreateTestInput(t, TestParams())

	var params types.Params

	res, errRes := queryParams(ctx, keeper)
	require.NoError(t, errRes)

	err := cdc.UnmarshalJSON(res, &params)
	require.NoError(t, err)
	require.Equal(t, keeper.GetParams(ctx), params)
}
