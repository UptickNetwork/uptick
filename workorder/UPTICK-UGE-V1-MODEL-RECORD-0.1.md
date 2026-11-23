# UPTICK-UGE-V1-MODEL-RECORD-0.1

> 日期：2026-10-26
> 维护方：Uptick 链团队
> 性质：**V1 权威模型链侧记录**（设计 + 实现 + 运行快照的唯一索引文档）
> 依据：UGE `13-UGE-AUTHORITATIVE-CHAIN-REQUIREMENTS-1.0.md`（权威需求）、
> `UPTICK-UGE-V1-ONLY-SCOPE-NOTICE-0.1.md`（DEC-09 范围裁定）、
> `UPTICK-UGE-CHAIN-ACTION-REPLY-0.1.md` §1.0（链侧同拍板）
> 实现基线：`uge/v1.1` commit `2d141eb3d860…`（测试链 uptick_7777-1 运行二进制同源）

---

## 0. 定位

本文记录 x/appmining **V1 模型（Entitlement 模型）** 的当前设计与实现全貌：实体、状态机、
Msg/Query 面、wire 签名、资金流、参数、错误码与实链快照。V1 是唯一设计与实现目标；
旧模型（attestation / epoch budget / gas credit）已按 DEC-09 退出（本文不收录其行为定义，
仅记录弃用边界，见 §9）。

---

## 1. 实体总览

| 实体 | proto | 说明 |
|---|---|---|
| **Pool** | `Pool` | 资金池：`pool_id`(wire id)/`denom`/`funded`/`available`/`reserved`/`paid`/`paused`/`version`。approve 时从 available 划 reserved，claim 时 paid 并释放 reserved，revoke 时 reserved 回 available |
| **RoleKey** | `RoleKey` | 五角色 Ed25519 密钥：PUBLISHER / APPROVER / REVIEWER / RISK / AUDIT；状态 ACTIVE/PAUSED/REVOKED；跨角色同公钥禁止（错误码 46，F-2） |
| **Candidate** | `Candidate` | 已发布候选（approve 前无池负债）：`entitlement_id`(64 hex)/`candidate_hash`/`evidence_root`/`gas_evidence_root`/`rule_version`/`pav_id`/`epoch`/`amount`/`pool_id`/`published_at`/`challenge_deadline`/`max_observed_time`（防回滚水位线）。状态 PUBLISHED/REJECTED |
| **EntitlementChallenge** | `EntitlementChallenge` | V1 挑战记录（独立于 legacy Challenge）：OPEN/UPHELD/DISMISSED；`evidence_type`∈7 类 EvidenceType；`resolution_digest`(64 hex) 由 reviewer resolve 时写入 |
| **Entitlement** | `Entitlement` | **仅 approve 后创建**（此时池负债确立）：CLAIMABLE/FROZEN/REVOKED/CLAIMED；`freeze_until`/`freeze_count`/`approval_tx`/`claim_tx` |

**Claim** 是动作而非实体：owner 对 CLAIMABLE 且未冻结的 Entitlement 发起资金领取。
**共享基础设施**（V1 依赖，保留）：`Application` 注册（ACTIVE/SUSPENDED/REVOKED）、
`Params`、`Pause`。

## 2. 状态机

```
Candidate:  PUBLISHED ──(approve)──► 创建 Entitlement
                 │
                 └─(挑战 UPHELD)──► REJECTED

EntitlementChallenge: OPEN ──(reviewer)──► UPHELD / DISMISSED
                          └─(超时未裁决, EndBlock 自动/任何人手动)──► EXPIRED（押金退回，候选仍 PUBLISHED）

Entitlement: CLAIMABLE ──freeze──► FROZEN ──(freeze_until 到期)──► CLAIMABLE
                 │                    │
                 └──claim──► CLAIMED  └──revoke(双签)──► REVOKED（资金回流池）
```

**业务主路径（F-3 收紧后，DEC-01 双路径）**：

1. `RegisterApplication`（owner）→ 2. `RegisterRoleKey` ×5 → 3. `OpenEntitlementPool` +
`FundEntitlementPool` → 4. `PublishCandidate`（PUBLISHED，挑战窗启动）→ 5a. 挑战路径
`OpenChallenge` → `ResolveEntitlementChallenge`(UPHELD|DISMISSED) → 5b. 无挑战 approve 走
scheme C（要求候选零挑战记录）→ 6. `ApproveEntitlement`（reserve，→CLAIMABLE）→
7. 可选 `FreezeEntitlement` / `RevokeEntitlement` → 8. `ClaimEntitlement`（→CLAIMED）。

**approve 校验序**（决定错误码优先级）：entitlement_flow_enabled/pause → 候选 PUBLISHED →
rule_version 白名单（77，F-1）→ 挑战窗实时判定 `now < published_at + ChallengeWindowSeconds`（11/64）→ wire 验签（55/56/57/58）→ 角色密钥（45/46/47）→
绑定链上已结算裁决（DISMISSED 或 **EXPIRED**）或零挑战（**78**，F-3/DEC-01）→ open/upheld 挑战阻断（71）→
池 available 足额（66）→ beneficiary=app owner（72）。

### 2.1 挑战押金语义（P2-2，链侧拍板方案 A，待 UGE 书面确认）

`challenge_open_access = deposit` 模式下押金（denom 强制 = pool denom）结算规则：

| 裁决 | 押金流向 | 事件标记 |
|---|---|---|
| UPHELD（挑战成立） | 退还挑战者 | `bond_flow=refund_in` |
| DISMISSED（挑战不成立） | **罚没进被挑战候选所在池**，`funded`+`available` 双加 | `bond_flow=slash_in` |
| **EXPIRED（超时未裁决）** | **全额退还挑战者**（中立结局，无人受罚） | `bond_flow=refund_in` |

- 这是**有意设计**（收取处代码注释明示 confiscation 进池动机），非账目缺陷：池不变量
  `funded = available + reserved + paid` 恒成立，两流向均有测试覆盖。
- 已知并接受的语义点：①罚没款入 available 后 fungible，失败挑战实质补贴该池后续 claimant
  （含自博弈场景，无套利空间）；②`funded` 口径因此含罚没款；③与 legacy（UPHELD 罚没进
  RewardPool、DISMISSED 退还）**方向相反**，属有意变更；④入池即不可回滚。
- 依据与请 UGE 确认的问题（Q1~Q4）：`UPTICK-UGE-BOND-SEMANTICS-REQUEST-0.1.md`（建议登记
  DEC-11）。实现位置：`entitlement_p1.go` 收取 :240-262 / 结算 `releaseEntitlementChallengeDeposit`
  :384-422。

### 2.2 超时未裁决（EXPIRED）

**问题**：`challenge_deadline` 到期只关闭挑战入口并放行 approve 的时间闸门；已存在的 OPEN 挑战
仍会通过 `HasOpenOrUpheldEntitlementChallenge` 永久阻断 approve，reviewer 不作为时候选会无限期停在
PUBLISHED，押金锁在 escrow。仅靠"任何人可发 MsgExpire"不够：零押金（`challenge_open_access=open`）
挑战没有任何人有经济动机去结算，挑战者私钥丢失时同理。

**结算门槛**：

```
now >= resolution_deadline
resolution_deadline = opened_at + (candidate.challenge_deadline - candidate.published_at) + grace
```

`resolution_deadline` 在 **open 时封印进挑战记录**（proto 字段 14），之后治理修改
`challenge_resolution_grace_seconds` 不能挪动在飞挑战（既不能缩短也不能延长）。字段为 0 的旧记录
（本字段上线前写入）读侧回退到动态计算，保证历史语义不变；`Migrate1to2` 与 `InitGenesis` 会把
它们补上快照。

**两条结算路径，语义完全一致**（共用 `settleExpiredChallenge`）：

1. **自动（主路径）**：模块 `EndBlock` 调用 `SweepExpiredChallenges`，按 deadline 桶（big-endian
   有序）扫描到第一个未到期项即停，每块最多结算 `MaxExpiredChallengesPerBlock=64` 条，剩余顺延到
   下一块。EndBlock 永不返回错误（返回错误会 halt 链），单条失败只记日志并在下一块重试。
2. **手动**：`MsgExpireEntitlementChallenge`（无角色、无 wire proof，任何人可发），用于审计留痕、
   被暂停链的手工推进，以及运维冷启动时确认状态。

两条路径都把挑战置为 **EXPIRED**：押金**全额退还挑战者**、候选保持 PUBLISHED、写入确定性 digest。
自动路径在事件的 `submitter` 属性记 `endblocker`，与真实交易来源区分。

设计要点：

- **中立结局**：既不判挑战成立（那会变成"500 押金即可否候选"的零成本审查攻击），也不判
  挑战不成立（那会罚没挑战者，等于"举报即罚款"）。超时 = 双方都无过错，各自回到原位。
- **不经过 `requireEntitlementFlow`**：池暂停 / entitlement_paused / 资金门槛不达标时，唯一的
  退款出口必须仍可用，否则事故处置会把押金锁死（实测过 code=41）。自动路径同样不带该闸门。
- **reviewer 仍保有优先权**：grace 期内只有 reviewer 能结案；过期后 reviewer 的裁决会被
  62 拒绝（挑战已不是 OPEN），所以不存在"重复结算"。reviewer 一旦裁决，挑战立即移出 deadline 桶，
  sweep 不会把它覆盖成 EXPIRED。
- **approve 仍需绑定 digest**：EXPIRED 后 `HasAnyEntitlementChallenge` 为真，空 hash 路径按
  DEC-01 scheme C 继续拒绝，必须传入 EXPIRED 的确定性 digest（可从事件或
  `q appmining entitlement-challenge` 取得）。

### 2.3 挑战索引（读写复杂度）

挑战记录以 `challenge_id` 为主键，但所有业务判定都是"按候选查"：approve 的三个闸门
（open/upheld、any、settled-digest）都在 tx 路径上。为避免 O(全部历史挑战) 扫描，模块维护两个
派生结构，**只在 `SetEntitlementChallenge` 这一个写入点维护**（open / resolve / expire / genesis
全部经过它，不会漂移）：

| 索引 | 键 | 用途 |
|---|---|---|
| 候选索引 | `candidate_id \| 0x00 \| challenge_id` | approve 三闸门、按候选列挑战；`0x00` 保证 id 边界（`aa` 不匹配 `aab`） |
| 到期桶 | `be(resolution_deadline) \| challenge_id` | EndBlock sweep 的有序游标；结算即删桶 |

一致性由不变量 `appmining/challenge-index-consistent` 守护（
`uptickd query crisis invariant appmining challenge-index-consistent`）：任一挑战缺候选索引、OPEN
挑战缺桶、或已结算挑战仍占桶，都会报 broken。

**历史数据的收敛方式（重要）**：v1 二进制写入的挑战没有任何派生索引。为避免“裸替换二进制”导致
OPEN 挑战对 approve 隐形（进而错误放行资金），模块用一个持久化的完整性标记
（`0x17` 前缀的单条记录：`done` + `cursor`）区分两种读法：

| `done` | 读路径 | 写入方 |
|---|---|---|
| `false` | 回落记录表 + 按 `candidate_id` 过滤（等价于改造前语义） | — |
| `true` | 走候选索引，O(该候选挑战数) | `EnsureChallengeIndexes` 遍历到表尾 / `Migrate1to2` 全量回填 |

`EnsureChallengeIndexes` 由 **EndBlocker 每块推进 256 条**（`MaxChallengeIndexBackfillPerBlock`；
Begin/EndBlock 用无限 gas meter，分块是给出块时延而非 gas），补齐候选索引、为 OPEN 且
`resolution_deadline=0` 的旧记录封存 deadline（从而铺出到期桶）、并补写 per-challenger 开放索引。
它幂等、可断点续跑、只写确有缺口的记录。顺序为 `Ensure → Sweep`，因此**升级后第一个块就能把已过期
的 OPEN 挑战结算为 EXPIRED 并退还押金**，不依赖任何迁移或人工交易。

不变量在 `done=false` 期间跳过（否则中途必然误报），`done=true` 后严格校验。
推论：**升级仍需全员同一高度切换**（首块即写状态），但不再需要 upgrade plan，也不再需要
`v0.5.1-cv2` 这个专用 plan 名（该包已删除）。

## 3. Msg 面（V1 保留 13 条）

| Msg | 签名方 | 要点 |
|---|---|---|
| `OpenEntitlementPool` | authority | `pool_id` 必过 ValidateWireID；denom 唯一 |
| `FundEntitlementPool` | 任意 | **amount 为纯数字串（不带 denom）**；entitlement_flow_enabled 由 funded ≥ `entitlement_enable_min_funded` 打开 |
| `SetEntitlementPoolPaused` | authority | 池级暂停（41） |
| `RegisterRoleKey` / `SetRoleKeyStatus` | authority/owner | wire 证明注册；跨角色同公钥拒（46） |
| `PublishCandidate` | PUBLISHER 密钥 | 真实 PUBLISH wire 签名；`challenge_deadline = published_at + params.challenge_window_seconds`（`EffectiveChallengeWindowSeconds`，默认 604800；封印在 candidate 上） |
| `OpenChallenge` | 任意 | 同挑战者单开挑战（60）；窗内才可开（64） |
| `ResolveEntitlementChallenge` | REVIEWER 密钥 | UPHELD（候选 REJECTED）/ DISMISSED（写 `resolution_digest`） |
| `ExpireEntitlementChallenge` | **任意（无 wire proof）** | 超时结算的**手动路径**（常规情况由 EndBlock 自动完成，见 §2.2）：`now ≥ 挑战记录内的 resolution_deadline`；候选仍 PUBLISHED；押金退还；写确定性 `resolution_digest = sha256("appmining/entitlement-challenge-expired/v1\|" + challenge_id)`；**不经过 `requireEntitlementFlow`**，池/流程暂停时仍可执行 |
| `ApproveEntitlement` | APPROVER 密钥 | 见 §2 校验序；创建 Entitlement 并 reserve |
| `ClaimEntitlement` | app owner | 仅 CLAIMABLE 且未冻结（69）；一次性（70）；payout 到 owner |
| `FreezeEntitlement` | risk/audit 之一 | `freeze_until` ≤ 72h 且在未来（74）；每 Entitlement 仅一次（73）；FROZEN 态二次 freeze 报 69 非 73（state 检查在前，集成契约 §9 注记不改码） |
| `RevokeEntitlement` | RISK + AUDIT 双签 | 仅冻结有效期内（75）；独立性校验（76）；reserved→available 回流 |

共享：`RegisterApplication` / `SetApplicationStatus` / `Pause`（kill-switch，只能置 true）/
`UpdateParams`（整份 Params JSON，authority；resume 唯一通道）。

## 4. Query 面（V1）

`Pool` / `Roles` / `Candidate` / `EntitlementChallenge` / `Entitlement` / `EntitlementsByApp`
+ 共享 `Params` / `Application` / `Applications`。REST 前缀 **`/uptick/appmining/v1/`**
（非 `/cosmos/`）；`Pool` 响应附 `module_balance` 与 `entitlement_flow_enabled`；
单实体查询路由中的 app_id/entitlement_id 传 **base64 形态**。

## 5. wire 签名（离线可验）

- `WireProof{key_id, nonce, expires_at(unix 秒十进制), signature_base64(Ed25519)}`。
- 待签载体 = `AuthorizationBytes` canonical 编码：`protocol_version / operation / role /
  chain_id / pool_id / key_id / nonce / expires_at`（`x/appmining/types/wirev1`，与
  `tools/ugeprov` 离线助手字节一致）。
- 服务端 `verifyAndConsumeWireAuth`：role 匹配 → record 字段与 proof 一致 → 未过期（55）→
  角色密钥存在且 ACTIVE（45/47）→ nonce 未重放（56，消费即写）→ Ed25519 验签。
- 防回滚：候选/Entitlement 的 `max_observed_time` 水位线取 `max(blockTime, watermark)`。

## 6. 参数（V1 相关字段）

| 字段 | 语义 | 测试链现值（2026-10-26） |
|---|---|---|
| `entitlement_enable_min_funded` | funded 阈值，空/0 = V1 关闭（default-deny） | `1000000` |
| `entitlement_allowed_rule_versions` | approve 规则版本白名单，空 = 全拒 | `["rule-1"]` |
| `pause_state.entitlement_paused` | V1 全流程总开关 | false |
| `pause_state.registration_paused` | Application 注册开关（共享） | false |
| `pause_state.attestation/challenge/settlement/gas_credit` | **legacy 四位，V1 下恒 true** | true（tx `27B21114…` @301535） |
| `authority` / `pause_authority` | 参数与 pause 的 signer | mykey（dev 链；改参数直发 MsgUpdateParams，**须带 `--gas auto --gas-adjustment 1.5 --gas-prices 0.025auptick`**） |
| `challenge_window_blocks` / bond / `allowed_profiles` 等 | legacy 字段 | 保留但随 legacy 冻结（500 blocks / 5e20 auptick） |
| `challenge_window_seconds` | V1 挑战窗（秒）；0 = Effective 默认 604800 | 可经 `MsgUpdateParams` 调整（≥60） |
| `max_freeze_seconds` | 单次 freeze 上限；0 = 259200 | 可经 `MsgUpdateParams` 调整 |
| `max_wire_proof_validity_seconds` | WireProof 有效期上限；0 = 604800 | 取值 0 或 [60, 2592000]（下限 60s 供测试网验过期路径）；可下调（fail-closed；CLI 需 `--allow-shorten-wire-proof-validity`；链上发事件） |
| `nonce_retention_seconds` | auth nonce 保留期；0 = 691200 | 须 ≥ wire proof validity |
| `challenge_resolution_grace_seconds` | reviewer 在挑战封印窗后仍独占裁决的时间；0 = 86400（DefaultParams 故意留 0，避免改变 v0.5.1 升级高度 InitGenesis 的参数字节、破坏 genesis replay） | 取值 0 或 [60, 2592000]；**open 时快照进挑战记录**（`resolution_deadline = opened_at + 封印窗 + 当刻 grace`），之后的参数变更不影响在飞挑战 |

**注意**：V1 挑战窗口已参数化（`challenge_window_seconds` / `EffectiveChallengeWindowSeconds`）；
Publish 时写入 `Candidate.challenge_deadline`，之后改参不移动在途 deadline。测试链可用
`update-params` 设 600s，无需改源码重编译。

## 7. 错误码（V1 段 40~78，号段不回收不复用）

40 pool not enabled · 41 pool paused · 42/43 not found/exists · 44 invariant ·
45/46/47 role key（46=F-2）· 48/49 invalid role/status · 50 invalid wire id ·
51 entitlement paused · 52/53/54 candidate · 55/56/57/58 auth（expired/replay/malformed/chain_id）·
59 invalid evidence type · 60 重复开挑战 · 61/62/63 challenge 查找/状态/outcome ·
64 窗口已关 · 65 pool mismatch · 66 available 不足 · 67/68 entitlement exists/not found ·
69 not claimable · 70 already claimed · 71 挑战阻断 approve · 72 beneficiary 必须 owner ·
73 freeze 一次性 · 74 freeze 时限 · 75 未冻结 · 76 双签独立 ·
**77 rule_version 不在白名单（F-1）** · **78 resolution hash 未绑定已结算裁决（DISMISSED/EXPIRED，F-3）** ·
**86 挑战裁决窗未到期（超时结算太早）**。
（1~39 为共享/legacy 段，其中 2/3/4/5/26/32/33/38/39 等仍被共享 Msg 使用。）

## 8. 事件

V1 事件统一携带：`chain_id / pool_id / entitlement_id / app_id / action / prev_state /
new_state / amount / denom / evidence_or_resolution_digest / height / tx_hash`
（`entitlement_events.go`）。observer 按 action+new_state 消费即可覆盖全部迁移。

## 9. legacy 弃用边界（对齐 DEC-09，链侧确认）

- **冻结时点：2026-10-26**（V1-ONLY-SCOPE-NOTICE 签收 + 链上四位 pause 置 true，
  tx `27B21114C48F690533F0CBB8EE65AF8F1FCD55D4BF458519721966127D490A9E` @301535）。
- 弃用清单（Msg/Query/State/Params/事件）以 `UPTICK-UGE-V1-ONLY-SCOPE-NOTICE-0.1.md` §2 为准。
- 代码保留冻结，不删状态、不回收错误码；历史 attest/settlement 记录（含 `21973b86…`
  SUSPENDED app、epoch 4 验收产物）仅作联调存档。
- 移除节奏：dev 链 `uptick_7777-1` 在**下一个链重置窗口**随重置清退（不走迁移）；
  主网/发布链由链团队评估后另行安排，移除前在模块文档标注
  `DEPRECATED: superseded by entitlement V1, do not build on`（待办）。
- `Migrate1to2` 已接线（ConsensusVersion 2），内容为 legacy pause 位 + 全量索引回填；它只在正式
  upgrade plan 触发时执行。裸替换二进制时由 EndBlocker 的 `EnsureChallengeIndexes` 完成同样的收敛，
  两者幂等且互相兼容（前者跑完会把重建标记置 `done`，后者随即成为 no-op）。

## 10. 实链运行快照（uptick_7777-1 @ test01，2026-10-26）

- **二进制**：`v0.5.1` / commit `2d141eb3d860…`；备份链 `~/bin/uptickd.bak-*` 可回滚。
- **Pool**：`pool-1` denom=auptick，funded=10,000,000 = available 7,000,000 + reserved 0 +
  paid 3,000,000，`module_balance` 一致，flow enabled，paused=false。
- **角色密钥**：pub-1 / apr-1 / rev-1 / risk-1 / aud-1 五把 Ed25519 ACTIVE（私钥仅本地
  `/tmp/uge-roles.json`）。
- **Application**：mall demo `4a250682…`（唯一 ACTIVE）；`21973b86…` SUSPENDED（legacy
  联调残留，tx `0394222A…` @299833）。
- **全流程取证**：C1/C2 CLAIMED（含 freeze 60s 分支）、C3 REVOKED（revoke 双签 + 资金回流）、
  C4（rule-9）PUBLISHED 留作 77 号阴性样本；阴性 8 项实锤
  （41/46/56/64/69×3/70/77/11）。
- **不变量**：`funded = available + reserved + paid` 每笔交易后自洽；invariant 注册于模块。

## 11. 相关文档索引

- 权威需求：UGE `13-UGE-AUTHORITATIVE-CHAIN-REQUIREMENTS-1.0.md`
- 集成契约（§8 对照动作、§9 绑定语义与 69/73 注记）：`UPTICK-UGE-V1-INTEGRATION-CONTRACT-0.1.md`
- 错误码/事件规范：`x/appmining/spec/app-mining-v1-events-errors-0.1.md`
- 合规检查（F-1/F-2/F-3）：`UPTICK-UGE-REQ13-COMPLIANCE-CHECK-0.1.md`
- 拍板往来：`UPTICK-UGE-DECISIONS-REQUEST-0.1.md`（DEC-01~05）、
  `UPTICK-UGE-CHAIN-ACTION-REPLY-0.1.md`（DEC-06~08）、
  `UPTICK-UGE-V1-ONLY-SCOPE-NOTICE-0.1.md`（DEC-09）、
  `UPTICK-UGE-BOND-SEMANTICS-REQUEST-0.1.md`（押金语义拍板请求，建议 DEC-11）
- 验收向量：`workorder/acceptance-vectors-0.4.json`、wire golden
  `app-mining-v1-wire-golden-1.0.json`
- 离线签名助手：`tools/ugeprov`（keygen/derive/sign/hash36）
