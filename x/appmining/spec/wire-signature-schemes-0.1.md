# WireProof 签名方案 0.1（UGE DEC-10）

**模块：** `uptick.appmining.v1`  
**冻结约束：** 授权记录字段集与 `UGE-APP-MINING-WIRE-V1` frame 编码不变；方案由注册键 `RoleKey.key_type` 推断，WireProof proto **无** `signature_scheme` 字段。

## 键类型

| `RoleKeyType` | 值 | `pubkey` 载荷 | 签名 | 允许角色 |
|---|---|---|---|---|
| `ED25519` | 0（默认） | 32B Ed25519 公钥 | 64B Ed25519，标准 base64 | 全部角色 |
| `SECP256K1_EVM_ADDRESS` | 1 | 20B EVM 地址 | 65B `r‖s‖v`，EIP-191 personal_sign | **仅 publisher** |

旧键缺省 `key_type=0`，零迁移。

## 类型 1 验签

1. `frame = AuthorizationBytes(record)`（与 Ed25519 相同）  
2. `hash = keccak256("\x19Ethereum Signed Message:\n" ‖ decimal(len(frame)) ‖ frame)`  
3. `signature_base64` → 65B；`v ∈ {27,28}`；强制 low-s（EIP-2）  
4. `ecrecover` → 地址与注册 20B 逐字节相等  

## 错误码

| 码 | 符号 | 语义 |
|---|---|---|
| 82 | `ErrWireProofSigLength` | 长度 / v 非法 |
| 83 | `ErrRoleKeyTypeUnsupported` | 角色或操作不允许该类型 |
| 84 | `ErrWireProofRecoverMismatch` | recover 地址不匹配 |
| 85 | `ErrWireProofHighS` | high-s |

## CLI

```bash
uptickd tx appmining register-role-key publisher <key-id> <0x地址或40hex> <version> \
  --key-type secp256k1-evm --from <authority>
```

`QueryRoles` 返回的 `RoleKey` 含 `key_type`。
