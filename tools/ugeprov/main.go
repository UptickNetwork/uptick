// ugeprov is an offline signing helper for the x/appmining UGE wire-v1
// protocol. It uses the in-repo wirev1 package so the canonical encoding is
// byte-identical to on-chain verification.
//
// Subcommands:
//
//	ugeprov keygen
//	    Generate an Ed25519 keypair; prints pubkey_hex / privkey_hex.
//
//	ugeprov derive [app_id_hex] [epoch] [amount] [denom] [gas_used]
//	             [qualified_tx_count] [pav_id] [evidence_root]
//	             [gas_evidence_root] [rule_version]
//	    Derive entitlement_id (EntitlementIDOf) and candidate_hash
//	    (CandidateHash) for a V1 candidate.
//
//	ugeprov sign [privkey_hex] [key=value]...
//	    Sign a wire authorization record (operation/role/... fields).
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		fatal("usage: ugeprov [keygen|derive|sign] ...")
	}
	switch os.Args[1] {
	case "keygen":
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			fatal("keygen: %v", err)
		}
		fmt.Printf("pubkey_hex=%s\nprivkey_hex=%s\n", hex.EncodeToString(pub), hex.EncodeToString(priv))

	case "derive":
		if len(os.Args) != 12 {
			fatal("derive needs 10 args: app_id_hex epoch amount denom gas_used qualified_tx_count pav_id evidence_root gas_evidence_root rule_version")
		}
		cand := wirev1.Candidate{
			SchemaVersion:    wirev1.CandidateSchemaVersion,
			AppID:            os.Args[2],
			Epoch:            os.Args[3],
			Amount:           os.Args[4],
			Denom:            os.Args[5],
			GasUsed:          os.Args[6],
			QualifiedTxCount: os.Args[7],
			PavID:            os.Args[8],
			EvidenceRoot:     os.Args[9],
			GasEvidenceRoot:  os.Args[10],
			RuleVersion:      os.Args[11],
		}
		cand.EntitlementID = wirev1.EntitlementIDOf(cand)
		hash, err := wirev1.CandidateHash(cand)
		if err != nil {
			fatal("candidate hash: %v", err)
		}
		fmt.Printf("entitlement_id=%s\ncandidate_hash=%s\n", cand.EntitlementID, hash)

	case "sign":
		if len(os.Args) < 4 {
			fatal("sign needs privkey_hex and at least one key=value pair")
		}
		priv, err := hex.DecodeString(os.Args[2])
		if err != nil || len(priv) != ed25519.PrivateKeySize {
			fatal("privkey_hex must be %d bytes hex", ed25519.PrivateKeySize)
		}
		record := map[string]string{}
		for _, kv := range os.Args[3:] {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) != 2 {
				fatal("bad key=value pair %q", kv)
			}
			record[parts[0]] = parts[1]
		}
		raw, err := wirev1.AuthorizationBytes(record)
		if err != nil {
			fatal("authorization bytes: %v", err)
		}
		sig := ed25519.Sign(ed25519.PrivateKey(priv), raw)
		digest, _ := wirev1.AuthorizationDigest(record)
		fmt.Printf("signature_base64=%s\nwire_digest=%s\n", base64.StdEncoding.EncodeToString(sig), digest)

	case "hash36":
		// convenience: print sha256 hex of the given string (for app_id / digests)
		if len(os.Args) != 3 {
			fatal("hash36 needs one string")
		}
		sum := sha256.Sum256([]byte(os.Args[2]))
		fmt.Printf("%s\n", hex.EncodeToString(sum[:]))

	default:
		fatal("unknown subcommand %q", os.Args[1])
	}
}
