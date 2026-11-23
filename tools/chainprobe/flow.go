package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

// allAppminingMsgs is the proto Tx service, in declaration order. The run end
// checks that every one of them produced at least one signed transaction; a
// missing entry means the audit covered a code path nobody can actually call.
var allAppminingMsgs = []string{
	"RegisterApplication", "SubmitAttestation", "ChallengeAttestation",
	"ResolveChallenge", "FinalizeAttestation", "GrantGasPayer", "RevokeGasPayer",
	"SetAttestationSubmitter", "ConsumeGasCredit", "SetEpochBudget",
	"UpdateParams", "SetApplicationStatus", "Pause", "FundRewardPool",
	"OpenEntitlementPool", "FundEntitlementPool", "SetEntitlementPoolPaused",
	"RegisterRoleKey", "SetRoleKeyStatus", "RegisterAssessmentSigner",
	"SetAssessmentSignerStatus", "PublishCandidate", "OpenChallenge",
	"ResolveEntitlementChallenge", "ExpireEntitlementChallenge",
	"ApproveEntitlement", "ClaimEntitlement", "FreezeEntitlement",
	"RevokeEntitlement",
}

func (p *probe) run() {
	// The node must already be governed by this key, otherwise every authority
	// message below would fail with an unrelated error.
	p.orig = p.params()
	p.haveOrig = true
	if got := p.orig.Authority; got != p.from.String() {
		p.die("params authority is %s, key is %s: use the authority key", got, p.from.String())
	}
	if !containsStr(p.orig.RewardDenomAllowlist, p.cfg.denom) {
		p.die("denom %s is not in reward_denom_allowlist %v", p.cfg.denom, p.orig.RewardDenomAllowlist)
	}

	runTag := fmt.Sprintf("%06d", time.Now().UnixNano()%1_000_000)
	// Every run gets its own epoch: a fixed epoch would collide with the budget
	// and attestation records left by the previous run, and "limit below spent"
	// is a legitimate rejection that would look like a failure here.
	epoch := uint64(1_000_000 + time.Now().UnixNano()%1_000_000)
	appID := urlSafeAppID([]byte("uge-surface-" + runTag))
	appIDHex := hex.EncodeToString(appID)
	poolID := "e2e-surface-" + runTag
	ruleVersion := "uge-candidate-rules-0.4"
	p.report.AppID, p.report.PoolID = appIDHex, poolID

	// A gas payer address can be bound to only one application, so a binding left
	// behind by an earlier (aborted) run blocks this one. Clear ours first.
	p.revokeStaleGasPayers()

	// --- 1. governance: make the lifecycle reachable -----------------------
	//
	// challenge window 2s / 3 blocks: short enough that "publish, wait, approve"
	// fits in one run. gas credit share 10% so settlement mints something
	// ConsumeGasCredit can actually spend. Legacy flow unpaused because the
	// production defaults keep it parked.
	p.groupf("governance")
	p.setParams(func(pp *types.Params) {
		// 60s is the smallest window ValidateBasic accepts; anything shorter is
		// rejected on chain before the flow even starts.
		pp.ChallengeWindowSeconds = 60
		pp.ChallengeWindowBlocks = 3
		pp.GasCreditShareBps = 1000
		pp.AllowManualGasCreditConsume = true
		pp.PauseState = &types.PauseState{}
		pp.EntitlementEnableMinFunded = "1000"
		// The chain's default deposit is 500 auoc per challenge; opening four of
		// them would drain the dev signer. One micro-unit keeps the escrow path
		// exercised (funds really do move to and from the module account) at a
		// cost that does not require topping the account up.
		// Same for the legacy attestation bond, which is a separate parameter and
		// is escrowed by ChallengeAttestation.
		pp.ChallengeBondAmount = "1000000000000"
		pp.ChallengeDepositAmount = "1000000000000"
		// V1 deadline = sealed window + grace, and the grace floor is 60s; the
		// default 24h grace would put "expire" out of reach of any test run.
		pp.ChallengeResolutionGraceSeconds = 60
		if !containsStr(pp.EntitlementAllowedRuleVersions, ruleVersion) {
			pp.EntitlementAllowedRuleVersions = append(pp.EntitlementAllowedRuleVersions, ruleVersion)
		}
	})

	// --- 2. application registry -------------------------------------------
	p.groupf("application")
	p.ok("RegisterApplication", &types.MsgRegisterApplication{
		Authority: p.from.String(), AppId: appID, Owner: p.from.String(), AttestationSubmitter: p.from.String(),
	})
	p.reject("RegisterApplication (duplicate)", &types.MsgRegisterApplication{
		Authority: p.from.String(), AppId: appID, Owner: p.from.String(), AttestationSubmitter: p.from.String(),
	}, "application already exists")
	p.ok("SetApplicationStatus (suspend)", &types.MsgSetApplicationStatus{
		Authority: p.from.String(), AppId: appID, Status: types.ApplicationStatusSuspended,
	})
	p.ok("SetApplicationStatus (reactivate)", &types.MsgSetApplicationStatus{
		Authority: p.from.String(), AppId: appID, Status: types.ApplicationStatusActive,
	})
	p.ok("SetAttestationSubmitter", &types.MsgSetAttestationSubmitter{
		Owner: p.from.String(), AppId: appID, AttestationSubmitter: p.from.String(),
	})
	// grant -> revoke -> re-grant: the credit spend further down needs it back.
	p.ok("GrantGasPayer", &types.MsgGrantGasPayer{
		Owner: p.from.String(), AppId: appID, GasPayer: p.from.String(),
	})
	p.ok("RevokeGasPayer", &types.MsgRevokeGasPayer{
		Owner: p.from.String(), AppId: appID, GasPayer: p.from.String(),
	})
	p.ok("GrantGasPayer (re-grant)", &types.MsgGrantGasPayer{
		Owner: p.from.String(), AppId: appID, GasPayer: p.from.String(),
	})

	// --- 3. assessment signer registry -------------------------------------
	p.groupf("assessment")
	keySeed := func(label string) ed25519.PrivateKey {
		h := sha256.Sum256([]byte(label + "/" + runTag))
		return ed25519.NewKeyFromSeed(h[:])
	}
	asmPriv := keySeed("asm")
	asmKeyID := "asm-" + runTag
	p.ok("RegisterAssessmentSigner", &types.MsgRegisterAssessmentSigner{
		Authority: p.from.String(), KeyId: asmKeyID, Pubkey: asmPriv.Public().(ed25519.PublicKey), Version: 1,
	})
	p.ok("SetAssessmentSignerStatus (pause)", &types.MsgSetAssessmentSignerStatus{
		Authority: p.from.String(), KeyId: asmKeyID, Status: types.RoleKeyStatusPaused,
	})
	p.ok("SetAssessmentSignerStatus (resume)", &types.MsgSetAssessmentSignerStatus{
		Authority: p.from.String(), KeyId: asmKeyID, Status: types.RoleKeyStatusActive,
	})

	// --- 4. entitlement pool -----------------------------------------------
	p.groupf("pool")
	p.ok("OpenEntitlementPool", &types.MsgOpenEntitlementPool{
		Authority: p.from.String(), PoolId: poolID, Denom: p.cfg.denom, Version: 1,
	})
	p.ok("FundEntitlementPool", &types.MsgFundEntitlementPool{
		Authority: p.from.String(), PoolId: poolID, Amount: "10000000000000000000",
	})
	p.ok("SetEntitlementPoolPaused (pause)", &types.MsgSetEntitlementPoolPaused{
		Authority: p.from.String(), PoolId: poolID, Paused: true,
	})
	p.ok("SetEntitlementPoolPaused (resume)", &types.MsgSetEntitlementPoolPaused{
		Authority: p.from.String(), PoolId: poolID, Paused: false,
	})
	if got := p.view("/uptick/appmining/v1/pools/"+poolID, "pool", "funded"); got != "10000000000000000000" {
		p.die("pool funded = %s, want 10000000000000000000", got)
	}

	// --- 5. role keys: five distinct ed25519 keys, one per role -------------
	p.groupf("rolekeys")
	pubPriv := keySeed("pub")
	revPriv := keySeed("rev")
	aprPriv := keySeed("apr")
	riskPriv := keySeed("risk")
	auditPriv := keySeed("audit")
	pubKeyID, revKeyID, aprKeyID := "pub-"+runTag, "rev-"+runTag, "apr-"+runTag
	riskKeyID, auditKeyID := "risk-"+runTag, "audit-"+runTag

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
		p.ok("RegisterRoleKey ("+rk.label+")", &types.MsgRegisterRoleKey{
			Authority: p.from.String(), Role: rk.role, KeyId: rk.keyID,
			Pubkey: rk.priv.Public().(ed25519.PublicKey), Version: 1,
		})
	}
	p.ok("SetRoleKeyStatus (pause)", &types.MsgSetRoleKeyStatus{
		Authority: p.from.String(), Role: types.EntitlementRolePublisher, KeyId: pubKeyID,
		Status: types.RoleKeyStatusPaused,
	})
	p.ok("SetRoleKeyStatus (resume)", &types.MsgSetRoleKeyStatus{
		Authority: p.from.String(), Role: types.EntitlementRolePublisher, KeyId: pubKeyID,
		Status: types.RoleKeyStatusActive,
	})

	// --- 6. legacy attestation flow (epoch 1) ------------------------------
	p.groupf("legacy")
	p.ok("FundRewardPool", &types.MsgFundRewardPool{
		Funder: p.from.String(), Denom: p.cfg.denom, Amount: "5000000000000000000",
	})
	p.ok("SetEpochBudget", &types.MsgSetEpochBudget{
		Authority: p.from.String(),
		Budget: types.EpochBudget{
			Epoch: epoch, Denom: p.cfg.denom, Limit: "2000000000000000000", Spent: "0",
		},
	})

	attestation := types.AttestationCommitment{
		// SubmitBasic requires attestation.submitter to be a valid bech32 address,
		// so the "empty means inherit from the message" branch is unreachable.
		Submitter: p.from.String(),
		AppId:     appID, Epoch: epoch,
		Profile: types.DefaultUGEProfile, ProfileVersion: types.DefaultUGEProfileVersion,
		AttestationHash: hashN(0x11), EvidenceRoot: hashN(0x22), ResultCommitment: hashN(0x33),
		RewardRequested: "1000000000000000000", GasCreditRequested: "0",
	}
	p.ok("SubmitAttestation", &types.MsgSubmitAttestation{Submitter: p.from.String(), Attestation: attestation})
	p.ok("ChallengeAttestation", &types.MsgChallengeAttestation{
		Challenger: p.from.String(), AppId: appID, Epoch: epoch, EvidenceHash: hashN(0x44),
	})

	// Resolve needs the challenge id the open-challenge index just minted.
	challengeID := uint64(0)
	{
		resp := p.get("/uptick/appmining/v1/open_challenges")
		if resp.status != 200 {
			p.die("open_challenges: status %d: %s", resp.status, truncate(resp.body))
		}
		var doc struct {
			Challenges []struct {
				ChallengeId string `json:"challenge_id"`
			} `json:"challenges"`
		}
		if err := json.Unmarshal(resp.body, &doc); err != nil {
			p.die("open_challenges: decode: %v", err)
		}
		if len(doc.Challenges) == 0 {
			p.die("challenge was not indexed as open")
		}
		challengeID, _ = strconv.ParseUint(doc.Challenges[len(doc.Challenges)-1].ChallengeId, 10, 64)
	}
	p.ok("ResolveChallenge", &types.MsgResolveChallenge{
		Authority: p.from.String(), ChallengeId: challengeID, Uphold: false, ResolutionHash: hashN(0x55),
	})

	// Finalize is height-gated on challenge_window_blocks (3): retry until a
	// block past the deadline exists rather than sleeping a fixed guess.
	p.retryOK("FinalizeAttestation", &types.MsgFinalizeAttestation{
		Caller: p.from.String(), AppId: appID, Epoch: epoch,
	}, 10, 2*time.Second)

	// --- 7. pause can only OR bits; resume is authority-only ---------------
	p.groupf("pause")
	p.ok("Pause (gas credit)", &types.MsgPause{
		PauseAuthority: p.from.String(),
		PauseBits:      types.PauseState{GasCreditPaused: true},
	})
	if !p.params().PauseState.GasCreditPaused {
		p.die("pause bit was not set on chain")
	}
	p.setParams(func(pp *types.Params) { pp.PauseState = &types.PauseState{} })
	if p.params().PauseState.GasCreditPaused {
		p.die("resume via UpdateParams did not clear the bit")
	}

	// Settlement credited 10% of the 1 denom reward; spend half of it.
	p.ok("ConsumeGasCredit", &types.MsgConsumeGasCredit{
		GasPayer: p.from.String(), AppId: appID,
		Amount: "50000000000000000", SourceTxHash: hashN(0x66),
	})

	// --- 8. V1 entitlement lifecycle ---------------------------------------
	p.groupf("entitlement")
	salt := time.Now().UnixNano()
	nonceOf := func(i int64) string { return strconv.FormatInt(salt+i, 10) }
	expires := strconv.FormatInt(time.Now().Add(24*time.Hour).Unix(), 10)

	candidate := func(amount, pavID string) wirev1.Candidate {
		c := wirev1.Candidate{
			SchemaVersion:    wirev1.CandidateSchemaVersion,
			AppID:            appIDHex,
			Epoch:            "1",
			Amount:           amount,
			Denom:            p.cfg.denom,
			GasUsed:          "21000",
			QualifiedTxCount: "1",
			PavID:            pavID,
			EvidenceRoot:     hashHex(0x77),
			GasEvidenceRoot:  hashHex(0x88),
			RuleVersion:      ruleVersion,
		}
		c.EntitlementID = wirev1.EntitlementIDOf(c)
		return c
	}

	publish := func(c wirev1.Candidate, nonce string) txResult {
		hash, err := wirev1.CandidateHash(c)
		if err != nil {
			p.die("candidate hash: %v", err)
		}
		rec := map[string]string{
			"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
			"chain_id": p.cfg.chainID, "pool_id": poolID, "key_id": pubKeyID,
			"nonce": nonce, "expires_at": expires,
			"entitlement_id": c.EntitlementID, "app_id": c.AppID,
			"candidate_hash": hash, "rule_version": c.RuleVersion,
		}
		return p.ok("PublishCandidate", &types.MsgPublishCandidate{
			Submitter: p.from.String(), ChainId: p.cfg.chainID, PoolId: poolID,
			EntitlementId: c.EntitlementID, AppId: c.AppID, Epoch: c.Epoch,
			Amount: c.Amount, Denom: c.Denom, GasUsed: c.GasUsed,
			QualifiedTxCount: c.QualifiedTxCount, PavId: c.PavID,
			EvidenceRoot: c.EvidenceRoot, GasEvidenceRoot: c.GasEvidenceRoot,
			RuleVersion: c.RuleVersion,
			Proof:       p.signWire(pubPriv, rec),
			Credential:  p.buildCredential(asmPriv, asmKeyID, c, ruleVersion, poolID, expires),
		})
	}

	approve := func(c wirev1.Candidate, digest, nonce string) {
		hash, err := wirev1.CandidateHash(c)
		if err != nil {
			p.die("candidate hash: %v", err)
		}
		rec := map[string]string{
			"protocol_version": "1", "operation": "APPROVE", "role": "approver",
			"chain_id": p.cfg.chainID, "pool_id": poolID, "key_id": aprKeyID,
			"nonce": nonce, "expires_at": expires,
			"entitlement_id": c.EntitlementID, "app_id": c.AppID,
			"candidate_hash": hash, "challenge_resolution_hash": digest,
			"amount": c.Amount, "denom": c.Denom, "rule_version": c.RuleVersion,
		}
		// Approve is window-gated: before the challenge window closes the keeper
		// refuses, so retry instead of guessing a sleep.
		p.retryOK("ApproveEntitlement", &types.MsgApproveEntitlement{
			Submitter: p.from.String(), ChainId: p.cfg.chainID, PoolId: poolID,
			EntitlementId: c.EntitlementID, ChallengeResolutionHash: digest,
			Proof: p.signWire(aprPriv, rec),
		}, 8, 2*time.Second)
	}

	// All four candidates are published first, then a single 60s wait covers the
	// challenge window for every approval: waiting once is the difference between
	// a three minute run and a six minute one, and the window is wall-clock, not
	// per-candidate.
	candA := candidate("3000", hashHex(0x91))
	publish(candA, nonceOf(1))
	candB := candidate("2000", hashHex(0x95))
	publish(candB, nonceOf(2))
	candC := candidate("1500", hashHex(0x96))
	publish(candC, nonceOf(3))
	candD := candidate("1200", hashHex(0x99))
	publish(candD, nonceOf(4))

	// Candidate A: challenge, dismiss. Candidate D: challenge, then let time
	// settle it (manual expire vs EndBlocker sweep — either proves the path).
	openA := p.ok("OpenChallenge", &types.MsgOpenChallenge{
		Challenger: p.from.String(), EntitlementId: candA.EntitlementID,
		EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: hashHex(0x92),
	})
	challengeA := p.txEvent(openA.hash, "entitlement_challenge_opened", "challenge_id")
	if challengeA == "" {
		p.die("challenge id missing from OpenChallenge events")
	}
	dismissDigest := hashHex(0x93)
	resolveRec := map[string]string{
		"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
		"chain_id": p.cfg.chainID, "pool_id": poolID, "key_id": revKeyID,
		"nonce": nonceOf(5), "expires_at": expires,
		"challenge_id": challengeA, "candidate_id": candA.EntitlementID,
		"outcome": "DISMISSED", "rationale_digest": dismissDigest,
	}
	p.ok("ResolveEntitlementChallenge (DISMISSED)", &types.MsgResolveEntitlementChallenge{
		Submitter: p.from.String(), ChainId: p.cfg.chainID, PoolId: poolID,
		ChallengeId: challengeA, Outcome: "DISMISSED", RationaleDigest: dismissDigest,
		Proof: p.signWire(revPriv, resolveRec),
	})
	challengeDOpenedAt := time.Now()
	openD := p.ok("OpenChallenge (expiry)", &types.MsgOpenChallenge{
		Challenger: p.from.String(), EntitlementId: candD.EntitlementID,
		EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: hashHex(0x9a),
	})
	challengeD := p.txEvent(openD.hash, "entitlement_challenge_opened", "challenge_id")
	if challengeD == "" {
		p.die("challenge id missing from the expiry OpenChallenge events")
	}
	// The deadline is sealed window (60s) + grace (60s), so this must be refused
	// now and accepted later; both halves are proven on chain.
	expireAt := challengeDOpenedAt.Add(125 * time.Second) // 60s window + 60s grace + margin
	p.reject("ExpireEntitlementChallenge (too early)", &types.MsgExpireEntitlementChallenge{
		Submitter: p.from.String(), PoolId: poolID, ChallengeId: challengeD,
	}, "resolution window has not elapsed")

	fmt.Printf("     waiting 65s for the 60s challenge window (approvals are window-gated)\n")
	time.Sleep(65 * time.Second)

	approve(candA, dismissDigest, nonceOf(6))
	approve(candB, "", nonceOf(7))
	approve(candC, "", nonceOf(8))

	// A: freeze, then the claim must be refused while frozen.
	freezeUntil := time.Now().Add(6 * time.Second).Unix()
	freezeRecA := map[string]string{
		"protocol_version": "1", "operation": "FREEZE", "role": "risk",
		"chain_id": p.cfg.chainID, "pool_id": poolID, "key_id": riskKeyID,
		"nonce": nonceOf(9), "expires_at": expires,
		"entitlement_id":  candA.EntitlementID,
		"evidence_type":   "CONFIRMED_REFUND_OR_REVERSAL",
		"evidence_digest": hashHex(0x94),
		"freeze_until":    strconv.FormatInt(freezeUntil, 10),
	}
	p.ok("FreezeEntitlement", &types.MsgFreezeEntitlement{
		Submitter: p.from.String(), ChainId: p.cfg.chainID, PoolId: poolID,
		EntitlementId:  candA.EntitlementID,
		EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
		EvidenceDigest: hashHex(0x94),
		FreezeUntil:    strconv.FormatInt(freezeUntil, 10),
		Proof:          p.signWire(riskPriv, freezeRecA),
	})
	p.reject("ClaimEntitlement (frozen)", &types.MsgClaimEntitlement{
		Owner: p.from.String(), EntitlementId: candA.EntitlementID,
	}, "not claimable")

	// B: no challenge at all, approve then claim, and the double claim is refused.
	if got := p.view("/uptick/appmining/v1/entitlements/"+candB.EntitlementID, "entitlement", "state"); !stateIs(got, "CLAIMABLE") {
		p.die("candidate B state = %s, want CLAIMABLE", got)
	}
	p.ok("ClaimEntitlement", &types.MsgClaimEntitlement{Owner: p.from.String(), EntitlementId: candB.EntitlementID})
	if got := p.view("/uptick/appmining/v1/entitlements/"+candB.EntitlementID, "entitlement", "state"); !stateIs(got, "CLAIMED") {
		p.die("candidate B state after claim = %s, want CLAIMED", got)
	}
	p.reject("ClaimEntitlement (already claimed)", &types.MsgClaimEntitlement{
		Owner: p.from.String(), EntitlementId: candB.EntitlementID,
	}, "already claimed")

	// C: freeze, then revoke with independent risk+audit proofs.
	revokeUntil := time.Now().Add(3600 * time.Second).Unix()
	freezeRecC := map[string]string{
		"protocol_version": "1", "operation": "FREEZE", "role": "risk",
		"chain_id": p.cfg.chainID, "pool_id": poolID, "key_id": riskKeyID,
		"nonce": nonceOf(10), "expires_at": expires,
		"entitlement_id":  candC.EntitlementID,
		"evidence_type":   "CONFIRMED_REFUND_OR_REVERSAL",
		"evidence_digest": hashHex(0x97),
		"freeze_until":    strconv.FormatInt(revokeUntil, 10),
	}
	p.ok("FreezeEntitlement (before revoke)", &types.MsgFreezeEntitlement{
		Submitter: p.from.String(), ChainId: p.cfg.chainID, PoolId: poolID,
		EntitlementId:  candC.EntitlementID,
		EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
		EvidenceDigest: hashHex(0x97),
		FreezeUntil:    strconv.FormatInt(revokeUntil, 10),
		Proof:          p.signWire(riskPriv, freezeRecC),
	})
	revokeBase := map[string]string{
		"protocol_version": "1", "operation": "REVOKE",
		"chain_id": p.cfg.chainID, "pool_id": poolID,
		"entitlement_id":  candC.EntitlementID,
		"evidence_type":   "CONFIRMED_REFUND_OR_REVERSAL",
		"evidence_digest": hashHex(0x97), "rationale_digest": hashHex(0x98),
	}
	riskRec := copyRecord(revokeBase)
	riskRec["role"], riskRec["key_id"] = "risk", riskKeyID
	riskRec["nonce"], riskRec["expires_at"] = nonceOf(11), expires
	auditRec := copyRecord(revokeBase)
	auditRec["role"], auditRec["key_id"] = "audit", auditKeyID
	auditRec["nonce"], auditRec["expires_at"] = nonceOf(12), expires
	p.ok("RevokeEntitlement", &types.MsgRevokeEntitlement{
		Submitter: p.from.String(), ChainId: p.cfg.chainID, PoolId: poolID,
		EntitlementId:  candC.EntitlementID,
		EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
		EvidenceDigest: hashHex(0x97), RationaleDigest: hashHex(0x98),
		RiskProof:  p.signWire(riskPriv, riskRec),
		AuditProof: p.signWire(auditPriv, auditRec),
	})
	if got := p.view("/uptick/appmining/v1/entitlements/"+candC.EntitlementID, "entitlement", "state"); !stateIs(got, "REVOKED") {
		p.die("candidate C state = %s, want REVOKED", got)
	}

	// D: wait out window + grace, then settle. The sweeper may win the race; the
	// assertion is on EXPIRED either way.
	if wait := time.Until(expireAt); wait > 0 {
		fmt.Printf("     waiting %s for window+grace before the expire message\n", wait.Round(time.Second))
		time.Sleep(wait)
	}
	start := time.Now()
	expireMsg := &types.MsgExpireEntitlementChallenge{
		Submitter: p.from.String(), PoolId: poolID, ChallengeId: challengeD,
	}
	expRes := p.signAndBroadcast(expireMsg)
	status := p.view("/uptick/appmining/v1/entitlement_challenges/"+challengeD, "challenge", "status")
	note := expRes.rawLog
	passed := stateIs(status, "EXPIRED")
	if expRes.code != 0 {
		passed = passed && strings.Contains(expRes.rawLog, "not open")
		if passed {
			note = "settled by the EndBlocker sweep before the manual message landed"
		}
	}
	p.addStep("ExpireEntitlementChallenge", expireMsg, "OK", expRes, passed, note, time.Since(start))
	if !passed {
		p.die("challenge %s status %q after expire tx (code %d): %s", challengeD, status, expRes.code, expRes.rawLog)
	}

	// Candidate A's freeze has expired by now; the claim must go through.
	p.retryOK("ClaimEntitlement (after freeze)", &types.MsgClaimEntitlement{
		Owner: p.from.String(), EntitlementId: candA.EntitlementID,
	}, 8, 2*time.Second)

	// --- 9. reconciliation: pool ledger ------------------------------------
	p.groupf("reconcile")
	resp := p.get("/uptick/appmining/v1/pools/" + poolID)
	if resp.status != 200 {
		p.die("pool query: status %d: %s", resp.status, truncate(resp.body))
	}
	var poolDoc struct {
		Pool poolSnapshot `json:"pool"`
	}
	if err := json.Unmarshal(resp.body, &poolDoc); err != nil {
		p.die("pool query: decode: %v", err)
	}
	p.report.Pool = poolDoc.Pool
	fmt.Printf("     pool funded=%s available=%s reserved=%s paid=%s\n",
		poolDoc.Pool.Funded, poolDoc.Pool.Available, poolDoc.Pool.Reserved, poolDoc.Pool.Paid)

	// --- 10. completeness self-check ---------------------------------------
	seen := map[string]bool{}
	for _, s := range p.report.Steps {
		seen[strings.TrimPrefix(s.Msg, "*types.Msg")] = true
	}
	var missing []string
	for _, m := range allAppminingMsgs {
		if !seen[m] {
			missing = append(missing, m)
		}
	}
	if len(missing) > 0 {
		p.die("messages never broadcast: %v", missing)
	}
	p.report.Summary["msg_total"] = len(allAppminingMsgs)
	p.report.Summary["msg_exercised"] = len(allAppminingMsgs) - len(missing)
	fmt.Printf("\nmessage surface: %d/%d Msg types broadcast on chain\n",
		len(allAppminingMsgs)-len(missing), len(allAppminingMsgs))
}

// --- wire helpers --------------------------------------------------------

func (p *probe) signWire(priv ed25519.PrivateKey, record map[string]string) types.WireProof {
	raw, err := wirev1.AuthorizationBytes(record)
	if err != nil {
		p.die("authorization bytes: %v", err)
	}
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
func (p *probe) buildCredential(priv ed25519.PrivateKey, keyID string, c wirev1.Candidate, ruleVersion, poolID, expires string) types.AssessmentCredential {
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
	if err != nil {
		p.die("credential id: %v", err)
	}
	cred.CredentialId = id
	rec["credential_id"] = id
	raw, err := wirev1.AssessmentBytes(rec)
	if err != nil {
		p.die("assessment bytes: %v", err)
	}
	cred.SignatureBase64 = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, raw))
	return cred
}

// revokeStaleGasPayers clears any binding of this signer left by an earlier run.
// The app must exist for a binding to be revoked, and the owner has to be this
// signer, which holds for anything this probe created.
func (p *probe) revokeStaleGasPayers() {
	nextKey := ""
	found := 0
	for {
		path := "/uptick/appmining/v1/applications?pagination.limit=100"
		if nextKey != "" {
			path += "&pagination.key=" + url.QueryEscape(nextKey)
		}
		resp := p.get(path)
		if resp.status != 200 {
			p.die("list applications: status %d: %s", resp.status, truncate(resp.body))
		}
		var doc struct {
			Applications []struct {
				AppID     string   `json:"app_id"`
				Owner     string   `json:"owner"`
				GasPayers []string `json:"gas_payers"`
			} `json:"applications"`
			Pagination struct {
				NextKey string `json:"next_key"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal(resp.body, &doc); err != nil {
			p.die("list applications: decode: %v", err)
		}
		for _, a := range doc.Applications {
			for _, gp := range a.GasPayers {
				if gp != p.from.String() {
					continue
				}
				appID, err := base64.StdEncoding.DecodeString(a.AppID)
				if err != nil {
					p.die("stale application id %q: %v", a.AppID, err)
				}
				found++
				p.ok("RevokeGasPayer (stale binding)", &types.MsgRevokeGasPayer{
					Owner: a.Owner, AppId: appID, GasPayer: gp,
				})
			}
		}
		if doc.Pagination.NextKey == "" {
			break
		}
		nextKey = doc.Pagination.NextKey
	}
	if found > 0 {
		fmt.Printf("     cleared %d stale gas-payer binding(s) from earlier runs\n", found)
	}
}

// stateIs compares a proto JSON enum value, which the gateway renders with its
// full enum name (ENTITLEMENT_STATE_CLAIMABLE), against the bare state name.
func stateIs(got, want string) bool {
	return got == want || strings.HasSuffix(got, "_"+want)
}
