package keeper

import (
	"context"
	"errors"
	"strings"
	"testing"

	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
)

// captureICS721 records the ClassId handed to the IBC nft-transfer refund.
// It must receive the original full-path ClassId so the fork's
// IsAwayFromOrigin mint/unescrow decision is preserved.
type captureICS721 struct {
	classIDs *[]string
	// localClasses names the class ids that exist on chain. Mirrors the
	// ICS-721 keeper's HasClass-first rule: an id that exists locally is
	// returned unchanged, any other id containing "/" resolves to ibc/<hash>.
	localClasses map[string]bool
}

func (m *captureICS721) OnAcknowledgementPacket(_ sdk.Context, _ channeltypes.Packet, data nfttransfertypes.NonFungibleTokenPacketData, _ channeltypes.Acknowledgement) error {
	*m.classIDs = append(*m.classIDs, data.ClassId)
	return nil
}

func (m *captureICS721) OnTimeoutPacket(_ sdk.Context, _ channeltypes.Packet, data nfttransfertypes.NonFungibleTokenPacketData) error {
	*m.classIDs = append(*m.classIDs, data.ClassId)
	return nil
}

func (m *captureICS721) GetVoucherClassID(_ sdk.Context, classID string) (string, error) {
	if !strings.Contains(classID, "/") || m.localClasses[classID] {
		return classID, nil
	}
	return nfttransfertypes.ParseClassTrace(classID).IBCClassID(), nil
}

// captureERC721 records the ClassId handed to the ERC721 reverse-conversion
// refund. It must receive the local voucher id (ibc/<hash> or bare) so the
// pair mapping lookup succeeds.
type captureERC721 struct {
	classIDs *[]string
}

func (m *captureERC721) ConvertNFT(context.Context, *erc721types.MsgConvertNFT) (*erc721types.MsgConvertNFTResponse, error) {
	return &erc721types.MsgConvertNFTResponse{}, nil
}

func (m *captureERC721) RefundPacketToken(_ sdk.Context, data nfttransfertypes.NonFungibleTokenPacketData) error {
	*m.classIDs = append(*m.classIDs, data.ClassId)
	return nil
}

// captureCW721 mirrors captureERC721 for the CW721 reverse-conversion refund.
type captureCW721 struct {
	classIDs *[]string
}

func (m *captureCW721) ConvertNFT(context.Context, *cw721types.MsgConvertNFT) (*cw721types.MsgConvertNFTResponse, error) {
	return &cw721types.MsgConvertNFTResponse{}, nil
}

func (m *captureCW721) RefundPacketToken(_ sdk.Context, data nfttransfertypes.NonFungibleTokenPacketData) error {
	*m.classIDs = append(*m.classIDs, data.ClassId)
	return nil
}

// TestRefundClassIDSplit guards S-1/S-2: the IBC nft-transfer refund must
// receive the original full-path ClassId while the erc721/cw721 reverse
// conversion must receive the local voucher id. Feeding a single id to both
// consumers flips the fork's IsAwayFromOrigin mint/unescrow decision (S-1:
// refund of a burned voucher tries unescrow → ErrNFTNotExists → liveness
// deadlock) or breaks the module pair lookup (S-2: silent non-refund).
func TestRefundClassIDSplit(t *testing.T) {
	// shape-2: voucher prefix matches this packet's (port, channel).
	// shape-3: voucher prefix does not match (multi-hop).
	// local:   a class id containing "/" that exists locally on chain -- a
	//          natively issued `sub/collection`, not a trace path.
	cases := []struct {
		name         string
		fullPath     string
		classIsLocal bool
	}{
		{"shape2 matching prefix", nfttransfertypes.PortID + "/channel-0/kitty", false},
		{"shape3 non-matching channel", nfttransfertypes.PortID + "/channel-1/kitty", false},
		{"local class containing a slash", "sub/collection", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The module-side refund must get the local class id: the id itself
			// when such a class exists locally, ibc/<hash> otherwise.
			localID := tc.fullPath
			if !tc.classIsLocal {
				localID = nfttransfertypes.ParseClassTrace(tc.fullPath).IBCClassID()
			}
			localClasses := map[string]bool{}
			if tc.classIsLocal {
				localClasses[tc.fullPath] = true
			}
			packet := channeltypes.Packet{SourcePort: nfttransfertypes.PortID, SourceChannel: "channel-0"}

			t.Run("timeout erc721", func(t *testing.T) {
				ibcIDs, ercIDs := &[]string{}, &[]string{}
				k := NewKeeper(&captureICS721{classIDs: ibcIDs, localClasses: localClasses})
				k.SetErc721Keeper(&captureERC721{classIDs: ercIDs})
				k.SetCw721Keeper(&captureCW721{classIDs: &[]string{}})

				data := nfttransfertypes.NonFungibleTokenPacketData{
					ClassId:  tc.fullPath,
					TokenIds: []string{"nft1"},
					Sender:   erc721types.AccModuleAddress.String(),
					Memo:     erc721types.TransferERC721Memo,
				}
				require.NoError(t, k.OnTimeoutPacket(newRefundCtx(t), packet, data))
				require.Equal(t, []string{tc.fullPath}, *ibcIDs, "IBC refund must receive the full path")
				require.Equal(t, []string{localID}, *ercIDs, "erc721 refund must receive the local id")
			})

			t.Run("timeout cw721", func(t *testing.T) {
				ibcIDs, cwIDs := &[]string{}, &[]string{}
				k := NewKeeper(&captureICS721{classIDs: ibcIDs, localClasses: localClasses})
				k.SetErc721Keeper(&captureERC721{classIDs: &[]string{}})
				k.SetCw721Keeper(&captureCW721{classIDs: cwIDs})

				data := nfttransfertypes.NonFungibleTokenPacketData{
					ClassId:  tc.fullPath,
					TokenIds: []string{"nft1"},
					Sender:   cw721types.AccModuleAddress.String(),
					Memo:     cw721types.TransferCW721Memo,
				}
				require.NoError(t, k.OnTimeoutPacket(newRefundCtx(t), packet, data))
				require.Equal(t, []string{tc.fullPath}, *ibcIDs)
				require.Equal(t, []string{localID}, *cwIDs)
			})

			t.Run("ack erc721", func(t *testing.T) {
				ibcIDs, ercIDs := &[]string{}, &[]string{}
				k := NewKeeper(&captureICS721{classIDs: ibcIDs, localClasses: localClasses})
				k.SetErc721Keeper(&captureERC721{classIDs: ercIDs})
				k.SetCw721Keeper(&captureCW721{classIDs: &[]string{}})

				data := nfttransfertypes.NonFungibleTokenPacketData{
					ClassId:  tc.fullPath,
					TokenIds: []string{"nft1"},
					Sender:   erc721types.AccModuleAddress.String(),
					Memo:     erc721types.TransferERC721Memo,
				}
				ack := channeltypes.NewErrorAcknowledgement(errors.New("boom"))
				require.NoError(t, k.OnAcknowledgementPacket(newRefundCtx(t), packet, data, ack))
				require.Equal(t, []string{tc.fullPath}, *ibcIDs)
				require.Equal(t, []string{localID}, *ercIDs)
			})

			t.Run("ack cw721", func(t *testing.T) {
				ibcIDs, cwIDs := &[]string{}, &[]string{}
				k := NewKeeper(&captureICS721{classIDs: ibcIDs, localClasses: localClasses})
				k.SetErc721Keeper(&captureERC721{classIDs: &[]string{}})
				k.SetCw721Keeper(&captureCW721{classIDs: cwIDs})

				data := nfttransfertypes.NonFungibleTokenPacketData{
					ClassId:  tc.fullPath,
					TokenIds: []string{"nft1"},
					Sender:   cw721types.AccModuleAddress.String(),
					Memo:     cw721types.TransferCW721Memo,
				}
				ack := channeltypes.NewErrorAcknowledgement(errors.New("boom"))
				require.NoError(t, k.OnAcknowledgementPacket(newRefundCtx(t), packet, data, ack))
				require.Equal(t, []string{tc.fullPath}, *ibcIDs)
				require.Equal(t, []string{localID}, *cwIDs)
			})
		})
	}
}
