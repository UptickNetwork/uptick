<!--
order: 0
title: Legacy NFT Overview
parent:
  title: "Legacy NFT"
-->

# Legacy NFT Specification

## Overview

The `x/nft` module contains the legacy NFT types from earlier Uptick versions. In v0.4.0 the
module is **not registered** in the application: native NFT state is managed by
`cosmossdk.io/x/nft` and `x/collection`, and conversions are handled by `x/erc721` and `x/cw721`.

The legacy types are retained for reference and backwards compatibility during migration.

## References

- Protobuf definitions: `proto/uptick/nft/v1beta1/`
- Implementation: `x/nft/`
