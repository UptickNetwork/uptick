package keeper

import (
	"testing"

	sdkerrors "cosmossdk.io/errors"
	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
	evmibctypes "github.com/UptickNetwork/uptick/x/evmibc/types"
)

// mockICS721 records the order in which IBC-layer refunds run.
type mockICS721 struct {
	order *[]string
}

func (m *mockICS721) OnAcknowledgementPacket(sdk.Context, channeltypes.Packet, nfttransfertypes.NonFungibleTokenPacketData, channeltypes.Acknowledgement) error {
	*m.order = append(*m.order, "ibc:ack")
	return nil
}

func (m *mockICS721) OnTimeoutPacket(sdk.Context, channeltypes.Packet, nfttransfertypes.NonFungibleTokenPacketData) error {
	*m.order = append(*m.order, "ibc:timeout")
	return nil
}

func (m *mockICS721) GetVoucherClassID(_ sdk.Context, classID string) (string, error) {
	return classID, nil
}

// mockERC721 records the order in which the ERC721 refund (reverse conversion) runs.
// refundErr, when set, makes RefundPacketToken return it (order still recorded),
// so tests can exercise the refund-failure path.
type mockERC721 struct {
	order     *[]string
	refundErr error
}

func (m *mockERC721) RefundPacketToken(sdk.Context, nfttransfertypes.NonFungibleTokenPacketData) error {
	*m.order = append(*m.order, "erc:refund")
	return m.refundErr
}

type mockCW721 struct {
	order     *[]string
	refundErr error
}

func (m *mockCW721) RefundPacketToken(sdk.Context, nfttransfertypes.NonFungibleTokenPacketData) error {
	*m.order = append(*m.order, "cw:refund")
	return m.refundErr
}

// TestOnAcknowledgementPacket_ERC721ReleasesIBCBeforeRefund reproduces the
// refund deadlock: for a failed ERC721 outbound packet the IBC-escrowed NFT MUST be
// released to the module account BEFORE RefundPacketToken burns it and returns
// the ERC721. If RefundPacketToken ran first, the NFT would still be escrowed
// (owner != module), BurnNFT would fail, and the whole cache-context would roll
// back, permanently locking the ERC721 and the NFT.
func TestOnAcknowledgementPacket_ERC721ReleasesIBCBeforeRefund(t *testing.T) {
	order := &[]string{}
	k := NewKeeper(&mockICS721{order: order})
	k.SetErc721Keeper(&mockERC721{order: order})
	k.SetCw721Keeper(&mockCW721{order: order})

	packet := channeltypes.Packet{SourcePort: "nft-transfer", SourceChannel: "channel-0"}
	data := nfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
		Sender:   erc721types.AccModuleAddress.String(),
		Memo:     erc721types.TransferERC721Memo,
	}
	ack := channeltypes.NewErrorAcknowledgement(sdkerrors.Wrap(errortypes.ErrInvalidRequest, "boom"))

	err := k.OnAcknowledgementPacket(sdk.Context{}, packet, data, ack)
	require.NoError(t, err)
	require.Equal(t, []string{"ibc:ack", "erc:refund"}, *order)
	require.Equal(t, evmibctypes.ConvertKindERC721, evmibctypes.OutboundConvertKind(data))
}

// TestOnTimeoutPacket_ERC721ReleasesIBCBeforeRefund ensures the same ordering for
// the timeout path (mirrors the CW721 branch).
func TestOnTimeoutPacket_ERC721ReleasesIBCBeforeRefund(t *testing.T) {
	order := &[]string{}
	k := NewKeeper(&mockICS721{order: order})
	k.SetErc721Keeper(&mockERC721{order: order})
	k.SetCw721Keeper(&mockCW721{order: order})

	packet := channeltypes.Packet{SourcePort: "nft-transfer", SourceChannel: "channel-0"}
	data := nfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
		Sender:   erc721types.AccModuleAddress.String(),
		Memo:     erc721types.TransferERC721Memo,
	}

	err := k.OnTimeoutPacket(sdk.Context{}, packet, data)
	require.NoError(t, err)
	require.Equal(t, []string{"ibc:timeout", "erc:refund"}, *order)
}
