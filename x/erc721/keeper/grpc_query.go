package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

var _ types.QueryServer = Keeper{}

// TokenPairs returns all registered pairs
func (k Keeper) TokenPairs(c context.Context, req *types.QueryTokenPairsRequest) (*types.QueryTokenPairsResponse, error) {
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
//
// The input is an untyped user-supplied string: the class-id namespace is
// tried first, with fallback to the contract map only when the input is
// unambiguously an EVM hex address (see GetTokenPairID for the shape-routing
// footgun this avoids).
func (k Keeper) TokenPair(c context.Context, req *types.QueryTokenPairRequest) (*types.QueryTokenPairResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	pair, found := lookupTokenPairQuery(k, ctx, req.Token)
	if !found {
		return nil, status.Errorf(codes.NotFound, "token pair with token '%s'", req.Token)
	}

	return &types.QueryTokenPairResponse{TokenPair: pair}, nil
}

// Params returns the params of the erc20 module
func (k Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

// EvmContract returns a given registered token pair. The `token` lookup key
// is derived from an IBC voucher (port + channel + class id) and is always a
// class-shaped identifier, so this RPC uses the unambiguous class namespace
// and never touches the contract map.
func (k Keeper) EvmContract(c context.Context, req *types.QueryEvmAddressRequest) (*types.QueryEvmAddressResponse, error) {

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.Port == "" || req.Channel == "" || req.ClassId == "" {
		return nil, status.Error(codes.InvalidArgument, "port, channel, and class_id are required")
	}

	ctx := sdk.UnwrapSDKContext(c)
	token := k.GetVoucherClassID(req.Port, req.Channel, req.ClassId)

	pair, err := k.GetPairByClass(ctx, token)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "token pair with token '%s'", token)
	}

	return &types.QueryEvmAddressResponse{TokenPair: pair}, nil

}

// lookupTokenPairQuery resolves a user-supplied token string for the TokenPair
// query: class namespace first, contract map only for unambiguous 40-nibble
// EVM addresses.
func lookupTokenPairQuery(k Keeper, ctx sdk.Context, token string) (types.TokenPair, bool) {
	if pair, err := k.GetPairByClass(ctx, token); err == nil {
		return pair, true
	}
	if common.IsHexAddress(token) {
		if pair, err := k.GetPairByEVM(ctx, token); err == nil {
			return pair, true
		}
	}
	return types.TokenPair{}, false
}
