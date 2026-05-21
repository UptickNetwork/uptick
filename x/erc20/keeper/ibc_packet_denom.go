package keeper

import (
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"

	"github.com/UptickNetwork/uptick/x/erc20/types"
)

// transferPacketDenom returns the denomination placed in ICS-20 packet data by
// the ibc-go transfer keeper (see sendTransfer). Bank balances use ibc/HASH
// vouchers while packet data carries the full trace path.
func (k Keeper) transferPacketDenom(ctx sdk.Context, bankDenom string) (string, error) {
	if !strings.HasPrefix(bankDenom, transfertypes.DenomPrefix+"/") {
		return bankDenom, nil
	}
	return k.ibcKeeper.DenomPathFromHash(ctx, bankDenom)
}

// tokenPairFromPacketDenom resolves a registered token pair from ICS-20 packet data.
func (k Keeper) tokenPairFromPacketDenom(ctx sdk.Context, packetDenom string) (types.TokenPair, error) {
	trace := transfertypes.ParseDenomTrace(packetDenom)
	if id := k.GetDenomMap(ctx, trace.IBCDenom()); len(id) > 0 {
		if pair, ok := k.GetTokenPair(ctx, id); ok {
			return pair, nil
		}
	}
	if strings.HasPrefix(packetDenom, transfertypes.DenomPrefix+"/") {
		if id := k.GetDenomMap(ctx, packetDenom); len(id) > 0 {
			if pair, ok := k.GetTokenPair(ctx, id); ok {
				return pair, nil
			}
		}
	}
	return types.TokenPair{}, sdkerrors.Wrapf(types.ErrInternalTokenPair, "pair is not registered for packet denom %s", packetDenom)
}
