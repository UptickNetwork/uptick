package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

func TestMsgConvertNFT_ValidateBasic_ERC721(t *testing.T) {
	validSender := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"

	tests := []struct {
		name            string
		sender          string
		evmReceiver     string
		evmContractAddr string
		classID         string
		nftIDs          []string
		wantErr         bool
	}{
		{
			name:        "valid complete",
			sender:      validSender,
			evmReceiver: "0x1234567890123456789012345678901234567890",
			classID:     "class-1",
			nftIDs:      []string{"nft-1", "nft-2"},
			wantErr:     false,
		},
		{
			name:        "empty sender",
			sender:      "",
			evmReceiver: "0x1234567890123456789012345678901234567890",
			classID:     "class-1",
			nftIDs:      []string{"nft-1"},
			wantErr:     true,
		},
		{
			name:        "invalid sender",
			sender:      "invalid",
			evmReceiver: "0x1234567890123456789012345678901234567890",
			classID:     "class-1",
			nftIDs:      []string{"nft-1"},
			wantErr:     true,
		},
		{
			name:        "invalid receiver hex",
			sender:      validSender,
			evmReceiver: "not-a-hex-address",
			classID:     "class-1",
			nftIDs:      []string{"nft-1"},
			wantErr:     true,
		},
		{
			name:        "empty receiver",
			sender:      validSender,
			evmReceiver: "",
			classID:     "class-1",
			nftIDs:      []string{"nft-1"},
			wantErr:     true,
		},
		{
			name:            "invalid contract hex address",
			sender:          validSender,
			evmReceiver:     "0x1234567890123456789012345678901234567890",
			evmContractAddr: "0x123",
			classID:         "class-1",
			nftIDs:          []string{"nft-1"},
			wantErr:         true,
		},
		{
			name:        "short nft id",
			sender:      validSender,
			evmReceiver: "0x1234567890123456789012345678901234567890",
			classID:     "class-1",
			nftIDs:      []string{"n1"},
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := MsgConvertNFT{
				CosmosSender:       tc.sender,
				EvmReceiver:        tc.evmReceiver,
				EvmContractAddress: tc.evmContractAddr,
				ClassId:            tc.classID,
				CosmosTokenIds:     tc.nftIDs,
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

func TestMsgConvertNFT_RouteAndType_ERC721(t *testing.T) {
	msg := MsgConvertNFT{}
	require.Equal(t, RouterKey, msg.Route())
	require.Equal(t, TypeMsgConvertNFT, msg.Type())
}

func TestMsgConvertNFT_GetSigners_ERC721(t *testing.T) {
	sender := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"
	msg := MsgConvertNFT{CosmosSender: sender}
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, sender, signers[0].String())
}

func TestMsgConvertNFT_GetSignBytes_ERC721(t *testing.T) {
	msg := &MsgConvertNFT{
		CosmosSender:   "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
		EvmReceiver:    "0x1234567890123456789012345678901234567890",
		ClassId:        "class-1",
		CosmosTokenIds: []string{"nft-1"},
	}
	bz := msg.GetSignBytes()
	require.NotEmpty(t, bz)
}

func TestMsgConvertERC721_ValidateBasic_ERC721(t *testing.T) {
	validSender := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"

	tests := []struct {
		name         string
		sender       string
		receiver     string
		contractAddr string
		classID      string
		tokenIDs     []string
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name:         "valid",
			sender:       validSender,
			receiver:     validSender,
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "class-1",
			tokenIDs:     []string{"1"},
			wantErr:      false,
		},
		{
			name:         "invalid sender",
			sender:       "bad",
			receiver:     validSender,
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "class-1",
			tokenIDs:     []string{"1"},
			wantErr:      true,
		},
		{
			name:         "invalid receiver",
			sender:       validSender,
			receiver:     "bad",
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "class-1",
			tokenIDs:     []string{"1"},
			wantErr:      true,
		},
		{
			name:         "invalid contract",
			sender:       validSender,
			receiver:     validSender,
			contractAddr: "not-a-contract",
			classID:      "class-1",
			tokenIDs:     []string{"1"},
			wantErr:      true,
		},
		{
			// N-2 sentinel (round 18): the dfe8cee tightening added
			// strings.TrimSpace(msg.ClassId) == "" to MsgConvertERC721; an
			// empty class id must be rejected here. Without this case the
			// defence could silently regress back to "valid with empty id".
			name:         "empty class id rejected",
			sender:       validSender,
			receiver:     validSender,
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "",
			tokenIDs:     []string{"1"},
			wantErr:      true,
			wantErrMsg:   "class id cannot be empty",
		},
		{
			// N-2 sentinel: a whitespace-only class id is the same class of
			// bug as empty (TrimSpace rejects both).
			name:         "whitespace-only class id rejected",
			sender:       validSender,
			receiver:     validSender,
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "   ",
			tokenIDs:     []string{"1"},
			wantErr:      true,
			wantErrMsg:   "class id cannot be empty",
		},
		{
			// N-2 reverse sentinel: a non-empty class id must continue to
			// validate. Locks in that the new guard does not regress
			// legitimate ids (e.g. an "ibc/<hash>" voucher id).
			name:         "ibc voucher class id accepted",
			sender:       validSender,
			receiver:     validSender,
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "ibc/7F1D4F9D63B1E4D9E5A2D6B3E5C5A8B7C1D2E3F4A5B6C7D8E9F0A1B2C3D4E5F60",
			tokenIDs:     []string{"1"},
			wantErr:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := MsgConvertERC721{
				CosmosSender:       tc.sender,
				CosmosReceiver:     tc.receiver,
				EvmContractAddress: tc.contractAddr,
				ClassId:            tc.classID,
				EvmTokenIds:        tc.tokenIDs,
			}
			err := msg.ValidateBasic()
			if tc.wantErr {
				require.Error(t, err)
				if tc.wantErrMsg != "" {
					require.Contains(t, err.Error(), tc.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgConvertERC721_RouteAndType_ERC721(t *testing.T) {
	msg := MsgConvertERC721{}
	require.Equal(t, RouterKey, msg.Route())
	require.Equal(t, TypeMsgConvertERC721, msg.Type())
}

func TestMsgConvertERC721_GetSigners_ERC721(t *testing.T) {
	sender := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"
	msg := MsgConvertERC721{CosmosSender: sender}
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, sender, signers[0].String())
}

func TestMsgTransferERC721_ValidateBasic_ERC721(t *testing.T) {
	validSender := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"
	timeout := clienttypes.NewHeight(1, 100)

	tests := []struct {
		name         string
		sender       string
		receiver     string
		contractAddr string
		classID      string
		tokenIDs     []string
		port         string
		channel      string
		timeout      clienttypes.Height
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name:         "valid",
			sender:       validSender,
			receiver:     validSender,
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "class-1",
			tokenIDs:     []string{"1"},
			port:         "nft-transfer",
			channel:      "channel-0",
			timeout:      timeout,
			wantErr:      false,
		},
		{
			name:         "invalid sender",
			sender:       "bad",
			receiver:     validSender,
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "class-1",
			tokenIDs:     []string{"1"},
			port:         "nft-transfer",
			channel:      "channel-0",
			timeout:      timeout,
			wantErr:      true,
		},
		{
			name:         "invalid contract",
			sender:       validSender,
			receiver:     validSender,
			contractAddr: "not-hex",
			classID:      "class-1",
			tokenIDs:     []string{"1"},
			port:         "nft-transfer",
			channel:      "channel-0",
			timeout:      timeout,
			wantErr:      true,
		},
		{
			name:         "empty sender",
			sender:       "",
			receiver:     validSender,
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "class-1",
			tokenIDs:     []string{"1"},
			port:         "nft-transfer",
			channel:      "channel-0",
			timeout:      timeout,
			wantErr:      true,
		},
		{
			name:         "empty port",
			sender:       validSender,
			receiver:     validSender,
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "class-1",
			tokenIDs:     []string{"1"},
			port:         "",
			channel:      "channel-0",
			timeout:      timeout,
			wantErr:      true,
		},
		{
			name:         "empty channel",
			sender:       validSender,
			receiver:     validSender,
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "class-1",
			tokenIDs:     []string{"1"},
			port:         "nft-transfer",
			channel:      "",
			timeout:      timeout,
			wantErr:      true,
		},
		{
			name:         "zero timeout",
			sender:       validSender,
			receiver:     validSender,
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "class-1",
			tokenIDs:     []string{"1"},
			port:         "nft-transfer",
			channel:      "channel-0",
			wantErr:      true,
		},
		{
			name:         "invalid receiver",
			sender:       validSender,
			receiver:     "bad",
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "class-1",
			tokenIDs:     []string{"1"},
			port:         "nft-transfer",
			channel:      "channel-0",
			timeout:      timeout,
			wantErr:      true,
		},
		{
			// N-2 sentinel (round 18): the dfe8cee tightening added
			// strings.TrimSpace(msg.ClassId) == "" to MsgTransferERC721.
			name:         "empty class id rejected",
			sender:       validSender,
			receiver:     validSender,
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "",
			tokenIDs:     []string{"1"},
			port:         "nft-transfer",
			channel:      "channel-0",
			timeout:      timeout,
			wantErr:      true,
			wantErrMsg:   "class id cannot be empty",
		},
		{
			// N-2 reverse sentinel: a non-empty class id (including the IBC
			// voucher form) must continue to validate.
			name:         "ibc voucher class id accepted",
			sender:       validSender,
			receiver:     validSender,
			contractAddr: "0x1234567890123456789012345678901234567890",
			classID:      "ibc/ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890",
			tokenIDs:     []string{"1"},
			port:         "nft-transfer",
			channel:      "channel-0",
			timeout:      timeout,
			wantErr:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := MsgTransferERC721{
				CosmosSender:       tc.sender,
				CosmosReceiver:     tc.receiver,
				EvmContractAddress: tc.contractAddr,
				ClassId:            tc.classID,
				EvmTokenIds:        tc.tokenIDs,
				SourcePort:         tc.port,
				SourceChannel:      tc.channel,
				TimeoutHeight:      tc.timeout,
			}
			err := msg.ValidateBasic()
			if tc.wantErr {
				require.Error(t, err)
				if tc.wantErrMsg != "" {
					require.Contains(t, err.Error(), tc.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgTransferERC721_RouteAndType_ERC721(t *testing.T) {
	msg := MsgTransferERC721{}
	require.Equal(t, RouterKey, msg.Route())
	require.Equal(t, TypeMsgTransferERC721, msg.Type())
}

func TestMsgTransferERC721_GetSigners_ERC721(t *testing.T) {
	sender := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"
	msg := MsgTransferERC721{CosmosSender: sender}
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, sender, signers[0].String())
}
