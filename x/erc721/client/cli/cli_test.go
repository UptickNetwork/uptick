package cli

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/app/params"
	"github.com/UptickNetwork/uptick/x/erc721/types"
)

func TestGetQueryCmd(t *testing.T) {
	t.Parallel()
	cmd := GetQueryCmd()
	require.Equal(t, "erc721", cmd.Use)
	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	require.ElementsMatch(t, []string{
		"token-pairs",
		"token-pair",
		"params",
		"evm-contract",
	}, names)
}

func TestNewTxCmd(t *testing.T) {
	t.Parallel()
	cmd := NewTxCmd()
	require.Equal(t, "erc721", cmd.Use)
	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	require.ElementsMatch(t, []string{
		"convert-nft",
		"convert-erc721",
		"ibc-transfer-erc721",
	}, names)
}

func newCLIClient(t *testing.T) (client.Context, sdk.AccAddress) {
	t.Helper()
	sdk.GetConfig().SetBech32PrefixForAccount("uptick", "uptickpub")

	enc := params.MakeEncodingConfig()
	types.RegisterInterfaces(enc.InterfaceRegistry)

	kb := keyring.NewInMemory(enc.Codec)
	record, _, err := kb.NewMnemonic("alice", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
	require.NoError(t, err)
	addr, err := record.GetAddress()
	require.NoError(t, err)

	ctx := client.Context{}.
		WithCodec(enc.Codec).
		WithInterfaceRegistry(enc.InterfaceRegistry).
		WithTxConfig(enc.TxConfig).
		WithLegacyAmino(enc.LegacyAmino).
		WithKeyring(kb).
		WithFromAddress(addr).
		WithFromName("alice")
	return ctx, addr
}

func generateOnlyArgs(from string, extra ...string) []string {
	args := append([]string{}, extra...)
	args = append(args,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, from),
		fmt.Sprintf("--%s=true", flags.FlagGenerateOnly),
		fmt.Sprintf("--%s=true", flags.FlagOffline),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=1", flags.FlagAccountNumber),
		fmt.Sprintf("--%s=1", flags.FlagSequence),
	)
	return args
}

func firstMsg[T sdk.Msg](t *testing.T, clientCtx client.Context, out []byte) T {
	t.Helper()
	tx, err := clientCtx.TxConfig.TxJSONDecoder()(out)
	require.NoError(t, err)
	msgs := tx.GetMsgs()
	require.Len(t, msgs, 1)
	msg, ok := msgs[0].(T)
	require.True(t, ok, "unexpected msg type %T", msgs[0])
	return msg
}

func TestConvertNFTCmd_GenerateOnly(t *testing.T) {
	clientCtx, addr := newCLIClient(t)
	evmReceiver := "0x2222222222222222222222222222222222222222"
	out, err := clitestutil.ExecTestCLICmd(clientCtx, NewConvertNFTCmd(), generateOnlyArgs("alice",
		"class-1", "nft-1,nft-2", "0x1111111111111111111111111111111111111111", "1,2", evmReceiver,
	))
	require.NoError(t, err)

	msg := firstMsg[*types.MsgConvertNFT](t, clientCtx, out.Bytes())
	require.Equal(t, "class-1", msg.ClassId)
	require.Equal(t, []string{"nft-1", "nft-2"}, msg.CosmosTokenIds)
	require.Equal(t, "0x1111111111111111111111111111111111111111", msg.EvmContractAddress)
	require.Equal(t, []string{"1", "2"}, msg.EvmTokenIds)
	require.Equal(t, evmReceiver, msg.EvmReceiver)
	require.Equal(t, addr.String(), msg.CosmosSender)
}

func TestConvertERC721Cmd_GenerateOnly(t *testing.T) {
	clientCtx, addr := newCLIClient(t)
	out, err := clitestutil.ExecTestCLICmd(clientCtx, NewConvertERC721Cmd(), generateOnlyArgs("alice",
		"0x1111111111111111111111111111111111111111", "1,2", "class-1", "nft-1,nft-2",
	))
	require.NoError(t, err)

	msg := firstMsg[*types.MsgConvertERC721](t, clientCtx, out.Bytes())
	require.Equal(t, "0x1111111111111111111111111111111111111111", msg.EvmContractAddress)
	require.Equal(t, []string{"1", "2"}, msg.EvmTokenIds)
	require.Equal(t, "class-1", msg.ClassId)
	require.Equal(t, addr.String(), msg.CosmosSender)
	require.Equal(t, addr.String(), msg.CosmosReceiver)
}

func TestTransferERC721Cmd_GenerateOnly(t *testing.T) {
	clientCtx, addr := newCLIClient(t)
	out, err := clitestutil.ExecTestCLICmd(clientCtx, NewTransferERC721Cmd(), generateOnlyArgs("alice",
		"0x1111111111111111111111111111111111111111", "1",
		"nonfungibletokentransfer", "channel-0",
		"uptick1dest", "class-1", "nft-1",
		fmt.Sprintf("--%s=true", flagAbsoluteTimeouts),
		fmt.Sprintf("--%s=1-100", flagPacketTimeoutHeight),
		fmt.Sprintf("--%s=1", flagPacketTimeoutTimestamp),
		fmt.Sprintf("--%s=%s", flagPacketMemo, `{"convert_to":"erc721"}`),
	))
	require.NoError(t, err)

	msg := firstMsg[*types.MsgTransferERC721](t, clientCtx, out.Bytes())
	require.Equal(t, "nonfungibletokentransfer", msg.SourcePort)
	require.Equal(t, "channel-0", msg.SourceChannel)
	require.Equal(t, "uptick1dest", msg.CosmosReceiver)
	require.Equal(t, addr.String(), msg.CosmosSender)
	require.Equal(t, `{"convert_to":"erc721"}`, msg.Memo)
	require.Equal(t, uint64(1), msg.TimeoutHeight.RevisionNumber)
	require.Equal(t, uint64(100), msg.TimeoutHeight.RevisionHeight)
}
