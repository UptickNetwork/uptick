package keeper

import (
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"github.com/UptickNetwork/uptick/x/erc721/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
)

// GetParams returns the total set of erc721 parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrefixParams)
	if len(bz) == 0 {
		return types.DefaultParams()
	}
	if err := k.cdc.Unmarshal(bz, &params); err != nil {
		k.Logger(ctx).Error("failed to unmarshal erc721 params", "error", err)
		return types.DefaultParams()
	}
	return params
}

// SetParams sets the erc721 parameters to the store.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to marshal erc721 params")
	}
	store.Set(types.KeyPrefixParams, bz)
	return nil
}

// GetEnableErc721 returns whether ERC721 conversion is enabled.
func (k Keeper) GetEnableErc721(ctx sdk.Context) bool {
	return k.GetParams(ctx).EnableErc721
}

// GetClassIDAndNFTID sets the erc721 parameters to the param space.
func (k Keeper) GetClassIDAndNFTID(ctx sdk.Context, msg *types.MsgConvertERC721) (string, []string, error) {

	var (
		CosmosTokenIds []string
		nftId          string
		classId        string
		err            error
		nftOrg         string
	)

	for i, tokenId := range msg.EvmTokenIds {

		uTokenId := types.CreateTokenUID(msg.EvmContractAddress, tokenId)
		savedPair := k.GetNFTUIDPairByTokenUID(ctx, uTokenId)
		var savedNftId, savedClassId string
		if len(savedPair) > 0 {
			savedNftId, savedClassId = types.GetNFTFromUID(string(savedPair))
			if savedNftId == "" || savedClassId == "" {
				return "", nil, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "invalid ERC721 NFT UID pair for token %s", uTokenId)
			}
		}

		nftOrg = ""
		if len(msg.CosmosTokenIds) > i {
			nftOrg = msg.CosmosTokenIds[i]
		}
		nftId, err = getNftData(nftOrg, tokenId, savedNftId, 0)

		CosmosTokenIds = append(CosmosTokenIds, nftId)
		if err != nil {
			return "", nil, err
		}

		classId, err = getNftData(msg.ClassId, msg.EvmContractAddress, savedClassId, 1)
		if err != nil {
			return "", nil, err
		}
	}

	return classId, CosmosTokenIds, nil

}

// GetContractAddressAndTokenIds sets the erc721 parameters to the param space.
func (k Keeper) GetContractAddressAndTokenIds(ctx sdk.Context, msg *types.MsgConvertNFT) (string, []string, error) {

	var (
		EvmContractAddress string
		EvmTokenIds        []string
		err                error
	)

	pair, err := k.GetPair(ctx, msg.ClassId)
	if err != nil {
		msg.EvmTokenIds, err = getNftDatas(msg.EvmTokenIds, msg.CosmosTokenIds, nil, 2)
		if err != nil {
			return "", nil, err
		}

		erc721ContractAddress, err := k.DeployERC721Contract(ctx, msg)
		if err != nil {
			return "", nil, err
		}
		if erc721ContractAddress == (common.Address{}) {
			return "", nil, sdkerrors.Wrap(types.ErrInternalTokenPair, "deployed erc721 contract address is empty")
		}

		return erc721ContractAddress.String(), msg.EvmTokenIds, nil

	} else {

		var (
			savedTokenIds        []string
			savedContractAddress string
			savedTokenId         string
			tempContractAddress  string
		)

		for _, nftId := range msg.CosmosTokenIds {

			uNftID := types.CreateNFTUID(msg.ClassId, nftId)

			savedTokenId, tempContractAddress = types.GetNFTFromUID(string(k.GetTokenUIDPairByNFTUID(ctx, uNftID)))
			if tempContractAddress != "" {
				savedContractAddress = tempContractAddress
			}
			savedTokenIds = append(savedTokenIds, savedTokenId)
		}

		EvmTokenIds, err = getNftDatas(msg.EvmTokenIds, msg.CosmosTokenIds, savedTokenIds, 2)
		if err != nil {
			return "", nil, err
		}

		EvmContractAddress, err = getNftData(msg.EvmContractAddress, msg.ClassId, savedContractAddress, 3)

		if EvmContractAddress == "" {
			EvmContractAddress = pair.Erc721Address
		}

		if err != nil {
			return "", nil, err
		}

		// A registered class has a canonical contract. Reject a caller-supplied
		// address that differs from the pair (this would allow conversion against
		// an external compatible contract and break the class↔contract identity).
		if EvmContractAddress != "" && strings.ToLower(EvmContractAddress) != pair.Erc721Address {
			return "", nil, sdkerrors.Wrapf(
				types.ErrContractAddressNotCorrect,
				"contract address is not correct, expect %s got %s",
				pair.Erc721Address, EvmContractAddress,
			)
		}
		EvmContractAddress = pair.Erc721Address

		// Enforce a strict one-to-one binding up front: if the resolved ERC721
		// tokenID is already forward-mapped to a NFT, that NFT MUST be the one
		// the caller is converting. Otherwise an attacker could pair their own
		// NFT with a victim's escrowed ERC721 token id and drain it.
		for i, tokenID := range EvmTokenIds {
			if tokenID == "" {
				continue
			}
			forward := k.GetNFTPairByContractTokenID(ctx, pair.Erc721Address, tokenID)
			if len(forward) != 0 {
				expectedNFTUID := types.CreateNFTUID(msg.ClassId, msg.CosmosTokenIds[i])
				if string(forward) != expectedNFTUID {
					return "", nil, sdkerrors.Wrapf(
						types.ErrNFTMappingConflict,
						"erc721 token %s is already bound to nft %s, not %s",
						tokenID, string(forward), expectedNFTUID,
					)
				}
			}
		}

		return EvmContractAddress, EvmTokenIds, nil

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
