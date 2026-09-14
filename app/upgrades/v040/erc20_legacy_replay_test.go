package v040

import (
	"crypto/sha256"
	"testing"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	erc20types "github.com/cosmos/evm/x/erc20/types"

	"github.com/UptickNetwork/uptick/app/upgrades"
)

// strv2AddressFor mirrors types.NewTokenPairSTRv2 -> utils.GetIBCDenomAddress:
// address = sha256(denom)[12:].
func strv2AddressFor(denom string) string {
	h := sha256.Sum256([]byte(denom))
	return common.BytesToAddress(h[12:]).Hex()
}

// TestReplayDoesNotDeleteStrv2Pairs documents a known defect and is the
// acceptance test for its fix. It is skipped until the fix lands.
//
// # DEFECT
//
// deleteLegacyOwnerModulePairs selects on ContractOwner == OWNER_MODULE alone.
// Under cosmos/evm the *new* IBC auto-registration path
// (x/erc20/keeper/ibc_callbacks.go:100 -> RegisterERC20Extension ->
// CreateNewTokenPair -> NewTokenPairSTRv2) registers pairs that are ALSO
// OWNER_MODULE. The two cannot be told apart by owner.
//
// On the first run this is harmless: a pre-upgrade chain only holds legacy
// pairs, and STRv2 pairs do not exist until new inbound ICS-20 packets arrive
// after the upgrade. The problem is replay. v040's handler
// (upgradeHandlerConstructor, upgrades.go:130-279) carries no
// UpgradeAlreadyApplied guard, so if the plan is re-submitted at a new height
// -- an operator mistake, or a crash-restart replay -- every STRv2 pair
// created in the meantime is deleted, and IBC assets that users are actively
// using lose their EVM adapter again.
//
// v041's handler comment used to claim "No UpgradeAlreadyApplied guard
// (unlike v040)", which read as though v040 had one. That half was wrong and
// has been corrected. The real distinction is reason, not presence: v041
// *cannot* use the guard, because it bumps no ConsensusVersion and the guard
// would report "applied" on its first legitimate run; v040 can, because a
// v0.3.3 version map already disagrees with the current one on several managed
// modules, so the guard reads false on the first pass and true on a replay.
// That is what makes fix A below available.
//
// The v0.3.3 side of that comparison is the live chain's stored map
// (/cosmos/upgrade/v1beta1/module_versions on rest.uptick.network); the
// current side is app.mm.GetVersionMap(). Differing managed modules:
// cw721 1->2, ibc 6->8, transfer 5->6, evm 5->1, feemarket 4->1.
//
// FIX (either is sufficient; both is safer)
//
//	A. add the UpgradeAlreadyApplied early-return to the v040 handler, and/or
//	B. skip pairs whose Erc20Address equals the STRv2 derivation of their own
//	   denom (types.NewTokenPairSTRv2 / utils.GetIBCDenomAddress), so the
//	   predicate can no longer match post-upgrade registrations.
//
// To enable: delete the t.Skip line. The test then fails while the defect is
// present and passes once fixed.
func TestReplayDoesNotDeleteStrv2Pairs(t *testing.T) {
	t.Skip("known defect: v040 replay deletes post-upgrade STRv2 pairs (no idempotency guard; owner-only predicate)")

	k, ctx, key := newErc20TestKeeper(t)
	box := upgrades.Toolbox{}
	box.Erc20Keeper = k

	// First run: legacy pairs from mainnet get dropped. This part is correct.
	for _, f := range mainnetLegacyPairs {
		seedLegacyPair(t, ctx, key, f, uint64(erc20types.OWNER_MODULE))
	}
	require.NoError(t, deleteLegacyOwnerModulePairs(ctx, box, log.NewNopLogger()))
	require.Empty(t, k.GetTokenPairs(ctx), "first run drops all legacy pairs")

	// The chain runs on. Inbound ICS-20 packets auto-register STRv2 pairs.
	strv2Denoms := []string{
		"ibc/9BD618B22904F348824AB73154F134E59FCA43EC002D64428247675EBBE74C64",
		"ibc/1AECCF04FB21E3B376E131F0027B29E3C1AA1957BECC039ACFCA761863E2CF77",
	}
	for _, d := range strv2Denoms {
		seedLegacyPair(t, ctx, key, legacyPairFixture{denom: d, erc20: strv2AddressFor(d)}, uint64(erc20types.OWNER_MODULE))
	}
	require.Len(t, k.GetTokenPairs(ctx), 2, "STRv2 pairs are seeded and live")

	// The plan is replayed. These pairs are newer than the migration and must
	// survive.
	require.NoError(t, deleteLegacyOwnerModulePairs(ctx, box, log.NewNopLogger()))
	require.Len(t, k.GetTokenPairs(ctx), 2,
		"replay must not touch STRv2 pairs registered after the upgrade")
}
