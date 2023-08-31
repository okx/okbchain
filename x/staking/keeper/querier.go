package keeper

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/okx/brczero/libs/cosmos-sdk/client"
	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/brczero/libs/cosmos-sdk/types/errors"
	stakingtypes "github.com/okx/brczero/libs/cosmos-sdk/x/staking/types"
	abci "github.com/okx/brczero/libs/tendermint/abci/types"
	"github.com/okx/brczero/libs/tendermint/crypto"
	"github.com/okx/brczero/x/common"
	"github.com/okx/brczero/x/staking/types"
)

// NewQuerier creates a querier for staking REST endpoints
func NewQuerier(k Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err error) {
		switch path[0] {
		case types.QueryValidators:
			return queryValidators(ctx, req, k)
		case types.QueryValidator:
			return queryValidator(ctx, req, k)
		case types.QueryPool:
			return queryPool(ctx, k)
		case types.QueryParameters:
			return queryParameters(ctx, k)
		case types.QueryParams4IBC:
			return queryParams4IBC(ctx, k)
		case types.QueryUnbondingDelegation:
			return queryUndelegation(ctx, req, k)
		case types.QueryValidatorAllShares:
			return queryValidatorAllShares(ctx, req, k)
		case types.QueryAddress:
			return queryAddress(ctx, k)
		case types.QueryForAddress:
			return queryForAddress(ctx, req, k)
		case types.QueryForAccAddress:
			return queryForAccAddress(ctx, req)
		case types.QueryProxy:
			return queryProxy(ctx, req, k)
		case types.QueryDelegator:
			return queryDelegator(ctx, req, k)
		case types.QueryDelegatorValidators:
			return queryDelegatorValidators(ctx, req, k)
		case types.QueryDelegatorValidator:
			return queryDelegatorValidator(ctx, req, k)
		case types.QueryHistoricalInfo:
			return queryHistoricalInfo(ctx, req, k)
		case types.QueryValidatorDelegations:
			return queryValidatorDelegations(ctx, req, k)
		case types.QueryValidatorDelegator:
			return queryValidatorDelegator(ctx, req, k)

			// required by wallet
		case types.QueryDelegatorDelegations:
			return queryDelegatorDelegations(ctx, req, k)
		case types.QueryUnbondingDelegation2:
			return queryUndelegation2(ctx, req, k)
		default:
			return nil, types.ErrUnknownStakingQueryType()
		}
	}
}

func queryDelegator(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryDelegatorParams

	if err := types.ModuleCdc.UnmarshalJSON(req.Data, &params); err != nil {
		return nil, common.ErrUnMarshalJSONFailed(err.Error())
	}

	delegator, found := k.GetDelegator(ctx, params.DelegatorAddr)
	if !found {
		return nil, types.ErrNoDelegatorExisted(params.DelegatorAddr.String())
	}

	res, err := codec.MarshalJSONIndent(types.ModuleCdc, delegator)
	if err != nil {
		return nil, common.ErrMarshalJSONFailed(err.Error())
	}

	return res, nil
}

func queryDelegatorValidator(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryBondsParams
	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, common.ErrUnMarshalJSONFailed(err.Error())
	}

	delegator, found := k.GetDelegator(ctx, params.DelegatorAddr)
	if !found {
		return nil, types.ErrNoDelegatorExisted(params.DelegatorAddr.String())
	}

	foundV := false
	var validator types.Validator
	for _, val := range delegator.ValidatorAddresses {
		if !val.Equals(params.ValidatorAddr) {
			continue
		}
		validator, found = k.GetValidator(ctx, val)
		if !found {
			return nil, types.ErrNoValidatorFound(val.String())
		}
		foundV = true
		break
	}

	if !foundV {
		return nil, types.ErrCodeNoDelegatorValidator(params.DelegatorAddr.String(), params.ValidatorAddr.String())
	}

	res, err := codec.MarshalJSONIndent(types.ModuleCdc, validator)
	if err != nil {
		return nil, common.ErrMarshalJSONFailed(err.Error())
	}

	return res, nil
}

func queryValidatorDelegator(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryBondsParams
	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, common.ErrUnMarshalJSONFailed(err.Error())
	}

	delegator, found := k.GetDelegator(ctx, params.DelegatorAddr)
	if !found {
		return nil, types.ErrNoDelegatorExisted(params.DelegatorAddr.String())
	}

	find := false
	for _, val := range delegator.ValidatorAddresses {
		if val.Equals(params.ValidatorAddr) {
			find = true
			break
		}
	}

	if !find {
		return nil, types.ErrCodeNoDelegatorValidator(params.DelegatorAddr.String(), params.ValidatorAddr.String())
	}

	res, err := codec.MarshalJSONIndent(types.ModuleCdc, delegator)
	if err != nil {
		return nil, common.ErrMarshalJSONFailed(err.Error())
	}

	return res, nil
}

func queryDelegatorValidators(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryDelegatorParams
	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, common.ErrUnMarshalJSONFailed(err.Error())
	}

	delegator, found := k.GetDelegator(ctx, params.DelegatorAddr)
	if !found {
		return nil, types.ErrNoDelegatorExisted(params.DelegatorAddr.String())
	}

	var validators []types.Validator
	for _, val := range delegator.ValidatorAddresses {
		validator, found := k.GetValidator(ctx, val)
		if !found {
			return nil, types.ErrNoValidatorFound(val.String())
		}
		validators = append(validators, validator)
	}

	res, err := codec.MarshalJSONIndent(types.ModuleCdc, validators)
	if err != nil {
		return nil, common.ErrMarshalJSONFailed(err.Error())
	}

	return res, nil
}

func queryValidatorDelegations(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryValidatorParams

	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, common.ErrUnMarshalJSONFailed(err.Error())
	}

	sharesDelegations := k.GetValidatorAllShares(ctx, params.ValidatorAddr)
	var delegators []types.Delegator
	for _, shareDelegator := range sharesDelegations {
		delegator, found := k.GetDelegator(ctx, shareDelegator.DelAddr)
		if !found {
			return nil, types.ErrNoDelegatorExisted(shareDelegator.DelAddr.String())
		}
		delegators = append(delegators, delegator)
	}

	res, err := codec.MarshalJSONIndent(types.ModuleCdc, delegators)
	if err != nil {
		return nil, common.ErrMarshalJSONFailed(err.Error())
	}

	return res, nil
}

func queryValidators(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryValidatorsParams

	if err := types.ModuleCdc.UnmarshalJSON(req.Data, &params); err != nil {
		return nil, common.ErrUnMarshalJSONFailed(err.Error())
	}

	validators := k.GetAllValidators(ctx)

	var filteredVals []types.Validator
	if params.Status == "all" {
		filteredVals = validators
	} else {
		filteredVals = make([]types.Validator, 0, len(validators))
		for _, val := range validators {
			if strings.EqualFold(val.GetStatus().String(), params.Status) {
				filteredVals = append(filteredVals, val)
			}
		}

		start, end := client.Paginate(len(filteredVals), params.Page, params.Limit, int(k.MaxValidators(ctx)))
		if start < 0 || end < 0 {
			filteredVals = []types.Validator{}
		} else {
			filteredVals = filteredVals[start:end]
		}
	}

	res, err := codec.MarshalJSONIndent(types.ModuleCdc, filteredVals)
	if err != nil {
		return nil, common.ErrMarshalJSONFailed(err.Error())
	}

	return res, nil
}

func queryValidator(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryValidatorParams

	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, common.ErrUnMarshalJSONFailed(err.Error())
	}

	validator, found := k.GetValidator(ctx, params.ValidatorAddr)
	if !found {
		return nil, types.ErrNoValidatorFound(params.ValidatorAddr.String())
	}

	res, err := codec.MarshalJSONIndent(types.ModuleCdc, validator)
	if err != nil {
		return nil, common.ErrMarshalJSONFailed(err.Error())
	}

	return res, nil
}

func queryPool(ctx sdk.Context, k Keeper) ([]byte, error) {
	bondDenom := k.BondDenom(ctx)
	bondedPool := k.GetBondedPool(ctx)
	notBondedPool := k.GetNotBondedPool(ctx)
	if bondedPool == nil || notBondedPool == nil {
		return nil, types.ErrBondedPoolOrNotBondedIsNotExist()
	}

	pool := types.NewPool(
		notBondedPool.GetCoins().AmountOf(bondDenom),
		bondedPool.GetCoins().AmountOf(bondDenom),
	)

	res, err := codec.MarshalJSONIndent(types.ModuleCdc, pool)
	if err != nil {
		return nil, common.ErrMarshalJSONFailed(err.Error())
	}

	return res, nil
}

func queryParameters(ctx sdk.Context, k Keeper) ([]byte, error) {
	params := k.GetParams(ctx)

	res, err := codec.MarshalJSONIndent(types.ModuleCdc, params)
	if err != nil {
		return nil, common.ErrMarshalJSONFailed(err.Error())
	}

	return res, nil
}

func queryParams4IBC(ctx sdk.Context, k Keeper) ([]byte, error) {
	params := k.GetParams(ctx)

	//QueryParamsResponse
	ret := &stakingtypes.QueryParamsResponse{
		Params: stakingtypes.IBCParams{
			UnbondingTime:     params.UnbondingTime,
			MaxValidators:     uint32(params.MaxValidators),
			MaxEntries:        uint32(params.MaxValsToAddShares),
			HistoricalEntries: params.HistoricalEntries,
			BondDenom:         sdk.DefaultBondDenom,
		},
	}
	res, err := k.cdcMarshl.GetProtocMarshal().MarshalBinaryBare(ret)
	if err != nil {
		return nil, common.ErrMarshalJSONFailed(err.Error())
	}

	return res, nil
}

func queryProxy(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryDelegatorParams
	if err := types.ModuleCdc.UnmarshalJSON(req.Data, &params); err != nil {
		return nil, common.ErrUnMarshalJSONFailed(err.Error())
	}

	delAddrs := k.GetDelegatorsByProxy(ctx, params.DelegatorAddr)
	resp, err := codec.MarshalJSONIndent(types.ModuleCdc, delAddrs)
	if err != nil {
		return nil, common.ErrMarshalJSONFailed(err.Error())
	}

	return resp, nil
}

func queryValidatorAllShares(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryValidatorParams

	if err := types.ModuleCdc.UnmarshalJSON(req.Data, &params); err != nil {
		return nil, common.ErrUnMarshalJSONFailed(err.Error())
	}

	sharesResponses := k.GetValidatorAllShares(ctx, params.ValidatorAddr)
	resp, err := codec.MarshalJSONIndent(types.ModuleCdc, sharesResponses)
	if err != nil {
		return nil, common.ErrMarshalJSONFailed(err.Error())
	}

	return resp, nil
}

func queryUndelegation(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryDelegatorParams
	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, common.ErrUnMarshalJSONFailed(err.Error())
	}

	undelegation, found := k.GetUndelegating(ctx, params.DelegatorAddr)
	if !found {
		return nil, types.ErrNoUnbondingDelegation()
	}

	res, err := codec.MarshalJSONIndent(types.ModuleCdc, undelegation)
	if err != nil {
		return nil, common.ErrMarshalJSONFailed(err.Error())
	}

	return res, nil
}

func queryAddress(ctx sdk.Context, k Keeper) (res []byte, err error) {

	ovPairs := k.GetOperAndValidatorAddr(ctx)
	res, errRes := codec.MarshalJSONIndent(types.ModuleCdc, ovPairs)
	if errRes != nil {
		return nil, common.ErrMarshalJSONFailed(errRes.Error())
	}
	return res, nil
}

func queryForAddress(ctx sdk.Context, req abci.RequestQuery, k Keeper) (res []byte, err error) {
	validatorAddr := string(req.Data)
	if len(validatorAddr) != crypto.AddressSize*2 {
		return nil, types.ErrBadValidatorAddr()
	}

	operAddr, found := k.GetOperAddrFromValidatorAddr(ctx, validatorAddr)
	if !found {
		return nil, types.ErrNoValidatorFound(validatorAddr)
	}

	res, errRes := codec.MarshalJSONIndent(types.ModuleCdc, operAddr)
	if errRes != nil {
		return nil, common.ErrMarshalJSONFailed(errRes.Error())
	}
	return res, nil
}

func queryForAccAddress(ctx sdk.Context, req abci.RequestQuery) (res []byte, err error) {

	valAddr, errBech32 := sdk.ValAddressFromBech32(string(req.Data))
	if errBech32 != nil {
		return nil, common.ErrCreateAddrFromBech32Failed(errBech32.Error(), errBech32.Error())
	}

	accAddr := sdk.AccAddress(valAddr)

	res, errMarshal := codec.MarshalJSONIndent(types.ModuleCdc, accAddr)
	if errMarshal != nil {
		return nil, common.ErrMarshalJSONFailed(errMarshal.Error())
	}
	return res, nil
}

func queryUndelegation2(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryDelegatorUnbondingDelegationsRequest
	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, common.ErrUnMarshalJSONFailed(err.Error())
	}

	if params.DelegatorAddr == "" {
		return nil, status.Errorf(codes.InvalidArgument, "delegator address cannot be empty")
	}

	delAddr, err := sdk.AccAddressFromBech32(params.DelegatorAddr)
	if err != nil {
		return nil, err
	}

	un, pageRes, err := k.GetDelegatorUnbondingDelegations(ctx, delAddr, params.Pagination)
	if nil != err {
		return nil, err
	}
	if un == nil {
		un = make(stakingtypes.UnbondingDelegations, 0)
	}
	marsh := &types.QueryDelegatorUnbondingDelegationsResponse{
		UnbondingResponses: un,
		Pagination:         pageRes,
	}
	res, err := codec.MarshalJSONIndent(types.ModuleCdc, marsh)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return res, nil
}

func queryDelegatorDelegations(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryDelegatorDelegationsRequest

	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	delegationResps, err := k.DelegatorDelegations(ctx, &params)
	if delegationResps == nil {
		delegationResps = &types.QueryDelegatorDelegationsResponse{}
	}

	res, err := codec.MarshalJSONIndent(types.ModuleCdc, delegationResps)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return res, nil
}

func delegationsToDelegationResponses(
	ctx sdk.Context, k Keeper, delegations stakingtypes.Delegations,
) (stakingtypes.DelegationResponses, error) {

	resp := make(stakingtypes.DelegationResponses, len(delegations))
	for i, del := range delegations {
		delResp, err := delegationToDelegationResponse(ctx, k, del)
		if err != nil {
			return nil, err
		}

		resp[i] = delResp
	}

	return resp, nil
}

// ///
// utils
func delegationToDelegationResponse(ctx sdk.Context, k Keeper, del stakingtypes.Delegation) (stakingtypes.DelegationResponse, error) {
	val, found := k.GetValidator(ctx, del.ValidatorAddress)
	if !found {
		return stakingtypes.DelegationResponse{}, stakingtypes.ErrNoValidatorFound
	}

	return stakingtypes.NewDelegationResp(
		del.DelegatorAddress,
		del.ValidatorAddress,
		del.Shares,
		sdk.NewCoin(k.BondDenom(ctx), val.TokensFromShares(del.Shares).TruncateInt()),
	), nil
}

func DelegationsToDelegationResponses(
	ctx sdk.Context, k Keeper, delegations stakingtypes.Delegations,
) (stakingtypes.DelegationResponses, error) {
	resp := make(stakingtypes.DelegationResponses, len(delegations))

	for i, del := range delegations {
		delResp, err := DelegationToDelegationResponse(ctx, k, del)
		if err != nil {
			return nil, err
		}

		resp[i] = delResp
	}

	return resp, nil
}

func DelegationToDelegationResponse(ctx sdk.Context, k Keeper, del stakingtypes.Delegation) (stakingtypes.DelegationResponse, error) {
	val, found := k.GetValidator(ctx, del.GetValidatorAddr())
	if !found {
		return stakingtypes.DelegationResponse{}, stakingtypes.ErrNoValidatorFound
	}

	delegatorAddress, err := sdk.AccAddressFromBech32(del.DelegatorAddress.String())
	if err != nil {
		return stakingtypes.DelegationResponse{}, err
	}

	return stakingtypes.NewDelegationResp(
		delegatorAddress,
		del.GetValidatorAddr(),
		del.Shares,
		sdk.NewCoin(k.BondDenom(ctx), val.TokensFromShares(del.Shares).TruncateInt()),
	), nil
}

func queryHistoricalInfo(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	var params types.QueryHistoricalInfoParams

	err := types.ModuleCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	hi, found := k.GetHistoricalInfo(ctx, params.Height)
	if !found {
		return nil, types.ErrNoHistoricalInfo
	}

	res, err := codec.MarshalJSONIndent(types.ModuleCdc, hi)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return res, nil
}
