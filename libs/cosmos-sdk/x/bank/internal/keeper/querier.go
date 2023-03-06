package keeper

import (
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	"github.com/okx/okbchain/libs/cosmos-sdk/types/query"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/bank/internal/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/bank/internal/typesadapter"
	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
)

const (
	// query balance path
	QueryBalance     = "balances"
	GrpcQueryBalance = "grpc_balances"
)

// NewQuerier returns a new sdk.Keeper instance.
func NewQuerier(k Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) ([]byte, error) {
		switch path[0] {
		case QueryBalance:
			return queryBalance(ctx, req, k)
		case GrpcQueryBalance:
			return grpcQueryBalanceAdapter(ctx, req, k)
		default:
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unknown query path: %s", path[0])
		}
	}
}

// queryBalance fetch an account's balance for the supplied height.
// Height and account address are passed as first and second path components respectively.
func queryBalance(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var ret []byte
	var params types.QueryBalanceParams
	if err := types.ModuleCdc.UnmarshalJSON(req.Data, &params); err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}
	coins := k.GetCoins(ctx, params.Address)
	if coins == nil {
		coins = sdk.NewCoins()
	}

	bz, err := codec.MarshalJSONIndent(types.ModuleCdc, coins)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	ret = bz

	return ret, nil
}

func grpcQueryBalanceAdapter(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	bk, ok := k.(*BaseKeeper)
	var ret []byte
	var a sdk.AccAddress
	var er error
	if ok {
		protoReq := typesadapter.QueryAllBalancesRequest{}
		if err := bk.marshal.GetProtocMarshal().UnmarshalBinaryBare(req.Data, &protoReq); nil != err {
			return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
		}
		a, er = sdk.AccAddressFromBech32(protoReq.Address)
		if nil != er {
			return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, er.Error())
		}
		coins := k.GetCoins(ctx, a)
		if coins == nil {
			coins = sdk.NewCoins()
		}
		bs := make(sdk.CoinAdapters, 0)
		for _, c := range coins {
			var ada sdk.CoinAdapter
			if c.Denom == sdk.DefaultBondDenom {
				ada = sdk.CoinAdapter{
					Denom:  sdk.DefaultIbcWei,
					Amount: sdk.NewIntFromBigInt(c.Amount.BigInt()),
				}
			} else {
				ada = sdk.CoinAdapter{
					Denom:  c.Denom,
					Amount: sdk.NewIntFromBigInt(c.Amount.BigInt()),
				}
			}

			bs = append(bs, ada)
		}
		resp := typesadapter.QueryAllBalancesResponse{
			Balances:   bs,
			Pagination: &query.PageResponse{},
		}
		bz, err := bk.marshal.GetProtocMarshal().MarshalBinaryBare(&resp)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
		}
		ret = bz
	} else {
		var params types.QueryBalanceParams
		if err := types.ModuleCdc.UnmarshalJSON(req.Data, &params); err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
		}
		coins := k.GetCoins(ctx, params.Address)
		if coins == nil {
			coins = sdk.NewCoins()
		}

		bz, err := codec.MarshalJSONIndent(types.ModuleCdc, coins)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
		}
		ret = bz
	}

	return ret, nil
}
