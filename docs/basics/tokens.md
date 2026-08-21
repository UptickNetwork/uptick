<!--
order: 5
-->

# Tokens

Learn about the the different types of tokens available in Uptick. {synopsis}

## Introduction

Uptick is a Cosmos SDK-based chain with full EVM support. Because of this architecture, tokens and assets in the network may come from different independent sources: the `bank` module and the `evm` module.

## Cosmos Coins

Accounts can own SDK coins in their balance, which are used for operations with other Cosmos modules and transactions. Examples of these are using the coins for staking, IBC transfers, governance deposits and EVM  

### UPTICK

The denomination used for staking, governance and gas consumption on the EVM is the UPTICK. The UPTICK provides the utility of: securing the Proof-of-Stake chain, token used for governance proposals, fee distribution and as a mean of gas for running smart contracts on the EVM.

Uptick uses [Atto](https://en.wikipedia.org/wiki/Atto-) UPTICK as the base denomination to maintain parity with Ethereum.

$$1 uptick = 1 ~ * ~ 10^{18} auptick$$

This matches Ethereum denomination of:

$$1 ETH = 1 ~ * ~ 10^{18} wei$$

### EVM Tokens

Uptick is compatible with ERC20 tokens and non-fungible token standards (EIP-721) that are
natively supported by the EVM.

### Native NFTs

Native Cosmos NFTs are managed by the `x/collection` module (and the SDK `x/nft` module). Each NFT
belongs to a class (denom) and carries an ID plus optional metadata (`tokenURI` / `data`).

### ERC721 and CW721 Conversions

Since v0.4.0, Uptick supports bidirectional conversion between native Cosmos NFTs and:

- **ERC721 tokens** (EVM contracts) through `x/erc721`.
- **CW721 tokens** (CosmWasm contracts) through `x/cw721`.

Conversions require a registered token pair (ERC721/CW721 contract address <-> native class ID) and
are gated by the module params (`enable_erc721` / `enable_cw721` and `enable_evm_hook`). Once
converted, ERC721/CW721 tokens can also be transferred to other chains over IBC.

ERC20 conversion is provided by `cosmos/evm`'s `x/erc20` module through the ERC20 IBC middleware.
