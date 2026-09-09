package types

import (
	"strings"
	"testing"

	ibctypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/stretchr/testify/require"
)

// Regression: x/erc721 validated its Cosmos token ids with the collection
// token-id rules while x/cw721 only checked them for emptiness, so ids the
// keeper would reject (too short/long, containing NUL or "/") reached the state
// machine. Both modules must apply the same rules.
const validationTestAddr = "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"

func nftIDCases() []struct {
	name    string
	nftID   string
	wantErr bool
} {
	return []struct {
		name    string
		nftID   string
		wantErr bool
	}{
		{name: "valid", nftID: "nft-1", wantErr: false},
		{name: "exactly min length", nftID: "abc", wantErr: false},
		{name: "exactly max length", nftID: strings.Repeat("a", 128), wantErr: false},
		{name: "too short", nftID: "ab", wantErr: true},
		{name: "too long", nftID: strings.Repeat("a", 129), wantErr: true},
		{name: "contains slash", nftID: "nft/1", wantErr: true},
		{name: "contains NUL", nftID: "nft\x001", wantErr: true},
		{name: "empty", nftID: "", wantErr: true},
	}
}

func TestMsgConvertNFT_ValidateBasic_NftIDRules(t *testing.T) {
	t.Parallel()

	for _, tc := range nftIDCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			msg := MsgConvertNFT{
				Sender:   validationTestAddr,
				Receiver: validationTestAddr,
				ClassId:  "class-1",
				NftIds:   []string{tc.nftID},
				TokenIds: []string{"1"},
			}
			err := msg.ValidateBasic()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgConvertCW721_ValidateBasic_NftIDRules(t *testing.T) {
	t.Parallel()

	for _, tc := range nftIDCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			msg := MsgConvertCW721{
				Sender:          validationTestAddr,
				Receiver:        validationTestAddr,
				ContractAddress: validationTestAddr,
				ClassId:         "class-1",
				TokenIds:        []string{"1"},
				NftIds:          []string{tc.nftID},
			}
			err := msg.ValidateBasic()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// NftIds is optional on MsgConvertCW721 (an empty list means "derive from the
// token id"), so an absent list must stay valid.
func TestMsgConvertCW721_ValidateBasic_AllowsOmittedNftIDs(t *testing.T) {
	t.Parallel()

	msg := MsgConvertCW721{
		Sender:          validationTestAddr,
		Receiver:        validationTestAddr,
		ContractAddress: validationTestAddr,
		ClassId:         "class-1",
		TokenIds:        []string{"1"},
	}
	require.NoError(t, msg.ValidateBasic())
}

func TestMsgTransferCW721_ValidateBasic_CosmosTokenIDRules(t *testing.T) {
	t.Parallel()

	for _, tc := range nftIDCases() {
		// An empty cosmos token id list is legal on this message: the ids are
		// derived at execution time.
		if tc.nftID == "" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			msg := MsgTransferCW721{
				CwSender:          validationTestAddr,
				CosmosReceiver:    validationTestAddr,
				CwContractAddress: validationTestAddr,
				ClassId:           "class-1",
				CwTokenIds:        []string{"1"},
				CosmosTokenIds:    []string{tc.nftID},
				SourcePort:        "transfer",
				SourceChannel:     "channel-0",
				TimeoutHeight:     ibctypes.Height{RevisionNumber: 1, RevisionHeight: 100},
			}
			err := msg.ValidateBasic()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
