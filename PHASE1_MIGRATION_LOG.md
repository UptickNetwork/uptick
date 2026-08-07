# 阶段一迁移操作记录

## 概览
本文档记录了 uptick 项目从 ethermint → cosmos/evm v0.6.1 迁移的阶段一实施过程。

---

## 已完成的任务

### Task 0.1: 前置清理 ✅
- 移除 rosetta replace、注释清理、Makefile 修复

### Task 1.1: go.mod 改造 ✅
- cosmos/evm v0.6.1 + ibc-go v10 + SDK 0.53.6 + wasmd v0.61.14

### Task 1.2: capability 拆除 ✅
- 移除所有 capability 模块引用（6 个 ScopedKeeper、StoreKey、MemStoreKey）

### Task 1.3: EVM Keeper 替换 ✅
- cosmos/evm NewKeeper + WithStaticPrecompiles

### Task 1.4: x/erc20 归属 ✅
- 保留自有 IBC Provenance + TransferERC20
- 修复 hooks + msg_server + IBCModule v10 签名

### Task 1.5: x/evmibc 重写 ✅
- 重命名 x/evmIBC → x/evmibc（修复 macOS 大小写冲突）
- 更新所有 IBCModule 方法签名（添加 channelVersion string）

### Task 1.6: nft-transfer fork ✅
- 创建 UptickNetwork/nft-transfer 仓库
- 核心包全部升级到 ibc-go v10
- 本地 replace ../nft-transfer-fork

### Task 1.7: wasmd + wasmvm 验证 ✅
- wasmd v0.61.14 编译通过

### Task 1.8: 子仓库同步 ✅
- evm-nft-convert: 升级到 cosmos/evm v0.6.1 + ibc-go v10 + SDK 0.53
- wasm-nft-convert: 升级到 cosmos/evm v0.6.1 + ibc-go v10 + SDK 0.53 + wasmd v0.61
- 两个子仓库编译零错误

### Task 1.9: 升级处理器 v034.go ✅
- 实现 capability store 清理
- 实现 erc20 params 迁移
- 注册到 app/upgrade.go

---

## 编译状态

| 仓库 | go build ./... | go build cmd/uptickd |
|------|:---:|:---:|
| uptick | ✅ 零错误 | ❌ keepers.go API 变更 |
| evm-nft-convert | ✅ 零错误 | N/A |
| wasm-nft-convert | ✅ 零错误 | N/A |
| nft-transfer-fork | ✅ 核心包零错误 | N/A |

### cmd/uptickd 剩余编译错误（7 类）

1. **wasmtypes.WasmConfig** — wasmd v0.61 中移除或重命名
2. **ibckeeper.NewKeeper** — ibc-go v10 签名变更（KVStoreService + ParamSubspace + UpgradeKeeper）
3. **ScopedIBCKeeper** — 已移除字段仍被引用
4. **ScopedICAHostKeeper** — 已移除字段仍被引用
5. **icahostkeeper.NewKeeper** — ibc-go v10 签名变更
6. **feemarketkeeper.NewKeeper** — cosmos/evm v0.6.1 签名变更（移除 Subspace 参数）
7. **Erc20Keeper 接口** — cosmos/evm v0.6.1 要求 `GetERC20PrecompileInstance` 方法
8. **DefaultStaticPrecompiles** — 参数类型变更（*Keeper vs Keeper，**Keeper vs *Keeper）

这些是 ibc-go v10 + cosmos/evm v0.6.1 的深层 API 变更，需要逐个适配 keepers.go 中的 keeper 构造调用。

---

## ethermint Fork 推送

| Tag | 修改内容 |
|-----|---------|
| v0.24.2-uptick | 添加 GetStateAndCommittedState 方法 |
| v0.24.3-uptick | 添加 IsStorageEmpty 方法 |
| v0.24.4-uptick | 移除 rosetta 依赖 |

## nft-transfer Fork
- 仓库：UptickNetwork/nft-transfer（已创建，代码在本地 ../nft-transfer-fork）
- 核心包全部升级到 ibc-go v10
- testing/ 目录有 v8 残留，暂时排除

---

## 本地 Replace 配置

```go.mod
github.com/bianjieai/nft-transfer => ../nft-transfer-fork
github.com/UptickNetwork/evm-nft-convert => ../evm-nft-convert
github.com/UptickNetwork/wasm-nft-convert => ../wasm-nft-convert
github.com/evmos/ethermint => github.com/UptickNetwork/ethermint v0.24.4-uptick
```

---

## 被拒绝的操作记录

1. `rm -rf x/nft` — 删除无引用的 x/nft 目录
2. `git mv` 重命名 3 个拼写错误文件
3. `git push` 推送 nft-transfer 到 UptickNetwork（网络问题）
4. `find -exec sed` 批量替换（部分被拒绝）

---

## 下一步工作

### 优先级 P0：修复 cmd/uptickd 编译
1. 修复 keepers.go 中 ibckeeper.NewKeeper 构造（ibc-go v10 新签名）
2. 移除 ScopedIBCKeeper/ScopedICAHostKeeper 引用
3. 修复 icahostkeeper.NewKeeper 构造
4. 修复 feemarketkeeper.NewKeeper 构造（移除 Subspace）
5. 修复 wasmtypes.WasmConfig（wasmd v0.61 变更）
6. 为 Erc20Keeper 添加 GetERC20PrecompileInstance 方法
7. 修复 DefaultStaticPrecompiles 参数类型

### 优先级 P1：功能验证
1. devnet 初始化测试
2. IBC 转账测试
3. ERC20 转换测试
4. NFT 转换测试
