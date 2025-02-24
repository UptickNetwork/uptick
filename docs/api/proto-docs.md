<!-- This file is auto-generated. Please do not modify it yourself. -->
# Protobuf Documentation
<a name="top"></a>

## Table of Contents

- [uptick/collection/v1/collection.proto](#uptick/collection/v1/collection.proto)
    - [BaseNFT](#uptick.collection.v1.BaseNFT)
    - [Collection](#uptick.collection.v1.Collection)
    - [Denom](#uptick.collection.v1.Denom)
    - [DenomMetadata](#uptick.collection.v1.DenomMetadata)
    - [IDCollection](#uptick.collection.v1.IDCollection)
    - [NFTMetadata](#uptick.collection.v1.NFTMetadata)
    - [Owner](#uptick.collection.v1.Owner)
  
- [uptick/collection/v1/genesis.proto](#uptick/collection/v1/genesis.proto)
    - [GenesisState](#uptick.collection.v1.GenesisState)
  
- [uptick/collection/v1/query.proto](#uptick/collection/v1/query.proto)
    - [QueryCollectionRequest](#uptick.collection.v1.QueryCollectionRequest)
    - [QueryCollectionResponse](#uptick.collection.v1.QueryCollectionResponse)
    - [QueryDenomRequest](#uptick.collection.v1.QueryDenomRequest)
    - [QueryDenomResponse](#uptick.collection.v1.QueryDenomResponse)
    - [QueryDenomsRequest](#uptick.collection.v1.QueryDenomsRequest)
    - [QueryDenomsResponse](#uptick.collection.v1.QueryDenomsResponse)
    - [QueryNFTRequest](#uptick.collection.v1.QueryNFTRequest)
    - [QueryNFTResponse](#uptick.collection.v1.QueryNFTResponse)
    - [QueryNFTsOfOwnerRequest](#uptick.collection.v1.QueryNFTsOfOwnerRequest)
    - [QueryNFTsOfOwnerResponse](#uptick.collection.v1.QueryNFTsOfOwnerResponse)
    - [QuerySupplyRequest](#uptick.collection.v1.QuerySupplyRequest)
    - [QuerySupplyResponse](#uptick.collection.v1.QuerySupplyResponse)
  
    - [Query](#uptick.collection.v1.Query)
  
- [uptick/collection/v1/tx.proto](#uptick/collection/v1/tx.proto)
    - [MsgBurnNFT](#uptick.collection.v1.MsgBurnNFT)
    - [MsgBurnNFTResponse](#uptick.collection.v1.MsgBurnNFTResponse)
    - [MsgEditNFT](#uptick.collection.v1.MsgEditNFT)
    - [MsgEditNFTResponse](#uptick.collection.v1.MsgEditNFTResponse)
    - [MsgIssueDenom](#uptick.collection.v1.MsgIssueDenom)
    - [MsgIssueDenomResponse](#uptick.collection.v1.MsgIssueDenomResponse)
    - [MsgMintNFT](#uptick.collection.v1.MsgMintNFT)
    - [MsgMintNFTResponse](#uptick.collection.v1.MsgMintNFTResponse)
    - [MsgTransferDenom](#uptick.collection.v1.MsgTransferDenom)
    - [MsgTransferDenomResponse](#uptick.collection.v1.MsgTransferDenomResponse)
    - [MsgTransferNFT](#uptick.collection.v1.MsgTransferNFT)
    - [MsgTransferNFTResponse](#uptick.collection.v1.MsgTransferNFTResponse)
  
    - [Msg](#uptick.collection.v1.Msg)
  
- [uptick/cw721/v1/cw721.proto](#uptick/cw721/v1/cw721.proto)
    - [TokenPair](#uptick.cw721.v1.TokenPair)
    - [UIDPair](#uptick.cw721.v1.UIDPair)
  
    - [Owner](#uptick.cw721.v1.Owner)
  
- [uptick/cw721/v1/genesis.proto](#uptick/cw721/v1/genesis.proto)
    - [GenesisState](#uptick.cw721.v1.GenesisState)
    - [Params](#uptick.cw721.v1.Params)
  
- [uptick/cw721/v1/query.proto](#uptick/cw721/v1/query.proto)
    - [QueryParamsRequest](#uptick.cw721.v1.QueryParamsRequest)
    - [QueryParamsResponse](#uptick.cw721.v1.QueryParamsResponse)
    - [QueryTokenPairRequest](#uptick.cw721.v1.QueryTokenPairRequest)
    - [QueryTokenPairResponse](#uptick.cw721.v1.QueryTokenPairResponse)
    - [QueryTokenPairsRequest](#uptick.cw721.v1.QueryTokenPairsRequest)
    - [QueryTokenPairsResponse](#uptick.cw721.v1.QueryTokenPairsResponse)
    - [QueryWasmAddressRequest](#uptick.cw721.v1.QueryWasmAddressRequest)
  
    - [Query](#uptick.cw721.v1.Query)
  
- [uptick/cw721/v1/tx.proto](#uptick/cw721/v1/tx.proto)
    - [MsgConvertC721Response](#uptick.cw721.v1.MsgConvertC721Response)
    - [MsgConvertCW721](#uptick.cw721.v1.MsgConvertCW721)
    - [MsgConvertCW721Response](#uptick.cw721.v1.MsgConvertCW721Response)
    - [MsgConvertNFT](#uptick.cw721.v1.MsgConvertNFT)
    - [MsgConvertNFTResponse](#uptick.cw721.v1.MsgConvertNFTResponse)
    - [MsgTransferCW721](#uptick.cw721.v1.MsgTransferCW721)
    - [MsgTransferCW721Response](#uptick.cw721.v1.MsgTransferCW721Response)
  
    - [Msg](#uptick.cw721.v1.Msg)
  
- [uptick/erc20/v1/erc20.proto](#uptick/erc20/v1/erc20.proto)
    - [RegisterCoinProposal](#uptick.erc20.v1.RegisterCoinProposal)
    - [RegisterERC20Proposal](#uptick.erc20.v1.RegisterERC20Proposal)
    - [ToggleTokenRelayProposal](#uptick.erc20.v1.ToggleTokenRelayProposal)
    - [TokenPair](#uptick.erc20.v1.TokenPair)
    - [UpdateTokenPairERC20Proposal](#uptick.erc20.v1.UpdateTokenPairERC20Proposal)
  
    - [Owner](#uptick.erc20.v1.Owner)
  
- [uptick/erc20/v1/genesis.proto](#uptick/erc20/v1/genesis.proto)
    - [GenesisState](#uptick.erc20.v1.GenesisState)
    - [Params](#uptick.erc20.v1.Params)
  
- [uptick/erc20/v1/query.proto](#uptick/erc20/v1/query.proto)
    - [QueryParamsRequest](#uptick.erc20.v1.QueryParamsRequest)
    - [QueryParamsResponse](#uptick.erc20.v1.QueryParamsResponse)
    - [QueryTokenPairRequest](#uptick.erc20.v1.QueryTokenPairRequest)
    - [QueryTokenPairResponse](#uptick.erc20.v1.QueryTokenPairResponse)
    - [QueryTokenPairsRequest](#uptick.erc20.v1.QueryTokenPairsRequest)
    - [QueryTokenPairsResponse](#uptick.erc20.v1.QueryTokenPairsResponse)
  
    - [Query](#uptick.erc20.v1.Query)
  
- [uptick/erc20/v1/tx.proto](#uptick/erc20/v1/tx.proto)
    - [MsgConvertCoin](#uptick.erc20.v1.MsgConvertCoin)
    - [MsgConvertCoinResponse](#uptick.erc20.v1.MsgConvertCoinResponse)
    - [MsgConvertERC20](#uptick.erc20.v1.MsgConvertERC20)
    - [MsgConvertERC20Response](#uptick.erc20.v1.MsgConvertERC20Response)
    - [MsgTransferERC20](#uptick.erc20.v1.MsgTransferERC20)
    - [MsgTransferERC20Response](#uptick.erc20.v1.MsgTransferERC20Response)
  
    - [Msg](#uptick.erc20.v1.Msg)
  
- [uptick/evmIBC/v1/evmIBC.proto](#uptick/evmIBC/v1/evmIBC.proto)
    - [TokenPair](#uptick.evmIBC.v1.TokenPair)
  
- [uptick/evmIBC/v1/query.proto](#uptick/evmIBC/v1/query.proto)
    - [QueryEvmAddressRequest](#uptick.evmIBC.v1.QueryEvmAddressRequest)
    - [QueryTokenPairResponse](#uptick.evmIBC.v1.QueryTokenPairResponse)
  
    - [Query](#uptick.evmIBC.v1.Query)
  
- [uptick/evmIBC/v1/tx.proto](#uptick/evmIBC/v1/tx.proto)
    - [MsgTransferERC721](#uptick.evmIBC.v1.MsgTransferERC721)
    - [MsgTransferERC721Response](#uptick.evmIBC.v1.MsgTransferERC721Response)
  
    - [Msg](#uptick.evmIBC.v1.Msg)
  
- [uptick/nft/v1beta1/event.proto](#uptick/nft/v1beta1/event.proto)
    - [EventBurn](#cosmos.nft.v1beta1.EventBurn)
    - [EventMint](#cosmos.nft.v1beta1.EventMint)
    - [EventSend](#cosmos.nft.v1beta1.EventSend)
  
- [uptick/nft/v1beta1/nft.proto](#uptick/nft/v1beta1/nft.proto)
    - [Class](#cosmos.nft.v1beta1.Class)
    - [NFT](#cosmos.nft.v1beta1.NFT)
  
- [uptick/nft/v1beta1/genesis.proto](#uptick/nft/v1beta1/genesis.proto)
    - [Entry](#cosmos.nft.v1beta1.Entry)
    - [GenesisState](#cosmos.nft.v1beta1.GenesisState)
  
- [uptick/nft/v1beta1/query.proto](#uptick/nft/v1beta1/query.proto)
    - [QueryBalanceRequest](#cosmos.nft.v1beta1.QueryBalanceRequest)
    - [QueryBalanceResponse](#cosmos.nft.v1beta1.QueryBalanceResponse)
    - [QueryClassRequest](#cosmos.nft.v1beta1.QueryClassRequest)
    - [QueryClassResponse](#cosmos.nft.v1beta1.QueryClassResponse)
    - [QueryClassesRequest](#cosmos.nft.v1beta1.QueryClassesRequest)
    - [QueryClassesResponse](#cosmos.nft.v1beta1.QueryClassesResponse)
    - [QueryNFTRequest](#cosmos.nft.v1beta1.QueryNFTRequest)
    - [QueryNFTResponse](#cosmos.nft.v1beta1.QueryNFTResponse)
    - [QueryNFTsRequest](#cosmos.nft.v1beta1.QueryNFTsRequest)
    - [QueryNFTsResponse](#cosmos.nft.v1beta1.QueryNFTsResponse)
    - [QueryOwnerRequest](#cosmos.nft.v1beta1.QueryOwnerRequest)
    - [QueryOwnerResponse](#cosmos.nft.v1beta1.QueryOwnerResponse)
    - [QuerySupplyRequest](#cosmos.nft.v1beta1.QuerySupplyRequest)
    - [QuerySupplyResponse](#cosmos.nft.v1beta1.QuerySupplyResponse)
  
    - [Query](#cosmos.nft.v1beta1.Query)
  
- [uptick/nft/v1beta1/tx.proto](#uptick/nft/v1beta1/tx.proto)
    - [MsgSend](#cosmos.nft.v1beta1.MsgSend)
    - [MsgSendResponse](#cosmos.nft.v1beta1.MsgSendResponse)
  
    - [Msg](#cosmos.nft.v1beta1.Msg)
  
- [Scalar Value Types](#scalar-value-types)



<a name="uptick/collection/v1/collection.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/collection/v1/collection.proto



<a name="uptick.collection.v1.BaseNFT"></a>

### BaseNFT
BaseNFT defines a non-fungible token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `id` | [string](#string) |  |  |
| `name` | [string](#string) |  |  |
| `uri` | [string](#string) |  |  |
| `data` | [string](#string) |  |  |
| `owner` | [string](#string) |  |  |
| `uri_hash` | [string](#string) |  |  |






<a name="uptick.collection.v1.Collection"></a>

### Collection
Collection defines a type of collection


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `denom` | [Denom](#uptick.collection.v1.Denom) |  |  |
| `nfts` | [BaseNFT](#uptick.collection.v1.BaseNFT) | repeated |  |






<a name="uptick.collection.v1.Denom"></a>

### Denom
Denom defines a type of NFT


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `id` | [string](#string) |  |  |
| `name` | [string](#string) |  |  |
| `schema` | [string](#string) |  |  |
| `creator` | [string](#string) |  |  |
| `symbol` | [string](#string) |  |  |
| `mint_restricted` | [bool](#bool) |  |  |
| `update_restricted` | [bool](#bool) |  |  |
| `description` | [string](#string) |  |  |
| `uri` | [string](#string) |  |  |
| `uri_hash` | [string](#string) |  |  |
| `data` | [string](#string) |  |  |






<a name="uptick.collection.v1.DenomMetadata"></a>

### DenomMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `creator` | [string](#string) |  |  |
| `schema` | [string](#string) |  |  |
| `mint_restricted` | [bool](#bool) |  |  |
| `update_restricted` | [bool](#bool) |  |  |
| `data` | [string](#string) |  |  |






<a name="uptick.collection.v1.IDCollection"></a>

### IDCollection
IDCollection defines a type of collection with specified ID


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `denom_id` | [string](#string) |  |  |
| `token_ids` | [string](#string) | repeated |  |






<a name="uptick.collection.v1.NFTMetadata"></a>

### NFTMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `name` | [string](#string) |  |  |
| `data` | [string](#string) |  |  |






<a name="uptick.collection.v1.Owner"></a>

### Owner
Owner defines a type of owner


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `address` | [string](#string) |  |  |
| `id_collections` | [IDCollection](#uptick.collection.v1.IDCollection) | repeated |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/collection/v1/genesis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/collection/v1/genesis.proto



<a name="uptick.collection.v1.GenesisState"></a>

### GenesisState
GenesisState defines the collection module's genesis state


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `collections` | [Collection](#uptick.collection.v1.Collection) | repeated |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/collection/v1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/collection/v1/query.proto



<a name="uptick.collection.v1.QueryCollectionRequest"></a>

### QueryCollectionRequest
QueryCollectionRequest is the request type for the Query/Collection RPC
method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `denom_id` | [string](#string) |  |  |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  | pagination defines an optional pagination for the request. |






<a name="uptick.collection.v1.QueryCollectionResponse"></a>

### QueryCollectionResponse
QueryCollectionResponse is the response type for the Query/Collection RPC
method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `collection` | [Collection](#uptick.collection.v1.Collection) |  |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  |  |






<a name="uptick.collection.v1.QueryDenomRequest"></a>

### QueryDenomRequest
QueryDenomRequest is the request type for the Query/Denom RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `denom_id` | [string](#string) |  |  |






<a name="uptick.collection.v1.QueryDenomResponse"></a>

### QueryDenomResponse
QueryDenomResponse is the response type for the Query/Denom RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `denom` | [Denom](#uptick.collection.v1.Denom) |  |  |






<a name="uptick.collection.v1.QueryDenomsRequest"></a>

### QueryDenomsRequest
QueryDenomsRequest is the request type for the Query/Denoms RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  | pagination defines an optional pagination for the request. |






<a name="uptick.collection.v1.QueryDenomsResponse"></a>

### QueryDenomsResponse
QueryDenomsResponse is the response type for the Query/Denoms RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `denoms` | [Denom](#uptick.collection.v1.Denom) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  |  |






<a name="uptick.collection.v1.QueryNFTRequest"></a>

### QueryNFTRequest
QueryNFTRequest is the request type for the Query/NFT RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `denom_id` | [string](#string) |  |  |
| `token_id` | [string](#string) |  |  |






<a name="uptick.collection.v1.QueryNFTResponse"></a>

### QueryNFTResponse
QueryNFTResponse is the response type for the Query/NFT RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `nft` | [BaseNFT](#uptick.collection.v1.BaseNFT) |  |  |






<a name="uptick.collection.v1.QueryNFTsOfOwnerRequest"></a>

### QueryNFTsOfOwnerRequest
QueryNFTsOfOwnerRequest is the request type for the Query/Owner RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `denom_id` | [string](#string) |  |  |
| `owner` | [string](#string) |  |  |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  | pagination defines an optional pagination for the request. |






<a name="uptick.collection.v1.QueryNFTsOfOwnerResponse"></a>

### QueryNFTsOfOwnerResponse
QueryNFTsOfOwnerResponse is the response type for the Query/Owner RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `owner` | [Owner](#uptick.collection.v1.Owner) |  |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  |  |






<a name="uptick.collection.v1.QuerySupplyRequest"></a>

### QuerySupplyRequest
QuerySupplyRequest is the request type for the Query/HTLC RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `denom_id` | [string](#string) |  |  |
| `owner` | [string](#string) |  |  |






<a name="uptick.collection.v1.QuerySupplyResponse"></a>

### QuerySupplyResponse
QuerySupplyResponse is the response type for the Query/Supply RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `amount` | [uint64](#uint64) |  |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.collection.v1.Query"></a>

### Query
Query defines the gRPC querier service for NFT module

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `Supply` | [QuerySupplyRequest](#uptick.collection.v1.QuerySupplyRequest) | [QuerySupplyResponse](#uptick.collection.v1.QuerySupplyResponse) | Supply queries the total supply of a given denom or owner | GET|/uptick/collection/collections/{denom_id}/supply|
| `NFTsOfOwner` | [QueryNFTsOfOwnerRequest](#uptick.collection.v1.QueryNFTsOfOwnerRequest) | [QueryNFTsOfOwnerResponse](#uptick.collection.v1.QueryNFTsOfOwnerResponse) | NFTsOfOwner queries the NFTs of the specified owner | GET|/uptick/collection/nfts|
| `Collection` | [QueryCollectionRequest](#uptick.collection.v1.QueryCollectionRequest) | [QueryCollectionResponse](#uptick.collection.v1.QueryCollectionResponse) | Collection queries the NFTs of the specified denom | GET|/uptick/collection/collections/{denom_id}|
| `Denom` | [QueryDenomRequest](#uptick.collection.v1.QueryDenomRequest) | [QueryDenomResponse](#uptick.collection.v1.QueryDenomResponse) | Denom queries the definition of a given denom | GET|/uptick/collection/nft/denoms/{denom_id}|
| `Denoms` | [QueryDenomsRequest](#uptick.collection.v1.QueryDenomsRequest) | [QueryDenomsResponse](#uptick.collection.v1.QueryDenomsResponse) | Denoms queries all the denoms | GET|/uptick/collection/nft/denoms|
| `NFT` | [QueryNFTRequest](#uptick.collection.v1.QueryNFTRequest) | [QueryNFTResponse](#uptick.collection.v1.QueryNFTResponse) | NFT queries the NFT for the given denom and token ID | GET|/uptick/collection/nfts/{denom_id}/{token_id}|

 <!-- end services -->



<a name="uptick/collection/v1/tx.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/collection/v1/tx.proto



<a name="uptick.collection.v1.MsgBurnNFT"></a>

### MsgBurnNFT
MsgBurnNFT defines an SDK message for burning a NFT.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `id` | [string](#string) |  |  |
| `denom_id` | [string](#string) |  |  |
| `sender` | [string](#string) |  |  |






<a name="uptick.collection.v1.MsgBurnNFTResponse"></a>

### MsgBurnNFTResponse
MsgBurnNFTResponse defines the Msg/BurnNFT response type.






<a name="uptick.collection.v1.MsgEditNFT"></a>

### MsgEditNFT
MsgEditNFT defines an SDK message for editing a nft.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `id` | [string](#string) |  |  |
| `denom_id` | [string](#string) |  |  |
| `name` | [string](#string) |  |  |
| `uri` | [string](#string) |  |  |
| `data` | [string](#string) |  |  |
| `sender` | [string](#string) |  |  |
| `uri_hash` | [string](#string) |  |  |






<a name="uptick.collection.v1.MsgEditNFTResponse"></a>

### MsgEditNFTResponse
MsgEditNFTResponse defines the Msg/EditNFT response type.






<a name="uptick.collection.v1.MsgIssueDenom"></a>

### MsgIssueDenom
MsgIssueDenom defines an SDK message for creating a new denom.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `id` | [string](#string) |  |  |
| `name` | [string](#string) |  |  |
| `schema` | [string](#string) |  |  |
| `sender` | [string](#string) |  |  |
| `symbol` | [string](#string) |  |  |
| `mint_restricted` | [bool](#bool) |  |  |
| `update_restricted` | [bool](#bool) |  |  |
| `description` | [string](#string) |  |  |
| `uri` | [string](#string) |  |  |
| `uri_hash` | [string](#string) |  |  |
| `data` | [string](#string) |  |  |






<a name="uptick.collection.v1.MsgIssueDenomResponse"></a>

### MsgIssueDenomResponse
MsgIssueDenomResponse defines the Msg/IssueDenom response type.






<a name="uptick.collection.v1.MsgMintNFT"></a>

### MsgMintNFT
MsgMintNFT defines an SDK message for creating a new NFT.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `id` | [string](#string) |  |  |
| `denom_id` | [string](#string) |  |  |
| `name` | [string](#string) |  |  |
| `uri` | [string](#string) |  |  |
| `data` | [string](#string) |  |  |
| `sender` | [string](#string) |  |  |
| `recipient` | [string](#string) |  |  |
| `uri_hash` | [string](#string) |  |  |






<a name="uptick.collection.v1.MsgMintNFTResponse"></a>

### MsgMintNFTResponse
MsgMintNFTResponse defines the Msg/MintNFT response type.






<a name="uptick.collection.v1.MsgTransferDenom"></a>

### MsgTransferDenom
MsgTransferDenom defines an SDK message for transferring an denom to
recipient.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `id` | [string](#string) |  |  |
| `sender` | [string](#string) |  |  |
| `recipient` | [string](#string) |  |  |






<a name="uptick.collection.v1.MsgTransferDenomResponse"></a>

### MsgTransferDenomResponse
MsgTransferDenomResponse defines the Msg/TransferDenom response type.






<a name="uptick.collection.v1.MsgTransferNFT"></a>

### MsgTransferNFT
MsgTransferNFT defines an SDK message for transferring an NFT to recipient.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `id` | [string](#string) |  |  |
| `denom_id` | [string](#string) |  |  |
| `name` | [string](#string) |  |  |
| `uri` | [string](#string) |  |  |
| `data` | [string](#string) |  |  |
| `sender` | [string](#string) |  |  |
| `recipient` | [string](#string) |  |  |
| `uri_hash` | [string](#string) |  |  |






<a name="uptick.collection.v1.MsgTransferNFTResponse"></a>

### MsgTransferNFTResponse
MsgTransferNFTResponse defines the Msg/TransferNFT response type.





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.collection.v1.Msg"></a>

### Msg
Msg defines the nft Msg service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `IssueDenom` | [MsgIssueDenom](#uptick.collection.v1.MsgIssueDenom) | [MsgIssueDenomResponse](#uptick.collection.v1.MsgIssueDenomResponse) | IssueDenom defines a method for issue a denom. | |
| `MintNFT` | [MsgMintNFT](#uptick.collection.v1.MsgMintNFT) | [MsgMintNFTResponse](#uptick.collection.v1.MsgMintNFTResponse) | MintNFT defines a method for mint a new nft | |
| `EditNFT` | [MsgEditNFT](#uptick.collection.v1.MsgEditNFT) | [MsgEditNFTResponse](#uptick.collection.v1.MsgEditNFTResponse) | RefundHTLC defines a method for editing a nft. | |
| `TransferNFT` | [MsgTransferNFT](#uptick.collection.v1.MsgTransferNFT) | [MsgTransferNFTResponse](#uptick.collection.v1.MsgTransferNFTResponse) | TransferNFT defines a method for transferring a nft. | |
| `BurnNFT` | [MsgBurnNFT](#uptick.collection.v1.MsgBurnNFT) | [MsgBurnNFTResponse](#uptick.collection.v1.MsgBurnNFTResponse) | BurnNFT defines a method for burning a nft. | |
| `TransferDenom` | [MsgTransferDenom](#uptick.collection.v1.MsgTransferDenom) | [MsgTransferDenomResponse](#uptick.collection.v1.MsgTransferDenomResponse) | TransferDenom defines a method for transferring a denom. | |

 <!-- end services -->



<a name="uptick/cw721/v1/cw721.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/cw721/v1/cw721.proto



<a name="uptick.cw721.v1.TokenPair"></a>

### TokenPair
TokenPair defines an instance that records a pairing consisting of a native
Cosmos Coin and an CW721 token address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `cw721_address` | [string](#string) |  | address of CW721 contract token |
| `class_id` | [string](#string) |  | cosmos nft class ID to be mapped to |






<a name="uptick.cw721.v1.UIDPair"></a>

### UIDPair
defines the unique id of nft asset


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `cw721_did` | [string](#string) |  | address of CW721 contract token + tokenId |
| `class_did` | [string](#string) |  | cosmos nft class ID to be mapped to + nftId |





 <!-- end messages -->


<a name="uptick.cw721.v1.Owner"></a>

### Owner
Owner enumerates the ownership of a CW721 contract.

| Name | Number | Description |
| ---- | ------ | ----------- |
| OWNER_UNSPECIFIED | 0 | OWNER_UNSPECIFIED defines an invalid/undefined owner. |
| OWNER_MODULE | 1 | OWNER_MODULE cw721 is owned by the cw721 module account. |
| OWNER_EXTERNAL | 2 | EXTERNAL cw721 is owned by an external account. |


 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/cw721/v1/genesis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/cw721/v1/genesis.proto



<a name="uptick.cw721.v1.GenesisState"></a>

### GenesisState
GenesisState defines the module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#uptick.cw721.v1.Params) |  | module parameters |
| `token_pairs` | [TokenPair](#uptick.cw721.v1.TokenPair) | repeated | registered token pairs |






<a name="uptick.cw721.v1.Params"></a>

### Params
Params defines the cw721 module params


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `enable_cw721` | [bool](#bool) |  | parameter to enable the conversion of Cosmos nft <--> CW721 tokens. |
| `enable_evm_hook` | [bool](#bool) |  | parameter to enable the EVM hook that converts an CW721 token to a Cosmos NFT by transferring the Tokens through a MsgEthereumTx to the ModuleAddress Ethereum address. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/cw721/v1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/cw721/v1/query.proto



<a name="uptick.cw721.v1.QueryParamsRequest"></a>

### QueryParamsRequest
QueryParamsRequest is the request type for the Query/Params RPC method.






<a name="uptick.cw721.v1.QueryParamsResponse"></a>

### QueryParamsResponse
QueryParamsResponse is the response type for the Query/Params RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#uptick.cw721.v1.Params) |  |  |






<a name="uptick.cw721.v1.QueryTokenPairRequest"></a>

### QueryTokenPairRequest
QueryTokenPairRequest is the request type for the Query/TokenPair RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token` | [string](#string) |  | token identifier can be either the hex contract address of the CW721 or the Cosmos nft classID |






<a name="uptick.cw721.v1.QueryTokenPairResponse"></a>

### QueryTokenPairResponse
QueryTokenPairResponse is the response type for the Query/TokenPair RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token_pair` | [TokenPair](#uptick.cw721.v1.TokenPair) |  |  |






<a name="uptick.cw721.v1.QueryTokenPairsRequest"></a>

### QueryTokenPairsRequest
QueryTokenPairsRequest is the request type for the Query/TokenPairs RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  | pagination defines an optional pagination for the request. |






<a name="uptick.cw721.v1.QueryTokenPairsResponse"></a>

### QueryTokenPairsResponse
QueryTokenPairsResponse is the response type for the Query/TokenPairs RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token_pairs` | [TokenPair](#uptick.cw721.v1.TokenPair) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  | pagination defines the pagination in the response. |






<a name="uptick.cw721.v1.QueryWasmAddressRequest"></a>

### QueryWasmAddressRequest
QueryTokenPairRequest is the request type for the Query/TokenPair RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `port` | [string](#string) |  | token identifier can be either the hex contract address of the ERC721 or the Cosmos nft classID |
| `channel` | [string](#string) |  |  |
| `classId` | [string](#string) |  |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.cw721.v1.Query"></a>

### Query
Query defines the gRPC querier service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `TokenPairs` | [QueryTokenPairsRequest](#uptick.cw721.v1.QueryTokenPairsRequest) | [QueryTokenPairsResponse](#uptick.cw721.v1.QueryTokenPairsResponse) | TokenPairs retrieves registered token pairs | GET|/uptick/cw721/v1/token_pairs|
| `TokenPair` | [QueryTokenPairRequest](#uptick.cw721.v1.QueryTokenPairRequest) | [QueryTokenPairResponse](#uptick.cw721.v1.QueryTokenPairResponse) | TokenPair retrieves a registered token pair | GET|/uptick/cw721/v1/token_pairs/{token}|
| `WasmContract` | [QueryWasmAddressRequest](#uptick.cw721.v1.QueryWasmAddressRequest) | [QueryTokenPairResponse](#uptick.cw721.v1.QueryTokenPairResponse) | WasmContract retrieves a registered wasm contract | GET|/uptick/erc721/v1/wasm_contract/{port}/{channel}/{classId}|
| `Params` | [QueryParamsRequest](#uptick.cw721.v1.QueryParamsRequest) | [QueryParamsResponse](#uptick.cw721.v1.QueryParamsResponse) | Params retrieves the cw721 module params | GET|/uptick/cw721/v1/params|

 <!-- end services -->



<a name="uptick/cw721/v1/tx.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/cw721/v1/tx.proto



<a name="uptick.cw721.v1.MsgConvertC721Response"></a>

### MsgConvertC721Response
MsgConvertCW721Response returns no fields






<a name="uptick.cw721.v1.MsgConvertCW721"></a>

### MsgConvertCW721
MsgConvertCW721 defines a Msg to convert a CW721 token to a native Cosmos
nft.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `contract_address` | [string](#string) |  | CW721 token contract address registered in a token pair |
| `token_ids` | [string](#string) | repeated | tokenID to convert |
| `receiver` | [string](#string) |  | bech32 address to receive native Cosmos coins |
| `sender` | [string](#string) |  | sender hex address from the owner of the given CW721 tokens |
| `class_id` | [string](#string) |  | nft classID to cnvert to CW721 |
| `nft_ids` | [string](#string) | repeated | nftID to cnvert to CW721 |






<a name="uptick.cw721.v1.MsgConvertCW721Response"></a>

### MsgConvertCW721Response
MsgConvertCW721Response returns no fields






<a name="uptick.cw721.v1.MsgConvertNFT"></a>

### MsgConvertNFT
MsgConvertNFT defines a Msg to convert a native Cosmos nft to a CW721 token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  | nft classID to cnvert to CW721 |
| `nft_ids` | [string](#string) | repeated | nftID to cnvert to CW721 |
| `receiver` | [string](#string) |  | recipient hex address to receive CW721 token |
| `sender` | [string](#string) |  | cosmos bech32 address from the owner of the given Cosmos coins |
| `contract_address` | [string](#string) |  | CW721 token contract address registered in a token pair |
| `token_ids` | [string](#string) | repeated | CW721 token id registered in a token pair |






<a name="uptick.cw721.v1.MsgConvertNFTResponse"></a>

### MsgConvertNFTResponse
MsgConvertNFTResponse returns no fields






<a name="uptick.cw721.v1.MsgTransferCW721"></a>

### MsgTransferCW721



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `cw_contract_address` | [string](#string) |  |  |
| `cw_token_ids` | [string](#string) | repeated | tokenID to convert |
| `source_port` | [string](#string) |  | the port on which the packet will be sent |
| `source_channel` | [string](#string) |  | the channel by which the packet will be sent |
| `class_id` | [string](#string) |  | the class_id of tokens to be transferred |
| `cosmos_token_ids` | [string](#string) | repeated | the non fungible tokens to be transferred |
| `cw_sender` | [string](#string) |  | the sender address |
| `cosmos_receiver` | [string](#string) |  | the recipient address on the destination chain |
| `timeout_height` | [ibc.core.client.v1.Height](#ibc.core.client.v1.Height) |  | Timeout height relative to the current block height. The timeout is disabled when set to 0. |
| `timeout_timestamp` | [uint64](#uint64) |  | Timeout timestamp in absolute nanoseconds since unix epoch. The timeout is disabled when set to 0. |
| `memo` | [string](#string) |  | optional memo |






<a name="uptick.cw721.v1.MsgTransferCW721Response"></a>

### MsgTransferCW721Response






 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.cw721.v1.Msg"></a>

### Msg
Msg defines the cw721 Msg service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `ConvertNFT` | [MsgConvertNFT](#uptick.cw721.v1.MsgConvertNFT) | [MsgConvertNFTResponse](#uptick.cw721.v1.MsgConvertNFTResponse) | ConvertNFT mints a CW721 representation of the native Cosmos nft that is registered on the token mapping. | GET|/uptick/cw721/v1/tx/convert_nft|
| `ConvertCW721` | [MsgConvertCW721](#uptick.cw721.v1.MsgConvertCW721) | [MsgConvertCW721](#uptick.cw721.v1.MsgConvertCW721) | ConvertCW721 mints a native Cosmos coin representation of the CW721 token contract that is registered on the token mapping. | GET|/uptick/cw721/v1/tx/convert_cw721|
| `TransferCW721` | [MsgTransferCW721](#uptick.cw721.v1.MsgTransferCW721) | [MsgTransferCW721Response](#uptick.cw721.v1.MsgTransferCW721Response) |  | GET|/uptick/cw721/v1/tx/ibc-transfer-cw721|

 <!-- end services -->



<a name="uptick/erc20/v1/erc20.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/erc20/v1/erc20.proto



<a name="uptick.erc20.v1.RegisterCoinProposal"></a>

### RegisterCoinProposal
RegisterCoinProposal is a gov Content type to register a token pair


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `title` | [string](#string) |  | title of the proposal |
| `description` | [string](#string) |  | proposal description |
| `metadata` | [cosmos.bank.v1beta1.Metadata](#cosmos.bank.v1beta1.Metadata) |  | token pair of Cosmos native denom and ERC20 token address |






<a name="uptick.erc20.v1.RegisterERC20Proposal"></a>

### RegisterERC20Proposal
RegisterCoinProposal is a gov Content type to register a token pair


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `title` | [string](#string) |  | title of the proposal |
| `description` | [string](#string) |  | proposal description |
| `erc20address` | [string](#string) |  | contract address of ERC20 token |






<a name="uptick.erc20.v1.ToggleTokenRelayProposal"></a>

### ToggleTokenRelayProposal
ToggleTokenRelayProposal is a gov Content type to toggle
the internal relaying of a token pair.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `title` | [string](#string) |  | title of the proposal |
| `description` | [string](#string) |  | proposal description |
| `token` | [string](#string) |  | token identifier can be either the hex contract address of the ERC20 or the Cosmos base denomination |






<a name="uptick.erc20.v1.TokenPair"></a>

### TokenPair
TokenPair defines an instance that records pairing consisting of a Cosmos
native Coin and an ERC20 token address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `erc20_address` | [string](#string) |  | address of ERC20 contract token |
| `denom` | [string](#string) |  | cosmos base denomination to be mapped to |
| `enabled` | [bool](#bool) |  | shows token mapping enable status |
| `contract_owner` | [Owner](#uptick.erc20.v1.Owner) |  | ERC20 owner address ENUM (0 invalid, 1 ModuleAccount, 2 external address) |






<a name="uptick.erc20.v1.UpdateTokenPairERC20Proposal"></a>

### UpdateTokenPairERC20Proposal
UpdateTokenPairERC20Proposal is a gov Content type to update a token pair's
ERC20 contract address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `title` | [string](#string) |  | title of the proposal |
| `description` | [string](#string) |  | proposal description |
| `erc20_address` | [string](#string) |  | contract address of ERC20 token |
| `new_erc20_address` | [string](#string) |  | new address of ERC20 token contract |





 <!-- end messages -->


<a name="uptick.erc20.v1.Owner"></a>

### Owner
Owner enumerates the ownership of a ERC20 contract.

| Name | Number | Description |
| ---- | ------ | ----------- |
| OWNER_UNSPECIFIED | 0 | OWNER_UNSPECIFIED defines an invalid/undefined owner. |
| OWNER_MODULE | 1 | OWNER_MODULE erc20 is owned by the erc20 module account. |
| OWNER_EXTERNAL | 2 | EXTERNAL erc20 is owned by an external account. |


 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/erc20/v1/genesis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/erc20/v1/genesis.proto



<a name="uptick.erc20.v1.GenesisState"></a>

### GenesisState
GenesisState defines the module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#uptick.erc20.v1.Params) |  | module parameters |
| `token_pairs` | [TokenPair](#uptick.erc20.v1.TokenPair) | repeated | registered token pairs |






<a name="uptick.erc20.v1.Params"></a>

### Params
Params defines the erc20 module params


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `enable_erc20` | [bool](#bool) |  | parameter to enable the intrarelaying of Cosmos coins <--> ERC20 tokens. |
| `enable_evm_hook` | [bool](#bool) |  | parameter to enable the EVM hook to convert an ERC20 token to a Cosmos Coin by transferring the Tokens through a MsgEthereumTx to the ModuleAddress Ethereum address. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/erc20/v1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/erc20/v1/query.proto



<a name="uptick.erc20.v1.QueryParamsRequest"></a>

### QueryParamsRequest
QueryParamsRequest is the request type for the Query/Params RPC method.






<a name="uptick.erc20.v1.QueryParamsResponse"></a>

### QueryParamsResponse
QueryParamsResponse is the response type for the Query/Params RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#uptick.erc20.v1.Params) |  |  |






<a name="uptick.erc20.v1.QueryTokenPairRequest"></a>

### QueryTokenPairRequest
QueryTokenPairRequest is the request type for the Query/TokenPair RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token` | [string](#string) |  | token identifier can be either the hex contract address of the ERC20 or the Cosmos base denomination |






<a name="uptick.erc20.v1.QueryTokenPairResponse"></a>

### QueryTokenPairResponse
QueryTokenPairResponse is the response type for the Query/TokenPair RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token_pair` | [TokenPair](#uptick.erc20.v1.TokenPair) |  |  |






<a name="uptick.erc20.v1.QueryTokenPairsRequest"></a>

### QueryTokenPairsRequest
QueryTokenPairsRequest is the request type for the Query/TokenPairs RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  | pagination defines an optional pagination for the request. |






<a name="uptick.erc20.v1.QueryTokenPairsResponse"></a>

### QueryTokenPairsResponse
QueryTokenPairsResponse is the response type for the Query/TokenPairs RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token_pairs` | [TokenPair](#uptick.erc20.v1.TokenPair) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  | pagination defines the pagination in the response. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.erc20.v1.Query"></a>

### Query
Query defines the gRPC querier service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `TokenPairs` | [QueryTokenPairsRequest](#uptick.erc20.v1.QueryTokenPairsRequest) | [QueryTokenPairsResponse](#uptick.erc20.v1.QueryTokenPairsResponse) | Retrieves registered token pairs | GET|/uptick/erc20/v1/token_pairs|
| `TokenPair` | [QueryTokenPairRequest](#uptick.erc20.v1.QueryTokenPairRequest) | [QueryTokenPairResponse](#uptick.erc20.v1.QueryTokenPairResponse) | Retrieves a registered token pair | GET|/uptick/erc20/v1/token_pairs/{token}|
| `Params` | [QueryParamsRequest](#uptick.erc20.v1.QueryParamsRequest) | [QueryParamsResponse](#uptick.erc20.v1.QueryParamsResponse) | Params retrieves the erc20 module params | GET|/uptick/erc20/v1/params|

 <!-- end services -->



<a name="uptick/erc20/v1/tx.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/erc20/v1/tx.proto



<a name="uptick.erc20.v1.MsgConvertCoin"></a>

### MsgConvertCoin
MsgConvertCoin defines a Msg to convert a Cosmos Coin to a ERC20 token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `coin` | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) |  | Cosmos coin which denomination is registered on erc20 bridge. The coin amount defines the total ERC20 tokens to convert. |
| `receiver` | [string](#string) |  | recipient hex address to receive ERC20 token |
| `sender` | [string](#string) |  | cosmos bech32 address from the owner of the given ERC20 tokens |






<a name="uptick.erc20.v1.MsgConvertCoinResponse"></a>

### MsgConvertCoinResponse
MsgConvertCoinResponse returns no fields






<a name="uptick.erc20.v1.MsgConvertERC20"></a>

### MsgConvertERC20
MsgConvertERC20 defines a Msg to convert an ERC20 token to a Cosmos SDK coin.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `contract_address` | [string](#string) |  | ERC20 token contract address registered on erc20 bridge |
| `amount` | [string](#string) |  | amount of ERC20 tokens to mint |
| `receiver` | [string](#string) |  | bech32 address to receive SDK coins. |
| `sender` | [string](#string) |  | sender hex address from the owner of the given ERC20 tokens |






<a name="uptick.erc20.v1.MsgConvertERC20Response"></a>

### MsgConvertERC20Response
MsgConvertERC20Response returns no fields






<a name="uptick.erc20.v1.MsgTransferERC20"></a>

### MsgTransferERC20



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `evm_contract_address` | [string](#string) |  |  |
| `amount` | [string](#string) |  | tokenID to convert |
| `source_port` | [string](#string) |  | the port on which the packet will be sent |
| `source_channel` | [string](#string) |  | the channel by which the packet will be sent |
| `evm_sender` | [string](#string) |  | the sender address |
| `cosmos_receiver` | [string](#string) |  | the recipient address on the destination chain |
| `timeout_height` | [ibc.core.client.v1.Height](#ibc.core.client.v1.Height) |  | Timeout height relative to the current block height. The timeout is disabled when set to 0. |
| `timeout_timestamp` | [uint64](#uint64) |  | Timeout timestamp in absolute nanoseconds since unix epoch. The timeout is disabled when set to 0. |
| `memo` | [string](#string) |  | optional memo |






<a name="uptick.erc20.v1.MsgTransferERC20Response"></a>

### MsgTransferERC20Response






 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.erc20.v1.Msg"></a>

### Msg
Msg defines the erc20 Msg service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `ConvertCoin` | [MsgConvertCoin](#uptick.erc20.v1.MsgConvertCoin) | [MsgConvertCoinResponse](#uptick.erc20.v1.MsgConvertCoinResponse) | ConvertCoin mints a ERC20 representation of the SDK Coin denom that is registered on the token mapping. | GET|/uptick/erc20/v1/tx/convert_coin|
| `ConvertERC20` | [MsgConvertERC20](#uptick.erc20.v1.MsgConvertERC20) | [MsgConvertERC20Response](#uptick.erc20.v1.MsgConvertERC20Response) | ConvertERC20 mints a Cosmos coin representation of the ERC20 token contract that is registered on the token mapping. | GET|/uptick/erc20/v1/tx/convert_erc20|
| `TransferERC20` | [MsgTransferERC20](#uptick.erc20.v1.MsgTransferERC20) | [MsgTransferERC20Response](#uptick.erc20.v1.MsgTransferERC20Response) |  | GET|/uptick/erc20/v1/tx/ibc-transfer-erc20|

 <!-- end services -->



<a name="uptick/evmIBC/v1/evmIBC.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/evmIBC/v1/evmIBC.proto



<a name="uptick.evmIBC.v1.TokenPair"></a>

### TokenPair
TokenPair defines an instance that records a pairing consisting of a native
Cosmos Coin and an ERC721 token address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `erc721_address` | [string](#string) |  | address of ERC721 contract token |
| `class_id` | [string](#string) |  | cosmos nft class ID to be mapped to |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/evmIBC/v1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/evmIBC/v1/query.proto



<a name="uptick.evmIBC.v1.QueryEvmAddressRequest"></a>

### QueryEvmAddressRequest
QueryEvmAddressRequest is the request type for the Query/TokenPair RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `port` | [string](#string) |  | token identifier can be either the hex contract address of the ERC721 or the Cosmos nft classID |
| `channel` | [string](#string) |  |  |
| `classId` | [string](#string) |  |  |






<a name="uptick.evmIBC.v1.QueryTokenPairResponse"></a>

### QueryTokenPairResponse
QueryTokenPairResponse is the response type for the Query/TokenPair RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token_pair` | [TokenPair](#uptick.evmIBC.v1.TokenPair) |  |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.evmIBC.v1.Query"></a>

### Query
Query defines the gRPC queried service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `EvmContract` | [QueryEvmAddressRequest](#uptick.evmIBC.v1.QueryEvmAddressRequest) | [QueryTokenPairResponse](#uptick.evmIBC.v1.QueryTokenPairResponse) | EvmContract retrieves a registered evm contract | GET|/uptick/evmIBC/v1/evm_contract/{port}/{channel}/{classId}|

 <!-- end services -->



<a name="uptick/evmIBC/v1/tx.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/evmIBC/v1/tx.proto



<a name="uptick.evmIBC.v1.MsgTransferERC721"></a>

### MsgTransferERC721



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `evm_contract_address` | [string](#string) |  |  |
| `evm_token_ids` | [string](#string) | repeated | tokenID to convert |
| `source_port` | [string](#string) |  | the port on which the packet will be sent |
| `source_channel` | [string](#string) |  | the channel by which the packet will be sent |
| `class_id` | [string](#string) |  | the class_id of tokens to be transferred |
| `cosmos_token_ids` | [string](#string) | repeated | the non fungible tokens to be transferred |
| `evm_sender` | [string](#string) |  | the sender address |
| `cosmos_receiver` | [string](#string) |  | the recipient address on the destination chain |
| `timeout_height` | [ibc.core.client.v1.Height](#ibc.core.client.v1.Height) |  | Timeout height relative to the current block height. The timeout is disabled when set to 0. |
| `timeout_timestamp` | [uint64](#uint64) |  | Timeout timestamp in absolute nanoseconds since unix epoch. The timeout is disabled when set to 0. |
| `memo` | [string](#string) |  | optional memo |






<a name="uptick.evmIBC.v1.MsgTransferERC721Response"></a>

### MsgTransferERC721Response






 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.evmIBC.v1.Msg"></a>

### Msg
Msg defines the erc721 Msg service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `TransferERC721` | [MsgTransferERC721](#uptick.evmIBC.v1.MsgTransferERC721) | [MsgTransferERC721Response](#uptick.evmIBC.v1.MsgTransferERC721Response) |  | GET|/uptick/erc721/v1/tx/ibc-transfer-erc721|

 <!-- end services -->



<a name="uptick/nft/v1beta1/event.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/nft/v1beta1/event.proto



<a name="cosmos.nft.v1beta1.EventBurn"></a>

### EventBurn
EventBurn is emitted on Burn


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |
| `id` | [string](#string) |  |  |
| `owner` | [string](#string) |  |  |






<a name="cosmos.nft.v1beta1.EventMint"></a>

### EventMint
EventMint is emitted on Mint


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |
| `id` | [string](#string) |  |  |
| `owner` | [string](#string) |  |  |






<a name="cosmos.nft.v1beta1.EventSend"></a>

### EventSend
EventSend is emitted on Msg/Send


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |
| `id` | [string](#string) |  |  |
| `sender` | [string](#string) |  |  |
| `receiver` | [string](#string) |  |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/nft/v1beta1/nft.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/nft/v1beta1/nft.proto



<a name="cosmos.nft.v1beta1.Class"></a>

### Class
Class defines the class of the nft type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `id` | [string](#string) |  | id defines the unique identifier of the NFT classification, similar to the contract address of ERC721 |
| `name` | [string](#string) |  | name defines the human-readable name of the NFT classification. Optional |
| `symbol` | [string](#string) |  | symbol is an abbreviated name for nft classification. Optional |
| `description` | [string](#string) |  | description is a brief description of nft classification. Optional |
| `uri` | [string](#string) |  | uri for the class metadata stored off chain. It can define schema for Class and NFT `Data` attributes. Optional |
| `uri_hash` | [string](#string) |  | uri_hash is a hash of the document pointed by uri. Optional |
| `data` | [google.protobuf.Any](#google.protobuf.Any) |  | data is the app specific metadata of the NFT class. Optional |






<a name="cosmos.nft.v1beta1.NFT"></a>

### NFT
NFT defines the NFT.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  | class_id associated with the NFT, similar to the contract address of ERC721 |
| `id` | [string](#string) |  | id is a unique identifier of the NFT |
| `uri` | [string](#string) |  | uri for the NFT metadata stored off chain |
| `uri_hash` | [string](#string) |  | uri_hash is a hash of the document pointed by uri |
| `data` | [google.protobuf.Any](#google.protobuf.Any) |  | data is an app specific data of the NFT. Optional |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/nft/v1beta1/genesis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/nft/v1beta1/genesis.proto



<a name="cosmos.nft.v1beta1.Entry"></a>

### Entry
Entry Defines all nft owned by a person


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `owner` | [string](#string) |  | owner is the owner address of the following nft |
| `nfts` | [NFT](#cosmos.nft.v1beta1.NFT) | repeated | nfts is a group of nfts of the same owner |






<a name="cosmos.nft.v1beta1.GenesisState"></a>

### GenesisState
GenesisState defines the nft module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `classes` | [Class](#cosmos.nft.v1beta1.Class) | repeated | class defines the class of the nft type. |
| `entries` | [Entry](#cosmos.nft.v1beta1.Entry) | repeated |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/nft/v1beta1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/nft/v1beta1/query.proto



<a name="cosmos.nft.v1beta1.QueryBalanceRequest"></a>

### QueryBalanceRequest
QueryBalanceRequest is the request type for the Query/Balance RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |
| `owner` | [string](#string) |  |  |






<a name="cosmos.nft.v1beta1.QueryBalanceResponse"></a>

### QueryBalanceResponse
QueryBalanceResponse is the response type for the Query/Balance RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `amount` | [uint64](#uint64) |  |  |






<a name="cosmos.nft.v1beta1.QueryClassRequest"></a>

### QueryClassRequest
QueryClassRequest is the request type for the Query/Class RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |






<a name="cosmos.nft.v1beta1.QueryClassResponse"></a>

### QueryClassResponse
QueryClassResponse is the response type for the Query/Class RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class` | [Class](#cosmos.nft.v1beta1.Class) |  |  |






<a name="cosmos.nft.v1beta1.QueryClassesRequest"></a>

### QueryClassesRequest
QueryClassesRequest is the request type for the Query/Classes RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  | pagination defines an optional pagination for the request. |






<a name="cosmos.nft.v1beta1.QueryClassesResponse"></a>

### QueryClassesResponse
QueryClassesResponse is the response type for the Query/Classes RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `classes` | [Class](#cosmos.nft.v1beta1.Class) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  |  |






<a name="cosmos.nft.v1beta1.QueryNFTRequest"></a>

### QueryNFTRequest
QueryNFTRequest is the request type for the Query/NFT RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |
| `id` | [string](#string) |  |  |






<a name="cosmos.nft.v1beta1.QueryNFTResponse"></a>

### QueryNFTResponse
QueryNFTResponse is the response type for the Query/NFT RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `nft` | [NFT](#cosmos.nft.v1beta1.NFT) |  |  |






<a name="cosmos.nft.v1beta1.QueryNFTsRequest"></a>

### QueryNFTsRequest
QueryNFTstRequest is the request type for the Query/NFTs RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |
| `owner` | [string](#string) |  |  |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  |  |






<a name="cosmos.nft.v1beta1.QueryNFTsResponse"></a>

### QueryNFTsResponse
QueryNFTsResponse is the response type for the Query/NFTs RPC methods


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `nfts` | [NFT](#cosmos.nft.v1beta1.NFT) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  |  |






<a name="cosmos.nft.v1beta1.QueryOwnerRequest"></a>

### QueryOwnerRequest
QueryOwnerRequest is the request type for the Query/Owner RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |
| `id` | [string](#string) |  |  |






<a name="cosmos.nft.v1beta1.QueryOwnerResponse"></a>

### QueryOwnerResponse
QueryOwnerResponse is the response type for the Query/Owner RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `owner` | [string](#string) |  |  |






<a name="cosmos.nft.v1beta1.QuerySupplyRequest"></a>

### QuerySupplyRequest
QuerySupplyRequest is the request type for the Query/Supply RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |






<a name="cosmos.nft.v1beta1.QuerySupplyResponse"></a>

### QuerySupplyResponse
QuerySupplyResponse is the response type for the Query/Supply RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `amount` | [uint64](#uint64) |  |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="cosmos.nft.v1beta1.Query"></a>

### Query
Query defines the gRPC querier service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `Balance` | [QueryBalanceRequest](#cosmos.nft.v1beta1.QueryBalanceRequest) | [QueryBalanceResponse](#cosmos.nft.v1beta1.QueryBalanceResponse) | Balance queries the number of NFTs of a given class owned by the owner, same as balanceOf in ERC721 | GET|/cosmos/nft/v1beta1/balance/{owner}/{class_id}|
| `Owner` | [QueryOwnerRequest](#cosmos.nft.v1beta1.QueryOwnerRequest) | [QueryOwnerResponse](#cosmos.nft.v1beta1.QueryOwnerResponse) | Owner queries the owner of the NFT based on its class and id, same as ownerOf in ERC721 | GET|/cosmos/nft/v1beta1/owner/{class_id}/{id}|
| `Supply` | [QuerySupplyRequest](#cosmos.nft.v1beta1.QuerySupplyRequest) | [QuerySupplyResponse](#cosmos.nft.v1beta1.QuerySupplyResponse) | Supply queries the number of NFTs from the given class, same as totalSupply of ERC721. | GET|/cosmos/nft/v1beta1/supply/{class_id}|
| `NFTs` | [QueryNFTsRequest](#cosmos.nft.v1beta1.QueryNFTsRequest) | [QueryNFTsResponse](#cosmos.nft.v1beta1.QueryNFTsResponse) | NFTs queries all NFTs of a given class or owner,choose at least one of the two, similar to tokenByIndex in ERC721Enumerable | GET|/cosmos/nft/v1beta1/nfts|
| `NFT` | [QueryNFTRequest](#cosmos.nft.v1beta1.QueryNFTRequest) | [QueryNFTResponse](#cosmos.nft.v1beta1.QueryNFTResponse) | NFT queries an NFT based on its class and id. | GET|/cosmos/nft/v1beta1/nfts/{class_id}/{id}|
| `Class` | [QueryClassRequest](#cosmos.nft.v1beta1.QueryClassRequest) | [QueryClassResponse](#cosmos.nft.v1beta1.QueryClassResponse) | Class queries an NFT class based on its id | GET|/cosmos/nft/v1beta1/classes/{class_id}|
| `Classes` | [QueryClassesRequest](#cosmos.nft.v1beta1.QueryClassesRequest) | [QueryClassesResponse](#cosmos.nft.v1beta1.QueryClassesResponse) | Classes queries all NFT classes | GET|/cosmos/nft/v1beta1/classes|

 <!-- end services -->



<a name="uptick/nft/v1beta1/tx.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/nft/v1beta1/tx.proto



<a name="cosmos.nft.v1beta1.MsgSend"></a>

### MsgSend
MsgSend represents a message to send a nft from one account to another
account.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  | class_id defines the unique identifier of the nft classification, similar to the contract address of ERC721 |
| `id` | [string](#string) |  | id defines the unique identification of nft |
| `sender` | [string](#string) |  | sender is the address of the owner of nft |
| `receiver` | [string](#string) |  | receiver is the receiver address of nft |






<a name="cosmos.nft.v1beta1.MsgSendResponse"></a>

### MsgSendResponse
MsgSendResponse defines the Msg/Send response type.





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="cosmos.nft.v1beta1.Msg"></a>

### Msg
Msg defines the nft Msg service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `Send` | [MsgSend](#cosmos.nft.v1beta1.MsgSend) | [MsgSendResponse](#cosmos.nft.v1beta1.MsgSendResponse) | Send defines a method to send a nft from one account to another account. | |

 <!-- end services -->



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

