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

## v0.4.1 - 2026-09-07

> **Release note**: v0.4.0 was never released standalone. This release ships the
> whole v0.4.0 change set (Cosmos SDK 0.53 / ibc-go v10 / cosmos-evm migration)
> together with the v0.4.1 fixes below, so a validator moving from v0.3.x
> performs a single upgrade covering both handlers.

### State Machine Breaking

* (security) Enforce a strict one-to-one NFT mapping for ERC721/CW721 conversions so a caller's own NFT can no longer release a module-escrowed token bound to a different NFT (C-1).
* (security) Fix the ERC721 IBC refund ordering: release the IBC-escrowed NFT to the module account before reversing the conversion, eliminating a permanent fund lock on error/timeout (C-2).
* (security) Pin the resolved class ID and contract address to the registered token pair, blocking minting into arbitrary third-party denoms or external compatible contracts (H-1 / M-01).
* (erc721) Heal a self-destructed token pair on the native→ERC721 conversion path: the stale pair and its per-token bindings/refund records are purged and a fresh module contract is deployed in the same successful transaction (the previous cleanup ran inside an erroring handler, where the SDK rolls all writes back, so the pair stayed stuck forever). Classes pinned to an external contract (`uptick-<addr>`) remain a terminal error.
* (erc721/cw721) Export and import the per-token conversion bindings and IBC refund receivers in genesis (`nft_uid_pairs`, `refund_receivers`) so reverse conversions and refunds survive an export/import round-trip (H-02).
* (collection) Empty optional fields now mean "keep the current value"; the new `[remove]` sentinel explicitly clears a field (M-1). `[remove]` is a reserved literal.
* (collection) Reserve the `uptick-` denom prefix for module-derived NFT classes; `MsgIssueDenom` rejects it (M-8 griefing protection).
* (ante) Reject malleable high-s EIP-712 signatures (EIP-2 low-s requirement).

### Client Breaking

* (collection) `MsgEditNFT` / `MsgUpdateNFT` / `MsgTransferNFT` treat an empty string as "do not modify"; clients that previously cleared a field by sending `""` must send `[remove]` instead (M-1).
* (collection) `MsgIssueDenom` no longer accepts `uptick-`-prefixed denom ids (M-8).

### Features

* (evm) Activate the EVM static precompiles (bank, staking, distribution, ICS20, gov, slashing, bech32, p256) on upgrade and for fresh chains — v0.4.0 shipped the `ActiveStaticPrecompiles` param empty (M-6 #1).

  The migration writes only when the stored list is **empty**, and it reads an empty list as "never configured". Read that together with the proto round-trip: an explicitly empty list collapses to nil on the way through the store, so a governance decision to clear the list (disable every static precompile) is indistinguishable from the untouched default, and the v0.4.1 upgrade re-activates the shipped set over it. Re-apply the governance change after the upgrade if clearing was intentional. The vesting precompile (0x803) is deliberately **absent** from the set — Uptick does not register it, and activating it panics the EVM with "precompiled contract not stored in memory".
* (ibc) Enable the ICA controller submodule on upgrade (legacy genesis templates default it to disabled).
* (keplr) Accept Keplr Web3-extension EIP-712 transactions: legacy `/ethermint.types.v1.ExtensionOptionsWeb3Tx` and `/ethermint.crypto.v1.ethsecp256k1.PubKey` type URLs map onto the complete cosmos/evm types at runtime (v0.4.1 Keplr compatibility).
* (erc721/cw721) Reject commas in denom ids and make the NFT-mapping UID parser split on the last comma, keeping NFT UID round-trips safe (L-3).
* (cli) Add `uptickd precheck-collection-migration`, an offline, read-only pre-upgrade check: it scans a pre-upgrade database copy with the collection module's own v1→v2 legacy-store precheck and reports every record that would abort the migration, so an operator sees the whole problem before scheduling the upgrade. It migrates nothing and writes no state; a clean store exits 0, a dirty one exits non-zero with the full report.

### Improvements

* (ante) EIP-712 signature verification copies the fee-payer signature before recovery-offset normalization so CheckTx can no longer mutate a shared mempool transaction; bind the signer pubkey self-containedly (M-2); scan every tx extension option for routing (P3-10).
* (ante/authz) Disallow authorizing software-upgrade, cancel-upgrade and IBC client update/upgrade messages through authz `MsgExec` (nested `MsgExec` cannot bypass the limiter).
* (erc721) Cap the internal EVM gas budget of conversion calls to the SDK transaction's remaining gas and meter used EVM gas back to the SDK gas meter (H-01/L-6).
* (upgrade) v0.4.0 handler: fail loudly with a complete report instead of an opaque halt when the legacy collection store contains records that would abort the 1→2 migration (L-4); collect every failing legacy auth account in one pass (P2-3).
* (upgrade) Register legacy `EthAccount` and erc20 proposal types for genesis bootstrap/export and legacy pubkey retention (M-7/P2-8).
* (genesis) erc721/cw721 genesis validation rejects empty, duplicated, or pair-unrelated per-token bindings and refund receivers instead of panicking in `InitGenesis` (H-02 follow-up).
* (contracts) Pin OpenZeppelin and solc; add checksums and deterministic recompile + artifact-integrity verification to CI (L-11, R-NEW-3).
* (ci) golangci-lint v2.13.2 (v2 config schema); run CI on every `release/**` push; drop the flaky e2e-keplr-eip712 job; contract bytecode reproducibility job.
* (ci) Repair the gates that returned 0 while checking nothing and add two fail-closed ones. Repaired: `make build` was skipped by Make once `build/` existed (missing `.PHONY`); the Swagger drift check used a single-`$` Make reference so it was always "in sync"; `make lint` no longer passes the golangci-lint v2-removed `--out-format`. Added: a `v040-frozen` job that pins the blob shas of the frozen `app/upgrades/v040/` tree in `scripts/v040-frozen.sha256`, read from the working tree (so an uncommitted edit is caught), failing on any content drift that is not accompanied by a matching re-pin, and requiring every surviving (frontier) change to the pin to be a `fix(v040):` commit so that a later re-pin clears an earlier slip (`make v040-frozen-manifest` re-pins it) -- the sole exemption is a base that has no pin (the window is keyed on the pin's absence at base, not on its first creation, so a base that is itself a post-deletion pin-absent commit reopens it); once the pin exists at base, any in-range commit that re-adds or rewrites the pin must be labelled, including a delete hidden inside a merge; and a `contributing-targets` job that fails when `CONTRIBUTING.md` documents a `make` target the Makefile does not define (the documented `make test-import` gate never existed). Test failures can no longer be masked by the `tparse` pipe in `run-tests`.
* (cli) Default `uptickd testnet --print-mnemonic` to `false`.
* (docker) Run as a non-root user and add a healthcheck.
* (build) Add the `ledger` build tag to goreleaser artifacts.
* (erc721/cw721) Document the per-transaction commit semantics of pair/per-token cleanup; give CW721 a symmetric `DeletePairPerTokenState` and case-insensitive (bech32) refund-key handling in genesis export/validation; make the `WasmContract` query return the registered pair or `NotFound` instead of an empty success.
* (cli) `uptickd migrate` explains that offline genesis migration is unsupported (upgrade in-app) instead of reporting an opaque "unknown migration function".
* (chore) Normalize comments across `app/` and `x/`: English-only, functional descriptions with audit-tracking references removed. Rename the ante `WasmSecurityDecorator` to `MessageSecurityDecorator` to match its actual scope (CosmWasm validation + authz flattening + EVM gas checks); move the erc721 ID-resolution helpers from `keeper/params.go` to `keeper/nft_data.go` (mirroring x/cw721). Drop the stale `x/erc721/proto` shadow directory — `proto/uptick` is the source of truth for `make proto-gen` — and the unreferenced `client/docs/uptick/statik.go` artifact.

### Bug Fixes

* (erc721/cw721) `ibc-transfer-erc721` / `ibc-transfer-cw721` accept exactly 7 positional arguments and fall back to a 10 minute relative timeout timestamp when both `--packet-timeout-height` and `--packet-timeout-timestamp` are left at their zero defaults, so the default invocation no longer fails msg validation with "timeout height and timeout timestamp cannot both be zero"; explicit `--absolute-timeouts=false` (relative timeouts against the counterparty state) is rejected as unsupported after the ibc-go v10 migration (G-15).
* (collection) Tolerate nil-`Data` NFTs during genesis export; reject NUL/comma and over-length schema/data inputs with bounded limits.
* (ante) Reject an empty/blank NFT name only when a non-empty value was supplied (M-1).
* (cw721) Skip the CW721 IBC refund when the module account no longer owns the token instead of returning an error: an error aborts the `OnTimeout`/`OnAcknowledgement` callback, after which the relayer's `MsgTimeout`/`MsgAcknowledgement` can never succeed and the packet is stranded (CW721 half of C-2).
* (erc721) Emit the EVM token id, not the Cosmos NFT id, in the `erc721_token_ids` attribute of `refund_packet_token`, and emit one event per (contract, receiver) pair so multi-contract batches are reported completely.
* (erc721) Validate the one-to-one NFT binding in `ConvertERC721` *before* minting or transferring, instead of relying on `SetNFTPairs` to reject the conflict afterwards, so a rejected conversion leaves no side effects.
* (cw721) Validate `nft_ids` and `cosmos_token_ids` against the collection token-id rules in `ValidateBasic`, closing the validation gap with x/erc721.
* (app) Fail fast when `upgrade-info.json` exists but cannot be parsed instead of silently skipping the store loader and booting into an inconsistent state.
* (build) Inject `AppVersion`/`GitCommit` into the `uptick/version` package from the Makefile and goreleaser; previously only the SDK's `version` package was populated, leaving the project's own package permanently at `dev`.

## v0.4.0 - Unreleased (folded into v0.4.1)

> Never released standalone; every change listed here ships as part of v0.4.1.

### Features

- (erc721) Add `x/erc721` module for native Cosmos NFT <-> ERC721 conversions and IBC transfers of ERC721 tokens (`MsgConvertNFT`, `MsgConvertERC721`, `MsgTransferERC721`).
- (cw721) Add `x/cw721` module for native Cosmos NFT <-> CW721 conversions and IBC transfers (`MsgConvertNFT`, `MsgConvertCW721`, `MsgTransferCW721`).
- (evm) Migrate the EVM engine to `cosmos/evm` `x/vm` v0.6.1 (go-ethereum v1.16). ChainConfig is now time-based and Shanghai/Cancun/Prague forks are activated at upgrade time; EIP-7702 `SetCodeTx` is natively supported.
- (ibc) Upgrade `ibc-go` from v8 to v10: capability module and `ScopedKeeper` removed, `IBCModule` interface signatures updated.
- (sdk) Upgrade Cosmos SDK from v0.50 to v0.53 (gov v1 proposals, `KVStoreService`-based keepers, params authority-based).
- (wasm) Upgrade `wasmd` to v0.61 (wasmvm v3) and add Uptick wasm node defaults: 50M smart-query gas limit and 512 MiB memory cache.
- (app) Derive the EVM chain-id from genesis / `--chain-id` so JSON-RPC `eth_chainId` matches the EIP-155 id (e.g. `uptick_1170-1`).
- (erc20) Replace the local `x/erc20` with `cosmos/evm`'s `x/erc20` module wired through the ERC20 IBC middleware.
- (upgrade) Add the `v0.4.0` upgrade handler: migrate legacy `EthAccount`s to `BaseAccount`, convert legacy `ethsecp256k1` pubkeys, migrate erc20 params to authority-based, delete the `capability` store and remove legacy IBC provenance records.

### Security

- (erc721/cw721) Harden NFT bridge validation: receiver checks, token-id normalization and module-account deployment.
- (evmIBC) Enforce provenance tracking for `OutboundConvertClassId` and use `strings.HasPrefix` for ClassID prefix validation.
- (collection) Fix `ValidateGenesis` to check `Id` and validate `TokenURI` in `MsgTransferNFT`.
- (upgrade) Harden the v0.4.0 state transition and legacy params migration; register legacy `EthAccount` as a genesis account for bootstrap/export compatibility.
- (erc20) Validate `EnableEVMHook` requires `EnableErc20` in params to prevent EVM hook without ERC20 token registry.
- (erc20) Add `SetIBCKeeper` double-call protection to prevent silent keeper overwrite.
- (erc20) Add `DenomUnits` boundary check in `RegisterCoin` to prevent index-out-of-range panic.
- (erc20) Fix `IsDenomRegistered` using `Name` instead of `Base` — broken duplicate registration guard.
- (erc20) Validate `receiver` address in `OnRecvPacket` — prevent token delivery to zero address.
- (erc20) Wrap `refundPacketToken` with `CacheContext` for atomic ERC20 mint + coin sweep + burn.
- (erc20) Add `OnTimeoutPacket` in IBCMiddleware to clean up provenance on timeout.
- (app) Add `ValidateTokenURI` decoration in `WasmSecurityDecorator`.

### Improvements

- (erc20) Replace `MustUnmarshal` with `Unmarshal` + error handling in token pair iteration and migration.
- (erc20) Restore `verifyMetadata` validation in `RegisterCoin` for IBC metadata consistency.
- (erc20/evmIBC) Remove debug `fmt.Printf` and commented-out debug code from production paths.

### Bug Fixes

- (erc721) Fix `ConvertNFT` against the `cosmos/evm` v0.6.1 `StateDB` API and base-10 token-id handling.
- (cw721) Fix the `WasmContract` REST path prefix (`/uptick/erc721/v1/wasm_contract/...` -> `/uptick/cw721/v1/wasm_contract/...`).
- (erc20) Re-register `MsgConvertERC20CustomGetSigner` in tx config.
- (mempool) Normalize cosmos mempool max tx and EVM mempool block gas limit.
- (collection) Allow empty/inline `--schema` in collection tx CLI.
- (ci) Fix golangci-lint config verify failure and pin the linter version.

### API Breaking

- (ibc) `IBCModule` handlers now receive the channel version parameter; `SendPacket`/`WriteAcknowledgement` no longer take `chanCap`.
- (erc20) Governance proposals for ERC20 registration now use gov v1 message types.
- (params) `x/params` is deprecated; module params are managed through the authority.
- (evmIBC) IBC middleware now properly handles `OutboundConvertClassId` with provenance tracking.

### State Machine Breaking

- (upgrade) The `v0.4.0` upgrade is **not reversible**: the `capability` store is deleted and the EVM `ChainConfig` is migrated to time-based activation. All validators must upgrade before the upgrade height.

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
