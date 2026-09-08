package keeper

import (
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// GetClassIDAndNFTID resolves the canonical Cosmos class id and NFT ids for an
// ERC721 → Cosmos conversion. The stored UID mapping is authoritative: when a
// token is already bound, its saved class/NFT id wins over caller input, and
// caller input that contradicts it is rejected.
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

// GetContractAddressAndTokenIds resolves the canonical ERC721 contract address
// and token ids for a Cosmos → ERC721 conversion, using saved mappings or
// generating new ids as needed. An unregistered class deploys a fresh module
// contract; a registered class is pinned to its canonical contract, and a
// caller-supplied address that differs is rejected.
func (k Keeper) GetContractAddressAndTokenIds(ctx sdk.Context, msg *types.MsgConvertNFT) (string, []string, error) {

	var (
		EvmContractAddress string
		EvmTokenIds        []string
		err                error
	)

	pair, err := k.GetPairByClass(ctx, msg.ClassId)
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

	}

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

// getNftDatas resolves a batch of ids with getNftData, element by element.
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

// getNftData resolves one identifier of the given nftType
// (0: nftId, 1: classId, 2: tokenId, 3: contract address).
// Caller input wins over saved mappings only when they agree; a contradiction
// is an error, and neither being present derives the value from nftPairOrg.
func getNftData(nftOrg string, nftPairOrg string, nftSaved string, nftType int) (string, error) {

	var nftRet string
	switch {
	case nftOrg == "" && nftSaved == "":
		nftRet = createNftDataByType(nftPairOrg, nftType)
	case nftOrg == "":
		nftRet = nftSaved
	case nftSaved == "", nftSaved == nftOrg:
		nftRet = nftOrg
	default:
		return "", getNftDataErrorByType(nftSaved, nftOrg, nftType)
	}

	return nftRet, nil

}

// createNftDataByType derives an id of the given nftType
// (0: nftId, 1: classId, 2: tokenId, 3: contract address) from its counterpart.
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

// getNftDataErrorByType builds the mismatch error for the given nftType
// (0: nftId, 1: classId, 2: tokenId, 3: contract address).
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
