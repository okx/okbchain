package keeper

import (
	"context"
	"encoding/binary"
	"runtime/debug"

	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	"github.com/okx/brczero/libs/cosmos-sdk/store/prefix"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/brczero/libs/cosmos-sdk/types/errors"
	"github.com/okx/brczero/libs/cosmos-sdk/types/query"
	"github.com/okx/brczero/x/wasm/proxy"
	"github.com/okx/brczero/x/wasm/types"
	"github.com/okx/brczero/x/wasm/watcher"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = &grpcQuerier{}

type grpcQuerier struct {
	cdc             codec.CodecProxy
	storeKey        sdk.StoreKey
	storageStoreKey sdk.StoreKey
	keeper          types.ViewKeeper
	queryGasLimit   sdk.Gas
}

// NewGrpcQuerier constructor
func NewGrpcQuerier(cdc codec.CodecProxy, storeKey sdk.StoreKey, storageStoreKey sdk.StoreKey, keeper types.ViewKeeper, queryGasLimit sdk.Gas) *grpcQuerier { //nolint:revive
	return &grpcQuerier{cdc: cdc, storeKey: storeKey, storageStoreKey: storageStoreKey, keeper: keeper, queryGasLimit: queryGasLimit}
}

func (q grpcQuerier) ContractInfo(c context.Context, req *types.QueryContractInfoRequest) (*types.QueryContractInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	contractAddr, err := sdk.WasmAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	ctx := q.UnwrapSDKContext(c)
	defer q.release(ctx)

	rsp, err := queryContractInfo(ctx, contractAddr, q.keeper)
	switch {
	case err != nil:
		return nil, err
	case rsp == nil:
		return nil, types.ErrNotFound
	}
	return rsp, nil
}

func (q grpcQuerier) ContractHistory(c context.Context, req *types.QueryContractHistoryRequest) (*types.QueryContractHistoryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	contractAddr, err := sdk.WasmAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	r := make([]types.ContractCodeHistoryEntry, 0)
	prefixStore := q.PrefixStore(c, types.GetContractCodeHistoryElementPrefix(contractAddr))

	pageRes, err := query.FilteredPaginate(prefixStore, req.Pagination, func(key []byte, value []byte, accumulate bool) (bool, error) {
		if accumulate {
			var e types.ContractCodeHistoryEntry
			if err := q.cdc.GetProtocMarshal().Unmarshal(value, &e); err != nil {
				return false, err
			}
			e.Updated = nil // redact
			r = append(r, e)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryContractHistoryResponse{
		Entries:    r,
		Pagination: pageRes,
	}, nil
}

// ContractsByCode lists all smart contracts for a code id
func (q grpcQuerier) ContractsByCode(c context.Context, req *types.QueryContractsByCodeRequest) (*types.QueryContractsByCodeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.CodeId == 0 {
		return nil, sdkerrors.Wrap(types.ErrInvalid, "code id")
	}

	r := make([]string, 0)
	prefixStore := q.PrefixStore(c, types.GetContractByCodeIDSecondaryIndexPrefix(req.CodeId))

	pageRes, err := query.FilteredPaginate(prefixStore, req.Pagination, func(key []byte, value []byte, accumulate bool) (bool, error) {
		if accumulate {
			var contractAddr sdk.WasmAddress = key[types.AbsoluteTxPositionLen:]
			r = append(r, contractAddr.String())
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryContractsByCodeResponse{
		Contracts:  r,
		Pagination: pageRes,
	}, nil
}

func (q grpcQuerier) AllContractState(c context.Context, req *types.QueryAllContractStateRequest) (*types.QueryAllContractStateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	contractAddr, err := sdk.WasmAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	ctx := q.UnwrapSDKContext(c)
	defer q.release(ctx)

	if !q.keeper.HasContractInfo(ctx, contractAddr) {
		return nil, types.ErrNotFound
	}

	r := make([]types.Model, 0)
	prefixStore := q.keeper.GetStorageStore4Query(ctx, contractAddr)

	pageRes, err := query.FilteredPaginate(prefixStore, req.Pagination, func(key []byte, value []byte, accumulate bool) (bool, error) {
		if accumulate {
			r = append(r, types.Model{
				Key:   key,
				Value: value,
			})
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryAllContractStateResponse{
		Models:     r,
		Pagination: pageRes,
	}, nil
}

func (q grpcQuerier) RawContractState(c context.Context, req *types.QueryRawContractStateRequest) (*types.QueryRawContractStateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	contractAddr, err := sdk.WasmAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	ctx := q.UnwrapSDKContext(c)
	defer q.release(ctx)

	if !q.keeper.HasContractInfo(ctx, contractAddr) {
		return nil, types.ErrNotFound
	}
	rsp := q.keeper.QueryRaw(ctx, contractAddr, req.QueryData)
	return &types.QueryRawContractStateResponse{Data: rsp}, nil
}

func (q grpcQuerier) SmartContractState(c context.Context, req *types.QuerySmartContractStateRequest) (rsp *types.QuerySmartContractStateResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if err := req.QueryData.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid query data")
	}
	contractAddr, err := sdk.WasmAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	ctx := q.UnwrapSDKContext(c)
	defer q.release(ctx)
	ctx.SetGasMeter(sdk.NewGasMeter(q.queryGasLimit))

	// recover from out-of-gas panic
	defer func() {
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				err = sdkerrors.Wrapf(sdkerrors.ErrOutOfGas,
					"out of gas in location: %v; gasWanted: %d, gasUsed: %d",
					rType.Descriptor, ctx.GasMeter().Limit(), ctx.GasMeter().GasConsumed(),
				)
			default:
				err = sdkerrors.ErrPanic
			}
			rsp = nil
			moduleLogger(ctx).
				Debug("smart query contract",
					"error", "recovering panic",
					"contract-address", req.Address,
					"stacktrace", string(debug.Stack()))
		}
	}()

	bz, err := q.keeper.QuerySmart(ctx, contractAddr, req.QueryData)
	switch {
	case err != nil:
		return nil, err
	case bz == nil:
		return nil, types.ErrNotFound
	}
	return &types.QuerySmartContractStateResponse{Data: bz}, nil
}

func (q grpcQuerier) Code(c context.Context, req *types.QueryCodeRequest) (*types.QueryCodeResponse, error) {

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.CodeId == 0 {
		return nil, sdkerrors.Wrap(types.ErrInvalid, "code id")
	}

	ctx := q.UnwrapSDKContext(c)
	defer q.release(ctx)
	rsp, err := queryCode(ctx, req.CodeId, q.keeper)
	switch {
	case err != nil:
		return nil, err
	case rsp == nil:
		return nil, types.ErrNotFound
	}
	return &types.QueryCodeResponse{
		CodeInfoResponse: rsp.CodeInfoResponse,
		Data:             rsp.Data,
	}, nil
}

func (q grpcQuerier) Codes(c context.Context, req *types.QueryCodesRequest) (*types.QueryCodesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	r := make([]types.CodeInfoResponse, 0)
	prefixStore := q.PrefixStore(c, types.CodeKeyPrefix)

	pageRes, err := query.FilteredPaginate(prefixStore, req.Pagination, func(key []byte, value []byte, accumulate bool) (bool, error) {
		if accumulate {
			var c types.CodeInfo
			if err := q.cdc.GetProtocMarshal().Unmarshal(value, &c); err != nil {
				return false, err
			}
			r = append(r, types.CodeInfoResponse{
				CodeID:                binary.BigEndian.Uint64(key),
				Creator:               c.Creator,
				DataHash:              c.CodeHash,
				InstantiatePermission: c.InstantiateConfig,
			})
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryCodesResponse{CodeInfos: r, Pagination: pageRes}, nil
}

func queryContractInfo(ctx sdk.Context, addr sdk.WasmAddress, keeper types.ViewKeeper) (*types.QueryContractInfoResponse, error) {
	info := keeper.GetContractInfo(ctx, addr)
	if info == nil {
		return nil, types.ErrNotFound
	}
	// redact the Created field (just used for sorting, not part of public API)
	info.Created = nil
	return &types.QueryContractInfoResponse{
		Address:      addr.String(),
		ContractInfo: *info,
	}, nil
}

func queryCode(ctx sdk.Context, codeID uint64, keeper types.ViewKeeper) (*types.QueryCodeResponse, error) {
	if codeID == 0 {
		return nil, nil
	}
	res := keeper.GetCodeInfo(ctx, codeID)
	if res == nil {
		// nil, nil leads to 404 in rest handler
		return nil, nil
	}
	info := types.CodeInfoResponse{
		CodeID:                codeID,
		Creator:               res.Creator,
		DataHash:              res.CodeHash,
		InstantiatePermission: res.InstantiateConfig,
	}

	code, err := keeper.GetByteCode(ctx, codeID)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "loading wasm code")
	}

	return &types.QueryCodeResponse{CodeInfoResponse: &info, Data: code}, nil
}

func (q grpcQuerier) PinnedCodes(c context.Context, req *types.QueryPinnedCodesRequest) (*types.QueryPinnedCodesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	r := make([]uint64, 0)
	prefixStore := q.PrefixStore(c, types.PinnedCodeIndexPrefix)

	pageRes, err := query.FilteredPaginate(prefixStore, req.Pagination, func(key []byte, _ []byte, accumulate bool) (bool, error) {
		if accumulate {
			r = append(r, sdk.BigEndianToUint64(key))
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryPinnedCodesResponse{
		CodeIDs:    r,
		Pagination: pageRes,
	}, nil
}

func (q grpcQuerier) UnwrapSDKContext(c context.Context) sdk.Context {
	return sdk.UnwrapSDKContext(c)
}

func (q grpcQuerier) PrefixStore(c context.Context, pre []byte) sdk.KVStore {
	ctx := sdk.UnwrapSDKContext(c)
	if watcher.Enable() {
		return watcher.NewReadStore(ctx.GetWasmSimulateCache(), prefix.NewStore(ctx.KVStore(q.storeKey), pre))
	}
	return prefix.NewStore(ctx.KVStore(q.storeKey), pre)

}

func (q grpcQuerier) release(ctx sdk.Context) {
	if !watcher.Enable() {
		return
	}
	proxy.PutBackStorePool(ctx.MultiStore().(sdk.CacheMultiStore))
}
