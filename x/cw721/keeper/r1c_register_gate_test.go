package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// T1 (R1-C A1): the cw721 registration gate is forward-only protection —
// cw721 has no legacy ibc-class pairs, but a future ibc/ class must still be
// refused so the R1 lockout cannot be reintroduced through the cw721 surface.
func TestRegisterNFT_RejectsIBCVoucherClass(t *testing.T) {
	k, ctx, _, _, _ := setupConvertKeeper(t)

	_, err := k.RegisterNFT(ctx, &types.MsgConvertNFT{
		ClassId:         "ibc/BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
		ContractAddress: sdk.AccAddress(bytes20(0x44)).String(),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, errortypes.ErrInvalidRequest)
	require.Contains(t, err.Error(), "R1-C")
	require.False(t, k.IsClassRegistered(ctx, "ibc/BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"))
}

// The gate must not over-reject: a fresh non-ibc class still registers.
func TestRegisterNFT_AcceptsNativeClass(t *testing.T) {
	k, ctx, _, _, _ := setupConvertKeeper(t)

	contract := sdk.AccAddress(bytes20(0x55)).String()
	pair, err := k.RegisterNFT(ctx, &types.MsgConvertNFT{
		ClassId:         "uptick-0x55",
		ContractAddress: contract,
	})
	require.NoError(t, err)
	require.Equal(t, "uptick-0x55", pair.ClassId)
	require.True(t, k.IsClassRegistered(ctx, "uptick-0x55"))
}
