package keeper

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// Bech32 addresses decode identically from their all-lowercase and
// all-uppercase spellings; the keeper canonicalizes the address, so a second
// registration using the uppercase alias of an already-registered contract
// is rejected.
func TestAudit_CW721CaseAliasCannotRegisterTwice(t *testing.T) {
	k, ctx, owner, _, _ := setupConvertKeeper(t)

	// A fresh contract not pre-registered by the test setup.
	lower := sdk.AccAddress(bytes20(0x44)).String()
	upper := strings.ToUpper(lower)

	// Sanity: the two spellings decode to the exact same account bytes.
	lowerBytes, err := sdk.AccAddressFromBech32(lower)
	require.NoError(t, err)
	upperBytes, err := sdk.AccAddressFromBech32(upper)
	require.NoError(t, err)
	require.Equal(t, lowerBytes, upperBytes)

	// First registration using the uppercase alias must succeed and store the
	// canonical (lowercase) address.
	pair, err := k.RegisterCW721(ctx, &types.MsgConvertCW721{
		ContractAddress: upper,
		TokenIds:        []string{"1"},
		ClassId:         "",
		Sender:          owner.String(),
		Receiver:        owner.String(),
	})
	require.NoError(t, err)
	require.Equal(t, lower, pair.Cw721Address, "stored pair must use the canonical lowercase address")

	// The lowercase spelling must now be considered registered...
	require.True(t, k.IsCW721Registered(ctx, lower))
	// ...and a second registration under either spelling must be rejected.
	for _, alias := range []string{lower, upper} {
		_, err := k.RegisterCW721(ctx, &types.MsgConvertCW721{
			ContractAddress: alias,
			TokenIds:        []string{"2"},
			ClassId:         "",
			Sender:          owner.String(),
			Receiver:        owner.String(),
		})
		require.ErrorIs(t, err, types.ErrTokenPairAlreadyExists,
			"alias %q must not bypass duplicate detection", alias)
	}

	// Exactly one pair for this contract exists and exported genesis validates.
	var mine []types.TokenPair
	for _, p := range k.GetTokenPairs(ctx) {
		if p.Cw721Address == lower {
			mine = append(mine, p)
		}
	}
	require.Len(t, mine, 1)
	gs := types.GenesisState{
		Params:     types.DefaultParams(),
		TokenPairs: mine,
	}
	require.NoError(t, gs.Validate())
}

// The canonical lookup path must resolve a differently-cased query.
func TestAudit_CW721CaseAliasResolvesSamePair(t *testing.T) {
	k, ctx, owner, _, _ := setupConvertKeeper(t)

	lower := sdk.AccAddress(bytes20(0x55)).String()

	pair, err := k.RegisterCW721(ctx, &types.MsgConvertCW721{
		ContractAddress: lower,
		TokenIds:        []string{"1"},
		ClassId:         "",
		Sender:          owner.String(),
		Receiver:        owner.String(),
	})
	require.NoError(t, err)

	got, err := k.GetPairByCW721(ctx, strings.ToUpper(lower))
	require.NoError(t, err)
	require.Equal(t, pair.GetID(), got.GetID())
}
