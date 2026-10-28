# AssessmentSigner 注册通道 0.1

**模块：** `uptick.appmining.v1`  
**决策：** UGE T1 D-B（`UPTICK-CHAIN-T1-EXECUTION-DIRECTIVE-0.1` §2）  
**权限：** `params.authority`（与 `MsgRegisterRoleKey` 同门禁）

## CLI

```bash
# 注册 / 轮换（撤销后可同 key_id 重新注册）
uptickd tx appmining register-assessment-signer <key-id> <pubkey-hex> <version> \
  --from <authority> --chain-id <id> -y

# 暂停或撤销
uptickd tx appmining set-assessment-signer-status <key-id> active|paused|revoked \
  --from <authority> --chain-id <id> -y
```

Msg：`MsgRegisterAssessmentSigner` / `MsgSetAssessmentSignerStatus`。

## key_id 约束

正则：`^[a-z0-9][a-z0-9-]{2,31}$`（长度 3–32，小写字母数字，允许内部连字符）。  
由 `ValidateAssessmentSignerKeyID` 在 `ValidateBasic` 强制。

## 轮换语义（对齐 RoleKey）

| 现状 | 再 Register 同 key_id | 结果 |
|---|---|---|
| ACTIVE + 同 pubkey | 幂等成功 | 不改状态 |
| ACTIVE + 异 pubkey | `ErrRoleKeyConflict` | 须先 revoke |
| REVOKED | 允许 | 写入新 pubkey，状态 ACTIVE |
| PAUSED + 同 pubkey | 覆盖为 ACTIVE | 与 RoleKey 一致 |

REVOKED 后不可经 `SetAssessmentSignerStatus` 直接改回 ACTIVE；须走 Register。

## 验签路径

`PublishCandidate` → `verifyAssessmentCredential` 仅接受 `Status=ACTIVE` 的 signer；pubkey 用于 Ed25519 验签冻结评估凭证 wire。

## 事件

- `assessment_signer_registered`：`authority`, `key_id`, `version`, `height`
- `assessment_signer_status_changed`：`authority`, `key_id`, `old_status`, `new_status`, `height`
