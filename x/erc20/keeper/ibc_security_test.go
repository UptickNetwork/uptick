package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	erc20types "github.com/UptickNetwork/uptick/x/erc20/types"
)

// TestCraftedMemoTriggersNoMintWithoutProvenance ensures a crafted ICS-20 memo
// cannot authorize ERC20 refund minting without MsgTransferERC20 provenance.
func TestCraftedMemoTriggersNoMintWithoutProvenance(t *testing.T) {
	k, ctx := newProvenanceKeeper(t)

	packet := channeltypes.Packet{
		Sequence:      9,
		SourcePort:    "transfer",
		SourceChannel: "channel-0",
	}
	data := transfertypes.FungibleTokenPacketData{
		Denom:    "ibc/ABC",
		Amount:   math.NewInt(100).String(),
		Sender:   "uptick1sender",
		Receiver: "cosmos1receiver",
		Memo:     "exploit" + erc20types.TransferERC20Memo,
	}

	ack := channeltypes.Acknowledgement{Response: &channeltypes.Acknowledgement_Error{Error: "failed"}}
	err := k.OnAcknowledgementPacket(ctx, packet, data, ack)
	require.NoError(t, err)
	require.False(t, k.HasIBCTransferProvenance(ctx, packet, data))
}

// TestErrorAckWithoutMarkerMemoNoProvenance covers ordinary ICS20 error acks without provenance.
func TestErrorAckWithoutMarkerMemoNoProvenance(t *testing.T) {
	k, ctx := newProvenanceKeeper(t)

	packet := channeltypes.Packet{Sequence: 10, SourcePort: "transfer", SourceChannel: "channel-0"}
	data := transfertypes.FungibleTokenPacketData{
		Denom: "ibc/ABC", Amount: "100", Sender: "uptick1sender", Memo: "",
	}

	ack := channeltypes.Acknowledgement{Response: &channeltypes.Acknowledgement_Error{Error: "failed"}}
	err := k.OnAcknowledgementPacket(ctx, packet, data, ack)
	require.NoError(t, err)
}

// TestErrorAckProvenanceConsumedOnRefund verifies provenance is single-use on error ack.
func TestErrorAckProvenanceConsumedOnRefund(t *testing.T) {
	k, ctx := newProvenanceKeeper(t)

	packet := channeltypes.Packet{Sequence: 11, SourcePort: "transfer", SourceChannel: "channel-0"}
	data := transfertypes.FungibleTokenPacketData{
		Denom:  "ibc/ABC",
		Amount: "100",
		Sender: "uptick1sender",
		Memo:   erc20types.TransferERC20Memo,
	}

	k.SetIBCTransferProvenance(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence, data.Sender, data.Denom, data.Amount)
	require.True(t, k.HasIBCTransferProvenance(ctx, packet, data))

	ack := channeltypes.Acknowledgement{Response: &channeltypes.Acknowledgement_Error{Error: "failed"}}
	_ = k.OnAcknowledgementPacket(ctx, packet, data, ack)
	require.False(t, k.HasIBCTransferProvenance(ctx, packet, data))
}

// TestProvenanceKeyUsesPacketDenomNotBankHash ensures ack matching uses the ICS-20
// packet denomination (trace path), not the bank ibc/HASH voucher denomination.
func TestProvenanceKeyUsesPacketDenomNotBankHash(t *testing.T) {
	k, ctx := newProvenanceKeeper(t)

	packet := channeltypes.Packet{
		Sequence:      13,
		SourcePort:    "transfer",
		SourceChannel: "channel-3",
	}
	packetData := transfertypes.FungibleTokenPacketData{
		Denom:  "transfer/channel-3/uptick/uptick",
		Amount: "100",
		Sender: "uptick1kwla7qu828jn4ljn3s2uz3823km0gctktmulat",
	}
	bankDenom := "ibc/4EB1D3A1CC4CC7A949A6CEB0D1E97FD37C71EE8754FA9AB6FC3E0F0B6DA74926"

	k.SetIBCTransferProvenance(
		ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence,
		packetData.Sender, bankDenom, packetData.Amount,
	)
	require.False(t, k.HasIBCTransferProvenance(ctx, packet, packetData))

	k.SetIBCTransferProvenance(
		ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence,
		packetData.Sender, packetData.Denom, packetData.Amount,
	)
	require.True(t, k.HasIBCTransferProvenance(ctx, packet, packetData))
}

// TestSuccessAckClearsProvenance ensures successful acknowledgements delete pending provenance.
func TestSuccessAckClearsProvenance(t *testing.T) {
	k, ctx := newProvenanceKeeper(t)

	packet := channeltypes.Packet{Sequence: 12, SourcePort: "transfer", SourceChannel: "channel-0"}
	data := transfertypes.FungibleTokenPacketData{
		Denom: "ibc/ABC", Amount: "100", Sender: "uptick1sender",
	}

	k.SetIBCTransferProvenance(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence, data.Sender, data.Denom, data.Amount)
	ack := channeltypes.Acknowledgement{Response: &channeltypes.Acknowledgement_Result{Result: []byte{1}}}

	err := k.OnAcknowledgementPacket(ctx, packet, data, ack)
	require.NoError(t, err)
	require.False(t, k.HasIBCTransferProvenance(ctx, packet, data))
}
