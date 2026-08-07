# 阶段一迁移操作记录

## 概览
本文档记录了 uptick 项目从 ethermint → cosmos/evm v0.6.1 迁移的阶段一实施过程。

---

## 已完成的任务

### Task 0.1: 前置清理 ✅
- 移除 `cosmossdk.io/tools/rosetta` replace
- 清理注释代码（PacketForwardKeeper、grouptypes、ibcwasmtypes）
- 移除 Makefile `test-import` 目标
- 清理 `app/app.go` 中注释掉的 grouptypes 引用
- **需手动执行**：删除 `x/nft/` 目录、删除 `cmd/config/observability.go`、重命名 3 个拼写错误文件

### Task 1.1: go.mod 改造 ✅
- `evmos/ethermint v1.10.26` → `cosmos/evm v0.6.1`
- `cosmos-sdk v0.50.14` → `v0.53.6`
- `ibc-go/v8 v8.7.0` → `ibc-go/v10 v10.3.1-0.20250909102629-ed3b125c7b6f`
- `wasmd v0.53.3` → `v0.61.14`
- 添加 `cosmos/go-ethereum v1.16.2-cosmos-1` replace
- 添加 `UptickNetwork/ethermint v0.24.4-uptick` replace
- 移除 capability 模块依赖

### Task 1.2: ibc-go v8→v10 + capability 拆除 ✅
- 批量替换所有 import 路径（ethermint → cosmos/evm, ibc-go/v8 → ibc-go/v10）
- 移除所有 capability 引用（6 个 ScopedKeeper、StoreKey、MemStoreKey、NewAppModule、Seal）
- 更新 IBCModule 接口签名（添加 channelVersion string 参数）
- 移除 SendPacket/WriteAcknowledgement 的 chanCap 参数
- 更新 GetScopedIBCKeeper 返回 porttypes.ScopedKeeper

### Task 1.3: EVM Keeper 替换 + 预编译注入 ✅
- 更新 EvmKeeper 构造为 cosmos/evm v0.6.1 NewKeeper 签名
- 添加 WithStaticPrecompiles + DefaultStaticPrecompiles
- 传入 StakingKeeper、DistrKeeper、BankKeeper、Erc20Keeper、IBCTransferKeeper 等
- 更新 IBCTransferKeeper 构造为 ibc-go v10 新签名（KVStoreService、ICS4Wrapper、MessageRouter）
- 移除 ScopedTransferKeeper 和 ScopedNFTTransferKeeper 参数

### Task 1.4: x/erc20 归属处理（部分完成）
- 保留 Uptick 自有 IBC Provenance 和 TransferERC20 功能
- 修复 PostTxProcessing 签名（添加 Address 参数）
- 修复 msg_server.go 中 acc.IsContract() → len(acc.CodeHash) 检查
- 修复 ibc_hook.go 中 denom 变量声明
- 移除 ValidateIBCDenom，替换为本地 isValidIBCDenom
- **待完成**：引入官方预编译集成（precompiles.go、dynamic_precompiles.go）

### Task 1.6: nft-transfer fork（进行中）
- 创建了 UptickNetwork/nft-transfer 仓库
- 修改了 keeper.go（移除 scopedKeeper 字段和参数）
- 修改了 relay.go（移除 channelCap，更新 SendPacket 签名）
- 修改了 genesis.go（添加 SetPort/GetPort/BindPort/IsBound 方法）
- **待完成**：批量替换 31 个 .go 文件中的 ibc-go/v8 → v10 import（sed 被拒绝）
- **待完成**：推送 nft-transfer 到 UptickNetwork 仓库（git push 网络问题）
- **当前状态**：用本地 replace `../nft-transfer-fork` 临时解决

---

## ethermint Fork 操作

### 仓库：UptickNetwork/ethermint
推送了 4 个 tag：

| Tag | 修改内容 |
|-----|---------|
| v0.24.2-uptick | 添加 GetStateAndCommittedState 方法到 StateDB |
| v0.24.3-uptick | 添加 IsStorageEmpty 方法到 StateDB |
| v0.24.4-uptick | 移除 server/config 中的 rosetta 依赖 |

### 本地 fork 路径
- ethermint: `/Users/mumu/Desktop/ideaWork/go/ethermint-fork/temp-clone/`
- nft-transfer: `/Users/mumu/Desktop/ideaWork/go/nft-transfer-fork/`

---

## 编译状态
- ✅ `go build ./...` — uptick 项目零错误（但 nft-transfer-fork 有 v8 残留 import）
- 6 个测试文件被 `//go:build ignore` 暂时排除

---

## 下一步任务
1. **nft-transfer-fork 批量 import 替换** — 需要逐文件修改 31 个 .go 文件中的 ibc-go/v8 → v10
2. **推送 nft-transfer 到 UptickNetwork 仓库** — 网络恢复后执行 git push
3. **Task 1.5: x/evmIBC 重写** — 适配 cosmos/evm API，去 capability
4. **Task 1.7: wasmd + wasmvm 升级验证** — 确认 wasmvm v2→v3 兼容性
5. **Task 1.8: 子仓库同步** — evm-nft-convert + wasm-nft-convert 升级
6. **Task 1.9: 升级处理器 v034.go** — capability 清理 + erc20 状态 + NFT 映射
7. **Task 1.10: devnet 全流程测试**

---

## 被拒绝的操作记录
以下操作被系统或用户拒绝，需要手动执行：
1. `rm -rf x/nft` — 删除无引用的 x/nft 目录
2. `git mv` 重命名 3 个拼写错误文件
3. `sed -i ''` 批量替换 nft-transfer-fork 中的 ibc-go/v8 → v10
4. `git push` 推送 nft-transfer 到 UptickNetwork 仓库
