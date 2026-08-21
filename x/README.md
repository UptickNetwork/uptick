<!--
order: 0
-->

# List of Modules

Here are some production-grade modules that can be used in Uptick applications, along with their respective documentation:

- [collection](collection/spec/README.md) - Managing native Cosmos NFTs that represent individual assets with unique features.
- [erc721](erc721/spec/README.md) - Converting between native Cosmos NFTs and ERC721 tokens, plus IBC transfers of ERC721 tokens.
- [cw721](cw721/spec/README.md) - Converting between native Cosmos NFTs and CW721 tokens, plus IBC transfers of CW721 tokens.
- [evmibc](evmibc/spec/README.md) - IBC middleware that converts ERC721/CW721 tokens to native Cosmos NFTs on IBC transfers.
- [internft](internft/spec/README.md) - Internal NFT keeper helpers shared by the conversion modules.
- [nft](nft/spec/README.md) - Legacy NFT module retained for backwards compatibility (not registered in the v0.4.0 app).

ERC-20 conversion is provided by `cosmos/evm`'s `x/erc20` module (see the
[cosmos/evm documentation](https://github.com/cosmos/evm/tree/main/x/erc20)); it is not a local Uptick module.

For the full module wiring and lifecycle, see the [app module list](../docs/intro/overview.md) and the
[protobuf reference](../docs/api/proto-docs.md).
