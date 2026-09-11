package keeper

import (
	"os"
	"strings"
	"testing"

	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
)

// refundAttr flattens an event's attributes for stable lookups.
func refundAttr(event sdk.Event) map[string]string {
	out := make(map[string]string, len(event.Attributes))
	for _, attr := range event.Attributes {
		out[attr.Key] = attr.Value
	}
	return out
}

func findEvent(events sdk.Events, eventType string) *sdk.Event {
	for i := range events {
		if events[i].Type == eventType {
			return &events[i]
		}
	}
	return nil
}

// moveNFTToModule transfers the native NFT into the cw721 module account so
// RefundPacketToken is authorised to burn it.
func moveNFTToModule(t *testing.T, k Keeper, ctx sdk.Context, from sdk.AccAddress, classID, nftID string) {
	t.Helper()
	nft, err := k.nftKeeper.GetNFT(ctx, classID, nftID)
	require.NoError(t, err)

	_, err = k.nftKeeper.TransferNFT(ctx, &collectiontypes.MsgTransferNFT{
		DenomId:   classID,
		Id:        nftID,
		Name:      nft.GetName(),
		URI:       nft.GetURI(),
		Data:      nft.GetData(),
		UriHash:   nft.GetURIHash(),
		Sender:    from.String(),
		Recipient: cw721types.AccModuleAddress.String(),
	})
	require.NoError(t, err)
}

// A token with no recorded pair has nothing to refund and no contract/token id
// to refund it to. Returning an error here aborts the whole IBC callback and
// strands every remaining token in the packet (round 10, G-4), so the token is
// skipped with a distinguishable event instead.
func TestRefundPacketToken_SkipsWhenPairMissing(t *testing.T) {
	k, ctx := setupKeeper(t)

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err, "a missing pair must be skipped, not abort the IBC callback")

	skip := findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketTokenSkip)
	require.NotNil(t, skip, "expected a %s event", cw721types.EventTypeRefundPacketTokenSkip)
	require.Equal(t, "cw721_pair_not_found", refundAttr(*skip)["reason"])
	require.Equal(t, "nft1", refundAttr(*skip)[cw721types.AttributeKeyNFTID])
	require.Equal(t, "kitty", refundAttr(*skip)[cw721types.AttributeKeyNFTClass])
}

// The point of the skip behaviour: one unrefundable token must not stop the
// rest of the packet from being refunded. Before the fix, a missing pair
// aborted the callback and every other token in the batch was stranded.
func TestRefundPacketToken_MissingPairDoesNotStrandOtherTokens(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	// Only nft1 has a pair; nft2 deliberately has none.
	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	moveNFTToModule(t, k, ctx, owner, "kitty", "nft1")
	wasm.setOwner(contract, "1", cw721types.AccModuleAddress.String())
	k.SetCwAddressByContractTokenId(ctx, contract, "1", owner.String())

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1", "nft2"},
	})
	require.NoError(t, err)

	// The refundable token was still refunded.
	refund := findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketToken)
	require.NotNil(t, refund, "nft1 must be refunded even though nft2 has no pair")
	require.Equal(t, owner.String(), wasm.ownerOf(contract, "1"))

	// And the unrefundable one is reported separately.
	skips := ctx.EventManager().Events()
	var seen bool
	for i := range skips {
		if skips[i].Type != cw721types.EventTypeRefundPacketTokenSkip {
			continue
		}
		attrs := refundAttr(skips[i])
		if attrs[cw721types.AttributeKeyNFTID] == "nft2" {
			seen = true
			require.Equal(t, "cw721_pair_not_found", attrs["reason"])
		}
	}
	require.True(t, seen, "expected a cw721_pair_not_found skip event for nft2")
}

// A missing refund receiver must SKIP (mirrors x/erc721): returning an error
// tore down the IBC callback's cache context, which left every other token in
// the packet unrefunded and made the packet retry forever.
func TestRefundPacketToken_MissingReceiverSkips(t *testing.T) {
	k, ctx, _, contract, wasm := setupConvertKeeper(t)

	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	// The module still escrows the CW721, so the refund path is taken and it
	// requires a receiver recorded at send time — none is recorded on purpose.
	wasm.setOwner(contract, "1", cw721types.AccModuleAddress.String())

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err, "a missing refund receiver must skip, not abort the packet")

	skip := findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketTokenSkip)
	require.NotNil(t, skip, "expected a %s event", cw721types.EventTypeRefundPacketTokenSkip)
	require.Equal(t, "cw721_refund_receiver_missing", refundAttr(*skip)["reason"])

	// The token was not moved and no refund was claimed, so a later repair can
	// still process it.
	require.Equal(t, cw721types.AccModuleAddress.String(), wasm.ownerOf(contract, "1"))
	require.Nil(t, findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketToken),
		"no refund event may be emitted for a token that was not refunded")
}

// Mixed batch: the FIRST token is missing its refund receiver, the SECOND is
// fully refundable. Before the fix the first token aborted the callback and
// nft1 was never refunded.
func TestRefundPacketToken_MissingReceiverDoesNotStrandOtherTokens(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	// nft1 is fully refundable (pair + escrow + recorded receiver)...
	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	moveNFTToModule(t, k, ctx, owner, "kitty", "nft1")
	wasm.setOwner(contract, "1", cw721types.AccModuleAddress.String())
	k.SetCwAddressByContractTokenId(ctx, contract, "1", owner.String())

	// ...while nft2 is escrowed by the module but has NO recorded receiver.
	require.NoError(t, k.nftKeeper.SaveNFT(ctx, "kitty", "nft2", "Spot2", "ipfs://nft2", "", "", owner))
	require.NoError(t, k.SetNFTPairs(ctx, contract, "2", "kitty", "nft2"))
	moveNFTToModule(t, k, ctx, owner, "kitty", "nft2")
	wasm.setOwner(contract, "2", cw721types.AccModuleAddress.String())

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft2", "nft1"},
	})
	require.NoError(t, err, "a missing receiver must not abort the packet")

	var nft2Reason string
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type != cw721types.EventTypeRefundPacketTokenSkip {
			continue
		}
		if attrs := refundAttr(ev); attrs[cw721types.AttributeKeyNFTID] == "nft2" {
			nft2Reason = attrs["reason"]
		}
	}
	require.Equal(t, "cw721_refund_receiver_missing", nft2Reason)

	require.NotNil(t, findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketToken),
		"nft1 must be refunded even though nft2 lacks a receiver")
	require.Equal(t, owner.String(), wasm.ownerOf(contract, "1"))
	require.False(t, k.nftKeeper.HasNFT(ctx, "kitty", "nft1"), "nft1 must be burned after the refund")
	require.True(t, k.nftKeeper.HasNFT(ctx, "kitty", "nft2"), "nft2 keeps its NFT for a later retry")
}

// When the module account no longer owns the CW721 the refund must be skipped
// instead of returning an error (an error would strand the IBC packet).
func TestRefundPacketToken_SkipsWhenModuleDoesNotOwnToken(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	moveNFTToModule(t, k, ctx, owner, "kitty", "nft1")
	// The CW721 is held by somebody else: already refunded, never escrowed, or
	// moved on after the failed transfer.
	wasm.setOwner(contract, "1", owner.String())

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err, "a non-escrowed token must be skipped, not abort the IBC callback")

	// A skip event explains why nothing was transferred.
	skip := findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketTokenSkip)
	require.NotNil(t, skip, "expected a %s event", cw721types.EventTypeRefundPacketTokenSkip)
	require.Equal(t, "owner_is_not_module_account", refundAttr(*skip)["reason"])
	require.Equal(t, "1", refundAttr(*skip)[cw721types.AttributeKeyCW721TokenID])

	// No refund event was emitted.
	require.Nil(t, findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketToken))

	// The token must not have moved to an arbitrary receiver.
	require.Equal(t, owner.String(), wasm.ownerOf(contract, "1"))

	// Cleanup still happens: the NFT is burned and the mappings released.
	require.Empty(t, k.GetNFTPairByContractTokenID(ctx, contract, "1"))
	require.Empty(t, k.GetTokenUIDPairByNFTUID(ctx, cw721types.CreateNFTUID("kitty", "nft1")))
	_, err = k.nftKeeper.GetNFT(ctx, "kitty", "nft1")
	require.Error(t, err, "the escrowed NFT must be burned even when no CW721 is refunded")
}

func TestRefundPacketToken_RefundsWhenModuleOwnsToken(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	moveNFTToModule(t, k, ctx, owner, "kitty", "nft1")
	wasm.setOwner(contract, "1", cw721types.AccModuleAddress.String())
	k.SetCwAddressByContractTokenId(ctx, contract, "1", owner.String())

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err)

	// The CW721 went back to the original owner.
	require.Equal(t, owner.String(), wasm.ownerOf(contract, "1"))

	// The event reports the CW721 token id (not the NFT id) and the receiver.
	ev := findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketToken)
	require.NotNil(t, ev)
	attrs := refundAttr(*ev)
	require.Equal(t, "1", attrs[cw721types.AttributeKeyCW721TokenID])
	require.Equal(t, "nft1", attrs[cw721types.AttributeKeyNFTID])
	require.Equal(t, contract, attrs[cw721types.AttributeKeyCW721Token])
	require.Equal(t, owner.String(), attrs[cw721types.AttributeKeyReceiver])

	// State is cleaned up.
	require.Empty(t, k.GetNFTPairByContractTokenID(ctx, contract, "1"))
	require.Empty(t, k.GetCwAddressByContractTokenId(ctx, contract, "1"))
	_, err = k.nftKeeper.GetNFT(ctx, "kitty", "nft1")
	require.Error(t, err)
}

// A token the CW721 contract does not know (never escrowed for the module, or
// minted straight to a user) must be skipped rather than abort the refund.
func TestRefundPacketToken_SkipsWhenOwnerUnknown(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	moveNFTToModule(t, k, ctx, owner, "kitty", "nft1")
	// No owner recorded: as far as the contract is concerned the token is absent.
	wasm.setOwner(contract, "1", "")
	k.SetCwAddressByContractTokenId(ctx, contract, "1", owner.String())

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err, "an unknown owner must skip, not abort the IBC callback")

	require.NotNil(t, findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketTokenSkip))
	require.Empty(t, wasm.ownerOf(contract, "1"))
	require.Empty(t, k.GetNFTPairByContractTokenID(ctx, contract, "1"))
}

func TestModuleOwnsCW721(t *testing.T) {
	t.Parallel()

	canonical := cw721types.AccModuleAddress.String()
	require.True(t, moduleOwnsCW721(canonical))
	require.False(t, moduleOwnsCW721("uptick1someoneelse"))
	require.False(t, moduleOwnsCW721(""))

	// bech32 is case-insensitive and the value comes from a contract query
	// response, so any case variant of the module address must match.
	require.True(t, moduleOwnsCW721(strings.ToUpper(canonical)))
	require.True(t, moduleOwnsCW721(strings.ToLower(canonical)))
	require.True(t, moduleOwnsCW721(strings.ToUpper(canonical[:1])+canonical[1:]),
		"mixed case must match too: a strict bech32 decoder may reject it, so a plain decode-only comparison is not enough")

	// A different address that merely decodes is still not the module.
	require.False(t, moduleOwnsCW721(sdk.AccAddress([]byte("someone-else-addr-01")).String()))
}

// The gate that decides whether the module still escrows the token reads its
// input straight out of the CW721 contract's all_nft_info response, and bech32
// is case-insensitive. Before the fix a contract echoing the module address in
// a non-canonical case was read as "not the module" and the refund was skipped
// even though the module really did hold the token — permanently stranding the
// escrowed native NFT. This asserts the whole refund path, not just the
// predicate in isolation.
func TestRefundPacketToken_RefundsOnNonCanonicalModuleAddressCase(t *testing.T) {
	for _, name := range []string{"upper", "mixed"} {
		t.Run(name, func(t *testing.T) {
			k, ctx, owner, contract, wasm := setupConvertKeeper(t)

			// Resolve the canonical string only AFTER the keeper setup has
			// installed the chain's bech32 prefix. sdk.AccAddress.String()
			// caches its result process-wide, so the first call anywhere in
			// this test binary fixes the display form for every later call —
			// computing it before the prefix is set yields a cosmos1... form
			// that the validators below then reject as "expected uptick".
			canonical := cw721types.AccModuleAddress.String()
			variant := strings.ToUpper(canonical)
			if name == "mixed" {
				variant = strings.ToUpper(canonical[:1]) + canonical[1:]
			}

			require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
			moveNFTToModule(t, k, ctx, owner, "kitty", "nft1")
			// The contract reports the module as owner, in a variant case.
			wasm.setOwner(contract, "1", variant)
			k.SetCwAddressByContractTokenId(ctx, contract, "1", owner.String())

			err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
				ClassId:  "kitty",
				TokenIds: []string{"nft1"},
			})
			require.NoError(t, err)

			require.Nil(t, findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketTokenSkip),
				"the module does own the token, so nothing may be skipped")
			require.NotNil(t, findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketToken),
				"a case variant of the module address must still be refunded")
			require.Equal(t, owner.String(), wasm.ownerOf(contract, "1"),
				"the CW721 token must be handed back to the depositor")
		})
	}
}

// Defense-in-depth: if the native NFT is already gone when the refund runs,
// the burn step must be skipped instead of returning an error.
func TestRefundPacketToken_SkipsWhenNativeNFTAlreadyGone(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	moveNFTToModule(t, k, ctx, owner, "kitty", "nft1")
	wasm.setOwner(contract, "1", cw721types.AccModuleAddress.String())
	k.SetCwAddressByContractTokenId(ctx, contract, "1", owner.String())

	// Simulate the NFT being gone before the burn step. The cw721 side
	// still thinks the module escrows it; only the native side is empty.
	require.NoError(t, k.nftKeeper.NFTkeeper().Burn(ctx, "kitty", "nft1"))

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err, "a missing NFT must skip the burn, not abort the IBC callback")

	// The skip event carries the reason so operators can distinguish this
	// case from the cw721-side skip.
	skip := findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketTokenSkip)
	require.NotNil(t, skip)
	require.Equal(t, "nft_already_gone", refundAttr(*skip)["reason"])

	// Round 25: this reason is shared with x/erc721, where it means the
	// opposite (pair mappings retained, human triage required) and is flagged
	// with pairs_retained=true. Here the mappings were already deleted above
	// because the CW721 side is settled, so the attribute must be absent —
	// absence is the "converged" half of the distinction an operator reads.
	_, hasPairsRetained := refundAttr(*skip)["pairs_retained"]
	require.False(t, hasPairsRetained,
		"cw721's nft_already_gone is the converged case and must not claim retained pairs")

	// The mappings really are gone, which is what makes the claim above true
	// rather than a convention.
	require.Empty(t, k.GetNFTPairByClassNFTID(ctx, "kitty", "nft1"),
		"the pair mapping must be deleted on this path, unlike x/erc721")
}

// Defense-in-depth: the native NFT exists but is owned by a
// non-module address (e.g. already refunded on a parallel path). Burn must
// be skipped instead of returning an error from BurnNFT.
func TestRefundPacketToken_SkipsWhenNativeNFTOwnerIsNotModule(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	moveNFTToModule(t, k, ctx, owner, "kitty", "nft1")
	// cw721 side happy; the native NFT moved elsewhere (a parallel refund path).
	wasm.setOwner(contract, "1", cw721types.AccModuleAddress.String())
	k.SetCwAddressByContractTokenId(ctx, contract, "1", owner.String())
	// Re-acquire from module and re-issue to a fresh user via the underlying
	// nft keeper so the collection-level state is updated without going
	// through the message-server path.
	require.NoError(t, k.nftKeeper.NFTkeeper().Transfer(ctx, "kitty", "nft1", owner))

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err, "an NFT not held by module must skip the burn")

	sk := findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketTokenSkip)
	require.NotNil(t, sk)
	require.Equal(t, "nft_owner_not_module", refundAttr(*sk)["reason"])
	require.Equal(t, owner.String(), refundAttr(*sk)[cw721types.AttributeKeyNFTOwner])

	// The recipient is not double-burned: it still owns the NFT.
	curr, err := k.nftKeeper.GetNFT(ctx, "kitty", "nft1")
	require.NoError(t, err)
	require.NotEmpty(t, curr.GetID())
}

func TestQueryCW721TokenOwner(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	wasm.setOwner(contract, "1", owner.String())
	got, err := k.QueryCW721TokenOwner(ctx, contract, "1")
	require.NoError(t, err)
	require.Equal(t, owner.String(), got)

	// An unknown token has no owner recorded; the empty result must not be
	// treated as "owned by the module".
	got, err = k.QueryCW721TokenOwner(ctx, contract, "does-not-exist")
	require.NoError(t, err)
	require.Empty(t, got)
	require.False(t, moduleOwnsCW721(got))
}

func TestAppendToRefundGroups_MergesAndKeepsOrder(t *testing.T) {
	t.Parallel()

	groups := appendToRefundGroups(nil, "c1", "r1", "t1", "nft1")
	groups = appendToRefundGroups(groups, "c2", "r2", "t2", "nft2")
	groups = appendToRefundGroups(groups, "c1", "r1", "t3", "nft3")

	require.Len(t, groups, 2)
	require.Equal(t, []string{"t1", "t3"}, groups[0].tokenIds)
	require.Equal(t, []string{"nft1", "nft3"}, groups[0].nftIds)
	require.Equal(t, []string{"t2"}, groups[1].tokenIds)
}

func TestCW721RefundEvents_GroupsByContractAndReceiver(t *testing.T) {
	t.Parallel()

	groups := []refundGroup{
		{contract: "c1", receiver: "r1", tokenIds: []string{"t1"}, nftIds: []string{"nft1"}},
		{contract: "c2", receiver: "r2", tokenIds: []string{"t2"}, nftIds: []string{"nft2"}},
	}

	events := cw721RefundEvents("sender", "kitty", groups)
	require.Len(t, events, 2)
	require.Equal(t, cw721types.EventTypeRefundPacketToken, events[0].Type)
	require.Equal(t, "c1", refundAttr(events[0])[cw721types.AttributeKeyCW721Token])
	require.Equal(t, "r1", refundAttr(events[0])[cw721types.AttributeKeyReceiver])
	require.Equal(t, "t2", refundAttr(events[1])[cw721types.AttributeKeyCW721TokenID])
}

// TestRefundPacketTokenReturnsNoError is a static guard on the function body:
// no per-token failure inside RefundPacketToken may return an error.
//
// A returned error tears down the IBC callback, so the ack is never written,
// the relayer retries the packet forever, and the tokens after the failing one
// are abandoned. Round 22 found the last site (a native burn failure) still
// returning an error in both this module and its erc721 twin, so the invariant
// is checked mechanically now.
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
