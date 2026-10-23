# Protocol Documentation

# uptick/appmining/v1/types.proto




## Application
Application is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | bytes |  |  |
| owner | string |  |  |
| attestation_submitter | string |  |  |
| gas_payers | string | repeated |  |
| status | ApplicationStatus |  |  |
| metadata_commitment | bytes |  | opaque 32-byte metadata commitment (empty allowed until set) |




## AttestationCommitment
AttestationCommitment is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | bytes |  |  |
| epoch | uint64 |  |  |
| profile | string |  |  |
| profile_version | string |  |  |
| attestation_hash | bytes |  |  |
| evidence_root | bytes |  |  |
| result_commitment | bytes |  |  |
| reward_requested | string |  |  |
| gas_credit_requested | string |  |  |
| submitter | string |  |  |
| submitted_height | int64 |  |  |
| challenge_deadline_height | int64 |  |  |
| status | AttestationStatus |  |  |




## AttestationProfile
AttestationProfile is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | string |  |  |
| version | string |  |  |
| active | bool |  |  |




## Candidate
Candidate is a published entitlement candidate (no pool liability until Approve).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entitlement_id | string |  | 64 hex |
| app_id | string |  | 64 hex (wire); registry lookup uses raw 32 bytes |
| epoch | string |  |  |
| amount | string |  |  |
| denom | string |  |  |
| gas_used | string |  |  |
| qualified_tx_count | string |  |  |
| pav_id | string |  |  |
| evidence_root | string |  |  |
| gas_evidence_root | string |  |  |
| rule_version | string |  |  |
| candidate_hash | string |  |  |
| pool_id | string |  |  |
| publish_tx_hash | string |  |  |
| publish_height | int64 |  |  |
| published_at | int64 |  | unix seconds (block time) |
| challenge_deadline | int64 |  | published_at + 604800 |
| publisher | string |  | cosmos bech32 submitter |
| status | CandidateStatus |  |  |
| max_observed_time | int64 |  | anti-rollback watermark |




## Challenge
Challenge is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| challenge_id | uint64 |  |  |
| app_id | bytes |  |  |
| epoch | uint64 |  |  |
| challenger | string |  |  |
| evidence_hash | bytes |  |  |
| bond_denom | string |  |  |
| bond_amount | string |  |  |
| status | ChallengeStatus |  |  |
| resolution_hash | bytes |  |  |
| resolved_height | int64 |  |  |




## Entitlement
Entitlement is created only after Approve (pool liability reserved).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entitlement_id | string |  |  |
| app_id | string |  |  |
| owner_at_last_query | string |  |  |
| amount | string |  |  |
| denom | string |  |  |
| state | EntitlementState |  |  |
| freeze_until | int64 |  |  |
| freeze_count | uint32 |  |  |
| approval_tx | string |  |  |
| claim_tx | string |  |  |
| evidence_root | string |  |  |
| rule_version | string |  |  |
| pool_id | string |  |  |
| approved_at | int64 |  |  |
| max_observed_time | int64 |  |  |




## EntitlementChallenge
EntitlementChallenge is the V1 challenge record (distinct from attestation Challenge).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| challenge_id | string |  | 64 hex |
| candidate_id | string |  | entitlement_id |
| challenger | string |  |  |
| evidence_type | EvidenceType |  |  |
| evidence_digest | string |  | 64 hex |
| reference_digest | string |  | 64 hex; may be empty -> store as 64 zero hex if unset at open |
| status | EntitlementChallengeStatus |  |  |
| opened_at | int64 |  |  |
| reviewer | string |  | key_id of resolving reviewer |
| resolution_digest | string |  | 64 hex |
| resolved_at | int64 |  |  |




## EpochBudget
EpochBudget is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| epoch | uint64 |  |  |
| denom | string |  |  |
| limit | string |  |  |
| spent | string |  |  |
| closed | bool |  |  |
| opened_height | int64 |  |  |
| closed_height | int64 |  |  |




## GasCreditAccount
GasCreditAccount is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | bytes |  |  |
| balance | string |  |  |
| total_issued | string |  |  |
| total_consumed | string |  |  |




## GasCreditConsumption
GasCreditConsumption is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | bytes |  |  |
| gas_payer | string |  |  |
| source_tx_hash | bytes |  |  |
| amount | string |  |  |
| consumed_height | int64 |  |  |




## Params
Params is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | string |  |  |
| challenge_window_blocks | uint64 |  |  |
| challenge_bond_denom | string |  |  |
| challenge_bond_amount | string |  |  |
| gas_credit_share_bps | uint32 |  |  |
| pause_state | PauseState |  |  |
| allowed_profiles | AttestationProfile | repeated |  |
| pause_authority | string |  | hot path: may only set pause bits to true; resume is authority-only (Q-06) |
| reward_denom_allowlist | string | repeated | governance-approved reward denoms (Q-08) |
| max_gas_payers | uint32 |  | max gas payers per application (0 = use module default 16) |
| allow_manual_gas_credit_consume | bool |  | when false (default), MsgConsumeGasCredit is rejected; PostHandler auto-debit remains |
| entitlement_enable_min_funded | string |  | V1: minimum Pool.funded (smallest units, unsigned decimal string) required before
entitlement msgs are enabled. Empty or "0" keeps the new flow disabled. |
| entitlement_allowed_rule_versions | string | repeated | V1: allowlist of rule_version values accepted by MsgApproveEntitlement (§3
"规则版本允许"). Empty list denies every approval (default-deny). |




## PauseState
PauseState is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| registration_paused | bool |  |  |
| attestation_paused | bool |  |  |
| challenge_paused | bool |  |  |
| settlement_paused | bool |  |  |
| gas_credit_paused | bool |  |  |
| entitlement_paused | bool |  | V1 entitlement flow (Publish/Approve/Claim/…); independent of attestation pause bits |




## Pool
Pool is the fixed entitlement fund ledger.
Invariant: funded = available + reserved + paid; reserved <= module account balance.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pool_id | string |  |  |
| denom | string |  |  |
| funded | string |  |  |
| available | string |  |  |
| reserved | string |  |  |
| paid | string |  |  |
| paused | bool |  |  |
| version | uint64 |  |  |




## RoleKey
RoleKey is a versioned Ed25519 public key for one UGE role.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | EntitlementRole |  |  |
| key_id | string |  |  |
| pubkey | bytes |  | raw Ed25519 public key, exactly 32 bytes |
| status | RoleKeyStatus |  |  |
| version | uint64 |  |  |
| registered_height | int64 |  |  |




## Settlement
Settlement is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | bytes |  |  |
| epoch | uint64 |  |  |
| profile | string |  |  |
| profile_version | string |  |  |
| attestation_hash | bytes |  |  |
| reward_released | string |  |  |
| gas_credit_issued | string |  |  |
| finalized_height | int64 |  |  |




## WireProof
WireProof is the Ed25519 domain-separated authorization attached to V1 msgs.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key_id | string |  |  |
| nonce | string |  |  |
| expires_at | string |  | unix seconds, unsigned decimal |
| signature_base64 | string |  |  |






## ApplicationStatus
ApplicationStatus enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| APPLICATION_STATUS_UNSPECIFIED | 0 | APPLICATION_STATUS_UNSPECIFIED is a valid enum value. |
| APPLICATION_STATUS_ACTIVE | 1 | APPLICATION_STATUS_ACTIVE is a valid enum value. |
| APPLICATION_STATUS_SUSPENDED | 2 | APPLICATION_STATUS_SUSPENDED is a valid enum value. |
| APPLICATION_STATUS_REVOKED | 3 | APPLICATION_STATUS_REVOKED is a valid enum value. |



## AttestationStatus
AttestationStatus enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ATTESTATION_STATUS_UNSPECIFIED | 0 | ATTESTATION_STATUS_UNSPECIFIED is a valid enum value. |
| ATTESTATION_STATUS_CHALLENGE_WINDOW | 1 | ATTESTATION_STATUS_CHALLENGE_WINDOW is a valid enum value. |
| ATTESTATION_STATUS_CHALLENGED | 2 | ATTESTATION_STATUS_CHALLENGED is a valid enum value. |
| ATTESTATION_STATUS_FINALIZED | 3 | ATTESTATION_STATUS_FINALIZED is a valid enum value. |
| ATTESTATION_STATUS_REJECTED | 4 | ATTESTATION_STATUS_REJECTED is a valid enum value. |



## CandidateStatus
CandidateStatus enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| CANDIDATE_STATUS_UNSPECIFIED | 0 | CANDIDATE_STATUS_UNSPECIFIED is a valid enum value. |
| CANDIDATE_STATUS_PUBLISHED | 1 | CANDIDATE_STATUS_PUBLISHED is a valid enum value. |
| CANDIDATE_STATUS_REJECTED | 2 | CANDIDATE_STATUS_REJECTED is a valid enum value. |



## ChallengeStatus
ChallengeStatus enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| CHALLENGE_STATUS_UNSPECIFIED | 0 | CHALLENGE_STATUS_UNSPECIFIED is a valid enum value. |
| CHALLENGE_STATUS_OPEN | 1 | CHALLENGE_STATUS_OPEN is a valid enum value. |
| CHALLENGE_STATUS_REJECTED | 2 | CHALLENGE_STATUS_REJECTED is a valid enum value. |
| CHALLENGE_STATUS_UPHELD | 3 | CHALLENGE_STATUS_UPHELD is a valid enum value. |
| CHALLENGE_STATUS_SUPERSEDED | 4 | closed without ruling because another upheld challenge terminated the
attestation; bond fully refunded (CLARIFICATIONS-REPLY Q-04) |



## EntitlementChallengeStatus
EntitlementChallengeStatus enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENTITLEMENT_CHALLENGE_STATUS_UNSPECIFIED | 0 | ENTITLEMENT_CHALLENGE_STATUS_UNSPECIFIED is a valid enum value. |
| ENTITLEMENT_CHALLENGE_STATUS_OPEN | 1 | ENTITLEMENT_CHALLENGE_STATUS_OPEN is a valid enum value. |
| ENTITLEMENT_CHALLENGE_STATUS_UPHELD | 2 | ENTITLEMENT_CHALLENGE_STATUS_UPHELD is a valid enum value. |
| ENTITLEMENT_CHALLENGE_STATUS_DISMISSED | 3 | ENTITLEMENT_CHALLENGE_STATUS_DISMISSED is a valid enum value. |



## EntitlementRole
EntitlementRole is the five UGE authorization roles (wire role strings).

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENTITLEMENT_ROLE_UNSPECIFIED | 0 | ENTITLEMENT_ROLE_UNSPECIFIED is a valid enum value. |
| ENTITLEMENT_ROLE_PUBLISHER | 1 | ENTITLEMENT_ROLE_PUBLISHER is a valid enum value. |
| ENTITLEMENT_ROLE_APPROVER | 2 | ENTITLEMENT_ROLE_APPROVER is a valid enum value. |
| ENTITLEMENT_ROLE_REVIEWER | 3 | ENTITLEMENT_ROLE_REVIEWER is a valid enum value. |
| ENTITLEMENT_ROLE_RISK | 4 | ENTITLEMENT_ROLE_RISK is a valid enum value. |
| ENTITLEMENT_ROLE_AUDIT | 5 | ENTITLEMENT_ROLE_AUDIT is a valid enum value. |



## EntitlementState
EntitlementState enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENTITLEMENT_STATE_UNSPECIFIED | 0 | ENTITLEMENT_STATE_UNSPECIFIED is a valid enum value. |
| ENTITLEMENT_STATE_CLAIMABLE | 1 | ENTITLEMENT_STATE_CLAIMABLE is a valid enum value. |
| ENTITLEMENT_STATE_FROZEN | 2 | ENTITLEMENT_STATE_FROZEN is a valid enum value. |
| ENTITLEMENT_STATE_REVOKED | 3 | ENTITLEMENT_STATE_REVOKED is a valid enum value. |
| ENTITLEMENT_STATE_CLAIMED | 4 | ENTITLEMENT_STATE_CLAIMED is a valid enum value. |



## EvidenceType
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



## RoleKeyStatus
RoleKeyStatus enumerates allowed values for the module.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ROLE_KEY_STATUS_UNSPECIFIED | 0 | ROLE_KEY_STATUS_UNSPECIFIED is a valid enum value. |
| ROLE_KEY_STATUS_ACTIVE | 1 | ROLE_KEY_STATUS_ACTIVE is a valid enum value. |
| ROLE_KEY_STATUS_PAUSED | 2 | ROLE_KEY_STATUS_PAUSED is a valid enum value. |
| ROLE_KEY_STATUS_REVOKED | 3 | ROLE_KEY_STATUS_REVOKED is a valid enum value. |









# uptick/appmining/v1/genesis.proto




## GenesisState
GenesisState defines the appmining module genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| params | Params |  |  |
| applications | Application | repeated |  |
| epoch_budgets | EpochBudget | repeated |  |
| attestations | AttestationCommitment | repeated |  |
| challenges | Challenge | repeated |  |
| settlements | Settlement | repeated |  |
| gas_credit_accounts | GasCreditAccount | repeated |  |
| gas_credit_consumptions | GasCreditConsumption | repeated |  |
| next_challenge_id | uint64 |  |  |
| entitlement_pools | Pool | repeated | V1 entitlement state (UGE requirements 1.0) |
| role_keys | RoleKey | repeated |  |
| candidates | Candidate | repeated |  |
| entitlement_challenges | EntitlementChallenge | repeated |  |
| used_auth_nonces | string | repeated | "role/key_id/nonce" consumed proofs |
| entitlements | Entitlement | repeated |  |












# uptick/appmining/v1/query.proto




## QueryApplicationRequest
QueryApplicationRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | bytes |  |  |




## QueryApplicationResponse
QueryApplicationResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| application | Application |  |  |




## QueryApplicationsRequest
QueryApplicationsRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | ApplicationStatus |  | optional status filter; UNSPECIFIED = all |
| pagination | cosmos.base.query.v1beta1.PageRequest |  |  |




## QueryApplicationsResponse
QueryApplicationsResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| applications | Application | repeated |  |
| pagination | cosmos.base.query.v1beta1.PageResponse |  |  |




## QueryAttestationRequest
QueryAttestationRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | bytes |  |  |
| epoch | uint64 |  |  |




## QueryAttestationResponse
QueryAttestationResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| attestation | AttestationCommitment |  |  |




## QueryAttestationsRequest
QueryAttestationsRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | AttestationStatus |  | optional status filter; UNSPECIFIED = all |
| pagination | cosmos.base.query.v1beta1.PageRequest |  |  |




## QueryAttestationsResponse
QueryAttestationsResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| attestations | AttestationCommitment | repeated |  |
| pagination | cosmos.base.query.v1beta1.PageResponse |  |  |




## QueryCandidateRequest
QueryCandidateRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entitlement_id | string |  |  |




## QueryCandidateResponse
QueryCandidateResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| candidate | Candidate |  |  |




## QueryChallengeRequest
QueryChallengeRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| challenge_id | uint64 |  |  |




## QueryChallengeResponse
QueryChallengeResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| challenge | Challenge |  |  |




## QueryEntitlementChallengeRequest
QueryEntitlementChallengeRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| challenge_id | string |  |  |




## QueryEntitlementChallengeResponse
QueryEntitlementChallengeResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| challenge | EntitlementChallenge |  |  |




## QueryEntitlementRequest
QueryEntitlementRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entitlement_id | string |  |  |




## QueryEntitlementResponse
QueryEntitlementResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entitlement | Entitlement |  |  |




## QueryEntitlementsByAppRequest
QueryEntitlementsByAppRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | string |  | 64 hex |
| pagination | cosmos.base.query.v1beta1.PageRequest |  |  |




## QueryEntitlementsByAppResponse
QueryEntitlementsByAppResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entitlements | Entitlement | repeated |  |
| pagination | cosmos.base.query.v1beta1.PageResponse |  |  |




## QueryEpochBudgetRequest
QueryEpochBudgetRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| epoch | uint64 |  |  |




## QueryEpochBudgetResponse
QueryEpochBudgetResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| budget | EpochBudget |  |  |




## QueryGasCreditConsumptionsRequest
QueryGasCreditConsumptionsRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | bytes |  |  |
| pagination | cosmos.base.query.v1beta1.PageRequest |  |  |




## QueryGasCreditConsumptionsResponse
QueryGasCreditConsumptionsResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| consumptions | GasCreditConsumption | repeated |  |
| pagination | cosmos.base.query.v1beta1.PageResponse |  |  |




## QueryGasCreditRequest
QueryGasCreditRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | bytes |  |  |




## QueryGasCreditResponse
QueryGasCreditResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| account | GasCreditAccount |  |  |




## QueryOpenChallengesRequest
QueryOpenChallengesRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | bytes |  | optional; empty = all apps |
| epoch | uint64 |  | optional; 0 = all epochs when app_id set still scans open index |
| pagination | cosmos.base.query.v1beta1.PageRequest |  |  |




## QueryOpenChallengesResponse
QueryOpenChallengesResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| challenges | Challenge | repeated |  |
| pagination | cosmos.base.query.v1beta1.PageResponse |  |  |




## QueryParamsRequest
QueryParamsRequest is the request/response type for the module API.




## QueryParamsResponse
QueryParamsResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| params | Params |  |  |




## QueryPoolRequest
QueryPoolRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pool_id | string |  |  |




## QueryPoolResponse
QueryPoolResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pool | Pool |  |  |
| module_balance | string |  | module account balance for pool.denom (for off-chain reconciliation) |
| entitlement_flow_enabled | bool |  | true when pool is open, not paused, and funded >= entitlement_enable_min_funded |




## QueryRolesRequest
QueryRolesRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | EntitlementRole |  | optional filter; UNSPECIFIED = all roles |
| status | RoleKeyStatus |  | optional filter; UNSPECIFIED = all statuses |




## QueryRolesResponse
QueryRolesResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| keys | RoleKey | repeated |  |




## QuerySettlementRequest
QuerySettlementRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | bytes |  |  |
| epoch | uint64 |  |  |




## QuerySettlementResponse
QuerySettlementResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| settlement | Settlement |  |  |




## QuerySettlementsRequest
QuerySettlementsRequest is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pagination | cosmos.base.query.v1beta1.PageRequest |  |  |




## QuerySettlementsResponse
QuerySettlementsResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| settlements | Settlement | repeated |  |
| pagination | cosmos.base.query.v1beta1.PageResponse |  |  |










## Query
Query defines the gRPC query/tx service for x/appmining.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Params | QueryParamsRequest | QueryParamsResponse | Params handles the corresponding module request. |
| Application | QueryApplicationRequest | QueryApplicationResponse | Application handles the corresponding module request. |
| Applications | QueryApplicationsRequest | QueryApplicationsResponse | Applications handles the corresponding module request. |
| EpochBudget | QueryEpochBudgetRequest | QueryEpochBudgetResponse | EpochBudget handles the corresponding module request. |
| Attestation | QueryAttestationRequest | QueryAttestationResponse | Attestation handles the corresponding module request. |
| Attestations | QueryAttestationsRequest | QueryAttestationsResponse | Attestations handles the corresponding module request. |
| Challenge | QueryChallengeRequest | QueryChallengeResponse | Challenge handles the corresponding module request. |
| OpenChallenges | QueryOpenChallengesRequest | QueryOpenChallengesResponse | OpenChallenges handles the corresponding module request. |
| Settlement | QuerySettlementRequest | QuerySettlementResponse | Settlement handles the corresponding module request. |
| Settlements | QuerySettlementsRequest | QuerySettlementsResponse | Settlements handles the corresponding module request. |
| GasCredit | QueryGasCreditRequest | QueryGasCreditResponse | GasCredit handles the corresponding module request. |
| GasCreditConsumptions | QueryGasCreditConsumptionsRequest | QueryGasCreditConsumptionsResponse | GasCreditConsumptions handles the corresponding module request. |
| Pool | QueryPoolRequest | QueryPoolResponse | V1 entitlement queries (UGE requirements 1.0) |
| Roles | QueryRolesRequest | QueryRolesResponse | Roles handles the corresponding module request. |
| Candidate | QueryCandidateRequest | QueryCandidateResponse | Candidate handles the corresponding module request. |
| EntitlementChallenge | QueryEntitlementChallengeRequest | QueryEntitlementChallengeResponse | EntitlementChallenge handles the corresponding module request. |
| Entitlement | QueryEntitlementRequest | QueryEntitlementResponse | Entitlement handles the corresponding module request. |
| EntitlementsByApp | QueryEntitlementsByAppRequest | QueryEntitlementsByAppResponse | EntitlementsByApp handles the corresponding module request. |




# uptick/appmining/v1/tx.proto




## MsgApproveEntitlement
MsgApproveEntitlement locks pool available→reserved and creates CLAIMABLE entitlement.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| submitter | string |  |  |
| chain_id | string |  |  |
| pool_id | string |  |  |
| entitlement_id | string |  |  |
| challenge_resolution_hash | string |  | challenge_resolution_hash binds approval to dispute state (wire APPROVE field). |
| proof | WireProof |  |  |




## MsgApproveEntitlementResponse
MsgApproveEntitlementResponse is the request/response type for the module API.




## MsgChallengeAttestation
MsgChallengeAttestation is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| challenger | string |  |  |
| app_id | bytes |  |  |
| epoch | uint64 |  |  |
| evidence_hash | bytes |  |  |




## MsgChallengeAttestationResponse
MsgChallengeAttestationResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| challenge_id | uint64 |  |  |




## MsgClaimEntitlement
MsgClaimEntitlement pays reserved funds to the current app owner (beneficiary = owner).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | string |  |  |
| entitlement_id | string |  |  |




## MsgClaimEntitlementResponse
MsgClaimEntitlementResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| amount | string |  |  |
| denom | string |  |  |




## MsgConsumeGasCredit
MsgConsumeGasCredit is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| gas_payer | string |  |  |
| app_id | bytes |  |  |
| amount | string |  |  |
| source_tx_hash | bytes |  |  |




## MsgConsumeGasCreditResponse
MsgConsumeGasCreditResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| remaining_balance | string |  |  |




## MsgFinalizeAttestation
MsgFinalizeAttestation is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| caller | string |  |  |
| app_id | bytes |  |  |
| epoch | uint64 |  |  |




## MsgFinalizeAttestationResponse
MsgFinalizeAttestationResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| settlement | Settlement |  |  |




## MsgFreezeEntitlement
MsgFreezeEntitlement temporarily blocks claim (max 72h, once per entitlement).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| submitter | string |  |  |
| chain_id | string |  |  |
| pool_id | string |  |  |
| entitlement_id | string |  |  |
| evidence_type | EvidenceType |  |  |
| evidence_digest | string |  |  |
| freeze_until | string |  | unix seconds decimal |
| proof | WireProof |  |  |




## MsgFreezeEntitlementResponse
MsgFreezeEntitlementResponse is the request/response type for the module API.




## MsgFundEntitlementPool
MsgFundEntitlementPool deposits coins into appmining_entitlement_pool and
increases Pool.funded and Pool.available by the same amount (no mint).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| funder | string |  |  |
| pool_id | string |  |  |
| amount | string |  |  |




## MsgFundEntitlementPoolResponse
MsgFundEntitlementPoolResponse is the request/response type for the module API.




## MsgFundRewardPool
MsgFundRewardPool deposits coins into appmining_reward_pool.
Any allowlisted-denom holder may fund (funder ≠ governance authority on purpose:
gov module accounts cannot practically sign/hold spendable balances).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| funder | string |  |  |
| denom | string |  |  |
| amount | string |  |  |




## MsgFundRewardPoolResponse
MsgFundRewardPoolResponse is the request/response type for the module API.




## MsgGrantGasPayer
MsgGrantGasPayer is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | string |  |  |
| app_id | bytes |  |  |
| gas_payer | string |  |  |




## MsgGrantGasPayerResponse
MsgGrantGasPayerResponse is the request/response type for the module API.




## MsgOpenChallenge
MsgOpenChallenge opens a public challenge against a published V1 candidate.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| challenger | string |  |  |
| entitlement_id | string |  |  |
| evidence_type | EvidenceType |  |  |
| evidence_digest | string |  |  |
| reference_digest | string |  | optional; empty allowed -> normalized to 64 zero hex |




## MsgOpenChallengeResponse
MsgOpenChallengeResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| challenge_id | string |  |  |




## MsgOpenEntitlementPool
MsgOpenEntitlementPool creates a V1 fixed fund pool ledger (funded=0).
Does not mint; activation requires FundEntitlementPool to reach enable min.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | string |  |  |
| pool_id | string |  |  |
| denom | string |  |  |
| version | uint64 |  |  |




## MsgOpenEntitlementPoolResponse
MsgOpenEntitlementPoolResponse is the request/response type for the module API.




## MsgPause
MsgPause applies only true bits from pause_bits onto current PauseState (OR).
Any attempt to clear a bit via this message is rejected (Q-06 PAUSE_ONLY_TRUE).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pause_authority | string |  |  |
| pause_bits | PauseState |  |  |




## MsgPauseResponse
MsgPauseResponse is the request/response type for the module API.




## MsgPublishCandidate
MsgPublishCandidate publishes an immutable candidate commitment (no lock).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| submitter | string |  |  |
| chain_id | string |  |  |
| pool_id | string |  |  |
| entitlement_id | string |  | Candidate body (schema fields); status/timestamps filled by keeper. |
| app_id | string |  |  |
| epoch | string |  |  |
| amount | string |  |  |
| denom | string |  |  |
| gas_used | string |  |  |
| qualified_tx_count | string |  |  |
| pav_id | string |  |  |
| evidence_root | string |  |  |
| gas_evidence_root | string |  |  |
| rule_version | string |  |  |
| proof | WireProof |  |  |




## MsgPublishCandidateResponse
MsgPublishCandidateResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| candidate_hash | string |  |  |
| challenge_deadline | int64 |  |  |




## MsgRegisterApplication
MsgRegisterApplication is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | string |  |  |
| app_id | bytes |  |  |
| owner | string |  |  |
| attestation_submitter | string |  |  |
| metadata_commitment | bytes |  | optional; empty or exactly 32 bytes |




## MsgRegisterApplicationResponse
MsgRegisterApplicationResponse is the request/response type for the module API.




## MsgRegisterRoleKey
MsgRegisterRoleKey registers or rotates an Ed25519 role public key.
Same (role, key_id) with identical pubkey is idempotent; conflicting pubkey rejected.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | string |  |  |
| role | EntitlementRole |  |  |
| key_id | string |  |  |
| pubkey | bytes |  | raw Ed25519, 32 bytes |
| version | uint64 |  |  |




## MsgRegisterRoleKeyResponse
MsgRegisterRoleKeyResponse is the request/response type for the module API.




## MsgResolveChallenge
MsgResolveChallenge is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | string |  |  |
| challenge_id | uint64 |  |  |
| uphold | bool |  |  |
| resolution_hash | bytes |  |  |




## MsgResolveChallengeResponse
MsgResolveChallengeResponse is the request/response type for the module API.




## MsgResolveEntitlementChallenge
MsgResolveEntitlementChallenge resolves an OPEN entitlement challenge.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| submitter | string |  |  |
| chain_id | string |  |  |
| pool_id | string |  |  |
| challenge_id | string |  |  |
| outcome | string |  | UPHELD | DISMISSED |
| rationale_digest | string |  |  |
| proof | WireProof |  |  |




## MsgResolveEntitlementChallengeResponse
MsgResolveEntitlementChallengeResponse is the request/response type for the module API.




## MsgRevokeEntitlement
MsgRevokeEntitlement requires independent risk + audit wire proofs while frozen.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| submitter | string |  |  |
| chain_id | string |  |  |
| pool_id | string |  |  |
| entitlement_id | string |  |  |
| evidence_type | EvidenceType |  |  |
| evidence_digest | string |  |  |
| rationale_digest | string |  |  |
| risk_proof | WireProof |  |  |
| audit_proof | WireProof |  |  |




## MsgRevokeEntitlementResponse
MsgRevokeEntitlementResponse is the request/response type for the module API.




## MsgRevokeGasPayer
MsgRevokeGasPayer is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | string |  |  |
| app_id | bytes |  |  |
| gas_payer | string |  |  |




## MsgRevokeGasPayerResponse
MsgRevokeGasPayerResponse is the request/response type for the module API.




## MsgSetApplicationStatus
MsgSetApplicationStatus is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | string |  |  |
| app_id | bytes |  |  |
| status | ApplicationStatus |  |  |




## MsgSetApplicationStatusResponse
MsgSetApplicationStatusResponse is the request/response type for the module API.




## MsgSetAttestationSubmitter
MsgSetAttestationSubmitter is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | string |  |  |
| app_id | bytes |  |  |
| attestation_submitter | string |  |  |




## MsgSetAttestationSubmitterResponse
MsgSetAttestationSubmitterResponse is the request/response type for the module API.




## MsgSetEntitlementPoolPaused
MsgSetEntitlementPoolPaused sets Pool.paused (authority only).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | string |  |  |
| pool_id | string |  |  |
| paused | bool |  |  |




## MsgSetEntitlementPoolPausedResponse
MsgSetEntitlementPoolPausedResponse is the request/response type for the module API.




## MsgSetEpochBudget
MsgSetEpochBudget is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | string |  |  |
| budget | EpochBudget |  |  |




## MsgSetEpochBudgetResponse
MsgSetEpochBudgetResponse is the request/response type for the module API.




## MsgSetRoleKeyStatus
MsgSetRoleKeyStatus pauses or revokes a role key (ACTIVE keys only authorize new msgs).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | string |  |  |
| role | EntitlementRole |  |  |
| key_id | string |  |  |
| status | RoleKeyStatus |  | ACTIVE | PAUSED | REVOKED |




## MsgSetRoleKeyStatusResponse
MsgSetRoleKeyStatusResponse is the request/response type for the module API.




## MsgSubmitAttestation
MsgSubmitAttestation is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| submitter | string |  |  |
| attestation | AttestationCommitment |  |  |




## MsgSubmitAttestationResponse
MsgSubmitAttestationResponse is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| challenge_deadline_height | int64 |  |  |




## MsgUpdateParams
MsgUpdateParams is the request/response type for the module API.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | string |  |  |
| params | Params |  |  |




## MsgUpdateParamsResponse
MsgUpdateParamsResponse is the request/response type for the module API.










## Msg
Msg defines the gRPC query/tx service for x/appmining.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| RegisterApplication | MsgRegisterApplication | MsgRegisterApplicationResponse | RegisterApplication handles the corresponding module request. |
| SubmitAttestation | MsgSubmitAttestation | MsgSubmitAttestationResponse | SubmitAttestation handles the corresponding module request. |
| ChallengeAttestation | MsgChallengeAttestation | MsgChallengeAttestationResponse | ChallengeAttestation handles the corresponding module request. |
| ResolveChallenge | MsgResolveChallenge | MsgResolveChallengeResponse | ResolveChallenge handles the corresponding module request. |
| FinalizeAttestation | MsgFinalizeAttestation | MsgFinalizeAttestationResponse | FinalizeAttestation handles the corresponding module request. |
| GrantGasPayer | MsgGrantGasPayer | MsgGrantGasPayerResponse | GrantGasPayer handles the corresponding module request. |
| RevokeGasPayer | MsgRevokeGasPayer | MsgRevokeGasPayerResponse | RevokeGasPayer handles the corresponding module request. |
| SetAttestationSubmitter | MsgSetAttestationSubmitter | MsgSetAttestationSubmitterResponse | SetAttestationSubmitter handles the corresponding module request. |
| ConsumeGasCredit | MsgConsumeGasCredit | MsgConsumeGasCreditResponse | ConsumeGasCredit handles the corresponding module request. |
| SetEpochBudget | MsgSetEpochBudget | MsgSetEpochBudgetResponse | SetEpochBudget handles the corresponding module request. |
| UpdateParams | MsgUpdateParams | MsgUpdateParamsResponse | UpdateParams handles the corresponding module request. |
| SetApplicationStatus | MsgSetApplicationStatus | MsgSetApplicationStatusResponse | SetApplicationStatus handles the corresponding module request. |
| Pause | MsgPause | MsgPauseResponse | Pause sets pause bits to true only (Q-06). Resume is authority-only via UpdateParams. |
| FundRewardPool | MsgFundRewardPool | MsgFundRewardPoolResponse | FundRewardPool transfers coins from authority into the module reward pool (C-5 / stage 2). |
| OpenEntitlementPool | MsgOpenEntitlementPool | MsgOpenEntitlementPoolResponse | --- V1 entitlement control plane (UGE requirements 1.0 / workorder P0) ---
OpenEntitlementPool creates a V1 fixed entitlement fund pool ledger. |
| FundEntitlementPool | MsgFundEntitlementPool | MsgFundEntitlementPoolResponse | FundEntitlementPool handles the corresponding module request. |
| SetEntitlementPoolPaused | MsgSetEntitlementPoolPaused | MsgSetEntitlementPoolPausedResponse | SetEntitlementPoolPaused handles the corresponding module request. |
| RegisterRoleKey | MsgRegisterRoleKey | MsgRegisterRoleKeyResponse | RegisterRoleKey handles the corresponding module request. |
| SetRoleKeyStatus | MsgSetRoleKeyStatus | MsgSetRoleKeyStatusResponse | SetRoleKeyStatus handles the corresponding module request. |
| PublishCandidate | MsgPublishCandidate | MsgPublishCandidateResponse | PublishCandidate publishes an immutable V1 candidate commitment (no pool lock). |
| OpenChallenge | MsgOpenChallenge | MsgOpenChallengeResponse | OpenChallenge is the V1 entitlement challenge opener (attestation uses ChallengeAttestation). |
| ResolveEntitlementChallenge | MsgResolveEntitlementChallenge | MsgResolveEntitlementChallengeResponse | ResolveEntitlementChallenge avoids clashing with attestation MsgResolveChallenge. |
| ApproveEntitlement | MsgApproveEntitlement | MsgApproveEntitlementResponse | ApproveEntitlement handles the corresponding module request. |
| ClaimEntitlement | MsgClaimEntitlement | MsgClaimEntitlementResponse | ClaimEntitlement handles the corresponding module request. |
| FreezeEntitlement | MsgFreezeEntitlement | MsgFreezeEntitlementResponse | FreezeEntitlement handles the corresponding module request. |
| RevokeEntitlement | MsgRevokeEntitlement | MsgRevokeEntitlementResponse | RevokeEntitlement handles the corresponding module request. |




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
QuerySupplyRequest is the request type for the Query/Supply RPC method


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
| EditNFT | MsgEditNFT | MsgEditNFTResponse | EditNFT defines a method for editing a nft. |
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
| nft_uid_pairs | NFTUIDPair | repeated | per-token conversion bindings between CW721 token UIDs and Cosmos NFT
UIDs. |
| refund_receivers | RefundReceiver | repeated | original CW721 owners recorded at IBC send, used for timeout/error
refunds. |




## NFTUIDPair
NFTUIDPair binds a CW721 token UID ("<tokenID>,<contractAddress>") to a
Cosmos NFT UID ("<nftID>,<classID>"). Import rebuilds both directions of
the runtime bidirectional index from this single entry.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token_uid | string |  |  |
| nft_uid | string |  |  |




## Params
Params defines the cw721 module params


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enable_cw721 | bool |  | parameter to enable the conversion of Cosmos nft <--> CW721 tokens. |
| enable_evm_hook | bool |  | parameter to enable the EVM hook that converts an CW721 token to a Cosmos
NFT by transferring the Tokens through a MsgEthereumTx to the
ModuleAddress Ethereum address. |
| wasm_code_id | uint64 |  | code ID of the CW721 wasm contract used for auto-instantiation. |




## RefundReceiver
RefundReceiver records the original CW721 owner of a converted token so an
IBC timeout/error refund can return the token to the right address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contract_address | string |  | lowercased CW721 contract address |
| token_id | string |  |  |
| owner | string |  | owner address of the original token |












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
| receiver | string |  | bech32 address to receive native Cosmos NFTs |
| sender | string |  | sender hex address from the owner of the given CW721 tokens |
| class_id | string |  | nft classID to convert to CW721 |
| nft_ids | string | repeated | nftID to convert to CW721 |




## MsgConvertCW721Response
MsgConvertCW721Response returns no fields


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contract_address | string |  | CW721 token contract address registered in a token pair |
| token_ids | string | repeated | tokenID to convert |
| receiver | string |  | bech32 address to receive native Cosmos coins |
| sender | string |  | sender hex address from the owner of the given CW721 tokens |
| class_id | string |  | nft classID to convert to CW721 |
| nft_ids | string | repeated | nftID to convert to CW721 |




## MsgConvertNFT
MsgConvertNFT defines a Msg to convert a native Cosmos nft to a CW721 token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  | nft classID to convert to CW721 |
| nft_ids | string | repeated | nftID to convert to CW721 |
| receiver | string |  | recipient hex address to receive CW721 token |
| sender | string |  | cosmos bech32 address from the owner of the given Cosmos NFTs |
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




## MsgUpdateParams
MsgUpdateParams is the Msg/UpdateParams request type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | string |  | authority is the address of the governance account. |
| params | Params |  | params defines the x/cw721 module parameters to update.

Since: v0.4.1 |




## MsgUpdateParamsResponse
MsgUpdateParamsResponse is the Msg/UpdateParams response type.










## Msg
Msg defines the cw721 Msg service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ConvertNFT | MsgConvertNFT | MsgConvertNFTResponse | ConvertNFT mints a CW721 representation of the native Cosmos nft
that is registered on the token mapping. |
| ConvertCW721 | MsgConvertCW721 | MsgConvertCW721Response | ConvertCW721 mints a native Cosmos coin representation of the CW721 token
contract that is registered on the token mapping. |
| TransferCW721 | MsgTransferCW721 | MsgTransferCW721Response | TransferCW721 defines a method to transfer CW721 tokens between chains via IBC |
| UpdateParams | MsgUpdateParams | MsgUpdateParamsResponse | UpdateParams updates the module parameters. Since the cw721 module has
no authority account, this RPC is registered for interface completeness
and is only callable by the governance authority. |




# uptick/erc721/v1/erc721.proto




## TokenPair
TokenPair defines an instance that records a pairing consisting of a native
Cosmos Coin and an ERC721 token address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| erc721_address | string |  | address of ERC721 contract token |
| class_id | string |  | cosmos nft class ID to be mapped to |




## UIDPair
defines the unique id of nft asset


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| erc721_did | string |  | address of ERC721 contract token + tokenId |
| class_did | string |  | cosmos nft class ID to be mapped to + nftId |






## Owner
Owner enumerates the ownership of a ERC721 contract.

| Name | Number | Description |
| ---- | ------ | ----------- |
| OWNER_UNSPECIFIED | 0 | OWNER_UNSPECIFIED defines an invalid/undefined owner. |
| OWNER_MODULE | 1 | OWNER_MODULE erc721 is owned by the erc721 module account. |
| OWNER_EXTERNAL | 2 | EXTERNAL erc721 is owned by an external account. |









# uptick/erc721/v1/genesis.proto




## GenesisState
GenesisState defines the module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| params | Params |  | module parameters |
| token_pairs | TokenPair | repeated | registered token pairs |
| nft_uid_pairs | NFTUIDPair | repeated | per-token conversion bindings between EVM token UIDs and Cosmos NFT UIDs. |
| refund_receivers | RefundReceiver | repeated | original ERC721 owners recorded at IBC send, used for timeout/error
refunds. |




## NFTUIDPair
NFTUIDPair binds an EVM token UID ("<tokenID>,<contractAddress>") to a
Cosmos NFT UID ("<nftID>,<classID>"). Import rebuilds both directions of
the runtime bidirectional index from this single entry.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token_uid | string |  |  |
| nft_uid | string |  |  |




## Params
Params defines the erc721 module params


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enable_erc721 | bool |  | parameter to enable the conversion of Cosmos nft <--> ERC721 tokens. |
| enable_evm_hook | bool |  | parameter to enable the EVM hook that converts an ERC721 token to a Cosmos
NFT by transferring the Tokens through a MsgEthereumTx to the
ModuleAddress Ethereum address. |




## RefundReceiver
RefundReceiver records the original ERC721 owner of a converted token so an
IBC timeout/error refund can return the token to the right address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| evm_contract_address | string |  | lowercased ERC721 contract address (0x + 40 hex) |
| token_id | string |  |  |
| evm_address | string |  | EVM address of the original owner (0x + 40 hex) |












# uptick/erc721/v1/query.proto




## QueryEvmAddressRequest
QueryTokenPairRequest is the request type for the Query/TokenPair RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| port | string |  | token identifier can be either the hex contract address of the ERC721 or
the Cosmos nft classID |
| channel | string |  |  |
| class_id | string |  |  |




## QueryEvmAddressResponse
QueryEvmAddressResponse is the response type for the Query/Params RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token_pair | TokenPair |  |  |




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
| token | string |  | token identifier can be either the hex contract address of the ERC721 or
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










## Query
Query defines the gRPC queried service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| TokenPairs | QueryTokenPairsRequest | QueryTokenPairsResponse | TokenPairs retrieves registered token pairs |
| TokenPair | QueryTokenPairRequest | QueryTokenPairResponse | TokenPair retrieves a registered token pair |
| EvmContract | QueryEvmAddressRequest | QueryEvmAddressResponse | EvmContract retrieves a registered evm contract |
| Params | QueryParamsRequest | QueryParamsResponse | Params retrieves the erc721 module params |




# uptick/erc721/v1/tx.proto




## MsgConvertERC721
MsgConvertERC721 defines a Msg to convert a ERC721 token to a native Cosmos
nft.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| evm_contract_address | string |  | ERC721 token contract address registered in a token pair |
| evm_token_ids | string | repeated | tokenID to convert |
| cosmos_receiver | string |  | bech32 address to receive native Cosmos coins |
| cosmos_sender | string |  | sender bech32 address from the owner of the given ERC721 tokens |
| class_id | string |  | nft classID to cnvert to ERC721 |
| cosmos_token_ids | string | repeated | nftID to cnvert to ERC721 |




## MsgConvertERC721Response
MsgConvertERC721Response returns no fields


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| evm_contract_address | string |  | ERC721 token contract address registered in a token pair |
| evm_token_ids | string | repeated | tokenID to convert |
| cosmos_receiver | string |  | bech32 address to receive native Cosmos coins |
| evm_sender | string |  | sender hex address from the owner of the given ERC721 tokens |
| class_id | string |  | nft classID to cnvert to ERC721 |
| cosmos_token_ids | string | repeated | nftID to cnvert to ERC721 |




## MsgConvertNFT
MsgConvertNFT defines a Msg to convert a native Cosmos nft to a ERC721 token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| class_id | string |  | nft classID to cnvert to ERC721 |
| cosmos_token_ids | string | repeated | nftID to cnvert to ERC721 |
| evm_receiver | string |  | recipient hex address to receive ERC721 token |
| cosmos_sender | string |  | cosmos bech32 address from the owner of the given Cosmos coins |
| evm_contract_address | string |  | ERC721 token contract address registered in a token pair |
| evm_token_ids | string | repeated | ERC721 token id registered in a token pair |




## MsgConvertNFTResponse
MsgConvertNFTResponse returns no fields




## MsgTransferERC721
MsgTransferERC721 defines a message for transferring erc721 tokens through IBC


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| evm_contract_address | string |  | evm_contract_address is the ERC721 token contract address |
| evm_token_ids | string | repeated | tokenID to convert |
| source_port | string |  | the port on which the packet will be sent |
| source_channel | string |  | the channel by which the packet will be sent |
| class_id | string |  | the class_id of tokens to be transferred |
| cosmos_token_ids | string | repeated | the non fungible tokens to be transferred |
| cosmos_sender | string |  | the sender address |
| cosmos_receiver | string |  | the recipient address on the destination chain |
| timeout_height | ibc.core.client.v1.Height |  | timeout_height is the timeout height relative to the current block height
The timeout is disabled when set to 0 |
| timeout_timestamp | uint64 |  | timeout_timestamp is the timeout timestamp in absolute nanoseconds since unix epoch
The timeout is disabled when set to 0 |
| memo | string |  | memo is an optional memo field |




## MsgTransferERC721Response
MsgTransferERC721Response defines the response type for Transfer erc721 RPC




## MsgUpdateParams
MsgUpdateParams is the Msg/UpdateParams request type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | string |  | authority is the address of the governance account. |
| params | Params |  | params defines the x/erc721 module parameters to update.

Since: v0.4.1 |




## MsgUpdateParamsResponse
MsgUpdateParamsResponse is the Msg/UpdateParams response type.










## Msg
Msg defines the erc721 Msg service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ConvertNFT | MsgConvertNFT | MsgConvertNFTResponse | ConvertNFT mints a ERC721 representation of the native Cosmos nft
that is registered on the token mapping. |
| ConvertERC721 | MsgConvertERC721 | MsgConvertERC721Response | ConvertERC721 mints a native Cosmos coin representation of the ERC721 token
contract that is registered on the token mapping. |
| TransferERC721 | MsgTransferERC721 | MsgTransferERC721Response | TransferERC721 transfers a erc721 token from one chain to another chain through IBC |
| UpdateParams | MsgUpdateParams | MsgUpdateParamsResponse | UpdateParams updates the module parameters. Since the erc721 module has
no authority account, this RPC is registered for interface completeness
and is only callable by the governance authority. |




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
QueryNFTsRequest is the request type for the Query/NFTs RPC method


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



 