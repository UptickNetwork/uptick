package types

import (
	"encoding/hex"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesisState returns an empty genesis with default params.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:                DefaultParams(),
		Applications:          []Application{},
		EpochBudgets:          []EpochBudget{},
		Attestations:          []AttestationCommitment{},
		Challenges:            []Challenge{},
		Settlements:           []Settlement{},
		GasCreditAccounts:     []GasCreditAccount{},
		GasCreditConsumptions: []GasCreditConsumption{},
		NextChallengeId:       1,
		EntitlementPools:      []Pool{},
		RoleKeys:              []RoleKey{},
		Candidates:            []Candidate{},
		EntitlementChallenges: []EntitlementChallenge{},
		UsedAuthNonces:        []string{},
		Entitlements:          []Entitlement{},
	}
}

// ValidateGenesis validates genesis state.
func ValidateGenesis(gs GenesisState) error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	if gs.NextChallengeId == 0 {
		return fmt.Errorf("next_challenge_id must be >= 1")
	}

	appSeen := make(map[string]struct{}, len(gs.Applications))
	payerSeen := make(map[string]struct{})
	for _, app := range gs.Applications {
		if err := ValidateAppID(app.AppId); err != nil {
			return err
		}
		key := string(app.AppId)
		if _, ok := appSeen[key]; ok {
			return fmt.Errorf("duplicate application")
		}
		appSeen[key] = struct{}{}
		if err := ValidateAddress(app.Owner); err != nil {
			return fmt.Errorf("invalid owner: %w", err)
		}
		if err := ValidateAddress(app.AttestationSubmitter); err != nil {
			return fmt.Errorf("invalid attestation_submitter: %w", err)
		}
		for _, p := range app.GasPayers {
			if err := ValidateAddress(p); err != nil {
				return fmt.Errorf("invalid gas_payer: %w", err)
			}
			if _, ok := payerSeen[p]; ok {
				return fmt.Errorf("gas payer %s bound to multiple applications", p)
			}
			payerSeen[p] = struct{}{}
		}
		if app.Status == ApplicationStatusUnspecified {
			return fmt.Errorf("application status unspecified")
		}
	}

	epochSeen := make(map[uint64]struct{}, len(gs.EpochBudgets))
	for _, b := range gs.EpochBudgets {
		if _, ok := epochSeen[b.Epoch]; ok {
			return fmt.Errorf("duplicate epoch budget %d", b.Epoch)
		}
		epochSeen[b.Epoch] = struct{}{}
		if !DenomAllowed(gs.Params, b.Denom) {
			return fmt.Errorf("epoch %d denom %q not in reward_denom_allowlist", b.Epoch, b.Denom)
		}
		if err := ValidateAmountString(b.Limit); err != nil {
			return err
		}
		if err := ValidateAmountString(b.Spent); err != nil {
			return err
		}
		cmp, err := CompareAmountStrings(b.Spent, b.Limit)
		if err != nil {
			return err
		}
		if cmp > 0 {
			return fmt.Errorf("epoch %d spent > limit", b.Epoch)
		}
	}

	attSeen := make(map[string]struct{}, len(gs.Attestations))
	for _, a := range gs.Attestations {
		if err := validateAttestationFields(a); err != nil {
			return err
		}
		key := string(a.AppId) + "/" + fmt.Sprint(a.Epoch)
		if _, ok := attSeen[key]; ok {
			return fmt.Errorf("duplicate attestation")
		}
		attSeen[key] = struct{}{}
	}

	chalSeen := make(map[uint64]struct{}, len(gs.Challenges))
	for _, c := range gs.Challenges {
		if _, ok := chalSeen[c.ChallengeId]; ok {
			return fmt.Errorf("duplicate challenge id")
		}
		chalSeen[c.ChallengeId] = struct{}{}
		if c.ChallengeId >= gs.NextChallengeId {
			return fmt.Errorf("challenge id >= next_challenge_id")
		}
		if err := ValidateAppID(c.AppId); err != nil {
			return err
		}
		if err := ValidateHash(c.EvidenceHash); err != nil {
			return err
		}
		if err := ValidateAddress(c.Challenger); err != nil {
			return err
		}
		if err := ValidateAmountString(c.BondAmount); err != nil {
			return err
		}
	}

	for _, s := range gs.Settlements {
		if err := ValidateAppID(s.AppId); err != nil {
			return err
		}
		if err := ValidateHash(s.AttestationHash); err != nil {
			return err
		}
		if err := ValidateAmountString(s.RewardReleased); err != nil {
			return err
		}
		if err := ValidateAmountString(s.GasCreditIssued); err != nil {
			return err
		}
	}

	for _, a := range gs.GasCreditAccounts {
		if err := ValidateAppID(a.AppId); err != nil {
			return err
		}
		for _, field := range []string{a.Balance, a.TotalIssued, a.TotalConsumed} {
			if err := ValidateAmountString(field); err != nil {
				return err
			}
		}
	}

	consSeen := make(map[string]struct{}, len(gs.GasCreditConsumptions))
	for _, c := range gs.GasCreditConsumptions {
		if err := ValidateAppID(c.AppId); err != nil {
			return err
		}
		if len(c.SourceTxHash) != SourceTxHashLength {
			return ErrInvalidHash
		}
		key := string(c.AppId) + "/" + string(c.SourceTxHash)
		if _, ok := consSeen[key]; ok {
			return fmt.Errorf("duplicate gas credit consumption")
		}
		consSeen[key] = struct{}{}
		if err := ValidateAmountString(c.Amount); err != nil {
			return err
		}
		if err := ValidateAddress(c.GasPayer); err != nil {
			return err
		}
	}
	return validateV1Genesis(gs)
}

// validateV1Genesis validates the V1 entitlement flow records (workorder 13).
func validateV1Genesis(gs GenesisState) error {
	poolSeen := make(map[string]struct{}, len(gs.EntitlementPools))
	pools := make(map[string]Pool, len(gs.EntitlementPools))
	for _, p := range gs.EntitlementPools {
		if p.PoolId == "" {
			return fmt.Errorf("entitlement pool id empty")
		}
		if err := sdk.ValidateDenom(p.Denom); err != nil {
			return fmt.Errorf("entitlement pool %q denom: %w", p.PoolId, err)
		}
		if err := ValidatePoolInvariant(p); err != nil {
			return fmt.Errorf("entitlement pool %q: %w", p.PoolId, err)
		}
		if _, ok := poolSeen[p.PoolId]; ok {
			return fmt.Errorf("duplicate entitlement pool %q", p.PoolId)
		}
		poolSeen[p.PoolId] = struct{}{}
		pools[p.PoolId] = p
	}

	roleKeySeen := make(map[string]struct{}, len(gs.RoleKeys))
	pubkeyRoles := make(map[string]string) // pubkey hex -> "role/key_id" (first ACTIVE owner)
	for _, rk := range gs.RoleKeys {
		if rk.Role == EntitlementRoleUnspecified {
			return fmt.Errorf("role key %q role unspecified", rk.KeyId)
		}
		if rk.Status == RoleKeyStatusUnspecified {
			return fmt.Errorf("role key %q status unspecified", rk.KeyId)
		}
		if rk.KeyId == "" || len(rk.KeyId) > 128 {
			return fmt.Errorf("role key id %q invalid", rk.KeyId)
		}
		if err := ValidateEd25519PubKey(rk.Pubkey); err != nil {
			return fmt.Errorf("role key %q pubkey: %w", rk.KeyId, err)
		}
		key := fmt.Sprintf("%d/%s", rk.Role, rk.KeyId)
		if _, ok := roleKeySeen[key]; ok {
			return fmt.Errorf("duplicate role key %s", key)
		}
		roleKeySeen[key] = struct{}{}
		// F-2 (§1.5): one pubkey must never be ACTIVE under two roles.
		if rk.Status == RoleKeyStatusActive {
			pk := hex.EncodeToString(rk.Pubkey)
			if prev, ok := pubkeyRoles[pk]; ok {
				return fmt.Errorf("same pubkey active under two roles: %s and %d/%s", prev, rk.Role, rk.KeyId)
			}
			pubkeyRoles[pk] = fmt.Sprintf("%d/%s", rk.Role, rk.KeyId)
		}
	}

	candSeen := make(map[string]struct{}, len(gs.Candidates))
	for _, c := range gs.Candidates {
		if err := ValidateWireID(c.EntitlementId); err != nil || len(c.EntitlementId) != HashLength*2 {
			return fmt.Errorf("candidate entitlement_id %q invalid", c.EntitlementId)
		}
		if _, ok := candSeen[c.EntitlementId]; ok {
			return fmt.Errorf("duplicate candidate %q", c.EntitlementId)
		}
		candSeen[c.EntitlementId] = struct{}{}
		if err := ValidateWireHash(c.CandidateHash); err != nil {
			return fmt.Errorf("candidate %q candidate_hash: %w", c.EntitlementId, err)
		}
		if err := ValidateWireHash(c.EvidenceRoot); err != nil {
			return fmt.Errorf("candidate %q evidence_root: %w", c.EntitlementId, err)
		}
		pool, ok := pools[c.PoolId]
		if !ok {
			return fmt.Errorf("candidate %q references missing pool %q", c.EntitlementId, c.PoolId)
		}
		if c.Denom != pool.Denom {
			return fmt.Errorf("candidate %q denom %q != pool denom", c.EntitlementId, c.Denom)
		}
		if err := ValidateAmountString(c.Amount); err != nil {
			return fmt.Errorf("candidate %q amount: %w", c.EntitlementId, err)
		}
		if c.PublishedAt <= 0 {
			return fmt.Errorf("candidate %q published_at unset", c.EntitlementId)
		}
		if c.ChallengeDeadline != c.PublishedAt+ChallengeWindowSeconds {
			return fmt.Errorf("candidate %q challenge_deadline mismatch", c.EntitlementId)
		}
	}

	entSeen := make(map[string]struct{}, len(gs.Entitlements))
	for _, e := range gs.Entitlements {
		if err := ValidateWireID(e.EntitlementId); err != nil || len(e.EntitlementId) != HashLength*2 {
			return fmt.Errorf("entitlement %q id invalid", e.EntitlementId)
		}
		if _, ok := entSeen[e.EntitlementId]; ok {
			return fmt.Errorf("duplicate entitlement %q", e.EntitlementId)
		}
		entSeen[e.EntitlementId] = struct{}{}
		pool, ok := pools[e.PoolId]
		if !ok {
			return fmt.Errorf("entitlement %q references missing pool %q", e.EntitlementId, e.PoolId)
		}
		if e.Denom != pool.Denom {
			return fmt.Errorf("entitlement %q denom %q != pool denom", e.EntitlementId, e.Denom)
		}
		if err := ValidateAmountString(e.Amount); err != nil {
			return fmt.Errorf("entitlement %q amount: %w", e.EntitlementId, err)
		}
		if e.State == EntitlementStateUnspecified {
			return fmt.Errorf("entitlement %q state unspecified", e.EntitlementId)
		}
	}

	chalSeen := make(map[string]struct{}, len(gs.EntitlementChallenges))
	for _, c := range gs.EntitlementChallenges {
		if len(c.ChallengeId) != HashLength*2 {
			return fmt.Errorf("entitlement challenge %q id invalid", c.ChallengeId)
		}
		if _, ok := chalSeen[c.ChallengeId]; ok {
			return fmt.Errorf("duplicate entitlement challenge %q", c.ChallengeId)
		}
		chalSeen[c.ChallengeId] = struct{}{}
		if _, ok := candSeen[c.CandidateId]; !ok {
			return fmt.Errorf("entitlement challenge %q references unknown candidate %q", c.ChallengeId, c.CandidateId)
		}
		if err := ValidateAddress(c.Challenger); err != nil {
			return fmt.Errorf("entitlement challenge %q challenger: %w", c.ChallengeId, err)
		}
		if err := ValidateWireHash(c.EvidenceDigest); err != nil {
			return fmt.Errorf("entitlement challenge %q evidence_digest: %w", c.ChallengeId, err)
		}
		if c.Status == EntitlementChallengeStatusUnspecified {
			return fmt.Errorf("entitlement challenge %q status unspecified", c.ChallengeId)
		}
	}

	nonceSeen := make(map[string]struct{}, len(gs.UsedAuthNonces))
	for _, n := range gs.UsedAuthNonces {
		parts := strings.Split(n, "/")
		if len(parts) != 3 {
			return fmt.Errorf("used_auth_nonce %q malformed", n)
		}
		if _, ok := nonceSeen[n]; ok {
			return fmt.Errorf("duplicate used_auth_nonce %q", n)
		}
		nonceSeen[n] = struct{}{}
	}
	return nil
}

func validateAttestationFields(a AttestationCommitment) error {
	if err := ValidateAppID(a.AppId); err != nil {
		return err
	}
	if err := ValidateHash(a.AttestationHash); err != nil {
		return err
	}
	if err := ValidateHash(a.EvidenceRoot); err != nil {
		return err
	}
	if err := ValidateHash(a.ResultCommitment); err != nil {
		return err
	}
	if err := ValidateAmountString(a.RewardRequested); err != nil {
		return err
	}
	if err := ValidateAmountString(a.GasCreditRequested); err != nil {
		return err
	}
	if err := ValidateAddress(a.Submitter); err != nil {
		return err
	}
	if a.Profile == "" || a.ProfileVersion == "" {
		return ErrInvalidCommitment
	}
	return nil
}
