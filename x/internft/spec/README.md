<!--
order: 0
title: Internal NFT Overview
parent:
  title: "Internal NFT"
-->

# Internal NFT Specification

## Overview

The `x/internft` package provides an internal ICS-721 keeper adapter used by the IBC NFT-transfer
integration. It wraps the `x/collection` NFT keeper and implements the standard
`CreateOrUpdateClass`, `Mint`, `Transfer` and ownership helpers expected by the
[`nft-transfer`](https://github.com/UptickNetwork/nft-transfer) IBC module.

It is an internal helper and does **not** register its own module, store, or messages in the
v0.4.0 application.

## Key APIs

- `CreateOrUpdateClass` - create or update a native NFT class from an ICS-721 class.
- `Mint` - mint a native NFT for an ICS-721 mint.
- `Transfer` - transfer a native NFT to a receiver.
- `BalanceOf` / `OwnerOf` - ownership helpers used by the IBC module.

## References

- Implementation: `x/internft/`
