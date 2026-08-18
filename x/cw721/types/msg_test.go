package types

import (
	"testing"

	ibctypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/stretchr/testify/require"
)

func TestMsgConvertNFT_ValidateBasic(t *testing.T) {
	validAddr := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"
	tests := []struct {
		name     string
		sender   string
		receiver string
		contract string
		classID  string
		nftIDs   []string
		tokenIDs []string
		wantErr  bool
	}{
		{
			name:     "valid",
			sender:   validAddr,
			receiver: validAddr,
			classID:  "class-1",
			nftIDs:   []string{"nft-1"},
			wantErr:  false,
		},
		{
			name:     "empty sender",
			sender:   "",
			receiver: validAddr,
			classID:  "class-1",
			nftIDs:   []string{"nft-1"},
			wantErr:  true,
		},
		{
			name:     "invalid sender",
			sender:   "invalid-address",
			receiver: validAddr,
			classID:  "class-1",
			nftIDs:   []string{"nft-1"},
			wantErr:  true,
		},
		{
			name:     "empty receiver",
			sender:   validAddr,
			receiver: "",
			classID:  "class-1",
			nftIDs:   []string{"nft-1"},
			wantErr:  true,
		},
		{
			name:     "invalid receiver",
			sender:   validAddr,
			receiver: "invalid-address",
			classID:  "class-1",
			nftIDs:   []string{"nft-1"},
			wantErr:  true,
		},
		{
			name:     "invalid contract",
			sender:   validAddr,
			receiver: validAddr,
			contract: "0x1234567890123456789012345678901234567890",
			classID:  "class-1",
			nftIDs:   []string{"nft-1"},
			wantErr:  true,
		},
		{
			name:     "empty class id",
			sender:   validAddr,
			receiver: validAddr,
			classID:  "",
			nftIDs:   []string{"nft-1"},
			wantErr:  true,
		},
		{
			name:     "empty nft ids",
			sender:   validAddr,
			receiver: validAddr,
			classID:  "class-1",
			nftIDs:   nil,
			wantErr:  true,
		},
		{
			name:     "blank nft id",
			sender:   validAddr,
			receiver: validAddr,
			classID:  "class-1",
			nftIDs:   []string{""},
			wantErr:  true,
		},
		{
			name:     "blank token id",
			sender:   validAddr,
			receiver: validAddr,
			classID:  "class-1",
			nftIDs:   []string{"nft-1"},
			tokenIDs: []string{""},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := MsgConvertNFT{
				Sender:          tc.sender,
				Receiver:        tc.receiver,
				ContractAddress: tc.contract,
				ClassId:         tc.classID,
				NftIds:          tc.nftIDs,
				TokenIds:        tc.tokenIDs,
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
	validAddr := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"
	tests := []struct {
		name     string
		sender   string
		receiver string
		contract string
		tokenIDs []string
		wantErr  bool
	}{
		{
			name:     "valid",
			sender:   validAddr,
			receiver: validAddr,
			contract: validAddr,
			tokenIDs: []string{"1"},
			wantErr:  false,
		},
		{
			name:     "empty receiver",
			sender:   validAddr,
			receiver: "",
			contract: validAddr,
			tokenIDs: []string{"1"},
			wantErr:  true,
		},
		{
			name:     "invalid receiver",
			sender:   validAddr,
			receiver: "not-a-valid-address",
			contract: validAddr,
			tokenIDs: []string{"1"},
			wantErr:  true,
		},
		{
			name:     "empty sender",
			sender:   "",
			receiver: validAddr,
			contract: validAddr,
			tokenIDs: []string{"1"},
			wantErr:  true,
		},
		{
			name:     "invalid sender",
			sender:   "invalid-address",
			receiver: validAddr,
			contract: validAddr,
			tokenIDs: []string{"1"},
			wantErr:  true,
		},
		{
			name:     "empty contract",
			sender:   validAddr,
			receiver: validAddr,
			contract: "",
			tokenIDs: []string{"1"},
			wantErr:  true,
		},
		{
			name:     "empty token ids",
			sender:   validAddr,
			receiver: validAddr,
			contract: validAddr,
			tokenIDs: nil,
			wantErr:  true,
		},
		{
			name:     "blank token id",
			sender:   validAddr,
			receiver: validAddr,
			contract: validAddr,
			tokenIDs: []string{""},
			wantErr:  true,
		},
		{
			name:     "invalid contract",
			sender:   validAddr,
			receiver: validAddr,
			contract: "0x1234567890123456789012345678901234567890",
			tokenIDs: []string{"1"},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := MsgConvertCW721{
				Sender:          tc.sender,
				Receiver:        tc.receiver,
				ContractAddress: tc.contract,
				TokenIds:        tc.tokenIDs,
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
	validReceiver := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"
	validContract := "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"

	tests := []struct {
		name           string
		cwSender       string
		cosmosReceiver string
		contractAddr   string
		cwTokenIds     []string
		sourcePort     string
		sourceChan     string
		timeoutHeight  ibctypes.Height
		timeoutTime    uint64
		wantErr        bool
	}{
		{
			name:           "valid",
			cwSender:       validSender,
			cosmosReceiver: validReceiver,
			contractAddr:   validContract,
			cwTokenIds:     []string{"1"},
			sourcePort:     "transfer",
			sourceChan:     "channel-0",
			timeoutHeight:  ibctypes.Height{RevisionNumber: 1, RevisionHeight: 100},
			wantErr:        false,
		},
		{
			name:           "empty sender",
			cwSender:       "",
			cosmosReceiver: validReceiver,
			contractAddr:   validContract,
			cwTokenIds:     []string{"1"},
			sourcePort:     "transfer",
			sourceChan:     "channel-0",
			timeoutHeight:  ibctypes.Height{RevisionNumber: 1, RevisionHeight: 100},
			wantErr:        true,
		},
		{
			name:           "invalid sender",
			cwSender:       "bad-address",
			cosmosReceiver: validReceiver,
			contractAddr:   validContract,
			cwTokenIds:     []string{"1"},
			sourcePort:     "transfer",
			sourceChan:     "channel-0",
			timeoutHeight:  ibctypes.Height{RevisionNumber: 1, RevisionHeight: 100},
			wantErr:        true,
		},
		{
			name:           "empty receiver",
			cwSender:       validSender,
			cosmosReceiver: "",
			contractAddr:   validContract,
			cwTokenIds:     []string{"1"},
			sourcePort:     "transfer",
			sourceChan:     "channel-0",
			timeoutHeight:  ibctypes.Height{RevisionNumber: 1, RevisionHeight: 100},
			wantErr:        true,
		},
		{
			name:           "invalid contract",
			cwSender:       validSender,
			cosmosReceiver: validReceiver,
			contractAddr:   "0xabcdef0000000000000000000000000000abcdef",
			cwTokenIds:     []string{"1"},
			sourcePort:     "transfer",
			sourceChan:     "channel-0",
			timeoutHeight:  ibctypes.Height{RevisionNumber: 1, RevisionHeight: 100},
			wantErr:        true,
		},
		{
			name:           "empty cw token ids",
			cwSender:       validSender,
			cosmosReceiver: validReceiver,
			contractAddr:   validContract,
			cwTokenIds:     nil,
			sourcePort:     "transfer",
			sourceChan:     "channel-0",
			timeoutHeight:  ibctypes.Height{RevisionNumber: 1, RevisionHeight: 100},
			wantErr:        true,
		},
		{
			name:           "empty source port",
			cwSender:       validSender,
			cosmosReceiver: validReceiver,
			contractAddr:   validContract,
			cwTokenIds:     []string{"1"},
			sourcePort:     "",
			sourceChan:     "channel-0",
			timeoutHeight:  ibctypes.Height{RevisionNumber: 1, RevisionHeight: 100},
			wantErr:        true,
		},
		{
			name:           "empty source channel",
			cwSender:       validSender,
			cosmosReceiver: validReceiver,
			contractAddr:   validContract,
			cwTokenIds:     []string{"1"},
			sourcePort:     "transfer",
			sourceChan:     "",
			timeoutHeight:  ibctypes.Height{RevisionNumber: 1, RevisionHeight: 100},
			wantErr:        true,
		},
		{
			name:           "both timeouts zero",
			cwSender:       validSender,
			cosmosReceiver: validReceiver,
			contractAddr:   validContract,
			cwTokenIds:     []string{"1"},
			sourcePort:     "transfer",
			sourceChan:     "channel-0",
			timeoutHeight:  ibctypes.Height{},
			wantErr:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := MsgTransferCW721{
				CwSender:          tc.cwSender,
				CosmosReceiver:    tc.cosmosReceiver,
				CwContractAddress: tc.contractAddr,
				CwTokenIds:        tc.cwTokenIds,
				SourcePort:        tc.sourcePort,
				SourceChannel:     tc.sourceChan,
				TimeoutHeight:     tc.timeoutHeight,
				TimeoutTimestamp:  tc.timeoutTime,
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
