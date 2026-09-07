package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/UptickNetwork/uptick/x/cw721/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

// TokenPairs returns all registered token pairs.
func (k Keeper) TokenPairs(
	c context.Context,
	req *types.QueryTokenPairsRequest,
) (*types.QueryTokenPairsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	var pairs []types.TokenPair
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)

	pageRes, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		var pair types.TokenPair
		if err := k.cdc.Unmarshal(value, &pair); err != nil {
			return err
		}
		pairs = append(pairs, pair)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTokenPairsResponse{
		TokenPairs: pairs,
		Pagination: pageRes,
	}, nil
}

// TokenPair returns a given registered token pair.
func (k Keeper) TokenPair(
	c context.Context,
	req *types.QueryTokenPairRequest,
) (*types.QueryTokenPairResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	id := k.GetTokenPairID(ctx, req.Token)
	if len(id) == 0 {
		return nil, status.Errorf(codes.NotFound, "token pair with token '%s'", req.Token)
	}

	pair, found := k.GetTokenPair(ctx, id)
	if !found {
		return nil, status.Errorf(codes.NotFound, "token pair with token '%s'", req.Token)
	}

	return &types.QueryTokenPairResponse{
		TokenPair: pair,
	}, nil
}

// Params returns the cw721 module params.
func (k Keeper) Params(
	c context.Context,
	_ *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

// WasmContract returns the registered token pair (and therefore the CW721
// contract) for the given class id. Port and channel are accepted for API /
// IBC-voucher compatibility, but the pair registry is keyed by class id, so
// the lookup resolves on ClassId alone. A missing pair is reported as NotFound
// — it must never return an empty "success" response, which used to silently
// hide the contract address from clients.
func (k Keeper) WasmContract(
	c context.Context,
	req *types.QueryWasmAddressRequest,
) (*types.QueryWasmContractResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	id := k.GetClassMap(ctx, req.ClassId)
	if len(id) == 0 {
		return nil, status.Errorf(codes.NotFound, "no token pair registered for class '%s'", req.ClassId)
	}

	pair, found := k.GetTokenPair(ctx, id)
	if !found {
		return nil, status.Errorf(codes.NotFound, "token pair for class '%s'", req.ClassId)
	}

	return &types.QueryWasmContractResponse{
		TokenPair: pair,
	}, nil
}
