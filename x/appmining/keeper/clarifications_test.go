package keeper_test

import (
	"crypto/sha256"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestAppIDDisplayCanonical(t *testing.T) {
	sum := sha256.Sum256([]byte("uge-sample-01"))
	require.Equal(t, 32, len(sum[:]))
	require.Equal(t, "IgcNFMVoZtLSg5Tt1BzpFMKhtyUL2ksjlpxVizGFW2g", types.AppIDDisplay(sum[:]))
}

func TestAppIDWrongLengthRejected(t *testing.T) {
	require.ErrorIs(t, types.ValidateAppID(make([]byte, 31)), types.ErrInvalidAppID)
	require.ErrorIs(t, types.ValidateAppID(make([]byte, 33)), types.ErrInvalidAppID)
	require.ErrorIs(t, types.ValidateAppID(nil), types.ErrInvalidAppID)
}

func TestAmountFormatRejected(t *testing.T) {
	for _, s := range []string{"007", "1.5", "-1", "+1", "1e3", ""} {
		require.Error(t, types.ValidateAmountString(s), s)
	}
}

func TestMultiOpenUpheldSupersedesOthers(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "10"),
	})
	require.NoError(t, err)

	challengers := []sdk.AccAddress{e.payer, sdk.AccAddress(bytes20(0x05)), sdk.AccAddress(bytes20(0x06))}
	var ids []uint64
	for i, c := range challengers {
		e.bank.fundAccount(c, types.DefaultChallengeBondDenom, math.NewInt(10_000_000))
		res, err := e.server.ChallengeAttestation(e.ctx, &types.MsgChallengeAttestation{
			Challenger: c.String(), AppId: appID, Epoch: 1, EvidenceHash: hash32(byte(0x40 + i)),
		})
		require.NoError(t, err)
		ids = append(ids, res.ChallengeId)
	}
	require.Len(t, ids, 3)

	_, err = e.server.ResolveChallenge(e.ctx, &types.MsgResolveChallenge{
		Authority: e.auth.String(), ChallengeId: ids[0], Uphold: true, ResolutionHash: hash32(0x70),
	})
	require.NoError(t, err)

	att, _ := e.k.GetAttestation(e.ctx, appID, 1)
	require.Equal(t, types.AttestationStatusRejected, att.Status)

	upheld, _ := e.k.GetChallenge(e.ctx, ids[0])
	require.Equal(t, types.ChallengeStatusUpheld, upheld.Status)
	for _, id := range ids[1:] {
		c, ok := e.k.GetChallenge(e.ctx, id)
		require.True(t, ok)
		require.Equal(t, types.ChallengeStatusSuperseded, c.Status)
	}
	require.False(t, e.k.HasOpenChallenge(e.ctx, appID, 1))
}

func TestPauseOnlyTrueAndResumeRequiresAuthority(t *testing.T) {
	e := setupTest(t)
	pauseAuth := sdk.AccAddress(bytes20(0x09))
	params := e.k.GetParams(e.ctx)
	params.PauseAuthority = pauseAuth.String()
	require.NoError(t, e.k.SetParams(e.ctx, params))

	_, err := e.server.Pause(e.ctx, &types.MsgPause{
		PauseAuthority: pauseAuth.String(),
		PauseBits:      types.PauseState{}, // all false
	})
	require.ErrorIs(t, err, types.ErrPauseOnlyTrue)

	_, err = e.server.Pause(e.ctx, &types.MsgPause{
		PauseAuthority: pauseAuth.String(),
		PauseBits:      types.PauseState{AttestationPaused: true},
	})
	require.NoError(t, err)
	require.True(t, e.k.GetParams(e.ctx).PauseState.AttestationPaused)

	// pause_authority cannot resume via UpdateParams
	cleared := e.k.GetParams(e.ctx)
	cleared.PauseState = &types.PauseState{}
	_, err = e.server.UpdateParams(e.ctx, &types.MsgUpdateParams{
		Authority: pauseAuth.String(),
		Params:    cleared,
	})
	require.ErrorIs(t, err, types.ErrResumeRequiresAuthority)

	// authority can resume
	_, err = e.server.UpdateParams(e.ctx, &types.MsgUpdateParams{
		Authority: e.auth.String(),
		Params:    cleared,
	})
	require.NoError(t, err)
	require.False(t, e.k.GetParams(e.ctx).PauseState.AttestationPaused)
}

func TestBudgetDenomNotAllowlisted(t *testing.T) {
	e := setupTest(t)
	_, err := e.server.SetEpochBudget(e.ctx, &types.MsgSetEpochBudget{
		Authority: e.auth.String(),
		Budget: types.EpochBudget{
			Epoch: 1, Denom: "not-listed", Limit: "10", Spent: "0",
		},
	})
	require.ErrorIs(t, err, types.ErrInvalidDenom)
}

func TestBudgetExceedsPoolRejected(t *testing.T) {
	e := setupTest(t)
	// pool was funded with 1e9; ask for more
	_, err := e.server.SetEpochBudget(e.ctx, &types.MsgSetEpochBudget{
		Authority: e.auth.String(),
		Budget: types.EpochBudget{
			Epoch: 1, Denom: types.DefaultChallengeBondDenom, Limit: "10000000000", Spent: "0",
		},
	})
	require.ErrorIs(t, err, types.ErrPoolInsufficient)
}

func TestRegisterEventCarriesAppIDDisplay(t *testing.T) {
	e := setupTest(t)
	sum := sha256.Sum256([]byte("uge-sample-01"))
	appID := sum[:]
	_, err := e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
		Authority: e.auth.String(), AppId: appID, Owner: e.owner.String(), AttestationSubmitter: e.sub.String(),
	})
	require.NoError(t, err)
	events := e.ctx.EventManager().Events()
	found := false
	for _, ev := range events {
		if ev.Type != types.EventTypeApplicationRegistered {
			continue
		}
		attrs := map[string]string{}
		for _, a := range ev.Attributes {
			attrs[a.Key] = a.Value
		}
		require.Equal(t, types.AppIDDisplay(appID), attrs[types.AttributeKeyAppIDDisplay])
		found = true
	}
	require.True(t, found)
}
