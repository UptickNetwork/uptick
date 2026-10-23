# x/appmining V1 — 事件与错误码对照表 0.1

**模块：** `uptick.appmining.v1`  
**范围：** Application Mining V1（权益 Claim）控制面与业务 Msg；POC attestation 线错误码一并列出供对照，**新权益验收不得依赖 Finalize/GasCredit 路径**。  
**真源：** `types/events.go`、`types/errors.go`

---

## 1. V1 事件（`sdk.Event` type）

| Event type | 触发 Msg / 时机 | 主要 attributes（§6.1 稳定字段 + 扩展） |
|---|---|---|
| `entitlement_pool_opened` | `MsgOpenEntitlementPool` | `pool_id`, `denom`, `version` |
| `entitlement_pool_funded` | `MsgFundEntitlementPool` | `pool_id`, `funder`, `amount`, `funded`, `available` |
| `entitlement_pool_paused` | `MsgSetEntitlementPoolPaused` | `pool_id`, `status` |
| `role_key_registered` | `MsgRegisterRoleKey` | `role`, `key_id`, `version` |
| `role_key_status_changed` | `MsgSetRoleKeyStatus` | `role`, `key_id`, `old_status`, `new_status` |
| `candidate_published` | `MsgPublishCandidate` | §6.1 + `action=publish` |
| `entitlement_challenge_opened` | `MsgOpenChallenge` | §6.1 + `action=open_challenge` + `challenge_id`, `challenger` |
| `entitlement_challenge_resolved` | `MsgResolveEntitlementChallenge` | §6.1 + `action=resolve` + `challenge_id`, challenge `status` |
| `entitlement_approved` | `MsgApproveEntitlement` | §6.1 + `action=approve` + `available` |
| `entitlement_claimed` | `MsgClaimEntitlement` | §6.1 + `action=claim` + `owner` |
| `entitlement_frozen` | `MsgFreezeEntitlement` | §6.1 + `action=freeze` + `deadline`(=freeze_until) |
| `entitlement_revoked` | `MsgRevokeEntitlement` | §6.1 + `action=revoke` + `available` |

**§6.1 稳定字段（业务状态变更事件）：**  
`chain_id`, `pool_id`, `entitlement_id`, `app_id`, `action`, `prev_state`, `new_state`, `amount`, `denom`, `evidence_or_resolution_digest`, `height`, `tx_hash`  
（单元测试无 `TxBytes` 时 `tx_hash` 可为空；链上 DeliverTx 为 `sha256(tx_bytes)` 小写 hex。）

`action` 枚举：`publish|open_challenge|resolve|approve|freeze|revoke|claim`。

Attribute 常量名见 `types/events.go`。

**读链提示：** `app_id` 在 V1 事件里为 **64 位小写 hex**；POC 登记事件可能仍带 raw/hex 展示差异，UGE 黑盒以 V1 事件为准。  
控制面 Msg 定名以 ACK Q-C 为准（非工单 §3.8 示例旧名）。

---

## 2. V1 错误码（codespace `appmining`）

| Code | 符号 | 文案 | 典型场景 |
|---|---|---|---|
| 40 | `ErrPoolNotEnabled` | entitlement pool not enabled | 未达 `entitlement_enable_min_funded` / 余额不覆盖 reserved（E-8） |
| 41 | `ErrPoolPaused` | entitlement pool paused | 池 paused |
| 42 | `ErrPoolNotFound` | entitlement pool not found | 未知 `pool_id` |
| 43 | `ErrPoolExists` | entitlement pool already exists | 重复开池 |
| 44 | `ErrPoolInvariant` | entitlement pool invariant violated | `funded ≠ available+reserved+paid` |
| 45 | `ErrRoleKeyNotFound` | role key not found | 非登记角色密钥（含非 approver 冒充） |
| 46 | `ErrRoleKeyConflict` | role key conflict | 同 role+key_id 换公钥 |
| 47 | `ErrRoleKeyInactive` | role key not active | PAUSED/REVOKED |
| 48 | `ErrInvalidRole` | invalid entitlement role | 角色枚举非法 |
| 49 | `ErrInvalidRoleKeyStatus` | invalid role key status | 状态非法 |
| 50 | `ErrInvalidWireID` | invalid wire identifier | pool_id/key_id/denom 等 |
| 51 | `ErrEntitlementPaused` | entitlement flow paused | 全局 entitlement pause |
| 52 | `ErrCandidateConflict` | candidate id content conflict | 同 ID 异内容 Publish |
| 53 | `ErrCandidateNotFound` | candidate not found | |
| 54 | `ErrCandidateNotPublished` | candidate not published | 已 REJECTED 等 |
| 55 | `ErrAuthExpired` | authorization expired | `expires_at` |
| 56 | `ErrAuthReplay` | authorization nonce replay | |
| 57 | `ErrAuthMalformed` | authorization malformed | |
| 58 | `ErrChainIDMismatch` | chain_id mismatch | E-2 |
| 59 | `ErrInvalidEvidenceType` | invalid evidence type | `INVALID_EVIDENCE_CANNOT_FREEZE` |
| 60 | `ErrDuplicateOpenChallenge` | challenger already has open challenge | |
| 61 | `ErrEntitlementChallengeNotFound` | entitlement challenge not found | |
| 62 | `ErrEntitlementChallengeNotOpen` | entitlement challenge not open | |
| 63 | `ErrInvalidOutcome` | invalid challenge outcome | 非 DISMISSED/UPHELD |
| 64 | `ErrChallengeWindowClosed` | challenge window closed | 逾期 OpenChallenge |
| 65 | `ErrPoolMismatch` | pool_id mismatch | E-2 跨 pool |
| 66 | `ErrInsufficientAvailable` | insufficient pool available | 池耗尽 |
| 67 | `ErrEntitlementExists` | entitlement already approved | 重复 Approve |
| 68 | `ErrEntitlementNotFound` | entitlement not found | |
| 69 | `ErrNotClaimable` | entitlement not claimable | 冻结中 / 已撤销 |
| 70 | `ErrAlreadyClaimed` | entitlement already claimed | Claim 后再 Freeze/Revoke |
| 71 | `ErrOpenOrUpheldChallenge` | open or upheld challenge blocks approval | |
| 72 | `ErrBeneficiaryMustBeOwner` | beneficiary must be application owner | 跨 app Claim |
| 73 | `ErrFreezeOnce` | entitlement already frozen once | |
| 74 | `ErrFreezeDuration` | freeze_until exceeds 72h or not in future | |
| 75 | `ErrNotFrozen` | entitlement not effectively frozen | Revoke 要求有效冻结 |
| 76 | `ErrDualAuthRequired` | independent risk and audit proofs required | 同 key / 缺一方 |
| 2 | `ErrUnauthorized` | unauthorized | 签名篡改、金额与候选不一致等（E-1 / UNAUTHORIZED_AMOUNT） |
| 11 | `ErrChallengeWindow` | challenge window not elapsed | Approve 早于 604800s（E-6） |

POC 线常用码（仅审计，勿作 V1 验收）：`3–39`（如 `ErrAttestationPaused`=27、`ErrSettlementPaused`=21、`ErrGasCreditPaused`=29）。升级 `ConsensusVersion=2` 后旧写路径默认 pause。

---

## 3. 对账字段（QueryPool）

| 字段 | 含义 |
|---|---|
| `pool.funded` / `available` / `reserved` / `paid` | 四分账；恒等式 `funded = available + reserved + paid` |
| `module_balance` | 模块账户 `appmining_entitlement_pool` 对 `pool.denom` 余额 |
| 期望 | `module_balance = available + reserved`（已 `paid` 部分已转出） |
| `entitlement_flow_enabled` | 未 pause 且 `funded >= entitlement_enable_min_funded` 且余额覆盖 `reserved` |

预检脚本：`scripts/check-entitlement-pool-funding.sh`。
