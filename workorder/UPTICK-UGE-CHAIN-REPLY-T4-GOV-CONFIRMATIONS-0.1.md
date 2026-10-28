# UPTICK-UGE-CHAIN-REPLY — T4 治理三项确认 0.1

**日期：** 2026-10-28  
**来自：** Uptick Chain  
**回复对象：** `UPTICK-CHAIN-T4-GOV-PROPOSAL-PACKAGE-DRAFT-0.1.md` §六 / 交接包 TASK-2  
**链仓基线：** `uge/v1.1`（含 DEC-10 `bde3cb6` 及后续入库提交）  
**发布链实测：** `uptick_7777-1` @ 34.177.82.11，块间隔约 **2s**（10s 内 +5 块）

---

## 确认项 ①：押金对价条款（定案）

与 as-built `releaseEntitlementChallengeDeposit` / T1c 单测一致，**写死进提案 metadata**：

| 裁决 | 押金流向 |
|---|---|
| **UPHELD**（挑战成立） | **全额退回 challenger**（`ChallengeEscrow` → challenger 账户；事件 `bond_flow=refund_in`） |
| **DISMISSED**（挑战驳回） | **没收入 entitlement pool**（`ChallengeEscrow` → `appmining_entitlement_pool` 模块账户，并计入该候选所属 pool 的 `funded` **与** `available`；事件 `bond_flow=slash_in`） |

**不是** `appmining_reward_pool`（旧 attest 挑战 bond 池）。V1 第三方挑战押金与 entitlement 池对账，避免与 POC reward_pool 混账。

---

## 确认项 ②：空串 = `"open"` 等价（定案）

- **是**：`Params.EffectiveChallengeOpenAccess()` 在 `challenge_open_access == ""` 时返回 `"open"`（`types/params.go`）。keeper OpenChallenge 只读该有效值。
- **MsgUpdateParams 写入 `"deposit"` 后即时生效**，无需重启、无需升级/迁移；下一笔 OpenChallenge 即按 deposit 扣押金。

---

## 确认项 ③：生产挑战窗口块数（定案）

| 项 | 值 |
|---|---|
| 发布链实测块速 | ≈ **2s/块**（`timeout_commit=2s`；2026-10-28 抽样 10s/+5） |
| 7 日历日等价块数 | `7 × 24 × 3600 / 2` = **302400** |
| **建议生产值** | **`challenge_window_blocks = 302400`**（≥ 7 天） |
| 提案形态 | **独立前置提案**（采纳 UGE 建议）：先恢复窗口，再提交 T4 强制凭证+押金+撤 pub-2，避免 T4 票被窗口争论拖住 |
| T4 JSON | 保持联调值 `500` **不得**误固化为生产；窗口恢复提案单独带完整 Params 快照只改 `challenge_window_blocks` |

联调说明：现网 `500` 块 × ~2s ≈ 1000s 量级；此前 e2e「+600s」口径若按 1.2s 估算，以本交接日实测 **2s** 为准折算生产值。

---

## 给 UGE 的下一步

1. 将 §①② 写入 T4 提案 metadata；§③ 另起 `challenge_window_blocks=302400` 提案 JSON。  
2. 升级提案包至 **1.0 EXECUTABLE** 后交链侧 submit（authority/gov 路径按现网）。  
3. 顺序建议：窗口恢复提案 PASS → T4 提案（UpdateParams 再 SetRoleKeyStatus）PASS。
