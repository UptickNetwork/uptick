<!--
order: 0
title: EVM IBC Overview
parent:
  title: "EVM IBC"
-->

# EVM IBC Specification

## Overview

The `x/evmibc` module is an IBC middleware that bridges the ERC721/CW721 conversion modules with
ICS-721 (NFT transfer). When an ERC721 or CW721 token is transferred to Uptick over IBC, the
middleware converts it into a native Cosmos NFT using the registered token pairs; when a native NFT
leaves the chain, it can be converted to the corresponding EVM/CW721 representation on the
destination chain.

In v0.4.0 the middleware tracks transfer provenance for `OutboundConvertClassId` so failed or timed
out transfers can be refunded correctly, and prefix validation uses `strings.HasPrefix`.

## Messages

### MsgTransferERC721

Transfers ERC721 tokens to another chain through IBC. The tokens are first converted to native
Cosmos NFTs and then sent using the ICS-721 packet format.

## Queries

| RPC           | HTTP endpoint                                              | Description                     |
|---------------|------------------------------------------------------------|---------------------------------|
| `EvmContract` | `GET /uptick/evmIBC/v1/evm_contract/{port}/{channel}/{class_id}` | Resolve an ERC721 contract from IBC info |

## References

- Protobuf definitions: `proto/uptick/evm_ibc/v1/`
- Implementation: `x/evmibc/`
