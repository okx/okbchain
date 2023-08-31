package keeper

import (
	"testing"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	abci "github.com/okx/brczero/libs/tendermint/abci/types"
	"github.com/okx/brczero/x/distribution/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/go-amino"
)

func TestQueryParams(t *testing.T) {
	ctx, _, k, _, _ := CreateTestInputDefault(t, false, 1000)
	querior := NewQuerier(k)
	commnutyTax, err := querior(ctx, []string{types.QueryParams, types.ParamCommunityTax}, abci.RequestQuery{})
	require.NoError(t, err)

	var taxData sdk.Dec
	_ = amino.UnmarshalJSON(commnutyTax, &taxData)
	require.Equal(t, sdk.NewDecWithPrec(2, 2), taxData)

	enabled, err := querior(ctx, []string{types.QueryParams, types.ParamWithdrawAddrEnabled}, abci.RequestQuery{})
	require.True(t, err == nil)
	var enableData bool
	err1 := amino.UnmarshalJSON(enabled, &enableData)
	require.NoError(t, err1)
	require.Equal(t, true, enableData)

	distrType, err := querior(ctx, []string{types.QueryParams, types.ParamDistributionType}, abci.RequestQuery{})
	require.True(t, err == nil)
	var distrTypeData uint32
	err2 := amino.UnmarshalJSON(distrType, &distrTypeData)
	require.NoError(t, err2)
	require.Equal(t, types.DistributionTypeOffChain, distrTypeData)

	enabled, err = querior(ctx, []string{types.QueryParams, types.ParamWithdrawRewardEnabled}, abci.RequestQuery{})
	require.True(t, err == nil)
	err = amino.UnmarshalJSON(enabled, &enableData)
	require.NoError(t, err)
	require.Equal(t, true, enableData)

	enabled, err = querior(ctx, []string{types.QueryParams, types.ParamRewardTruncatePrecision}, abci.RequestQuery{})
	require.True(t, err == nil)
	var precision int64
	err = amino.UnmarshalJSON(enabled, &precision)
	require.NoError(t, err)
	require.Equal(t, int64(0), precision)

	_, err = querior(ctx, []string{"unknown"}, abci.RequestQuery{})
	require.Error(t, err)
	_, err = querior(ctx, []string{types.QueryParams, "unknown"}, abci.RequestQuery{})
	require.Error(t, err)
}

func TestQueryValidatorCommission(t *testing.T) {
	ctx, _, k, _, _ := CreateTestInputDefault(t, false, 1000)
	querior := NewQuerier(k)
	k.SetValidatorAccumulatedCommission(ctx, valOpAddr1, NewTestSysCoins(15, 1))

	bz, err := amino.MarshalJSON(types.NewQueryValidatorCommissionParams(valOpAddr1))
	require.NoError(t, err)
	commission, err := querior(ctx, []string{types.QueryValidatorCommission}, abci.RequestQuery{Data: bz})
	require.NoError(t, err)

	var data sdk.SysCoins
	err = amino.UnmarshalJSON(commission, &data)
	require.NoError(t, err)
	require.Equal(t, NewTestSysCoins(15, 1), data)
}

func TestQueryDelegatorWithdrawAddress(t *testing.T) {
	ctx, _, k, _, _ := CreateTestInputDefault(t, false, 1000)
	querior := NewQuerier(k)
	require.NoError(t, k.SetWithdrawAddr(ctx, valAccAddr1, valAccAddr2))

	bz, err := amino.MarshalJSON(types.NewQueryDelegatorWithdrawAddrParams(valAccAddr1))
	require.NoError(t, err)
	withdrawAddr, err := querior(ctx, []string{types.QueryWithdrawAddr}, abci.RequestQuery{Data: bz})
	require.NoError(t, err)

	var data sdk.AccAddress
	err = amino.UnmarshalJSON(withdrawAddr, &data)
	require.NoError(t, err)
	require.Equal(t, valAccAddr2, data)
}

func TestQueryCommunityPool(t *testing.T) {
	ctx, _, k, _, _ := CreateTestInputDefault(t, false, 1000)
	querior := NewQuerier(k)
	feePool := k.GetFeePool(ctx)
	feePool.CommunityPool = feePool.CommunityPool.Add(NewTestSysCoins(123, 2)...)
	k.SetFeePool(ctx, feePool)

	communityPool, err := querior(ctx, []string{types.QueryCommunityPool}, abci.RequestQuery{})
	require.NoError(t, err)

	var data sdk.SysCoins
	err1 := amino.UnmarshalJSON(communityPool, &data)
	require.NoError(t, err1)
	require.Equal(t, NewTestSysCoins(123, 2), data)
}
