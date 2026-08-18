package testutil

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/app/params"
	nftcli "github.com/UptickNetwork/uptick/x/collection/client/cli"
	"github.com/UptickNetwork/uptick/x/collection/types"
)

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

func generateOnlyArgs(extra ...string) []string {
	return append(append([]string{}, extra...),
		fmt.Sprintf("--%s=true", flags.FlagGenerateOnly),
		fmt.Sprintf("--%s=true", flags.FlagOffline),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=1", flags.FlagAccountNumber),
		fmt.Sprintf("--%s=1", flags.FlagSequence),
	)
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

func TestIssueDenomExec_GenerateOnly(t *testing.T) {
	clientCtx, addr := newCLIClient(t)
	out, err := IssueDenomExec(clientCtx, "alice", "kitty", generateOnlyArgs(
		fmt.Sprintf("--%s=%s", nftcli.FlagDenomName, "Kitty"),
		fmt.Sprintf("--%s=%s", nftcli.FlagSymbol, "KIT"),
		fmt.Sprintf("--%s=false", nftcli.FlagMintRestricted),
		fmt.Sprintf("--%s=false", nftcli.FlagUpdateRestricted),
		fmt.Sprintf("--%s=%s", nftcli.FlagURI, "ipfs://class"),
	)...)
	require.NoError(t, err)

	msg := firstMsg[*types.MsgIssueDenom](t, clientCtx, out.Bytes())
	require.Equal(t, "kitty", msg.Id)
	require.Equal(t, "Kitty", msg.Name)
	require.Equal(t, "KIT", msg.Symbol)
	require.Equal(t, addr.String(), msg.Sender)
	require.Equal(t, "ipfs://class", msg.Uri)
}

func TestMintNFTExec_GenerateOnly(t *testing.T) {
	clientCtx, addr := newCLIClient(t)
	out, err := MintNFTExec(clientCtx, "alice", "kitty", "nft1", generateOnlyArgs(
		fmt.Sprintf("--%s=%s", nftcli.FlagTokenName, "Spot"),
		fmt.Sprintf("--%s=%s", nftcli.FlagURI, "ipfs://nft"),
	)...)
	require.NoError(t, err)

	msg := firstMsg[*types.MsgMintNFT](t, clientCtx, out.Bytes())
	require.Equal(t, "kitty", msg.DenomId)
	require.Equal(t, "nft1", msg.Id)
	require.Equal(t, "Spot", msg.Name)
	require.Equal(t, addr.String(), msg.Sender)
	require.Equal(t, addr.String(), msg.Recipient)
}

func TestEditNFTExec_GenerateOnly(t *testing.T) {
	clientCtx, addr := newCLIClient(t)
	out, err := EditNFTExec(clientCtx, "alice", "kitty", "nft1", generateOnlyArgs(
		fmt.Sprintf("--%s=%s", nftcli.FlagURI, "ipfs://updated"),
	)...)
	require.NoError(t, err)

	msg := firstMsg[*types.MsgEditNFT](t, clientCtx, out.Bytes())
	require.Equal(t, "kitty", msg.DenomId)
	require.Equal(t, "nft1", msg.Id)
	require.Equal(t, "ipfs://updated", msg.URI)
	require.Equal(t, addr.String(), msg.Sender)
}

func TestTransferNFTExec_GenerateOnly(t *testing.T) {
	clientCtx, addr := newCLIClient(t)
	recipient := sdk.AccAddress(make([]byte, 20)).String()
	out, err := TransferNFTExec(clientCtx, "alice", recipient, "kitty", "nft1", generateOnlyArgs()...)
	require.NoError(t, err)

	msg := firstMsg[*types.MsgTransferNFT](t, clientCtx, out.Bytes())
	require.Equal(t, "kitty", msg.DenomId)
	require.Equal(t, "nft1", msg.Id)
	require.Equal(t, recipient, msg.Recipient)
	require.Equal(t, addr.String(), msg.Sender)
}

func TestBurnNFTExec_GenerateOnly(t *testing.T) {
	clientCtx, addr := newCLIClient(t)
	out, err := BurnNFTExec(clientCtx, "alice", "kitty", "nft1", generateOnlyArgs()...)
	require.NoError(t, err)

	msg := firstMsg[*types.MsgBurnNFT](t, clientCtx, out.Bytes())
	require.Equal(t, "kitty", msg.DenomId)
	require.Equal(t, "nft1", msg.Id)
	require.Equal(t, addr.String(), msg.Sender)
}

func TestTransferDenomExec_GenerateOnly(t *testing.T) {
	clientCtx, addr := newCLIClient(t)
	recipient := sdk.AccAddress(make([]byte, 20)).String()
	out, err := TransferDenomExec(clientCtx, "alice", recipient, "kitty", generateOnlyArgs()...)
	require.NoError(t, err)

	msg := firstMsg[*types.MsgTransferDenom](t, clientCtx, out.Bytes())
	require.Equal(t, "kitty", msg.Id)
	require.Equal(t, recipient, msg.Recipient)
	require.Equal(t, addr.String(), msg.Sender)
}
