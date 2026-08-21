<!--
order: 0
title: ERC721 Overview
parent:
  title: "ERC721"
-->

# ERC721 Specification

## Overview

The `x/erc721` module (introduced in v0.4.0) provides bidirectional conversion between native
Cosmos NFTs (managed by `x/collection` / `cosmossdk.io/x/nft`) and ERC721 tokens on the Uptick
EVM, as well as IBC transfers of ERC721 tokens between chains.

Conversions are only possible for contracts that are registered in the module's **token pair**
registry. A token pair maps an ERC721 contract address to a native Cosmos NFT class ID, so the
module always knows which EVM contract represents which native class.

## State

### TokenPair

`TokenPair` records a mapping between a native Cosmos NFT class and an ERC721 contract:

| Field          | Type   | Description                          |
|----------------|--------|--------------------------------------|
| `erc721_address` | string | Address of the ERC721 contract      |
| `class_id`       | string | Native Cosmos NFT class ID to map   |

### UIDPair

`UIDPair` records the unique asset-level mapping used during conversions:

| Field        | Type   | Description                                    |
|--------------|--------|------------------------------------------------|
| `erc721_did` | string | ERC721 contract address + token ID             |
| `class_did`  | string | Native Cosmos NFT class ID + NFT ID            |

### Params

| Parameter       | Type | Description                                                               |
|-----------------|------|---------------------------------------------------------------------------|
| `enable_erc721` | bool | Enable Cosmos NFT <-> ERC721 conversions                                  |
| `enable_evm_hook` | bool | Enable the EVM hook that converts ERC721 tokens to native NFTs           |

## Messages

### MsgConvertNFT

Converts a native Cosmos NFT into an ERC721 representation. The ERC721 token is minted to the
`evm_receiver` address (defaults to the sender when omitted).

### MsgConvertERC721

Converts an ERC721 token back into a native Cosmos NFT, minting the NFT to the `cosmos_receiver`
address (defaults to the sender when omitted).

### MsgTransferERC721

Converts ERC721 tokens to native NFTs and transfers them to another chain through IBC
(ICS-721), including timeout height/timestamp and memo support.

## Queries

| RPC          | HTTP endpoint                                        | Description                        |
|--------------|------------------------------------------------------|------------------------------------|
| `TokenPairs` | `GET /uptick/erc721/v1/token_pairs`                  | List registered token pairs        |
| `TokenPair`  | `GET /uptick/erc721/v1/token_pairs/{token}`          | Get a token pair by contract/class |
| `EvmContract`| `GET /uptick/erc721/v1/evm_contract/{port}/{channel}/{class_id}` | Resolve a contract from IBC info |
| `Params`     | `GET /uptick/erc721/v1/params`                       | Get module parameters              |

## Events

The module emits `token_lock`, `token_unlock`, `mint`, `burn`, `convert_nft`,
`convert_erc721`, `register_nft`, `register_erc721`, `toggle_token_conversion`,
`refund_packet_token` and `refund_packet_token_skip` events.

## CLI

```bash
# Query
uptickd query erc721 token-pairs
uptickd query erc721 token-pair <token>
uptickd query erc721 evm-contract <port> <channel> <classId>
uptickd query erc721 params

# Tx
uptickd tx erc721 convert-nft <class_id> <cosmos_token_ids> <evm_contract_address> <evm_token_ids> [receiver_hex]
uptickd tx erc721 convert-erc721 <evm_contract_address> <evm_token_ids> <class_id> <cosmos_token_ids> [cosmos_receiver]
uptickd tx erc721 ibc-transfer-erc721 <evm_contract_address> <evm_token_ids> <src_port> <src_channel> <cosmos_receiver> <class_id> <cosmos_token_ids>
```

## References

- Protobuf definitions: `proto/uptick/erc721/v1/`
- Implementation: `x/erc721/`
