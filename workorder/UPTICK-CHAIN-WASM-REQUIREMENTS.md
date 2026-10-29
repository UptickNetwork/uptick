# Uptick Chain × UGE：wasm 能力对接需求

| 项 | 值 |
|---|---|
| 文档标识 | `UPTICK-CHAIN-WASM-REQUIREMENTS`（本文为唯一有效版本，无版本号） |
| 编制日期 | 2026-10-29 |
| 发起方 | UGE |
| 接收方 | Uptick 链团队 |
| 目标链 | `uptick_7777-1`（发布链，在线 `v0.5.2-rc1` / commit `8529508`） |
| 复核依据 | 链仓 `uge/v1.1` @ `c80d4bb`（含 tag `v0.5.2-rc1`）逐文件核对 + 链上实测（链高 418651） |
| 已合并的旧版 | `UPTICK-CHAIN-WASM-ENABLE-REQUEST-0.1.md`、`UPTICK-CHAIN-WASM-ENABLE-REQUIREMENTS-1.0.md`、`UPTICK-CHAIN-WASM-REQUIREMENTS-2.0.md`（三份均删除，此三者内容已全部并入本文，历史留存于 git） |
| 性质 | **不是"请启用 wasm"**。wasm 已接线并在生产使用；本文是 **4 项接口契约确认 + 6 项裁定/确认**，均不阻塞 UGE 联调 |

---

## 0. 一页结论

| # | 结论 | 依据 |
|---|---|---|
| 1 | **wasm 模块已完整接线，无需任何链侧启用动作** | 链仓 `app/app.go:369/420/424/512/547/587/627/991`、`app/keepers/keepers.go:548-573`；链上 `/cosmwasm/wasm/v1/codes/params` 返回 200 |
| 2 | **wasm 已在生产使用**，不是尚未启用的新功能 | 链上已有 `code_id` 1/2/3 与合约 `uptick14hj2tav…l95g`，存在真实 `MsgExecuteContract` 交易 |
| 3 | **UGE 唯一硬依赖（事件进 `block_results`）天然满足** | `block_results@212893` 中 wasm 事件带 `_contract_address` 与完整业务 attribute |
| 4 | **不存在阻塞项**：本文 6 项问题里只有 1 项需要裁定，其余是书面确认 | 见 §4 |

**需要链团队给的三个产出**：

| # | 产出 | 章节 |
|---|---|---|
| 1 | **Q-W1 裁定**：上传/实例化权限维持 `Everybody` 还是收敛为白名单（唯一需要拍板项） | §4 Q-W1 |
| 2 | Q-W2…Q-W6 书面确认（事件边界 / gas 口径 / 版本一致性 / 生态边界 / 治理流程） | §4 Q-W2…Q-W6 |
| 3 | C1–C4 接口契约书面确认 | §3 |

---

## 1. 背景：UGE 为什么要 wasm 事件源

UGE 的 QBT（Qualified Billable Transaction）事件源按 VM 分型：

| 事件源 | 现状 | 摄取方式 |
|---|---|---|
| EVM（`EVM_QBTReported`） | **已完成端到端联调验收**（2026-10-26，mall app epoch 4 全流程） | 合约 emit log → `eth_getLogs` |
| WASM（`wasm-uge.qbt_reported`） | **链侧已就绪**，UGE 侧代码已落地，待自身上线 | 合约 emit ABCI event → `/block_results` |

**为什么必须有第二类**：Uptick 生态中 RWA、NFT 与第三方应用合约大量采用 CosmWasm。若只有 EVM 事件源，这些应用必须在 EVM 侧另建一层适配合约才能报单，增加接入成本与信任面。wasm emitter 让合约**自己**报单，UGE 只做摄取与计酬，链路更短、可审计性更强。

**已在 Uptick 自己的链代码上验证过（不是纸面方案）**：`release/v0.4.1` 本地 devnet 端到端 PASS，证据见 `release/uptick-v041-local-build-validation.json` 与 `docs/evidence/Uptick-Local-Dual-VM-QBT-PAV-Validation-0.1.md`：

| 项 | 本地 v0.4.1 devnet 实测 |
|---|---|
| 链侧 wasmd | `v0.61.14`（与发布链 `build_deps` 一致） |
| 合约 store / instantiate / 报单 | PASS |
| 跨 VM family 收敛 | PASS，`family_id = b166ed55…806a4`（与 EVM 侧同值） |

---

## 2. 链侧现状：代码级复核 + 链上实测

### 2.1 代码级复核（链仓 `uge/v1.1` @ `c80d4bb`）

wasm 的接线是**完整且经过加固**的，以下各项不需要链团队再做任何事：

| 能力 | 位置 | 结论 |
|---|---|---|
| 依赖版本 | `go.mod:8` `wasmd v0.61.14`；`go.mod:83` `wasmvm/v3 v3.0.7` | 与 UGE 本地已跑通的 v0.4.1 devnet 同版本 |
| 模块管理器注册 | `app/app.go:369` `module.NewManager` → `:420` `wasm.NewAppModule` | 已装配 |
| Begin/End/InitGenesis 顺序 | `app/app.go:512` / `:547` / `:587` / `:627` | 四个顺序表均含 `wasmtypes.ModuleName` |
| BasicManager（决定 HTTP 路由） | `app/app.go:424` `NewBasicManagerFromManager(app.mm, …)`；`:991` `app.bm.RegisterGRPCGatewayRoutes` | `/cosmwasm/wasm/v1/*` 因此注册 |
| Store key | `app/keepers/keepers.go:550` `runtime.NewKVStoreService(appKeepers.keys[wasmtypes.StoreKey])` | 已挂载（链从创世即含，无需 store 升级） |
| Keeper 装配 | `app/keepers/keepers.go:548-573`（account/bank/staking/distr/ICS4Wrapper/ChannelKeeper v1+v2/ICS20 端口源/GRPCQueryRouter），authority = 治理模块地址（`:571`） | 标准装配，无自定义偏差 |
| IBC 集成 | `app/keepers/keepers.go:634`（v1 handler）、`:641`（PortIDPrefixV2） | ICS20 能力已开（`GetWasmCapabilities()` 返回 `BuiltInCapabilities()`） |
| ante 计量链 | `app/ante/handler_options.go:172-187`：`SetUpContext` → `CountTXDecorator` → `GasRegisterDecorator` → `TxContractsDecorator` → `LimitSimulationGasDecorator` | wasm 四个装饰器齐全，顺序被 `app/ante/handler_options_test.go` 钉住 |
| 消息安全加固 | `app/ante/message_security.go:66-88`：`MsgExecuteContract` / `MsgInstantiateContract` / `MsgInstantiateContract2` / `MsgStoreCode` 在**签名验证前**校验；`MsgStoreCode` 复跑 `ValidateBasic` 拦住超限二进制（`MaxWasmSize` = 800 KB） | 已有加固（O-3） |
| wasmvm v3 兼容性 | `app/wasm_vm_compat_test.go`：把链上随附的 `release/wasm/cw721_base.wasm`（`interface_version_8`）推进**真实 wasmvm v3 实例**，覆盖 Create/Instantiate/QuerySmart/Execute | v0.4.0 的 wasmd 0.53.3→0.61.14、wasmvm 2.1.5→v3 跃迁已验证 |

### 2.2 参数状态（实测 2026-10-29）

```
GET /cosmwasm/wasm/v1/codes/params → 200
{"params":{"code_upload_access":{"permission":"Everybody","addresses":[]},
           "instantiate_default_permission":"Everybody"}}
```

| 参数 | 当前值 | 含义 |
|---|---|---|
| `code_upload_access.permission` | **`Everybody`** | 任意地址可上传合约代码 |
| `instantiate_default_permission` | **`Everybody`** | 任意地址可实例化合约 |

注：wasmd v0.61 的共识层 `Params` **只有这两个字段**；合约大小由 `wasmtypes.MaxWasmSize = 800 KB` 常量 + ante 校验约束（不在共识参数里）；`query_gas_limit` / `simulation_gas_limit` / `memory_cache_size` / `contract_debug_mode` 属节点本地 `app.toml [wasm]`，不参与共识。

### 2.3 链上已有 wasm 资产（生产在用）

| 项 | 实测 |
|---|---|
| 已上传代码 | `code_id` 1 / 2 / 3（`/cosmwasm/wasm/v1/code`），创建者 `uptick10xj8tc3vx3eqd7mhffmgruakvjckw0nepklgw9` |
| 已实例化合约 | `uptick14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s59l95g`（`code_id=1`） |

### 2.4 事件透出（UGE 唯一硬依赖，已自证）

取历史区块 `212893` 的 `/block_results`，`txs_results[].events`：

| 事件类型 | attributes |
|---|---|
| `wasm` | `_contract_address` / `action=send` / `from` / `to` / `amount` / `msg_index` |
| `wasm` | `_contract_address` / `action=place_order` / `buyer` / `order_id` / `product_id` / `total` / `msg_index` |

即：**事件进了 `block_results`、`_contract_address` 自动附加、合约自定义业务 attribute 完整保留、`tx_result.gas_used` 可读。** UGE 的硬门槛已被满足，无需链侧任何改动。

### 2.5 gas 口径实测（供 UGE 计酬校准）

| 样本 | 数量 | `gas_used` 区间 | 中位数 |
|---|---|---|---|
| wasm `MsgExecuteContract` | 20 笔 | 151,189 – 257,365 | ≈ 200,000 |
| EVM `MsgEthereumTx`（全类型） | 100 笔 | 21,000 – 463,258 | 57,251 |

结论：两者**同一 `gas_used` 字段、同一量级**，不存在"wasm gas 与 EVM gas 差两个数量级"的问题。UGE 会把 wasm 换算系数（当前占位 0.2）用自家 emitter 首笔真实交易重新校准，**这一项不需要链侧提供数据**，只需确认 §4 的 Q-W3。

---

## 3. 接口契约（无需链端改码，仅需书面确认）

UGE 参考实现 `protocol/wasm/qbt-emitter`（`cosmwasm-std 1.5.11`）已定型，链端不需要为它写任何特殊逻辑。以下四项请书面确认：

| # | 契约内容 | 需要确认的点 |
|---|---|---|
| **C1** | **事件类型**：合约 `Event::new("uge.qbt_reported")` → 链上呈现为 `wasm-uge.qbt_reported`（wasmd 对自定义事件统一加 `wasm-` 前缀，`x/wasm/types/events.go:16`） | 本版本前缀规则行为不变；UGE 摄取端同时兼容裸名 `uge.qbt_reported` |
| **C2** | **必需 attribute（snake_case，共 13 项）**：`schema_version` / `vm` / `qbt_id` / `family_id` / `app_id` / `tier` / `action_type` / `subject` / `underlying_action_id` / `denom` / `gross` / `refund` / `proof_hash`，外加 wasmd 自动附加的 `_contract_address`（UGE 用它做 emitter 白名单匹配，**必须有**） | 不会被截断/重命名；`_contract_address` 必然出现 |
| **C3** | **合约地址形态**：32 字节载荷的 bech32（`uptick1…`，52 字符）。UGE registry 已加结构 + polymod 校验和校验，写错一个字符即加载失败（不再静默丢事件） | wasmd v0.61.14 产出的 `_contract_address` 与此形态一致 |
| **C4** | **`x/appmining` 零改动**：`MsgPublishCandidate` / `ApproveEntitlement` / `ClaimEntitlement` 与 VM 无关，wasm 事件进入 UGE 后走同一套链上消息 | 书面确认 wasm 接入**不隐含** proto 演进需求 |

**另需知悉（无需答复）**：wasmd 要求事件类型长度 > 2 字符、attribute key 不得以 `_` 开头（`x/wasm/keeper/events.go`）；`Response::attributes` 会被塞进通用 `wasm` 事件而非自定义事件。我方合约用的是 `Response::new().add_event(Event::new("uge.qbt_reported").add_attribute(...))`（`protocol/wasm/qbt-emitter/src/lib.rs:82-89`），业务属性全部挂在**自定义事件**上（不是 `Response::attributes` 的通用属性列表），已遵守这两条。

---

## 4. 需要链侧答复的 6 项（1 项裁定 + 5 项确认）

### Q-W1（裁定）上传/实例化权限是否收敛 —— **唯一需要拍板的一条**

**现状**：`Everybody` / `Everybody`，任意地址都能部署并实例化合约。

**为什么与 UGE 有关**：UGE 的计酬归因靠**自己维护的 emitter 白名单**（registry + 签名器 `WASM_EMITTERS`），技术上不依赖链上限制 —— 非白名单地址发出的 `wasm-uge.qbt_reported` 会被直接丢弃。因此**这不是阻塞项**，但需要链侧明确立场：

| 方案 | 内容 | 对 UGE 的影响 |
|---|---|---|
| **A（维持现状）** | 保持 `Everybody`，UGE 继续以 registry 白名单归因 | 零改动；风险是任意合约可冒用事件名（UGE 侧已拦） |
| **B（收敛）** | 上传/实例化改白名单（`AnyOfAddresses`），由治理维护 | UGE 应用合约地址需走一次治理提案加入白名单；更干净，但增加链侧运营动作 |

**需要的答复**：A 还是 B；若选 B，白名单的治理流程、生效方式与前置时长。

### Q-W2（确认）事件行为边界

| 问 | 为什么要问 |
|---|---|
| 失败的 wasm 交易是否仍在 `block_results` 产出 events？ | UGE 已按 `tx_result.code != 0` 一律丢弃（防回滚后重放刷单）。若链侧**不**产事件，我方可以删掉这段防护，少一处逻辑 |
| 是否存在 attribute 数量 / 单事件字节 / 单块事件总量的裁剪或上限？ | UGE 单个 emitter 事件 13 个 attribute，需确认不触碰任何隐藏阈值。已在 wasmd v0.61.14 源码检索，**未见**事件或 attribute 数量上限；此处只需确认节点侧与 CometBFT 侧未额外裁剪 |

### Q-W3（确认）gas 计量口径

wasmd v0.61 的 `GasRegisterDecorator` 使用 keeper 的 gas register。链仓 `app/keepers/keepers.go:548-573` **没有**传入自定义 gas register（`wasmOpts` 中亦无 `WithGasRegister`），即走 wasmd 默认。

**需要的答复**：

1. 书面确认"wasm gas 与 EVM gas 同为 Cosmos `gas_used` 口径，且未额外乘以自定义系数"；
2. 给出生产节点 `app.toml` 中 `[wasm]` 的实际取值（`query_gas_limit` / `simulation_gas_limit` / `memory_cache_size` / `contract_debug_mode`）—— 这几项是节点本地的，UGE 从链上读不到，但决定我方合约查询调试的天花板。

### Q-W4（确认）117 与 7777 的版本与加固一致性

链仓 `app/wasm_vm_compat_test.go` 记录了 v0.4.0 的跃迁（wasmd 0.53.3→0.61.14、wasmvm 2.1.5→v3）。

**需要的答复**：117 测试网与 7777 发布链使用的 wasmd / wasmvm 版本与安全加固补丁是否为同一份；若不同，列出差异项（UGE 需按 7777 实际版本复跑合约测试）。

版本溯源沿用链侧既有记录（annotated tag `v0.5.2-rc1` = commit `8529508` = 在线 `application_version`，`vcs.modified=false`），本文不重复索取。

### Q-W5（确认）生态边界：wasm 与 EVM / erc20 / ICS20

首版 UGE **不需要** wasm 调 EVM precompile，也不需要 EVM 调 wasm。但需要书面边界，避免后续设计踩空：

1. wasm 合约能否调用 EVM precompile / `x/erc20` 的转换能力？当前是否已开放？
2. `BuiltInCapabilities()` 已开，说明 wasm 具备 ICS20 跨链能力 —— 该能力当前是否对外可用、有无额外准入？

### Q-W6（确认）wasm 参数变更的治理流程

若 Q-W1 选 B，需要 `MsgUpdateParams`（authority = 治理）或升级处理器。链仓 `app/keepers/keepers.go:571` 已把 authority 设为治理模块地址。

**需要的答复**：改 `code_upload_access` / `instantiate_default_permission` 的标准动作（治理提案文本样例、投票参数、生效高度），UGE 侧据此排期。

---

## 5. 明确不需要链侧做的事

| 项 | 说明 |
|---|---|
| 模块接线 / store key / 查询路由 | **已全部就绪**（§2.1），零改动，**也不需要为此安排任何链升级** |
| `x/appmining` 与 proto | **零改动**（C4）。不要为此做 proto 演进 |
| 事件转发 / UGE 专用模块 | 不需要。链上如实把合约事件放进 `block_results` 即可（已满足） |
| 提高合约大小上限 | 不需要。UGE 参考实现产物 **248,194 B（242 KB）**，远低于 `MaxWasmSize` 800 KB |
| 为 UGE 定制 gas 定价 | 不需要。UGE 自己在计酬侧按 VM 折算，链上维持默认 |
| 主网 `uptick_117-1` 立即跟进 | 本次只涉及发布链 `uptick_7777-1`；主网排期由链团队另定 |

---

## 6. 复核清单（任何一方可独立复跑）

以下命令在 2026-10-29 全部实测通过，链团队可自行复跑核对；UGE 侧收到答复后会再复跑一遍。

```bash
LCD=http://34.177.82.11:1317
RPC=http://34.177.82.11:26657

# V1 模块可用性与参数（注意前缀是 cosmwasm，params 挂在 codes/ 下）
curl -s "$LCD/cosmwasm/wasm/v1/codes/params" | jq .params
curl -s "$LCD/cosmwasm/wasm/v1/code?pagination.limit=5" | jq '.code_infos[] | {code_id, creator}'

# V2 事件是否进 block_results（UGE 唯一硬门槛）
curl -s "$RPC/block_results?height=212893" \
  | jq '.result.txs_results[].events[] | select(.type | startswith("wasm"))'

# V3 gas 口径
curl -s "$LCD/cosmos/tx/v1beta1/txs?query=message.action%3D'%2Fcosmwasm.wasm.v1.MsgExecuteContract'&pagination.limit=3&order_by=ORDER_BY_DESC" \
  | jq '.tx_responses[] | {height, code, gas_used, gas_wanted}'
```

| # | 复核项 | 期望 | 2026-10-29 实测 |
|---|---|---|---|
| V1 | 查询路由与参数 | 200，结构完整 | 通过（`Everybody` / `Everybody`） |
| V2 | 事件透出 | 含 `wasm*` 事件与 `_contract_address`；`gas_used > 0` | 通过（区块 `212893`） |
| V3 | gas 口径 | wasm 与 EVM 同字段同量级 | 通过（见 §2.5） |
| V4 | 版本溯源四元一致 | tag = commit = checksum = 在线 version，`vcs.modified=false` | 沿用链侧记录，未重复索取 |

---

## 7. UGE 侧就绪状态与下一步

| 项 | 状态 |
|---|---|
| 合约参考实现 + 单测 4/4（跨 VM `family_id` 黄金向量、reporter 守卫、重放守卫、`refund ≤ gross`） | 已有（`protocol/wasm/qbt-emitter`，`cosmwasm-std 1.5.11`） |
| wasm32 release 产物 | 248,194 B / `sha256 b57007ce…19c1d4f`，导出含 `interface_version_8` + `instantiate/execute/query`，工具链 Rust 1.81.0 |
| 签名器 wasm 事件摄取（`WASM_EMITTERS` 白名单、`/identity`、启动日志） | 已落地（`services/chain-source-signer/server.mjs`） |
| 失败交易事件过滤（`code != 0` 一律丢弃） | 已落地（`services/protocol-service/origin3-chain-source.mjs`） |
| registry 的 bech32 结构与校验和校验（32 字节合约载荷） | 已落地（`protocol/qbt-resolver/{bech32,registry}.mjs`） |
| 混合信封 / 白名单 / 失败交易 / `event_index` 唯一性测试 | 已落地（`services/chain-source-signer/wasm-envelope.test.mjs`） |
| 部署透传 `EMITTER_WASM_EMITTERS`（缺省空 = 安全默认） | 已落地（`deploy/integration/compose.emitter.yaml`） |
| 全仓回归 | **227/227 通过**（`make verify-node`）；`make verify-wasm` 4/4 |

**下一步（UGE 自己执行，不需要等链侧）**：向 7777 上传 `uge_qbt_emitter.wasm` → 实例化 → 把合约地址写入 registry 与 `EMITTER_WASM_EMITTERS` → 重启签名器 → 走一笔全链路（事件 → 候选 → 核定）。

**为什么"失败的交易要过滤"（链端可对照 Q-W2）**：链上 tx 失败时会回滚合约状态（含重放守卫 `SEEN_QBT`），但 ABCI 事件仍会出现在 `block_results` 里，重试一次就能重复产出可计酬事件。EVM 侧天然免疫（回滚日志不进 `eth_getLogs`），wasm 必须显式拦截。若链端对失败交易**不**产出 events，请一并告知，我方好删掉这段死代码。

---

## 8. 风险

| # | 风险 | 影响 | 建议 / 缓解 |
|---|---|---|---|
| 1 | `Everybody` 下任意合约可冒用 `uge.qbt_reported` 事件名 | 假事件进入摄取层 | UGE registry 白名单已拦住（非白名单地址事件直接丢弃）；若链侧希望从源头收敛，走 Q-W1 方案 B |
| 2 | wasm 查询路径特殊（`/codes/params`、`/code`），易被误判为"未接线" | 误判模块状态、白等排期 | 附录 A 附正确路径速查，建议链侧在对外文档中标注 wasm 的实际 REST 路径 |
| 3 | wasmvm 大版本（v2→v3）已跨过一次，后续再 bump 时动态链接需全节点预装 `.so` | 若为动态链接，`libwasmvm` 未预装则 panic 发生在新版本**首次调用** wasm 时，排查成本高 | 既有 `app/wasm_vm_compat_test.go` 用真实 wasmvm 跑随附合约；建议把 UGE emitter 合约也纳入该回归集（UGE 可提供 wasm 二进制） |
| 4 | 计酬侧的 wasm 换算系数为占位值 0.2 | 首笔真实交易前金额不准 | 用 UGE emitter 首笔真实交易校准，无需链侧参与 |
| 5 | 事件被节点侧另行裁剪（源码头未见上限，但无法从外部证明） | wasm 事件源失效，只能退化为状态轮询 | 由 Q-W2 书面确认兜底 |

---

## 9. 答复方式

1. 请按 **Q-W1…Q-W6** 逐条答复，**C1–C4** 一并书面确认；可直接在本文追加"链侧答复"列，或另出回复文档。**Q-W1 是唯一需要拍板项。**
2. 确认后 UGE 侧可立即联调，无需链侧再出人力：应用 reporter 地址（白名单用）、合约 wasm 二进制（含 checksum）、复核脚本均由 UGE 提供。
3. 往来文档以本仓 `docs/workorder/` 为主载体，链仓同步留副本。

---

## 附录 A：wasm REST 路径速查（wasmd v0.61.14 实际注册）

| 查询 | 路径 |
|---|---|
| 参数 | `/cosmwasm/wasm/v1/codes/params` |
| 代码列表 | `/cosmwasm/wasm/v1/code` |
| 单个代码 | `/cosmwasm/wasm/v1/code/{code_id}` |
| 指定代码的合约 | `/cosmwasm/wasm/v1/code/{code_id}/contracts` |
| 合约信息 | `/cosmwasm/wasm/v1/contract/{address}` |
| 合约历史 | `/cosmwasm/wasm/v1/contract/{address}/history` |
| 原始状态 | `/cosmwasm/wasm/v1/contract/{address}/raw/{query_data}` |
| 智能查询 | `/cosmwasm/wasm/v1/contract/{address}/smart/{query_data}` |
| 按创建者的合约 | `/cosmwasm/wasm/v1/contracts/creator/{creator_address}` |
| 已 pin 的代码 | `/cosmwasm/wasm/v1/codes/pinned` |

> 两个易错点：前缀是 `cosmwasm`（不是 `cosmos`）；`params` 挂在 `codes/params` 下（不是顶层 `params`）。

## 附录 B：代码引用清单

链仓 `uge/v1.1` @ `c80d4bb`：

- `go.mod:8,83` —— wasmd v0.61.14 / wasmvm v3.0.7
- `app/app.go:121-123, 207, 369, 420, 424, 512, 547, 587, 627, 675, 698-700, 982-991`
- `app/keepers/keepers.go:538-573, 580, 634, 641, 720-722`
- `app/ante/handler_options.go:44, 148-187`
- `app/ante/message_security.go:66-101`
- `app/wasm_vm_compat_test.go`（wasmvm v3 + 随附 cw721 合约）

上游 wasmd v0.61.14：

- `x/wasm/types/events.go:16`（`CustomContractEventPrefix = "wasm-"`）
- `x/wasm/keeper/events.go`（事件类型长度 > 2、attribute key 不以 `_` 开头）
- `proto/cosmwasm/wasm/v1/query.proto:64,80-84`（Codes / Params 的 HTTP 注解）

UGE 仓：

- `protocol/wasm/qbt-emitter/src/lib.rs:72-90`（事件契约）
- `protocol/qbt-resolver/{bech32,registry}.mjs`（地址校验）
- `services/chain-source-signer/server.mjs`（`WASM_EMITTERS`）
- `services/protocol-service/origin3-chain-source.mjs`（`extractWasmQbtEvents` + 失败交易过滤）
- `release/uge-qbt-emitter-wasm-0.1.0.json`（产物证据）

## 附录 C：一个方法论留痕（避免双方重复踩坑）

本文早期版本曾判定"发布链未接线 `x/wasm`"，依据是 `/cosmos/wasm/v1/params` 返回 `501 Not Implemented`。该结论是错的，错在探测路径：wasmd v0.61.14 的 Params 在 `/cosmwasm/wasm/v1/codes/params`，而该节点对**任何未注册路径**都返回逐字相同的 `501 {"code":12,"message":"Not Implemented"}`。当时虽做了随机路径对照，但对照只能证明"这个路径没注册"，不能证明"这个模块没接"。

**结论：判定模块是否存在，应直接读链仓代码（模块管理器 / keeper / 路由注册）或查 proto 的 `google.api.http` 注解，不要只看 REST 的 501。** 这条已写入 UGE 侧文档纪律；也建议链侧在外文档中标明 wasm 的实际路径（附录 A）。
