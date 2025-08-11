# Protocol Documentation

# uptick/collection/v1/collection.proto




## BaseNFT
BaseNFT defines a non-fungible token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | string |  |  |
| name | string |  |  |
| uri | string |  |  |
| data | string |  |  |
| owner | string |  |  |
| uri_hash | string |  |  |




## Collection
Collection defines a type of collection


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| denom | Denom |  |  |
| nfts | BaseNFT | repeated |  |




## Denom
Denom defines a type of NFT


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | string |  |  |
| name | string |  |  |
| schema | string |  |  |
| creator | string |  |  |
| symbol | string |  |  |
| mint_restricted | bool |  |  |
| update_restricted | bool |  |  |
| description | string |  |  |
| uri | string |  |  |
| uri_hash | string |  |  |
| data | string |  |  |




## DenomMetadata
DenomMetadata defines the metadata for a Denom
Contains information about creator, schema, restrictions and additional data


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| creator | string |  |  |
| schema | string |  |  |
| mint_restricted | bool |  |  |
| update_restricted | bool |  |  |
| data | string |  |  |




## IDCollection
IDCollection defines a type of collection with specified ID


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| denom_id | string |  |  |
| token_ids | string | repeated |  |




## NFTMetadata
NFTMetadata defines the metadata for a NFT
Contains basic information like name and data


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | string |  |  |
| data | string |  |  |




## Owner
Owner defines a type of owner


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | string |  |  |
| id_collections | IDCollection | repeated |  |












# uptick/collection/v1/genesis.proto




## GenesisState
GenesisState defines the collection module's genesis state


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| collections | Collection | repeated |  |












# uptick/collection/v1/query.proto




## QueryCollectionRequest
QueryCollectionRequest is the request type for the Query/Collection RPC
method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| denom_id | string |  |  |
| pagination | cosmos.base.query.v1beta1.PageRequest |  | pagination defines an optional pagination for the request. |




## QueryCollectionResponse
QueryCollectionResponse is the response type for the Query/Collection RPC
method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| collection | Collection |  |  |
| pagination | cosmos.base.query.v1beta1.PageResponse |  |  |




## QueryDenomRequest
QueryDenomRequest is the request type for the Query/Denom RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| denom_id | string |  |  |




## QueryDenomResponse
QueryDenomResponse is the response type for the Query/Denom RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| denom | Denom |  |  |




## QueryDenomsRequest
QueryDenomsRequest is the request type for the Query/Denoms RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pagination | cosmos.base.query.v1beta1.PageRequest |  | pagination defines an optional pagination for the request. |




## QueryDenomsResponse
QueryDenomsResponse is the response type for the Query/Denoms RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| denoms | Denom | repeated |  |
| pagination | cosmos.base.query.v1beta1.PageResponse |  |  |




## QueryNFTRequest
QueryNFTRequest is the request type for the Query/NFT RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| denom_id | string |  |  |
| token_id | string |  |  |




## QueryNFTResponse
QueryNFTResponse is the response type for the Query/NFT RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nft | BaseNFT |  |  |




## QueryNFTsOfOwnerRequest
QueryNFTsOfOwnerRequest is the request type for the Query/Owner RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| denom_id | string |  |  |
| owner | string |  |  |
| pagination | cosmos.base.query.v1beta1.PageRequest |  | pagination defines an optional pagination for the request. |




## QueryNFTsOfOwnerResponse
QueryNFTsOfOwnerResponse is the response type for the Query/Owner RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | Owner |  |  |
| pagination | cosmos.base.query.v1beta1.PageResponse |  |  |




## QuerySupplyRequest
QuerySupplyRequest is the request type for the Query/HTLC RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| denom_id | string |  |  |
| owner | string |  |  |




## QuerySupplyResponse
QuerySupplyResponse is the response type for the Query/Supply RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| amount | uint64 |  |  |










## Query
Query defines the gRPC querier service for NFT module

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Supply | QuerySupplyRequest | QuerySupplyResponse | Supply queries the total supply of a given denom or owner |
| NFTsOfOwner | QueryNFTsOfOwnerRequest | QueryNFTsOfOwnerResponse | NFTsOfOwner queries the NFTs of the specified owner |
| Collection | QueryCollectionRequest | QueryCollectionResponse | Collection queries the NFTs of the specified denom |
| Denom | QueryDenomRequest | QueryDenomResponse | Denom queries the definition of a given denom |
| Denoms | QueryDenomsRequest | QueryDenomsResponse | Denoms queries all the denoms |
| NFT | QueryNFTRequest | QueryNFTResponse | NFT queries the NFT for the given denom and token ID |




# uptick/collection/v1/tx.proto




## MsgBurnNFT
MsgBurnNFT defines an SDK message for burning a NFT.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | string |  |  |
| denom_id | string |  |  |
| sender | string |  |  |




## MsgBurnNFTResponse
MsgBurnNFTResponse defines the Msg/BurnNFT response type.




## MsgEditNFT
MsgEditNFT defines an SDK message for editing a nft.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | string |  |  |
| denom_id | string |  |  |
| name | string |  |  |
| uri | string |  |  |
| data | string |  |  |
| sender | string |  |  |
| uri_hash | string |  |  |




## MsgEditNFTResponse
MsgEditNFTResponse defines the Msg/EditNFT response type.




## MsgIssueDenom
MsgIssueDenom defines an SDK message for creating a new denom.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | string |  |  |
| name | string |  |  |
| schema | string |  |  |
| sender | string |  |  |
| symbol | string |  |  |
| mint_restricted | bool |  |  |
| update_restricted | bool |  |  |
| description | string |  |  |
| uri | string |  |  |
| uri_hash | string |  |  |
| data | string |  |  |




## MsgIssueDenomResponse
MsgIssueDenomResponse defines the Msg/IssueDenom response type.




## MsgMintNFT
MsgMintNFT defines an SDK message for creating a new NFT.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | string |  |  |
| denom_id | string |  |  |
| name | string |  |  |
| uri | string |  |  |
| data | string |  |  |
| sender | string |  |  |
| recipient | string |  |  |
| uri_hash | string |  |  |




## MsgMintNFTResponse
MsgMintNFTResponse defines the Msg/MintNFT response type.




## MsgTransferDenom
MsgTransferDenom defines an SDK message for transferring an denom to
recipient.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | string |  |  |
| sender | string |  |  |
| recipient | string |  |  |




## MsgTransferDenomResponse
MsgTransferDenomResponse defines the Msg/TransferDenom response type.




## MsgTransferNFT
MsgTransferNFT defines an SDK message for transferring an NFT to recipient.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | string |  |  |
| denom_id | string |  |  |
| name | string |  |  |
| uri | string |  |  |
| data | string |  |  |
| sender | string |  |  |
| recipient | string |  |  |
| uri_hash | string |  |  |




## MsgTransferNFTResponse
MsgTransferNFTResponse defines the Msg/TransferNFT response type.










## Msg
Msg defines the nft Msg service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| IssueDenom | MsgIssueDenom | MsgIssueDenomResponse | IssueDenom defines a method for issue a denom. |
| MintNFT | MsgMintNFT | MsgMintNFTResponse | MintNFT defines a method for mint a new nft |
| EditNFT | MsgEditNFT | MsgEditNFTResponse | RefundHTLC defines a method for editing a nft. |
| TransferNFT | MsgTransferNFT | MsgTransferNFTResponse | TransferNFT defines a method for transferring a nft. |
| BurnNFT | MsgBurnNFT | MsgBurnNFTResponse | BurnNFT defines a method for burning a nft. |
| TransferDenom | MsgTransferDenom | MsgTransferDenomResponse | TransferDenom defines a method for transferring a denom. |




# uptick/cw721/v1/cw721.proto




## TokenPair
TokenPair defines an instance that records a pairing consisting of a native
Cosmos Coin and an CW721 token address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cw721_address | string |  | address of CW721 contract token |
| class_id | string |  | cosmos nft class ID to be mapped to |




## UIDPair
defines the unique id of nft asset


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cw721_did | string |  | address of CW721 contract token + tokenId |
| class_did | string |  | cosmos nft class ID to be mapped to + nftId |






## Owner
Owner enumerates the ownership of a CW721 contract.

| Name | Number | Description |
| ---- | ------ | ----------- |
| OWNER_UNSPECIFIED | 0 | OWNER_UNSPECIFIED defines an invalid/undefined owner. |
| OWNER_MODULE | 1 | OWNER_MODULE cw721 is owned by the cw721 module account. |
| OWNER_EXTERNAL | 2 | EXTERNAL cw721 is owned by an external account. |









# uptick/cw721/v1/genesis.proto




## GenesisState
GenesisState defines the module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| params | Params |  | module parameters |
| token_pairs | TokenPair | repeated | registered token pairs |




## Params
Params defines the cw721 module params


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enable_cw721 | bool |  | parameter to enable the conversion of Cosmos nft <--> CW721 tokens. |
| enable_evm_hook | bool |  | parameter to enable the EVM hook that converts an CW721 token to a Cosmos
NFT by transferring the Tokens through a MsgEthereumTx to the
ModuleAddress Ethereum address. |












# uptick/cw721/v1/query.proto




## QueryParamsRequest
QueryParamsRequest is the request type for the Query/Params RPC method.




## QueryParamsResponse
QueryParamsResponse is the response type for the Query/Params RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| params | Params |  |  |




## QueryTokenPairRequest
QueryTokenPairRequest is the request type for the Query/TokenPair RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | string |  | token identifier can be either the hex contract address of the CW721 or
the Cosmos nft classID |




## QueryTokenPairResponse
QueryTokenPairResponse is the response type for the Query/TokenPair RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token_pair | TokenPair |  |  |




## QueryTokenPairsRequest
QueryTokenPairsRequest is the request type for the Query/TokenPairs RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pagination | cosmos.base.query.v1beta1.PageRequest |  | pagination defines an optional pagination for the request. |




## QueryTokenPairsResponse
QueryTokenPairsResponse is the response type for the Query/TokenPairs RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token_pairs | TokenPair | repeated |  |
| pagination | cosmos.base.query.v1beta1.PageResponse |  | pagination defines the pagination in the response. |




## QueryWasmAddressRequest
QueryTokenPairRequest is the request type for the Query/TokenPair RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| port | string |  | token identifier can be either the hex contract address of the ERC721 or
the Cosmos nft classID |
| channel | string |  |  |
| class_id | string |  |  |




## QueryWasmContractResponse
QueryWasmContractResponse is the response type for the Query/WasmContract RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token_pair | TokenPair |  |  |










## Query
Query defines the gRPC querier service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| TokenPairs | QueryTokenPairsRequest | QueryTokenPairsResponse | TokenPairs retrieves registered token pairs |
| TokenPair | QueryTokenPairRequest | QueryTokenPairResponse | TokenPair retrieves a registered token pair |
| WasmContract | QueryWasmAddressRequest | QueryWasmContractResponse | WasmContract retrieves a registered wasm contract |
| Params | QueryParamsRequest | QueryParamsResponse | Params retrieves the cw721 module params |




# uptick/cw721/v1/tx.proto




## MsgConvertC721Response
MsgConvertCW721Response returns no fields




## MsgConvertCW721
MsgConvertCW721 defines a Msg to convert a CW721 token to a native Cosmos
nft.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contract_address | string |  | CW721 token contract address registered in a token pair |
| token_ids | string | repeated | tokenID to convert |
| receiver | string |  | bech32 address to receive native Cosmos coins |
| sender | string |  | sender hex address from the owner of the given CW721 tokens |
| class_id | string |  | nft classID to cnvert to CW721 |
| nft_ids | string | repeated | nftID to cnvert to CW721 |




## MsgConvertCW721Response
MsgConvertCW721Response returns no fields


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contract_address | string |  | CW721 token contract address registered in a token pair |
| token_ids | string | repeated | tokenID to convert |
| receiver | string |  | bech32 address to receive native Cosmos coins |
| sender | string |  | sender hex address from the owner of the given CW721 tokens |
| class_id | string |  | nft classID to cnvert to CW721 |
| nft_ids | string | repeated | nftID to cnvert to CW721 |




## MsgConvertNFT
MsgConvertNFT defines a Msg to convert a native Cosmos nft to a CW721 token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  | nft classID to cnvert to CW721 |
| nft_ids | string | repeated | nftID to cnvert to CW721 |
| receiver | string |  | recipient hex address to receive CW721 token |
| sender | string |  | cosmos bech32 address from the owner of the given Cosmos coins |
| contract_address | string |  | CW721 token contract address registered in a token pair |
| token_ids | string | repeated | CW721 token id registered in a token pair |




## MsgConvertNFTResponse
MsgConvertNFTResponse returns no fields




## MsgTransferCW721
MsgTransferCW721 defines a message to transfer CW721 tokens between chains via IBC
It contains information about the source and destination of the transfer,
token identifiers, timeout parameters and optional memo


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cw_contract_address | string |  |  |
| cw_token_ids | string | repeated | tokenID to convert |
| source_port | string |  | the port on which the packet will be sent |
| source_channel | string |  | the channel by which the packet will be sent |
| class_id | string |  | the class_id of tokens to be transferred |
| cosmos_token_ids | string | repeated | the non fungible tokens to be transferred |
| cw_sender | string |  | the sender address |
| cosmos_receiver | string |  | the recipient address on the destination chain |
| timeout_height | ibc.core.client.v1.Height |  | Timeout height relative to the current block height.
The timeout is disabled when set to 0. |
| timeout_timestamp | uint64 |  | Timeout timestamp in absolute nanoseconds since unix epoch.
The timeout is disabled when set to 0. |
| memo | string |  | optional memo |




## MsgTransferCW721Response
MsgTransferCW721Response defines the response type for TransferCW721 RPC










## Msg
Msg defines the cw721 Msg service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ConvertNFT | MsgConvertNFT | MsgConvertNFTResponse | ConvertNFT mints a CW721 representation of the native Cosmos nft
that is registered on the token mapping. |
| ConvertCW721 | MsgConvertCW721 | MsgConvertCW721Response | ConvertCW721 mints a native Cosmos coin representation of the CW721 token
contract that is registered on the token mapping. |
| TransferCW721 | MsgTransferCW721 | MsgTransferCW721Response | TransferCW721 defines a method to transfer CW721 tokens between chains via IBC |




# uptick/erc20/v1/erc20.proto




## RegisterCoinProposal
RegisterCoinProposal is a gov Content type to register a token pair


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | string |  | title of the proposal |
| description | string |  | proposal description |
| metadata | cosmos.bank.v1beta1.Metadata |  | token pair of Cosmos native denom and ERC20 token address |




## RegisterERC20Proposal
RegisterCoinProposal is a gov Content type to register a token pair


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | string |  | title of the proposal |
| description | string |  | proposal description |
| erc20address | string |  | contract address of ERC20 token |




## ToggleTokenRelayProposal
ToggleTokenRelayProposal is a gov Content type to toggle
the internal relaying of a token pair.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | string |  | title of the proposal |
| description | string |  | proposal description |
| token | string |  | token identifier can be either the hex contract address of the ERC20 or the
Cosmos base denomination |




## TokenPair
TokenPair defines an instance that records pairing consisting of a Cosmos
native Coin and an ERC20 token address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| erc20_address | string |  | address of ERC20 contract token |
| denom | string |  | cosmos base denomination to be mapped to |
| enabled | bool |  | shows token mapping enable status |
| contract_owner | Owner |  | ERC20 owner address ENUM (0 invalid, 1 ModuleAccount, 2 external address) |




## UpdateTokenPairERC20Proposal
UpdateTokenPairERC20Proposal is a gov Content type to update a token pair's
ERC20 contract address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | string |  | title of the proposal |
| description | string |  | proposal description |
| erc20_address | string |  | contract address of ERC20 token |
| new_erc20_address | string |  | new address of ERC20 token contract |






## Owner
Owner enumerates the ownership of a ERC20 contract.

| Name | Number | Description |
| ---- | ------ | ----------- |
| OWNER_UNSPECIFIED | 0 | OWNER_UNSPECIFIED defines an invalid/undefined owner. |
| OWNER_MODULE | 1 | OWNER_MODULE erc20 is owned by the erc20 module account. |
| OWNER_EXTERNAL | 2 | EXTERNAL erc20 is owned by an external account. |









# uptick/erc20/v1/genesis.proto




## GenesisState
GenesisState defines the module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| params | Params |  | module parameters |
| token_pairs | TokenPair | repeated | registered token pairs |




## Params
Params defines the erc20 module params


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enable_erc20 | bool |  | parameter to enable the intrarelaying of Cosmos coins <--> ERC20 tokens. |
| enable_evm_hook | bool |  | parameter to enable the EVM hook to convert an ERC20 token to a Cosmos
Coin by transferring the Tokens through a MsgEthereumTx to the
ModuleAddress Ethereum address. |












# uptick/erc20/v1/query.proto




## QueryParamsRequest
QueryParamsRequest is the request type for the Query/Params RPC method.




## QueryParamsResponse
QueryParamsResponse is the response type for the Query/Params RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| params | Params |  |  |




## QueryTokenPairRequest
QueryTokenPairRequest is the request type for the Query/TokenPair RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | string |  | token identifier can be either the hex contract address of the ERC20 or the
Cosmos base denomination |




## QueryTokenPairResponse
QueryTokenPairResponse is the response type for the Query/TokenPair RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token_pair | TokenPair |  |  |




## QueryTokenPairsRequest
QueryTokenPairsRequest is the request type for the Query/TokenPairs RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pagination | cosmos.base.query.v1beta1.PageRequest |  | pagination defines an optional pagination for the request. |




## QueryTokenPairsResponse
QueryTokenPairsResponse is the response type for the Query/TokenPairs RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token_pairs | TokenPair | repeated |  |
| pagination | cosmos.base.query.v1beta1.PageResponse |  | pagination defines the pagination in the response. |










## Query
Query defines the gRPC querier service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| TokenPairs | QueryTokenPairsRequest | QueryTokenPairsResponse | Retrieves registered token pairs |
| TokenPair | QueryTokenPairRequest | QueryTokenPairResponse | Retrieves a registered token pair |
| Params | QueryParamsRequest | QueryParamsResponse | Params retrieves the erc20 module params |




# uptick/erc20/v1/tx.proto




## MsgConvertCoin
MsgConvertCoin defines a Msg to convert a Cosmos Coin to a ERC20 token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| coin | cosmos.base.v1beta1.Coin |  | Cosmos coin which denomination is registered on erc20 bridge.
The coin amount defines the total ERC20 tokens to convert. |
| receiver | string |  | recipient hex address to receive ERC20 token |
| sender | string |  | cosmos bech32 address from the owner of the given ERC20 tokens |




## MsgConvertCoinResponse
MsgConvertCoinResponse returns no fields




## MsgConvertERC20
MsgConvertERC20 defines a Msg to convert an ERC20 token to a Cosmos SDK coin.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contract_address | string |  | ERC20 token contract address registered on erc20 bridge |
| amount | string |  | amount of ERC20 tokens to mint |
| receiver | string |  | bech32 address to receive SDK coins. |
| sender | string |  | sender hex address from the owner of the given ERC20 tokens |




## MsgConvertERC20Response
MsgConvertERC20Response returns no fields




## MsgTransferERC20
MsgTransferERC20 defines a message to transfer ERC20 tokens between chains via IBC
It contains information about the token contract, amount, source and destination of the transfer,
timeout parameters and optional memo


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| evm_contract_address | string |  |  |
| amount | string |  | tokenID to convert |
| source_port | string |  | the port on which the packet will be sent |
| source_channel | string |  | the channel by which the packet will be sent |
| cosmos_sender | string |  | the sender address |
| cosmos_receiver | string |  | the recipient address on the destination chain |
| timeout_height | ibc.core.client.v1.Height |  | Timeout height relative to the current block height.
The timeout is disabled when set to 0. |
| timeout_timestamp | uint64 |  | Timeout timestamp in absolute nanoseconds since unix epoch.
The timeout is disabled when set to 0. |
| memo | string |  | optional memo |




## MsgTransferERC20Response
MsgTransferERC20Response defines the response type for TransferERC20 RPC










## Msg
Msg defines the erc20 Msg service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ConvertCoin | MsgConvertCoin | MsgConvertCoinResponse | ConvertCoin mints a ERC20 representation of the SDK Coin denom that is
registered on the token mapping. |
| ConvertERC20 | MsgConvertERC20 | MsgConvertERC20Response | ConvertERC20 mints a Cosmos coin representation of the ERC20 token contract
that is registered on the token mapping. |
| TransferERC20 | MsgTransferERC20 | MsgTransferERC20Response | TransferERC20 defines a method to transfer ERC20 tokens between chains via IBC |




# uptick/evm_ibc/v1/evm_ibc.proto




## TokenPair
TokenPair defines an instance that records a pairing consisting of a native
Cosmos Coin and an ERC721 token address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| erc721_address | string |  | address of ERC721 contract token |
| class_id | string |  | cosmos nft class ID to be mapped to |












# uptick/evm_ibc/v1/query.proto




## QueryEvmAddressRequest
QueryEvmAddressRequest is the request type for the Query/TokenPair RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| port | string |  | token identifier can be either the hex contract address of the ERC721 or
the Cosmos nft classID |
| channel | string |  |  |
| class_id | string |  |  |




## QueryTokenPairResponse
QueryTokenPairResponse is the response type for the Query/TokenPair RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token_pair | TokenPair |  |  |










## Query
Query defines the gRPC queried service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| EvmContract | QueryEvmAddressRequest | QueryTokenPairResponse | EvmContract retrieves a registered evm contract |




# uptick/evm_ibc/v1/tx.proto




## MsgTransferERC721
MsgTransferERC721 defines a message to transfer ERC721 tokens between chains via IBC
It contains information about the token contract, token IDs, source and destination of the transfer,
timeout parameters and optional memo


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| evm_contract_address | string |  |  |
| evm_token_ids | string | repeated | tokenID to convert |
| source_port | string |  | the port on which the packet will be sent |
| source_channel | string |  | the channel by which the packet will be sent |
| class_id | string |  | the class_id of tokens to be transferred |
| cosmos_token_ids | string | repeated | the non fungible tokens to be transferred |
| cosmos_sender | string |  | the sender address |
| cosmos_receiver | string |  | the recipient address on the destination chain |
| timeout_height | ibc.core.client.v1.Height |  | Timeout height relative to the current block height.
The timeout is disabled when set to 0. |
| timeout_timestamp | uint64 |  | Timeout timestamp in absolute nanoseconds since unix epoch.
The timeout is disabled when set to 0. |
| memo | string |  | optional memo |




## MsgTransferERC721Response
MsgTransferERC721Response defines the response type for TransferERC721 RPC










## Msg
Msg defines the erc721 Msg service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| TransferERC721 | MsgTransferERC721 | MsgTransferERC721Response | TransferERC721 defines a method to transfer ERC721 tokens between chains via IBC |




# uptick/nft/v1beta1/event.proto




## EventBurn
EventBurn is emitted on Burn


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  |  |
| id | string |  |  |
| owner | string |  |  |




## EventMint
EventMint is emitted on Mint


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  |  |
| id | string |  |  |
| owner | string |  |  |




## EventSend
EventSend is emitted on Msg/Send


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  |  |
| id | string |  |  |
| sender | string |  |  |
| receiver | string |  |  |












# uptick/nft/v1beta1/nft.proto




## Class
Class defines the class of the nft type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | string |  | id defines the unique identifier of the NFT classification, similar to the
contract address of ERC721 |
| name | string |  | name defines the human-readable name of the NFT classification. Optional |
| symbol | string |  | symbol is an abbreviated name for nft classification. Optional |
| description | string |  | description is a brief description of nft classification. Optional |
| uri | string |  | uri for the class metadata stored off chain. It can define schema for Class
and NFT `Data` attributes. Optional |
| uri_hash | string |  | uri_hash is a hash of the document pointed by uri. Optional |
| data | google.protobuf.Any |  | data is the app specific metadata of the NFT class. Optional |




## NFT
NFT defines the NFT.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  | class_id associated with the NFT, similar to the contract address of ERC721 |
| id | string |  | id is a unique identifier of the NFT |
| uri | string |  | uri for the NFT metadata stored off chain |
| uri_hash | string |  | uri_hash is a hash of the document pointed by uri |
| data | google.protobuf.Any |  | data is an app specific data of the NFT. Optional |












# uptick/nft/v1beta1/genesis.proto




## Entry
Entry Defines all nft owned by a person


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | string |  | owner is the owner address of the following nft |
| nfts | NFT | repeated | nfts is a group of nfts of the same owner |




## GenesisState
GenesisState defines the nft module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| classes | Class | repeated | class defines the class of the nft type. |
| entries | Entry | repeated |  |












# uptick/nft/v1beta1/query.proto




## QueryBalanceRequest
QueryBalanceRequest is the request type for the Query/Balance RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  |  |
| owner | string |  |  |




## QueryBalanceResponse
QueryBalanceResponse is the response type for the Query/Balance RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| amount | uint64 |  |  |




## QueryClassRequest
QueryClassRequest is the request type for the Query/Class RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  |  |




## QueryClassResponse
QueryClassResponse is the response type for the Query/Class RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class | Class |  |  |




## QueryClassesRequest
QueryClassesRequest is the request type for the Query/Classes RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pagination | cosmos.base.query.v1beta1.PageRequest |  | pagination defines an optional pagination for the request. |




## QueryClassesResponse
QueryClassesResponse is the response type for the Query/Classes RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| classes | Class | repeated |  |
| pagination | cosmos.base.query.v1beta1.PageResponse |  |  |




## QueryNFTRequest
QueryNFTRequest is the request type for the Query/NFT RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  |  |
| id | string |  |  |




## QueryNFTResponse
QueryNFTResponse is the response type for the Query/NFT RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nft | NFT |  |  |




## QueryNFTsRequest
QueryNFTstRequest is the request type for the Query/NFTs RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  |  |
| owner | string |  |  |
| pagination | cosmos.base.query.v1beta1.PageRequest |  |  |




## QueryNFTsResponse
QueryNFTsResponse is the response type for the Query/NFTs RPC methods


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nfts | NFT | repeated |  |
| pagination | cosmos.base.query.v1beta1.PageResponse |  |  |




## QueryOwnerRequest
QueryOwnerRequest is the request type for the Query/Owner RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  |  |
| id | string |  |  |




## QueryOwnerResponse
QueryOwnerResponse is the response type for the Query/Owner RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | string |  |  |




## QuerySupplyRequest
QuerySupplyRequest is the request type for the Query/Supply RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  |  |




## QuerySupplyResponse
QuerySupplyResponse is the response type for the Query/Supply RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| amount | uint64 |  |  |










## Query
Query defines the gRPC querier service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Balance | QueryBalanceRequest | QueryBalanceResponse | Balance queries the number of NFTs of a given class owned by the owner,
same as balanceOf in ERC721 |
| Owner | QueryOwnerRequest | QueryOwnerResponse | Owner queries the owner of the NFT based on its class and id, same as
ownerOf in ERC721 |
| Supply | QuerySupplyRequest | QuerySupplyResponse | Supply queries the number of NFTs from the given class, same as totalSupply
of ERC721. |
| NFTs | QueryNFTsRequest | QueryNFTsResponse | NFTs queries all NFTs of a given class or owner,choose at least one of the
two, similar to tokenByIndex in ERC721Enumerable |
| NFT | QueryNFTRequest | QueryNFTResponse | NFT queries an NFT based on its class and id. |
| Class | QueryClassRequest | QueryClassResponse | Class queries an NFT class based on its id |
| Classes | QueryClassesRequest | QueryClassesResponse | Classes queries all NFT classes |




# uptick/nft/v1beta1/tx.proto




## MsgSend
MsgSend represents a message to send a nft from one account to another
account.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  | class_id defines the unique identifier of the nft classification, similar
to the contract address of ERC721 |
| id | string |  | id defines the unique identification of nft |
| sender | string |  | sender is the address of the owner of nft |
| receiver | string |  | receiver is the receiver address of nft |




## MsgSendResponse
MsgSendResponse defines the Msg/Send response type.










## Msg
Msg defines the nft Msg service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Send | MsgSend | MsgSendResponse | Send defines a method to send a nft from one account to another account. |



 