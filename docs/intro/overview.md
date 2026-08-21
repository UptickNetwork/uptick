<!--
order: 1
-->

# High-level Overview

Learn about Uptick and its primary features. {synopsis}

## What is Uptick

Uptick is a scalable, high-throughput Proof-of-Stake blockchain that is fully compatible and
interoperable with Ethereum. It's built using the [Cosmos SDK](https://github.com/cosmos/cosmos-sdk/) which runs on top of [Tendermint Core](https://github.com/cometbft/cometbft) consensus engine.

Uptick allows for running vanilla Ethereum as a [Cosmos](https://cosmos.network/)
application-specific blockchain. This allows developers to have all the desired features of
Ethereum, while at the same time, benefit from Tendermint’s PoS implementation. Also, because it is
built on top of the Cosmos SDK, it will be able to exchange value with the rest of the Cosmos
Ecosystem through the Inter Blockchain Communication Protocol (IBC).

### Features

Here’s a glance at some of the key features of Uptick:

* Web3 and EVM compatibility
* High throughput via [Tendermint Core](https://github.com/cometbft/cometbft)
* Horizontal scalability via [IBC](https://cosmos.network/ibc)
* Fast transaction finality
* Native NFT interoperability: convert between Cosmos NFTs, ERC721 and CW721, and transfer them across chains with IBC (v0.4.0)

Uptick enables these key features by:

* Implementing Tendermint Core's Application Blockchain Interface ([ABCI](https://docs.tendermint.com/master/spec/abci/)) to manage the blockchain
* Leveraging [modules](https://docs.cosmos.network/master/building-modules/intro.html) and other mechanisms implemented by the [Cosmos SDK](https://docs.cosmos.network/).
* Utilizing [`cosmos/evm`](https://github.com/cosmos/evm) (go-ethereum v1.16) to provide the EVM execution layer.
* Exposing a fully compatible Web3 [JSON-RPC](./../basic/json_rpc.md) layer for interacting with existing Ethereum clients and tooling ([Metamask](./../guides/keys-wallets/metamask.md), [Remix](./../guides/tools/remix.md), [Truffle](./../guides/tools/truffle.md), etc).

The sum of these features allows developers to leverage existing Ethereum ecosystem tooling and
software to seamlessly deploy smart contracts which interact with the rest of the Cosmos
[ecosystem](https://cosmos.network/ecosystem)!

## v0.4.0 Module Highlights

- `x/erc721` - native Cosmos NFT <-> ERC721 conversion and IBC transfers of ERC721 tokens.
- `x/cw721` - native Cosmos NFT <-> CW721 conversion and IBC transfers of CW721 tokens.
- `x/evmibc` - IBC middleware bridging ERC721/CW721 conversions with ICS-721.
- `cosmos/evm` `x/vm` - the EVM execution engine (migrated from the legacy Ethermint `x/evm`), with `x/feemarket` and `x/erc20`.
- `x/collection` - native Cosmos NFT management (classes, NFTs, metadata).
- `cosmossdk.io/x/nft` - the SDK NFT module used by the conversion modules.
- `x/internft` - internal ICS-721 keeper adapter for the IBC NFT transfer integration.

See the [module directory](../x/README.md) and the [protobuf reference](../api/proto-docs.md) for details.

## Quick Facts Table

| Property                     | Value                                                |
|------------------------------|------------------------------------------------------|
| Uptick Testnet                | `{{ $themeConfig.project.testnet_chain_id }}`        |
| Uptick Mainnet (not yet live) | `{{ $themeConfig.project.chain_id }}`                |
| Block Time                   | ~7 seconds                                           |
