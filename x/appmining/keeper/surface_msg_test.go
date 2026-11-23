package keeper_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

// This file closes the message-surface gaps the behavioural vector tests leave
// open: every Msg handler is exercised on the branches that decide whether a
// transaction may run — the authority gate, the pause gate, the not-found gate,
// the owner gate and the chain/pool/denom gate. Those branches were the bulk of
// the uncovered statements in the keeper package.
//
// A handler callable by the wrong signer, or while its pause bit is set, is a
// worse defect than a mis-computed amount, so each gate gets its own assertion
// rather than being implied by a happy path.

func strangerAddr() string {
	return sdk.AccAddress(bytes20(0x7e)).String()
}

func digestOf(seed byte) string {
	return hex.EncodeToString(hash32(seed))
}

// TestSurface_AuthorityGates asserts every authority-gated message rejects a
// signer that is not the params authority before touching any state.
func TestSurface_AuthorityGates(t *testing.T) {
	e := setupTest(t)
	other := strangerAddr()

	pub := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x31}, ed25519.SeedSize))
	e.k.SetRoleKey(e.ctx, types.RoleKey{
		Role: types.EntitlementRolePublisher, KeyId: "pub-1",
		Pubkey: pub.Public().(ed25519.PublicKey), Status: types.RoleKeyStatusActive,
	})

	appID := hash32(0xAA)
	e.k.SetApplication(e.ctx, types.Application{
		AppId: appID, Owner: e.owner.String(), AttestationSubmitter: e.sub.String(),
		Status: types.ApplicationStatusActive,
	})

	cases := []struct {
		name string
		call func() error
	}{
		{"RegisterApplication", func() error {
			_, err := e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
				Authority: other, AppId: hash32(0x01), Owner: e.owner.String(), AttestationSubmitter: e.sub.String(),
			})
			return err
		}},
		{"SetApplicationStatus", func() error {
			_, err := e.server.SetApplicationStatus(e.ctx, &types.MsgSetApplicationStatus{
				Authority: other, AppId: appID, Status: types.ApplicationStatusSuspended,
			})
			return err
		}},
		{"ResolveChallenge", func() error {
			_, err := e.server.ResolveChallenge(e.ctx, &types.MsgResolveChallenge{
				Authority: other, ChallengeId: 1, Uphold: true, ResolutionHash: hash32(0x02),
			})
			return err
		}},
		{"SetEpochBudget", func() error {
			_, err := e.server.SetEpochBudget(e.ctx, &types.MsgSetEpochBudget{
				Authority: other,
				Budget: types.EpochBudget{
					Epoch: 1, Denom: types.DefaultChallengeBondDenom, Limit: "1000", Spent: "0",
				},
			})
			return err
		}},
		{"Pause", func() error {
			_, err := e.server.Pause(e.ctx, &types.MsgPause{
				PauseAuthority: other,
				PauseBits:      types.PauseState{AttestationPaused: true},
			})
			return err
		}},
		{"OpenEntitlementPool", func() error {
			_, err := e.server.OpenEntitlementPool(e.ctx, &types.MsgOpenEntitlementPool{
				Authority: other, PoolId: "p1", Denom: types.DefaultChallengeBondDenom, Version: 1,
			})
			return err
		}},
		{"FundEntitlementPool", func() error {
			_, err := e.server.FundEntitlementPool(e.ctx, &types.MsgFundEntitlementPool{
				Authority: other, PoolId: "p1", Amount: "10",
			})
			return err
		}},
		{"SetEntitlementPoolPaused", func() error {
			_, err := e.server.SetEntitlementPoolPaused(e.ctx, &types.MsgSetEntitlementPoolPaused{
				Authority: other, PoolId: "p1", Paused: true,
			})
			return err
		}},
		{"RegisterRoleKey", func() error {
			_, err := e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
				Authority: other, Role: types.EntitlementRoleApprover, KeyId: "apr-x",
				Pubkey: pub.Public().(ed25519.PublicKey), Version: 1,
			})
			return err
		}},
		{"SetRoleKeyStatus", func() error {
			_, err := e.server.SetRoleKeyStatus(e.ctx, &types.MsgSetRoleKeyStatus{
				Authority: other, Role: types.EntitlementRolePublisher, KeyId: "pub-1",
				Status: types.RoleKeyStatusPaused,
			})
			return err
		}},
		{"RegisterAssessmentSigner", func() error {
			_, err := e.server.RegisterAssessmentSigner(e.ctx, &types.MsgRegisterAssessmentSigner{
				Authority: other, KeyId: "asm-x", Pubkey: pub.Public().(ed25519.PublicKey), Version: 1,
			})
			return err
		}},
		{"SetAssessmentSignerStatus", func() error {
			_, err := e.server.SetAssessmentSignerStatus(e.ctx, &types.MsgSetAssessmentSignerStatus{
				Authority: other, KeyId: "asm-x", Status: types.RoleKeyStatusPaused,
			})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err)
			require.ErrorIs(t, err, types.ErrUnauthorized)
		})
	}
}

// TestSurface_UpdateParamsAuthority separates the two authorities: pause_authority
// may pause but must not resume, and a third party may not write params at all.
func TestSurface_UpdateParamsAuthority(t *testing.T) {
	e := setupTest(t)
	pauseAuth := sdk.AccAddress(bytes20(0x5a))
	params := e.k.GetParams(e.ctx)
	params.PauseAuthority = pauseAuth.String()
	require.NoError(t, e.k.SetParams(e.ctx, params))

	_, err := e.server.UpdateParams(e.ctx, &types.MsgUpdateParams{
		Authority: pauseAuth.String(), Params: e.k.GetParams(e.ctx),
	})
	require.ErrorIs(t, err, types.ErrResumeRequiresAuthority)

	_, err = e.server.UpdateParams(e.ctx, &types.MsgUpdateParams{
		Authority: strangerAddr(), Params: e.k.GetParams(e.ctx),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")

	bad := e.k.GetParams(e.ctx)
	bad.ChallengeBondAmount = "-1"
	_, err = e.server.UpdateParams(e.ctx, &types.MsgUpdateParams{Authority: e.auth.String(), Params: bad})
	require.Error(t, err)

	good := e.k.GetParams(e.ctx)
	good.MaxGasPayers = 32
	_, err = e.server.UpdateParams(e.ctx, &types.MsgUpdateParams{Authority: e.auth.String(), Params: good})
	require.NoError(t, err)
	require.Equal(t, good.MaxGasPayers, e.k.GetParams(e.ctx).MaxGasPayers)
}

func setPauseBits(t *testing.T, e testEnv, bits *types.PauseState) {
	t.Helper()
	params := e.k.GetParams(e.ctx)
	params.PauseState = bits
	require.NoError(t, e.k.SetParams(e.ctx, params))
}

// TestSurface_PauseGates drives each legacy pause bit plus the V1 entitlement
// pause bit through the message that must be blocked by it.
func TestSurface_PauseGates(t *testing.T) {
	t.Run("registration", func(t *testing.T) {
		e := setupTest(t)
		setPauseBits(t, e, &types.PauseState{RegistrationPaused: true})
		_, err := e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
			Authority: e.auth.String(), AppId: hash32(0x01), Owner: e.owner.String(), AttestationSubmitter: e.sub.String(),
		})
		require.ErrorIs(t, err, types.ErrRegistrationPaused)
	})

	t.Run("attestation", func(t *testing.T) {
		e := setupTest(t)
		setPauseBits(t, e, &types.PauseState{AttestationPaused: true})
		appID := registerApp(t, e)
		_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
			Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "10"),
		})
		require.ErrorIs(t, err, types.ErrAttestationPaused)
	})

	t.Run("challenge", func(t *testing.T) {
		e := setupTest(t)
		setPauseBits(t, e, &types.PauseState{ChallengePaused: true})
		_, err := e.server.ChallengeAttestation(e.ctx, &types.MsgChallengeAttestation{
			Challenger: e.owner.String(), AppId: hash32(0xAA), Epoch: 1, EvidenceHash: hash32(0x09),
		})
		require.ErrorIs(t, err, types.ErrChallengePaused)
	})

	t.Run("settlement", func(t *testing.T) {
		e := setupTest(t)
		setPauseBits(t, e, &types.PauseState{SettlementPaused: true})
		_, err := e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
			Caller: e.auth.String(), AppId: hash32(0xAA), Epoch: 1,
		})
		require.ErrorIs(t, err, types.ErrSettlementPaused)
	})

	t.Run("gas credit", func(t *testing.T) {
		e := setupTest(t)
		enableManualConsume(t, e)
		setPauseBits(t, e, &types.PauseState{GasCreditPaused: true})
		_, err := e.server.ConsumeGasCredit(e.ctx, &types.MsgConsumeGasCredit{
			AppId: hash32(0xAA), GasPayer: e.payer.String(), Amount: "1", SourceTxHash: hash32(0x0a),
		})
		require.ErrorIs(t, err, types.ErrGasCreditPaused)
	})

	t.Run("entitlement fund", func(t *testing.T) {
		e, _, _ := setupP1(t)
		setPauseBits(t, e, &types.PauseState{EntitlementPaused: true})
		_, err := e.server.FundEntitlementPool(e.ctx, &types.MsgFundEntitlementPool{
			Authority: e.auth.String(), PoolId: testPoolID, Amount: "10",
		})
		require.ErrorIs(t, err, types.ErrEntitlementPaused)
	})

	t.Run("entitlement publish", func(t *testing.T) {
		e, pubPriv, _ := setupP1(t)
		setPauseBits(t, e, &types.PauseState{EntitlementPaused: true})
		c := sampleCandidateFields()
		_, err := e.server.PublishCandidate(e.ctx, publishMsgFor(t, e, pubPriv, c, "7001", wireExp(e.ctx), false))
		require.ErrorIs(t, err, types.ErrEntitlementPaused)
	})

	t.Run("entitlement approve", func(t *testing.T) {
		e, _, _, aprPriv, c := setupP2(t)
		cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
		e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.DefaultChallengeWindowSeconds, 0))
		setPauseBits(t, e, &types.PauseState{EntitlementPaused: true})
		_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "7002"))
		require.ErrorIs(t, err, types.ErrEntitlementPaused)
	})

	t.Run("entitlement claim", func(t *testing.T) {
		e, _, _, _, c := setupP3(t)
		setPauseBits(t, e, &types.PauseState{EntitlementPaused: true})
		_, err := e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
			Owner: e.owner.String(), EntitlementId: c.EntitlementID,
		})
		require.ErrorIs(t, err, types.ErrEntitlementPaused)
	})
}

// TestSurface_LegacyNotFound exercises the "record absent" branch of every
// legacy message so a wrong id fails with the documented code.
func TestSurface_LegacyNotFound(t *testing.T) {
	e := setupTest(t)
	unknown := hash32(0x99)
	noAttApp := hash32(0xAC)
	e.k.SetApplication(e.ctx, types.Application{
		AppId: noAttApp, Owner: e.owner.String(), AttestationSubmitter: e.sub.String(),
		Status: types.ApplicationStatusActive,
	})
	e.bank.fundAccount(e.owner, types.DefaultChallengeBondDenom, math.NewInt(1_000_000_000))

	cases := []struct {
		name string
		call func() error
		want error
	}{
		{"SetApplicationStatus", func() error {
			_, err := e.server.SetApplicationStatus(e.ctx, &types.MsgSetApplicationStatus{
				Authority: e.auth.String(), AppId: unknown, Status: types.ApplicationStatusSuspended,
			})
			return err
		}, types.ErrApplicationNotFound},
		{"SubmitAttestation", func() error {
			_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
				Submitter: e.sub.String(), Attestation: sampleAttestation(unknown, 1, "10"),
			})
			return err
		}, types.ErrApplicationNotFound},
		{"ChallengeAttestation", func() error {
			_, err := e.server.ChallengeAttestation(e.ctx, &types.MsgChallengeAttestation{
				Challenger: e.owner.String(), AppId: noAttApp, Epoch: 7, EvidenceHash: hash32(0x09),
			})
			return err
		}, types.ErrAttestationNotFound},
		{"ResolveChallenge", func() error {
			_, err := e.server.ResolveChallenge(e.ctx, &types.MsgResolveChallenge{
				Authority: e.auth.String(), ChallengeId: 424242, Uphold: false, ResolutionHash: hash32(0x03),
			})
			return err
		}, types.ErrChallengeNotFound},
		{"FinalizeUnknownApp", func() error {
			_, err := e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
				Caller: e.auth.String(), AppId: unknown, Epoch: 1,
			})
			return err
		}, types.ErrApplicationNotFound},
		{"FinalizeUnknownAttestation", func() error {
			_, err := e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
				Caller: e.auth.String(), AppId: noAttApp, Epoch: 9,
			})
			return err
		}, types.ErrAttestationNotFound},
		{"GrantGasPayer", func() error {
			_, err := e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
				Owner: e.owner.String(), AppId: unknown, GasPayer: e.payer.String(),
			})
			return err
		}, types.ErrApplicationNotFound},
		{"SetAttestationSubmitter", func() error {
			_, err := e.server.SetAttestationSubmitter(e.ctx, &types.MsgSetAttestationSubmitter{
				Owner: e.owner.String(), AppId: unknown, AttestationSubmitter: e.payer.String(),
			})
			return err
		}, types.ErrApplicationNotFound},
		{"RevokeGasPayer", func() error {
			_, err := e.server.RevokeGasPayer(e.ctx, &types.MsgRevokeGasPayer{
				Owner: e.owner.String(), AppId: unknown, GasPayer: e.payer.String(),
			})
			return err
		}, types.ErrApplicationNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorIs(t, tc.call(), tc.want)
		})
	}
}

// TestSurface_OwnerGates covers the owner-only messages.
func TestSurface_OwnerGates(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	other := strangerAddr()

	_, err := e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
		Owner: other, AppId: appID, GasPayer: e.payer.String(),
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)

	_, err = e.server.RevokeGasPayer(e.ctx, &types.MsgRevokeGasPayer{
		Owner: other, AppId: appID, GasPayer: e.payer.String(),
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)

	_, err = e.server.SetAttestationSubmitter(e.ctx, &types.MsgSetAttestationSubmitter{
		Owner: other, AppId: appID, AttestationSubmitter: e.payer.String(),
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)

	// The owner can grant twice; the duplicate is a no-op, not a second entry.
	for i := 0; i < 2; i++ {
		_, err = e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
			Owner: e.owner.String(), AppId: appID, GasPayer: e.payer.String(),
		})
		require.NoError(t, err)
	}
	app, found := e.k.GetApplication(e.ctx, appID)
	require.True(t, found)
	require.Equal(t, []string{e.payer.String()}, app.GasPayers)

	// Revoking a payer that was never granted leaves the list unchanged.
	_, err = e.server.RevokeGasPayer(e.ctx, &types.MsgRevokeGasPayer{
		Owner: e.owner.String(), AppId: appID, GasPayer: strangerAddr(),
	})
	require.NoError(t, err)
	app, _ = e.k.GetApplication(e.ctx, appID)
	require.Equal(t, []string{e.payer.String()}, app.GasPayers)

	// Rotation is recorded and visible.
	_, err = e.server.SetAttestationSubmitter(e.ctx, &types.MsgSetAttestationSubmitter{
		Owner: e.owner.String(), AppId: appID, AttestationSubmitter: e.payer.String(),
	})
	require.NoError(t, err)
	app, _ = e.k.GetApplication(e.ctx, appID)
	require.Equal(t, e.payer.String(), app.AttestationSubmitter)
}

// TestSurface_RegisterApplicationGuards covers the duplicate and status-machine
// branches of the registration/status pair.
func TestSurface_RegisterApplicationGuards(t *testing.T) {
	e := setupTest(t)
	appID := hash32(0xAA)

	reg := func() error {
		_, err := e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
			Authority: e.auth.String(), AppId: appID, Owner: e.owner.String(), AttestationSubmitter: e.sub.String(),
		})
		return err
	}
	require.NoError(t, reg())
	require.ErrorIs(t, reg(), types.ErrApplicationExists)

	setStatus := func(status types.ApplicationStatus) error {
		_, err := e.server.SetApplicationStatus(e.ctx, &types.MsgSetApplicationStatus{
			Authority: e.auth.String(), AppId: appID, Status: status,
		})
		return err
	}
	require.NoError(t, setStatus(types.ApplicationStatusSuspended))
	require.NoError(t, setStatus(types.ApplicationStatusActive))
	require.NoError(t, setStatus(types.ApplicationStatusRevoked))
	// Revoked is terminal for the status machine.
	require.ErrorIs(t, setStatus(types.ApplicationStatusActive), types.ErrApplicationRevoked)
	require.ErrorIs(t, setStatus(types.ApplicationStatusUnspecified), types.ErrInvalidStatusTransition)
}

// TestSurface_LegacyLifecycleBranches walks the attestation lifecycle across the
// boundaries the vector tests do not touch.
func TestSurface_LegacyLifecycleBranches(t *testing.T) {
	t.Run("finalize without budget", func(t *testing.T) {
		e := setupTest(t)
		appID := registerApp(t, e)
		// Attestation on disk with no matching epoch budget: the only way the
		// ErrBudgetUnavailable branch can be reached.
		att := sampleAttestation(appID, 5, "100")
		att.Status = types.AttestationStatusChallengeWindow
		att.ChallengeDeadlineHeight = 1
		e.k.SetAttestation(e.ctx, att)
		e.ctx = e.ctx.WithBlockHeight(types.DefaultChallengeWindowBlocks + 10)
		_, err := e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
			Caller: e.auth.String(), AppId: appID, Epoch: 5,
		})
		require.ErrorIs(t, err, types.ErrBudgetUnavailable)
	})

	t.Run("challenge after deadline", func(t *testing.T) {
		e := setupTest(t)
		appID := registerApp(t, e)
		setBudget(t, e, 1, "1000")
		_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
			Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "100"),
		})
		require.NoError(t, err)
		e.bank.fundAccount(e.owner, types.DefaultChallengeBondDenom, math.NewInt(1_000_000_000))
		e.ctx = e.ctx.WithBlockHeight(types.DefaultChallengeWindowBlocks + 5)
		_, err = e.server.ChallengeAttestation(e.ctx, &types.MsgChallengeAttestation{
			Challenger: e.owner.String(), AppId: appID, Epoch: 1, EvidenceHash: hash32(0x09),
		})
		require.Error(t, err)
		require.False(t, e.k.HasOpenChallenge(e.ctx, appID, 1))
	})

	t.Run("reward pool insufficient", func(t *testing.T) {
		e := setupTest(t)
		appID := registerApp(t, e)
		// Budget written straight to the store so the open-commitment check that
		// guards SetEpochBudget cannot reject it: the ledger says the reward is
		// affordable, the module account says otherwise.
		e.k.StoreEpochBudget(e.ctx, types.EpochBudget{
			Epoch: 1, Denom: types.DefaultChallengeBondDenom, Limit: "5000000000", Spent: "0", OpenedHeight: 1,
		})
		_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
			Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "4000000000"),
		})
		require.NoError(t, err)
		e.ctx = e.ctx.WithBlockHeight(types.DefaultChallengeWindowBlocks + 10)
		_, err = e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
			Caller: e.auth.String(), AppId: appID, Epoch: 1,
		})
		require.ErrorIs(t, err, types.ErrPoolInsufficient)
	})
}

// TestSurface_BudgetBranches covers SetEpochBudget accumulation/close logic and
// the FundRewardPool validation gates.
func TestSurface_BudgetBranches(t *testing.T) {
	e := setupTest(t)

	// Spent defaults to "0" when omitted.
	_, err := e.server.SetEpochBudget(e.ctx, &types.MsgSetEpochBudget{
		Authority: e.auth.String(),
		Budget:    types.EpochBudget{Epoch: 3, Denom: types.DefaultChallengeBondDenom, Limit: "1000"},
	})
	require.NoError(t, err)
	b, found := e.k.GetEpochBudget(e.ctx, 3)
	require.True(t, found)
	require.Equal(t, "0", b.Spent)
	require.Equal(t, e.ctx.BlockHeight(), b.OpenedHeight)

	// Re-setting keeps the recorded spend and pins the opening height.
	b.Spent = "100"
	e.k.StoreEpochBudget(e.ctx, b)
	e.ctx = e.ctx.WithBlockHeight(50)
	_, err = e.server.SetEpochBudget(e.ctx, &types.MsgSetEpochBudget{
		Authority: e.auth.String(),
		Budget:    types.EpochBudget{Epoch: 3, Denom: types.DefaultChallengeBondDenom, Limit: "2000", Spent: "0"},
	})
	require.NoError(t, err)
	b, _ = e.k.GetEpochBudget(e.ctx, 3)
	require.Equal(t, "100", b.Spent, "spent must never be reset by an update")
	require.Equal(t, int64(1), b.OpenedHeight)

	// Closing stamps the height.
	_, err = e.server.SetEpochBudget(e.ctx, &types.MsgSetEpochBudget{
		Authority: e.auth.String(),
		Budget:    types.EpochBudget{Epoch: 3, Denom: types.DefaultChallengeBondDenom, Limit: "2000", Closed: true},
	})
	require.NoError(t, err)
	b, _ = e.k.GetEpochBudget(e.ctx, 3)
	require.True(t, b.Closed)
	require.Equal(t, int64(50), b.ClosedHeight)

	// A limit below the recorded spend is refused.
	_, err = e.server.SetEpochBudget(e.ctx, &types.MsgSetEpochBudget{
		Authority: e.auth.String(),
		Budget:    types.EpochBudget{Epoch: 3, Denom: types.DefaultChallengeBondDenom, Limit: "50"},
	})
	require.ErrorIs(t, err, types.ErrBudgetExceeded)

	// FundRewardPool: amount must be positive and the denom allowlisted.
	_, err = e.server.FundRewardPool(e.ctx, &types.MsgFundRewardPool{
		Funder: e.owner.String(), Denom: types.DefaultChallengeBondDenom, Amount: "0",
	})
	require.Error(t, err)
	_, err = e.server.FundRewardPool(e.ctx, &types.MsgFundRewardPool{
		Funder: e.owner.String(), Denom: "not-in-allowlist", Amount: "10",
	})
	require.ErrorIs(t, err, types.ErrInvalidDenom)
	_, err = e.server.FundRewardPool(e.ctx, &types.MsgFundRewardPool{
		Funder: "not-an-address", Denom: types.DefaultChallengeBondDenom, Amount: "10",
	})
	require.Error(t, err)
}

// TestSurface_V1Gates covers the V1 guards that sit before any state change.
func TestSurface_V1Gates(t *testing.T) {
	t.Run("publish chain id", func(t *testing.T) {
		e, pubPriv, _ := setupP1(t)
		c := sampleCandidateFields()
		msg := publishMsgFor(t, e, pubPriv, c, "7101", wireExp(e.ctx), false)
		msg.ChainId = "other_1-1"
		_, err := e.server.PublishCandidate(e.ctx, msg)
		require.ErrorIs(t, err, types.ErrChainIDMismatch)
	})

	t.Run("publish denom", func(t *testing.T) {
		e, pubPriv, _ := setupP1(t)
		c := sampleCandidateFields()
		msg := publishMsgFor(t, e, pubPriv, c, "7102", wireExp(e.ctx), false)
		msg.Denom = "uosmo"
		_, err := e.server.PublishCandidate(e.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidDenom)
	})

	t.Run("publish unknown app", func(t *testing.T) {
		e, pubPriv, _ := setupP1(t)
		c := sampleCandidateFields()
		c.AppID = hex.EncodeToString(hash32(0x77))
		c.EntitlementID = wirev1.EntitlementIDOf(c)
		msg := publishMsgFor(t, e, pubPriv, c, "7103", wireExp(e.ctx), false)
		_, err := e.server.PublishCandidate(e.ctx, msg)
		require.ErrorIs(t, err, types.ErrApplicationNotFound)
	})

	t.Run("publish suspended app", func(t *testing.T) {
		e, pubPriv, _ := setupP1(t)
		appID, _ := hex.DecodeString(sampleCandidateFields().AppID)
		_, err := e.server.SetApplicationStatus(e.ctx, &types.MsgSetApplicationStatus{
			Authority: e.auth.String(), AppId: appID, Status: types.ApplicationStatusSuspended,
		})
		require.NoError(t, err)
		c := sampleCandidateFields()
		_, err = e.server.PublishCandidate(e.ctx, publishMsgFor(t, e, pubPriv, c, "7104", wireExp(e.ctx), false))
		require.ErrorIs(t, err, types.ErrApplicationNotActive)
	})

	t.Run("publish identity mismatch", func(t *testing.T) {
		e, _, _ := setupP1(t)
		c := sampleCandidateFields()
		// The keeper recomputes the candidate hash from the message body and
		// compares it with the claimed entitlement id; a client that patches
		// one field without re-deriving the id is rejected before any wire
		// signature is even looked at.
		_, err := e.server.PublishCandidate(e.ctx, &types.MsgPublishCandidate{
			Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
			EntitlementId: c.EntitlementID, AppId: c.AppID, Epoch: c.Epoch,
			Amount: "999", Denom: c.Denom, GasUsed: c.GasUsed, QualifiedTxCount: c.QualifiedTxCount,
			PavId: c.PavID, EvidenceRoot: c.EvidenceRoot, GasEvidenceRoot: c.GasEvidenceRoot,
			RuleVersion: c.RuleVersion,
		})
		require.ErrorIs(t, err, types.ErrInvalidCommitment)
	})

	t.Run("publish nonce replay", func(t *testing.T) {
		e, pubPriv, _ := setupP1(t)
		c1 := sampleCandidateFields()
		_, err := e.server.PublishCandidate(e.ctx, publishMsgFor(t, e, pubPriv, c1, "7106", wireExp(e.ctx), false))
		require.NoError(t, err)

		c2 := sampleCandidateFields()
		c2.Amount = "999"
		c2.EntitlementID = wirev1.EntitlementIDOf(c2)
		_, err = e.server.PublishCandidate(e.ctx, publishMsgFor(t, e, pubPriv, c2, "7106", wireExp(e.ctx), false))
		require.ErrorIs(t, err, types.ErrAuthReplay)
	})

	t.Run("challenge access none", func(t *testing.T) {
		e, _, _, _, c := setupP2(t)
		params := e.k.GetParams(e.ctx)
		params.ChallengeOpenAccess = types.ChallengeOpenAccessNone
		require.NoError(t, e.k.SetParams(e.ctx, params))
		_, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
			Challenger: e.owner.String(), EntitlementId: c.EntitlementID,
			EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: digestOf(0x0b),
		})
		require.ErrorIs(t, err, types.ErrUnauthorized)
	})

	t.Run("challenge deposit denom mismatch", func(t *testing.T) {
		e, _, _, _, c := setupP2(t)
		params := e.k.GetParams(e.ctx)
		// The denom has to pass governance validation first (it must be
		// allowlisted), so the mismatch is between two *legal* denoms.
		params.RewardDenomAllowlist = append(params.RewardDenomAllowlist, "uosmo")
		params.ChallengeOpenAccess = types.ChallengeOpenAccessDeposit
		params.ChallengeDepositAmount = "10"
		params.ChallengeDepositDenom = "uosmo"
		require.NoError(t, e.k.SetParams(e.ctx, params))
		_, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
			Challenger: e.owner.String(), EntitlementId: c.EntitlementID,
			EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: digestOf(0x0b),
		})
		require.ErrorIs(t, err, types.ErrInvalidDenom)
	})

	t.Run("challenge deposit amount unset", func(t *testing.T) {
		e, _, _, _, _ := setupP2(t)
		params := e.k.GetParams(e.ctx)
		params.ChallengeOpenAccess = types.ChallengeOpenAccessDeposit
		params.ChallengeDepositAmount = ""
		params.ChallengeDepositDenom = types.DefaultChallengeBondDenom
		// Governance validation refuses the combination, so the handler's own
		// ErrInvalidAmount guard is defence in depth rather than a reachable
		// path: params can never enter the store in this shape.
		require.Error(t, e.k.SetParams(e.ctx, params))
	})

	t.Run("challenge closed window", func(t *testing.T) {
		e, _, _, _, c := setupP2(t)
		cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
		e.ctx = e.ctx.WithBlockTime(time.Unix(cand.ChallengeDeadline+1, 0))
		_, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
			Challenger: e.owner.String(), EntitlementId: c.EntitlementID,
			EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: digestOf(0x0b),
		})
		require.ErrorIs(t, err, types.ErrChallengeWindowClosed)
	})

	t.Run("challenge duplicate", func(t *testing.T) {
		e, _, _, _, c := setupP2(t)
		msg := &types.MsgOpenChallenge{
			Challenger: e.owner.String(), EntitlementId: c.EntitlementID,
			EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: digestOf(0x0b),
		}
		_, err := e.server.OpenChallenge(e.ctx, msg)
		require.NoError(t, err)
		_, err = e.server.OpenChallenge(e.ctx, msg)
		require.ErrorIs(t, err, types.ErrDuplicateOpenChallenge)
	})

	t.Run("challenge non-published candidate", func(t *testing.T) {
		e, _, _, _, c := setupP2(t)
		cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
		cand.Status = types.CandidateStatusRejected
		e.k.SetCandidate(e.ctx, cand)
		_, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
			Challenger: e.owner.String(), EntitlementId: c.EntitlementID,
			EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: digestOf(0x0b),
		})
		require.ErrorIs(t, err, types.ErrCandidateNotPublished)
	})

	t.Run("approve chain id", func(t *testing.T) {
		e, _, _, aprPriv, c := setupP2(t)
		cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
		e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.DefaultChallengeWindowSeconds, 0))
		msg := approveMsg(t, e, aprPriv, c, "7201")
		msg.ChainId = "other_1-1"
		_, err := e.server.ApproveEntitlement(e.ctx, msg)
		require.ErrorIs(t, err, types.ErrChainIDMismatch)
	})

	t.Run("approve already approved", func(t *testing.T) {
		e, _, _, aprPriv, c := setupP3(t)
		_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "7202"))
		require.ErrorIs(t, err, types.ErrEntitlementExists)
	})

	t.Run("approve pool mismatch", func(t *testing.T) {
		e, _, _, aprPriv, c := setupP2(t)
		cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
		e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.DefaultChallengeWindowSeconds, 0))
		msg := approveMsg(t, e, aprPriv, c, "7203")
		msg.PoolId = "other-pool"
		_, err := e.server.ApproveEntitlement(e.ctx, msg)
		require.ErrorIs(t, err, types.ErrPoolMismatch)
	})

	t.Run("claim not owner and double claim", func(t *testing.T) {
		e, _, _, _, c := setupP3(t)
		_, err := e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
			Owner: strangerAddr(), EntitlementId: c.EntitlementID,
		})
		require.ErrorIs(t, err, types.ErrBeneficiaryMustBeOwner)

		_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
			Owner: e.owner.String(), EntitlementId: c.EntitlementID,
		})
		require.NoError(t, err)
		_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
			Owner: e.owner.String(), EntitlementId: c.EntitlementID,
		})
		require.ErrorIs(t, err, types.ErrAlreadyClaimed)
	})

	t.Run("freeze guards", func(t *testing.T) {
		e, riskPriv, _, _, c := setupP3(t)
		until := e.ctx.BlockTime().Unix() + 3600

		wrongChain := freezeMsg(t, e, riskPriv, c, until, "7301")
		wrongChain.ChainId = "other_1-1"
		_, err := e.server.FreezeEntitlement(e.ctx, wrongChain)
		require.ErrorIs(t, err, types.ErrChainIDMismatch)

		wrongPool := freezeMsg(t, e, riskPriv, c, until, "7302")
		wrongPool.PoolId = "other-pool"
		_, err = e.server.FreezeEntitlement(e.ctx, wrongPool)
		require.ErrorIs(t, err, types.ErrPoolMismatch)

		_, err = e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, until, "7303"))
		require.NoError(t, err)
		// Second freeze is refused because the entitlement is no longer CLAIMABLE.
		_, err = e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, until, "7304"))
		require.ErrorIs(t, err, types.ErrNotClaimable)
	})

	t.Run("revoke guards", func(t *testing.T) {
		e, riskPriv, auditPriv, _, c := setupP3(t)

		// Not frozen yet: the not-frozen gate fires before wire verification.
		_, err := e.server.RevokeEntitlement(e.ctx, revokeMsg(t, e, riskPriv, auditPriv, c, "7401", "7402"))
		require.ErrorIs(t, err, types.ErrNotFrozen)

		wrongPool := revokeMsg(t, e, riskPriv, auditPriv, c, "7403", "7404")
		wrongPool.PoolId = "other-pool"
		_, err = e.server.RevokeEntitlement(e.ctx, wrongPool)
		require.ErrorIs(t, err, types.ErrPoolMismatch)

		until := e.ctx.BlockTime().Unix() + 3600
		_, err = e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, until, "7405"))
		require.NoError(t, err)

		wrongChain := revokeMsg(t, e, riskPriv, auditPriv, c, "7406", "7407")
		wrongChain.ChainId = "other_1-1"
		_, err = e.server.RevokeEntitlement(e.ctx, wrongChain)
		require.ErrorIs(t, err, types.ErrChainIDMismatch)

		// Same proof reused as both halves is not two independent approvals.
		dup := revokeMsg(t, e, riskPriv, auditPriv, c, "7408", "7409")
		dup.AuditProof = dup.RiskProof
		_, err = e.server.RevokeEntitlement(e.ctx, dup)
		require.ErrorIs(t, err, types.ErrDualAuthRequired)
	})
}

// TestSurface_V1NotFound exercises the absent-record branch of the V1 handlers.
func TestSurface_V1NotFound(t *testing.T) {
	e, _, _ := setupP1(t)
	missingID := "ff" + strings.Repeat("00", 31)

	_, err := e.server.OpenEntitlementPool(e.ctx, &types.MsgOpenEntitlementPool{
		Authority: e.auth.String(), PoolId: testPoolID, Denom: types.DefaultChallengeBondDenom, Version: 1,
	})
	require.ErrorIs(t, err, types.ErrPoolExists)

	_, err = e.server.OpenEntitlementPool(e.ctx, &types.MsgOpenEntitlementPool{
		Authority: e.auth.String(), PoolId: "p-x", Denom: "uosmo", Version: 1,
	})
	require.ErrorIs(t, err, types.ErrInvalidDenom)

	_, err = e.server.FundEntitlementPool(e.ctx, &types.MsgFundEntitlementPool{
		Authority: e.auth.String(), PoolId: "absent-pool", Amount: "1",
	})
	require.ErrorIs(t, err, types.ErrPoolNotFound)

	_, err = e.server.SetEntitlementPoolPaused(e.ctx, &types.MsgSetEntitlementPoolPaused{
		Authority: e.auth.String(), PoolId: "absent-pool", Paused: true,
	})
	require.ErrorIs(t, err, types.ErrPoolNotFound)

	_, err = e.server.SetRoleKeyStatus(e.ctx, &types.MsgSetRoleKeyStatus{
		Authority: e.auth.String(), Role: types.EntitlementRoleApprover, KeyId: "never-registered",
		Status: types.RoleKeyStatusPaused,
	})
	require.ErrorIs(t, err, types.ErrRoleKeyNotFound)

	_, err = e.server.SetAssessmentSignerStatus(e.ctx, &types.MsgSetAssessmentSignerStatus{
		Authority: e.auth.String(), KeyId: "never-registered", Status: types.RoleKeyStatusPaused,
	})
	require.ErrorIs(t, err, types.ErrRoleKeyNotFound)

	_, err = e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.owner.String(), EntitlementId: missingID,
		EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: digestOf(0x0b),
	})
	require.ErrorIs(t, err, types.ErrCandidateNotFound)

	_, err = e.server.ResolveEntitlementChallenge(e.ctx, &types.MsgResolveEntitlementChallenge{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		ChallengeId: missingID, Outcome: "DISMISSED", RationaleDigest: digestOf(0x0c),
	})
	require.ErrorIs(t, err, types.ErrEntitlementChallengeNotFound)

	_, err = e.server.ApproveEntitlement(e.ctx, &types.MsgApproveEntitlement{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID, EntitlementId: missingID,
	})
	require.ErrorIs(t, err, types.ErrCandidateNotFound)

	_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
		Owner: e.owner.String(), EntitlementId: missingID,
	})
	require.ErrorIs(t, err, types.ErrEntitlementNotFound)

	_, err = e.server.FreezeEntitlement(e.ctx, &types.MsgFreezeEntitlement{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID, EntitlementId: missingID,
	})
	require.ErrorIs(t, err, types.ErrEntitlementNotFound)

	_, err = e.server.RevokeEntitlement(e.ctx, &types.MsgRevokeEntitlement{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID, EntitlementId: missingID,
	})
	require.ErrorIs(t, err, types.ErrEntitlementNotFound)

	_, err = e.server.ExpireEntitlementChallenge(e.ctx, &types.MsgExpireEntitlementChallenge{
		Submitter: e.sub.String(), PoolId: testPoolID, ChallengeId: missingID,
	})
	require.ErrorIs(t, err, types.ErrEntitlementChallengeNotFound)
}

// TestSurface_PoolPause covers the per-pool pause switch on the money-in path
// and on the candidate flow.
func TestSurface_PoolPause(t *testing.T) {
	e, pubPriv, _ := setupP1(t)
	_, err := e.server.SetEntitlementPoolPaused(e.ctx, &types.MsgSetEntitlementPoolPaused{
		Authority: e.auth.String(), PoolId: testPoolID, Paused: true,
	})
	require.NoError(t, err)

	_, err = e.server.FundEntitlementPool(e.ctx, &types.MsgFundEntitlementPool{
		Authority: e.auth.String(), PoolId: testPoolID, Amount: "10",
	})
	require.ErrorIs(t, err, types.ErrPoolPaused)

	c := sampleCandidateFields()
	_, err = e.server.PublishCandidate(e.ctx, publishMsgFor(t, e, pubPriv, c, "7501", wireExp(e.ctx), false))
	require.ErrorIs(t, err, types.ErrPoolPaused)

	// Unpause restores both paths.
	_, err = e.server.SetEntitlementPoolPaused(e.ctx, &types.MsgSetEntitlementPoolPaused{
		Authority: e.auth.String(), PoolId: testPoolID, Paused: false,
	})
	require.NoError(t, err)
	_, err = e.server.PublishCandidate(e.ctx, publishMsgFor(t, e, pubPriv, c, "7502", wireExp(e.ctx), false))
	require.NoError(t, err)
}
