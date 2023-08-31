package keeper

import (
	"context"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	"github.com/okx/brczero/libs/ibc-go/modules/apps/27-interchain-accounts/controller/types"
)

var _ types.QueryServer = Keeper{}

// Params implements the Query/Params gRPC method
func (q Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := q.GetParams(ctx)

	return &types.QueryParamsResponse{
		Params: &params,
	}, nil
}
