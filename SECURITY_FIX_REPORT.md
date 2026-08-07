# 安全漏洞修复报告

**分支**: `release/v0.3.3`  
**日期**: 2026-05-22  
**模块**: `x/erc20`, `x/evmIBC`

---

## 概述

本文档记录了在 Uptick Network IBC 中间件模块中发现并修复的四个安全漏洞。所有漏洞都与 IBC 错误确认（error-ack）和超时退款流程相关，在这些流程中双层中间件处理可能导致资产重复或丢失。

---

## 漏洞 1：ERC20 基于 Memo 的授权绕过

**严重程度**: HIGH  
**文件**: `x/erc20/keeper/msg_server.go:520`（原始）  
**提交**: `f00275a`, `eebfb52`

### 描述

`refundPacketToken` 函数基于用户可控的 memo 子串来授权 ERC20 退款铸造：

```go
if !strings.Contains(data.Memo, types.TransferERC20Memo) {
    return nil
}
```

任何发送普通 ICS-20 转账的用户都可以在 memo 中附加 `":IBCTransferFromERC20"`，使得 ERC20 模块在错误确认时铸造 ERC20 代币。由于 memo 完全由用户控制，这将导致未经授权的代币铸造。

### 攻击场景

1. 攻击者发送普通 ICS-20 转账，memo 为 `"exploit:IBCTransferFromERC20"`
2. 接收链返回错误确认
3. `refundPacketToken` 检查 memo → 匹配 → 向攻击者铸造 ERC20 代币
4. ibc-go 同时退还 Cosmos 代币 → 攻击者获得双重资产

### 修复方案

用密码学绑定的 **IBC 转账来源证明（IBC Transfer Provenance）** 系统替代基于 memo 的检查：

- `MsgTransferERC20` 在发起 IBC 转账前写入来源记录，绑定 `(port, channel, sequence, sender, denom, amount)`
- `refundPacketToken` 使用 `ConsumeIBCTransferProvenance` — 原子性的检查并删除操作，验证全部六个字段
- Memo 被排除在来源证明键之外，因此 memo 操作无法授权退款
- 来源证明为一次性使用（首次访问即消费）

### 变更文件

| 文件 | 变更 |
|------|------|
| `x/erc20/types/keys.go` | 新增 `prefixIBCTransferProvenance` KV 存储前缀 |
| `x/erc20/types/ibc_provenance.go` | `IBCTransferProvenanceKey` 键构建器 |
| `x/erc20/keeper/ibc_provenance.go` | Set/Has/Delete/Consume 操作 |
| `x/erc20/keeper/ibc_packet_denom.go` | 代币面额解析辅助函数 |
| `x/erc20/keeper/msg_server.go` | `TransferERC20` 设置来源证明；`refundPacketToken` 消费来源证明 |
| `x/erc20/keeper/ibc_hook.go` | 成功确认删除来源证明；错误确认返回退款错误 |

---

## 漏洞 2：ERC20 IBC 错误确认双重退款

**严重程度**: HIGH  
**文件**: `x/erc20/ibc_middleware.go:86-115`, `x/erc20/keeper/msg_server.go:535-589`  
**状态**: 未提交的工作变更

### 描述

`IBCMiddleware.OnAcknowledgementPacket` 在错误确认时同时调用了 ERC20 keeper 的退款处理器和底层 ibc-go 转账模块的退款处理器：

```
步骤 A: im.keeper.OnAcknowledgementPacket → 向发送者铸造 ERC20 代币
步骤 B: im.Module.OnAcknowledgementPacket  → 向发送者解冻 Cosmos 代币
```

两个层级各自独立地向发送者退款，导致发送者同时收到 **ERC20 代币和 Cosmos 代币** — 资产翻倍。

### 攻击场景（NativeCoin 交易对）

| 步骤 | 发送者 ERC20 | 发送者 Cosmos | IBC 托管 |
|------|-------------|---------------|------------|
| 初始 | 100 ERC20 | 0 | 0 |
| ConvertERC20 + IBC 发送 | 0 | 0 | 100 Coin |
| **错误确认步骤 A**（ERC20 铸造） | **100 ERC20** | 0 | 100 Coin |
| **错误确认步骤 B**（ibc-go 解冻） | 100 ERC20 | **100 Coin** | 0 |

发送者资产翻倍：从原本的 100 ERC20 变为 100 ERC20 + 100 Coin。

### 修复方案

包含两部分修复：

1. **反转处理器顺序**（`ibc_middleware.go`）：ibc-go 先执行（将 Cosmos 代币退还给发送者），然后 ERC20 处理器执行（铸造 ERC20 + 归集 Cosmos 代币）

2. **新增银行归集**（`msg_server.go:565-588`）：铸造 ERC20 后，将 Cosmos 代币从发送者归集回模块：
   - **NativeCoin 交易对**：代币保留在模块托管中（与原始解冻相反的操作）
   - **NativeERC20 交易对**：代币被销毁（与原始铸造相反的操作）

两个操作在同一笔交易中原子执行。

---

## 漏洞 3：EvmIBC IBC 错误确认双重退款

**严重程度**: HIGH  
**文件**: `x/evmIBC/ibc_middleware.go:119-148`, `x/evmIBC/keeper/ibc_hook.go:120-148`  
**状态**: 未提交的工作变更

### 描述

与 ERC20 相同的模式 — evmIBC 中间件同时调用了两个处理器：

```
步骤 A: im.keeper.OnAcknowledgementPacket → ERC721 safeTransferFrom 给发送者
步骤 B: im.Module.OnAcknowledgementPacket  → nft-transfer 将 NFT 退还给发送者
```

发送者同时收到 ERC721 代币和 ICS-721 NFT。

### 修复方案

在中间件和 keeper 中进行两部分修复：

**中间件**（`ibc_middleware.go`）：在带有 convert memo 的错误确认时，完全跳过 nft-transfer 模块。由 evmIBC keeper 处理两端。

**Keeper**（`ibc_hook.go`）：退还 ERC721/CW721 之后，调用 nft-transfer keeper，并将 `data.Sender` 覆写为模块地址：

```go
nftData := data
nftData.Sender = erc721types.AccModuleAddress.String()
return k.ibcKeeper.OnAcknowledgementPacket(ctx, packet, nftData, ack)
```

NFT 被铸造/转移至模块账户而非发送者，从而防止双重退款。

---

## 漏洞 4：EvmIBC 超时遗漏底层模块调用

**严重程度**: MEDIUM  
**文件**: `x/evmIBC/ibc_middleware.go:178-205`  
**状态**: 未提交的工作变更

### 描述

`OnTimeoutPacket` 调用了 `im.keeper.OnTimeoutPacket`（ERC721 退款），但**从未**调用 `im.Module.OnTimeoutPacket`（nft-transfer 退款）。超时时：

- ERC721 已退还给发送者 ✓
- NFT **从未归还** — 永久卡在 nft-transfer 托管中

这是双重退款的反面：**退款不足**导致资产永久丢失。

### 攻击场景

1. 用户通过 IBC 发送 ERC721（转换为 NFT）
2. 数据包超时（目标链未收到）
3. ERC721 退还给发送者（通过 evmIBC keeper）
4. NFT 永久锁定在 nft-transfer 托管地址中
5. 该 NFT 无法访问，代表被锁定的链状态

### 修复方案

**中间件**（`ibc_middleware.go:198-209`）：对于非 convert 数据包，委托给 `im.Module.OnTimeoutPacket`。对于 convert 数据包，keeper 处理两端（与错误确认修复相同的 NFT 重定向模式）：

```go
if hasConvertMemo(data.Memo) {
    // Keeper 处理 ERC721 退款 + NFT 重定向至模块
    return im.keeper.OnTimeoutPacket(ctx, packet, data)
}
// 非 convert：委托给 nft-transfer 模块
return im.Module.OnTimeoutPacket(ctx, packet, relayer)
```

---

## 修复策略对比

| 对比项 | ERC20 | EvmIBC |
|---|---|---|
| **策略** | 让 ibc-go 退款，然后归集 | 跳过底层模块，重定向到模块 |
| **资产类型** | 同质化（ERC20 + Coins） | 非同质化（ERC721 + NFT） |
| **归集机制** | `bankKeeper.SendCoinsFromAccountToModule` | `ibcKeeper.OnAckPacket`（修改 sender） |
| **授权模型** | 来源证明记录（port+channel+seq+sender+denom+amount） | Memo 标记（ERC721/CW721 convert 标记） |
| **差异原因** | Bank 模块支持直接代币归集 | 无直接 NFT 归集 API；通过 keeper 重定向 |

---

## 安全保证

1. **原子性**：所有退款和归集操作在单笔 Cosmos SDK 交易中执行。任何失败都会导致完全回滚。
2. **防抢跑**：IBC 处理器同步执行，退款和归集之间无法插入外部交易。
3. **一次性来源证明**：`ConsumeIBCTransferProvenance` 原子性地检查并删除，防止重放。
4. **禁止基于 memo 的授权（ERC20）**：来源证明键不包含 memo；用户可控的 memo 无法授权铸造。
5. **NFT 隔离（EvmIBC）**：重定向的 NFT 进入模块账户，用户无法访问。

---

## 测试覆盖

| 测试 | 文件 |
|------|------|
| `TestIBCTransferProvenanceLifecycle` | `ibc_provenance_test.go` |
| `TestCraftedMemoTriggersNoMintWithoutProvenance` | `ibc_security_test.go` |
| `TestErrorAckWithoutMarkerMemoNoProvenance` | `ibc_security_test.go` |
| `TestErrorAckProvenanceConsumedOnRefund` | `ibc_security_test.go` |
| `TestProvenanceKeyUsesPacketDenomNotBankHash` | `ibc_security_test.go` |
| `TestSuccessAckClearsProvenance` | `ibc_security_test.go` |
| `TestProvenanceKeyDiffersOnMemoMarkerFieldsUnchanged` | `ibc_provenance_test.go` |
