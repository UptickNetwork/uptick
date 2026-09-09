package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/x/nft"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

var _ types.QueryServer = Keeper{}

// Supply queries the total supply of a given denom or owner
func (k Keeper) Supply(c context.Context, request *types.QuerySupplyRequest) (*types.QuerySupplyResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	var supply uint64
	switch {
	case len(request.Owner) == 0 && len(request.DenomId) > 0:
		supply = k.GetTotalSupply(ctx, request.DenomId)
	case len(request.Owner) == 0 && len(request.DenomId) == 0:
		return nil, status.Errorf(codes.InvalidArgument, "must specify at least one of owner or denom_id")
	default:
		// Owner was provided. If DenomId is empty, GetTotalSupplyOfOwner
		// would return 0 (no class to scope the lookup), silently lying
		// "owner owns nothing". Require an explicit denom_id instead.
		if len(request.DenomId) == 0 {
			return nil, status.Error(codes.InvalidArgument, "denom_id is required")
		}
		owner, err := sdk.AccAddressFromBech32(request.Owner)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid owner address %s", request.Owner)
		}
		supply = k.GetTotalSupplyOfOwner(ctx, request.DenomId, owner)
	}
	return &types.QuerySupplyResponse{Amount: supply}, nil
}

// NFTsOfOwner queries the NFTs of the specified owner
func (k Keeper) NFTsOfOwner(c context.Context, request *types.QueryNFTsOfOwnerRequest) (*types.QueryNFTsOfOwnerResponse, error) {
	if len(request.Owner) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "owner address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(request.Owner); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid owner address %s", request.Owner)
	}

	r := &nft.QueryNFTsRequest{
		ClassId: request.DenomId,
		Owner:   request.Owner,
	}
	page, err := shapePageRequest(request.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	r.Pagination = page

	result, err := k.nk.NFTs(c, r)
	if err != nil {
		return nil, err
	}

	var denomMap = make(map[string][]string)
	var denoms []string
	for _, token := range result.Nfts {
		if denomMap[token.ClassId] == nil {
			denomMap[token.ClassId] = []string{}
			denoms = append(denoms, token.ClassId)
		}
		denomMap[token.ClassId] = append(denomMap[token.ClassId], token.Id)
	}

	var idc []types.IDCollection
	for _, denomID := range denoms {
		idc = append(idc, types.IDCollection{
			DenomId:  denomID,
			TokenIds: denomMap[denomID],
		})
	}

	response := &types.QueryNFTsOfOwnerResponse{
		Owner: &types.Owner{
			Address:       request.Owner,
			IDCollections: idc,
		},
		Pagination: result.Pagination,
	}

	return response, nil
}

// Collection queries the NFTs of the specified denom
func (k Keeper) Collection(c context.Context, request *types.QueryCollectionRequest) (*types.QueryCollectionResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	denom, err := k.GetDenomInfo(ctx, request.DenomId)
	if err != nil {
		return nil, err
	}

	r := &nft.QueryNFTsRequest{
		ClassId: request.DenomId,
	}
	page, err := shapePageRequest(request.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	r.Pagination = page

	result, err := k.nk.NFTs(c, r)
	if err != nil {
		return nil, err
	}

	var nfts []types.BaseNFT
	for _, token := range result.Nfts {
		owner := k.nk.GetOwner(ctx, request.DenomId, token.Id)

		// A legacy / migrated NFT may carry nil Data or undecodable metadata.
		// Match GetNFTs: downgrade to empty metadata so every minted NFT
		// shows up in the Collection query.
		nftMetadata, mdErr := types.UnmarshalNFTMetadata(k.cdc, token.Data.GetValue())
		if mdErr != nil {
			ctx.Logger().Debug("Collection: NFT with undecodable metadata; substituting empty",
				"denom", request.DenomId, "token_id", token.Id, "err", mdErr.Error())
			nftMetadata = types.NFTMetadata{}
		}

		nfts = append(nfts, types.BaseNFT{
			Id:      token.Id,
			URI:     token.Uri,
			UriHash: token.UriHash,
			Name:    nftMetadata.Name,
			Owner:   owner.String(),
			Data:    nftMetadata.Data,
		})
	}

	collection := &types.Collection{
		Denom: *denom,
		NFTs:  nfts,
	}

	response := &types.QueryCollectionResponse{
		Collection: collection,
		Pagination: result.Pagination,
	}

	return response, nil
}

// Denom queries the definition of a given denom
func (k Keeper) Denom(c context.Context, request *types.QueryDenomRequest) (*types.QueryDenomResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	denom, err := k.GetDenomInfo(ctx, request.DenomId)
	if err != nil {
		return nil, err
	}
	return &types.QueryDenomResponse{Denom: denom}, nil
}

// Denoms queries all the denoms
func (k Keeper) Denoms(c context.Context, req *types.QueryDenomsRequest) (*types.QueryDenomsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	page, err := shapePageRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	result, err := k.nk.Classes(c, &nft.QueryClassesRequest{
		Pagination: page,
	})
	if err != nil {
		return nil, err
	}

	var denoms []types.Denom
	for _, class := range result.Classes {
		// GetDenomInfo now tolerates nil Data (post M-C fix); only real
		// errors (e.g. class vanished between iterations) bubble up.
		d, err := k.GetDenomInfo(ctx, class.Id)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "denom %s: %s", class.Id, err.Error())
		}
		denoms = append(denoms, *d)
	}

	return &types.QueryDenomsResponse{
		Denoms:     denoms,
		Pagination: result.Pagination,
	}, nil
}

// NFT queries the NFT for the given denom and token ID
func (k Keeper) NFT(c context.Context, request *types.QueryNFTRequest) (*types.QueryNFTResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	nft, err := k.GetNFT(ctx, request.DenomId, request.TokenId)
	if err != nil {
		return nil, sdkerrors.Wrapf(types.ErrUnknownNFT, "invalid NFT %s from collection %s: %v", request.TokenId, request.DenomId, err)
	}

	baseNFT, ok := nft.(types.BaseNFT)
	if !ok {
		return nil, sdkerrors.Wrapf(types.ErrInvalidNFT, "invalid type NFT %s from collection %s", request.TokenId, request.DenomId)
	}

	return &types.QueryNFTResponse{NFT: &baseNFT}, nil
}
