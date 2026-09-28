<!--
Guiding Principles:

Changelogs are for humans, not machines.
There should be an entry for every single version.
The same types of changes should be grouped.
Versions and sections should be linkable.
The latest version comes first.
The release date of each version is displayed.
Mention whether you follow Semantic Versioning.

Usage:

Change log entries are to be added to the Unreleased section under the
appropriate stanza (see below). Each entry should ideally include a tag and
the Github issue reference in the following format:

* (<tag>) \#<issue-number> message

The issue numbers will later be link-ified during the release process so you do
not have to worry about including a link manually, but you can if you wish.

Types of changes (Stanzas):

"Features" for new features.
"Improvements" for changes in existing functionality.
"Deprecated" for soon-to-be removed features.
"Bug Fixes" for any bug fixes.
"Client Breaking" for breaking CLI commands and REST routes used by end-users.
"API Breaking" for breaking exported APIs used by developers building on SDK.
"State Machine Breaking" for any changes that result in a different AppState given same genesisState and txList.

Ref: https://keepachangelog.com/en/1.0.0/
-->

# Changelog

## Unreleased

### State Machine Breaking

* (upgrade) New `v0.5.0` plan: the one-hop upgrade mainnet follows from v0.3.3. It runs the whole v0.4.0
  change set and then the v0.4.1 repairs inside ONE governance plan, where the testnet executed those as two
  plans (`v0.4.0` first, then `v0.4.1`). The handler probes its starting state from state rather than from a
  flag — the cosmos/evm `EvmCoinInfo` record (EVM store prefix 0x05), which the v0.4.0 handler writes and the
  legacy ethermint layout cannot — so one binary still serves a chain already on v0.4.x, where the v0.4.0 set
  must NOT be replayed: its legacy-pair deletion removes by `ContractOwner == OWNER_MODULE` and would take the
  live STRv2 pairs with it. A marker in the x/upgrade store makes a re-proposed or crash-restarted plan return
  early. The plan repeats the `capability` store deletion, because the store loader is keyed on the plan name.
  The governance proposal must carry the name `v0.5.0` verbatim; see `app/upgrades/v050`.

* (deps) Cosmos SDK v0.53.6 → **v0.53.8** and CometBFT v0.38.21 → **v0.38.25**, in one coordinated step. Upstream
  marks v0.53.8 a security release that has to be coordinated with the chain, and the backports genuinely move
  consensus, so the binary must not be rolled out node-by-node against a running chain. Everything below arrives
  with the version bump alone — no migration, no store change, no module consensus version bump:

  * `x/distribution` reward and commission withdrawals resolve a destination instead of letting the bank module
    fail. On a user transaction — `x/staking` now marks the context with `WithStrictWithdraw` in `Delegate`,
    `BeginRedelegate`, `Undelegate` and `CancelUnbondingDelegation` — a withdraw address in the bank module's
    blocked set is rejected with `ErrUnauthorized`, so a delegation can now fail where the withdrawal used to fall
    through. On non-transaction paths (`AfterValidatorRemoved`, `BeforeDelegationSharesModified`) the same address
    falls back to the owner and then to the community pool, emitting `withdraw_addr_redirected` with the new
    `original_withdraw_address` attribute, so indexers must expect both the event and the rerouting. That fallback
    also removes a halt: those hooks run from `BeginBlocker`/`EndBlocker`, where returning the bank error stops the
    chain.
  * `x/distribution` reads historical rewards strictly: `calculateDelegationRewardsBetween` now errors on a missing
    `ValidatorHistoricalRewards` entry, where the old read handed `nil` bytes to `Unmarshal` — no error, zero value.
    A withdrawal that used to silently credit nothing now fails, and so does `CalculateDelegationRewards` on the
    query path.
  * `x/gov` `EndBlocker`: an inactive or active proposal that fails to decode used to `return nil` out of the whole
    loop, leaving its queue key in place and skipping every proposal behind it on every block. It now fails that one
    proposal, removes its queue key, deletes it and continues.
  * `x/staking` `getBeginInfo` returns `completeNow = true` when the source validator is already gone instead of
    erroring, so a redelegation whose source was removed by `Unbond` consuming its last shares can complete.
  * `x/auth` and `crypto` add checks that reject what used to panic or slip through: `SetPubKeyDecorator` rejects a
    signer-info / public-key count mismatch, nested multisig flattening is bounded at depth 2 and breadth 32,
    multisig bit-array and signature indexing is bounds-checked (previously an index panic on attacker-controlled
    sizes), `crypto/keys/secp256k1.PubKey.UnmarshalAmino` rejects a leading byte outside `0x02`/`0x03`, and
    `CompactBitArray.GetIndex`/`SetIndex` are bounded by `Elems`. Well-formed transactions produce the same
    `AppState`; malformed ones no longer reach the compiler's luck.

  CometBFT v0.38.25 moves no protocol version: `P2PProtocol = 8` and `BlockProtocol = 11` are identical to v0.38.21
  and no wire `proto` type changed, so peer traffic stays compatible and a node can be upgraded without a
  coordinated halt *on that side*. Two node-local defaults do change and neither is a consensus rule:
  `consensus.block_time_tolerance` (new in v0.38.22, default `1m0s`, applied even when the key is absent from
  `config.toml`, and rejected by `ValidateBasic` if written as `0`) rejects any block whose header time is at or
  past the local clock plus the tolerance — see `make check-clock-skew` — and `double_sign_check_height = 1` now
  checks the previous block, which an off-by-one had stopped it from ever doing (#5668).

### Features

* (tooling) `make check-clock-skew` (`scripts/check-clock-skew.sh`): preflight a node's clock against the
  `consensus.block_time_tolerance` cometbft enforces from v0.38.22. Block time tracks the network's median clock,
  so the host that trips the check is the slow one: at the `1m0s` default, a node more than 60s behind rejects
  every block and stalls at one height. The script reports the effective tolerance (a commented-out key is
  reported as absent, not as "set"), measures the offset against independent HTTPS `Date` headers, exits 1 when the
  clock is behind by at least the tolerance, and exits 2 when nothing could be measured: an unverifiable clock is
  never reported as a pass.

### Bug Fixes

* (rpc) Restore the geth tracer engines in `app/app.go` (`eth/tracers/js`, `eth/tracers/native`). The v0.4.x tree imports neither package, so their `init()` never runs and `tracers.DefaultDirectory` stays empty: every `debug_traceCall` / `debug_traceTransaction` that names a tracer panics the node with `invalid memory address or nil pointer dereference`, including the inline JS tracer our bundler sends for ERC-7562 validation (`eth/tracers/dir.go` falls through to the nil `jsEval`). Both the v0.4.0 and the v0.4.1 release binaries are affected (`strings <bin> | grep -c RegisterJSEval` → 0; the pre-upgrade binary on the same host → 1). `app/tracer_directory_test.go` pins both imports, and `eth/tracers/js` pulls in `dop251/goja`, hence the new `go.mod`/`go.sum` entries.
  Not state machine breaking: neither package's `init()` writes state, gas or consensus data — it only registers lookups in an RPC-level directory, so mixed-version validators produce the same `AppState` for the same genesisState and txList.

* (rpc) Report the build's version, commit and build time from `web3_clientVersion`. The endpoint is served by the cosmos/evm `web3` namespace, which formats `github.com/cosmos/evm/version` — a package no build in this repository injected into (the Makefile, `.goreleaser.yml` and the rbuilder image all injected only the SDK's package), so every node answered `"Version dev ()\nCompiled at  using Go ..."`: no version number, no commit, no build time, and therefore no way for a client to tell two nodes built from different commits apart. `version/version.go` now bridges the injected metadata into that package (`Sync`, run from the package `init()` and again from `NewRootCmd`), the build injects `BuildDate` for the first time, and a binary built outside the release pipeline reports its VCS revision in place of the commit. `version/version_test.go` asserts the string the endpoint's own service method returns, `cmd/uptickd/root_version_wiring_test.go` pins the CLI bridge, and `tests/e2e/rpc_smoke_test.go` now fails a node whose `web3_clientVersion` carries neither a commit nor a build date.
  Not state machine breaking: the string is RPC/CLI metadata, and nothing it reads is consensus state.

## v0.4.1 - 2026-09-07

> v0.4.0 was never released standalone; this release ships the whole v0.4.0
> change set (Cosmos SDK v0.53.6 / CometBFT v0.38.21 / ibc-go v10.5.0 /
> cosmos-evm v0.6.2 / wasmd v0.61.14) plus the v0.4.1 fixes below. The binary
> registers both the `v0.4.0` and `v0.4.1` handlers, but a v0.3.x chain must
> still execute two governance plans IN ORDER (first `v0.4.0`, then `v0.4.1`).
> See docs/guides/upgrades/upgrade_node.md.

### State Machine Breaking

The `v0.4.1` handler repairs state left by v0.4.0; each repair is idempotent, and `v0.4.0` must run first.

* (evm) Activate the EVM static precompiles (bank, staking, distribution, ICS20, gov, slashing, bech32, p256); vesting (0x803) stays off. A governance-cleared list is indistinguishable from "never configured" and is re-activated — re-apply it after the upgrade if clearing was intended (M-6 #1).
* (ica) Enable the ICA controller submodule and carry existing controller params through the write-back.
* (feemarket) Repair the base fee 10^18 re-encoding (legacy `math.Int` vs `LegacyDec`): without it, `BeginBlock` permanently bakes in a 10^-9 wei base fee.
* (erc721) Prune the redundant UID reverse-index entries from v0.3.3-era conversions; conflicts/orphans are kept and reported deterministically.
* (security) Enforce a strict one-to-one NFT mapping for ERC721/CW721 conversions (C-1).
* (security) Fix the ERC721 IBC refund ordering, eliminating a permanent fund lock on error/timeout (C-2).
* (security) Pin class ID/contract address to the registered token pair, blocking minting into arbitrary denoms or contracts (H-1/M-01).
* (erc721) Heal self-destructed token pairs on native→ERC721 conversion: purge stale state and redeploy in one transaction.
* (erc721/cw721) Export/import per-token conversion bindings and IBC refund receivers in genesis (H-02).
* (collection) Empty optional fields mean "keep current"; new `[remove]` sentinel clears a field (M-1). Reserve the `uptick-` denom prefix for module-derived classes (M-8).
* (ante) Reject malleable high-s EIP-712 signatures (EIP-2 low-s).

### Client Breaking

* (collection) `MsgEditNFT`/`MsgUpdateNFT`/`MsgTransferNFT` treat `""` as "do not modify"; send `[remove]` to clear (M-1). `MsgIssueDenom` rejects `uptick-`-prefixed ids (M-8).

### Features

* (cli) `uptickd precheck-collection-migration`: offline, read-only scan of a database copy reporting every record that would abort the v1→v2 collection migration.
* (keplr) Accept Keplr Web3-extension EIP-712 transactions (legacy ethermint type URLs map at runtime).
* (erc721/cw721) Reject commas in denom ids; UID parser splits on the last comma (L-3).
* (app) Genesis export diagnostics: degraded exports write an atomic `<home>/export-issues.json` sidecar; a lost report exits with code 3 plus a JSON notice on stderr. The export itself never fails over the sidecar.

### Improvements

* (ante) EIP-712: copy the fee-payer signature before normalization so CheckTx cannot mutate a shared tx (M-2); scan every extension option for routing (P3-10).
* (ante/authz) Disallow authorizing software-upgrade, cancel-upgrade and IBC client update/upgrade messages.
* (erc721) Cap conversion EVM gas to the tx's remaining gas and meter it back to the SDK meter (H-01/L-6).
* (upgrade) v0.4.0 handler reports all bad legacy records at once instead of halting opaquely (L-4/P2-3); register legacy `EthAccount` and erc20 proposal types (M-7/P2-8).
* (genesis) erc721/cw721 genesis validation rejects bad per-token bindings/refund receivers instead of panicking (H-02).
* (contracts) Pin OpenZeppelin and solc; checksums + deterministic recompile gate (L-11).
* (ci) golangci-lint v2.13.2; CI on `release/**`; contract reproducibility job; govulncheck pinned to v1.7.0 with the pin asserted in both workflow and script; repaired three no-op gates (`make build` .PHONY, Swagger drift check, lint `--out-format`); added fail-closed `v040-frozen` (working-tree content pin, `fix(v040):` frontier labels, git-context probe) and `contributing-targets` gates.
* (cli) `testnet --print-mnemonic` defaults to false; `migrate` explains offline migration is unsupported.
* (erc721/cw721) Symmetric CW721 pair/refund-key cleanup; `WasmContract` query returns the pair or `NotFound`.
* (docker/build) Non-root container + healthcheck; `ledger` build tag in goreleaser; inject `AppVersion`/`GitCommit` into `uptick/version`.
* (chore) Comment normalization; rename `WasmSecurityDecorator` → `MessageSecurityDecorator`; drop the stale `x/erc721/proto` shadow dir and `statik.go`.

### Bug Fixes

* (app) `export` again writes only the genesis to stdout (server logger rebuilt on stderr); normalize ibc-go v10 voucher denom metadata so the exported genesis passes `validate-genesis` (rewrites recorded in the sidecar).
* (erc721/cw721) `ibc-transfer-erc721`/`ibc-transfer-cw721` take 7 positional args and default to a 10-minute relative timeout (G-15).
* (internft) Converted NFTs can no longer be burned via ICS-721 (unbind first); removed the blanket panic recovery.
* (ibc) Route the pre-v0.4.0 `nft-transfer` port to the ICS-721 stack.
* (erc721/cw721) Consistent validation of `TokenPairs` gRPC page requests.
* (collection) Tolerate nil-`Data` NFTs at export; bounded schema/data validation; supply invariant reports into the export diagnostics.
* (cw721) Skip the IBC refund when the module no longer owns the token instead of stranding the packet (C-2); validate `nft_ids`/`cosmos_token_ids` in `ValidateBasic`.
* (erc721) Emit EVM token ids in refund events, one per (contract, receiver); validate the NFT binding before minting/transferring so rejected conversions leave no side effects.
* (ante) Reject a blank NFT name only when a non-empty value was supplied (M-1).
* (upgrade) v0.4.0 rejects malformed legacy erc20 proposal protos instead of panicking.
* (app) Fail fast on an unparseable `upgrade-info.json`.
* (cmd) `ibc-denom` no longer reports a malformed channel id as a native denom; fail loud on an unresolvable EVM chain-id; pin consensus constants.

## v0.4.0 - Unreleased (folded into v0.4.1)

> Never released standalone; every change here ships as part of v0.4.1.

### Features

- (erc721) New `x/erc721` module: native Cosmos NFT <-> ERC721 conversion and IBC transfers.
- (cw721) New `x/cw721` module: native Cosmos NFT <-> CW721 conversion and IBC transfers.
- (evm) Migrate the EVM engine to `cosmos/evm` v0.6.2 (cosmos/go-ethereum fork `v1.16.2-cosmos-1`); time-based ChainConfig, Shanghai/Cancun/Prague at upgrade time, EIP-7702 `SetCodeTx`.
- (ibc) Upgrade ibc-go v8 → v10: capability module and `ScopedKeeper` removed, `IBCModule` signatures updated.
- (sdk) Upgrade Cosmos SDK v0.50 → v0.53 and CometBFT → v0.38.21 (gov v1, `KVStoreService` keepers, authority-based params).
- (wasm) Upgrade wasmd → v0.61.14 (wasmvm v3); Uptick defaults: 50M smart-query gas, 512 MiB cache.
- (app) Derive the EVM chain-id from genesis/`--chain-id` so `eth_chainId` matches EIP-155.
- (erc20) Replace the local `x/erc20` with cosmos/evm's, wired through the ERC20 IBC middleware.
- (upgrade) `v0.4.0` handler: legacy `EthAccount` → `BaseAccount`, legacy `ethsecp256k1` pubkey conversion, erc20 params → authority, `capability` store deleted, legacy IBC provenance removed.

### Security

- (erc721/cw721) Harden NFT bridge validation: receiver checks, token-id normalization, module-account deployment.
- (evmIBC) Provenance tracking for `OutboundConvertClassId`; prefix validation via `strings.HasPrefix`.
- (erc20) IBC hardening bundle: receiver validation on recv, atomic refund via `CacheContext`, timeout provenance cleanup, `DenomUnits` bounds, `IsDenomRegistered` by `Base` (was `Name`), `SetIBCKeeper` double-call guard, `EnableEVMHook` requires `EnableErc20`.
- (collection) `ValidateGenesis` checks `Id`; validate `TokenURI` in `MsgTransferNFT`.
- (upgrade/app) Harden the v0.4.0 state transition and legacy params migration; `ValidateTokenURI` decoration in `WasmSecurityDecorator`.

### Improvements

- (erc20) `Unmarshal` + error handling in pair iteration/migration (was `MustUnmarshal`); restore `verifyMetadata` in `RegisterCoin`; drop debug prints from production paths.

### Bug Fixes

- (erc721) Fix `ConvertNFT` against the cosmos/evm v0.6.2 `StateDB` API and base-10 token ids.
- (cw721) Fix the `WasmContract` REST path prefix to `/uptick/cw721/v1/...`.
- (erc20) Re-register `MsgConvertERC20CustomGetSigner` in tx config.
- (mempool) Normalize cosmos mempool max tx and EVM mempool block gas limit.
- (collection) Allow empty/inline `--schema` in the collection tx CLI.
- (ci) Fix golangci-lint config verify failure and pin the linter version.

### API Breaking

- (ibc) `IBCModule` handlers take the channel version; `SendPacket`/`WriteAcknowledgement` no longer take `chanCap`.
- (erc20) ERC20 registration proposals use gov v1 message types.
- (params) `x/params` deprecated; params managed through the authority.

### State Machine Breaking

- (upgrade) The `v0.4.0` upgrade is **not reversible**: the `capability` store is deleted and EVM `ChainConfig` moves to time-based activation. All validators must upgrade before the height.

## v0.3.3 - 2026-06-09

### Bug Fixes

- Harden erc20 IBC outbound refund authorization: require `MsgTransferERC20` provenance instead of user-controlled memo substrings (`v0.3.3` upgrade).
- Fix erc20 IBC error-ack rollback: record transfer provenance using the ICS-20 packet denomination path (not the bank `ibc/HASH` denom) so ERC20 supply is restored when the transfer fails.
- (evmIBC) Fix IBC error-ack double refund and missing underlying module call on timeout.
- (evmIBC) Strengthen refund identification and restrict authz executable messages.

### Improvements

- Update the `evm-nft-convert` and `wasm-nft-convert` libraries.

## v0.3.2 - 2026-04-28

### Features

- Wire the v0.3.2 (Prague) fork and adapt Uptick to go-ethereum v1.16.
- Enable Shanghai/Cancun forks and EIP-3855 (PUSH0); EVM injects the CREATE2 factory runtime.
- Add EIP-7702 `SetCodeTx` support with positive/negative test coverage.

### Improvements

- Update `ethermint` (cosmos/evm) dependency references.

## v0.3.1 - 2026-03-13

### Features

- Add the `v0.3.1` upgrade handler.

### Improvements

- Bump Cosmos SDK v0.50.11 → v0.50.14 and CometBFT 0.38.17 → 0.38.19.
- Bump go-ethereum 1.10.26 → 1.13.15, `golang.org/x/crypto` 0.36.0 → 0.45.0, `ulikunitz/xz` 0.5.11 → 0.5.14, `consensys/gnark-crypto` 0.12.1 → 0.18.1, `hashicorp/go-getter` 1.7.5 → 1.7.9, `dvsekhvalnov/jose2go` 1.6.0 → 1.7.0 and `filippo.io/edwards25519` → 1.1.1.
- Bump `@openzeppelin/contracts` 4.4.2 → 4.9.6.
- Remove `EventManager` to avoid triggering events repeatedly.
- Update README documentation.

## v0.3.0 - 2025-08-11

### Features

- Prepare the v3.0.0 upgrade compatibility path.
- Migrate EVM keepers to the updated Ethermint / cosmos/evm stack.

### Improvements

- Upgrade Cosmos SDK to v0.50.x (v0.50.11).
- Change `evm_sender` to `cosmos_sender` in conversion messages.
- Fix address type conversion and vesting exception handling.
- Update tx proto, swagger spec and module registration info.

## v0.2.19 - 2024-02-26

- Fix token pair search and empty params handling.
- Fix EVM-NFT and wasm-NFT conversion for v0.2.19.

## v0.2.18 - 2024-02-03

- Add upgrade logic for the `ibcnfttransfertypes` module.
- Fix the v0.2.18 upgrade problem.

## v0.2.17 - 2024-01-19

### Features

- Add wasm contract conversion support (native NFT <-> wasm/CW721) and fix the wasm download flow.
- Add module settings for crisis, consensus params and IBC nft-transfer types in the upgrade process.

### Bug Fixes

- Fix cross-chain multi-hop conversion bugs and docker image naming.
- Remove stale wasm configuration files and debug logs.

## v0.2.16 - 2024-01-10

### Features

- Implement ICS721 → EVM conversion and cross-chain data queries.
- Add mainnet v0.2.15/v0.2.16 upgrade handling: IBC transfer module, version checks, nft-transfer params init/upgrade and legacy gov router fix.
- Implement ics721 and complete wasm <-> native NFT and Cosmos NFT <-> EVM <-> wasm triangular conversion (v0.2.12 work, folded here).

### Improvements

- Upgrade to Cosmos SDK v0.47.5.

### Bug Fixes

- Fix class ID containing IBC path information and change `-packet-memo` to JSON format.
- Fix nft-transfer parameter and crisis module settings; fix the v0.2.16 upgrade bug.

> Note: no standalone local tags exist for v0.2.12–v0.2.15; their changes are folded into v0.2.16. v0.2.15-origin / v0.2.16-origin are origin-network-specific tags and are not listed separately.

## v0.2.11 - 2023-08-28

### Features

- Add the wasm module.
- Add the Interchain Accounts (ICA) module (`icahost` types).

### Bug Fixes

- Fix executing authz `MsgDelegate` and add missing codec on ante handler options.

## v0.2.8 - 2023-06-12

### Features

- Upgrade `nft-transfer` to v1.1.2-beta and add NFT/IBC Swagger docs.
- Add nft module manager; add v0.2.4-hotfix release (min commission > 0 and IBC client handler).

### Bug Fixes

- Fix NFT id checks, token id setting and EVM <-> Cosmos conversion bugs.
- Fix `getNftDatas` and IBC client update bugs.

### Improvements

- Update Cosmos SDK to v0.46.13 to fix the Barberry security vulnerability; remove the ICS721 module.

> Note: v0.2.4-hotfix and v0.2.7 have no standalone local tags; their changes are folded into v0.2.8.

## v0.2.6 - 2023-02-22

- Add initial nft-transfer genesis handling and parameter changes.
- Clean up log sources and package the v0.2.6 release.

## v0.2.5 - 2023-02-17

### Features

- Add bidirectional EVM <-> Cosmos NFT conversion (ERC721 <-> native NFT).
- Merge GON NFT into the Uptick collection and add NFT cross-chain transfer for Iris (`nft-transfer` v1.1.1-beta).
- Add module account so conversion uses `safeTransferFrom` instead of burn.

### Bug Fixes

- Fix EVM <-> Cosmos format conversion and conversion permission issues.
- Fix compatibility with original ERC721 and GON test empty-string handling.
- Fix compile version and release packaging for v0.2.5.

## v0.2.4 - 2022-11-07

### Bug Fixes

- Fix state sync bugs that caused node crashes.
- Unify binary package file naming/format and remove redundant logs.
- Modify the testnet script generation path.

### Improvements

- Apply the ICS23 patch 0.8 for security and add the ICS23 logic package.
- Update testnet documentation.

## v0.2.3 - 2022-10-12

### Features

- Release the testnet phase-2 basic functionality: IBC cross-chain transfer of Cosmos denoms, Cosmos denom <-> ERC20 conversion, chain-id `uptick_7000-1`, IBC proposal naming rules and decimal digit setting logic.
- Add the `v0.2.3` upgrade handler for Cosmovisor upgrades.
- Release binary packages for macOS and Linux (v0.2.1 / v0.2.2).

## [v0.2.0] - 2022-05-09

### Features

- [\#21](https://github.com/UptickNetwork/uptick/issues/21) Convert to ERC20 on receiving IBC token.

### Improvements

- [\#20](https://github.com/UptickNetwork/uptick/issues/20) Bump Cosmos SDK
  to [`v0.45.3`](https://github.com/cosmos/cosmos-sdk/releases/tag/v0.45.3).
- [\#20](https://github.com/UptickNetwork/uptick/issues/20) Bump Ethermint
  to [`v0.14.0`](https://github.com/evmos/ethermint/releases/tag/v0.14.0).
- [\#40](https://github.com/UptickNetwork/uptick/pull/40) Improve ERC20 module.


## [v0.1.0] - 2022-03-18

- Build Uptick NFT infrastructure.
