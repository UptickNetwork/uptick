package keeper

import (
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RefundPacketToken reverses an outbound ERC721 convert by delegating to
// x/erc721, which returns the ERC721 and deletes pair/refund mappings.
func (k Keeper) RefundPacketToken(
	ctx sdk.Context,
	data ibcnfttransfertypes.NonFungibleTokenPacketData,
) error {
	return k.erc721keeper.RefundPacketToken(ctx, data)
}
