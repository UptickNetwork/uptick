// Command chainprobe drives x/appmining's whole message surface against a live
// node: one real, signed transaction per Msg, plus the negative cases that must
// be rejected by the keeper rather than only by a unit test.
//
// It exists because `go test -c ./tests/e2e` cannot cross-compile: the e2e
// package pulls testutil -> wasmd keeper test helpers, and wasmd's test_common.go
// does not build against this repo's wasmd revision. A plain main package builds
// cleanly with CGO_ENABLED=0, so the node-side half of the audit ships as this
// binary instead of a test binary.
//
// It mutates the node's appmining params (short challenge window, legacy flow
// unpaused) because production defaults park the lifecycle: a 1500-block window
// plus attestation_paused=true makes "submit and finalize" unreachable in a
// short run. The original params are captured first and restored on every exit
// path, including panic.
//
//	UPTICK_CHAIN_REST=http://127.0.0.1:1317 \
//	UPTICK_CHAIN_ID=origin_1170-3 UPTICK_CHAIN_DENOM=auoc \
//	UPTICK_CHAIN_HOME=/home/ubuntu/.uptickd UPTICK_CHAIN_KEY=authority \
//	UPTICK_CHAIN_OUT=/tmp/appmining-chain-surface.json \
//	./chainprobe
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
	enccodec "github.com/cosmos/evm/encoding/codec"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// init sets the bech32 prefixes the chain uses. It is spelled out here instead
// of importing cmd/config because that package pulls the whole app (and with it
// wasmd's keeper package, whose test_common.go does not compile against this
// repo's wasmd revision) — a signing client needs none of that.
// newCodec builds the protobuf codec the signer needs, without importing
// app/params: that package builds on app/upgrades, which pulls the entire app
// and with it wasmd's keeper (whose test_common.go does not compile here).
// Signing only needs proto marshalling, the appmining registrations and the
// keyring, none of which require the app.
func newCodec() codec.Codec {
	ir := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(ir)
	// The keyring record holds an Any-wrapped ethsecp256k1 private key; without
	// these registrations the Any cannot be unpacked and reading the key fails
	// with a length-prefix error that points nowhere near the real cause.
	enccodec.RegisterInterfaces(ir)
	types.RegisterInterfaces(ir)
	return cdc
}

func init() {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("uptick", "uptickpub")
	cfg.SetBech32PrefixForValidator("uptickvaloper", "uptickvaloperpub")
	cfg.SetBech32PrefixForConsensusNode("uptickvalcons", "uptickvalconspub")
	cfg.SetCoinType(60)
	cfg.Seal()
}

// --- configuration -------------------------------------------------------

type config struct {
	rest    string
	chainID string
	denom   string
	home    string
	keyName string
	out     string
	fee     math.Int
	gas     uint64
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}

func loadConfig() config {
	return config{
		rest:    envOr("UPTICK_CHAIN_REST", "http://127.0.0.1:1317"),
		chainID: envOr("UPTICK_CHAIN_ID", "origin_1170-3"),
		denom:   envOr("UPTICK_CHAIN_DENOM", "auoc"),
		home:    envOr("UPTICK_CHAIN_HOME", "/home/ubuntu/.uptickd"),
		keyName: envOr("UPTICK_CHAIN_KEY", "authority"),
		out:     envOr("UPTICK_CHAIN_OUT", "/tmp/appmining-chain-surface.json"),
		fee:     math.NewInt(envIntOr("UPTICK_CHAIN_FEE", 10_000_000_000_000_000)),
		gas:     uint64(envIntOr("UPTICK_CHAIN_GAS", 1_500_000)),
	}
}

// --- evidence ------------------------------------------------------------

type step struct {
	Group   string `json:"group"`
	Name    string `json:"name"`
	Msg     string `json:"msg"`
	Expect  string `json:"expect"` // OK | REJECT
	TxHash  string `json:"tx_hash,omitempty"`
	Code    int    `json:"code"`
	Height  string `json:"height,omitempty"`
	GasUsed string `json:"gas_used,omitempty"`
	TxBytes int    `json:"tx_bytes,omitempty"`
	Passed  bool   `json:"passed"`
	Note    string `json:"note,omitempty"`
	Elapsed string `json:"elapsed,omitempty"`
}

type poolSnapshot struct {
	Funded    string `json:"funded"`
	Available string `json:"available"`
	Reserved  string `json:"reserved"`
	Paid      string `json:"paid"`
}

type report struct {
	ChainID    string         `json:"chain_id"`
	Denom      string         `json:"denom"`
	Signer     string         `json:"signer"`
	AppID      string         `json:"app_id"`
	PoolID     string         `json:"pool_id"`
	StartedAt  string         `json:"started_at"`
	FinishedAt string         `json:"finished_at,omitempty"`
	Aborted    string         `json:"aborted,omitempty"`
	Pool       poolSnapshot   `json:"pool"`
	Steps      []step         `json:"steps"`
	Summary    map[string]int `json:"summary"`
}

// --- probe ---------------------------------------------------------------

type probe struct {
	cfg  config
	cdc  codec.Codec
	priv cryptotypes.PrivKey
	pub  *ethsecp256k1.PubKey
	from sdk.AccAddress

	report   report
	orig     types.Params
	haveOrig bool
	restored bool
	group    string
}

// abort unwinds so main can restore params before the process exits.
type abort struct{ reason string }

func (p *probe) die(format string, args ...any) {
	panic(abort{reason: fmt.Sprintf(format, args...)})
}

func (p *probe) groupf(name string) { p.group = name }

// --- http plumbing -------------------------------------------------------

type httpResp struct {
	status int
	body   []byte
}

var client = &http.Client{Timeout: 20 * time.Second}

func (p *probe) get(path string) httpResp {
	resp, err := client.Get(p.cfg.rest + path)
	if err != nil {
		p.die("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.die("GET %s: read body: %v", path, err)
	}
	return httpResp{status: resp.StatusCode, body: body}
}

func truncate(b []byte) string {
	if len(b) > 400 {
		return string(b[:400]) + "..."
	}
	return string(b)
}

func (p *probe) view(path, container, field string) string {
	resp := p.get(path)
	if resp.status != 200 {
		p.die("GET %s: status %d: %s", path, resp.status, truncate(resp.body))
	}
	var doc map[string]any
	if err := json.Unmarshal(resp.body, &doc); err != nil {
		p.die("GET %s: decode: %v", path, err)
	}
	obj, ok := doc[container].(map[string]any)
	if !ok {
		p.die("GET %s: no %q object in %s", path, container, truncate(resp.body))
	}
	return fmt.Sprintf("%v", obj[field])
}

func (p *probe) account() (accNum, seq uint64) {
	resp := p.get("/cosmos/auth/v1beta1/accounts/" + p.from.String())
	if resp.status != 200 {
		p.die("account query: status %d: %s", resp.status, truncate(resp.body))
	}
	// The account may be flattened or wrapped in an Any with base_account,
	// depending on the auth module version; accept both shapes.
	var doc struct {
		Account struct {
			AccountNumber string `json:"account_number"`
			Sequence      string `json:"sequence"`
			BaseAccount   struct {
				AccountNumber string `json:"account_number"`
				Sequence      string `json:"sequence"`
			} `json:"base_account"`
		} `json:"account"`
	}
	if err := json.Unmarshal(resp.body, &doc); err != nil {
		p.die("account query: decode: %v", err)
	}
	accStr, seqStr := doc.Account.AccountNumber, doc.Account.Sequence
	if accStr == "" {
		accStr = doc.Account.BaseAccount.AccountNumber
	}
	if seqStr == "" {
		seqStr = doc.Account.BaseAccount.Sequence
	}
	n, err := strconv.ParseUint(accStr, 10, 64)
	if err != nil {
		p.die("account query: account_number %q: %v", accStr, err)
	}
	s, err := strconv.ParseUint(seqStr, 10, 64)
	if err != nil {
		p.die("account query: sequence %q: %v", seqStr, err)
	}
	return n, s
}

type txResult struct {
	hash    string
	code    int
	rawLog  string
	height  string
	gasUsed string
	txBytes int
}

// signAndBroadcast builds a SIGN_MODE_DIRECT tx for one message, broadcasts it
// and waits for commit. A CheckTx rejection is returned as a non-zero code as
// well, so callers can assert that a keeper error surfaced on chain.
func (p *probe) signAndBroadcast(msg sdk.Msg) txResult {
	accNum, seq := p.account()

	msgAny, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		p.die("%T: pack any: %v", msg, err)
	}
	pubAny, err := codectypes.NewAnyWithValue(&ethsecp256k1.PubKey{Key: p.pub.Bytes()})
	if err != nil {
		p.die("pack pubkey: %v", err)
	}
	// Keep the native type URL produced by the concrete pubkey type
	// (/cosmos.evm.crypto.v1.ethsecp256k1.PubKey on this chain): that is what the
	// keyring holds and what the ante handler's own decoder expects.
	bodyBz, err := p.cdc.Marshal(&txtypes.TxBody{Messages: []*codectypes.Any{msgAny}})
	if err != nil {
		p.die("%T: marshal body: %v", msg, err)
	}
	authBz, err := p.cdc.Marshal(&txtypes.AuthInfo{
		SignerInfos: []*txtypes.SignerInfo{{
			PublicKey: pubAny,
			ModeInfo: &txtypes.ModeInfo{Sum: &txtypes.ModeInfo_Single_{
				Single: &txtypes.ModeInfo_Single{Mode: signing.SignMode_SIGN_MODE_DIRECT},
			}},
			Sequence: seq,
		}},
		Fee: &txtypes.Fee{
			Amount:   sdk.NewCoins(sdk.NewCoin(p.cfg.denom, p.cfg.fee)),
			GasLimit: p.cfg.gas,
		},
	})
	if err != nil {
		p.die("%T: marshal authinfo: %v", msg, err)
	}
	signDocBz, err := p.cdc.Marshal(&txtypes.SignDoc{
		BodyBytes: bodyBz, AuthInfoBytes: authBz,
		ChainId: p.cfg.chainID, AccountNumber: accNum,
	})
	if err != nil {
		p.die("%T: marshal signdoc: %v", msg, err)
	}
	sig, err := p.priv.Sign(signDocBz)
	if err != nil {
		p.die("%T: sign: %v", msg, err)
	}
	txRawBz, err := p.cdc.Marshal(&txtypes.TxRaw{
		BodyBytes: bodyBz, AuthInfoBytes: authBz, Signatures: [][]byte{sig},
	})
	if err != nil {
		p.die("%T: marshal txraw: %v", msg, err)
	}

	payload, _ := json.Marshal(map[string]string{
		"tx_bytes": base64.StdEncoding.EncodeToString(txRawBz),
		"mode":     "BROADCAST_MODE_SYNC",
	})
	req, err := http.NewRequest(http.MethodPost, p.cfg.rest+"/cosmos/tx/v1beta1/txs", bytes.NewReader(payload))
	if err != nil {
		p.die("broadcast: build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		p.die("broadcast %T: %v", msg, err)
	}
	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		p.die("broadcast %T: read: %v", msg, err)
	}

	var syncEnvelope struct {
		TxResponse struct {
			TxHash string `json:"txhash"`
			Code   int    `json:"code"`
			RawLog string `json:"raw_log"`
		} `json:"tx_response"`
		Message string `json:"message"`
		Details string `json:"details"`
	}
	if err := json.Unmarshal(raw, &syncEnvelope); err != nil {
		p.die("broadcast %T: decode %s: %v", msg, truncate(raw), err)
	}
	if syncEnvelope.TxResponse.TxHash == "" {
		// Gateway error envelope (e.g. undecodable tx): keep it as a rejected
		// step instead of crashing, so the audit trail names the message.
		return txResult{code: 1, rawLog: "broadcast rejected: " + truncate(raw), txBytes: len(txRawBz)}
	}
	res := txResult{hash: syncEnvelope.TxResponse.TxHash, txBytes: len(txRawBz)}
	if syncEnvelope.TxResponse.Code != 0 {
		res.code = syncEnvelope.TxResponse.Code
		res.rawLog = syncEnvelope.TxResponse.RawLog
		return res
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		r := p.get("/cosmos/tx/v1beta1/txs/" + res.hash)
		if r.status == 200 {
			var out struct {
				TxResponse struct {
					Code    int    `json:"code"`
					Height  string `json:"height"`
					RawLog  string `json:"raw_log"`
					GasUsed string `json:"gas_used"`
				} `json:"tx_response"`
			}
			if json.Unmarshal(r.body, &out) == nil && out.TxResponse.Height != "" {
				res.code = out.TxResponse.Code
				res.rawLog = out.TxResponse.RawLog
				res.height = out.TxResponse.Height
				res.gasUsed = out.TxResponse.GasUsed
				return res
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
	p.die("tx %s (%T) was never committed", res.hash, msg)
	return res
}

// --- step assertions -----------------------------------------------------

func (p *probe) addStep(name string, msg sdk.Msg, expect string, res txResult, passed bool, note string, elapsed time.Duration) {
	p.report.Steps = append(p.report.Steps, step{
		Group: p.group, Name: name, Msg: fmt.Sprintf("%T", msg), Expect: expect,
		TxHash: res.hash, Code: res.code, Height: res.height, GasUsed: res.gasUsed,
		TxBytes: res.txBytes, Passed: passed, Note: note,
		Elapsed: elapsed.Round(time.Millisecond).String(),
	})
	mark := "OK  "
	if !passed {
		mark = "FAIL"
	}
	if res.code == 0 {
		mark = "OK  "
	} else if expect == "REJECT" && passed {
		mark = "OK* "
	}
	fmt.Printf("[%02d] %s %-52s code=%d h=%-6s %s\n",
		len(p.report.Steps), mark, name, res.code, res.height, res.hash)
	if note != "" && !passed {
		fmt.Printf("     └─ %s\n", note)
	}
}

// ok broadcasts a message that must be accepted.
func (p *probe) ok(name string, msg sdk.Msg) txResult {
	start := time.Now()
	res := p.signAndBroadcast(msg)
	p.addStep(name, msg, "OK", res, res.code == 0, res.rawLog, time.Since(start))
	if res.code != 0 {
		p.die("%s rejected: %s", name, res.rawLog)
	}
	return res
}

// reject broadcasts a message that must be refused by the keeper.
func (p *probe) reject(name string, msg sdk.Msg, contains string) {
	start := time.Now()
	res := p.signAndBroadcast(msg)
	passed := res.code != 0 && strings.Contains(res.rawLog, contains)
	note := res.rawLog
	if res.code != 0 && !passed {
		note = fmt.Sprintf("want %q in %q", contains, res.rawLog)
	}
	if res.code == 0 {
		note = "unexpectedly accepted"
	}
	p.addStep(name, msg, "REJECT", res, passed, note, time.Since(start))
	if !passed {
		p.die("%s: %s", name, note)
	}
}

// retryOK repeats a height- or time-gated message until the chain accepts it.
// The message is identical each attempt and the nonce is re-read from the
// account, so a retry is the same request, not a different one.
func (p *probe) retryOK(name string, msg sdk.Msg, attempts int, wait time.Duration) txResult {
	start := time.Now()
	var last txResult
	for i := 0; i < attempts; i++ {
		last = p.signAndBroadcast(msg)
		if last.code == 0 {
			p.addStep(fmt.Sprintf("%s (attempt %d)", name, i+1), msg, "OK", last, true, "", time.Since(start))
			return last
		}
		time.Sleep(wait)
	}
	p.addStep(name, msg, "OK", last, false, last.rawLog, time.Since(start))
	p.die("%s never accepted after %d attempts: %s", name, attempts, last.rawLog)
	return last
}

// --- params --------------------------------------------------------------

func (p *probe) params() types.Params {
	resp := p.get("/uptick/appmining/v1/params")
	if resp.status != 200 {
		p.die("params: status %d: %s", resp.status, truncate(resp.body))
	}
	var doc struct {
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(resp.body, &doc); err != nil {
		p.die("params: decode: %v", err)
	}
	var out types.Params
	if err := p.cdc.UnmarshalJSON(doc.Params, &out); err != nil {
		p.die("params: unmarshal: %v", err)
	}
	return out
}

func (p *probe) setParams(mutate func(*types.Params)) types.Params {
	p0 := p.params()
	mutate(&p0)
	p.ok("UpdateParams", &types.MsgUpdateParams{Authority: p.from.String(), Params: p0})
	return p0
}

func (p *probe) restoreParams() {
	if p.restored || !p.haveOrig {
		return
	}
	p.restored = true
	res := p.signAndBroadcast(&types.MsgUpdateParams{Authority: p.from.String(), Params: p.orig})
	if res.code != 0 {
		fmt.Printf("WARNING: could not restore appmining params: %s\n", res.rawLog)
		return
	}
	fmt.Printf("restored appmining params (tx %s)\n", res.hash)
}

// --- tx events -----------------------------------------------------------

func (p *probe) txEvent(hash, eventType, attrKey string) string {
	resp := p.get("/cosmos/tx/v1beta1/txs/" + hash)
	if resp.status != 200 {
		p.die("tx events %s: status %d", hash, resp.status)
	}
	var out struct {
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
	if err := json.Unmarshal(resp.body, &out); err != nil {
		p.die("tx events %s: decode: %v", hash, err)
	}
	for _, ev := range out.TxResponse.Events {
		if ev.Type != eventType {
			continue
		}
		for _, a := range ev.Attributes {
			if a.Key == attrKey {
				return a.Value
			}
		}
	}
	return ""
}

// --- helpers -------------------------------------------------------------

func containsStr(list []string, want string) bool {
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

// urlSafeAppID hashes seed into 32 bytes whose standard-base64 form contains
// only [A-Za-z0-9=]: the gateway runs bytes path params through a URL decode
// where '+' becomes a space and '/' splits the route.
func urlSafeAppID(seed []byte) []byte {
	for i := int64(0); ; i++ {
		digest := sha256.Sum256(append(append([]byte{}, seed...), []byte(strconv.FormatInt(i, 10))...))
		if !strings.ContainsAny(base64.StdEncoding.EncodeToString(digest[:]), "+/") {
			return digest[:]
		}
	}
}

func loadKey(cfg config, cdc codec.Codec) (cryptotypes.PrivKey, *ethsecp256k1.PubKey, error) {
	kr, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, cfg.home, nil, cdc)
	if err != nil {
		return nil, nil, fmt.Errorf("open keyring %s: %w", cfg.home, err)
	}
	rec, err := kr.Key(cfg.keyName)
	if err != nil {
		return nil, nil, fmt.Errorf("no %q key in keyring %s: %w", cfg.keyName, cfg.home, err)
	}
	local := rec.GetLocal()
	if local == nil {
		return nil, nil, fmt.Errorf("%q is not a local key", cfg.keyName)
	}
	priv, ok := local.PrivKey.GetCachedValue().(cryptotypes.PrivKey)
	if !ok {
		return nil, nil, fmt.Errorf("cannot extract private key for %q", cfg.keyName)
	}
	pub, ok := priv.PubKey().(*ethsecp256k1.PubKey)
	if !ok {
		return nil, nil, fmt.Errorf("%q is not an ethsecp256k1 key", cfg.keyName)
	}
	return priv, pub, nil
}

func main() {
	cfg := loadConfig()

	cdc := newCodec()

	priv, pub, err := loadKey(cfg, cdc)
	if err != nil {
		fmt.Printf("cannot load signing key: %v\n", err)
		os.Exit(2)
	}
	p := &probe{
		cfg:  cfg,
		cdc:  cdc,
		priv: priv,
		pub:  pub,
		from: sdk.AccAddress(pub.Address().Bytes()),
	}
	p.report = report{
		ChainID: cfg.chainID, Denom: cfg.denom, Signer: p.from.String(),
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Summary:   map[string]int{},
	}

	fmt.Printf("chainprobe: rest=%s chain=%s denom=%s signer=%s\n",
		cfg.rest, cfg.chainID, cfg.denom, p.from.String())

	// Maintenance mode: broadcast a params document verbatim. Used to put a node
	// back the way it was without hand-editing a protobuf-JSON blob.
	if file := os.Getenv("UPTICK_CHAIN_PARAMS_FILE"); file != "" {
		bz, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("read %s: %v\n", file, err)
			os.Exit(2)
		}
		var pp types.Params
		if err := cdc.UnmarshalJSON(bz, &pp); err != nil {
			fmt.Printf("decode %s: %v\n", file, err)
			os.Exit(2)
		}
		p.ok("UpdateParams (from file)", &types.MsgUpdateParams{Authority: p.from.String(), Params: pp})
		return
	}

	// Publish-one mode: broadcast a single PublishCandidate carrying the body the
	// protocol service built, so the service can later approve it. See
	// publish_one.go for why this cannot go through chainprobe's own flow.
	if file := os.Getenv("UGE_PUBLISH_CANDIDATE_FILE"); file != "" {
		func() {
			defer func() {
				if r := recover(); r != nil {
					if a, ok := r.(abort); ok {
						fmt.Printf("\nABORT: %s\n", a.reason)
						os.Exit(1)
					}
					panic(r)
				}
			}()
			p.publishFromCandidateFile(file)
		}()
		return
	}

	exit := 0
	func() {
		defer func() {
			// Always put the node's params back: a dev chain left with a 60s
			// window, a micro-deposit and the legacy flow unpaused is a trap for
			// the next person, whether this run succeeded or died midway.
			p.restoreParams()
			if r := recover(); r != nil {
				if a, ok := r.(abort); ok {
					p.report.Aborted = a.reason
					fmt.Printf("\nABORT: %s\n", a.reason)
					exit = 1
					return
				}
				p.report.Aborted = fmt.Sprintf("panic: %v", r)
				fmt.Printf("\nUNEXPECTED PANIC: %v\n", r)
				exit = 2
				panic(r)
			}
		}()
		p.run()
	}()

	p.report.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	for _, s := range p.report.Steps {
		p.report.Summary["total"]++
		if s.Expect == "OK" {
			p.report.Summary["expected_ok"]++
		} else {
			p.report.Summary["expected_reject"]++
		}
		if s.Passed {
			p.report.Summary["passed"]++
		} else {
			p.report.Summary["failed"]++
		}
		if s.Code == 0 {
			p.report.Summary["accepted_tx"]++
		} else {
			p.report.Summary["rejected_tx"]++
		}
	}

	bz, err := json.MarshalIndent(p.report, "", "  ")
	if err != nil {
		fmt.Printf("marshal report: %v\n", err)
		exit = 2
	} else if err := os.WriteFile(cfg.out, bz, 0o644); err != nil {
		fmt.Printf("write report %s: %v\n", cfg.out, err)
		exit = 2
	} else {
		fmt.Printf("\nwrote %d steps to %s\n", len(p.report.Steps), cfg.out)
		fmt.Printf("summary: %v\n", p.report.Summary)
	}

	if exit != 0 {
		os.Exit(exit)
	}
}
