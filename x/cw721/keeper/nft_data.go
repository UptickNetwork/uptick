package keeper

import (
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// GetClassIDAndNFTID resolves class ID and NFT IDs from a MsgConvertCW721,
// using saved mappings or generating new IDs as needed.
func (k Keeper) GetClassIDAndNFTID(ctx sdk.Context, msg *types.MsgConvertCW721) (string, []string, error) {
	var (
		cosmosTokenIds []string
		nftId          string
		classId        string
		err            error
		nftOrg         string
	)

	for i, tokenId := range msg.TokenIds {
		uTokenId := types.CreateTokenUID(msg.ContractAddress, tokenId)
		savedNftId, savedClassId := types.GetNFTFromUID(string(k.GetNFTUIDPairByTokenUID(ctx, uTokenId)))

		nftOrg = ""
		if len(msg.NftIds) > i {
			nftOrg = msg.NftIds[i]
		}
		nftId, err = getNftData(nftOrg, tokenId, savedNftId, 0)
		if err != nil {
			return "", nil, err
		}
		cosmosTokenIds = append(cosmosTokenIds, nftId)

		classId, err = getNftData(msg.ClassId, msg.ContractAddress, savedClassId, 1)
		if err != nil {
			return "", nil, err
		}
	}

	return classId, cosmosTokenIds, nil
}

// GetContractAddressAndTokenIds resolves the contract address and token IDs
// from a MsgConvertNFT, using saved mappings or generating new IDs as needed.
func (k Keeper) GetContractAddressAndTokenIds(ctx sdk.Context, msg *types.MsgConvertNFT) (string, []string, error) {
	var (
		evmContractAddress string
		evmTokenIds        []string
		err                error
	)

	pair, err := k.GetPair(ctx, msg.ClassId)
	if err != nil {
		// No registered pair: generate token IDs, then use the provided
		// contract or instantiate a new CW721 (module as minter).
		msg.TokenIds, err = getNftDatas(msg.TokenIds, msg.NftIds, nil, 2)
		if err != nil {
			return "", nil, err
		}
		if strings.TrimSpace(msg.ContractAddress) != "" {
			return "", nil, sdkerrors.Wrapf(
				types.ErrContractAddressNotCorrect,
				"unregistered class %s cannot bind to a caller-supplied contract", msg.ClassId,
			)
		}
		contractAddress, err := k.DeployCW721Contract(ctx, msg)
		if err != nil {
			return "", nil, err
		}
		return contractAddress, msg.TokenIds, nil
	}

	var (
		savedTokenIds        []string
		savedContractAddress string
		savedTokenId         string
		tempContractAddress  string
	)

	for _, nftId := range msg.NftIds {
		uNftID := types.CreateNFTUID(msg.ClassId, nftId)
		savedTokenId, tempContractAddress = types.GetNFTFromUID(string(k.GetTokenUIDPairByNFTUID(ctx, uNftID)))
		if tempContractAddress != "" {
			savedContractAddress = tempContractAddress
		}
		savedTokenIds = append(savedTokenIds, savedTokenId)
	}

	evmTokenIds, err = getNftDatas(msg.TokenIds, msg.NftIds, savedTokenIds, 2)
	if err != nil {
		return "", nil, err
	}

	evmContractAddress, err = getNftData(msg.ContractAddress, msg.ClassId, savedContractAddress, 3)
	if evmContractAddress == "" {
		evmContractAddress = pair.Cw721Address
	}
	if err != nil {
		return "", nil, err
	}

	return evmContractAddress, evmTokenIds, nil
}

// nftType: 0:nftId 1:classId 2:tokenId 3:contract address
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

func getNftDatas(nftOrgs []string, nftPairOrgs []string, nftSaveds []string, nftType int) ([]string, error) {
	var rets []string
	var nftSaved = ""
	var nftOrg = ""
	nftLen := len(nftPairOrgs)
	for n := 0; n < nftLen; n++ {
		if nftSaveds != nil {
			nftSaved = nftSaveds[n]
		}
		if nftOrgs != nil && nftLen == len(nftOrgs) {
			nftOrg = nftOrgs[n]
		}
		ret, err := getNftData(nftOrg, nftPairOrgs[n], nftSaved, nftType)
		if err != nil {
			return nil, err
		}
		rets = append(rets, ret)
	}
	return rets, nil
}

// nftType: 0:nftId 1:classId 2:tokenId 3:contract address
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

// nftType: 0:nftId 1:classId 2:tokenId 3:contract address
func getNftDataErrorByType(nftSaved string, nftOrg string, nftType int) error {
	switch nftType {
	case 0:
		return sdkerrors.Wrapf(types.ErrNftIdNotCorrect,
			"nft id is not correct expect %s - get %s", nftSaved, nftOrg)
	case 1:
		return sdkerrors.Wrapf(types.ErrClassIdNotCorrect,
			"class id is not correct expect %s - get %s", nftSaved, nftOrg)
	case 2:
		return sdkerrors.Wrapf(types.ErrTokenIdNotCorrect,
			"token id is not correct expect %s - get %s", nftSaved, nftOrg)
	case 3:
		return sdkerrors.Wrapf(types.ErrContractAddressNotCorrect,
			"contract address is not correct expect %s - get %s", nftSaved, nftOrg)
	default:
		return nil
	}
}
