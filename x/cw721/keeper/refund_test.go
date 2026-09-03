package keeper

import (
	"testing"

	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
)

func TestRefundPacketToken_MissingPair(t *testing.T) {
	k, ctx := setupKeeper(t)
	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.ErrorIs(t, err, cw721types.ErrTokenPairNotFound)
}

func TestRefundPacketToken_MissingReceiver(t *testing.T) {
	k, ctx := setupKeeper(t)
	sdk.GetConfig().SetBech32PrefixForAccount("uptick", "uptickpub")
	contract := sdk.AccAddress(make([]byte, 20)).String()

	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.ErrorIs(t, err, errortypes.ErrInvalidAddress)
}
