package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestValidateGenesis_V1CandidatesAndEntitlements(t *testing.T) {
	gs := types.DefaultGenesisState()
	gs.EntitlementPools = []types.Pool{{
		PoolId: "pool-a", Denom: types.DefaultChallengeBondDenom,
		Funded: "100", Available: "100", Reserved: "0", Paid: "0",
	}}

	hash64 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	appHex := "1111111111111111111111111111111111111111111111111111111111111111"
	challenger := sdk.AccAddress(make([]byte, 20)).String()

	gs.Candidates = []types.Candidate{{
		EntitlementId: hash64, AppId: appHex, Epoch: "1", Amount: "10",
		Denom: types.DefaultChallengeBondDenom, CandidateHash: hash64,
		EvidenceRoot: hash64, PoolId: "pool-a",
		PublishedAt: 100, ChallengeDeadline: 100 + types.ChallengeWindowSeconds,
		Status: types.CandidateStatusPublished,
	}}
	require.NoError(t, types.ValidateGenesis(*gs))

	bad := *gs
	bad.Candidates = append([]types.Candidate{}, gs.Candidates...)
	bad.Candidates[0].PoolId = "missing"
	require.Error(t, types.ValidateGenesis(bad))

	gs.Entitlements = []types.Entitlement{{
		EntitlementId: hash64, AppId: appHex, Amount: "10",
		Denom: types.DefaultChallengeBondDenom, PoolId: "pool-a",
		State: types.EntitlementStateClaimable,
	}}
	require.NoError(t, types.ValidateGenesis(*gs))

	gs.EntitlementChallenges = []types.EntitlementChallenge{{
		ChallengeId:    "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		CandidateId:    hash64,
		Challenger:     challenger,
		EvidenceDigest: hash64,
		Status:         types.EntitlementChallengeStatusOpen,
	}}
	require.NoError(t, types.ValidateGenesis(*gs))

	gs.EntitlementChallenges[0].Challenger = "not-a-bech32"
	require.Error(t, types.ValidateGenesis(*gs))
}

func TestValidateGenesis_DuplicateCandidate(t *testing.T) {
	gs := types.DefaultGenesisState()
	gs.EntitlementPools = []types.Pool{{
		PoolId: "pool-a", Denom: types.DefaultChallengeBondDenom,
		Funded: "1", Available: "1", Reserved: "0", Paid: "0",
	}}
	hash64 := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	appHex := "2222222222222222222222222222222222222222222222222222222222222222"
	c := types.Candidate{
		EntitlementId: hash64, AppId: appHex, Epoch: "1", Amount: "1",
		Denom: types.DefaultChallengeBondDenom, CandidateHash: hash64,
		EvidenceRoot: hash64, PoolId: "pool-a",
		PublishedAt: 1, ChallengeDeadline: 1 + types.ChallengeWindowSeconds,
		Status: types.CandidateStatusPublished,
	}
	gs.Candidates = []types.Candidate{c, c}
	require.Error(t, types.ValidateGenesis(*gs))
}
