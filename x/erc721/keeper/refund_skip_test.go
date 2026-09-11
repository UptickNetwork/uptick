package keeper

import (
	"errors"
	"os"
	"strings"
	"testing"

	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	nftTypes "github.com/UptickNetwork/uptick/x/collection/types"
	"github.com/UptickNetwork/uptick/x/erc721/contracts"
	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// These tests pin the round-10 N-1 fix: x/erc721's RefundPacketToken used to
// return an error for six per-token failure modes, aborting the whole IBC
// callback and stranding every other token in the packet. Every skip-able mode
// now emits a refund_packet_token_skip event and continues, mirroring x/cw721.
//
// Round 22 found the seventh site (a native burn failure) still returning an
// error in both modules; it skips too now. The invariant is enforced
// mechanically by TestRefundPacketTokenReturnsNoError below, because "fix one
// of the twins" has now happened three times.

// packOwnerOf encodes an ownerOf return carrying the given owner address.
func packOwnerOf(t *testing.T, owner common.Address) []byte {
	t.Helper()
	m, ok := contracts.ERC721UpticksContract.ABI.Methods["ownerOf"]
	require.True(t, ok, "ownerOf missing from ABI")
	bz, err := m.Outputs.Pack(owner)
	require.NoError(t, err, "pack ownerOf output")
	return bz
}

// moveNFTToModuleErc721 moves the preset native NFT (setupConvertKeeper mints
// kitty/nft1 to `owner`) into the module account, which is where the refund
// loop expects to find it.
func moveNFTToModuleErc721(t *testing.T, k Keeper, ctx sdk.Context, owner sdk.AccAddress, classID, nftID string) {
	t.Helper()
	nft, err := k.nftKeeper.GetNFT(ctx, classID, nftID)
	require.NoError(t, err)
	_, err = k.nftKeeper.TransferNFT(ctx, &nftTypes.MsgTransferNFT{
		DenomId:   classID,
		Id:        nftID,
		Name:      nft.GetName(),
		URI:       nft.GetURI(),
		Data:      nft.GetData(),
		UriHash:   nft.GetURIHash(),
		Sender:    owner.String(),
		Recipient: types.AccModuleAddress.String(),
	})
	require.NoError(t, err)
}

// setupRefundableToken records the reverse NFT-UID mapping, moves the native
// NFT to the module account, and records the refund receiver — the full state
// RefundPacketToken needs to complete a refund for this token.
func setupRefundableToken(t *testing.T, k Keeper, ctx sdk.Context, owner sdk.AccAddress, classID, nftID, evmTokenID string) {
	t.Helper()
	// setupConvertKeeper already saved the "kitty" denom (creator = owner),
	// minted kitty/nft1 to owner, and registered the class<->contract pair.
	require.NoError(t, k.SetNFTPairs(ctx, registerTestContract, evmTokenID, classID, nftID))
	moveNFTToModuleErc721(t, k, ctx, owner, classID, nftID)
	k.SetEvmRefundReceiver(ctx, registerTestContract, []string{nftID}, []string{evmTokenID}, common.BytesToAddress(owner.Bytes()).Hex())
}

// refundSkipReasons collects reason events keyed by the skipped NFT id.
func refundSkipReasons(ctx sdk.Context) map[string]string {
	reasons := make(map[string]string)
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type != types.EventTypeRefundPacketTokenSkip {
			continue
		}
		nftID, reason := "", ""
		for _, attr := range ev.Attributes {
			switch attr.Key {
			case types.AttributeKeyNFTID:
				nftID = attr.Value
			case "reason":
				reason = attr.Value
			}
		}
		reasons[nftID] = reason
	}
	return reasons
}

func hasRefundEvent(ctx sdk.Context) bool {
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == types.EventTypeRefundPacketToken {
			return true
		}
	}
	return false
}

// Mixed batch: nft2 has no pair at all, nft1 is fully refundable. Before the
// fix the missing pair aborted the callback before nft1 was ever refunded.
func TestRefundPacketToken_MissingPairDoesNotStrandOtherTokens(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)
	setupRefundableToken(t, k, ctx, owner, "kitty", "nft1", "1")

	// nft2 deliberately has no pair: no EVM call is made for it, so the seq
	// only covers nft1's ownerOf + safeTransferFrom.
	evm := k.evmKeeper.(*fakeEVMKeeper)
	evm.seq = []seqResp{
		{ret: packOwnerOf(t, types.ModuleAddress)},
		{ret: nil},
	}

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1", "nft2"},
	})
	require.NoError(t, err)

	// The refundable token was fully refunded.
	require.False(t, k.nftKeeper.HasNFT(ctx, "kitty", "nft1"), "nft1 must be burned after a successful refund")
	require.True(t, hasRefundEvent(ctx), "a refund event must be emitted for nft1")

	// And the unrefundable one is reported separately.
	reasons := refundSkipReasons(ctx)
	require.Equal(t, "erc721_pair_not_found", reasons["nft2"], "nft2 must produce a pair_not_found skip event")
}

// A stored pair whose UID cannot be split into (contract, token) is corrupt
// state; it must skip, not abort.
func TestRefundPacketToken_InvalidPairUIDSkips(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)
	setupRefundableToken(t, k, ctx, owner, "kitty", "nft1", "1")

	// nft2's reverse mapping holds a UID with no comma -> unusable.
	k.SetNFTUIDPairByNFTUID(ctx, types.CreateNFTUID("kitty", "nft2"), "not-a-pair")

	evm := k.evmKeeper.(*fakeEVMKeeper)
	evm.seq = []seqResp{
		{ret: packOwnerOf(t, types.ModuleAddress)},
		{ret: nil},
	}

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1", "nft2"},
	})
	require.NoError(t, err)

	require.True(t, hasRefundEvent(ctx), "nft1 must still be refunded")
	require.Equal(t, "erc721_pair_uid_invalid", refundSkipReasons(ctx)["nft2"])
}

// An EVM owner-query failure for one token must not block the others.
func TestRefundPacketToken_OwnerQueryFailureSkips(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)
	setupRefundableToken(t, k, ctx, owner, "kitty", "nft1", "1")

	evm := k.evmKeeper.(*fakeEVMKeeper)
	evm.seq = []seqResp{{err: errors.New("evm rpc down")}}

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err, "an owner-query failure must be skipped, not returned")

	require.Equal(t, "erc721_owner_query_failed", refundSkipReasons(ctx)["nft1"])
	// The token and its mappings stay intact so a later retry can refund it.
	require.True(t, k.nftKeeper.HasNFT(ctx, "kitty", "nft1"), "the native NFT must NOT be burned")
	require.NotEmpty(t, k.GetTokenUIDPairByNFTUID(ctx, types.CreateNFTUID("kitty", "nft1")), "the pair mapping must be kept for retry")
}

// A failed EVM transfer is the most dangerous skip: burning the native NFT
// without the EVM refund having completed would destroy the user's asset. The
// token must be left fully intact for retry.
func TestRefundPacketToken_TransferFailureSkipsAndKeepsToken(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)
	setupRefundableToken(t, k, ctx, owner, "kitty", "nft1", "1")

	evm := k.evmKeeper.(*fakeEVMKeeper)
	evm.seq = []seqResp{
		{ret: packOwnerOf(t, types.ModuleAddress)}, // ownerOf: module owns it
		{err: errors.New("evm revert")},            // safeTransferFrom fails
	}

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err)

	require.Equal(t, "erc721_transfer_failed", refundSkipReasons(ctx)["nft1"])
	require.True(t, k.nftKeeper.HasNFT(ctx, "kitty", "nft1"), "the native NFT must NOT be burned when the EVM refund did not happen")
	require.NotEmpty(t, k.GetTokenUIDPairByNFTUID(ctx, types.CreateNFTUID("kitty", "nft1")), "the pair mapping must be kept for retry")
	require.False(t, hasRefundEvent(ctx), "no refund event may be emitted for a token that was not refunded")
}

// Symmetry sentinel with x/cw721 (TestRefundPacketToken_MissingReceiver):
// a missing refund receiver must SKIP on both sides. Returning an error tore
// down the IBC callback's cache context, which left every other token in the
// packet unrefunded and made the packet retry forever. The token keeps its
// mapping and its native NFT so it can be refunded once the receiver record is
// repaired.
func TestRefundPacketToken_MissingReceiverSkips(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)
	require.NoError(t, k.SetNFTPairs(ctx, registerTestContract, "1", "kitty", "nft1"))
	moveNFTToModuleErc721(t, k, ctx, owner, "kitty", "nft1")
	// No SetEvmRefundReceiver call on purpose.

	evm := k.evmKeeper.(*fakeEVMKeeper)
	evm.seq = []seqResp{{ret: packOwnerOf(t, types.ModuleAddress)}}

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err, "a missing refund receiver must skip, not abort the packet")

	require.Equal(t, "erc721_refund_receiver_missing", refundSkipReasons(ctx)["nft1"])
	require.True(t, k.nftKeeper.HasNFT(ctx, "kitty", "nft1"), "the native NFT must stay for a later retry")
	require.NotEmpty(t, k.GetTokenUIDPairByNFTUID(ctx, types.CreateNFTUID("kitty", "nft1")), "the pair mapping must be kept for retry")
	require.False(t, hasRefundEvent(ctx), "no refund event may be emitted for a token that was not refunded")
}

// Mixed batch: the FIRST token is missing its refund receiver, the SECOND is
// fully refundable. Before the fix the first token aborted the callback and
// nft1 was never refunded.
func TestRefundPacketToken_MissingReceiverDoesNotStrandOtherTokens(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)
	setupRefundableToken(t, k, ctx, owner, "kitty", "nft1", "1")

	// nft2 has a pair and a module-owned NFT but no recorded receiver, so it is
	// the token that hits the missing-receiver branch. It is listed first.
	require.NoError(t, k.nftKeeper.SaveNFT(ctx, "kitty", "nft2", "Spot2", "ipfs://nft2", "", "", owner))
	require.NoError(t, k.SetNFTPairs(ctx, registerTestContract, "2", "kitty", "nft2"))
	moveNFTToModuleErc721(t, k, ctx, owner, "kitty", "nft2")

	evm := k.evmKeeper.(*fakeEVMKeeper)
	evm.seq = []seqResp{
		{ret: packOwnerOf(t, types.ModuleAddress)}, // nft2 ownerOf: module owns it
		{ret: packOwnerOf(t, types.ModuleAddress)}, // nft1 ownerOf
		{ret: nil}, // nft1 safeTransferFrom
	}

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft2", "nft1"},
	})
	require.NoError(t, err, "a missing receiver must not abort the packet")

	require.Equal(t, "erc721_refund_receiver_missing", refundSkipReasons(ctx)["nft2"])
	require.True(t, k.nftKeeper.HasNFT(ctx, "kitty", "nft2"), "nft2 keeps its native NFT for a later retry")

	require.False(t, k.nftKeeper.HasNFT(ctx, "kitty", "nft1"), "nft1 must still be refunded")
	require.True(t, hasRefundEvent(ctx), "nft1 must produce a refund event")
}

// TestRefundPacketTokenReturnsNoError is a static guard on the function body:
// no per-token failure inside RefundPacketToken may return an error.
//
// A returned error tears down the IBC callback, so the ack is never written,
// the relayer retries the packet forever, the tokens after the failing one are
// abandoned, and the retry observes a half-unwound state. The policy existed
// since round 10, but it was enforced by eye -- and the eye missed a site in
// each module (round 10 left six in erc721, round 22 found the burn-failure
// site in both twins). Checking it mechanically is the only version that
// survives the next twin-module edit.
func TestRefundPacketTokenReturnsNoError(t *testing.T) {
	body := functionSource(t, "msg_server.go", "func (k Keeper) RefundPacketToken")
	require.NotEmpty(t, body)
	// Prove the extraction really landed on RefundPacketToken; a silently empty
	// or wrong slice would make the loop below pass for the wrong reason.
	require.Contains(t, strings.Join(body, "\n"), "EventTypeRefundPacketTokenSkip")

	for _, line := range body {
		code := strings.TrimSpace(line)
		if strings.HasPrefix(code, "//") {
			continue
		}
		require.NotRegexp(t, `\breturn\s+\S*[Ee]rr`, code,
			"RefundPacketToken must skip and emit an event instead of returning an error (offending line: %q)", code)
	}
}

// functionSource returns the lines of the top-level function whose declaration
// starts with sig, up to and including its closing brace. gofmt puts that brace
// at column 0, which makes this a safe way to look at one function at a time.
func functionSource(t *testing.T, file, sig string) []string {
	t.Helper()

	src, err := os.ReadFile(file)
	require.NoError(t, err, "the guard must read the production source")

	lines := strings.Split(string(src), "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, sig) {
			start = i
			break
		}
	}
	require.GreaterOrEqual(t, start, 0, "function %q not found in %s", sig, file)

	for i := start + 1; i < len(lines); i++ {
		if lines[i] == "}" {
			return lines[start : i+1]
		}
	}
	t.Fatalf("unterminated function %q in %s", sig, file)
	return nil
}
