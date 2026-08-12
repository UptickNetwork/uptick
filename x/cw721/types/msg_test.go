package types

import (
	"testing"

	ibctypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/stretchr/testify/require"
)

func TestMsgConvertNFT_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		sender  string
		classID string
		nftIDs  []string
		wantErr bool
	}{
		{
			name:    "valid",
			sender:  "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
			classID: "class-1",
			nftIDs:  []string{"nft-1"},
			wantErr: false,
		},
		{
			name:    "empty sender",
			sender:  "",
			classID: "class-1",
			nftIDs:  []string{"nft-1"},
			wantErr: true,
		},
		{
			name:    "invalid sender",
			sender:  "invalid-address",
			classID: "class-1",
			nftIDs:  []string{"nft-1"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := MsgConvertNFT{
				Sender:  tc.sender,
				ClassId: tc.classID,
				NftIds:  tc.nftIDs,
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

func TestMsgConvertNFT_Route(t *testing.T) {
	msg := MsgConvertNFT{}
	require.Equal(t, RouterKey, msg.Route())
}

func TestMsgConvertNFT_Type(t *testing.T) {
	msg := MsgConvertNFT{}
	require.Equal(t, TypeMsgConvertNFT, msg.Type())
}

func TestMsgConvertNFT_GetSigners(t *testing.T) {
	sender := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"
	msg := MsgConvertNFT{Sender: sender}
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, sender, signers[0].String())
}

func TestMsgConvertNFT_GetSignBytes(t *testing.T) {
	msg := &MsgConvertNFT{
		Sender:  "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
		ClassId: "class-1",
		NftIds:  []string{"nft-1"},
	}
	bz := msg.GetSignBytes()
	require.NotEmpty(t, bz)
}

func TestMsgConvertCW721_ValidateBasic(t *testing.T) {
	tests := []struct {
		name     string
		sender   string
		receiver string
		contract string
		wantErr  bool
	}{
		{
			name:     "valid",
			sender:   "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
			receiver: "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
			contract: "0x1234567890123456789012345678901234567890",
			wantErr:  false,
		},
		{
			name:     "empty receiver",
			sender:   "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
			receiver: "",
			contract: "0x1234567890123456789012345678901234567890",
			wantErr:  true,
		},
		{
			name:     "invalid receiver",
			sender:   "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
			receiver: "not-a-valid-address",
			contract: "0x1234567890123456789012345678901234567890",
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := MsgConvertCW721{
				Sender:          tc.sender,
				Receiver:        tc.receiver,
				ContractAddress: tc.contract,
				TokenIds:        []string{"1"},
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

func TestMsgConvertCW721_Route(t *testing.T) {
	msg := MsgConvertCW721{}
	require.Equal(t, RouterKey, msg.Route())
}

func TestMsgConvertCW721_Type(t *testing.T) {
	msg := MsgConvertCW721{}
	require.Equal(t, TypeMsgConvertCW721, msg.Type())
}

func TestMsgConvertCW721_GetSigners(t *testing.T) {
	sender := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"
	msg := MsgConvertCW721{Sender: sender}
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, sender, signers[0].String())
}

func TestMsgConvertCW721_GetSignBytes(t *testing.T) {
	msg := &MsgConvertCW721{
		Sender:          "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
		ContractAddress: "0x1234567890123456789012345678901234567890",
		TokenIds:        []string{"1"},
		Receiver:        "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
	}
	bz := msg.GetSignBytes()
	require.NotEmpty(t, bz)
}

func TestMsgTransferCW721_ValidateBasic(t *testing.T) {
	validSender := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"

	tests := []struct {
		name         string
		cwSender     string
		contractAddr string
		sourcePort   string
		sourceChan   string
		wantErr      bool
	}{
		{
			name:         "valid",
			cwSender:     validSender,
			contractAddr: "0xabcdef0000000000000000000000000000abcdef",
			sourcePort:   "transfer",
			sourceChan:   "channel-0",
			wantErr:      false,
		},
		{
			name:         "empty sender",
			cwSender:     "",
			contractAddr: "0xabcdef0000000000000000000000000000abcdef",
			sourcePort:   "transfer",
			sourceChan:   "channel-0",
			wantErr:      true,
		},
		{
			name:         "invalid sender",
			cwSender:     "bad-address",
			contractAddr: "0xabcdef0000000000000000000000000000abcdef",
			sourcePort:   "transfer",
			sourceChan:   "channel-0",
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := MsgTransferCW721{
				CwSender:          tc.cwSender,
				CwContractAddress: tc.contractAddr,
				SourcePort:        tc.sourcePort,
				SourceChannel:     tc.sourceChan,
				TimeoutHeight:     ibctypes.Height{},
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

func TestMsgTransferCW721_Route(t *testing.T) {
	msg := MsgTransferCW721{}
	require.Equal(t, RouterKey, msg.Route())
}

func TestMsgTransferCW721_Type(t *testing.T) {
	msg := MsgTransferCW721{}
	require.Equal(t, TypeMsgTransferCW721, msg.Type())
}

func TestMsgTransferCW721_GetSigners(t *testing.T) {
	sender := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"
	msg := MsgTransferCW721{CwSender: sender}
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, sender, signers[0].String())
}

func TestMsgTransferCW721_GetSignBytes(t *testing.T) {
	msg := &MsgTransferCW721{
		CwSender:          "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
		CosmosReceiver:    "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
		CwContractAddress: "0xabcdef0000000000000000000000000000abcdef",
		CwTokenIds:        []string{"42"},
		SourcePort:        "transfer",
		SourceChannel:     "channel-0",
		TimeoutHeight:     ibctypes.Height{},
	}
	bz := msg.GetSignBytes()
	require.NotEmpty(t, bz)
}
