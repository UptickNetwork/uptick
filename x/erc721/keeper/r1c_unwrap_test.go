package keeper

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// recordingEVMKeeper wraps fakeEVMKeeper and captures the calldata of every
// EVM call so a test can tell burn(uint256) from safeTransferFrom (the two
// behaviours R1-C A3 switches between for voucher vs non-voucher classes).
type recordingEVMKeeper struct {
	*fakeEVMKeeper
	calldata [][]byte
}

func (r *recordingEVMKeeper) ApplyMessage(
	ctx sdk.Context,
	stateDB *statedb.StateDB,
	msg core.Message,
	hooks *tracing.Hooks,
	commit bool,
	a bool,
	b bool,
) (*evmtypes.MsgEthereumTxResponse, error) {
	r.calldata = append(r.calldata, msg.Data)
	return r.fakeEVMKeeper.ApplyMessage(ctx, stateDB, msg, hooks, commit, a, b)
}

func methodSelector(sig string) []byte {
	return crypto.Keccak256([]byte(sig))[:4]
}

func hasCalldata(rec *recordingEVMKeeper, selector []byte) bool {
	for _, cd := range rec.calldata {
		if len(cd) >= 4 && bytes.Equal(cd[:4], selector) {
			return true
		}
	}
	return false
}

// registerVoucherPair simulates a pre-R1-C legacy voucher pair: it bypasses the
// registration gate (which now rejects ibc/ classes) and writes the pair, its
// two lookup maps, a bidirectional binding, and a native NFT escrowed by the
// module account — the exact state a wrapped voucher is in before un-wrapping.
func registerVoucherPair(t *testing.T, k Keeper, ctx sdk.Context, owner sdk.AccAddress, classID, nftID, contractHex, evmTokenID string) common.Address {
	t.Helper()
	contract := common.HexToAddress(contractHex)
	pair := types.NewTokenPair(contract, classID)
	require.NoError(t, k.SetTokenPair(ctx, pair))
	k.SetClassMap(ctx, classID, pair.GetID())
	k.SetERC721Map(ctx, contract, pair.GetID())

	require.NoError(t, k.nftKeeper.SaveDenom(ctx, classID, "Voucher", "", "VCH", owner, false, false, "", "", "", ""))
	require.NoError(t, k.nftKeeper.SaveNFT(ctx, classID, nftID, "Voucher NFT", "ipfs://v", "", "", owner))
	moveNFTToModuleErc721(t, k, ctx, owner, classID, nftID)

	require.NoError(t, k.SetNFTPairs(ctx, contractHex, evmTokenID, classID, nftID))
	return contract
}

// T3 (R1-C A3): un-wrapping a voucher pair must be terminal — the ERC721 half
// is burned, the native voucher is handed to the receiver, and the bidirectional
// binding is deleted (not rewritten), which is what lets the voucher later pass
// the ICS-721 burn guard on its way back to the origin chain.
func TestConvertERC721_VoucherPairTerminalUnwrap(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)

	const (
		voucherClass    = "ibc/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		voucherContract = "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
		evmTokenID      = "1"
		nftID           = "voucher1"
	)
	contract := registerVoucherPair(t, k, ctx, owner, voucherClass, nftID, voucherContract, evmTokenID)

	rec := &recordingEVMKeeper{
		fakeEVMKeeper: &fakeEVMKeeper{
			accounts: map[common.Address]*statedb.Account{
				contract: {CodeHash: []byte{1, 2, 3}},
			},
		},
	}
	k.evmKeeper = rec

	sender := common.BytesToAddress(owner.Bytes())
	rec.seq = []seqResp{
		{ret: packNFTEnhanceOutputs(t, "Voucher", "ipfs://v", "", "")},
		{ret: packOwnerOf(t, sender)},
		{ret: nil}, // burn
	}

	receiver := sdk.AccAddress(bytes20(0x22))
	res, err := k.ConvertERC721(ctx, &types.MsgConvertERC721{
		ClassId:            voucherClass,
		CosmosTokenIds:     []string{nftID},
		EvmContractAddress: voucherContract,
		EvmTokenIds:        []string{evmTokenID},
		CosmosSender:       owner.String(),
		CosmosReceiver:     receiver.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	// The ERC721 half was burned, not safeTransferFrom'd back to the module.
	require.True(t, hasCalldata(rec, methodSelector("burn(uint256)")), "voucher un-wrap must burn the ERC721 half")
	require.False(t, hasCalldata(rec, methodSelector("safeTransferFrom(address,address,uint256)")), "voucher un-wrap must NOT escrow the ERC721 half")

	// The bidirectional binding is deleted, not rewritten.
	require.Empty(t, k.GetNFTPairByContractTokenID(ctx, voucherContract, evmTokenID), "forward binding must be deleted")
	require.Empty(t, k.GetNFTPairByClassNFTID(ctx, voucherClass, nftID), "reverse binding must be deleted")

	// The native voucher is owned by the receiver.
	got, err := k.nftKeeper.GetNFT(ctx, voucherClass, nftID)
	require.NoError(t, err)
	require.Equal(t, receiver.String(), got.GetOwner().String(), "native voucher must be handed to the receiver")
}

// T3 canary: a module-native ("uptick-…") class keeps the original escrow
// semantics — safeTransferFrom to the module and a rewritten binding — so A3
// does not change behaviour for non-voucher pairs.
func TestConvertERC721_UptickPairStillEscrows(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)

	const (
		classID    = "kitty"
		nftID      = "nft1"
		contract   = "0x1111111111111111111111111111111111111111"
		evmTokenID = "1"
	)
	require.NoError(t, k.SetNFTPairs(ctx, contract, evmTokenID, classID, nftID))
	moveNFTToModuleErc721(t, k, ctx, owner, classID, nftID)

	rec := &recordingEVMKeeper{
		fakeEVMKeeper: &fakeEVMKeeper{
			accounts: map[common.Address]*statedb.Account{
				common.HexToAddress(contract): {CodeHash: []byte{1, 2, 3}},
			},
		},
	}
	k.evmKeeper = rec

	sender := common.BytesToAddress(owner.Bytes())
	rec.seq = []seqResp{
		{ret: packNFTEnhanceOutputs(t, "Kitty", "ipfs://nft", "", "")},
		{ret: packOwnerOf(t, sender)},
		{ret: nil}, // safeTransferFrom
	}

	receiver := sdk.AccAddress(bytes20(0x22))
	_, err := k.ConvertERC721(ctx, &types.MsgConvertERC721{
		ClassId:            classID,
		CosmosTokenIds:     []string{nftID},
		EvmContractAddress: contract,
		EvmTokenIds:        []string{evmTokenID},
		CosmosSender:       owner.String(),
		CosmosReceiver:     receiver.String(),
	})
	require.NoError(t, err)

	require.True(t, hasCalldata(rec, methodSelector("safeTransferFrom(address,address,uint256)")), "uptick- class must still escrow via safeTransferFrom")
	require.False(t, hasCalldata(rec, methodSelector("burn(uint256)")), "uptick- class must NOT burn")

	// The binding is rewritten (still present), not deleted.
	require.NotEmpty(t, k.GetNFTPairByContractTokenID(ctx, contract, evmTokenID))
	require.NotEmpty(t, k.GetNFTPairByClassNFTID(ctx, classID, nftID))
}
