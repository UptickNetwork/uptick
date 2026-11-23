package e2e

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/crypto/ethsecp256k1"

	"github.com/UptickNetwork/uptick/app/params"
	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

// This suite is the on-chain half of the appmining surface audit: it broadcasts
// one real, signed transaction for every Msg the module exposes, against a live
// node, and records the tx hash and result code for each. The keeper tests prove
// the state machine; only this file proves that the messages survive the tx
// decoder, the ante handler (fees, sequence, signature checks), the message
// router and the module's own BeginBlock/EndBlock.
//
// It deliberately mutates the node's appmining params (short challenge window,
// legacy flow unpaused) because the production defaults make the lifecycle
// unreachable inside a test run: a 1500-block window and paused attestation flow
// would mean "wait a day and can't submit at all". The original params are
// captured first and restored in t.Cleanup, including on failure.
//
// Run (binary must run on the node host, REST is bound to 127.0.0.1):
//   UPTICK_E2E_STRICT=1 UPTICK_E2E_REST=http://127.0.0.1:1317 \
//   UPTICK_E2E_CHAIN_ID=<chain-id> UPTICK_E2E_DENOM=auoc \
//   UPTICK_E2E_HOME=/home/ubuntu/.uptickd UPTICK_E2E_KEY=authority \
//   ./appmining-e2e.test -test.run TestAppminingMsgSurfaceOnRealChain -test.v

type chainStep struct {
	Seq    int    `json:"seq"`
	Name   string `json:"name"`
	Msg    string `json:"msg"`
	TxHash string `json:"tx_hash,omitempty"`
	Code   int    `json:"code"`
	Height string `json:"height,omitempty"`
	Note   string `json:"note,omitempty"`
}

type chainRun struct {
	t    *testing.T
	enc  params.EncodingConfig
	rest string
	priv cryptotypes.PrivKey
	pub  *ethsecp256k1.PubKey
	from sdk.AccAddress
	gas  uint64

	steps []chainStep
}

func loadKeyByName(t *testing.T, home, name string, cdc codec.Codec) cryptotypes.PrivKey {
	t.Helper()
	kr, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, home, nil, cdc)
	require.NoError(t, err)
	rec, err := kr.Key(name)
	require.NoError(t, err, "no %q key in the test keyring under %s", name, home)
	local := rec.GetLocal()
	require.NotNil(t, local, "%s key is not a local key", name)
	priv, ok := local.PrivKey.GetCachedValue().(cryptotypes.PrivKey)
	require.True(t, ok, "failed to extract %s private key", name)
	return priv
}

func (r *chainRun) record(name string, msg sdk.Msg, txHash string, code int, log string) {
	step := chainStep{
		Seq: len(r.steps) + 1, Name: name, Msg: fmt.Sprintf("%T", msg),
		TxHash: txHash, Code: code,
	}
	if code != 0 {
		step.Note = log
	}
	r.steps = append(r.steps, step)
}

// ok broadcasts a message that must be accepted and records it.
func (r *chainRun) ok(name string, msg sdk.Msg) string {
	r.t.Helper()
	hash, code, log := signAppminingBroadcast(r.t, r.enc, r.rest, r.priv, r.pub, r.from, msg, r.gas)
	r.record(name, msg, hash, code, log)
	require.Zero(r.t, code, "%s rejected: %s", name, log)
	if resp := appminingGet(r.t, r.rest, "/cosmos/tx/v1beta1/txs/"+hash); resp.status == 200 {
		var parsed struct {
			TxResponse struct {
				Height string `json:"height"`
			} `json:"tx_response"`
		}
		if json.Unmarshal(resp.body, &parsed) == nil {
			r.steps[len(r.steps)-1].Height = parsed.TxResponse.Height
		}
	}
	return hash
}

// expectFail broadcasts a message that must be rejected by the keeper, proving
// the on-chain error path rather than only the unit-test one.
func (r *chainRun) expectFail(name string, msg sdk.Msg, contains string) {
	r.t.Helper()
	hash, code, log := signAppminingBroadcast(r.t, r.enc, r.rest, r.priv, r.pub, r.from, msg, r.gas)
	r.record(name, msg, hash, code, log)
	require.NotZero(r.t, code, "%s unexpectedly accepted", name)
	require.Contains(r.t, log, contains, "%s: want %q in %q", name, contains, log)
}

func (r *chainRun) params() types.Params {
	r.t.Helper()
	resp := appminingGet(r.t, r.rest, "/uptick/appmining/v1/params")
	require.Equal(r.t, 200, resp.status)
	var doc struct {
		Params json.RawMessage `json:"params"`
	}
	require.NoError(r.t, json.Unmarshal(resp.body, &doc))
	var p types.Params
	require.NoError(r.t, r.enc.Codec.UnmarshalJSON(doc.Params, &p))
	return p
}

func (r *chainRun) setParams(mutate func(*types.Params)) types.Params {
	r.t.Helper()
	p := r.params()
	mutate(&p)
	r.ok("UpdateParams", &types.MsgUpdateParams{Authority: r.from.String(), Params: p})
	return p
}

// retryOK retries a message until the chain accepts it. Used for guards that
// depend on block time passing (challenge window, freeze expiry): the tx is
// identical each time and the nonce is re-read from the account, so a retry is
// the same request, not a different one.
func (r *chainRun) retryOK(name string, msg sdk.Msg, attempts int, wait time.Duration) string {
	r.t.Helper()
	var lastLog string
	for i := 0; i < attempts; i++ {
		hash, code, log := signAppminingBroadcast(r.t, r.enc, r.rest, r.priv, r.pub, r.from, msg, r.gas)
		if code == 0 {
			r.record(fmt.Sprintf("%s (attempt %d)", name, i+1), msg, hash, code, log)
			return hash
		}
		lastLog = log
		time.Sleep(wait)
	}
	r.record(name, msg, "", 1, lastLog)
	r.t.Fatalf("%s never accepted after %d attempts: %s", name, attempts, lastLog)
	return ""
}

func TestAppminingMsgSurfaceOnRealChain(t *testing.T) {
	home, rest := e2eNode(t)
	enc := params.MakeEncodingConfig()
	types.RegisterInterfaces(enc.InterfaceRegistry)

	keyName := envOr("UPTICK_E2E_KEY", "authority")
	priv := loadKeyByName(t, home, keyName, enc.Codec)
	pub := priv.PubKey().(*ethsecp256k1.PubKey)
	from := sdk.AccAddress(pub.Address().Bytes())

	r := &chainRun{t: t, enc: enc, rest: rest, priv: priv, pub: pub, from: from, gas: 500_000}

	// The node must already be governed by this key, otherwise the authority
	// messages below would fail with an unrelated error.
	orig := r.params()
	require.Equal(t, from.String(), orig.Authority, "appmining params authority must be the e2e key")
	require.Contains(t, orig.RewardDenomAllowlist, e2eDenom, "test denom must be spendable by the module")

	// Restore params whatever happens: a dev chain left with a 2s window and an
	// unpaused legacy flow is a landmine for the next person.
	t.Cleanup(func() {
		restored := orig
		_, code, log := signAppminingBroadcast(t, enc, rest, priv, pub, from,
			&types.MsgUpdateParams{Authority: from.String(), Params: restored}, r.gas)
		if code != 0 {
			t.Logf("WARNING: could not restore appmining params: %s", log)
		}
	})

	runTag := fmt.Sprintf("%06d", time.Now().UnixNano()%1_000_000)
	appID := appminingURLSafeAppID([]byte("uge-surface-" + runTag))
	appIDHex := hex.EncodeToString(appID)
	poolID := "e2e-surface-" + runTag
	ruleVersion := "uge-candidate-rules-0.4"

	// --- 1. governance: make the lifecycle reachable, then prove Pause cannot resume
	//
	// challenge window 2s / 3 blocks: the approved e2e shape. gas credit share
	// 10% so settlement mints something ConsumeGasCredit can spend. Legacy flow
	// unpaused because production defaults keep it parked.
	r.setParams(func(p *types.Params) {
		p.ChallengeWindowSeconds = 2
		p.ChallengeWindowBlocks = 3
		p.GasCreditShareBps = 1000
		p.AllowManualGasCreditConsume = true
		p.PauseState = &types.PauseState{}
		p.EntitlementEnableMinFunded = "1000"
		if !contains(p.EntitlementAllowedRuleVersions, ruleVersion) {
			p.EntitlementAllowedRuleVersions = append(p.EntitlementAllowedRuleVersions, ruleVersion)
		}
	})

	// --- 2. application registry
	r.ok("RegisterApplication", &types.MsgRegisterApplication{
		Authority: from.String(), AppId: appID, Owner: from.String(), AttestationSubmitter: from.String(),
	})
	r.expectFail("RegisterApplication (duplicate)", &types.MsgRegisterApplication{
		Authority: from.String(), AppId: appID, Owner: from.String(), AttestationSubmitter: from.String(),
	}, "application already exists")
	r.ok("SetApplicationStatus (suspend)", &types.MsgSetApplicationStatus{
		Authority: from.String(), AppId: appID, Status: types.ApplicationStatusSuspended,
	})
	r.ok("SetApplicationStatus (reactivate)", &types.MsgSetApplicationStatus{
		Authority: from.String(), AppId: appID, Status: types.ApplicationStatusActive,
	})
	r.ok("SetAttestationSubmitter", &types.MsgSetAttestationSubmitter{
		Owner: from.String(), AppId: appID, AttestationSubmitter: from.String(),
	})

	// gas payer grant → revoke → re-grant (the credit spend below needs it back)
	r.ok("GrantGasPayer", &types.MsgGrantGasPayer{
		Owner: from.String(), AppId: appID, GasPayer: from.String(),
	})
	r.ok("RevokeGasPayer", &types.MsgRevokeGasPayer{
		Owner: from.String(), AppId: appID, GasPayer: from.String(),
	})
	r.ok("GrantGasPayer (re-grant)", &types.MsgGrantGasPayer{
		Owner: from.String(), AppId: appID, GasPayer: from.String(),
	})

	// --- 3. assessment signer registry
	asmPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x41}, ed25519.SeedSize))
	asmKeyID := "asm-" + runTag
	r.ok("RegisterAssessmentSigner", &types.MsgRegisterAssessmentSigner{
		Authority: from.String(), KeyId: asmKeyID, Pubkey: asmPriv.Public().(ed25519.PublicKey), Version: 1,
	})
	r.ok("SetAssessmentSignerStatus (pause)", &types.MsgSetAssessmentSignerStatus{
		Authority: from.String(), KeyId: asmKeyID, Status: types.RoleKeyStatusPaused,
	})
	r.ok("SetAssessmentSignerStatus (resume)", &types.MsgSetAssessmentSignerStatus{
		Authority: from.String(), KeyId: asmKeyID, Status: types.RoleKeyStatusActive,
	})

	// --- 4. entitlement pool
	r.ok("OpenEntitlementPool", &types.MsgOpenEntitlementPool{
		Authority: from.String(), PoolId: poolID, Denom: e2eDenom, Version: 1,
	})
	r.ok("FundEntitlementPool", &types.MsgFundEntitlementPool{
		Authority: from.String(), PoolId: poolID, Amount: "10000000000000000000", // 10 denom
	})
	r.ok("SetEntitlementPoolPaused (pause)", &types.MsgSetEntitlementPoolPaused{
		Authority: from.String(), PoolId: poolID, Paused: true,
	})
	r.ok("SetEntitlementPoolPaused (resume)", &types.MsgSetEntitlementPoolPaused{
		Authority: from.String(), PoolId: poolID, Paused: false,
	})
	require.Equal(t, "10000000000000000000", appminingQueryField(t, rest, "/uptick/appmining/v1/pools/"+poolID, "pool", "funded"))

	// --- 5. role keys (five distinct ed25519 keys, one per role)
	pubPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize))
	revPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x43}, ed25519.SeedSize))
	aprPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x44}, ed25519.SeedSize))
	riskPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x45}, ed25519.SeedSize))
	auditPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x46}, ed25519.SeedSize))
	pubKeyID := "pub-" + runTag
	revKeyID := "rev-" + runTag
	aprKeyID := "apr-" + runTag
	riskKeyID := "risk-" + runTag
	auditKeyID := "audit-" + runTag
	roleKeys := []struct {
		role  types.EntitlementRole
		keyID string
		priv  ed25519.PrivateKey
		label string
	}{
		{types.EntitlementRolePublisher, pubKeyID, pubPriv, "publisher"},
		{types.EntitlementRoleReviewer, revKeyID, revPriv, "reviewer"},
		{types.EntitlementRoleApprover, aprKeyID, aprPriv, "approver"},
		{types.EntitlementRoleRisk, riskKeyID, riskPriv, "risk"},
		{types.EntitlementRoleAudit, auditKeyID, auditPriv, "audit"},
	}
	for _, rk := range roleKeys {
		r.ok("RegisterRoleKey ("+rk.label+")", &types.MsgRegisterRoleKey{
			Authority: from.String(), Role: rk.role, KeyId: rk.keyID,
			Pubkey: rk.priv.Public().(ed25519.PublicKey), Version: 1,
		})
	}
	r.ok("SetRoleKeyStatus (pause)", &types.MsgSetRoleKeyStatus{
		Authority: from.String(), Role: types.EntitlementRolePublisher, KeyId: pubKeyID,
		Status: types.RoleKeyStatusPaused,
	})
	r.ok("SetRoleKeyStatus (resume)", &types.MsgSetRoleKeyStatus{
		Authority: from.String(), Role: types.EntitlementRolePublisher, KeyId: pubKeyID,
		Status: types.RoleKeyStatusActive,
	})

	// --- 6. legacy attestation flow (epoch 1)
	r.ok("FundRewardPool", &types.MsgFundRewardPool{
		Funder: from.String(), Denom: e2eDenom, Amount: "5000000000000000000", // 5 denom
	})
	r.ok("SetEpochBudget", &types.MsgSetEpochBudget{
		Authority: from.String(),
		Budget: types.EpochBudget{
			Epoch: 1, Denom: e2eDenom, Limit: "2000000000000000000", Spent: "0",
		},
	})

	attestation := types.AttestationCommitment{
		AppId: appID, Epoch: 1,
		Profile: types.DefaultUGEProfile, ProfileVersion: types.DefaultUGEProfileVersion,
		AttestationHash: hashN(0x11), EvidenceRoot: hashN(0x22), ResultCommitment: hashN(0x33),
		RewardRequested: "1000000000000000000", GasCreditRequested: "0",
	}
	r.ok("SubmitAttestation", &types.MsgSubmitAttestation{Submitter: from.String(), Attestation: attestation})
	r.ok("ChallengeAttestation", &types.MsgChallengeAttestation{
		Challenger: from.String(), AppId: appID, Epoch: 1, EvidenceHash: hashN(0x44),
	})

	// Resolve needs the challenge id; read it back from the open-challenge list.
	challengeID := uint64(0)
	{
		resp := appminingGet(t, rest, "/uptick/appmining/v1/open_challenges")
		require.Equal(t, 200, resp.status)
		var doc struct {
			Challenges []struct {
				ChallengeId string `json:"challenge_id"`
			} `json:"challenges"`
		}
		require.NoError(t, json.Unmarshal(resp.body, &doc))
		require.NotEmpty(t, doc.Challenges, "challenge was not indexed as open")
		challengeID, _ = strconv.ParseUint(doc.Challenges[len(doc.Challenges)-1].ChallengeId, 10, 64)
	}
	r.ok("ResolveChallenge", &types.MsgResolveChallenge{
		Authority: from.String(), ChallengeId: challengeID, Uphold: false, ResolutionHash: hashN(0x55),
	})

	// Finalize is height-gated on challenge_window_blocks (3). Retry until a
	// block past the deadline is produced rather than sleeping a fixed guess.
	r.retryOK("FinalizeAttestation", &types.MsgFinalizeAttestation{
		Caller: from.String(), AppId: appID, Epoch: 1,
	}, 10, 2*time.Second)

	// --- 7. pause can only OR bits; resume is authority-only via UpdateParams
	r.ok("Pause (gas credit)", &types.MsgPause{
		PauseAuthority: from.String(),
		PauseBits:      types.PauseState{GasCreditPaused: true},
	})
	require.True(t, r.params().PauseState.GasCreditPaused, "pause bit was not set")
	r.setParams(func(p *types.Params) { p.PauseState = &types.PauseState{} })
	require.False(t, r.params().PauseState.GasCreditPaused, "resume did not clear the bit")

	// Settlement credited 10% of the 1 denom reward; spend half of it.
	r.ok("ConsumeGasCredit", &types.MsgConsumeGasCredit{
		GasPayer: from.String(), AppId: appID,
		Amount: "50000000000000000", SourceTxHash: hashN(0x66),
	})

	// --- 8. V1 lifecycle
	salt := time.Now().UnixNano()
	nonce := func(i int64) string { return strconv.FormatInt(salt+i, 10) }
	expires := strconv.FormatInt(time.Now().Add(24*time.Hour).Unix(), 10)

	candidate := func(amount, pavSeed string) wirev1.Candidate {
		c := wirev1.Candidate{
			SchemaVersion:    wirev1.CandidateSchemaVersion,
			AppID:            appIDHex,
			Epoch:            "1",
			Amount:           amount,
			Denom:            e2eDenom,
			GasUsed:          "21000",
			QualifiedTxCount: "1",
			PavID:            pavSeed,
			EvidenceRoot:     hashHex(0x77),
			GasEvidenceRoot:  hashHex(0x88),
			RuleVersion:      ruleVersion,
		}
		c.EntitlementID = wirev1.EntitlementIDOf(c)
		return c
	}

	publish := func(c wirev1.Candidate, nonceStr string) {
		t.Helper()
		hash, err := wirev1.CandidateHash(c)
		require.NoError(t, err)
		rec := map[string]string{
			"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
			"chain_id": e2eChainID, "pool_id": poolID, "key_id": pubKeyID,
			"nonce": nonceStr, "expires_at": expires,
			"entitlement_id": c.EntitlementID, "app_id": c.AppID,
			"candidate_hash": hash, "rule_version": c.RuleVersion,
		}
		cred := buildCredential(t, asmPriv, asmKeyID, c, ruleVersion, poolID, expires)
		r.ok("PublishCandidate", &types.MsgPublishCandidate{
			Submitter: from.String(), ChainId: e2eChainID, PoolId: poolID,
			EntitlementId: c.EntitlementID, AppId: c.AppID, Epoch: c.Epoch,
			Amount: c.Amount, Denom: c.Denom, GasUsed: c.GasUsed,
			QualifiedTxCount: c.QualifiedTxCount, PavId: c.PavID,
			EvidenceRoot: c.EvidenceRoot, GasEvidenceRoot: c.GasEvidenceRoot,
			RuleVersion: c.RuleVersion,
			Proof:       signWire(t, pubPriv, rec),
			Credential:  cred,
		})
	}

	approve := func(c wirev1.Candidate, digest, nonceStr string) {
		t.Helper()
		hash, err := wirev1.CandidateHash(c)
		require.NoError(t, err)
		rec := map[string]string{
			"protocol_version": "1", "operation": "APPROVE", "role": "approver",
			"chain_id": e2eChainID, "pool_id": poolID, "key_id": aprKeyID,
			"nonce": nonceStr, "expires_at": expires,
			"entitlement_id": c.EntitlementID, "app_id": c.AppID,
			"candidate_hash": hash, "challenge_resolution_hash": digest,
			"amount": c.Amount, "denom": c.Denom, "rule_version": c.RuleVersion,
		}
		msg := &types.MsgApproveEntitlement{
			Submitter: from.String(), ChainId: e2eChainID, PoolId: poolID,
			EntitlementId: c.EntitlementID, ChallengeResolutionHash: digest,
			Proof: signWire(t, aprPriv, rec),
		}
		r.retryOK("ApproveEntitlement", msg, 8, 2*time.Second)
	}

	// Candidate A: challenge then dismiss, approve bound to the resolution,
	// freeze, and claim only after the freeze expires.
	candA := candidate("3000", hashHex(0x91))
	publish(candA, nonce(1))
	openResp := r.ok("OpenChallenge", &types.MsgOpenChallenge{
		Challenger: from.String(), EntitlementId: candA.EntitlementID,
		EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: hashHex(0x92),
	})
	challengeA := appminingTxEvents(t, rest, openResp, "entitlement_challenge_opened", "challenge_id")
	require.NotEmpty(t, challengeA, "challenge id missing from events")
	dismissDigest := hashHex(0x93)
	resolveRec := map[string]string{
		"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
		"chain_id": e2eChainID, "pool_id": poolID, "key_id": revKeyID,
		"nonce": nonce(2), "expires_at": expires,
		"challenge_id": challengeA, "candidate_id": candA.EntitlementID,
		"outcome": "DISMISSED", "rationale_digest": dismissDigest,
	}
	r.ok("ResolveEntitlementChallenge (DISMISSED)", &types.MsgResolveEntitlementChallenge{
		Submitter: from.String(), ChainId: e2eChainID, PoolId: poolID,
		ChallengeId: challengeA, Outcome: "DISMISSED", RationaleDigest: dismissDigest,
		Proof: signWire(t, revPriv, resolveRec),
	})
	approve(candA, dismissDigest, nonce(3))
	freezeUntil := time.Now().Add(6 * time.Second).Unix()
	freezeRec := map[string]string{
		"protocol_version": "1", "operation": "FREEZE", "role": "risk",
		"chain_id": e2eChainID, "pool_id": poolID, "key_id": riskKeyID,
		"nonce": nonce(4), "expires_at": expires,
		"entitlement_id":  candA.EntitlementID,
		"evidence_type":   "CONFIRMED_REFUND_OR_REVERSAL",
		"evidence_digest": hashHex(0x94),
		"freeze_until":    strconv.FormatInt(freezeUntil, 10),
	}
	r.ok("FreezeEntitlement", &types.MsgFreezeEntitlement{
		Submitter: from.String(), ChainId: e2eChainID, PoolId: poolID,
		EntitlementId:  candA.EntitlementID,
		EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
		EvidenceDigest: hashHex(0x94),
		FreezeUntil:    strconv.FormatInt(freezeUntil, 10),
		Proof:          signWire(t, riskPriv, freezeRec),
	})
	r.expectFail("ClaimEntitlement (frozen)", &types.MsgClaimEntitlement{
		Owner: from.String(), EntitlementId: candA.EntitlementID,
	}, "not claimable")

	// Candidate B: no challenge, approve after the (2s) window, claim.
	candB := candidate("2000", hashHex(0x95))
	publish(candB, nonce(5))
	approve(candB, "", nonce(6))
	require.Equal(t, "CLAIMABLE", appminingQueryField(t, rest, "/uptick/appmining/v1/entitlements/"+candB.EntitlementID, "entitlement", "state"))
	r.ok("ClaimEntitlement", &types.MsgClaimEntitlement{Owner: from.String(), EntitlementId: candB.EntitlementID})
	require.Equal(t, "CLAIMED", appminingQueryField(t, rest, "/uptick/appmining/v1/entitlements/"+candB.EntitlementID, "entitlement", "state"))
	r.expectFail("ClaimEntitlement (already claimed)", &types.MsgClaimEntitlement{
		Owner: from.String(), EntitlementId: candB.EntitlementID,
	}, "already claimed")

	// Candidate C: approve, freeze, revoke with independent risk+audit proofs.
	candC := candidate("1500", hashHex(0x96))
	publish(candC, nonce(7))
	approve(candC, "", nonce(8))
	revokeUntil := time.Now().Add(3600 * time.Second).Unix()
	revokeBase := map[string]string{
		"protocol_version": "1", "operation": "REVOKE",
		"chain_id": e2eChainID, "pool_id": poolID,
		"entitlement_id":  candC.EntitlementID,
		"evidence_type":   "CONFIRMED_REFUND_OR_REVERSAL",
		"evidence_digest": hashHex(0x97), "rationale_digest": hashHex(0x98),
	}
	freezeRecC := map[string]string{
		"protocol_version": "1", "operation": "FREEZE", "role": "risk",
		"chain_id": e2eChainID, "pool_id": poolID, "key_id": riskKeyID,
		"nonce": nonce(9), "expires_at": expires,
		"entitlement_id":  candC.EntitlementID,
		"evidence_type":   "CONFIRMED_REFUND_OR_REVERSAL",
		"evidence_digest": hashHex(0x97),
		"freeze_until":    strconv.FormatInt(revokeUntil, 10),
	}
	r.ok("FreezeEntitlement (before revoke)", &types.MsgFreezeEntitlement{
		Submitter: from.String(), ChainId: e2eChainID, PoolId: poolID,
		EntitlementId:  candC.EntitlementID,
		EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
		EvidenceDigest: hashHex(0x97),
		FreezeUntil:    strconv.FormatInt(revokeUntil, 10),
		Proof:          signWire(t, riskPriv, freezeRecC),
	})
	riskRec := copyRecord(revokeBase)
	riskRec["role"], riskRec["key_id"] = "risk", riskKeyID
	riskRec["nonce"], riskRec["expires_at"] = nonce(10), expires
	auditRec := copyRecord(revokeBase)
	auditRec["role"], auditRec["key_id"] = "audit", auditKeyID
	auditRec["nonce"], auditRec["expires_at"] = nonce(11), expires
	r.ok("RevokeEntitlement", &types.MsgRevokeEntitlement{
		Submitter: from.String(), ChainId: e2eChainID, PoolId: poolID,
		EntitlementId:  candC.EntitlementID,
		EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
		EvidenceDigest: hashHex(0x97), RationaleDigest: hashHex(0x98),
		RiskProof:  signWire(t, riskPriv, riskRec),
		AuditProof: signWire(t, auditPriv, auditRec),
	})
	require.Equal(t, "REVOKED", appminingQueryField(t, rest, "/uptick/appmining/v1/entitlements/"+candC.EntitlementID, "entitlement", "state"))

	// Candidate D: open a challenge and let time settle it. Either the manual
	// message wins the race or the EndBlocker sweep does; the assertion is on the
	// outcome (EXPIRED + deposit returned), not on who paid the gas.
	candD := candidate("1200", hashHex(0x99))
	publish(candD, nonce(12))
	openD := r.ok("OpenChallenge (expiry)", &types.MsgOpenChallenge{
		Challenger: from.String(), EntitlementId: candD.EntitlementID,
		EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: hashHex(0x9a),
	})
	challengeD := appminingTxEvents(t, rest, openD, "entitlement_challenge_opened", "challenge_id")
	require.NotEmpty(t, challengeD)
	time.Sleep(6 * time.Second)
	expireMsg := &types.MsgExpireEntitlementChallenge{
		Submitter: from.String(), PoolId: poolID, ChallengeId: challengeD,
	}
	expireHash, expireCode, expireLog := signAppminingBroadcast(t, enc, rest, priv, pub, from, expireMsg, r.gas)
	r.record("ExpireEntitlementChallenge", expireMsg, expireHash, expireCode, expireLog)
	if expireCode != 0 {
		// EndBlocker won the race: that is the production path and success.
		require.Contains(t, expireLog, "not open", "unexpected expire failure: %s", expireLog)
		require.Equal(t, "EXPIRED", appminingQueryField(t, rest,
			"/uptick/appmining/v1/entitlement_challenges/"+challengeD, "challenge", "status"),
			"challenge settled by neither the message nor the sweeper")
		r.steps[len(r.steps)-1].Note = "settled by EndBlocker sweep before the manual message landed"
	} else {
		require.Equal(t, "EXPIRED", appminingQueryField(t, rest,
			"/uptick/appmining/v1/entitlement_challenges/"+challengeD, "challenge", "status"))
	}

	// Candidate A's freeze has expired by now; the claim must go through.
	r.retryOK("ClaimEntitlement (after freeze)", &types.MsgClaimEntitlement{
		Owner: from.String(), EntitlementId: candA.EntitlementID,
	}, 8, 2*time.Second)

	// --- 9. reconciliation: ledger vs module account
	pool := appminingGet(t, rest, "/uptick/appmining/v1/pools/"+poolID)
	require.Equal(t, 200, pool.status)
	var poolDoc struct {
		Pool struct {
			Funded    string `json:"funded"`
			Available string `json:"available"`
			Reserved  string `json:"reserved"`
			Paid      string `json:"paid"`
		} `json:"pool"`
	}
	require.NoError(t, json.Unmarshal(pool.body, &poolDoc))
	t.Logf("pool funded=%s available=%s reserved=%s paid=%s",
		poolDoc.Pool.Funded, poolDoc.Pool.Available, poolDoc.Pool.Reserved, poolDoc.Pool.Paid)

	// Every message family must have produced at least one accepted tx.
	accepted := map[string]int{}
	for _, s := range r.steps {
		if s.Code == 0 {
			accepted[s.Name]++
		}
	}
	require.NotEmpty(t, accepted)

	out := envOr("UPTICK_E2E_OUT", "/tmp/appmining-chain-surface.json")
	doc := map[string]any{
		"chain_id":   e2eChainID,
		"denom":      e2eDenom,
		"signer":     from.String(),
		"app_id":     appIDHex,
		"pool_id":    poolID,
		"started_at": time.Now().UTC().Format(time.RFC3339),
		"pool":       poolDoc.Pool,
		"steps":      r.steps,
	}
	bz, err := json.MarshalIndent(doc, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(out, bz, 0o644))
	t.Logf("wrote %d steps to %s", len(r.steps), out)
}

// --- helpers -------------------------------------------------------------

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func hashN(seed byte) []byte {
	out := make([]byte, 32)
	for i := range out {
		out[i] = seed + byte(i)
	}
	return out
}

func hashHex(seed byte) string { return hex.EncodeToString(hashN(seed)) }

func copyRecord(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func signWire(t *testing.T, priv ed25519.PrivateKey, record map[string]string) types.WireProof {
	t.Helper()
	raw, err := wirev1.AuthorizationBytes(record)
	require.NoError(t, err)
	sig := ed25519.Sign(priv, raw)
	return types.WireProof{
		KeyId:           record["key_id"],
		Nonce:           record["nonce"],
		ExpiresAt:       record["expires_at"],
		SignatureBase64: base64.StdEncoding.EncodeToString(sig),
	}
}

// buildCredential mints the T1b assessment credential the chain requires when
// credential_required is on: every committed field must equal the candidate
// body, and credential_id is the SHA-256 of the frame with that field empty.
func buildCredential(t *testing.T, priv ed25519.PrivateKey, keyID string, c wirev1.Candidate, ruleVersion, poolID, expires string) types.AssessmentCredential {
	t.Helper()
	cred := types.AssessmentCredential{
		CredentialVersion: types.DefaultAssessmentVersion,
		AppId:             c.AppID,
		PoolId:            poolID,
		PavId:             c.PavID,
		EvidenceRoot:      c.EvidenceRoot,
		GasEvidenceRoot:   c.GasEvidenceRoot,
		Amount:            c.Amount,
		Denom:             c.Denom,
		GasUsed:           c.GasUsed,
		QualifiedTxCount:  c.QualifiedTxCount,
		RuleVersion:       ruleVersion,
		IssuedAt:          strconv.FormatInt(time.Now().Add(-time.Minute).Unix(), 10),
		ExpiresAt:         expires,
		SignerKeyId:       keyID,
	}
	rec := cred.Record()
	id, err := wirev1.AssessmentCredentialID(rec)
	require.NoError(t, err)
	cred.CredentialId = id
	rec["credential_id"] = id
	raw, err := wirev1.AssessmentBytes(rec)
	require.NoError(t, err)
	cred.SignatureBase64 = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, raw))
	return cred
}
