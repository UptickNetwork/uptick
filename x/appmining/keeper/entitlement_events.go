package keeper

import (
	"encoding/hex"
	"strconv"

	"github.com/cometbft/cometbft/crypto/tmhash"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// currentTxHashHex returns sha256(tx_bytes) as lowercase hex, or empty when
// TxBytes are unavailable (e.g. unit tests without a wrapped tx).
func currentTxHashHex(ctx sdk.Context) string {
	bz := ctx.TxBytes()
	if len(bz) == 0 {
		return ""
	}
	return hex.EncodeToString(tmhash.Sum(bz))
}

// emitV1StateEvent emits a §6.1-aligned entitlement event. extra is appended
// after the stable schema attributes (e.g. challenge_id).
func emitV1StateEvent(
	ctx sdk.Context,
	eventType, action, poolID, entitlementID, appID, prevState, newState, amount, denom, digest string,
	extra ...sdk.Attribute,
) {
	attrs := []sdk.Attribute{
		sdk.NewAttribute(types.AttributeKeyChainID, ctx.ChainID()),
		sdk.NewAttribute(types.AttributeKeyPoolID, poolID),
		sdk.NewAttribute(types.AttributeKeyEntitlementID, entitlementID),
		sdk.NewAttribute(types.AttributeKeyAppID, appID),
		sdk.NewAttribute(types.AttributeKeyAction, action),
		sdk.NewAttribute(types.AttributeKeyPrevState, prevState),
		sdk.NewAttribute(types.AttributeKeyNewState, newState),
		sdk.NewAttribute(types.AttributeKeyAmount, amount),
		sdk.NewAttribute(types.AttributeKeyDenom, denom),
		sdk.NewAttribute(types.AttributeKeyEvidenceOrResolutionDigest, digest),
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
		sdk.NewAttribute(types.AttributeKeyTxHash, currentTxHashHex(ctx)),
	}
	attrs = append(attrs, extra...)
	ctx.EventManager().EmitEvent(sdk.NewEvent(eventType, attrs...))
}
