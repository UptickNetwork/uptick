package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// These helpers exist so the external test package can reproduce the state a
// chain carries when it is upgraded by a bare binary swap: challenge records
// written by a v1 binary have neither a candidate index entry nor a timeout
// bucket, and the rebuild state record does not exist. They are compiled only
// into tests.

// StripChallengeIndexesForTest removes every derived entry of one challenge.
func (k Keeper) StripChallengeIndexesForTest(ctx sdk.Context, c types.EntitlementChallenge) {
	store := k.store(ctx)
	store.Delete(types.EntitlementChallengeByCandidateKey(c.CandidateId, c.ChallengeId))
	k.deleteChallengeDeadlineBucket(ctx, c)
	store.Delete(types.OpenEntitlementChallengeKey(c.CandidateId, c.Challenger))
}

// ResetChallengeIndexStateForTest forgets the rebuild progress.
func (k Keeper) ResetChallengeIndexStateForTest(ctx sdk.Context) {
	k.store(ctx).Delete(types.ChallengeIndexStateKey())
}
