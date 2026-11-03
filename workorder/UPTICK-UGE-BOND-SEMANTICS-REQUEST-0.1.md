# UPTICK-UGE-BOND-SEMANTICS-REQUEST-0.1

> 日期：2026-11-03
> 发起方：Uptick 链团队
> 接收方：UGE
> 性质：**V1 挑战押金语义拍板请求**（审计编号 P2-2，建议登记为 DEC-11，编号以 UGE 台账为准）
> 代码基线：`uge/v1.1` commit `70ae1da`（含 2026-11-03 审计修复批）
> 链侧拍板：**方案 A —— 维持现状**（本文请求 UGE 书面确认，将其定为有意设计而非 drift）

---

## 1. 事项一句话

V1 `ResolveEntitlementChallenge` 裁决为 **DISMISSED**（挑战不成立）时，挑战押金**不退还
挑战者**，而是罚没进**被挑战候选所在的 EntitlementPool**，且 pool 的 `funded` 与
`available` 同时增加。链侧确认此为有意设计，请 UGE 确认接受其经济语义；若 UGE 对
`funded` 口径有异议，需在合并 main 前提出（改动窗口）。

## 2. 现行实现事实（uge/v1.1 @ 70ae1da）

**收取**（`x/appmining/keeper/entitlement_p1.go:240-262`，OpenChallenge）：

- 挑战开放模式参数化：`challenge_open_access = none | open | deposit`，仅 `deposit` 模式收押金；
  金额/denom 来自 `challenge_deposit_amount` / `challenge_deposit_denom`。
- 硬约束：押金 denom 必须等于候选所在 pool 的 denom（:245-249），代码注释原文：
  > "Deposit must match the candidate pool denom so confiscation can credit that pool
  > without cross-denom bookkeeping."
- 押金托管进 `appmining_challenge_escrow` 模块账户。

**结算**（同文件 :344-356 → `releaseEntitlementChallengeDeposit` :384-422）：

| 裁决结果 | 候选状态 | 押金流向 | 事件标记 |
|---|---|---|---|
| UPHELD（挑战成立，候选被拒） | candidate → REJECTED | escrow → **退还挑战者** | `bond_flow=refund_in` |
| DISMISSED（挑战不成立，候选保留） | candidate 保持 PUBLISHED | escrow → EntitlementPool 模块账户；pool `Funded`+`Available` **同时增加**（:407-416） | `bond_flow=slash_in` |

**账目自洽**：池不变量 `funded = available + reserved + paid`（`types/pool.go`）在双加后成立，
已有测试覆盖两种流向（`challenge_deposit_test.go`）。**此条不是账目缺陷，是经济语义问题。**

## 3. 两代模型对比（重要：方向相反）

| | legacy 挑战流（已冻结，DEC-09） | V1 entitlement 流（现行） |
|---|---|---|
| 挑战成功（UPHELD） | **罚没** → 全网 RewardPool（`msg_server.go:238-240`） | 退还挑战者 |
| 挑战失败（DISMISSED） | 退还挑战者 | **罚没** → 被挑战候选所在池 |

legacy 是"赢家也付费"（押金 = 挑战成本价，罚没款补偿网络）；V1 是标准"输家买单"。
两代罚没方向相反，本请求即要把 V1 行为定为**有意变更**，避免日后被当 drift 追问。

## 4. 业界对比（2026-11-03 调研）

| 链 / 协议 | 罚没款去向 | 要点 |
|---|---|---|
| Cosmos SDK x/slashing | **销毁** | slashed tokens 退出流通 |
| Cosmos gov 押金 | 通过/拒绝退还；仅 **vetoed** 销毁 | "正常失败不罚，明显恶意才烧" |
| Ethereum PoS | **大部分销毁** + 举报/出块者小分成 | |
| Polkadot | **Treasury**（2026-03 起入 DAP 池） | 不销毁、留 27/28 天 unapplied 宽限、**治理可回滚错误罚没** |
| UMA Optimistic Oracle（与 V1 挑战机制最同构） | 败方押金 **50% 给胜方 + 50% 协议金库** | 对半分防 self-dispute 洗押金 |
| Polygon | treasury | |

**业界归纳**：销毁 / 进协议金库（可追溯可回滚）/ 败方赔胜方+协议分成 / 进业务池补贴该池
后续用户（**Uptick V1 独有**）。没有主流链把罚没款定向补贴"被挑战标的所在的业务池"；
最接近的 UMA 给的是**挑战赢家**，而 V1 给的是与挑战胜负无关的该池后续 claimant。

## 5. 链侧已识别并接受的语义点（方案 A 前提）

1. **失败挑战实质补贴该池后续 claimant**：押金入 `available` 后 fungible，与干净资金不可区分；
   含自博弈场景（app 诱导挑战后故意输，押金转一圈回池——无套利空间，成本=押金+gas，仅语义怪）。
2. **`funded` 口径含罚没款**：`funded` 本义为外部出资人注入（`FundEntitlementPool` 唯一写入方）；
   罚没款计入后，以 funded 为口径的对账/报表会含罚没部分。只加 available 不加 funded 会破坏
   不变量，实现选择了"不变量成立"优先。
3. **与 legacy 行为相反**：见 §3，需作为有意变更记录。
4. **不可回滚**：入池即 fungible，无 Polkadot 式 unapplied 宽限与治理回滚通道。

## 6. 请 UGE 确认的问题（Q1~Q4）

- **Q1**：DISMISSED 押金进被挑战候选所在池（方案 A）——确认接受？
- **Q2**：若接受，是否接受罚没款计入 `funded`（外部出资口径含罚没款）？
- **Q3**：是否接受"失败挑战实质补贴该池后续 claimant"的经济效果（含自博弈场景）？
- **Q4**：V1 与 legacy（进 RewardPool）的行为差异，确认为**有意变更**并记录在案？

## 7. 链侧建议与后续动作

链侧倾向：**Q1~Q4 全部确认**（方案 A）。理由：代码注释本意如此、账目自洽、测试已覆盖、
押金 denom=pool denom 的约束即为该设计服务、改动收益低于风险。

UGE 确认后链侧动作：

- 本文档登记为 DEC-11（编号待 UGE 台账确认）；
- `UPTICK-UGE-V1-MODEL-RECORD-0.1.md` 已同步补押金语义小节（§2.1）；
- 若 UGE 对 Q2 有异议，备选方案 B（新增 `confiscated` 独立口径字段，不入 funded）需动 proto
  并重走 FileDescriptor 生成纪律，须在合并 main 前决策。

---

*附：审计来源 `uge/v1.1` 安全审计报告 P2-2；押金相关测试 `challenge_deposit_test.go`
（UPHELD 退款 / DISMISSED 进池两用例）。*
