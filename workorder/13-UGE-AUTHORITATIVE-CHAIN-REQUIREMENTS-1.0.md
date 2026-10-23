# UGE 对 Uptick Chain 的 Application Mining 开发要求 1.0

状态：**UGE 团队提出的实施基线**。这是需求，不是链端已实现或已部署的声明。09–12 是形成过程与参考测试；如有冲突，本文件的 V0 业务语义优先。UGE 不修改 Uptick Chain 源码。适用范围先限 UGE/UIP 候选私链；通过联合验收后另行决定 Origin3 升级。

## 1. 不再留给链团队选择的原则

1. `x/appmining` 使用**已实际注资的固定基金池**，不是逐块新增代币、每期应用提案预算或旧 Gas Credit。基金池目标规模是原生代币总供应量 10 亿的 50%，即 **5 亿枚原生代币**；仅在这笔储备已转入隔离模块账户并完成余额核验后启用新权益流程。链升级不得为此隐式增发。若现有链的资金分配与目标不一致，链团队应报告缺口并保持新流程禁用，而不是假定已注资。
2. UGE 计算合格 QBT、PAV、有效 Gas 与版本化奖励，输出已授权金额。链端验证授权、证据承诺、挑战状态、应用身份和资金守恒，不重算复杂业务算法。应用不得自行填写可领取金额。
3. 必须先在链上发布不可覆盖的候选承诺；**链上区块时间**是 7 天挑战期唯一权威起点。公开挑战及裁决状态由 `x/appmining` 保存，未决或成立的挑战阻止核定。不得仅凭 UGE 服务器时钟或离链声明认定挑战期结束。
4. 核定时原子锁定真实可兑付资产；应用当前登记 owner 的钱包主动发起 Claim，V0 受益地址必须是该 owner 地址。系统自动打款、终端用户无感 Gas、额外生态奖励曲线不属于 V0 链端验收。
5. 临时冻结最长 72 小时，每笔权益在 V0 最多冻结一次。冻结不释放负债；到期无需管理员许可即可重新 Claim。撤销只能针对冻结中、未领取的权益，需 risk 与 audit 两把不同公钥授权；撤销原子释放锁定资产。已领取不可撤销转账。

## 2. 规范对象与状态

`Pool`：`pool_id, denom, funded, available, reserved, paid, paused, version`。`funded = available + reserved + paid` 恒成立；`reserved <= 模块账户实际余额`。所有金额是资产最小单位的无符号十进制整数，不得用浮点数。

`Candidate`：`entitlement_id, app_id, epoch, amount, denom, gas_used, qualified_tx_count, pav_id, evidence_root, gas_evidence_root, rule_version`，另含 `candidate_hash`、`pool_id`、发布交易/高度/链上时间、发布者和挑战截止。`entitlement_id` 与完整候选的规范化编码绑定；同 ID 异内容永远拒绝。

`Challenge`：`candidate_id, challenger, evidence_type, evidence_digest, reference_digest, status(OPEN/UPHELD/DISMISSED), opened_at, reviewer, resolution_digest`。允许任何链账户发起，每个账户对同一候选只能有一个未决挑战。链只存摘要，不存个人信息或原始证据。允许的证据类型固定为 `TX_MISSING_OR_UNFINALIZED`、`RECEIPT_QBT_MISMATCH`、`DUPLICATE_OR_FAMILY_CONFLICT`、`CONFIRMED_REFUND_OR_REVERSAL`、`APP_OR_EMITTER_INELIGIBLE`、`REPRODUCIBLE_CALCULATION_ERROR`。裁决由注册 reviewer 授权并留痕。

`Entitlement`：`entitlement_id, app_id, owner_at_last_query, amount, denom, state(CLAIMABLE/FROZEN/REVOKED/CLAIMED), freeze_until, freeze_count, approval_tx, claim_tx, evidence_root, rule_version`。候选可以 `PUBLISHED/REJECTED`，但在核定前不产生链端负债。`REVOKED`、`CLAIMED` 为终态。

## 3. 必须实现的 Msg / Query（名称与字段语义均为 UGE 要求）

| Msg | 授权及校验 | 原子结果 |
|---|---|---|
| `MsgPublishCandidate` | UGE publisher；完整候选哈希、app、池、链 ID、算法版本；同 ID 同内容幂等，异内容拒绝 | 写入区块时间及 7 天截止；不锁款 |
| `MsgOpenChallenge` | 任意链账户签名，证据类型与摘要合规、挑战期内、同一账户无未决重复 | 新建 `OPEN` 挑战；不改变池账 |
| `MsgResolveChallenge` | 注册 reviewer；绑定挑战 ID、结果与理由摘要；仅 `OPEN` 可裁决 | `UPHELD` 则候选 `REJECTED`；`DISMISSED` 保留候选；均不可静默删除 |
| `MsgApproveEntitlement` | UGE approver 的域分隔授权；候选发布满 7 天、无 `OPEN/UPHELD` 挑战；应用 active、owner 可查、金额>0、denom 匹配、规则版本允许、同 ID 未核定、池可用额充足 | `available -= amount; reserved += amount`，状态 `CLAIMABLE` |
| `MsgFreezeEntitlement` | 注册 risk；证据类型/摘要、nonce/到期；仅 `CLAIMABLE`、未领取、`freeze_count=0`，到期不超过当前链上时间+72 小时 | 状态 `FROZEN`、`freeze_count=1`，`reserved` 不变 |
| `MsgRevokeEntitlement` | risk 与 audit 两个独立注册公钥对同一权益、链/池、证据和理由摘要分别签名；仅有效冻结中且未 Claim | 状态 `REVOKED`，`reserved -= amount; available += amount` |
| `MsgClaimEntitlement` | 当前登记的应用 owner 钱包签名；受益地址=owner；状态 `CLAIMABLE`，或冻结期已到自动视为 `CLAIMABLE`；同 ID 最多一次 | `reserved -= amount; paid += amount`，从隔离账户真实转账，状态 `CLAIMED` |

链端提供 `QueryPool(pool_id)`、`QueryCandidate(entitlement_id)`、`QueryChallenge(challenge_id)`、`QueryEntitlement(entitlement_id)`、`QueryEntitlementsByApp(app_id, page)`、`QueryRoles`。所有成功状态变更发稳定、版本化事件，至少含链 ID、池 ID、权益 ID、应用 ID、动作、前后状态、金额/denom、证据/决议摘要、区块高度和交易哈希。不得把旧 `MsgFinalizeAttestation` 或 `GasCreditAccount` 当作新权益状态。

## 4. 授权、时间与升级规则

- UGE 定义五种权限：publisher、approver、reviewer、risk、audit；业务金额只能由 approver 核定。角色公钥在链上版本化登记，支持轮换、撤销和暂停；旧密钥只用于验证历史，不再授权新消息。
- 所有链端授权证明必须绑定协议版本、操作类型、角色、链 ID、池 ID、权益/挑战 ID、完整业务字段、key ID、nonce 和到期；链端拒绝重放、跨链/跨池、过期和非规范编码。V0 要求链端实现 Ed25519 域分隔验签及固定字段顺序的规范字节编码。权威编码在 [`wire-v1.mjs`](../../protocol/app-mining-v1/wire-v1.mjs)：`UGE-APP-MINING-WIRE-V1` 的 UTF-8 字节加 `0x00`，然后逐字段拼接 `u16be(字段名 UTF-8 长度) || 字段名 || u32be(值 UTF-8 长度) || 值`；字段顺序由源码中的操作表固定。完整候选使用独立域 `UGE-APP-MINING-CANDIDATE-WIRE-V1`，对全部候选字段作同样编码后取 SHA-256。**编码不采用 JSON.stringify**。交叉语言黄金向量见 [`app-mining-v1-wire-golden-1.0.json`](../../spec/settlement/app-mining-v1-wire-golden-1.0.json)；旧 `approval.mjs` 和 `dispute-rules.mjs` 的 0.1/V0 签名仅为历史原型，链端应以此 1.0 wire 规范实现。
- 链上时间统一取当前区块时间；7 天=604800 秒，72 小时=259200 秒。挑战截止边界为 `now >= published_at + 604800` 可核定；冻结截止边界为 `now >= freeze_until` 可 Claim。若因节点时间异常导致倒退，不能提前释放。
- 单模块交易必须在账户转账失败时全部回滚。节点重启、升级、重复消息、并发提交不得破坏 `funded=available+reserved+paid`。旧 POC 状态保留审计，但不得自动兑换为新权益；迁移应先暂停旧资金流，逐类核对历史资金和在途状态。

## 5. 交付与验收

链团队交付：protobuf/Go Msg 与 Query、模块账户与升级/注资方案、角色注册和验签、事件与错误码、迁移脚本、单元/集成测试、私链二进制及版本/commit。每个[机器验收向量](../../spec/settlement/app-mining-v1-acceptance-vectors-0.1.json)的 case ID 必须在链端测试中有对应结果；另须覆盖真实签名篡改、跨链/池重放、并发 Claim/撤销、账户余额、升级重启与异常时间。UGE 将以读链方式验证所有结果，不能以本地参考模型替代链端黑盒测试。

## 6. UGE 自身继续完成的工作（不构成链端等待条件）

UGE 负责规范编码/黄金签名向量、真实发布与挑战事件 ingest、奖励计算及挑战裁决服务、角色密钥运维、读链对账、应用 SDK 的候选/可领取展示及 owner Claim 交易构建、退款与反刷量证据。V0 链端要求已经由本文件定下；实现中的接口细节如需偏离，应作为版本化变更提案返回 UGE，而不是让链团队自行决定业务模式。
