package keeper

import (
	"testing"

	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// T1 (R1-C A1): RegisterNFT must refuse to bind an IBC voucher class to an
// ERC721 contract, while still accepting module-native ("uptick-…") and
// externally-registered class ids — the gate must close the two R1 sources
// (convert memo + manual MsgConvertNFT) without over-rejecting legal classes.
func TestRegisterNFT_RejectsIBCVoucherClass(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)

	_, err := k.RegisterNFT(ctx, &types.MsgConvertNFT{
		ClassId:            "ibc/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		EvmContractAddress: "0x2222222222222222222222222222222222222222",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, errortypes.ErrInvalidRequest,
		"an ibc/ voucher class must be rejected at the registration gate")
	require.Contains(t, err.Error(), "R1-C")

	// The rejected class must not have written any pair or map entry.
	require.False(t, k.IsClassRegistered(ctx, "ibc/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"))
}

func TestRegisterNFT_AcceptsUptickClass(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)

	class := "uptick-0x2222222222222222222222222222222222222222"
	contract := "0x2222222222222222222222222222222222222222"
	pair, err := k.RegisterNFT(ctx, &types.MsgConvertNFT{
		ClassId:            class,
		EvmContractAddress: contract,
	})
	require.NoError(t, err)
	require.Equal(t, class, pair.ClassId)
	require.True(t, k.IsClassRegistered(ctx, class))
}

func TestRegisterNFT_AcceptsExternalHexClass(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)

	class := "0x3333333333333333333333333333333333333333"
	contract := "0x3333333333333333333333333333333333333333"
	pair, err := k.RegisterNFT(ctx, &types.MsgConvertNFT{
		ClassId:            class,
		EvmContractAddress: contract,
	})
	require.NoError(t, err)
	require.Equal(t, class, pair.ClassId)
	require.True(t, k.IsClassRegistered(ctx, class))
}
