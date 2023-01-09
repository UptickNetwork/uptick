package keeper

import (
	"github.com/UptickNetwork/uptick/x/erc721/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// GetParams returns the total set of erc20 parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramstore.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the erc20 parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// GetClassIDAndNFTID sets the erc20 parameters to the param space.
func (k Keeper) GetClassIDAndNFTID(ctx sdk.Context, msg *types.MsgConvertERC721) (string, string, error) {

	var (
		nftID   string
		classID string
		err     error
	)

	uTokenID := types.CreateTokenUID(msg.ContractAddress, msg.TokenId)
	savedNftID, savedClassID := types.GetNFTFromUID(string(k.GetNFTUIDPairByTokenUID(ctx, uTokenID)))
	nftID, err = getNftData(msg.NftId, msg.TokenId, savedNftID, 0)
	if err != nil {
		return "", "", err
	}

	classID, err = getNftData(msg.ClassId, msg.ContractAddress, savedClassID, 1)
	if err != nil {
		return "", "", err
	}

	return classID, nftID, nil

}

// GetContractAddressAndTokenID sets the erc20 parameters to the param space.
func (k Keeper) GetContractAddressAndTokenID(ctx sdk.Context, msg *types.MsgConvertNFT) (string, string, error) {

	var (
		contractAddress string
		tokenID         string
		err             error
	)

	uNftID := types.CreateNFTUID(msg.ClassId, msg.NftId)
	savedTokenID, saveContractAddress := types.GetNFTFromUID(string(k.GetTokenUIDPairByNFTUID(ctx, uNftID)))
	tokenID, err = getNftData(msg.TokenId, msg.NftId, savedTokenID, 2)
	if err != nil {
		return "", "", err
	}

	contractAddress, err = getNftData(msg.ContractAddress, msg.ClassId, saveContractAddress, 3)
	if err != nil {
		return "", "", err
	}

	// return tokenID, contractAddress, nil
	return contractAddress, tokenID, nil

}

// getNftData nftType 0:nftId 1:classId 2:contract address 3:tokenId
func getNftData(nftOrg string, nftPairOrg string, nftSaved string, nftType int) (string, error) {

	var nftRet string
	if nftOrg == "" {
		if nftSaved == "" {
			nftRet = createNftDataByType(nftPairOrg, nftType)
		} else {
			nftRet = nftSaved
		}
	} else {
		if nftSaved == "" {
			nftRet = nftOrg
		} else if nftSaved == nftOrg {
			nftRet = nftOrg
		} else {
			return "", getNftDataErrorByType(nftSaved, nftOrg, nftType)
		}
	}

	return nftRet, nil

}

// createNftDataByType nftType 0:nftId 1:classId 2:contract address 3:tokenId
func createNftDataByType(nftOrg string, nftType int) string {

	switch nftType {
	case 0:
		return types.CreateNFTIDFromTokenID(nftOrg)
	case 1:
		return types.CreateClassIDFromContractAddress(nftOrg)
	case 2:
		return types.CreateTokenIDFromNFTID(nftOrg)
	case 3:
		return types.CreateContractAddressFromClassID(nftOrg)
	default:
		return ""
	}

}

// createNftDataByType nftType 0:nftId 1:classId 2:contract address 3:tokenId
func getNftDataErrorByType(nftSaved string, nftOrg string, nftType int) error {

	switch nftType {
	case 0:
		return sdkerrors.Wrapf(types.ErrNftIdNotCorrect,
			"nft id is not correct expect %s - get %s",
			nftSaved, nftOrg)
	case 1:
		return sdkerrors.Wrapf(types.ErrClassIdNotCorrect,
			"class id is not correct expect %s - get %s",
			nftSaved, nftOrg)
	case 2:
		return sdkerrors.Wrapf(types.ErrTokenIdNotCorrect,
			"token id is not correct expect %s - get %s",
			nftSaved, nftOrg)
	case 3:
		return sdkerrors.Wrapf(types.ErrContractAddressNotCorrect,
			"contract address is not correct expect %s - get %s",
			nftSaved, nftOrg)
	default:
		return nil
	}

}
