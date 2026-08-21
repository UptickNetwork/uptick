<!--
order: 0
title: CW721 Overview
parent:
  title: "CW721"
-->

# CW721 Specification

## Overview

The `x/cw721` module (introduced in v0.4.0) provides bidirectional conversion between native
Cosmos NFTs (managed by `x/collection` / `cosmossdk.io/x/nft`) and CW721 tokens deployed as
CosmWasm contracts on Uptick, as well as IBC transfers of CW721 tokens between chains.

As with `x/erc721`, conversions require a registered **token pair** that maps a CW721 contract
address to a native Cosmos NFT class ID.

## State

### TokenPair

| Field           | Type   | Description                         |
|-----------------|--------|-------------------------------------|
| `cw721_address` | string | Address of the CW721 contract       |
| `class_id`      | string | Native Cosmos NFT class ID to map   |

### UIDPair

| Field       | Type   | Description                                   |
|-------------|--------|-----------------------------------------------|
| `cw721_did` | string | CW721 contract address + token ID             |
| `class_did` | string | Native Cosmos NFT class ID + NFT ID           |

### Params

| Parameter       | Type | Description                                                              |
|-----------------|------|--------------------------------------------------------------------------|
| `enable_cw721`  | bool | Enable Cosmos NFT <-> CW721 conversions                                  |
| `enable_evm_hook` | bool | Enable the EVM hook that converts CW721 tokens to native NFTs          |

## Messages

### MsgConvertNFT

Converts a native Cosmos NFT into a CW721 representation, minting to `receiver` (defaults to the
sender when omitted).

### MsgConvertCW721

Converts a CW721 token back into a native Cosmos NFT, minting to `receiver` (defaults to the sender
when omitted).

### MsgTransferCW721

Converts CW721 tokens to native NFTs and transfers them to another chain through IBC (ICS-721),
including timeout height/timestamp and memo support.

## Queries

| RPC            | HTTP endpoint                                        | Description                       |
|----------------|------------------------------------------------------|-----------------------------------|
| `TokenPairs`   | `GET /uptick/cw721/v1/token_pairs`                   | List registered token pairs       |
| `TokenPair`    | `GET /uptick/cw721/v1/token_pairs/{token}`           | Get a token pair by contract/class|
| `WasmContract` | `GET /uptick/cw721/v1/wasm_contract/{port}/{channel}/{class_id}` | Resolve a contract from IBC info |
| `Params`       | `GET /uptick/cw721/v1/params`                        | Get module parameters             |

## Events

The module emits `token_lock`, `token_unlock`, `mint`, `burn`, `convert_nft`,
`convert_cw721`, `register_nft`, `register_cw721` and `toggle_token_conversion` events.

## CLI

```bash
# Query
uptickd query cw721 token-pairs
uptickd query cw721 token-pair <token>
uptickd query cw721 wasm-contract <port> <channel> <classId>
uptickd query cw721 params

# Tx
uptickd tx cw721 convert-nft <class_id> <nft_ids> <contract_address> <token_ids> [receiver_hex]
uptickd tx cw721 convert-cw721 <contract_address> <token_ids> <class_id> <nft_ids> [receiver]
uptickd tx cw721 ibc-transfer-cw721 <cw_contract_address> <cw_token_ids> <src_port> <src_channel> <cosmos_receiver> <class_id> <cosmos_token_ids>
```

## References

- Protobuf definitions: `proto/uptick/cw721/v1/`
- Implementation: `x/cw721/`
