# UPTICK-UGE-DEC10-INTEROP-CONFIRMATION-0.1

> 日期：2026-10-28
> 发起：UGE 侧
> 主题：**DEC-10（publisher secp256k1 / MetaMask 直签）链侧实现确认 + 跨实现向量互验**
> 链侧实现：链仓 `uge/v1.1` @ `bde3cb6`（feat(appmining): DEC-10 publisher secp256k1 EVM-address WireProof）
> 结论：**确认通过**。实现与 REQ-D 方案 1 及链侧 REPLY §2 定点澄清完全一致；UGE→链黄金向量互验 **3/3 PASS**。

---

## 1. 链侧实现审查（逐项对照 REQ-D D1–D9）

| 项 | 要求（REQ-D） | 链侧实现（bde3cb6） | 结论 |
|---|---|---|---|
| D1 键类型 | `RoleKeyType` 枚举，0=Ed25519 缺省零迁移，1=20B EVM 地址 | `types.proto` 枚举 + `RoleKey.key_type`(字段8) + `MsgRegisterRoleKey.key_type`；`ValidateEvmAddressBytes` 20B 强校验；`NormalizeEvmAddressInput` 接受 20B/40hex/0x+40hex | ✅ |
| D2 签名域 | EIP-191 personal_sign over wire frame 原始字节，65B (r,s,v)，v∈{27,28} | `wirev1/auth_secp256k1.go`：`accounts.TextHash`（与 EIP-191 同构）+ v 严格 {27,28}（拒绝 EIP-155）+ base64 canonical 复核 | ✅ |
| D3 验签分流 | 按注册键类型推断，Msg/授权记录零改动 | `keeper/wire_auth.go` switch `rk.KeyType`；Ed25519 路径零变化；WireProof proto 未动 | ✅ |
| D4 low-s | 强制 EIP-2 | `s ≤ n/2` 检查，高 s 拒绝 | ✅ |
| D5 范围 | 仅 publisher + PUBLISH | 注册侧 `ValidateRoleKeyMaterial`（非 publisher 拒绝类型 1）+ 验签侧 `role != publisher || operation != PUBLISH` 双重拦截 | ✅ |
| D6 不变量 | 字段集/frame/向量零改动 | 仅增验签分支；UGE 侧 SDK 全量回归 42/42 绿 | ✅ |
| D8 错误码 | 接 78 后 | 82 `ErrWireProofSigLength` / 83 `ErrRoleKeyTypeUnsupported` / 84 `ErrWireProofRecoverMismatch` / 85 `ErrWireProofHighS` | ✅ |
| D9 查询 | roles 带 key_type | proto 随 RoleKey 序列化带出 | ✅ |
| CLI | register-role-key 支持 | `--key-type`（`ed25519`/`secp256k1-evm`），地址 hex 归一化 | ✅ |

另核：`SetRoleKeyStatus` 未加 key_type（类型注册时固定，符合 REPLY §2.1）；`MsgRegisterApplication.publisher_pubkey` 仍仅 Ed25519（符合 REPLY 非目标）。

## 2. 链侧测试

- `go test ./x/appmining/...` 全绿（含新增 `auth_secp256k1_test.go` 77 行 + `secp256k1_publisher_test.go` 114 行：正路径 / high-s / 错地址 / 非 publisher 拒绝）。
- UGE 侧回归：`node --test "sdk/javascript/*.test.mjs"` **42/42 pass**（错误码表扩展 40–85 后同步更新边界断言）。

## 3. 跨实现黄金向量互验（本确认的核心证据）

**UGE 生成**（`scripts/gen-secp256k1-wire-golden-vectors.mjs`）：SDK `authorizationBytes` 产 `UGE-APP-MINING-WIRE-V1` frame → EIP-191 personal_sign 哈希 → @noble/curves secp256k1 签名（65B r‖s‖v）。向量文件：`sdk/javascript/golden-vectors/appmining-v1-wire-secp256k1-golden-1.0.json`（链仓副本 `x/appmining/spec/` 同名，待链侧入库）。

**链侧验证**（`x/appmining/types/wirev1.VerifyAuthorizationSecp256k1`，临时 Go 测试跑通后删除）：

| 向量 | 期望 | 链侧结果 |
|---|---|---|
| `publish_low_s_v27_v28_canonical`（正路径，v=28，low-s） | ok | ✅ PASS（verified） |
| `publish_high_s_rejected`（lowS:false 产出，已确认 s>n/2） | high_s | ✅ PASS（拒绝，reason 精确匹配） |
| `publish_sig_length_64_rejected`（64B Ed25519 形态签名） | sig_length | ✅ PASS（拒绝，reason 精确匹配） |

**3/3 全 PASS**：UGE Node 侧与链 Go 侧的 frame 编码、EIP-191 域哈希、签名格式、reason token 完全互通。测试用私钥为固定测试值（仅向量复现用，严禁生产）。

## 4. UGE 侧已完成

- 错误码 82–85 登记进 `sdk/javascript/appmining-v1-errors.mjs`（含溯源注释）+ 测试同步（`appmining-v1-errors.test.mjs` 边界 40–85、新增 82–85 as-built 断言）。
- 黄金向量生成器 + 向量文件入库（UGE 提交见 git log；向量 JSON 已放链仓 spec 待链侧随下个提交入库）。

## 5. UGE 侧剩余（不阻塞链侧发版）

1. **SDK**：`verifyAuthorizationEvm`（ecrecover 本地复核）+ `buildV1PublishCandidate` wireProof 本地校验按签名长度分流（64=Ed25519 / 65=secp256k1）。
2. **console**：③ 发布区 MetaMask 连接 → `personal_sign(frameBytes)` → 发布按钮。
3. **真网 e2e**：待链侧发版（tag + 发布链部署）后：`register-role-key publisher <key-id> <0x地址> --key-type secp256k1-evm` → console MetaMask 发布全链路。
4. 链侧回报四元：HEAD + tag + 向量互验结果 + 发布链版本字符串（REPLY §7 要求，本轮互验已完成第 3 项的 UGE 侧）。
