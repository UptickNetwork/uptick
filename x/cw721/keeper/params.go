package keeper

import (
	"github.com/UptickNetwork/uptick/x/cw721/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strconv"
)

const UPTICK_CW721_LABLE = "Uptick CW721"
const UPTICK_CW721_NAME = "Uptick CW721"
const UPTICK_CW721_SYMBOL = "UCW721"

const BASE_WASM_URL = "https://raw.githubusercontent.com/UptickNetwork/uptick/main/release/wasm/"

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
func (k Keeper) GetClassIDAndNFTID(ctx sdk.Context, msg *types.MsgConvertCW721) (string, []string, error) {

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

		if msg.ClassId != "" && msg.ClassId != classId {

			return "", nil, sdkerrors.Wrapf(types.ErrContractAddressNotCorrect,
				"nft id is not correct expect %s - get %s",
				msg.ClassId, classId)
		}
	}

	return classId, nftIds, nil

}

func (k Keeper) LoadCw721Base(ctx sdk.Context) (error, uint64) {

	// come to load cw721 base ...
	var codeId uint64
	var err error
	resultBytes := k.GetWasmCode(ctx, types.AccModuleAddress.String())
	codeId, _ = strconv.ParseUint(string(resultBytes), 10, 64)

	if codeId <= 0 {
		codeId, err = k.StoreWasmContract(ctx, BASE_WASM_URL+"/cw721_base.wasm", types.AccModuleAddress.String())
		if err != nil {
			k.Logger(ctx).Error("LoadCw721Base ", "err:", err.Error())
			return err, 0
		} else {
			k.Logger(ctx).Info("LoadCw721Base codeId ", "codeId is : ", codeId)
			k.SetWasmCode(ctx, types.AccModuleAddress.String(), codeId)
		}
	}

	return nil, codeId
}

// GetContractAddressAndTokenIds sets the CW721 parameters to the param space.
func (k Keeper) GetContractAddressAndTokenIds(ctx sdk.Context, msg *types.MsgConvertNFT) (string, []string, error) {

	var (
		contractAddress string
		tokenIds        []string
		err             error
	)

	pair, err := k.GetPair(ctx, msg.ClassId)
	if err != nil {

		var codeId uint64
		resultBytes := k.GetWasmCode(ctx, types.AccModuleAddress.String())
		codeId, _ = strconv.ParseUint(string(resultBytes), 10, 64)

		err, codeId = k.LoadCw721Base(ctx)
		if err != nil {
			return "", nil, err
		}

		k.Logger(ctx).Info("GetContractAddressAndTokenIds", "codeId is ", codeId)
		contractAddress, err = k.InstantiateWasmContract(
			ctx,
			types.AccModuleAddress.String(),
			codeId,
			UPTICK_CW721_LABLE,
			UPTICK_CW721_NAME,
			UPTICK_CW721_SYMBOL,
			types.AccModuleAddress.String(),
		)

		k.Logger(ctx).Info("GetContractAddressAndTokenIds", "contractAddress", contractAddress)
		if err == nil {
			k.Logger(ctx).Info(" GetContractAddressAndTokenIds", "contractAddress", contractAddress, "NftIds", msg.NftIds)
			return contractAddress, msg.NftIds, err
		} else {
			k.Logger(ctx).Error(" GetContractAddressAndTokenIds", "NftIds", msg.NftIds, "error", err.Error())
			return "", msg.NftIds, err
		}

	} else {
		var (
			savedTokenIds []string
			savedTokenId  string
		)

		for _, nftId := range msg.NftIds {

			uNftID := types.CreateNFTUID(msg.ClassId, nftId)

			savedTokenId, _ = types.GetNFTFromUID(string(k.GetTokenUIDPairByNFTUID(ctx, uNftID)))
			savedTokenIds = append(savedTokenIds, savedTokenId)
		}

		tokenIds, err = getNftDatas(msg.TokenIds, msg.NftIds, savedTokenIds, 2)
		if err != nil {
			return "", nil, err
		}
		k.Logger(ctx).Info(" GetContractAddressAndTokenIds", "tokenIds", tokenIds)
		//
		//contractAddress, err = getNftData(msg.ContractAddress, msg.ClassId, savedContractAddress, 3)
		//k.Logger(ctx).Info(" GetContractAddressAndTokenIds", "contractAddress", contractAddress)
		//
		//if contractAddress == "" {
		//	contractAddress = pair.Cw721Address
		//}
		//k.Logger(ctx).Info("	 GetContractAddressAndTokenIds", "pair.Cw721Address", pair.Cw721Address)
		//
		//if err != nil {
		//	return "", nil, err
		//}
		if msg.ContractAddress != "" && msg.ContractAddress != pair.Cw721Address {

			return "", nil, sdkerrors.Wrapf(types.ErrContractAddressNotCorrect,
				"nft id is not correct expect %s - get %s",
				msg.ContractAddress, pair.Cw721Address)
		}

		return pair.Cw721Address, tokenIds, nil

	}

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
			// nftRet = createNftDataByType(nftPairOrg, nftType)
			nftRet = nftPairOrg
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
