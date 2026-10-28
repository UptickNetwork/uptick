# Uptick 链侧 T1 决策记录 0.1

> 来源：UGE `UPTICK-CHAIN-T1-EXECUTION-DIRECTIVE-0.1` §2（D-A / D-B / D-C）  
> 日期：2026-10-28  
> 基线分支：`uge/v1.1`（T1 R1–R5 基线 `185a0d9`；本决策落点提交见文末）

---

## D-A：credential_id = 永久一次性

**定案：** `HasCredentialID` 在候选 `REJECTED` 后**仍不释放**。同一凭证对同一 pool 永久一次性。此为最终语义，非 T1 临时态。

**落点：**
- `keeper/assessment.go` — `verifyAssessmentCredential` 注释
- `keeper/store.go` — `HasCredentialID` 注释
- `types/keys.go` — `CredentialIndexKey` 注释
- 向量：`CREDENTIAL_ID_HELD_AFTER_REJECTED`（`acceptance-vectors-0.5.json` / `TestVector_CREDENTIAL_ID_HELD_AFTER_REJECTED`）
- 错误码 81 `ErrAssessmentCredentialReuse` 保持不变

---

## D-B：AssessmentSigner 注册通道必须补齐

**定案：** 提供 authority 门禁的 Msg + CLI（方案 A+B 合并：Msg 为真源，CLI 广播 Msg）。

**落点：**
- Msg：`MsgRegisterAssessmentSigner` / `MsgSetAssessmentSignerStatus`
- CLI：`register-assessment-signer` / `set-assessment-signer-status`
- key_id：`^[a-z0-9][a-z0-9-]{2,31}$`（`ValidateAssessmentSignerKeyID`）
- 轮换语义对齐 RoleKey（同 pubkey 幂等；异 pubkey 冲突须先 revoke；revoke 后可重注册）
- 文档：`x/appmining/spec/assessment-signer-registration-0.1.md`
- 单测：`keeper/assessment_signer_test.go`

---

## D-C：`challenge_open_access` 默认 `open` 保留

**定案：** T1/T2 联调期保持默认 `open`，**本次不改默认值代码**。

**T4 治理计划（部署放行前）：**
1. 经 `MsgUpdateParams`（CLI 已存在）将发布链参数切为 `challenge_open_access=deposit`
2. 配置 `challenge_deposit_amount` / `challenge_deposit_denom`
3. 可选同步 `credential_required=true` 并撤销全局 publisher（见向量 T4 case）

---

## 生效提交

| 项 | 值 |
|---|---|
| 基线（R1–R5） | `185a0d9` |
| 本决策包（S-1…S-5） | `1fd9c097a5f81f5acbd99f0265583a19d66e2407` |
| 发布提交（含 proto FileDescriptor 修复） | `66c9f7873dae87958883f92fa0470ad940c241b7` |
| 发布 tag | `v0.5.1-t1` |
