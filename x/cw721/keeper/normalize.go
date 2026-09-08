package keeper

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// NormalizeCW721Address returns the canonical bech32 form of a CW721 contract
// address. Bech32 decodes identically from its all-lowercase and all-uppercase
// spellings, so the same contract can be presented under multiple alias
// strings. Every KV key derived from a CW721 address (token pair maps, token
// UIDs, refund receivers) MUST use the canonical re-encoded form, otherwise:
//   - the same contract can be registered twice under case aliases;
//   - lookups with a differently-cased address miss the stored pair;
//   - exported genesis fails validation, because genesis duplicate detection
//     is case-insensitive (strings.ToLower) while runtime keys are not.
func NormalizeCW721Address(addr string) (string, error) {
	a, err := sdk.AccAddressFromBech32(addr)
	if err != nil {
		return "", sdkerrors.Wrapf(
			types.ErrInternalTokenPair, "invalid CW721 contract address: %s", addr,
		)
	}
	return a.String(), nil
}

// canonicalCW721Key returns the canonical KV key for a CW721 address: the
// re-encoded lowercase bech32 form when the input is a valid bech32 address,
// and the raw string otherwise (hex/legacy inputs are passed through). Keeper
// map functions wrap every CW721-address key with this so a differently-cased
// alias can never miss — or bypass — the stored entry.
func canonicalCW721Key(addr string) string {
	if a, err := sdk.AccAddressFromBech32(addr); err == nil {
		return a.String()
	}
	return addr
}
