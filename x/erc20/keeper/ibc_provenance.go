package keeper

import (
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"github.com/UptickNetwork/uptick/x/erc20/types"
)

func ibcTransferProvenanceKey(
	sourcePort, sourceChannel string,
	sequence uint64,
	sender, denom, amount string,
) []byte {
	return types.IBCTransferProvenanceKey(sourcePort, sourceChannel, sequence, sender, denom, amount)
}

func ibcTransferProvenanceKeyFromPacket(
	packet channeltypes.Packet,
	data transfertypes.FungibleTokenPacketData,
) []byte {
	return ibcTransferProvenanceKey(
		packet.GetSourcePort(),
		packet.GetSourceChannel(),
		packet.GetSequence(),
		data.Sender,
		data.Denom,
		data.Amount,
	)
}

// SetIBCTransferProvenance records that an outbound packet was created by
// MsgTransferERC20 and is eligible for ERC20-side error-ack refund handling.
func (k Keeper) SetIBCTransferProvenance(
	ctx sdk.Context,
	sourcePort, sourceChannel string,
	sequence uint64,
	sender, denom, amount string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixIBCTransferProvenance)
	store.Set(ibcTransferProvenanceKey(sourcePort, sourceChannel, sequence, sender, denom, amount), []byte{1})
}

// HasIBCTransferProvenance reports whether provenance exists for the packet.
func (k Keeper) HasIBCTransferProvenance(
	ctx sdk.Context,
	packet channeltypes.Packet,
	data transfertypes.FungibleTokenPacketData,
) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixIBCTransferProvenance)
	return store.Has(ibcTransferProvenanceKeyFromPacket(packet, data))
}

// DeleteIBCTransferProvenance removes provenance after ack handling to prevent replay.
func (k Keeper) DeleteIBCTransferProvenance(
	ctx sdk.Context,
	packet channeltypes.Packet,
	data transfertypes.FungibleTokenPacketData,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixIBCTransferProvenance)
	store.Delete(ibcTransferProvenanceKeyFromPacket(packet, data))
}

// ConsumeIBCTransferProvenance deletes provenance and returns whether it existed.
func (k Keeper) ConsumeIBCTransferProvenance(
	ctx sdk.Context,
	packet channeltypes.Packet,
	data transfertypes.FungibleTokenPacketData,
) bool {
	if !k.HasIBCTransferProvenance(ctx, packet, data) {
		return false
	}
	k.DeleteIBCTransferProvenance(ctx, packet, data)
	return true
}
