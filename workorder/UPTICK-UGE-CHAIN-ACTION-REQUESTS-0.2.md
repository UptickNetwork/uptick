# UPTICK-UGE-CHAIN-ACTION-REQUESTS-0.2

> 日期：2026-10-28
> 发起：UGE 侧
> 接收：Uptick 链团队
> 状态：**链侧已 ACK**（DEC-10 定案方案 1）→ [UPTICK-UGE-CHAIN-ACTION-REPLY-REQ-D-SECP256K1-0.1.md](./UPTICK-UGE-CHAIN-ACTION-REPLY-REQ-D-SECP256K1-0.1.md)
> 背景：V1 发布自助化（T2/T2.1）已上线——console 提供评估凭证下载与本地签名脚本，发布签名模型为 Ed25519 WireProof（`UGE-APP-MINING-WIRE-V1` frame）。放量目标下，开发者普遍只有 MetaMask（secp256k1），无法产出 Ed25519 签名；现要求链侧授权内核扩展支持 secp256k1 直签（本文件 REQ-D）。

---

## REQ-D：publisher 角色键支持 secp256k1 / MetaMask 直签（DEC-10）

### 背景与动机

1. **开发者侧现状**：发布 `MsgPublishCandidate` 需要 publisher 角色键（链上注册 ACTIVE）对 WireProof 授权记录做 Ed25519 签名。开发者需在本机保管 Ed25519 私钥文件（PEM/32 字节 seed），用 UGE SDK `signAuthorization` 签名后广播。
2. **痛点**：区块链开发者的事实标准是 MetaMask（secp256k1）。要求他们额外生成、保管、按 app 轮换 Ed25519 密钥文件，是放量期最大的自助化摩擦点；且 Ed25519 私钥文件无钱包级 UX（无地址可视化、无硬件钱包生态）。
3. **密码学事实**：MetaMask 私钥是 secp256k1，密码学上**产不出** Ed25519 签名——这不是前端接入问题，必须链侧授权验签扩展。

### 两侧 as-built 事实（本需求的约束边界）

**wire 层（链仓 wirev1 + UGE sdk/javascript/appmining-v1-wire.mjs，两侧互验一致）**：

- 授权记录 = `COMMON(protocol_version, operation, role, chain_id, pool_id, key_id, nonce, expires_at)` + 操作字段，TLV 长度前缀编码为 frame `UGE-APP-MINING-WIRE-V1`，签名对象就是该 frame 的原始字节。
- **授权记录字段集是冻结的**（严格 key 集合校验，多一字段/少一字段即拒绝）：新增任何字段都会破坏现有 Ed25519 路径与全部黄金向量。
- WireProof proto 结构 = `{key_id, nonce, expires_at, signature_base64}`，**不携带公钥**——链上按 `key_id` 查注册键验签。
- 角色-操作绑定：PUBLISH=publisher / RESOLVE=reviewer / APPROVE=approver / FREEZE=risk / REVOKE=risk+audit。

**键注册**：`SetRoleKeyStatus` 现仅支持 Ed25519 公钥（32 字节）；链上现状全部角色键均为 Ed25519。

### 需求（链团队评估，UGE 推荐方案 1）

**方案 1（推荐）：角色键类型扩展 + EIP-191 personal_sign 验签**

- **D1 键类型**：`RoleKey` / `SetRoleKeyStatus` 增加 key type 字段（建议枚举：`ED25519 = 0`（默认，现值不变）、`SECP256K1_EVM_ADDRESS = 1`）。类型 1 的注册值 = **20 字节 EVM 地址**（hex 展示，`0x` 前缀可选）——因 secp256k1 用可恢复签名（recoverable），链上 recover 出的地址与注册地址比对即可，注册侧无需保管公钥。
- **D2 签名域**：类型 1 键的 WireProof `signature_base64` = 对 **wire frame 原始字节**做 EIP-191 personal_sign 域哈希后 secp256k1 签名：`hash(0x19Ethereum Signed Message:\n<len(frame)> ‖ frame)`，签名为 65 字节 `(r, s, v)`（v ∈ {27,28} 或 EIP-155 形态，以实现定，文档写死一种）。
- **D3 验签分支**：keeper 按 key_id 查注册键类型分流——Ed25519 → 现行 64 字节 `crypto/ed25519.Verify`（零改动）；SECP256K1_EVM_ADDRESS → 65 字节 ecrecover → 地址比对。**按注册类型推断方案，Msg proto 与授权记录字段集零改动**（这是推荐方案 1 而非 Msg 层加 `signature_scheme` 字段的原因：字段集冻结 + 现有向量兼容）。
- **D4 malleability**：强制 low-s（EIP-2 风格，`s ≤ n/2`），高 s 拒绝。
- **D5 范围限定**：仅 **publisher（PUBLISH）** 开放类型 1 键；reviewer/approver/risk/audit 保持 Ed25519（UGE 内部角色，无钱包需求，收敛攻击面与实现面）。
- **D6 不变量（UGE 侧已确认不受影响）**：授权记录字段集、frame 编码、`candidateHash`/`entitlement_id` 语义、credential 一次性占用、nonce/expires_at 校验、现有 Ed25519 黄金向量与 UGE 回归（66 例）全部不变。
- **D7 域分离说明**：submitter 的 Cosmos tx 签名（secp256k1，SIGN_MODE_DIRECT）与 WireProof 的 EIP-191 personal_sign 域不同，无冲突；**不强制** publisher 地址 = submitter 地址（可专用发布热钱包）。uptick 链为 EVM 兼容（ethermint 系），同一把 MetaMask 私钥天然可派生 Cosmos 账户签 tx，开发者一把钥匙完成 tx 签名 + WireProof 签名。
- **D8 错误码**：新增错误码接现有 40~78 序列之后（建议：签名长度不符 / 键类型与签名不符 / recover 地址不匹配 / high-s），UGE 登记进 `sdk/javascript/appmining-v1-errors.mjs` 与 V1 对接文档。
- **D9 查询面**：roles 查询返回 key_type，供 UGE console 判定「该键可否钱包直签」。

**方案 2（备选，可作为方案 1 的后续增强）**：EIP-712 typed data 直签——钱包内结构化展示授权内容（UX 更好），但链侧需复刻 typed-data 二级编码并解决 EVM chainId（7777）与 `chain_id`（`uptick_7777-1`）的映射，实现与审计成本显著更高。建议首版 personal_sign，EIP-712 立项后置。

**方案 3（UGE 已排除，列出仅备查）**：Msg 层加 `signature_scheme` 字段——破坏授权记录字段集冻结性，且 key 类型已能推断方案，否决。

### 链侧实施点清单（供评估工作量）

1. proto：`SetRoleKeyStatus` / 角色键存储结构加 key type 字段（protojson 兼容：缺省 = 0 = Ed25519，旧数据零迁移）。
2. keeper：publish 验签分流 + ecrecover（ethermint 栈已有 secp256k1 依赖，无新增依赖）。
3. 错误码 + 事件：沿用现有事件结构，验签失败路径走新错误码。
4. spec 增补：`x/appmining/spec/` wire spec 增「签名方案」节；UGE 出 secp256k1 黄金向量（personal_sign 路径，SDK 侧生成后对链互验）。
5. 兼容性验收：现有全部 Ed25519 向量与在链角色键不受影响（回归）。

### UGE 侧配套（UGE 自排期，**待 DEC-10 ACK 后实施**）

- SDK：`verifyAuthorizationEvm(record, signatureBase64, expectedEvmAddress)`（ecrecover 本地复核，发布前 fail-fast）；`buildV1PublishCandidate` 的 wireProof 本地校验按签名长度分流（64=Ed25519 / 65=secp256k1）。
- console ③ 发布区：MetaMask 连接 → personal_sign 发布（签名对象 = wire frame bytes）→ 广播，全程私钥不出钱包。
- 签名工具与对接文档同步更新。

### 验收标准

- [ ] 链侧书面回复接受方案 1（或提出修订），DEC-10 落跟踪
- [ ] 类型 1 键注册 → personal_sign 发布 → 链上验签通过，端到端 demo（UGE 侧出向量与脚本）
- [ ] 现有 Ed25519 全量向量回归通过（链侧跑放行门槛不受影响）
- [ ] D8 新错误码登记进 UGE 错误码表

---

## 附：相关决策与文档交叉引用

- DEC-01（approve 双路径）、DEC-09（V1-only 范围）：本需求在 V1 模型（Pool/Candidate/Challenge/Entitlement/Claim）内，不触及旧 attest 路径。
- #13 治理提案包（`UPTICK-CHAIN-T4-GOV-PROPOSAL-PACKAGE-DRAFT-0.1.md`）：credential_required / challenge_open_access 等参数与本需求无冲突；若 DEC-10 落地，publisher 键生命周期治理（注册/撤销）沿用现有 `SetRoleKeyStatus`。
- UGE 侧流程文档：`docs/workorder/` V1 系列；SDK wire 实现以 `sdk/javascript/appmining-v1-wire.mjs` 为准（与链仓 wirev1 互验一致）。
