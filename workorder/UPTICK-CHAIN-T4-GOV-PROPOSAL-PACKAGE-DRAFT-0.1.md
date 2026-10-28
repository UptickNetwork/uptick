# UPTICK 链 T4 治理提案包草案（publisher-binding 下放完成切换）0.1

- 状态：DRAFT（供链侧复核后升级 1.0 EXECUTABLE）
- 日期：2026-10-28
- 关联：UPTICK-APPMINING-V1-PUBLISHER-BINDING-REVISION-0.1（R1-R5、T0-T4 迁移）、UPTICK-CHAIN-T1-EXECUTION-DIRECTIVE-0.1（D-C 决策）、UPTICK-UGE-T1-COMPLIANCE-REVIEW-0.1 §4
- 配套 JSON（可直接 submit-proposal）：`gov-proposal-t4-publisher-binding-enforcement.json`

## 一、一句话任务

以单个治理提案携带 2 条 authority Msg，完成 publisher-binding 修订的 T4 切换：**全局 publisher 键 pub-2 撤销（下放完成的链上标记）+ 发布强制评估凭证 + 第三方挑战切押金制**。

## 二、链上现状（2026-10-28 实测，LCD /params + /roles）

| 项 | 现值 | T4 目标 |
|---|---|---|
| credential_required | false | **true** |
| challenge_open_access | ""（空=默认 open） | **"deposit"** |
| challenge_deposit_amount / denom | "" / "" | **"500000000000000000000" / "auptick"**（对齐既有 challenge_bond_amount） |
| entitlement_allowed_assessment_versions | ["uge-assessment-0.1"] | 不变（allowlist 已就位） |
| entitlement_allowed_rule_versions | ["rule-1","uge-candidate-rules-0.1"] | 不变 |
| publisher 全局键 pub-2 | ACTIVE（version 2） | **REVOKED** |
| publisher app-scoped 键 mall-pub-0 | ACTIVE（绑定 mall 4a2506…） | 不动 |
| approver / reviewer / risk / audit | apr-2、rev-1、risk-1 ACTIVE | 不动 |
| 暂停位（attestation/challenge/settlement/gas_credit） | true（旧内核已冻结） | 不动 |

## 三、提案消息（2 条）

### Msg-1 MsgSetRoleKeyStatus —— 撤销全局 publisher 键

```
/uptick.appmining.v1.MsgSetRoleKeyStatus
authority = <gov module account>
role      = ENTITLEMENT_ROLE_PUBLISHER
key_id    = pub-2
status    = ROLE_KEY_STATUS_REVOKED
```

语义：**下放完成的链上标记**。pub-2 撤销后，无 app 绑定的全局 publisher 键不复存在，任何 publish 必须走 app-scoped 键（R1）+ 凭证（R3）。

### Msg-2 MsgUpdateParams —— 强制凭证 + 押金制

在**完整 Params 快照**（见 JSON，与上表现值逐字段一致）之上改三处：
- `credential_required = true`
- `challenge_open_access = "deposit"`
- `challenge_deposit_amount = "500000000000000000000"`、`challenge_deposit_denom = "auptick"`

注意：MsgUpdateParams 是全量替换语义，JSON 里必须携带全部 18 个字段的现值，只改目标三项；漏带字段会把现值清空。

## 四、前置条件（红绿灯，全部满足才可提交）

- **A. app-scoped publisher 键全覆盖**：所有会真实发布候选的 app 均已注册绑定键。当前仅 mall（4a2506…→mall-pub-0）；其余登记为 SHADOW/NON-FINANCIAL 的 app（UGE-RELEASE-APP-LEDGER-0.1）不上链发布，不受影响。若放量前新增真实 app，须先注册其绑定键再提交本提案。
- **B. 评估签名链路稳定**：uge-assess-0 已注册、allowlist 已写入、publish→approve→claim 全链路真网验收通过（2026-10-28 完成）。
- **C. 挑战窗口恢复生产值**：当前 `challenge_window_blocks=500` 为联调值（本次 e2e 实测 deadline=publish+600s）。**放量提案须同票或前置单独提案恢复生产窗口**；生产块数由链侧按发布链实测块速折算（口径：≥ 7 天等价块数）。本草案 JSON 保持 500 不变并显式标注，避免把联调值误固化为生产值。
- **D. D-C 决策兑现确认**：T1 任务书已定案「联调 open、T4 切 deposit」，本提案即其兑现动作，链侧无需再确认方向，只需确认押金对价条款（见 §六）。

## 五、执行方式（链侧）

```bash
~/bin/uptickd tx gov submit-proposal gov-proposal-t4-publisher-binding-enforcement.json \
  --from mykey --keyring-backend test --gas-prices 0.002auptick --yes
```

投票期结束 auto-exec：先撤键、后改参数（消息数组顺序即执行顺序；先撤 pub-2 会造成窗口期内「无键可发布」，故**建议顺序为先 MsgUpdateParams 再 MsgSetRoleKeyStatus**——JSON 已按此排序：参数先行，credential_required=true 生效后全局键仍有数分钟双轨期，再撤键收口）。

## 六、待链侧书面确认项（3 条）

1. **押金对价条款**：deposit 模式下押金流向——DISMISSED 时没收给谁（模块池/申损方）、UPHELD 时是否全额退回 challenger？R4 草案条款需在提案 metadata 里写死，避免执行后解释分歧。
2. **challenge_open_access 当前值语义**：链上现值为空串，行为等同 "open"（T1 实现）。请确认空串与 "open" 在 keeper 判断里等价，且 MsgUpdateParams 写入 "deposit" 后无需重启/迁移。
3. **生产挑战窗口块数**：给出发布链放量用 `challenge_window_blocks` 建议值（≥7 天等价），本包要么并入本提案一并投票，要么前置独立提案。

## 七、风险与回滚

| 风险 | 缓解 / 回滚 |
|---|---|
| credential_required=true 后某 app 无凭证发布被拒（80/81） | 预期行为；回滚 = 治理再置 false（单字段提案） |
| pub-2 撤销后应急需要全局代发 | 应急路径 = 注册新 version 全局键（违背下放目标，仅限事故恢复，用后即撤） |
| deposit 押金过高抑制第三方挑战 | 500 auptick 与既有 challenge_bond 对齐；后续可治理调低 |
| MsgUpdateParams 漏字段清空现值 | JSON 为完整 18 字段快照，提交前链侧用 query params diff 校验 |

## 八、验收（DoD）

- [ ] 提案 PASS 且消息全部执行成功（tx code 0）
- [ ] query params：credential_required=true、challenge_open_access="deposit"、deposit 非空
- [ ] query roles：pub-2 = REVOKED，mall-pub-0 仍 ACTIVE
- [ ] 负向验证：不带 credential 的 publish 请求被拒（错误码 80）
- [ ] mall app 用 mall-pub-0 + 凭证完成一次 publish 冒烟（证明合法路径未被误伤）
