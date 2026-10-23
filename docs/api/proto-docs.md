<!-- This file is auto-generated. Please do not modify it yourself. -->
# Protobuf Documentation
<a name="top"></a>

## Table of Contents

- [uptick/appmining/v1/types.proto](#uptick/appmining/v1/types.proto)
    - [Application](#uptick.appmining.v1.Application)
    - [AttestationCommitment](#uptick.appmining.v1.AttestationCommitment)
    - [AttestationProfile](#uptick.appmining.v1.AttestationProfile)
    - [Candidate](#uptick.appmining.v1.Candidate)
    - [Challenge](#uptick.appmining.v1.Challenge)
    - [Entitlement](#uptick.appmining.v1.Entitlement)
    - [EntitlementChallenge](#uptick.appmining.v1.EntitlementChallenge)
    - [EpochBudget](#uptick.appmining.v1.EpochBudget)
    - [GasCreditAccount](#uptick.appmining.v1.GasCreditAccount)
    - [GasCreditConsumption](#uptick.appmining.v1.GasCreditConsumption)
    - [Params](#uptick.appmining.v1.Params)
    - [PauseState](#uptick.appmining.v1.PauseState)
    - [Pool](#uptick.appmining.v1.Pool)
    - [RoleKey](#uptick.appmining.v1.RoleKey)
    - [Settlement](#uptick.appmining.v1.Settlement)
    - [WireProof](#uptick.appmining.v1.WireProof)
  
    - [ApplicationStatus](#uptick.appmining.v1.ApplicationStatus)
    - [AttestationStatus](#uptick.appmining.v1.AttestationStatus)
    - [CandidateStatus](#uptick.appmining.v1.CandidateStatus)
    - [ChallengeStatus](#uptick.appmining.v1.ChallengeStatus)
    - [EntitlementChallengeStatus](#uptick.appmining.v1.EntitlementChallengeStatus)
    - [EntitlementRole](#uptick.appmining.v1.EntitlementRole)
    - [EntitlementState](#uptick.appmining.v1.EntitlementState)
    - [EvidenceType](#uptick.appmining.v1.EvidenceType)
    - [RoleKeyStatus](#uptick.appmining.v1.RoleKeyStatus)
  
- [uptick/appmining/v1/genesis.proto](#uptick/appmining/v1/genesis.proto)
    - [GenesisState](#uptick.appmining.v1.GenesisState)
  
- [uptick/appmining/v1/query.proto](#uptick/appmining/v1/query.proto)
    - [QueryApplicationRequest](#uptick.appmining.v1.QueryApplicationRequest)
    - [QueryApplicationResponse](#uptick.appmining.v1.QueryApplicationResponse)
    - [QueryApplicationsRequest](#uptick.appmining.v1.QueryApplicationsRequest)
    - [QueryApplicationsResponse](#uptick.appmining.v1.QueryApplicationsResponse)
    - [QueryAttestationRequest](#uptick.appmining.v1.QueryAttestationRequest)
    - [QueryAttestationResponse](#uptick.appmining.v1.QueryAttestationResponse)
    - [QueryAttestationsRequest](#uptick.appmining.v1.QueryAttestationsRequest)
    - [QueryAttestationsResponse](#uptick.appmining.v1.QueryAttestationsResponse)
    - [QueryCandidateRequest](#uptick.appmining.v1.QueryCandidateRequest)
    - [QueryCandidateResponse](#uptick.appmining.v1.QueryCandidateResponse)
    - [QueryChallengeRequest](#uptick.appmining.v1.QueryChallengeRequest)
    - [QueryChallengeResponse](#uptick.appmining.v1.QueryChallengeResponse)
    - [QueryEntitlementChallengeRequest](#uptick.appmining.v1.QueryEntitlementChallengeRequest)
    - [QueryEntitlementChallengeResponse](#uptick.appmining.v1.QueryEntitlementChallengeResponse)
    - [QueryEntitlementRequest](#uptick.appmining.v1.QueryEntitlementRequest)
    - [QueryEntitlementResponse](#uptick.appmining.v1.QueryEntitlementResponse)
    - [QueryEntitlementsByAppRequest](#uptick.appmining.v1.QueryEntitlementsByAppRequest)
    - [QueryEntitlementsByAppResponse](#uptick.appmining.v1.QueryEntitlementsByAppResponse)
    - [QueryEpochBudgetRequest](#uptick.appmining.v1.QueryEpochBudgetRequest)
    - [QueryEpochBudgetResponse](#uptick.appmining.v1.QueryEpochBudgetResponse)
    - [QueryGasCreditConsumptionsRequest](#uptick.appmining.v1.QueryGasCreditConsumptionsRequest)
    - [QueryGasCreditConsumptionsResponse](#uptick.appmining.v1.QueryGasCreditConsumptionsResponse)
    - [QueryGasCreditRequest](#uptick.appmining.v1.QueryGasCreditRequest)
    - [QueryGasCreditResponse](#uptick.appmining.v1.QueryGasCreditResponse)
    - [QueryOpenChallengesRequest](#uptick.appmining.v1.QueryOpenChallengesRequest)
    - [QueryOpenChallengesResponse](#uptick.appmining.v1.QueryOpenChallengesResponse)
    - [QueryParamsRequest](#uptick.appmining.v1.QueryParamsRequest)
    - [QueryParamsResponse](#uptick.appmining.v1.QueryParamsResponse)
    - [QueryPoolRequest](#uptick.appmining.v1.QueryPoolRequest)
    - [QueryPoolResponse](#uptick.appmining.v1.QueryPoolResponse)
    - [QueryRolesRequest](#uptick.appmining.v1.QueryRolesRequest)
    - [QueryRolesResponse](#uptick.appmining.v1.QueryRolesResponse)
    - [QuerySettlementRequest](#uptick.appmining.v1.QuerySettlementRequest)
    - [QuerySettlementResponse](#uptick.appmining.v1.QuerySettlementResponse)
    - [QuerySettlementsRequest](#uptick.appmining.v1.QuerySettlementsRequest)
    - [QuerySettlementsResponse](#uptick.appmining.v1.QuerySettlementsResponse)
  
    - [Query](#uptick.appmining.v1.Query)
  
- [uptick/appmining/v1/tx.proto](#uptick/appmining/v1/tx.proto)
    - [MsgApproveEntitlement](#uptick.appmining.v1.MsgApproveEntitlement)
    - [MsgApproveEntitlementResponse](#uptick.appmining.v1.MsgApproveEntitlementResponse)
    - [MsgChallengeAttestation](#uptick.appmining.v1.MsgChallengeAttestation)
    - [MsgChallengeAttestationResponse](#uptick.appmining.v1.MsgChallengeAttestationResponse)
    - [MsgClaimEntitlement](#uptick.appmining.v1.MsgClaimEntitlement)
    - [MsgClaimEntitlementResponse](#uptick.appmining.v1.MsgClaimEntitlementResponse)
    - [MsgConsumeGasCredit](#uptick.appmining.v1.MsgConsumeGasCredit)
    - [MsgConsumeGasCreditResponse](#uptick.appmining.v1.MsgConsumeGasCreditResponse)
    - [MsgFinalizeAttestation](#uptick.appmining.v1.MsgFinalizeAttestation)
    - [MsgFinalizeAttestationResponse](#uptick.appmining.v1.MsgFinalizeAttestationResponse)
    - [MsgFreezeEntitlement](#uptick.appmining.v1.MsgFreezeEntitlement)
    - [MsgFreezeEntitlementResponse](#uptick.appmining.v1.MsgFreezeEntitlementResponse)
    - [MsgFundEntitlementPool](#uptick.appmining.v1.MsgFundEntitlementPool)
    - [MsgFundEntitlementPoolResponse](#uptick.appmining.v1.MsgFundEntitlementPoolResponse)
    - [MsgFundRewardPool](#uptick.appmining.v1.MsgFundRewardPool)
    - [MsgFundRewardPoolResponse](#uptick.appmining.v1.MsgFundRewardPoolResponse)
    - [MsgGrantGasPayer](#uptick.appmining.v1.MsgGrantGasPayer)
    - [MsgGrantGasPayerResponse](#uptick.appmining.v1.MsgGrantGasPayerResponse)
    - [MsgOpenChallenge](#uptick.appmining.v1.MsgOpenChallenge)
    - [MsgOpenChallengeResponse](#uptick.appmining.v1.MsgOpenChallengeResponse)
    - [MsgOpenEntitlementPool](#uptick.appmining.v1.MsgOpenEntitlementPool)
    - [MsgOpenEntitlementPoolResponse](#uptick.appmining.v1.MsgOpenEntitlementPoolResponse)
    - [MsgPause](#uptick.appmining.v1.MsgPause)
    - [MsgPauseResponse](#uptick.appmining.v1.MsgPauseResponse)
    - [MsgPublishCandidate](#uptick.appmining.v1.MsgPublishCandidate)
    - [MsgPublishCandidateResponse](#uptick.appmining.v1.MsgPublishCandidateResponse)
    - [MsgRegisterApplication](#uptick.appmining.v1.MsgRegisterApplication)
    - [MsgRegisterApplicationResponse](#uptick.appmining.v1.MsgRegisterApplicationResponse)
    - [MsgRegisterRoleKey](#uptick.appmining.v1.MsgRegisterRoleKey)
    - [MsgRegisterRoleKeyResponse](#uptick.appmining.v1.MsgRegisterRoleKeyResponse)
    - [MsgResolveChallenge](#uptick.appmining.v1.MsgResolveChallenge)
    - [MsgResolveChallengeResponse](#uptick.appmining.v1.MsgResolveChallengeResponse)
    - [MsgResolveEntitlementChallenge](#uptick.appmining.v1.MsgResolveEntitlementChallenge)
    - [MsgResolveEntitlementChallengeResponse](#uptick.appmining.v1.MsgResolveEntitlementChallengeResponse)
    - [MsgRevokeEntitlement](#uptick.appmining.v1.MsgRevokeEntitlement)
    - [MsgRevokeEntitlementResponse](#uptick.appmining.v1.MsgRevokeEntitlementResponse)
    - [MsgRevokeGasPayer](#uptick.appmining.v1.MsgRevokeGasPayer)
    - [MsgRevokeGasPayerResponse](#uptick.appmining.v1.MsgRevokeGasPayerResponse)
    - [MsgSetApplicationStatus](#uptick.appmining.v1.MsgSetApplicationStatus)
    - [MsgSetApplicationStatusResponse](#uptick.appmining.v1.MsgSetApplicationStatusResponse)
    - [MsgSetAttestationSubmitter](#uptick.appmining.v1.MsgSetAttestationSubmitter)
    - [MsgSetAttestationSubmitterResponse](#uptick.appmining.v1.MsgSetAttestationSubmitterResponse)
    - [MsgSetEntitlementPoolPaused](#uptick.appmining.v1.MsgSetEntitlementPoolPaused)
    - [MsgSetEntitlementPoolPausedResponse](#uptick.appmining.v1.MsgSetEntitlementPoolPausedResponse)
    - [MsgSetEpochBudget](#uptick.appmining.v1.MsgSetEpochBudget)
    - [MsgSetEpochBudgetResponse](#uptick.appmining.v1.MsgSetEpochBudgetResponse)
    - [MsgSetRoleKeyStatus](#uptick.appmining.v1.MsgSetRoleKeyStatus)
    - [MsgSetRoleKeyStatusResponse](#uptick.appmining.v1.MsgSetRoleKeyStatusResponse)
    - [MsgSubmitAttestation](#uptick.appmining.v1.MsgSubmitAttestation)
    - [MsgSubmitAttestationResponse](#uptick.appmining.v1.MsgSubmitAttestationResponse)
    - [MsgUpdateParams](#uptick.appmining.v1.MsgUpdateParams)
    - [MsgUpdateParamsResponse](#uptick.appmining.v1.MsgUpdateParamsResponse)
  
    - [Msg](#uptick.appmining.v1.Msg)
  
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
    - [NFTUIDPair](#uptick.cw721.v1.NFTUIDPair)
    - [Params](#uptick.cw721.v1.Params)
    - [RefundReceiver](#uptick.cw721.v1.RefundReceiver)
  
- [uptick/cw721/v1/query.proto](#uptick/cw721/v1/query.proto)
    - [QueryParamsRequest](#uptick.cw721.v1.QueryParamsRequest)
    - [QueryParamsResponse](#uptick.cw721.v1.QueryParamsResponse)
    - [QueryTokenPairRequest](#uptick.cw721.v1.QueryTokenPairRequest)
    - [QueryTokenPairResponse](#uptick.cw721.v1.QueryTokenPairResponse)
    - [QueryTokenPairsRequest](#uptick.cw721.v1.QueryTokenPairsRequest)
    - [QueryTokenPairsResponse](#uptick.cw721.v1.QueryTokenPairsResponse)
    - [QueryWasmAddressRequest](#uptick.cw721.v1.QueryWasmAddressRequest)
    - [QueryWasmContractResponse](#uptick.cw721.v1.QueryWasmContractResponse)
  
    - [Query](#uptick.cw721.v1.Query)
  
- [uptick/cw721/v1/tx.proto](#uptick/cw721/v1/tx.proto)
    - [MsgConvertC721Response](#uptick.cw721.v1.MsgConvertC721Response)
    - [MsgConvertCW721](#uptick.cw721.v1.MsgConvertCW721)
    - [MsgConvertCW721Response](#uptick.cw721.v1.MsgConvertCW721Response)
    - [MsgConvertNFT](#uptick.cw721.v1.MsgConvertNFT)
    - [MsgConvertNFTResponse](#uptick.cw721.v1.MsgConvertNFTResponse)
    - [MsgTransferCW721](#uptick.cw721.v1.MsgTransferCW721)
    - [MsgTransferCW721Response](#uptick.cw721.v1.MsgTransferCW721Response)
    - [MsgUpdateParams](#uptick.cw721.v1.MsgUpdateParams)
    - [MsgUpdateParamsResponse](#uptick.cw721.v1.MsgUpdateParamsResponse)
  
    - [Msg](#uptick.cw721.v1.Msg)
  
- [uptick/erc721/v1/erc721.proto](#uptick/erc721/v1/erc721.proto)
    - [TokenPair](#uptick.erc721.v1.TokenPair)
    - [UIDPair](#uptick.erc721.v1.UIDPair)
  
    - [Owner](#uptick.erc721.v1.Owner)
  
- [uptick/erc721/v1/genesis.proto](#uptick/erc721/v1/genesis.proto)
    - [GenesisState](#uptick.erc721.v1.GenesisState)
    - [NFTUIDPair](#uptick.erc721.v1.NFTUIDPair)
    - [Params](#uptick.erc721.v1.Params)
    - [RefundReceiver](#uptick.erc721.v1.RefundReceiver)
  
- [uptick/erc721/v1/query.proto](#uptick/erc721/v1/query.proto)
    - [QueryEvmAddressRequest](#uptick.erc721.v1.QueryEvmAddressRequest)
    - [QueryEvmAddressResponse](#uptick.erc721.v1.QueryEvmAddressResponse)
    - [QueryParamsRequest](#uptick.erc721.v1.QueryParamsRequest)
    - [QueryParamsResponse](#uptick.erc721.v1.QueryParamsResponse)
    - [QueryTokenPairRequest](#uptick.erc721.v1.QueryTokenPairRequest)
    - [QueryTokenPairResponse](#uptick.erc721.v1.QueryTokenPairResponse)
    - [QueryTokenPairsRequest](#uptick.erc721.v1.QueryTokenPairsRequest)
    - [QueryTokenPairsResponse](#uptick.erc721.v1.QueryTokenPairsResponse)
  
    - [Query](#uptick.erc721.v1.Query)
  
- [uptick/erc721/v1/tx.proto](#uptick/erc721/v1/tx.proto)
    - [MsgConvertERC721](#uptick.erc721.v1.MsgConvertERC721)
    - [MsgConvertERC721Response](#uptick.erc721.v1.MsgConvertERC721Response)
    - [MsgConvertNFT](#uptick.erc721.v1.MsgConvertNFT)
    - [MsgConvertNFTResponse](#uptick.erc721.v1.MsgConvertNFTResponse)
    - [MsgTransferERC721](#uptick.erc721.v1.MsgTransferERC721)
    - [MsgTransferERC721Response](#uptick.erc721.v1.MsgTransferERC721Response)
    - [MsgUpdateParams](#uptick.erc721.v1.MsgUpdateParams)
    - [MsgUpdateParamsResponse](#uptick.erc721.v1.MsgUpdateParamsResponse)
  
    - [Msg](#uptick.erc721.v1.Msg)
  
- [uptick/evm_ibc/v1/evm_ibc.proto](#uptick/evm_ibc/v1/evm_ibc.proto)
    - [TokenPair](#uptick.evm_ibc.v1.TokenPair)
  
- [uptick/evm_ibc/v1/query.proto](#uptick/evm_ibc/v1/query.proto)
    - [QueryEvmAddressRequest](#uptick.evm_ibc.v1.QueryEvmAddressRequest)
    - [QueryTokenPairResponse](#uptick.evm_ibc.v1.QueryTokenPairResponse)
  
    - [Query](#uptick.evm_ibc.v1.Query)
  
- [uptick/evm_ibc/v1/tx.proto](#uptick/evm_ibc/v1/tx.proto)
    - [MsgTransferERC721](#uptick.evm_ibc.v1.MsgTransferERC721)
    - [MsgTransferERC721Response](#uptick.evm_ibc.v1.MsgTransferERC721Response)
  
    - [Msg](#uptick.evm_ibc.v1.Msg)
  
- [uptick/nft/v1beta1/event.proto](#uptick/nft/v1beta1/event.proto)
    - [EventBurn](#uptick.nft.v1beta1.EventBurn)
    - [EventMint](#uptick.nft.v1beta1.EventMint)
    - [EventSend](#uptick.nft.v1beta1.EventSend)
  
- [uptick/nft/v1beta1/nft.proto](#uptick/nft/v1beta1/nft.proto)
    - [Class](#uptick.nft.v1beta1.Class)
    - [NFT](#uptick.nft.v1beta1.NFT)
  
- [uptick/nft/v1beta1/genesis.proto](#uptick/nft/v1beta1/genesis.proto)
    - [Entry](#uptick.nft.v1beta1.Entry)
    - [GenesisState](#uptick.nft.v1beta1.GenesisState)
  
- [uptick/nft/v1beta1/query.proto](#uptick/nft/v1beta1/query.proto)
    - [QueryBalanceRequest](#uptick.nft.v1beta1.QueryBalanceRequest)
    - [QueryBalanceResponse](#uptick.nft.v1beta1.QueryBalanceResponse)
    - [QueryClassRequest](#uptick.nft.v1beta1.QueryClassRequest)
    - [QueryClassResponse](#uptick.nft.v1beta1.QueryClassResponse)
    - [QueryClassesRequest](#uptick.nft.v1beta1.QueryClassesRequest)
    - [QueryClassesResponse](#uptick.nft.v1beta1.QueryClassesResponse)
    - [QueryNFTRequest](#uptick.nft.v1beta1.QueryNFTRequest)
    - [QueryNFTResponse](#uptick.nft.v1beta1.QueryNFTResponse)
    - [QueryNFTsRequest](#uptick.nft.v1beta1.QueryNFTsRequest)
    - [QueryNFTsResponse](#uptick.nft.v1beta1.QueryNFTsResponse)
    - [QueryOwnerRequest](#uptick.nft.v1beta1.QueryOwnerRequest)
    - [QueryOwnerResponse](#uptick.nft.v1beta1.QueryOwnerResponse)
    - [QuerySupplyRequest](#uptick.nft.v1beta1.QuerySupplyRequest)
    - [QuerySupplyResponse](#uptick.nft.v1beta1.QuerySupplyResponse)
  
    - [Query](#uptick.nft.v1beta1.Query)
  
- [uptick/nft/v1beta1/tx.proto](#uptick/nft/v1beta1/tx.proto)
    - [MsgSend](#uptick.nft.v1beta1.MsgSend)
    - [MsgSendResponse](#uptick.nft.v1beta1.MsgSendResponse)
  
    - [Msg](#uptick.nft.v1beta1.Msg)
  
- [Scalar Value Types](#scalar-value-types)



<a name="uptick/appmining/v1/types.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/appmining/v1/types.proto



<a name="uptick.appmining.v1.Application"></a>

### Application
Application is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `app_id` | [bytes](#bytes) |  |  |
| `owner` | [string](#string) |  |  |
| `attestation_submitter` | [string](#string) |  |  |
| `gas_payers` | [string](#string) | repeated |  |
| `status` | [ApplicationStatus](#uptick.appmining.v1.ApplicationStatus) |  |  |
| `metadata_commitment` | [bytes](#bytes) |  | opaque 32-byte metadata commitment (empty allowed until set) |






<a name="uptick.appmining.v1.AttestationCommitment"></a>

### AttestationCommitment
AttestationCommitment is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `app_id` | [bytes](#bytes) |  |  |
| `epoch` | [uint64](#uint64) |  |  |
| `profile` | [string](#string) |  |  |
| `profile_version` | [string](#string) |  |  |
| `attestation_hash` | [bytes](#bytes) |  |  |
| `evidence_root` | [bytes](#bytes) |  |  |
| `result_commitment` | [bytes](#bytes) |  |  |
| `reward_requested` | [string](#string) |  |  |
| `gas_credit_requested` | [string](#string) |  |  |
| `submitter` | [string](#string) |  |  |
| `submitted_height` | [int64](#int64) |  |  |
| `challenge_deadline_height` | [int64](#int64) |  |  |
| `status` | [AttestationStatus](#uptick.appmining.v1.AttestationStatus) |  |  |






<a name="uptick.appmining.v1.AttestationProfile"></a>

### AttestationProfile
AttestationProfile is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `profile` | [string](#string) |  |  |
| `version` | [string](#string) |  |  |
| `active` | [bool](#bool) |  |  |






<a name="uptick.appmining.v1.Candidate"></a>

### Candidate
Candidate is a published entitlement candidate (no pool liability until Approve).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `entitlement_id` | [string](#string) |  | 64 hex |
| `app_id` | [string](#string) |  | 64 hex (wire); registry lookup uses raw 32 bytes |
| `epoch` | [string](#string) |  |  |
| `amount` | [string](#string) |  |  |
| `denom` | [string](#string) |  |  |
| `gas_used` | [string](#string) |  |  |
| `qualified_tx_count` | [string](#string) |  |  |
| `pav_id` | [string](#string) |  |  |
| `evidence_root` | [string](#string) |  |  |
| `gas_evidence_root` | [string](#string) |  |  |
| `rule_version` | [string](#string) |  |  |
| `candidate_hash` | [string](#string) |  |  |
| `pool_id` | [string](#string) |  |  |
| `publish_tx_hash` | [string](#string) |  |  |
| `publish_height` | [int64](#int64) |  |  |
| `published_at` | [int64](#int64) |  | unix seconds (block time) |
| `challenge_deadline` | [int64](#int64) |  | published_at + 604800 |
| `publisher` | [string](#string) |  | cosmos bech32 submitter |
| `status` | [CandidateStatus](#uptick.appmining.v1.CandidateStatus) |  |  |
| `max_observed_time` | [int64](#int64) |  | anti-rollback watermark |






<a name="uptick.appmining.v1.Challenge"></a>

### Challenge
Challenge is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `challenge_id` | [uint64](#uint64) |  |  |
| `app_id` | [bytes](#bytes) |  |  |
| `epoch` | [uint64](#uint64) |  |  |
| `challenger` | [string](#string) |  |  |
| `evidence_hash` | [bytes](#bytes) |  |  |
| `bond_denom` | [string](#string) |  |  |
| `bond_amount` | [string](#string) |  |  |
| `status` | [ChallengeStatus](#uptick.appmining.v1.ChallengeStatus) |  |  |
| `resolution_hash` | [bytes](#bytes) |  |  |
| `resolved_height` | [int64](#int64) |  |  |






<a name="uptick.appmining.v1.Entitlement"></a>

### Entitlement
Entitlement is created only after Approve (pool liability reserved).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `entitlement_id` | [string](#string) |  |  |
| `app_id` | [string](#string) |  |  |
| `owner_at_last_query` | [string](#string) |  |  |
| `amount` | [string](#string) |  |  |
| `denom` | [string](#string) |  |  |
| `state` | [EntitlementState](#uptick.appmining.v1.EntitlementState) |  |  |
| `freeze_until` | [int64](#int64) |  |  |
| `freeze_count` | [uint32](#uint32) |  |  |
| `approval_tx` | [string](#string) |  |  |
| `claim_tx` | [string](#string) |  |  |
| `evidence_root` | [string](#string) |  |  |
| `rule_version` | [string](#string) |  |  |
| `pool_id` | [string](#string) |  |  |
| `approved_at` | [int64](#int64) |  |  |
| `max_observed_time` | [int64](#int64) |  |  |






<a name="uptick.appmining.v1.EntitlementChallenge"></a>

### EntitlementChallenge
EntitlementChallenge is the V1 challenge record (distinct from attestation Challenge).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `challenge_id` | [string](#string) |  | 64 hex |
| `candidate_id` | [string](#string) |  | entitlement_id |
| `challenger` | [string](#string) |  |  |
| `evidence_type` | [EvidenceType](#uptick.appmining.v1.EvidenceType) |  |  |
| `evidence_digest` | [string](#string) |  | 64 hex |
| `reference_digest` | [string](#string) |  | 64 hex; may be empty -> store as 64 zero hex if unset at open |
| `status` | [EntitlementChallengeStatus](#uptick.appmining.v1.EntitlementChallengeStatus) |  |  |
| `opened_at` | [int64](#int64) |  |  |
| `reviewer` | [string](#string) |  | key_id of resolving reviewer |
| `resolution_digest` | [string](#string) |  | 64 hex |
| `resolved_at` | [int64](#int64) |  |  |






<a name="uptick.appmining.v1.EpochBudget"></a>

### EpochBudget
EpochBudget is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `epoch` | [uint64](#uint64) |  |  |
| `denom` | [string](#string) |  |  |
| `limit` | [string](#string) |  |  |
| `spent` | [string](#string) |  |  |
| `closed` | [bool](#bool) |  |  |
| `opened_height` | [int64](#int64) |  |  |
| `closed_height` | [int64](#int64) |  |  |






<a name="uptick.appmining.v1.GasCreditAccount"></a>

### GasCreditAccount
GasCreditAccount is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `app_id` | [bytes](#bytes) |  |  |
| `balance` | [string](#string) |  |  |
| `total_issued` | [string](#string) |  |  |
| `total_consumed` | [string](#string) |  |  |






<a name="uptick.appmining.v1.GasCreditConsumption"></a>

### GasCreditConsumption
GasCreditConsumption is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `app_id` | [bytes](#bytes) |  |  |
| `gas_payer` | [string](#string) |  |  |
| `source_tx_hash` | [bytes](#bytes) |  |  |
| `amount` | [string](#string) |  |  |
| `consumed_height` | [int64](#int64) |  |  |






<a name="uptick.appmining.v1.Params"></a>

### Params
Params is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  |  |
| `challenge_window_blocks` | [uint64](#uint64) |  |  |
| `challenge_bond_denom` | [string](#string) |  |  |
| `challenge_bond_amount` | [string](#string) |  |  |
| `gas_credit_share_bps` | [uint32](#uint32) |  |  |
| `pause_state` | [PauseState](#uptick.appmining.v1.PauseState) |  |  |
| `allowed_profiles` | [AttestationProfile](#uptick.appmining.v1.AttestationProfile) | repeated |  |
| `pause_authority` | [string](#string) |  | hot path: may only set pause bits to true; resume is authority-only (Q-06) |
| `reward_denom_allowlist` | [string](#string) | repeated | governance-approved reward denoms (Q-08) |
| `max_gas_payers` | [uint32](#uint32) |  | max gas payers per application (0 = use module default 16) |
| `allow_manual_gas_credit_consume` | [bool](#bool) |  | when false (default), MsgConsumeGasCredit is rejected; PostHandler auto-debit remains |
| `entitlement_enable_min_funded` | [string](#string) |  | V1: minimum Pool.funded (smallest units, unsigned decimal string) required before entitlement msgs are enabled. Empty or "0" keeps the new flow disabled. |
| `entitlement_allowed_rule_versions` | [string](#string) | repeated | V1: allowlist of rule_version values accepted by MsgApproveEntitlement (§3 "规则版本允许"). Empty list denies every approval (default-deny). |






<a name="uptick.appmining.v1.PauseState"></a>

### PauseState
PauseState is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `registration_paused` | [bool](#bool) |  |  |
| `attestation_paused` | [bool](#bool) |  |  |
| `challenge_paused` | [bool](#bool) |  |  |
| `settlement_paused` | [bool](#bool) |  |  |
| `gas_credit_paused` | [bool](#bool) |  |  |
| `entitlement_paused` | [bool](#bool) |  | V1 entitlement flow (Publish/Approve/Claim/…); independent of attestation pause bits |






<a name="uptick.appmining.v1.Pool"></a>

### Pool
Pool is the fixed entitlement fund ledger.
Invariant: funded = available + reserved + paid; reserved <= module account balance.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `pool_id` | [string](#string) |  |  |
| `denom` | [string](#string) |  |  |
| `funded` | [string](#string) |  |  |
| `available` | [string](#string) |  |  |
| `reserved` | [string](#string) |  |  |
| `paid` | [string](#string) |  |  |
| `paused` | [bool](#bool) |  |  |
| `version` | [uint64](#uint64) |  |  |






<a name="uptick.appmining.v1.RoleKey"></a>

### RoleKey
RoleKey is a versioned Ed25519 public key for one UGE role.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `role` | [EntitlementRole](#uptick.appmining.v1.EntitlementRole) |  |  |
| `key_id` | [string](#string) |  |  |
| `pubkey` | [bytes](#bytes) |  | raw Ed25519 public key, exactly 32 bytes |
| `status` | [RoleKeyStatus](#uptick.appmining.v1.RoleKeyStatus) |  |  |
| `version` | [uint64](#uint64) |  |  |
| `registered_height` | [int64](#int64) |  |  |






<a name="uptick.appmining.v1.Settlement"></a>

### Settlement
Settlement is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `app_id` | [bytes](#bytes) |  |  |
| `epoch` | [uint64](#uint64) |  |  |
| `profile` | [string](#string) |  |  |
| `profile_version` | [string](#string) |  |  |
| `attestation_hash` | [bytes](#bytes) |  |  |
| `reward_released` | [string](#string) |  |  |
| `gas_credit_issued` | [string](#string) |  |  |
| `finalized_height` | [int64](#int64) |  |  |






<a name="uptick.appmining.v1.WireProof"></a>

### WireProof
WireProof is the Ed25519 domain-separated authorization attached to V1 msgs.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `key_id` | [string](#string) |  |  |
| `nonce` | [string](#string) |  |  |
| `expires_at` | [string](#string) |  | unix seconds, unsigned decimal |
| `signature_base64` | [string](#string) |  |  |





 <!-- end messages -->


<a name="uptick.appmining.v1.ApplicationStatus"></a>

### ApplicationStatus
ApplicationStatus enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| APPLICATION_STATUS_UNSPECIFIED | 0 | APPLICATION_STATUS_UNSPECIFIED is a valid enum value. |
| APPLICATION_STATUS_ACTIVE | 1 | APPLICATION_STATUS_ACTIVE is a valid enum value. |
| APPLICATION_STATUS_SUSPENDED | 2 | APPLICATION_STATUS_SUSPENDED is a valid enum value. |
| APPLICATION_STATUS_REVOKED | 3 | APPLICATION_STATUS_REVOKED is a valid enum value. |



<a name="uptick.appmining.v1.AttestationStatus"></a>

### AttestationStatus
AttestationStatus enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ATTESTATION_STATUS_UNSPECIFIED | 0 | ATTESTATION_STATUS_UNSPECIFIED is a valid enum value. |
| ATTESTATION_STATUS_CHALLENGE_WINDOW | 1 | ATTESTATION_STATUS_CHALLENGE_WINDOW is a valid enum value. |
| ATTESTATION_STATUS_CHALLENGED | 2 | ATTESTATION_STATUS_CHALLENGED is a valid enum value. |
| ATTESTATION_STATUS_FINALIZED | 3 | ATTESTATION_STATUS_FINALIZED is a valid enum value. |
| ATTESTATION_STATUS_REJECTED | 4 | ATTESTATION_STATUS_REJECTED is a valid enum value. |



<a name="uptick.appmining.v1.CandidateStatus"></a>

### CandidateStatus
CandidateStatus enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| CANDIDATE_STATUS_UNSPECIFIED | 0 | CANDIDATE_STATUS_UNSPECIFIED is a valid enum value. |
| CANDIDATE_STATUS_PUBLISHED | 1 | CANDIDATE_STATUS_PUBLISHED is a valid enum value. |
| CANDIDATE_STATUS_REJECTED | 2 | CANDIDATE_STATUS_REJECTED is a valid enum value. |



<a name="uptick.appmining.v1.ChallengeStatus"></a>

### ChallengeStatus
ChallengeStatus enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| CHALLENGE_STATUS_UNSPECIFIED | 0 | CHALLENGE_STATUS_UNSPECIFIED is a valid enum value. |
| CHALLENGE_STATUS_OPEN | 1 | CHALLENGE_STATUS_OPEN is a valid enum value. |
| CHALLENGE_STATUS_REJECTED | 2 | CHALLENGE_STATUS_REJECTED is a valid enum value. |
| CHALLENGE_STATUS_UPHELD | 3 | CHALLENGE_STATUS_UPHELD is a valid enum value. |
| CHALLENGE_STATUS_SUPERSEDED | 4 | closed without ruling because another upheld challenge terminated the attestation; bond fully refunded (CLARIFICATIONS-REPLY Q-04) |



<a name="uptick.appmining.v1.EntitlementChallengeStatus"></a>

### EntitlementChallengeStatus
EntitlementChallengeStatus enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENTITLEMENT_CHALLENGE_STATUS_UNSPECIFIED | 0 | ENTITLEMENT_CHALLENGE_STATUS_UNSPECIFIED is a valid enum value. |
| ENTITLEMENT_CHALLENGE_STATUS_OPEN | 1 | ENTITLEMENT_CHALLENGE_STATUS_OPEN is a valid enum value. |
| ENTITLEMENT_CHALLENGE_STATUS_UPHELD | 2 | ENTITLEMENT_CHALLENGE_STATUS_UPHELD is a valid enum value. |
| ENTITLEMENT_CHALLENGE_STATUS_DISMISSED | 3 | ENTITLEMENT_CHALLENGE_STATUS_DISMISSED is a valid enum value. |



<a name="uptick.appmining.v1.EntitlementRole"></a>

### EntitlementRole
EntitlementRole is the five UGE authorization roles (wire role strings).

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENTITLEMENT_ROLE_UNSPECIFIED | 0 | ENTITLEMENT_ROLE_UNSPECIFIED is a valid enum value. |
| ENTITLEMENT_ROLE_PUBLISHER | 1 | ENTITLEMENT_ROLE_PUBLISHER is a valid enum value. |
| ENTITLEMENT_ROLE_APPROVER | 2 | ENTITLEMENT_ROLE_APPROVER is a valid enum value. |
| ENTITLEMENT_ROLE_REVIEWER | 3 | ENTITLEMENT_ROLE_REVIEWER is a valid enum value. |
| ENTITLEMENT_ROLE_RISK | 4 | ENTITLEMENT_ROLE_RISK is a valid enum value. |
| ENTITLEMENT_ROLE_AUDIT | 5 | ENTITLEMENT_ROLE_AUDIT is a valid enum value. |



<a name="uptick.appmining.v1.EntitlementState"></a>

### EntitlementState
EntitlementState enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENTITLEMENT_STATE_UNSPECIFIED | 0 | ENTITLEMENT_STATE_UNSPECIFIED is a valid enum value. |
| ENTITLEMENT_STATE_CLAIMABLE | 1 | ENTITLEMENT_STATE_CLAIMABLE is a valid enum value. |
| ENTITLEMENT_STATE_FROZEN | 2 | ENTITLEMENT_STATE_FROZEN is a valid enum value. |
| ENTITLEMENT_STATE_REVOKED | 3 | ENTITLEMENT_STATE_REVOKED is a valid enum value. |
| ENTITLEMENT_STATE_CLAIMED | 4 | ENTITLEMENT_STATE_CLAIMED is a valid enum value. |



<a name="uptick.appmining.v1.EvidenceType"></a>

### EvidenceType
EvidenceType enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| EVIDENCE_TYPE_UNSPECIFIED | 0 | EVIDENCE_TYPE_UNSPECIFIED is a valid enum value. |
| EVIDENCE_TYPE_TX_MISSING_OR_UNFINALIZED | 1 | EVIDENCE_TYPE_TX_MISSING_OR_UNFINALIZED is a valid enum value. |
| EVIDENCE_TYPE_RECEIPT_QBT_MISMATCH | 2 | EVIDENCE_TYPE_RECEIPT_QBT_MISMATCH is a valid enum value. |
| EVIDENCE_TYPE_DUPLICATE_OR_FAMILY_CONFLICT | 3 | EVIDENCE_TYPE_DUPLICATE_OR_FAMILY_CONFLICT is a valid enum value. |
| EVIDENCE_TYPE_CONFIRMED_REFUND_OR_REVERSAL | 4 | EVIDENCE_TYPE_CONFIRMED_REFUND_OR_REVERSAL is a valid enum value. |
| EVIDENCE_TYPE_APP_OR_EMITTER_INELIGIBLE | 5 | EVIDENCE_TYPE_APP_OR_EMITTER_INELIGIBLE is a valid enum value. |
| EVIDENCE_TYPE_REPRODUCIBLE_CALCULATION_ERROR | 6 | EVIDENCE_TYPE_REPRODUCIBLE_CALCULATION_ERROR is a valid enum value. |



<a name="uptick.appmining.v1.RoleKeyStatus"></a>

### RoleKeyStatus
RoleKeyStatus enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ROLE_KEY_STATUS_UNSPECIFIED | 0 | ROLE_KEY_STATUS_UNSPECIFIED is a valid enum value. |
| ROLE_KEY_STATUS_ACTIVE | 1 | ROLE_KEY_STATUS_ACTIVE is a valid enum value. |
| ROLE_KEY_STATUS_PAUSED | 2 | ROLE_KEY_STATUS_PAUSED is a valid enum value. |
| ROLE_KEY_STATUS_REVOKED | 3 | ROLE_KEY_STATUS_REVOKED is a valid enum value. |


 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/appmining/v1/genesis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/appmining/v1/genesis.proto



<a name="uptick.appmining.v1.GenesisState"></a>

### GenesisState
GenesisState defines the appmining module genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#uptick.appmining.v1.Params) |  |  |
| `applications` | [Application](#uptick.appmining.v1.Application) | repeated |  |
| `epoch_budgets` | [EpochBudget](#uptick.appmining.v1.EpochBudget) | repeated |  |
| `attestations` | [AttestationCommitment](#uptick.appmining.v1.AttestationCommitment) | repeated |  |
| `challenges` | [Challenge](#uptick.appmining.v1.Challenge) | repeated |  |
| `settlements` | [Settlement](#uptick.appmining.v1.Settlement) | repeated |  |
| `gas_credit_accounts` | [GasCreditAccount](#uptick.appmining.v1.GasCreditAccount) | repeated |  |
| `gas_credit_consumptions` | [GasCreditConsumption](#uptick.appmining.v1.GasCreditConsumption) | repeated |  |
| `next_challenge_id` | [uint64](#uint64) |  |  |
| `entitlement_pools` | [Pool](#uptick.appmining.v1.Pool) | repeated | V1 entitlement state (UGE requirements 1.0) |
| `role_keys` | [RoleKey](#uptick.appmining.v1.RoleKey) | repeated |  |
| `candidates` | [Candidate](#uptick.appmining.v1.Candidate) | repeated |  |
| `entitlement_challenges` | [EntitlementChallenge](#uptick.appmining.v1.EntitlementChallenge) | repeated |  |
| `used_auth_nonces` | [string](#string) | repeated | "role/key_id/nonce" consumed proofs |
| `entitlements` | [Entitlement](#uptick.appmining.v1.Entitlement) | repeated |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/appmining/v1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/appmining/v1/query.proto



<a name="uptick.appmining.v1.QueryApplicationRequest"></a>

### QueryApplicationRequest
QueryApplicationRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `app_id` | [bytes](#bytes) |  |  |






<a name="uptick.appmining.v1.QueryApplicationResponse"></a>

### QueryApplicationResponse
QueryApplicationResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `application` | [Application](#uptick.appmining.v1.Application) |  |  |






<a name="uptick.appmining.v1.QueryApplicationsRequest"></a>

### QueryApplicationsRequest
QueryApplicationsRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `status` | [ApplicationStatus](#uptick.appmining.v1.ApplicationStatus) |  | optional status filter; UNSPECIFIED = all |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  |  |






<a name="uptick.appmining.v1.QueryApplicationsResponse"></a>

### QueryApplicationsResponse
QueryApplicationsResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `applications` | [Application](#uptick.appmining.v1.Application) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  |  |






<a name="uptick.appmining.v1.QueryAttestationRequest"></a>

### QueryAttestationRequest
QueryAttestationRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `app_id` | [bytes](#bytes) |  |  |
| `epoch` | [uint64](#uint64) |  |  |






<a name="uptick.appmining.v1.QueryAttestationResponse"></a>

### QueryAttestationResponse
QueryAttestationResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `attestation` | [AttestationCommitment](#uptick.appmining.v1.AttestationCommitment) |  |  |






<a name="uptick.appmining.v1.QueryAttestationsRequest"></a>

### QueryAttestationsRequest
QueryAttestationsRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `status` | [AttestationStatus](#uptick.appmining.v1.AttestationStatus) |  | optional status filter; UNSPECIFIED = all |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  |  |






<a name="uptick.appmining.v1.QueryAttestationsResponse"></a>

### QueryAttestationsResponse
QueryAttestationsResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `attestations` | [AttestationCommitment](#uptick.appmining.v1.AttestationCommitment) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  |  |






<a name="uptick.appmining.v1.QueryCandidateRequest"></a>

### QueryCandidateRequest
QueryCandidateRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `entitlement_id` | [string](#string) |  |  |






<a name="uptick.appmining.v1.QueryCandidateResponse"></a>

### QueryCandidateResponse
QueryCandidateResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `candidate` | [Candidate](#uptick.appmining.v1.Candidate) |  |  |






<a name="uptick.appmining.v1.QueryChallengeRequest"></a>

### QueryChallengeRequest
QueryChallengeRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `challenge_id` | [uint64](#uint64) |  |  |






<a name="uptick.appmining.v1.QueryChallengeResponse"></a>

### QueryChallengeResponse
QueryChallengeResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `challenge` | [Challenge](#uptick.appmining.v1.Challenge) |  |  |






<a name="uptick.appmining.v1.QueryEntitlementChallengeRequest"></a>

### QueryEntitlementChallengeRequest
QueryEntitlementChallengeRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `challenge_id` | [string](#string) |  |  |






<a name="uptick.appmining.v1.QueryEntitlementChallengeResponse"></a>

### QueryEntitlementChallengeResponse
QueryEntitlementChallengeResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `challenge` | [EntitlementChallenge](#uptick.appmining.v1.EntitlementChallenge) |  |  |






<a name="uptick.appmining.v1.QueryEntitlementRequest"></a>

### QueryEntitlementRequest
QueryEntitlementRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `entitlement_id` | [string](#string) |  |  |






<a name="uptick.appmining.v1.QueryEntitlementResponse"></a>

### QueryEntitlementResponse
QueryEntitlementResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `entitlement` | [Entitlement](#uptick.appmining.v1.Entitlement) |  |  |






<a name="uptick.appmining.v1.QueryEntitlementsByAppRequest"></a>

### QueryEntitlementsByAppRequest
QueryEntitlementsByAppRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `app_id` | [string](#string) |  | 64 hex |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  |  |






<a name="uptick.appmining.v1.QueryEntitlementsByAppResponse"></a>

### QueryEntitlementsByAppResponse
QueryEntitlementsByAppResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `entitlements` | [Entitlement](#uptick.appmining.v1.Entitlement) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  |  |






<a name="uptick.appmining.v1.QueryEpochBudgetRequest"></a>

### QueryEpochBudgetRequest
QueryEpochBudgetRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `epoch` | [uint64](#uint64) |  |  |






<a name="uptick.appmining.v1.QueryEpochBudgetResponse"></a>

### QueryEpochBudgetResponse
QueryEpochBudgetResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `budget` | [EpochBudget](#uptick.appmining.v1.EpochBudget) |  |  |






<a name="uptick.appmining.v1.QueryGasCreditConsumptionsRequest"></a>

### QueryGasCreditConsumptionsRequest
QueryGasCreditConsumptionsRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `app_id` | [bytes](#bytes) |  |  |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  |  |






<a name="uptick.appmining.v1.QueryGasCreditConsumptionsResponse"></a>

### QueryGasCreditConsumptionsResponse
QueryGasCreditConsumptionsResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `consumptions` | [GasCreditConsumption](#uptick.appmining.v1.GasCreditConsumption) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  |  |






<a name="uptick.appmining.v1.QueryGasCreditRequest"></a>

### QueryGasCreditRequest
QueryGasCreditRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `app_id` | [bytes](#bytes) |  |  |






<a name="uptick.appmining.v1.QueryGasCreditResponse"></a>

### QueryGasCreditResponse
QueryGasCreditResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `account` | [GasCreditAccount](#uptick.appmining.v1.GasCreditAccount) |  |  |






<a name="uptick.appmining.v1.QueryOpenChallengesRequest"></a>

### QueryOpenChallengesRequest
QueryOpenChallengesRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `app_id` | [bytes](#bytes) |  | optional; empty = all apps |
| `epoch` | [uint64](#uint64) |  | optional; 0 = all epochs when app_id set still scans open index |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  |  |






<a name="uptick.appmining.v1.QueryOpenChallengesResponse"></a>

### QueryOpenChallengesResponse
QueryOpenChallengesResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `challenges` | [Challenge](#uptick.appmining.v1.Challenge) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  |  |






<a name="uptick.appmining.v1.QueryParamsRequest"></a>

### QueryParamsRequest
QueryParamsRequest is the request/response type for the module API.






<a name="uptick.appmining.v1.QueryParamsResponse"></a>

### QueryParamsResponse
QueryParamsResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#uptick.appmining.v1.Params) |  |  |






<a name="uptick.appmining.v1.QueryPoolRequest"></a>

### QueryPoolRequest
QueryPoolRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `pool_id` | [string](#string) |  |  |






<a name="uptick.appmining.v1.QueryPoolResponse"></a>

### QueryPoolResponse
QueryPoolResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `pool` | [Pool](#uptick.appmining.v1.Pool) |  |  |
| `module_balance` | [string](#string) |  | module account balance for pool.denom (for off-chain reconciliation) |
| `entitlement_flow_enabled` | [bool](#bool) |  | true when pool is open, not paused, and funded >= entitlement_enable_min_funded |






<a name="uptick.appmining.v1.QueryRolesRequest"></a>

### QueryRolesRequest
QueryRolesRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `role` | [EntitlementRole](#uptick.appmining.v1.EntitlementRole) |  | optional filter; UNSPECIFIED = all roles |
| `status` | [RoleKeyStatus](#uptick.appmining.v1.RoleKeyStatus) |  | optional filter; UNSPECIFIED = all statuses |






<a name="uptick.appmining.v1.QueryRolesResponse"></a>

### QueryRolesResponse
QueryRolesResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `keys` | [RoleKey](#uptick.appmining.v1.RoleKey) | repeated |  |






<a name="uptick.appmining.v1.QuerySettlementRequest"></a>

### QuerySettlementRequest
QuerySettlementRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `app_id` | [bytes](#bytes) |  |  |
| `epoch` | [uint64](#uint64) |  |  |






<a name="uptick.appmining.v1.QuerySettlementResponse"></a>

### QuerySettlementResponse
QuerySettlementResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `settlement` | [Settlement](#uptick.appmining.v1.Settlement) |  |  |






<a name="uptick.appmining.v1.QuerySettlementsRequest"></a>

### QuerySettlementsRequest
QuerySettlementsRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  |  |






<a name="uptick.appmining.v1.QuerySettlementsResponse"></a>

### QuerySettlementsResponse
QuerySettlementsResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `settlements` | [Settlement](#uptick.appmining.v1.Settlement) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.appmining.v1.Query"></a>

### Query
Query defines the gRPC query/tx service for x/appmining.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `Params` | [QueryParamsRequest](#uptick.appmining.v1.QueryParamsRequest) | [QueryParamsResponse](#uptick.appmining.v1.QueryParamsResponse) | Params handles the corresponding module request. | GET|/uptick/appmining/v1/params|
| `Application` | [QueryApplicationRequest](#uptick.appmining.v1.QueryApplicationRequest) | [QueryApplicationResponse](#uptick.appmining.v1.QueryApplicationResponse) | Application handles the corresponding module request. | GET|/uptick/appmining/v1/applications/{app_id}|
| `Applications` | [QueryApplicationsRequest](#uptick.appmining.v1.QueryApplicationsRequest) | [QueryApplicationsResponse](#uptick.appmining.v1.QueryApplicationsResponse) | Applications handles the corresponding module request. | GET|/uptick/appmining/v1/applications|
| `EpochBudget` | [QueryEpochBudgetRequest](#uptick.appmining.v1.QueryEpochBudgetRequest) | [QueryEpochBudgetResponse](#uptick.appmining.v1.QueryEpochBudgetResponse) | EpochBudget handles the corresponding module request. | GET|/uptick/appmining/v1/epoch_budgets/{epoch}|
| `Attestation` | [QueryAttestationRequest](#uptick.appmining.v1.QueryAttestationRequest) | [QueryAttestationResponse](#uptick.appmining.v1.QueryAttestationResponse) | Attestation handles the corresponding module request. | GET|/uptick/appmining/v1/attestations/{app_id}/{epoch}|
| `Attestations` | [QueryAttestationsRequest](#uptick.appmining.v1.QueryAttestationsRequest) | [QueryAttestationsResponse](#uptick.appmining.v1.QueryAttestationsResponse) | Attestations handles the corresponding module request. | GET|/uptick/appmining/v1/attestations|
| `Challenge` | [QueryChallengeRequest](#uptick.appmining.v1.QueryChallengeRequest) | [QueryChallengeResponse](#uptick.appmining.v1.QueryChallengeResponse) | Challenge handles the corresponding module request. | GET|/uptick/appmining/v1/challenges/{challenge_id}|
| `OpenChallenges` | [QueryOpenChallengesRequest](#uptick.appmining.v1.QueryOpenChallengesRequest) | [QueryOpenChallengesResponse](#uptick.appmining.v1.QueryOpenChallengesResponse) | OpenChallenges handles the corresponding module request. | GET|/uptick/appmining/v1/open_challenges|
| `Settlement` | [QuerySettlementRequest](#uptick.appmining.v1.QuerySettlementRequest) | [QuerySettlementResponse](#uptick.appmining.v1.QuerySettlementResponse) | Settlement handles the corresponding module request. | GET|/uptick/appmining/v1/settlements/{app_id}/{epoch}|
| `Settlements` | [QuerySettlementsRequest](#uptick.appmining.v1.QuerySettlementsRequest) | [QuerySettlementsResponse](#uptick.appmining.v1.QuerySettlementsResponse) | Settlements handles the corresponding module request. | GET|/uptick/appmining/v1/settlements|
| `GasCredit` | [QueryGasCreditRequest](#uptick.appmining.v1.QueryGasCreditRequest) | [QueryGasCreditResponse](#uptick.appmining.v1.QueryGasCreditResponse) | GasCredit handles the corresponding module request. | GET|/uptick/appmining/v1/gas_credits/{app_id}|
| `GasCreditConsumptions` | [QueryGasCreditConsumptionsRequest](#uptick.appmining.v1.QueryGasCreditConsumptionsRequest) | [QueryGasCreditConsumptionsResponse](#uptick.appmining.v1.QueryGasCreditConsumptionsResponse) | GasCreditConsumptions handles the corresponding module request. | GET|/uptick/appmining/v1/gas_credits/{app_id}/consumptions|
| `Pool` | [QueryPoolRequest](#uptick.appmining.v1.QueryPoolRequest) | [QueryPoolResponse](#uptick.appmining.v1.QueryPoolResponse) | V1 entitlement queries (UGE requirements 1.0) | GET|/uptick/appmining/v1/pools/{pool_id}|
| `Roles` | [QueryRolesRequest](#uptick.appmining.v1.QueryRolesRequest) | [QueryRolesResponse](#uptick.appmining.v1.QueryRolesResponse) | Roles handles the corresponding module request. | GET|/uptick/appmining/v1/roles|
| `Candidate` | [QueryCandidateRequest](#uptick.appmining.v1.QueryCandidateRequest) | [QueryCandidateResponse](#uptick.appmining.v1.QueryCandidateResponse) | Candidate handles the corresponding module request. | GET|/uptick/appmining/v1/candidates/{entitlement_id}|
| `EntitlementChallenge` | [QueryEntitlementChallengeRequest](#uptick.appmining.v1.QueryEntitlementChallengeRequest) | [QueryEntitlementChallengeResponse](#uptick.appmining.v1.QueryEntitlementChallengeResponse) | EntitlementChallenge handles the corresponding module request. | GET|/uptick/appmining/v1/entitlement_challenges/{challenge_id}|
| `Entitlement` | [QueryEntitlementRequest](#uptick.appmining.v1.QueryEntitlementRequest) | [QueryEntitlementResponse](#uptick.appmining.v1.QueryEntitlementResponse) | Entitlement handles the corresponding module request. | GET|/uptick/appmining/v1/entitlements/{entitlement_id}|
| `EntitlementsByApp` | [QueryEntitlementsByAppRequest](#uptick.appmining.v1.QueryEntitlementsByAppRequest) | [QueryEntitlementsByAppResponse](#uptick.appmining.v1.QueryEntitlementsByAppResponse) | EntitlementsByApp handles the corresponding module request. | GET|/uptick/appmining/v1/applications/{app_id}/entitlements|

 <!-- end services -->



<a name="uptick/appmining/v1/tx.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/appmining/v1/tx.proto



<a name="uptick.appmining.v1.MsgApproveEntitlement"></a>

### MsgApproveEntitlement
MsgApproveEntitlement locks pool available→reserved and creates CLAIMABLE entitlement.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `submitter` | [string](#string) |  |  |
| `chain_id` | [string](#string) |  |  |
| `pool_id` | [string](#string) |  |  |
| `entitlement_id` | [string](#string) |  |  |
| `challenge_resolution_hash` | [string](#string) |  | challenge_resolution_hash binds approval to dispute state (wire APPROVE field). |
| `proof` | [WireProof](#uptick.appmining.v1.WireProof) |  |  |






<a name="uptick.appmining.v1.MsgApproveEntitlementResponse"></a>

### MsgApproveEntitlementResponse
MsgApproveEntitlementResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgChallengeAttestation"></a>

### MsgChallengeAttestation
MsgChallengeAttestation is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `challenger` | [string](#string) |  |  |
| `app_id` | [bytes](#bytes) |  |  |
| `epoch` | [uint64](#uint64) |  |  |
| `evidence_hash` | [bytes](#bytes) |  |  |






<a name="uptick.appmining.v1.MsgChallengeAttestationResponse"></a>

### MsgChallengeAttestationResponse
MsgChallengeAttestationResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `challenge_id` | [uint64](#uint64) |  |  |






<a name="uptick.appmining.v1.MsgClaimEntitlement"></a>

### MsgClaimEntitlement
MsgClaimEntitlement pays reserved funds to the current app owner (beneficiary = owner).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `owner` | [string](#string) |  |  |
| `entitlement_id` | [string](#string) |  |  |






<a name="uptick.appmining.v1.MsgClaimEntitlementResponse"></a>

### MsgClaimEntitlementResponse
MsgClaimEntitlementResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `amount` | [string](#string) |  |  |
| `denom` | [string](#string) |  |  |






<a name="uptick.appmining.v1.MsgConsumeGasCredit"></a>

### MsgConsumeGasCredit
MsgConsumeGasCredit is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `gas_payer` | [string](#string) |  |  |
| `app_id` | [bytes](#bytes) |  |  |
| `amount` | [string](#string) |  |  |
| `source_tx_hash` | [bytes](#bytes) |  |  |






<a name="uptick.appmining.v1.MsgConsumeGasCreditResponse"></a>

### MsgConsumeGasCreditResponse
MsgConsumeGasCreditResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `remaining_balance` | [string](#string) |  |  |






<a name="uptick.appmining.v1.MsgFinalizeAttestation"></a>

### MsgFinalizeAttestation
MsgFinalizeAttestation is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `caller` | [string](#string) |  |  |
| `app_id` | [bytes](#bytes) |  |  |
| `epoch` | [uint64](#uint64) |  |  |






<a name="uptick.appmining.v1.MsgFinalizeAttestationResponse"></a>

### MsgFinalizeAttestationResponse
MsgFinalizeAttestationResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `settlement` | [Settlement](#uptick.appmining.v1.Settlement) |  |  |






<a name="uptick.appmining.v1.MsgFreezeEntitlement"></a>

### MsgFreezeEntitlement
MsgFreezeEntitlement temporarily blocks claim (max 72h, once per entitlement).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `submitter` | [string](#string) |  |  |
| `chain_id` | [string](#string) |  |  |
| `pool_id` | [string](#string) |  |  |
| `entitlement_id` | [string](#string) |  |  |
| `evidence_type` | [EvidenceType](#uptick.appmining.v1.EvidenceType) |  |  |
| `evidence_digest` | [string](#string) |  |  |
| `freeze_until` | [string](#string) |  | unix seconds decimal |
| `proof` | [WireProof](#uptick.appmining.v1.WireProof) |  |  |






<a name="uptick.appmining.v1.MsgFreezeEntitlementResponse"></a>

### MsgFreezeEntitlementResponse
MsgFreezeEntitlementResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgFundEntitlementPool"></a>

### MsgFundEntitlementPool
MsgFundEntitlementPool deposits coins into appmining_entitlement_pool and
increases Pool.funded and Pool.available by the same amount (no mint).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `funder` | [string](#string) |  |  |
| `pool_id` | [string](#string) |  |  |
| `amount` | [string](#string) |  |  |






<a name="uptick.appmining.v1.MsgFundEntitlementPoolResponse"></a>

### MsgFundEntitlementPoolResponse
MsgFundEntitlementPoolResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgFundRewardPool"></a>

### MsgFundRewardPool
MsgFundRewardPool deposits coins into appmining_reward_pool.
Any allowlisted-denom holder may fund (funder ≠ governance authority on purpose:
gov module accounts cannot practically sign/hold spendable balances).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `funder` | [string](#string) |  |  |
| `denom` | [string](#string) |  |  |
| `amount` | [string](#string) |  |  |






<a name="uptick.appmining.v1.MsgFundRewardPoolResponse"></a>

### MsgFundRewardPoolResponse
MsgFundRewardPoolResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgGrantGasPayer"></a>

### MsgGrantGasPayer
MsgGrantGasPayer is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `owner` | [string](#string) |  |  |
| `app_id` | [bytes](#bytes) |  |  |
| `gas_payer` | [string](#string) |  |  |






<a name="uptick.appmining.v1.MsgGrantGasPayerResponse"></a>

### MsgGrantGasPayerResponse
MsgGrantGasPayerResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgOpenChallenge"></a>

### MsgOpenChallenge
MsgOpenChallenge opens a public challenge against a published V1 candidate.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `challenger` | [string](#string) |  |  |
| `entitlement_id` | [string](#string) |  |  |
| `evidence_type` | [EvidenceType](#uptick.appmining.v1.EvidenceType) |  |  |
| `evidence_digest` | [string](#string) |  |  |
| `reference_digest` | [string](#string) |  | optional; empty allowed -> normalized to 64 zero hex |






<a name="uptick.appmining.v1.MsgOpenChallengeResponse"></a>

### MsgOpenChallengeResponse
MsgOpenChallengeResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `challenge_id` | [string](#string) |  |  |






<a name="uptick.appmining.v1.MsgOpenEntitlementPool"></a>

### MsgOpenEntitlementPool
MsgOpenEntitlementPool creates a V1 fixed fund pool ledger (funded=0).
Does not mint; activation requires FundEntitlementPool to reach enable min.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  |  |
| `pool_id` | [string](#string) |  |  |
| `denom` | [string](#string) |  |  |
| `version` | [uint64](#uint64) |  |  |






<a name="uptick.appmining.v1.MsgOpenEntitlementPoolResponse"></a>

### MsgOpenEntitlementPoolResponse
MsgOpenEntitlementPoolResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgPause"></a>

### MsgPause
MsgPause applies only true bits from pause_bits onto current PauseState (OR).
Any attempt to clear a bit via this message is rejected (Q-06 PAUSE_ONLY_TRUE).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `pause_authority` | [string](#string) |  |  |
| `pause_bits` | [PauseState](#uptick.appmining.v1.PauseState) |  |  |






<a name="uptick.appmining.v1.MsgPauseResponse"></a>

### MsgPauseResponse
MsgPauseResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgPublishCandidate"></a>

### MsgPublishCandidate
MsgPublishCandidate publishes an immutable candidate commitment (no lock).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `submitter` | [string](#string) |  |  |
| `chain_id` | [string](#string) |  |  |
| `pool_id` | [string](#string) |  |  |
| `entitlement_id` | [string](#string) |  | Candidate body (schema fields); status/timestamps filled by keeper. |
| `app_id` | [string](#string) |  |  |
| `epoch` | [string](#string) |  |  |
| `amount` | [string](#string) |  |  |
| `denom` | [string](#string) |  |  |
| `gas_used` | [string](#string) |  |  |
| `qualified_tx_count` | [string](#string) |  |  |
| `pav_id` | [string](#string) |  |  |
| `evidence_root` | [string](#string) |  |  |
| `gas_evidence_root` | [string](#string) |  |  |
| `rule_version` | [string](#string) |  |  |
| `proof` | [WireProof](#uptick.appmining.v1.WireProof) |  |  |






<a name="uptick.appmining.v1.MsgPublishCandidateResponse"></a>

### MsgPublishCandidateResponse
MsgPublishCandidateResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `candidate_hash` | [string](#string) |  |  |
| `challenge_deadline` | [int64](#int64) |  |  |






<a name="uptick.appmining.v1.MsgRegisterApplication"></a>

### MsgRegisterApplication
MsgRegisterApplication is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  |  |
| `app_id` | [bytes](#bytes) |  |  |
| `owner` | [string](#string) |  |  |
| `attestation_submitter` | [string](#string) |  |  |
| `metadata_commitment` | [bytes](#bytes) |  | optional; empty or exactly 32 bytes |






<a name="uptick.appmining.v1.MsgRegisterApplicationResponse"></a>

### MsgRegisterApplicationResponse
MsgRegisterApplicationResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgRegisterRoleKey"></a>

### MsgRegisterRoleKey
MsgRegisterRoleKey registers or rotates an Ed25519 role public key.
Same (role, key_id) with identical pubkey is idempotent; conflicting pubkey rejected.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  |  |
| `role` | [EntitlementRole](#uptick.appmining.v1.EntitlementRole) |  |  |
| `key_id` | [string](#string) |  |  |
| `pubkey` | [bytes](#bytes) |  | raw Ed25519, 32 bytes |
| `version` | [uint64](#uint64) |  |  |






<a name="uptick.appmining.v1.MsgRegisterRoleKeyResponse"></a>

### MsgRegisterRoleKeyResponse
MsgRegisterRoleKeyResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgResolveChallenge"></a>

### MsgResolveChallenge
MsgResolveChallenge is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  |  |
| `challenge_id` | [uint64](#uint64) |  |  |
| `uphold` | [bool](#bool) |  |  |
| `resolution_hash` | [bytes](#bytes) |  |  |






<a name="uptick.appmining.v1.MsgResolveChallengeResponse"></a>

### MsgResolveChallengeResponse
MsgResolveChallengeResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgResolveEntitlementChallenge"></a>

### MsgResolveEntitlementChallenge
MsgResolveEntitlementChallenge resolves an OPEN entitlement challenge.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `submitter` | [string](#string) |  |  |
| `chain_id` | [string](#string) |  |  |
| `pool_id` | [string](#string) |  |  |
| `challenge_id` | [string](#string) |  |  |
| `outcome` | [string](#string) |  | UPHELD | DISMISSED |
| `rationale_digest` | [string](#string) |  |  |
| `proof` | [WireProof](#uptick.appmining.v1.WireProof) |  |  |






<a name="uptick.appmining.v1.MsgResolveEntitlementChallengeResponse"></a>

### MsgResolveEntitlementChallengeResponse
MsgResolveEntitlementChallengeResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgRevokeEntitlement"></a>

### MsgRevokeEntitlement
MsgRevokeEntitlement requires independent risk + audit wire proofs while frozen.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `submitter` | [string](#string) |  |  |
| `chain_id` | [string](#string) |  |  |
| `pool_id` | [string](#string) |  |  |
| `entitlement_id` | [string](#string) |  |  |
| `evidence_type` | [EvidenceType](#uptick.appmining.v1.EvidenceType) |  |  |
| `evidence_digest` | [string](#string) |  |  |
| `rationale_digest` | [string](#string) |  |  |
| `risk_proof` | [WireProof](#uptick.appmining.v1.WireProof) |  |  |
| `audit_proof` | [WireProof](#uptick.appmining.v1.WireProof) |  |  |






<a name="uptick.appmining.v1.MsgRevokeEntitlementResponse"></a>

### MsgRevokeEntitlementResponse
MsgRevokeEntitlementResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgRevokeGasPayer"></a>

### MsgRevokeGasPayer
MsgRevokeGasPayer is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `owner` | [string](#string) |  |  |
| `app_id` | [bytes](#bytes) |  |  |
| `gas_payer` | [string](#string) |  |  |






<a name="uptick.appmining.v1.MsgRevokeGasPayerResponse"></a>

### MsgRevokeGasPayerResponse
MsgRevokeGasPayerResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgSetApplicationStatus"></a>

### MsgSetApplicationStatus
MsgSetApplicationStatus is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  |  |
| `app_id` | [bytes](#bytes) |  |  |
| `status` | [ApplicationStatus](#uptick.appmining.v1.ApplicationStatus) |  |  |






<a name="uptick.appmining.v1.MsgSetApplicationStatusResponse"></a>

### MsgSetApplicationStatusResponse
MsgSetApplicationStatusResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgSetAttestationSubmitter"></a>

### MsgSetAttestationSubmitter
MsgSetAttestationSubmitter is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `owner` | [string](#string) |  |  |
| `app_id` | [bytes](#bytes) |  |  |
| `attestation_submitter` | [string](#string) |  |  |






<a name="uptick.appmining.v1.MsgSetAttestationSubmitterResponse"></a>

### MsgSetAttestationSubmitterResponse
MsgSetAttestationSubmitterResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgSetEntitlementPoolPaused"></a>

### MsgSetEntitlementPoolPaused
MsgSetEntitlementPoolPaused sets Pool.paused (authority only).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  |  |
| `pool_id` | [string](#string) |  |  |
| `paused` | [bool](#bool) |  |  |






<a name="uptick.appmining.v1.MsgSetEntitlementPoolPausedResponse"></a>

### MsgSetEntitlementPoolPausedResponse
MsgSetEntitlementPoolPausedResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgSetEpochBudget"></a>

### MsgSetEpochBudget
MsgSetEpochBudget is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  |  |
| `budget` | [EpochBudget](#uptick.appmining.v1.EpochBudget) |  |  |






<a name="uptick.appmining.v1.MsgSetEpochBudgetResponse"></a>

### MsgSetEpochBudgetResponse
MsgSetEpochBudgetResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgSetRoleKeyStatus"></a>

### MsgSetRoleKeyStatus
MsgSetRoleKeyStatus pauses or revokes a role key (ACTIVE keys only authorize new msgs).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  |  |
| `role` | [EntitlementRole](#uptick.appmining.v1.EntitlementRole) |  |  |
| `key_id` | [string](#string) |  |  |
| `status` | [RoleKeyStatus](#uptick.appmining.v1.RoleKeyStatus) |  | ACTIVE | PAUSED | REVOKED |






<a name="uptick.appmining.v1.MsgSetRoleKeyStatusResponse"></a>

### MsgSetRoleKeyStatusResponse
MsgSetRoleKeyStatusResponse is the request/response type for the module API.






<a name="uptick.appmining.v1.MsgSubmitAttestation"></a>

### MsgSubmitAttestation
MsgSubmitAttestation is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `submitter` | [string](#string) |  |  |
| `attestation` | [AttestationCommitment](#uptick.appmining.v1.AttestationCommitment) |  |  |






<a name="uptick.appmining.v1.MsgSubmitAttestationResponse"></a>

### MsgSubmitAttestationResponse
MsgSubmitAttestationResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `challenge_deadline_height` | [int64](#int64) |  |  |






<a name="uptick.appmining.v1.MsgUpdateParams"></a>

### MsgUpdateParams
MsgUpdateParams is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  |  |
| `params` | [Params](#uptick.appmining.v1.Params) |  |  |






<a name="uptick.appmining.v1.MsgUpdateParamsResponse"></a>

### MsgUpdateParamsResponse
MsgUpdateParamsResponse is the request/response type for the module API.





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.appmining.v1.Msg"></a>

### Msg
Msg defines the gRPC query/tx service for x/appmining.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `RegisterApplication` | [MsgRegisterApplication](#uptick.appmining.v1.MsgRegisterApplication) | [MsgRegisterApplicationResponse](#uptick.appmining.v1.MsgRegisterApplicationResponse) | RegisterApplication handles the corresponding module request. | POST|/uptick/appmining/v1/tx/register_application|
| `SubmitAttestation` | [MsgSubmitAttestation](#uptick.appmining.v1.MsgSubmitAttestation) | [MsgSubmitAttestationResponse](#uptick.appmining.v1.MsgSubmitAttestationResponse) | SubmitAttestation handles the corresponding module request. | POST|/uptick/appmining/v1/tx/submit_attestation|
| `ChallengeAttestation` | [MsgChallengeAttestation](#uptick.appmining.v1.MsgChallengeAttestation) | [MsgChallengeAttestationResponse](#uptick.appmining.v1.MsgChallengeAttestationResponse) | ChallengeAttestation handles the corresponding module request. | POST|/uptick/appmining/v1/tx/challenge_attestation|
| `ResolveChallenge` | [MsgResolveChallenge](#uptick.appmining.v1.MsgResolveChallenge) | [MsgResolveChallengeResponse](#uptick.appmining.v1.MsgResolveChallengeResponse) | ResolveChallenge handles the corresponding module request. | |
| `FinalizeAttestation` | [MsgFinalizeAttestation](#uptick.appmining.v1.MsgFinalizeAttestation) | [MsgFinalizeAttestationResponse](#uptick.appmining.v1.MsgFinalizeAttestationResponse) | FinalizeAttestation handles the corresponding module request. | POST|/uptick/appmining/v1/tx/finalize_attestation|
| `GrantGasPayer` | [MsgGrantGasPayer](#uptick.appmining.v1.MsgGrantGasPayer) | [MsgGrantGasPayerResponse](#uptick.appmining.v1.MsgGrantGasPayerResponse) | GrantGasPayer handles the corresponding module request. | |
| `RevokeGasPayer` | [MsgRevokeGasPayer](#uptick.appmining.v1.MsgRevokeGasPayer) | [MsgRevokeGasPayerResponse](#uptick.appmining.v1.MsgRevokeGasPayerResponse) | RevokeGasPayer handles the corresponding module request. | |
| `SetAttestationSubmitter` | [MsgSetAttestationSubmitter](#uptick.appmining.v1.MsgSetAttestationSubmitter) | [MsgSetAttestationSubmitterResponse](#uptick.appmining.v1.MsgSetAttestationSubmitterResponse) | SetAttestationSubmitter handles the corresponding module request. | |
| `ConsumeGasCredit` | [MsgConsumeGasCredit](#uptick.appmining.v1.MsgConsumeGasCredit) | [MsgConsumeGasCreditResponse](#uptick.appmining.v1.MsgConsumeGasCreditResponse) | ConsumeGasCredit handles the corresponding module request. | POST|/uptick/appmining/v1/tx/consume_gas_credit|
| `SetEpochBudget` | [MsgSetEpochBudget](#uptick.appmining.v1.MsgSetEpochBudget) | [MsgSetEpochBudgetResponse](#uptick.appmining.v1.MsgSetEpochBudgetResponse) | SetEpochBudget handles the corresponding module request. | |
| `UpdateParams` | [MsgUpdateParams](#uptick.appmining.v1.MsgUpdateParams) | [MsgUpdateParamsResponse](#uptick.appmining.v1.MsgUpdateParamsResponse) | UpdateParams handles the corresponding module request. | |
| `SetApplicationStatus` | [MsgSetApplicationStatus](#uptick.appmining.v1.MsgSetApplicationStatus) | [MsgSetApplicationStatusResponse](#uptick.appmining.v1.MsgSetApplicationStatusResponse) | SetApplicationStatus handles the corresponding module request. | |
| `Pause` | [MsgPause](#uptick.appmining.v1.MsgPause) | [MsgPauseResponse](#uptick.appmining.v1.MsgPauseResponse) | Pause sets pause bits to true only (Q-06). Resume is authority-only via UpdateParams. | |
| `FundRewardPool` | [MsgFundRewardPool](#uptick.appmining.v1.MsgFundRewardPool) | [MsgFundRewardPoolResponse](#uptick.appmining.v1.MsgFundRewardPoolResponse) | FundRewardPool transfers coins from authority into the module reward pool (C-5 / stage 2). | POST|/uptick/appmining/v1/tx/fund_reward_pool|
| `OpenEntitlementPool` | [MsgOpenEntitlementPool](#uptick.appmining.v1.MsgOpenEntitlementPool) | [MsgOpenEntitlementPoolResponse](#uptick.appmining.v1.MsgOpenEntitlementPoolResponse) | --- V1 entitlement control plane (UGE requirements 1.0 / workorder P0) --- OpenEntitlementPool creates a V1 fixed entitlement fund pool ledger. | POST|/uptick/appmining/v1/tx/open_entitlement_pool|
| `FundEntitlementPool` | [MsgFundEntitlementPool](#uptick.appmining.v1.MsgFundEntitlementPool) | [MsgFundEntitlementPoolResponse](#uptick.appmining.v1.MsgFundEntitlementPoolResponse) | FundEntitlementPool handles the corresponding module request. | POST|/uptick/appmining/v1/tx/fund_entitlement_pool|
| `SetEntitlementPoolPaused` | [MsgSetEntitlementPoolPaused](#uptick.appmining.v1.MsgSetEntitlementPoolPaused) | [MsgSetEntitlementPoolPausedResponse](#uptick.appmining.v1.MsgSetEntitlementPoolPausedResponse) | SetEntitlementPoolPaused handles the corresponding module request. | POST|/uptick/appmining/v1/tx/set_entitlement_pool_paused|
| `RegisterRoleKey` | [MsgRegisterRoleKey](#uptick.appmining.v1.MsgRegisterRoleKey) | [MsgRegisterRoleKeyResponse](#uptick.appmining.v1.MsgRegisterRoleKeyResponse) | RegisterRoleKey handles the corresponding module request. | POST|/uptick/appmining/v1/tx/register_role_key|
| `SetRoleKeyStatus` | [MsgSetRoleKeyStatus](#uptick.appmining.v1.MsgSetRoleKeyStatus) | [MsgSetRoleKeyStatusResponse](#uptick.appmining.v1.MsgSetRoleKeyStatusResponse) | SetRoleKeyStatus handles the corresponding module request. | POST|/uptick/appmining/v1/tx/set_role_key_status|
| `PublishCandidate` | [MsgPublishCandidate](#uptick.appmining.v1.MsgPublishCandidate) | [MsgPublishCandidateResponse](#uptick.appmining.v1.MsgPublishCandidateResponse) | PublishCandidate publishes an immutable V1 candidate commitment (no pool lock). | POST|/uptick/appmining/v1/tx/publish_candidate|
| `OpenChallenge` | [MsgOpenChallenge](#uptick.appmining.v1.MsgOpenChallenge) | [MsgOpenChallengeResponse](#uptick.appmining.v1.MsgOpenChallengeResponse) | OpenChallenge is the V1 entitlement challenge opener (attestation uses ChallengeAttestation). | POST|/uptick/appmining/v1/tx/open_challenge|
| `ResolveEntitlementChallenge` | [MsgResolveEntitlementChallenge](#uptick.appmining.v1.MsgResolveEntitlementChallenge) | [MsgResolveEntitlementChallengeResponse](#uptick.appmining.v1.MsgResolveEntitlementChallengeResponse) | ResolveEntitlementChallenge avoids clashing with attestation MsgResolveChallenge. | POST|/uptick/appmining/v1/tx/resolve_entitlement_challenge|
| `ApproveEntitlement` | [MsgApproveEntitlement](#uptick.appmining.v1.MsgApproveEntitlement) | [MsgApproveEntitlementResponse](#uptick.appmining.v1.MsgApproveEntitlementResponse) | ApproveEntitlement handles the corresponding module request. | POST|/uptick/appmining/v1/tx/approve_entitlement|
| `ClaimEntitlement` | [MsgClaimEntitlement](#uptick.appmining.v1.MsgClaimEntitlement) | [MsgClaimEntitlementResponse](#uptick.appmining.v1.MsgClaimEntitlementResponse) | ClaimEntitlement handles the corresponding module request. | POST|/uptick/appmining/v1/tx/claim_entitlement|
| `FreezeEntitlement` | [MsgFreezeEntitlement](#uptick.appmining.v1.MsgFreezeEntitlement) | [MsgFreezeEntitlementResponse](#uptick.appmining.v1.MsgFreezeEntitlementResponse) | FreezeEntitlement handles the corresponding module request. | POST|/uptick/appmining/v1/tx/freeze_entitlement|
| `RevokeEntitlement` | [MsgRevokeEntitlement](#uptick.appmining.v1.MsgRevokeEntitlement) | [MsgRevokeEntitlementResponse](#uptick.appmining.v1.MsgRevokeEntitlementResponse) | RevokeEntitlement handles the corresponding module request. | POST|/uptick/appmining/v1/tx/revoke_entitlement|

 <!-- end services -->



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
DenomMetadata defines the metadata for a Denom
Contains information about creator, schema, restrictions and additional data


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
NFTMetadata defines the metadata for a NFT
Contains basic information like name and data


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
QuerySupplyRequest is the request type for the Query/Supply RPC method


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
| `EditNFT` | [MsgEditNFT](#uptick.collection.v1.MsgEditNFT) | [MsgEditNFTResponse](#uptick.collection.v1.MsgEditNFTResponse) | EditNFT defines a method for editing a nft. | |
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
| `nft_uid_pairs` | [NFTUIDPair](#uptick.cw721.v1.NFTUIDPair) | repeated | per-token conversion bindings between CW721 token UIDs and Cosmos NFT UIDs. |
| `refund_receivers` | [RefundReceiver](#uptick.cw721.v1.RefundReceiver) | repeated | original CW721 owners recorded at IBC send, used for timeout/error refunds. |






<a name="uptick.cw721.v1.NFTUIDPair"></a>

### NFTUIDPair
NFTUIDPair binds a CW721 token UID ("<tokenID>,<contractAddress>") to a
Cosmos NFT UID ("<nftID>,<classID>"). Import rebuilds both directions of
the runtime bidirectional index from this single entry.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token_uid` | [string](#string) |  |  |
| `nft_uid` | [string](#string) |  |  |






<a name="uptick.cw721.v1.Params"></a>

### Params
Params defines the cw721 module params


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `enable_cw721` | [bool](#bool) |  | parameter to enable the conversion of Cosmos nft <--> CW721 tokens. |
| `enable_evm_hook` | [bool](#bool) |  | parameter to enable the EVM hook that converts an CW721 token to a Cosmos NFT by transferring the Tokens through a MsgEthereumTx to the ModuleAddress Ethereum address. |
| `wasm_code_id` | [uint64](#uint64) |  | code ID of the CW721 wasm contract used for auto-instantiation. |






<a name="uptick.cw721.v1.RefundReceiver"></a>

### RefundReceiver
RefundReceiver records the original CW721 owner of a converted token so an
IBC timeout/error refund can return the token to the right address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `contract_address` | [string](#string) |  | lowercased CW721 contract address |
| `token_id` | [string](#string) |  |  |
| `owner` | [string](#string) |  | owner address of the original token |





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
| `class_id` | [string](#string) |  |  |






<a name="uptick.cw721.v1.QueryWasmContractResponse"></a>

### QueryWasmContractResponse
QueryWasmContractResponse is the response type for the Query/WasmContract RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token_pair` | [TokenPair](#uptick.cw721.v1.TokenPair) |  |  |





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
| `WasmContract` | [QueryWasmAddressRequest](#uptick.cw721.v1.QueryWasmAddressRequest) | [QueryWasmContractResponse](#uptick.cw721.v1.QueryWasmContractResponse) | WasmContract retrieves a registered wasm contract | GET|/uptick/cw721/v1/wasm_contract/{port}/{channel}/{class_id}|
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
| `receiver` | [string](#string) |  | bech32 address to receive native Cosmos NFTs |
| `sender` | [string](#string) |  | sender hex address from the owner of the given CW721 tokens |
| `class_id` | [string](#string) |  | nft classID to convert to CW721 |
| `nft_ids` | [string](#string) | repeated | nftID to convert to CW721 |






<a name="uptick.cw721.v1.MsgConvertCW721Response"></a>

### MsgConvertCW721Response
MsgConvertCW721Response returns no fields


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `contract_address` | [string](#string) |  | CW721 token contract address registered in a token pair |
| `token_ids` | [string](#string) | repeated | tokenID to convert |
| `receiver` | [string](#string) |  | bech32 address to receive native Cosmos coins |
| `sender` | [string](#string) |  | sender hex address from the owner of the given CW721 tokens |
| `class_id` | [string](#string) |  | nft classID to convert to CW721 |
| `nft_ids` | [string](#string) | repeated | nftID to convert to CW721 |






<a name="uptick.cw721.v1.MsgConvertNFT"></a>

### MsgConvertNFT
MsgConvertNFT defines a Msg to convert a native Cosmos nft to a CW721 token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  | nft classID to convert to CW721 |
| `nft_ids` | [string](#string) | repeated | nftID to convert to CW721 |
| `receiver` | [string](#string) |  | recipient hex address to receive CW721 token |
| `sender` | [string](#string) |  | cosmos bech32 address from the owner of the given Cosmos NFTs |
| `contract_address` | [string](#string) |  | CW721 token contract address registered in a token pair |
| `token_ids` | [string](#string) | repeated | CW721 token id registered in a token pair |






<a name="uptick.cw721.v1.MsgConvertNFTResponse"></a>

### MsgConvertNFTResponse
MsgConvertNFTResponse returns no fields






<a name="uptick.cw721.v1.MsgTransferCW721"></a>

### MsgTransferCW721
MsgTransferCW721 defines a message to transfer CW721 tokens between chains via IBC
It contains information about the source and destination of the transfer,
token identifiers, timeout parameters and optional memo


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
MsgTransferCW721Response defines the response type for TransferCW721 RPC






<a name="uptick.cw721.v1.MsgUpdateParams"></a>

### MsgUpdateParams
MsgUpdateParams is the Msg/UpdateParams request type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  | authority is the address of the governance account. |
| `params` | [Params](#uptick.cw721.v1.Params) |  | params defines the x/cw721 module parameters to update.

Since: v0.4.1 |






<a name="uptick.cw721.v1.MsgUpdateParamsResponse"></a>

### MsgUpdateParamsResponse
MsgUpdateParamsResponse is the Msg/UpdateParams response type.





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.cw721.v1.Msg"></a>

### Msg
Msg defines the cw721 Msg service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `ConvertNFT` | [MsgConvertNFT](#uptick.cw721.v1.MsgConvertNFT) | [MsgConvertNFTResponse](#uptick.cw721.v1.MsgConvertNFTResponse) | ConvertNFT mints a CW721 representation of the native Cosmos nft that is registered on the token mapping. | GET|/uptick/cw721/v1/tx/convert_nft|
| `ConvertCW721` | [MsgConvertCW721](#uptick.cw721.v1.MsgConvertCW721) | [MsgConvertCW721Response](#uptick.cw721.v1.MsgConvertCW721Response) | ConvertCW721 mints a native Cosmos coin representation of the CW721 token contract that is registered on the token mapping. | GET|/uptick/cw721/v1/tx/convert_cw721|
| `TransferCW721` | [MsgTransferCW721](#uptick.cw721.v1.MsgTransferCW721) | [MsgTransferCW721Response](#uptick.cw721.v1.MsgTransferCW721Response) | TransferCW721 defines a method to transfer CW721 tokens between chains via IBC | GET|/uptick/cw721/v1/tx/ibc-transfer-cw721|
| `UpdateParams` | [MsgUpdateParams](#uptick.cw721.v1.MsgUpdateParams) | [MsgUpdateParamsResponse](#uptick.cw721.v1.MsgUpdateParamsResponse) | UpdateParams updates the module parameters. Since the cw721 module has no authority account, this RPC is registered for interface completeness and is only callable by the governance authority. | |

 <!-- end services -->



<a name="uptick/erc721/v1/erc721.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/erc721/v1/erc721.proto



<a name="uptick.erc721.v1.TokenPair"></a>

### TokenPair
TokenPair defines an instance that records a pairing consisting of a native
Cosmos Coin and an ERC721 token address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `erc721_address` | [string](#string) |  | address of ERC721 contract token |
| `class_id` | [string](#string) |  | cosmos nft class ID to be mapped to |






<a name="uptick.erc721.v1.UIDPair"></a>

### UIDPair
defines the unique id of nft asset


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `erc721_did` | [string](#string) |  | address of ERC721 contract token + tokenId |
| `class_did` | [string](#string) |  | cosmos nft class ID to be mapped to + nftId |





 <!-- end messages -->


<a name="uptick.erc721.v1.Owner"></a>

### Owner
Owner enumerates the ownership of a ERC721 contract.

| Name | Number | Description |
| ---- | ------ | ----------- |
| OWNER_UNSPECIFIED | 0 | OWNER_UNSPECIFIED defines an invalid/undefined owner. |
| OWNER_MODULE | 1 | OWNER_MODULE erc721 is owned by the erc721 module account. |
| OWNER_EXTERNAL | 2 | EXTERNAL erc721 is owned by an external account. |


 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/erc721/v1/genesis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/erc721/v1/genesis.proto



<a name="uptick.erc721.v1.GenesisState"></a>

### GenesisState
GenesisState defines the module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#uptick.erc721.v1.Params) |  | module parameters |
| `token_pairs` | [TokenPair](#uptick.erc721.v1.TokenPair) | repeated | registered token pairs |
| `nft_uid_pairs` | [NFTUIDPair](#uptick.erc721.v1.NFTUIDPair) | repeated | per-token conversion bindings between EVM token UIDs and Cosmos NFT UIDs. |
| `refund_receivers` | [RefundReceiver](#uptick.erc721.v1.RefundReceiver) | repeated | original ERC721 owners recorded at IBC send, used for timeout/error refunds. |






<a name="uptick.erc721.v1.NFTUIDPair"></a>

### NFTUIDPair
NFTUIDPair binds an EVM token UID ("<tokenID>,<contractAddress>") to a
Cosmos NFT UID ("<nftID>,<classID>"). Import rebuilds both directions of
the runtime bidirectional index from this single entry.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token_uid` | [string](#string) |  |  |
| `nft_uid` | [string](#string) |  |  |






<a name="uptick.erc721.v1.Params"></a>

### Params
Params defines the erc721 module params


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `enable_erc721` | [bool](#bool) |  | parameter to enable the conversion of Cosmos nft <--> ERC721 tokens. |
| `enable_evm_hook` | [bool](#bool) |  | parameter to enable the EVM hook that converts an ERC721 token to a Cosmos NFT by transferring the Tokens through a MsgEthereumTx to the ModuleAddress Ethereum address. |






<a name="uptick.erc721.v1.RefundReceiver"></a>

### RefundReceiver
RefundReceiver records the original ERC721 owner of a converted token so an
IBC timeout/error refund can return the token to the right address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `evm_contract_address` | [string](#string) |  | lowercased ERC721 contract address (0x + 40 hex) |
| `token_id` | [string](#string) |  |  |
| `evm_address` | [string](#string) |  | EVM address of the original owner (0x + 40 hex) |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/erc721/v1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/erc721/v1/query.proto



<a name="uptick.erc721.v1.QueryEvmAddressRequest"></a>

### QueryEvmAddressRequest
QueryTokenPairRequest is the request type for the Query/TokenPair RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `port` | [string](#string) |  | token identifier can be either the hex contract address of the ERC721 or the Cosmos nft classID |
| `channel` | [string](#string) |  |  |
| `class_id` | [string](#string) |  |  |






<a name="uptick.erc721.v1.QueryEvmAddressResponse"></a>

### QueryEvmAddressResponse
QueryEvmAddressResponse is the response type for the Query/Params RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token_pair` | [TokenPair](#uptick.erc721.v1.TokenPair) |  |  |






<a name="uptick.erc721.v1.QueryParamsRequest"></a>

### QueryParamsRequest
QueryParamsRequest is the request type for the Query/Params RPC method.






<a name="uptick.erc721.v1.QueryParamsResponse"></a>

### QueryParamsResponse
QueryParamsResponse is the response type for the Query/Params RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#uptick.erc721.v1.Params) |  |  |






<a name="uptick.erc721.v1.QueryTokenPairRequest"></a>

### QueryTokenPairRequest
QueryTokenPairRequest is the request type for the Query/TokenPair RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token` | [string](#string) |  | token identifier can be either the hex contract address of the ERC721 or the Cosmos nft classID |






<a name="uptick.erc721.v1.QueryTokenPairResponse"></a>

### QueryTokenPairResponse
QueryTokenPairResponse is the response type for the Query/TokenPair RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token_pair` | [TokenPair](#uptick.erc721.v1.TokenPair) |  |  |






<a name="uptick.erc721.v1.QueryTokenPairsRequest"></a>

### QueryTokenPairsRequest
QueryTokenPairsRequest is the request type for the Query/TokenPairs RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  | pagination defines an optional pagination for the request. |






<a name="uptick.erc721.v1.QueryTokenPairsResponse"></a>

### QueryTokenPairsResponse
QueryTokenPairsResponse is the response type for the Query/TokenPairs RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token_pairs` | [TokenPair](#uptick.erc721.v1.TokenPair) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  | pagination defines the pagination in the response. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.erc721.v1.Query"></a>

### Query
Query defines the gRPC queried service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `TokenPairs` | [QueryTokenPairsRequest](#uptick.erc721.v1.QueryTokenPairsRequest) | [QueryTokenPairsResponse](#uptick.erc721.v1.QueryTokenPairsResponse) | TokenPairs retrieves registered token pairs | GET|/uptick/erc721/v1/token_pairs|
| `TokenPair` | [QueryTokenPairRequest](#uptick.erc721.v1.QueryTokenPairRequest) | [QueryTokenPairResponse](#uptick.erc721.v1.QueryTokenPairResponse) | TokenPair retrieves a registered token pair | GET|/uptick/erc721/v1/token_pairs/{token}|
| `EvmContract` | [QueryEvmAddressRequest](#uptick.erc721.v1.QueryEvmAddressRequest) | [QueryEvmAddressResponse](#uptick.erc721.v1.QueryEvmAddressResponse) | EvmContract retrieves a registered evm contract | GET|/uptick/erc721/v1/evm_contract/{port}/{channel}/{class_id}|
| `Params` | [QueryParamsRequest](#uptick.erc721.v1.QueryParamsRequest) | [QueryParamsResponse](#uptick.erc721.v1.QueryParamsResponse) | Params retrieves the erc721 module params | GET|/uptick/erc721/v1/params|

 <!-- end services -->



<a name="uptick/erc721/v1/tx.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/erc721/v1/tx.proto



<a name="uptick.erc721.v1.MsgConvertERC721"></a>

### MsgConvertERC721
MsgConvertERC721 defines a Msg to convert a ERC721 token to a native Cosmos
nft.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `evm_contract_address` | [string](#string) |  | ERC721 token contract address registered in a token pair |
| `evm_token_ids` | [string](#string) | repeated | tokenID to convert |
| `cosmos_receiver` | [string](#string) |  | bech32 address to receive native Cosmos coins |
| `cosmos_sender` | [string](#string) |  | sender bech32 address from the owner of the given ERC721 tokens |
| `class_id` | [string](#string) |  | nft classID to cnvert to ERC721 |
| `cosmos_token_ids` | [string](#string) | repeated | nftID to cnvert to ERC721 |






<a name="uptick.erc721.v1.MsgConvertERC721Response"></a>

### MsgConvertERC721Response
MsgConvertERC721Response returns no fields


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `evm_contract_address` | [string](#string) |  | ERC721 token contract address registered in a token pair |
| `evm_token_ids` | [string](#string) | repeated | tokenID to convert |
| `cosmos_receiver` | [string](#string) |  | bech32 address to receive native Cosmos coins |
| `evm_sender` | [string](#string) |  | sender hex address from the owner of the given ERC721 tokens |
| `class_id` | [string](#string) |  | nft classID to cnvert to ERC721 |
| `cosmos_token_ids` | [string](#string) | repeated | nftID to cnvert to ERC721 |






<a name="uptick.erc721.v1.MsgConvertNFT"></a>

### MsgConvertNFT
MsgConvertNFT defines a Msg to convert a native Cosmos nft to a ERC721 token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  | nft classID to cnvert to ERC721 |
| `cosmos_token_ids` | [string](#string) | repeated | nftID to cnvert to ERC721 |
| `evm_receiver` | [string](#string) |  | recipient hex address to receive ERC721 token |
| `cosmos_sender` | [string](#string) |  | cosmos bech32 address from the owner of the given Cosmos coins |
| `evm_contract_address` | [string](#string) |  | ERC721 token contract address registered in a token pair |
| `evm_token_ids` | [string](#string) | repeated | ERC721 token id registered in a token pair |






<a name="uptick.erc721.v1.MsgConvertNFTResponse"></a>

### MsgConvertNFTResponse
MsgConvertNFTResponse returns no fields






<a name="uptick.erc721.v1.MsgTransferERC721"></a>

### MsgTransferERC721
MsgTransferERC721 defines a message for transferring erc721 tokens through IBC


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `evm_contract_address` | [string](#string) |  | evm_contract_address is the ERC721 token contract address |
| `evm_token_ids` | [string](#string) | repeated | tokenID to convert |
| `source_port` | [string](#string) |  | the port on which the packet will be sent |
| `source_channel` | [string](#string) |  | the channel by which the packet will be sent |
| `class_id` | [string](#string) |  | the class_id of tokens to be transferred |
| `cosmos_token_ids` | [string](#string) | repeated | the non fungible tokens to be transferred |
| `cosmos_sender` | [string](#string) |  | the sender address |
| `cosmos_receiver` | [string](#string) |  | the recipient address on the destination chain |
| `timeout_height` | [ibc.core.client.v1.Height](#ibc.core.client.v1.Height) |  | timeout_height is the timeout height relative to the current block height The timeout is disabled when set to 0 |
| `timeout_timestamp` | [uint64](#uint64) |  | timeout_timestamp is the timeout timestamp in absolute nanoseconds since unix epoch The timeout is disabled when set to 0 |
| `memo` | [string](#string) |  | memo is an optional memo field |






<a name="uptick.erc721.v1.MsgTransferERC721Response"></a>

### MsgTransferERC721Response
MsgTransferERC721Response defines the response type for Transfer erc721 RPC






<a name="uptick.erc721.v1.MsgUpdateParams"></a>

### MsgUpdateParams
MsgUpdateParams is the Msg/UpdateParams request type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  | authority is the address of the governance account. |
| `params` | [Params](#uptick.erc721.v1.Params) |  | params defines the x/erc721 module parameters to update.

Since: v0.4.1 |






<a name="uptick.erc721.v1.MsgUpdateParamsResponse"></a>

### MsgUpdateParamsResponse
MsgUpdateParamsResponse is the Msg/UpdateParams response type.





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.erc721.v1.Msg"></a>

### Msg
Msg defines the erc721 Msg service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `ConvertNFT` | [MsgConvertNFT](#uptick.erc721.v1.MsgConvertNFT) | [MsgConvertNFTResponse](#uptick.erc721.v1.MsgConvertNFTResponse) | ConvertNFT mints a ERC721 representation of the native Cosmos nft that is registered on the token mapping. | GET|/uptick/erc721/v1/tx/convert_nft|
| `ConvertERC721` | [MsgConvertERC721](#uptick.erc721.v1.MsgConvertERC721) | [MsgConvertERC721Response](#uptick.erc721.v1.MsgConvertERC721Response) | ConvertERC721 mints a native Cosmos coin representation of the ERC721 token contract that is registered on the token mapping. | GET|/uptick/erc721/v1/tx/convert_erc721|
| `TransferERC721` | [MsgTransferERC721](#uptick.erc721.v1.MsgTransferERC721) | [MsgTransferERC721Response](#uptick.erc721.v1.MsgTransferERC721Response) | TransferERC721 transfers a erc721 token from one chain to another chain through IBC | GET|/uptick/erc721/v1/transfer_erc721|
| `UpdateParams` | [MsgUpdateParams](#uptick.erc721.v1.MsgUpdateParams) | [MsgUpdateParamsResponse](#uptick.erc721.v1.MsgUpdateParamsResponse) | UpdateParams updates the module parameters. Since the erc721 module has no authority account, this RPC is registered for interface completeness and is only callable by the governance authority. | |

 <!-- end services -->



<a name="uptick/evm_ibc/v1/evm_ibc.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/evm_ibc/v1/evm_ibc.proto



<a name="uptick.evm_ibc.v1.TokenPair"></a>

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



<a name="uptick/evm_ibc/v1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/evm_ibc/v1/query.proto



<a name="uptick.evm_ibc.v1.QueryEvmAddressRequest"></a>

### QueryEvmAddressRequest
QueryEvmAddressRequest is the request type for the Query/TokenPair RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `port` | [string](#string) |  | token identifier can be either the hex contract address of the ERC721 or the Cosmos nft classID |
| `channel` | [string](#string) |  |  |
| `class_id` | [string](#string) |  |  |






<a name="uptick.evm_ibc.v1.QueryTokenPairResponse"></a>

### QueryTokenPairResponse
QueryTokenPairResponse is the response type for the Query/TokenPair RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `token_pair` | [TokenPair](#uptick.evm_ibc.v1.TokenPair) |  |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.evm_ibc.v1.Query"></a>

### Query
Query defines the gRPC queried service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `EvmContract` | [QueryEvmAddressRequest](#uptick.evm_ibc.v1.QueryEvmAddressRequest) | [QueryTokenPairResponse](#uptick.evm_ibc.v1.QueryTokenPairResponse) | EvmContract retrieves a registered evm contract | GET|/uptick/evmIBC/v1/evm_contract/{port}/{channel}/{class_id}|

 <!-- end services -->



<a name="uptick/evm_ibc/v1/tx.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/evm_ibc/v1/tx.proto



<a name="uptick.evm_ibc.v1.MsgTransferERC721"></a>

### MsgTransferERC721
MsgTransferERC721 defines a message to transfer ERC721 tokens between chains via IBC
It contains information about the token contract, token IDs, source and destination of the transfer,
timeout parameters and optional memo


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `evm_contract_address` | [string](#string) |  |  |
| `evm_token_ids` | [string](#string) | repeated | tokenID to convert |
| `source_port` | [string](#string) |  | the port on which the packet will be sent |
| `source_channel` | [string](#string) |  | the channel by which the packet will be sent |
| `class_id` | [string](#string) |  | the class_id of tokens to be transferred |
| `cosmos_token_ids` | [string](#string) | repeated | the non fungible tokens to be transferred |
| `cosmos_sender` | [string](#string) |  | the sender address |
| `cosmos_receiver` | [string](#string) |  | the recipient address on the destination chain |
| `timeout_height` | [ibc.core.client.v1.Height](#ibc.core.client.v1.Height) |  | Timeout height relative to the current block height. The timeout is disabled when set to 0. |
| `timeout_timestamp` | [uint64](#uint64) |  | Timeout timestamp in absolute nanoseconds since unix epoch. The timeout is disabled when set to 0. |
| `memo` | [string](#string) |  | optional memo |






<a name="uptick.evm_ibc.v1.MsgTransferERC721Response"></a>

### MsgTransferERC721Response
MsgTransferERC721Response defines the response type for TransferERC721 RPC





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.evm_ibc.v1.Msg"></a>

### Msg
NOTE (legacy surface): x/evmibc no longer runs as a Cosmos module with its
own Msg/Query servers. Since the v0.4.0 IBC v10 rework it is wired purely as
an IBC middleware (app/keepers -> evmibc.NewIBCMiddleware) on top of
x/erc721 / x/cw721, whose own messages (MsgConvertNFT, MsgTransferERC721,
...) perform the transfers. The Msg service and messages below are therefore
NOT registered anywhere: no codec registers them, no msg server is wired and
this REST annotation is never served. They are retained only for ABI
documentation of the historical transfer surface. Do not rely on them for
new clients; removing them requires a proto regeneration.

Msg defines the (legacy, unregistered) evmibc Msg service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `TransferERC721` | [MsgTransferERC721](#uptick.evm_ibc.v1.MsgTransferERC721) | [MsgTransferERC721Response](#uptick.evm_ibc.v1.MsgTransferERC721Response) | TransferERC721 defines a method to transfer ERC721 tokens between chains via IBC | GET|/uptick/erc721/v1/tx/ibc-transfer-erc721|

 <!-- end services -->



<a name="uptick/nft/v1beta1/event.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/nft/v1beta1/event.proto



<a name="uptick.nft.v1beta1.EventBurn"></a>

### EventBurn
EventBurn is emitted on Burn


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |
| `id` | [string](#string) |  |  |
| `owner` | [string](#string) |  |  |






<a name="uptick.nft.v1beta1.EventMint"></a>

### EventMint
EventMint is emitted on Mint


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |
| `id` | [string](#string) |  |  |
| `owner` | [string](#string) |  |  |






<a name="uptick.nft.v1beta1.EventSend"></a>

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



<a name="uptick.nft.v1beta1.Class"></a>

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






<a name="uptick.nft.v1beta1.NFT"></a>

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



<a name="uptick.nft.v1beta1.Entry"></a>

### Entry
Entry Defines all nft owned by a person


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `owner` | [string](#string) |  | owner is the owner address of the following nft |
| `nfts` | [NFT](#uptick.nft.v1beta1.NFT) | repeated | nfts is a group of nfts of the same owner |






<a name="uptick.nft.v1beta1.GenesisState"></a>

### GenesisState
GenesisState defines the nft module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `classes` | [Class](#uptick.nft.v1beta1.Class) | repeated | class defines the class of the nft type. |
| `entries` | [Entry](#uptick.nft.v1beta1.Entry) | repeated |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="uptick/nft/v1beta1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/nft/v1beta1/query.proto



<a name="uptick.nft.v1beta1.QueryBalanceRequest"></a>

### QueryBalanceRequest
QueryBalanceRequest is the request type for the Query/Balance RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |
| `owner` | [string](#string) |  |  |






<a name="uptick.nft.v1beta1.QueryBalanceResponse"></a>

### QueryBalanceResponse
QueryBalanceResponse is the response type for the Query/Balance RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `amount` | [uint64](#uint64) |  |  |






<a name="uptick.nft.v1beta1.QueryClassRequest"></a>

### QueryClassRequest
QueryClassRequest is the request type for the Query/Class RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |






<a name="uptick.nft.v1beta1.QueryClassResponse"></a>

### QueryClassResponse
QueryClassResponse is the response type for the Query/Class RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class` | [Class](#uptick.nft.v1beta1.Class) |  |  |






<a name="uptick.nft.v1beta1.QueryClassesRequest"></a>

### QueryClassesRequest
QueryClassesRequest is the request type for the Query/Classes RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  | pagination defines an optional pagination for the request. |






<a name="uptick.nft.v1beta1.QueryClassesResponse"></a>

### QueryClassesResponse
QueryClassesResponse is the response type for the Query/Classes RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `classes` | [Class](#uptick.nft.v1beta1.Class) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  |  |






<a name="uptick.nft.v1beta1.QueryNFTRequest"></a>

### QueryNFTRequest
QueryNFTRequest is the request type for the Query/NFT RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |
| `id` | [string](#string) |  |  |






<a name="uptick.nft.v1beta1.QueryNFTResponse"></a>

### QueryNFTResponse
QueryNFTResponse is the response type for the Query/NFT RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `nft` | [NFT](#uptick.nft.v1beta1.NFT) |  |  |






<a name="uptick.nft.v1beta1.QueryNFTsRequest"></a>

### QueryNFTsRequest
QueryNFTsRequest is the request type for the Query/NFTs RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |
| `owner` | [string](#string) |  |  |
| `pagination` | [cosmos.base.query.v1beta1.PageRequest](#cosmos.base.query.v1beta1.PageRequest) |  |  |






<a name="uptick.nft.v1beta1.QueryNFTsResponse"></a>

### QueryNFTsResponse
QueryNFTsResponse is the response type for the Query/NFTs RPC methods


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `nfts` | [NFT](#uptick.nft.v1beta1.NFT) | repeated |  |
| `pagination` | [cosmos.base.query.v1beta1.PageResponse](#cosmos.base.query.v1beta1.PageResponse) |  |  |






<a name="uptick.nft.v1beta1.QueryOwnerRequest"></a>

### QueryOwnerRequest
QueryOwnerRequest is the request type for the Query/Owner RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |
| `id` | [string](#string) |  |  |






<a name="uptick.nft.v1beta1.QueryOwnerResponse"></a>

### QueryOwnerResponse
QueryOwnerResponse is the response type for the Query/Owner RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `owner` | [string](#string) |  |  |






<a name="uptick.nft.v1beta1.QuerySupplyRequest"></a>

### QuerySupplyRequest
QuerySupplyRequest is the request type for the Query/Supply RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  |  |






<a name="uptick.nft.v1beta1.QuerySupplyResponse"></a>

### QuerySupplyResponse
QuerySupplyResponse is the response type for the Query/Supply RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `amount` | [uint64](#uint64) |  |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.nft.v1beta1.Query"></a>

### Query
Query defines the gRPC querier service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `Balance` | [QueryBalanceRequest](#uptick.nft.v1beta1.QueryBalanceRequest) | [QueryBalanceResponse](#uptick.nft.v1beta1.QueryBalanceResponse) | Balance queries the number of NFTs of a given class owned by the owner, same as balanceOf in ERC721 | GET|/cosmos/nft/v1beta1/balance/{owner}/{class_id}|
| `Owner` | [QueryOwnerRequest](#uptick.nft.v1beta1.QueryOwnerRequest) | [QueryOwnerResponse](#uptick.nft.v1beta1.QueryOwnerResponse) | Owner queries the owner of the NFT based on its class and id, same as ownerOf in ERC721 | GET|/cosmos/nft/v1beta1/owner/{class_id}/{id}|
| `Supply` | [QuerySupplyRequest](#uptick.nft.v1beta1.QuerySupplyRequest) | [QuerySupplyResponse](#uptick.nft.v1beta1.QuerySupplyResponse) | Supply queries the number of NFTs from the given class, same as totalSupply of ERC721. | GET|/cosmos/nft/v1beta1/supply/{class_id}|
| `NFTs` | [QueryNFTsRequest](#uptick.nft.v1beta1.QueryNFTsRequest) | [QueryNFTsResponse](#uptick.nft.v1beta1.QueryNFTsResponse) | NFTs queries all NFTs of a given class or owner,choose at least one of the two, similar to tokenByIndex in ERC721Enumerable | GET|/cosmos/nft/v1beta1/nfts|
| `NFT` | [QueryNFTRequest](#uptick.nft.v1beta1.QueryNFTRequest) | [QueryNFTResponse](#uptick.nft.v1beta1.QueryNFTResponse) | NFT queries an NFT based on its class and id. | GET|/cosmos/nft/v1beta1/nfts/{class_id}/{id}|
| `Class` | [QueryClassRequest](#uptick.nft.v1beta1.QueryClassRequest) | [QueryClassResponse](#uptick.nft.v1beta1.QueryClassResponse) | Class queries an NFT class based on its id | GET|/cosmos/nft/v1beta1/classes/{class_id}|
| `Classes` | [QueryClassesRequest](#uptick.nft.v1beta1.QueryClassesRequest) | [QueryClassesResponse](#uptick.nft.v1beta1.QueryClassesResponse) | Classes queries all NFT classes | GET|/cosmos/nft/v1beta1/classes|

 <!-- end services -->



<a name="uptick/nft/v1beta1/tx.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## uptick/nft/v1beta1/tx.proto



<a name="uptick.nft.v1beta1.MsgSend"></a>

### MsgSend
MsgSend represents a message to send a nft from one account to another
account.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `class_id` | [string](#string) |  | class_id defines the unique identifier of the nft classification, similar to the contract address of ERC721 |
| `id` | [string](#string) |  | id defines the unique identification of nft |
| `sender` | [string](#string) |  | sender is the address of the owner of nft |
| `receiver` | [string](#string) |  | receiver is the receiver address of nft |






<a name="uptick.nft.v1beta1.MsgSendResponse"></a>

### MsgSendResponse
MsgSendResponse defines the Msg/Send response type.





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="uptick.nft.v1beta1.Msg"></a>

### Msg
Msg defines the nft Msg service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `Send` | [MsgSend](#uptick.nft.v1beta1.MsgSend) | [MsgSendResponse](#uptick.nft.v1beta1.MsgSendResponse) | Send defines a method to send a nft from one account to another account. | |

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

