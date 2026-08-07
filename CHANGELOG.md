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

## v0.3.3 - 2023-12-12

### Bug Fixes

- Harden erc20 IBC outbound refund authorization: require `MsgTransferERC20` provenance instead of user-controlled memo substrings (`v0.3.3` upgrade).
- Fix erc20 IBC error-ack rollback: record transfer provenance using the ICS-20 packet denomination path (not the bank `ibc/HASH` denom) so ERC20 supply is restored when the transfer fails.

## v0.3.4 - 2026-08-03

### Security

- (erc20) Validate `EnableEVMHook` requires `EnableErc20` in params to prevent EVM hook without ERC20 token registry.
- (erc20) Add `SetIBCKeeper` double-call protection to prevent silent keeper overwrite.
- (erc20) Add `DenomUnits` boundary check in `RegisterCoin` to prevent index-out-of-range panic.
- (erc20) Fix `IsDenomRegistered` using `Name` instead of `Base` — broken duplicate registration guard.
- (erc20) Validate `receiver` address in `OnRecvPacket` — prevent token delivery to zero address.
- (erc20) Wrap `refundPacketToken` with `CacheContext` for atomic ERC20 mint + coin sweep + burn.
- (evmIBC) Replace `strings.Contains` with `strings.HasPrefix` in ClassID prefix validation.
- (erc20) Add `OnTimeoutPacket` in IBCMiddleware to clean up provenance on timeout.
- (collection) Fix `ValidateGenesis` checking `Name` instead of `Id`.
- (collection) Add `ValidateTokenURI` in `MsgTransferNFT` for consistency with mint/edit messages.
- (app) Add `ValidateTokenURI` decoration in `WasmSecurityDecorator`.

### Improvements

- (erc20) Replace `MustUnmarshal` with `Unmarshal` + error handling in token pair iteration and migration.
- (erc20) Restore `verifyMetadata` validation in `RegisterCoin` for IBC metadata consistency.
- (erc20/evmIBC) Remove debug `fmt.Printf` and commented-out debug code from production paths.

### API Breaking

- (evmIBC) IBC middleware now properly handles `OutboundConvertClassId` with provenance tracking.

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