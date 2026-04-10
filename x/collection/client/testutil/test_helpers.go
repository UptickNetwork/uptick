package testutil

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/cli"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"

	nftcli "github.com/UptickNetwork/uptick/x/collection/client/cli"
)

func withExtraArgs(base []string, extraArgs ...string) []string {
	return append(base, extraArgs...)
}

func makeFromArgs(from string, leading ...string) []string {
	args := append([]string{}, leading...)
	args = append(args, fmt.Sprintf("--%s=%s", flags.FlagFrom, from))
	return args
}

func makeJSONQueryArgs(leading ...string) []string {
	args := append([]string{}, leading...)
	args = append(args, fmt.Sprintf("--%s=json", cli.OutputFlag))
	return args
}

// IssueDenomExec creates a redelegate message.
func IssueDenomExec(clientCtx client.Context, from string, denom string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := withExtraArgs(makeFromArgs(from, denom), extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, nftcli.GetCmdIssueDenom(), args)
}

func BurnNFTExec(clientCtx client.Context, from string, denomID string, tokenID string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := withExtraArgs(makeFromArgs(from, denomID, tokenID), extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, nftcli.GetCmdBurnNFT(), args)
}

func MintNFTExec(clientCtx client.Context, from string, denomID string, tokenID string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := withExtraArgs(makeFromArgs(from, denomID, tokenID), extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, nftcli.GetCmdMintNFT(), args)
}

func EditNFTExec(clientCtx client.Context, from string, denomID string, tokenID string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := withExtraArgs(makeFromArgs(from, denomID, tokenID), extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, nftcli.GetCmdEditNFT(), args)
}

func TransferNFTExec(clientCtx client.Context, from string, recipient string, denomID string, tokenID string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := withExtraArgs(makeFromArgs(from, recipient, denomID, tokenID), extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, nftcli.GetCmdTransferNFT(), args)
}

func QueryDenomExec(clientCtx client.Context, denomID string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := withExtraArgs(makeJSONQueryArgs(denomID), extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, nftcli.GetCmdQueryDenom(), args)
}

func QueryCollectionExec(clientCtx client.Context, denomID string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := withExtraArgs(makeJSONQueryArgs(denomID), extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, nftcli.GetCmdQueryCollection(), args)
}

func QueryDenomsExec(clientCtx client.Context, extraArgs ...string) (testutil.BufferWriter, error) {
	args := withExtraArgs(makeJSONQueryArgs(), extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, nftcli.GetCmdQueryDenoms(), args)
}

func QuerySupplyExec(clientCtx client.Context, denom string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := withExtraArgs(makeJSONQueryArgs(denom), extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, nftcli.GetCmdQuerySupply(), args)
}

func QueryOwnerExec(clientCtx client.Context, address string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := withExtraArgs(makeJSONQueryArgs(address), extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, nftcli.GetCmdQueryOwner(), args)
}

func QueryNFTExec(clientCtx client.Context, denomID string, tokenID string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := withExtraArgs(makeJSONQueryArgs(denomID, tokenID), extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, nftcli.GetCmdQueryNFT(), args)
}

func TransferDenomExec(clientCtx client.Context, from string, recipient string, denomID string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := withExtraArgs(makeFromArgs(from, recipient, denomID), extraArgs...)
	return clitestutil.ExecTestCLICmd(clientCtx, nftcli.GetCmdTransferDenom(), args)
}
