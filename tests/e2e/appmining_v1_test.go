package e2e

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/evm/crypto/ethsecp256k1"

	"github.com/UptickNetwork/uptick/app/params"
	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

// This file gives x/appmining V1 what the collection lifecycle has next door:
// a real-node write path. keeper tests construct their own keepers with a mock
// bank and call msgServer directly, so they never see the TxDecoder, the ante
// handler, fee deduction or a committed store. Everything here is signed with
// SIGN_MODE_DIRECT by the localnet key and broadcast over the REST gateway.
//
// Scope note: the challenge window is a hardcoded 7-day wall-clock constant
// (types.ChallengeWindowSeconds), so the approve/claim/freeze success paths
// cannot complete inside an e2e run. They are covered by keeper tests with
// controllable timestamps and already have signed evidence on the release
// chain (three approvals, one revocation, two claims at uge/v1.1 b24c7a1).
// What this suite proves on a real node: the authority-gated setup msgs, the
// publisher/reviewer wire-signature pipeline, the challenge lifecycle, and
// that approve/claim/freeze fail closed at the tx layer before the window
// elapses.

// Wire keys for the five entitlement roles. Public deterministic fixtures —
// registered on-chain by the test itself, never production keys.
func appminingWireKey(seed byte) ed25519.PrivateKey {
	return ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, ed25519.SeedSize))
}

func appminingSignWire(t *testing.T, priv ed25519.PrivateKey, record map[string]string) types.WireProof {
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

// signAppminingBroadcast mirrors signDirectAndBroadcast but returns the commit
// result instead of asserting a zero code: the negative cases below rely on a
// real tx failing with a specific appmining error.
func signAppminingBroadcast(
	t *testing.T,
	enc params.EncodingConfig,
	rest string,
	privKey cryptotypes.PrivKey,
	pubKey *ethsecp256k1.PubKey,
	from sdk.AccAddress,
	msg sdk.Msg,
	gas uint64,
) (txHash string, code int, rawLog string) {
	t.Helper()

	accNum, seq := queryAccount(t, rest, from.String())

	msgAny, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	pubAny := newLegacyAny(t, "/ethermint.crypto.v1.ethsecp256k1.PubKey", &ethsecp256k1.PubKey{Key: pubKey.Bytes()})

	bodyBz, err := enc.Codec.Marshal(&txtypes.TxBody{Messages: []*codectypes.Any{msgAny}})
	require.NoError(t, err)
	authInfoBz, err := enc.Codec.Marshal(&txtypes.AuthInfo{
		SignerInfos: []*txtypes.SignerInfo{{
			PublicKey: pubAny,
			ModeInfo: &txtypes.ModeInfo{
				Sum: &txtypes.ModeInfo_Single_{
					Single: &txtypes.ModeInfo_Single{Mode: signing.SignMode_SIGN_MODE_DIRECT},
				},
			},
			Sequence: seq,
		}},
		Fee: &txtypes.Fee{
			Amount:   sdk.NewCoins(sdk.NewCoin(e2eDenom, e2eFeeAmount)),
			GasLimit: gas,
		},
	})
	require.NoError(t, err)
	signDocBz, err := enc.Codec.Marshal(&txtypes.SignDoc{
		BodyBytes:     bodyBz,
		AuthInfoBytes: authInfoBz,
		ChainId:       e2eChainID,
		AccountNumber: accNum,
	})
	require.NoError(t, err)
	sig, err := privKey.Sign(signDocBz)
	require.NoError(t, err)
	txRawBz, err := enc.Codec.Marshal(&txtypes.TxRaw{
		BodyBytes:     bodyBz,
		AuthInfoBytes: authInfoBz,
		Signatures:    [][]byte{sig},
	})
	require.NoError(t, err)

	txHash = broadcastTx(t, rest, base64.StdEncoding.EncodeToString(txRawBz))
	require.Eventuallyf(t, func() bool {
		var ok bool
		code, _, ok = appminingTxResult(t, rest, txHash)
		return ok
	}, 20*time.Second, 500*time.Millisecond, "%T tx %s was never committed", msg, txHash)
	return txHash, code, rawLogOf(t, rest, txHash)
}

// appminingTxResult returns (code, height, true) once the tx is committed.
func appminingTxResult(t *testing.T, rest, txHash string) (int, string, bool) {
	t.Helper()
	resp := appminingGet(t, rest, "/cosmos/tx/v1beta1/txs/"+txHash)
	if resp.status != 200 {
		return 0, "", false
	}
	var parsed struct {
		TxResponse struct {
			Code   int    `json:"code"`
			Height string `json:"height"`
		} `json:"tx_response"`
	}
	require.NoError(t, json.Unmarshal(resp.body, &parsed))
	return parsed.TxResponse.Code, parsed.TxResponse.Height, parsed.TxResponse.Height != ""
}

func rawLogOf(t *testing.T, rest, txHash string) string {
	t.Helper()
	resp := appminingGet(t, rest, "/cosmos/tx/v1beta1/txs/"+txHash)
	var parsed struct {
		TxResponse struct {
			RawLog string `json:"raw_log"`
		} `json:"tx_response"`
	}
	if resp.status != 200 || json.Unmarshal(resp.body, &parsed) != nil {
		return ""
	}
	return parsed.TxResponse.RawLog
}

// appminingTxEvents extracts a string attribute value from a committed tx's
// events, e.g. the challenge_id minted by MsgOpenChallenge.
func appminingTxEvents(t *testing.T, rest, txHash, eventType, attrKey string) string {
	t.Helper()
	resp := appminingGet(t, rest, "/cosmos/tx/v1beta1/txs/"+txHash)
	require.Equal(t, 200, resp.status)
	var parsed struct {
		TxResponse struct {
			Events []struct {
				Type       string `json:"type"`
				Attributes []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"attributes"`
			} `json:"events"`
		} `json:"tx_response"`
	}
	require.NoError(t, json.Unmarshal(resp.body, &parsed))
	for _, ev := range parsed.TxResponse.Events {
		if ev.Type != eventType {
			continue
		}
		for _, attr := range ev.Attributes {
			if attr.Key == attrKey {
				return attr.Value
			}
		}
	}
	return ""
}


// appminingURLSafeAppID hashes seed into 32 bytes whose standard-base64 form
// contains only [A-Za-z0-9=]: the gateway passes bytes path params through a
// URL decode where '+' becomes a space and '/' splits the route.
func appminingURLSafeAppID(seed []byte) []byte {
	for i := int64(0); ; i++ {
		digest := sha256.Sum256(append(append([]byte{}, seed...), []byte(strconv.FormatInt(i, 10))...))
		if !strings.ContainsAny(base64.StdEncoding.EncodeToString(digest[:]), "+/") {
			return digest[:]
		}
	}
}

type appminingResp struct {
	status int
	body   []byte
}

func appminingGet(t *testing.T, rest, path string) appminingResp {
	t.Helper()
	httpResp, err := http.Get(rest + path)
	require.NoError(t, err)
	defer httpResp.Body.Close()
	body, err := io.ReadAll(httpResp.Body)
	require.NoError(t, err)
	return appminingResp{status: httpResp.StatusCode, body: body}
}

// appminingQueryField fetches <container>.<field> from a query response (the
// gateway wraps each response message: {"application": {...}}, {"pool": ...}).
func appminingQueryField(t *testing.T, rest, path, container, field string) string {
	t.Helper()
	resp := appminingGet(t, rest, path)
	require.Equal(t, 200, resp.status, "GET %s", path)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(resp.body, &doc))
	obj, ok := doc[container].(map[string]any)
	require.True(t, ok, "GET %s: no %q object in %s", path, container, string(resp.body)[:min(200, len(resp.body))])
	return fmt.Sprintf("%v", obj[field])
}

func TestAppminingV1EntitlementLifecycleOnRealNode(t *testing.T) {
	home, rest := e2eNode(t)

	enc := params.MakeEncodingConfig()
	types.RegisterInterfaces(enc.InterfaceRegistry)

	privKey := loadKeplrPrivKey(t, home, enc.Codec)
	pubKey := privKey.PubKey().(*ethsecp256k1.PubKey)
	from := sdk.AccAddress(pubKey.Address().Bytes())

	const (
		ruleVersion = "gas-reward-candidate-0.1"
		fundAmount  = "5000000"
	)
	gas := uint64(400000)

	// Every run namespaces its state (app id, pool id, wire nonces) off the
	// current time so the suite is rerunnable against a live chain without
	// colliding with earlier runs (application/pool/candidate existence is
	// permanent on-chain).
	runSalt := fmt.Sprintf("uge-e2e-%d", time.Now().UnixNano())
	appID := appminingURLSafeAppID([]byte(runSalt))
	poolID := fmt.Sprintf("e2e-pool-%x", time.Now().UnixNano())
	nonceCounter := time.Now().UnixNano()
	nextNonce := func() string {
		nonceCounter++
		return strconv.FormatInt(nonceCounter, 10)
	}

	// The e2e-localnet.sh genesis patch must have granted this key the
	// appmining params authority and enabled the V1 flow; verify before any
	// broadcast so a mis-patched genesis fails here, not three msgs deep.
	paramsResp := appminingGet(t, rest, "/uptick/appmining/v1/params")
	require.Equal(t, 200, paramsResp.status)
	var chainParams struct {
		Params struct {
			Authority                 string   `json:"authority"`
			EntitlementEnableMinFunded string  `json:"entitlement_enable_min_funded"`
			AllowedRuleVersions       []string `json:"entitlement_allowed_rule_versions"`
		} `json:"params"`
	}
	require.NoError(t, json.Unmarshal(paramsResp.body, &chainParams))
	require.Equal(t, from.String(), chainParams.Params.Authority, "appmining params authority must be the e2e key")
	require.Equal(t, "1000", chainParams.Params.EntitlementEnableMinFunded)
	require.Contains(t, chainParams.Params.AllowedRuleVersions, ruleVersion)

	// --- authority-gated setup, now signed by the e2e key directly ---

	appIDHex := hex.EncodeToString(appID)
	_, code, logStr := signAppminingBroadcast(t, enc, rest, privKey, pubKey, from,
		&types.MsgRegisterApplication{Authority: from.String(), AppId: appID, Owner: from.String(), AttestationSubmitter: from.String()}, gas)
	require.Zero(t, code, "register application failed: %s", logStr)
	// bytes path params are base64 (std, padded) in the gRPC gateway.
	appIDPath := url.PathEscape(base64.StdEncoding.EncodeToString(appID))
	require.Equal(t, from.String(), appminingQueryField(t, rest,
		"/uptick/appmining/v1/applications/"+appIDPath, "application", "owner"))

	_, code, logStr = signAppminingBroadcast(t, enc, rest, privKey, pubKey, from,
		&types.MsgOpenEntitlementPool{Authority: from.String(), PoolId: poolID, Denom: e2eDenom, Version: 1}, gas)
	require.Zero(t, code, "open entitlement pool failed: %s", logStr)

	// F-2: five distinct ed25519 wire keys, one per role.
	wireKeys := map[string]ed25519.PrivateKey{
		"pub-1":   appminingWireKey(0xb1),
		"rev-1":   appminingWireKey(0xb2),
		"apr-1":   appminingWireKey(0xb3),
		"risk-1":  appminingWireKey(0xb4),
		"audit-1": appminingWireKey(0xb5),
	}
	roles := map[string]types.EntitlementRole{
		"pub-1":   types.EntitlementRolePublisher,
		"rev-1":   types.EntitlementRoleReviewer,
		"apr-1":   types.EntitlementRoleApprover,
		"risk-1":  types.EntitlementRoleRisk,
		"audit-1": types.EntitlementRoleAudit,
	}
	for keyID, role := range roles {
		_, code, logStr = signAppminingBroadcast(t, enc, rest, privKey, pubKey, from,
			&types.MsgRegisterRoleKey{
				Authority: from.String(), Role: role, KeyId: keyID,
				Pubkey: wireKeys[keyID].Public().(ed25519.PublicKey), Version: 1,
			}, gas)
		require.Zero(t, code, "register role key %s failed: %s", keyID, logStr)
	}

	// --- business flow, all real signatures ---

	_, code, logStr = signAppminingBroadcast(t, enc, rest, privKey, pubKey, from,
		&types.MsgFundEntitlementPool{Funder: from.String(), PoolId: poolID, Amount: fundAmount}, gas)
	require.Zero(t, code, "fund entitlement pool failed: %s", logStr)
	require.Equal(t, fundAmount, appminingQueryField(t, rest,
		"/uptick/appmining/v1/pools/"+poolID, "pool", "funded"))
	require.Equal(t, fundAmount, appminingQueryField(t, rest,
		"/uptick/appmining/v1/pools/"+poolID, "pool", "available"))

	newCandidate := func(amount string) wirev1.Candidate {
		c := wirev1.Candidate{
			SchemaVersion:    wirev1.CandidateSchemaVersion,
			AppID:            appIDHex,
			Epoch:            "1",
			Amount:           amount,
			Denom:            e2eDenom,
			GasUsed:          "21000",
			QualifiedTxCount: "1",
			PavID:            "2222222222222222222222222222222222222222222222222222222222222222",
			EvidenceRoot:     "3333333333333333333333333333333333333333333333333333333333333333",
			GasEvidenceRoot:  "4444444444444444444444444444444444444444444444444444444444444444",
			RuleVersion:      ruleVersion,
		}
		c.EntitlementID = wirev1.EntitlementIDOf(c)
		return c
	}

	publish := func(c wirev1.Candidate, nonce string) string {
		hash, err := wirev1.CandidateHash(c)
		require.NoError(t, err)
		record := map[string]string{
			"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
			"chain_id": e2eChainID, "pool_id": poolID, "key_id": "pub-1",
			"nonce": nonce, "expires_at": "1893456000",
			"entitlement_id": c.EntitlementID, "app_id": c.AppID,
			"candidate_hash": hash, "rule_version": c.RuleVersion,
		}
		txHash, code, logStr := signAppminingBroadcast(t, enc, rest, privKey, pubKey, from,
			&types.MsgPublishCandidate{
				Submitter: from.String(), ChainId: e2eChainID, PoolId: poolID,
				EntitlementId: c.EntitlementID, AppId: c.AppID, Epoch: c.Epoch,
				Amount: c.Amount, Denom: c.Denom, GasUsed: c.GasUsed,
				QualifiedTxCount: c.QualifiedTxCount, PavId: c.PavID,
				EvidenceRoot: c.EvidenceRoot, GasEvidenceRoot: c.GasEvidenceRoot,
				RuleVersion: c.RuleVersion,
				Proof: appminingSignWire(t, wireKeys["pub-1"], record),
			}, gas)
		require.Zero(t, code, "publish %s failed: %s", c.EntitlementID[:12], logStr)
		require.Equal(t, "CANDIDATE_STATUS_PUBLISHED", appminingQueryField(t, rest,
			"/uptick/appmining/v1/candidates/"+c.EntitlementID, "candidate", "status"))
		return txHash
	}

	const dismissedDigest = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

	cand1 := newCandidate("31500")
	publishTx1 := publish(cand1, nextNonce())

	// Challenge cand1 and dismiss it: the reviewer's wire signature lands on
	// chain and stores the rationale digest that a bound approve would pin.
	openTx1, code, logStr := signAppminingBroadcast(t, enc, rest, privKey, pubKey, from,
		&types.MsgOpenChallenge{
			Challenger: from.String(), EntitlementId: cand1.EntitlementID,
			EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
			EvidenceDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}, gas)
	require.Zero(t, code, "open challenge failed: %s", logStr)
	challengeID := appminingTxEvents(t, rest, openTx1, "entitlement_challenge_opened", "challenge_id")
	require.NotEmpty(t, challengeID, "challenge id missing from open-challenge events")
	require.Equal(t, "ENTITLEMENT_CHALLENGE_STATUS_OPEN", appminingQueryField(t, rest,
		"/uptick/appmining/v1/entitlement_challenges/"+challengeID, "challenge", "status"))

	// The stored publish tx hash is lowercase hex; broadcastTx returns
	// uppercase. Compare case-insensitively so the report stays traceable.
	require.Equal(t, strings.ToLower(publishTx1), appminingQueryField(t, rest,
		"/uptick/appmining/v1/candidates/"+cand1.EntitlementID, "candidate", "publish_tx_hash"))

	_, code, logStr = signAppminingBroadcast(t, enc, rest, privKey, pubKey, from,
		&types.MsgResolveEntitlementChallenge{
			Submitter: from.String(), ChainId: e2eChainID, PoolId: poolID,
			ChallengeId: challengeID, Outcome: "DISMISSED",
			RationaleDigest: dismissedDigest,
			Proof: appminingSignWire(t, wireKeys["rev-1"], map[string]string{
				"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
				"chain_id": e2eChainID, "pool_id": poolID, "key_id": "rev-1",
				"nonce": nextNonce(), "expires_at": "1893456000",
				"challenge_id": challengeID, "candidate_id": cand1.EntitlementID,
				"outcome": "DISMISSED", "rationale_digest": dismissedDigest,
			}),
		}, gas)
	require.Zero(t, code, "resolve DISMISSED failed: %s", logStr)
	require.Equal(t, "ENTITLEMENT_CHALLENGE_STATUS_DISMISSED", appminingQueryField(t, rest,
		"/uptick/appmining/v1/entitlement_challenges/"+challengeID, "challenge", "status"))
	require.Equal(t, dismissedDigest, appminingQueryField(t, rest,
		"/uptick/appmining/v1/entitlement_challenges/"+challengeID, "challenge", "resolution_digest"))

	// Bound-approve before the window elapses must fail at the tx layer with
	// the specific appmining error — a real signed tx rejected by the keeper.
	approve := func(c wirev1.Candidate, digest, nonce string) (string, int, string) {
		hash, err := wirev1.CandidateHash(c)
		require.NoError(t, err)
		record := map[string]string{
			"protocol_version": "1", "operation": "APPROVE", "role": "approver",
			"chain_id": e2eChainID, "pool_id": poolID, "key_id": "apr-1",
			"nonce": nonce, "expires_at": "1893456000",
			"entitlement_id": c.EntitlementID, "app_id": c.AppID,
			"candidate_hash": hash, "challenge_resolution_hash": digest,
			"amount": c.Amount, "denom": c.Denom, "rule_version": c.RuleVersion,
		}
		return signAppminingBroadcast(t, enc, rest, privKey, pubKey, from,
			&types.MsgApproveEntitlement{
				Submitter: from.String(), ChainId: e2eChainID, PoolId: poolID,
				EntitlementId: c.EntitlementID, ChallengeResolutionHash: digest,
				Proof: appminingSignWire(t, wireKeys["apr-1"], record),
			}, gas)
	}
	_, code, logStr = approve(cand1, dismissedDigest, nextNonce())
	require.NotZero(t, code, "bound approve before window must fail")
	require.Contains(t, logStr, "challenge window not elapsed")

	// Empty-digest unbound approve is equally gated by the window.
	_, code, logStr = approve(cand1, "", nextNonce())
	require.NotZero(t, code, "empty-digest approve before window must fail")
	require.Contains(t, logStr, "challenge window not elapsed")

	// Second candidate: an UPHELD challenge must reject the candidate itself,
	// closing every approval path.
	cand2 := newCandidate("21000")
	publish(cand2, nextNonce())
	openTx2, code, logStr := signAppminingBroadcast(t, enc, rest, privKey, pubKey, from,
		&types.MsgOpenChallenge{
			Challenger: from.String(), EntitlementId: cand2.EntitlementID,
			EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
			EvidenceDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}, gas)
	require.Zero(t, code, "open challenge 2 failed: %s", logStr)
	challengeID2 := appminingTxEvents(t, rest, openTx2, "entitlement_challenge_opened", "challenge_id")
	require.NotEmpty(t, challengeID2)
	_, code, logStr = signAppminingBroadcast(t, enc, rest, privKey, pubKey, from,
		&types.MsgResolveEntitlementChallenge{
			Submitter: from.String(), ChainId: e2eChainID, PoolId: poolID,
			ChallengeId: challengeID2, Outcome: "UPHELD",
			RationaleDigest: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			Proof: appminingSignWire(t, wireKeys["rev-1"], map[string]string{
				"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
				"chain_id": e2eChainID, "pool_id": poolID, "key_id": "rev-1",
				"nonce": nextNonce(), "expires_at": "1893456000",
				"challenge_id": challengeID2, "candidate_id": cand2.EntitlementID,
				"outcome": "UPHELD", "rationale_digest": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			}),
		}, gas)
	require.Zero(t, code, "resolve UPHELD failed: %s", logStr)
	require.Equal(t, "CANDIDATE_STATUS_REJECTED", appminingQueryField(t, rest,
		"/uptick/appmining/v1/candidates/"+cand2.EntitlementID, "candidate", "status"))

	// Claim and freeze on a non-existent entitlement fail closed at the tx
	// layer (no approval ever completed, so no entitlement record exists).
	_, code, logStr = signAppminingBroadcast(t, enc, rest, privKey, pubKey, from,
		&types.MsgClaimEntitlement{Owner: from.String(), EntitlementId: cand1.EntitlementID}, gas)
	require.NotZero(t, code, "claim without approval must fail")
	require.Contains(t, logStr, "entitlement not found")

	freezeUntil := strconv.FormatInt(time.Now().Add(72*time.Hour).Unix(), 10)
	_, code, logStr = signAppminingBroadcast(t, enc, rest, privKey, pubKey, from,
		&types.MsgFreezeEntitlement{
			Submitter: from.String(), ChainId: e2eChainID, PoolId: poolID,
			EntitlementId: cand1.EntitlementID,
			EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
			EvidenceDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			FreezeUntil:    freezeUntil,
			Proof: appminingSignWire(t, wireKeys["risk-1"], map[string]string{
				"protocol_version": "1", "operation": "FREEZE", "role": "risk",
				"chain_id": e2eChainID, "pool_id": poolID, "key_id": "risk-1",
				"nonce": nextNonce(), "expires_at": "1893456000",
				"entitlement_id":  cand1.EntitlementID,
				"evidence_type":   "CONFIRMED_REFUND_OR_REVERSAL",
				"evidence_digest": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"freeze_until":    freezeUntil,
			}),
		}, gas)
	require.NotZero(t, code, "freeze without approval must fail")
	require.Contains(t, logStr, "entitlement not found")

	// Ledger reconciles with the module account: two challenge bonds may be
	// held in escrow, but no approval ever reserved anything, so the pool
	// ledger is untouched by the approve attempts above.
	poolFunded := appminingQueryField(t, rest, "/uptick/appmining/v1/pools/"+poolID, "pool", "funded")
	poolAvailable := appminingQueryField(t, rest, "/uptick/appmining/v1/pools/"+poolID, "pool", "available")
	poolReserved := appminingQueryField(t, rest, "/uptick/appmining/v1/pools/"+poolID, "pool", "reserved")
	require.Equal(t, fundAmount, poolFunded)
	require.Equal(t, fundAmount, poolAvailable)
	require.Equal(t, "0", poolReserved)
}
