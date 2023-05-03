package keeper

import (
	"fmt"
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
func (k Keeper) GetClassIDAndNFTID(ctx sdk.Context, msg *types.MsgConvertERC721) (string, []string, error) {

	var (
		nftIds  []string
		nftId   string
		classId string
		err     error
		nftOrg  string
	)

	for i, tokenId := range msg.TokenIds {

		uTokenId := types.CreateTokenUID(msg.ContractAddress, tokenId)
		savedNftId, savedClassId := types.GetNFTFromUID(string(k.GetNFTUIDPairByTokenUID(ctx, uTokenId)))
		//
		fmt.Printf("###xxl GetClassIDAndNFTID 1 savedNftId %v,savedClassId %v \n", savedNftId, savedClassId)
		nftOrg = ""
		if len(msg.NftIds) > i {
			nftOrg = msg.NftIds[i]
		}
		nftId, err = getNftData(nftOrg, tokenId, savedNftId, 0)

		nftIds = append(nftIds, nftId)
		if err != nil {
			return "", nil, err
		}

		classId, err = getNftData(msg.ClassId, msg.ContractAddress, savedClassId, 1)
		if err != nil {
			return "", nil, err
		}
	}

	return classId, nftIds, nil

}

// GetContractAddressAndTokenIds sets the erc721 parameters to the param space.
func (k Keeper) GetContractAddressAndTokenIds(ctx sdk.Context, msg *types.MsgConvertNFT) (string, []string, error) {

	var (
		contractAddress string
		tokenIds        []string
		err             error
	)

	pair, err := k.GetPair(ctx, msg.ClassId)
	if err != nil {

		fmt.Printf("###xxl GetContractAddressAndTokenIds 4 %v %v \n", msg.TokenIds, msg.NftIds)
		msg.TokenIds, _ = getNftDatas(msg.NftIds, msg.NftIds, nil, 3)
		//Stop here ... ...
		erc721ContractAddress, err := k.DeployERC721Contract(ctx, msg)
		if err == nil {
			contractAddress = erc721ContractAddress.String()
		} else {
			fmt.Printf("###xxl GetContractAddressAndTokenIds 6 %v \n", err)
		}

		return contractAddress, msg.TokenIds, nil

	} else {
		var (
			savedTokenIds        []string
			savedContractAddress string
			savedTokenId         string
			tempContractAddress  string
		)

		for _, nftId := range msg.NftIds {

			uNftID := types.CreateNFTUID(msg.ClassId, nftId)
			fmt.Printf("###xxl GetContractAddressAndTokenIds 0 uNftID %v \n", uNftID)

			savedTokenId, tempContractAddress = types.GetNFTFromUID(string(k.GetTokenUIDPairByNFTUID(ctx, uNftID)))
			if tempContractAddress != "" {
				savedContractAddress = tempContractAddress
			}
			fmt.Printf("###xxl GetContractAddressAndTokenIds 1 savedTokenId %v,saveContractAddress %v \n", savedTokenId, savedContractAddress)

			savedTokenIds = append(savedTokenIds, savedTokenId)
		}

		fmt.Printf("###xxl GetContractAddressAndTokenIds 1 msg %v \n", msg.TokenIds)
		tokenIds, err = getNftDatas(msg.TokenIds, msg.NftIds, savedTokenIds, 3)
		fmt.Printf("###xxl GetContractAddressAndTokenIds 2 tokenIds %v \n", tokenIds)
		if err != nil {
			return "", nil, err
		}

		contractAddress, err = getNftData(msg.ContractAddress, msg.ClassId, savedContractAddress, 2)
		fmt.Printf("###xxl GetContractAddressAndTokenIds 3 contractAddress %v \n", contractAddress)

		if contractAddress == "" {
			contractAddress = pair.Erc721Address
		}

		if err != nil {
			return "", nil, err
		}
		return contractAddress, tokenIds, nil

	}

}

func getNftDatas(nftOrgs []string, nftPairOrgs []string, nftSaveds []string, nftType int) ([]string, error) {

	var rets []string
	var nftSaved = ""
	var nftOrg = ""
	nftLen := len(nftPairOrgs)
	fmt.Printf("###xxl getNftDatas nftOrgs %v,Len %v \n", nftOrgs, nftLen)
	for n := 0; n < nftLen; n++ {
		if nftSaveds != nil {
			nftSaved = nftSaveds[n]
		}
		if nftOrgs != nil && nftLen == len(nftOrgs) {
			nftOrg = nftOrgs[n]
		}
		fmt.Printf("###xxl getNftDatas nftOrgs %v,nftPairOrgs %v,nftSaved %v \n", nftOrg, nftPairOrgs[n], nftSaved)

		ret, err := getNftData(nftOrg, nftPairOrgs[n], nftSaved, nftType)
		if err != nil {
			return nil, err
		} else {
			rets = append(rets, ret)
		}
	}

	return rets, nil
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
