# 阶段一迁移操作记录

## 概览
本文档记录了 uptick 项目从 ethermint → cosmos/evm v0.6.1 迁移的阶段一实施过程。

## 关键决策：ethermint → cosmos/evm + cosmos/go-ethereum

### 为什么可以完全废弃 UptickNetwork/ethermint

| 功能 | UptickNetwork/ethermint | cosmos/go-ethereum v1.16.2 | 结论 |
|------|:---:|:---:|------|
| EIP-7702 SetCodeTx | ❌ 不支持 | ✅ 原生支持 | cosmos/go-ethereum 更优 |
| Shanghai | ✅ ShanghaiBlock | ✅ ShanghaiTime | 已迁移到 Time-based |
| Dencun (Cancun) | ✅ CancunBlock | ✅ CancunTime + BlobSchedule | 已迁移 |
| Prague | ✅ PragueBlock | ✅ PragueTime + EIP-7702 | 已迁移 |
| go-ethereum 版本 | v1.10.17 (2022) | v1.16.2 (2025) | 差距 6 个大版本 |
| 维护方 | Uptick 自行 fork | Cosmos 团队维护 | 无需自维护 |

### v034 升级处理器中的 ChainConfig 迁移

旧方式（ethermint v032 升级）：
```go
evmParams.ChainConfig.ShanghaiBlock = &zero  // 基于区块高度
evmParams.ChainConfig.CancunBlock = &zero
evmParams.ChainConfig.PragueBlock = &zero
```

新方式（cosmos/evm v0.6.1 v034 升级）：
```go
chainConfig.ShanghaiTime = &zero  // 基于时间戳（以太坊标准）
chainConfig.CancunTime = &zero
chainConfig.PragueTime = &zero    // 启用 EIP-7702 SetCodeTx
evmtypes.SetChainConfig(chainConfig)  // 全局变量设置
```

### EIP-7702 SetCodeTx 激活机制

- EIP-7702 在 `PragueTime` 被设置时自动激活
- 交易类型 `0x04` (SetCodeTx)
- 允许 EOA 账户委托到智能合约代码（账户抽象）
- `cosmos/go-ethereum v1.16.2` 原生包含，无需自定义实现

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
