<!--
order: 0
title: Security
-->

# Security

Uptick treats security as a first-class concern. See [SECURITY.md](../../SECURITY.md) for the
vulnerability disclosure policy and contacts.

## v0.4.0 Security Hardening

The v0.4.0 release includes several security-relevant changes:

- **NFT bridge hardening** (`x/erc721` / `x/cw721`): receiver validation, token-id normalization
  and module-account deployment fixes.
- **EVM migration**: the legacy Ethermint EVM stack was replaced by `cosmos/evm` `x/vm`, which is
  actively maintained upstream (go-ethereum v1.16).
- **IBC provenance** (`x/evmibc`): `OutboundConvertClassId` provenance tracking and stricter
  ClassID prefix validation (`strings.HasPrefix`).
- **Collection module**: `ValidateGenesis` now checks `Id`, and `MsgTransferNFT` validates
  `TokenURI` consistently with mint/edit messages.
- **Upgrade safety**: the `v0.4.0` upgrade migrates legacy accounts and pubkeys, removes the
  deprecated `capability` store, and is intentionally **not reversible** — see the
  [node upgrade guide](../guides/upgrades/upgrade_node.md).

## Historical Security Fixes

Security fixes from earlier releases are tracked in [CHANGELOG.md](../../CHANGELOG.md) (e.g. the
v0.3.4 ERC20 IBC refund and EVM-hook validation fixes).
