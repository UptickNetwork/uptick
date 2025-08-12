package keeper

import (
	"fmt"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	sdkerrors "cosmossdk.io/errors"
	erc721types "github.com/UptickNetwork/evm-nft-convert/types"
	"github.com/UptickNetwork/uptick/contracts"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RefundPacketToken handles the erc721 conversion for a native erc721 token
// pair:
//   - escrow tokens on module account
//   - mint nft to the receiver: nftId: tokenAddress|tokenID
func (k Keeper) RefundPacketToken(
	ctx sdk.Context,
	data ibcnfttransfertypes.NonFungibleTokenPacketData,
) error {

	erc721 := contracts.ERC721UpticksContract.ABI
	for _, tokenId := range data.TokenIds {

		uNftID := erc721types.CreateNFTUID(data.ClassId, tokenId)
		emvTokenId, evmContractAddress := erc721types.GetNFTFromUID(string(k.erc721keeper.GetTokenUIDPairByNFTUID(ctx, uNftID)))

		bigTokenId := new(big.Int)
		_, err := fmt.Sscan(emvTokenId, bigTokenId)
		if err != nil {
			sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s error scanning value", err)
			return err
		}

		evmReceiver := k.erc721keeper.GetEvmAddressByContractTokenId(ctx, evmContractAddress, tokenId)
		_, err = k.erc721keeper.CallEVM(
			ctx, erc721, erc721types.ModuleAddress, common.HexToAddress(evmContractAddress), true,
			"safeTransferFrom", erc721types.ModuleAddress, common.HexToAddress(string(evmReceiver)), bigTokenId)
		if err != nil {
			return err
		}
	}

	return nil
}
