# Uptick Chain — v0.5.2-rc2 Deploy Record (uptick_7777-1, test01)

- Date: 2026-11-06
- Operator: chain side (agent-assisted)
- Scope: dev/test chain test01 only. **No mainnet impact.**

## 1. Four-way identity

| Element | Value |
|---|---|
| Intended tag | `v0.5.2-rc2` (tag to be pushed manually by release owner) |
| Commit | `0fbf479a04e7c48e86e9a04b40ca03a48968e400` (uge/v1.1 HEAD, == origin) |
| Binary sha256 | `c26403e5fc23f921e18befc01e8fa6cd763bdb0f4d3daf85789815fbfd4c15f4` |
| On-chain version | `v0.5.2-rc2` (verified via `uptickd version --long` post-restart) |

Build metadata: `go1.26.10 linux/amd64`, CGO_ENABLED=1, source staged via
`git archive` into `~/uptick-build-0fbf479` on test01 (no git history needed).

## 2. What changed vs the running v0.5.2-rc1 (8529508)

18 commits. Only two carry runtime behavior:

- `70ae1da` — audit fixes (P1-1 authz denylist +6 authority msgs, P1-2 auth-nonce
  pruning, P2-1 publish verify-before-idempotency, P2-3 wire expires_at 7-day cap).
- `9eda8b1` — regenerated protobuf artifacts aligned to the pinned toolchain.

Migration requirements checked before swap: ConsensusVersion unchanged, no new
StoreKey, zero `.proto` source changes ⇒ plain binary replacement, no on-chain
migration.

## 3. Deploy steps executed

1. Source staged: `git archive HEAD | scp | tar` → `~/uptick-build-0fbf479`.
2. Built on test01 (GOTOOLCHAIN=auto pulled go1.26.10 per go.mod):
   `CGO_ENABLED=1 go build -tags=ledger -ldflags "…version/commit/builddate…" ./cmd/uptickd`.
   NOTE: a first attempt with `CGO_ENABLED=0` fails — cosmos/evm eip712 ante
   requires the cgo secp256k1 package. Keep CGO on for linux builds.
3. Backup: `~/bin/uptickd.bak-8529508` (old binary, sha 433c6ed7…).
4. Swap: `systemctl stop uptickd` → `install` → `systemctl start uptickd` (sudo).
5. Verify: service active; blocks advancing (752878 → 752886 in 15s); only log
   noise is the expected `use of closed network connection` from the old process
   shutting down its RPC listener.

## 4. Behavioral changes to communicate to integrators

1. **Wire `expires_at` cap (P2-3):** newly submitted wire ops must satisfy
   `expires_at <= now + 7 days`. Far-future signatures generated before this
   deploy are rejected with `ErrAuthMalformed` — regenerate the wire proof.
2. **PUBLISH retry (P2-1):** the idempotent same-content path now requires the
   wire signature to verify first. Full replay with the original signature is
   unaffected; unsigned probes now get `ErrUnauthorized`.
3. Legacy-format consumed auth nonces on disk are read-only compatible (pruning
   only touches the new 8-byte expiry encoding).

## 5. Follow-ups

- Push tag `v0.5.2-rc2` manually (mainline-guard will skip goreleaser by design;
  the tag stays as the four-way anchor only).
- UGE acks pending: DEC-09 receipt, DEC-11 bond-semantics sign-off.
