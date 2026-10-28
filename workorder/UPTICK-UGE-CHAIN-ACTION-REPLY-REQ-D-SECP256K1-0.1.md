# UPTICK-UGE-CHAIN-ACTION-REPLY — REQ-D secp256k1 / MetaMask 直签 0.1

**日期：** 2026-10-28  
**来自：** Uptick Chain（链侧）  
**回复对象：** [UPTICK-UGE-CHAIN-ACTION-REQUESTS-0.2.md](./UPTICK-UGE-CHAIN-ACTION-REQUESTS-0.2.md)（UGE 真源同路径）  
**链仓基线：** `uge/v1.1` @ `174df34`（运行二进制 tag `v0.5.1-t1.1` = `e6b35e1`）  
**决策编号：** **DEC-10**（纳入跟踪）  
**权威边界：** 文档 13；DEC-01 / DEC-09 不变；Wire 授权记录字段集与 `UGE-APP-MINING-WIRE-V1` frame **冻结零改动**

---

## 0. 回执总表

| 项 | 链侧结论 |
|---|---|
| 方案 1（key type + EIP-191 personal_sign） | **接受** |
| 方案 2（EIP-712） | **后置**（不进本版） |
| 方案 3（Msg `signature_scheme`） | **同意否决** |
| D1–D9 | **全部采纳**；实现口径见 §2（少量定点澄清） |
| DEC-10 | **立档跟踪**；本文件即 ACK |
| 错误码 | **82–85**（§3） |
| 现有 Ed25519 向量 / 在链键 | **承诺零回归**（缺省 `key_type=0`） |

---

## 1. 结论

链侧**接受方案 1**作为 DEC-10 定案：在 **不改 WireProof proto、不改授权记录字段集、不改 frame 编码** 的前提下，为 **publisher** 角色键增加 `SECP256K1_EVM_ADDRESS` 类型，验签走 EIP-191 `personal_sign` + ecrecover + 地址比对；其余角色保持 Ed25519。

动机与 UGE 一致：MetaMask 无法产出 Ed25519；放量自助化必须链侧验签扩展。

---

## 2. 实现口径（相对 REQ-D 的定点澄清）

### 2.1 Proto / 存储（D1）

| 项 | 定案 |
|---|---|
| 枚举名 | `RoleKeyType`：`ROLE_KEY_TYPE_ED25519 = 0`（自定义名 `RoleKeyTypeEd25519`）、`ROLE_KEY_TYPE_SECP256K1_EVM_ADDRESS = 1`（`RoleKeyTypeSecp256k1EvmAddress`） |
| 落点 | `RoleKey.key_type`（字段 **8**，缺省 0）；`MsgRegisterRoleKey.key_type`（同语义） |
| `SetRoleKeyStatus` | **不加** key_type（只改状态；类型在注册时固定） |
| 类型 1 的 `pubkey` 载荷 | **复用现有 `bytes pubkey`**：存 **20 字节 raw EVM 地址**（非 33B 压缩公钥）。CLI/JSON 可接受 `0x`+40 hex 或 40 hex，入库前规范化为 20B |
| 旧数据 | proto3 缺省 0 = Ed25519；**零迁移** |
| 嵌入 publisher | `MsgRegisterApplication.publisher_pubkey` **本版仍仅 Ed25519（类型 0）**；类型 1 请走 `MsgRegisterRoleKey`（避免同字段双语义） |

### 2.2 签名域与 v（D2）

- 待签字节 = **现有** `wirev1` frame 原始字节（`UGE-APP-MINING-WIRE-V1`），**不**改字段集。
- 哈希：`keccak256("\x19Ethereum Signed Message:\n" ‖ decimal(len(frame)) ‖ frame)`（go-ethereum `accounts.TextHash` 同构）。
- `signature_base64`：标准 base64（带 padding），解码后 **恰好 65 字节** `[R‖S‖V]`。
- **V 定死为 `{27,28}`**（经典 personal_sign）。**拒绝** EIP-155 形态 `v = chainId*2+35/36`，避免与 `uptick_7777-1` / EVM `7777` 映射纠缠。
- 验签：`ecrecover(hash, sig)` → 20B 地址，与注册地址 **逐字节相等**（大小写无关，内部一律 raw bytes）。

### 2.3 验签分流（D3）

落点：`keeper.verifyAndConsumeWireAuth` + `wirev1` 增补 `VerifyAuthorizationSecp256k1`（或等价）：

| `RoleKey.key_type` | 签名长度 | 算法 |
|---|---|---|
| `0` Ed25519 | 64 | 现行 `ed25519.Verify`（零行为变化） |
| `1` secp256k1 地址 | 65 | TextHash + ecrecover + 地址比对 + low-s |

**非 publisher 角色**注册或使用 `key_type=1` → 拒绝（D5）。  
PUBLISH 路径以外的 WireProof（RESOLVE/APPROVE/…）遇到类型 1 键 → 拒绝（即使误注册）。

### 2.4 malleability（D4）

强制 EIP-2 **low-s**：`s ≤ secp256k1n/2`；高 s → 新错误码（§3）。不接受签名延展性变体。

### 2.5 范围（D5）

仅 **`EntitlementRolePublisher` + operation PUBLISH`**。reviewer/approver/risk/audit：**禁止类型 1**。

### 2.6 不变量（D6）— 链侧确认

授权记录字段集、frame、`candidateHash` / `entitlement_id`、credential 一次性、nonce/expires、现有 Ed25519 黄金向量：**全部不改**。DEC-10 仅增加验签分支。

### 2.7 域分离（D7）

接受：Cosmos tx 签名（SIGN_MODE_DIRECT / eth_secp256k1 账户）与 WireProof EIP-191 **域不同**；**不强制** publisher 地址 = submitter。同一 MetaMask 钥可派生两者，属产品便利非共识约束。

### 2.8 查询（D9）

`QueryRoles` / `RoleKey` 响应带出 `key_type`（protojson 数字或枚举名均可；UGE console 以字段存在为准）。无需新 query RPC。

---

## 3. 错误码（D8）— 接 81 之后

| 码 | 符号（拟） | 语义 |
|---|---|---|
| **82** | `ErrWireProofSigLength` | 签名解码后长度与键类型不符（期望 64 或 65） |
| **83** | `ErrRoleKeyTypeUnsupported` | 该角色/操作不允许此 `key_type`（含非 publisher 类型 1） |
| **84** | `ErrWireProofRecoverMismatch` | ecrecover 地址 ≠ 注册 20B 地址 |
| **85** | `ErrWireProofHighS` | secp256k1 签名 high-s（非 low-s） |

既有 `ErrUnauthorized`（验签失败兜底）对 Ed25519 路径保持；类型 1 优先走 82–85 以便 UGE SDK 映射。  
请 UGE 登记 `sdk/javascript/appmining-v1-errors.mjs` 与对接文档。

---

## 4. 工作量与排期（评估）

| # | 工作项 | 规模（估） |
|---|---|---|
| 1 | proto：`RoleKeyType` + `RoleKey`/`MsgRegisterRoleKey` 字段；**全量 regen FileDescriptor**（吸取 FIELD18 教训） | S |
| 2 | 注册校验：类型 0=32B Ed25519；类型 1=20B 地址；角色门禁 | S |
| 3 | `wirev1` + keeper 验签分流；go-ethereum / cosmos-evm 已有 secp256k1，无新依赖 | M |
| 4 | 错误码 82–85；CLI `register-role-key` 增加 `--key-type` | S |
| 5 | 单测：类型 1 正路径 / high-s / 错地址 / 错长度 / 非 publisher 拒绝；Ed25519 全回归 | M |
| 6 | spec：`x/appmining/spec/` 增「签名方案」节；等 UGE secp256k1 黄金向量互验 | S（向量等 UGE） |
| 7 | 发版部署（tag + 发布链替换） | S |

**合计：** 约 **1.5–2.5 人日**（含测与发版），阻塞依赖：UGE 提供 personal_sign 黄金向量（可与链并行）。

**实施触发：** 本 ACK 即启动链侧实现；UGE SDK/console 可按 §2/§3 并行，无需再等二次确认。

---

## 5. 验收对照（相对请求 §验收标准）

| 标准 | 状态 |
|---|---|
| 链侧书面接受方案 1，DEC-10 落跟踪 | **本文件完成** |
| 类型 1 注册 → personal_sign 发布 → 链上通过（E2E） | 待实现 + UGE 向量/脚本 |
| Ed25519 全量向量回归 | 待实现 PR 放行门槛 |
| D8 错误码进 UGE 表 | 码号已定 82–85；UGE 登记 |

---

## 6. 非目标（本版不做）

- EIP-712 typed data（方案 2）
- Msg / 授权记录加 `signature_scheme`
- reviewer 等非 publisher 角色的钱包键
- `RegisterApplication` 内嵌类型 1 publisher
- 变更 credential / challenge_open_access / T4 治理包

---

## 7. 给 UGE 的并行清单

1. 冻结 personal_sign 黄金向量（frame → TextHash → 65B sig → 期望地址），与链仓互验。  
2. SDK：`verifyAuthorizationEvm` + `buildV1PublishCandidate` 按签名长度分流。  
3. 错误码表登记 **82–85**。  
4. console：MetaMask `personal_sign(frameBytes)` → 填入 `WireProof.signature_base64`。  

链侧实现完成后回报：**HEAD + tag + 向量互验结果 + 发布链版本字符串**。
