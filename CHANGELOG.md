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
