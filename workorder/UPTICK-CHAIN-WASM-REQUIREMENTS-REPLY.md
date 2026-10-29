# UPTICK-CHAIN-WASM-REQUIREMENTS — 链侧答复

| 项 | 值 |
|---|---|
| 答复对象 | `UPTICK-CHAIN-WASM-REQUIREMENTS`（UGE，2026-10-29） |
| 日期 | 2026-10-29 |
| 来自 | Uptick Chain |
| 链仓基线 | `uge/v1.1` @ `c80d4bb`（tag `v0.5.2-rc1` = `8529508`） |
| 发布链 | `uptick_7777-1` @ 34.177.82.11 |

---

## 0. 总表

| # | 项 | 结论 |
|---|---|---|
| Q-W1 | 上传/实例化权限 | **裁定 A：维持 `Everybody` / `Everybody`（7777）**；主网收敛另议 |
| Q-W2 | 事件边界 | 确认：失败 tx 仍进 `block_results`；**务必保留 `code!=0` 丢弃**；无额外事件裁剪 |
| Q-W3 | gas 口径 | 确认：同 Cosmos `gas_used`、无自定义系数；附 `[wasm]` 实测 |
| Q-W4 | 117 / 7777 一致性 | 确认：wasmd **v0.61.14** / wasmvm **v3.0.7** 同源；加固补丁同属本仓 `app/` |
| Q-W5 | 生态边界 | 确认：无 wasm→EVM precompile 绑定；ICS20 对 wasm 可用（BuiltIn） |
| Q-W6 | 参数治理 | 确认：authority = gov；标准 `MsgUpdateParams` 流程 |
| C1–C4 | 接口契约 | **全部书面确认** |

**链侧无需改码、无需升级；UGE 可立即联调。**

---

## 1. Q-W1 裁定（唯一拍板项）

**选择：方案 A — 维持现状 `Everybody` / `Everybody`。**

理由（发布链 `uptick_7777-1`）：

1. UGE 归因已依赖 registry / `WASM_EMITTERS` 白名单，链上开放不构成计酬冒用；
2. 已有生产 wasm 资产（code_id 1/2/3）与第三方实例化路径；收敛白名单会打断联调与既有合约运维；
3. ante 已对 `MsgStoreCode` 等做签名前校验与 800KB 上限，开放不等于无防护。

**主网 `uptick_117-1`：** 不在本次范围。若日后收敛为方案 B（`AnyOfAddresses`），走 §Q-W6 治理改参；UGE 排期按「提案通过高度 +1」生效即可，无需链升级。

---

## 2. Q-W2 事件行为边界

| 问 | 答复 |
|---|---|
| 失败 wasm 交易是否仍在 `block_results`？ | **是。** CometBFT 对失败 DeliverTx 仍返回 `txs_results[]`（`code != 0`，可含 `message`/`transfer`/`tx` 等事件）。2026-10-29 抽失败样本 height `414020`（`code=11` out of gas）：`block_results` 有该 tx，`events` 非空但**无** `wasm*` 自定义事件。 |
| 是否可删 UGE `code!=0` 过滤？ | **不可删。** 失败路径上仍可能出现 ABCI 事件；合约执行中途失败 / 与其它 Msg 组合时行为因版本与失败点而异。UGE 以 `code!=0` 一律丢弃是正确防御，请保留。 |
| attribute / 事件裁剪上限？ | **链节点与 wasmd v0.61.14 未对自定义事件 attribute 数量另设裁剪。** UGE 单事件 13 attribute 远低于风险区。块级约束仅为 CometBFT 块大小 / gas；本发布链未额外挂事件过滤器。 |

---

## 3. Q-W3 gas 计量口径

1. **确认**：wasm 与 EVM 均进入 Cosmos SDK `gas_used`；`app/keepers` **未** `WithGasRegister` 自定义系数，走 wasmd 默认 `GasRegister` + ante `GasRegisterDecorator`。
2. 发布链节点 `/home/ubuntu/.uptickd/config/app.toml` **`[wasm]` 实测（2026-10-29）**：

```toml
[wasm]
query_gas_limit = 50000000
memory_cache_size = 512          # MiB
simulation_gas_limit = 25000000
# contract_debug_mode 未写入 → wasmd 默认 false
```

---

## 4. Q-W4 117 与 7777 版本 / 加固

| 组件 | 7777（`v0.5.2-rc1`） | 117 主网（v0.5.0 公告口径） |
|---|---|---|
| wasmd | **v0.61.14** | **v0.61.14** |
| wasmvm | **v3.0.7** | **v3**（同跃迁线） |
| ante 加固 | `message_security.go` + wasm 四装饰器 | 同源 `app/`（随主网二进制版本） |

**差异不在 wasm 栈**：7777 多 appmining V1 / DEC-10 等业务模块与联调参数；**UGE 合约测试以 7777（wasmd 0.61.14 + wasmvm v3）为准即可**，与本地 v0.4.1 已跑通栈一致。

---

## 5. Q-W5 生态边界

1. **wasm → EVM precompile / x/erc20：** **当前未开放。** 无自定义 wasm binding；`GetWasmCapabilities()` = wasmd `BuiltInCapabilities()`（`iterator/staking/stargate/cosmwasm_1_1…2_2/ibc2`），**不含** EVM/erc20。首版 UGE 不依赖此能力 — 与边界一致。
2. **ICS20：** **对外可用。** WasmKeeper 挂了 IBC Channel v1/v2 + `IBCTransferKeeper` 作 ICS20 端口源；合约具备 IBC 能力即可走 ICS20，**无**额外链上白名单准入（与 Q-W1 Everybody 一致）。跨链发送仍受 IBC 通道/客户端状态约束。

---

## 6. Q-W6 wasm 参数治理流程

- **Authority：** `authtypes.NewModuleAddress(govtypes.ModuleName)`（`keepers.go:571`）。
- **动作：** 治理提案提交 wasmd `MsgUpdateParams`（改 `code_upload_access` / `instantiate_default_permission`）。
- **生效：** 提案 PASS 后在**执行高度**写入，**无需**软件升级 / 节点重启。
- **样例结构（示意）：**

```json
{
  "messages": [{
    "@type": "/cosmwasm.wasm.v1.MsgUpdateParams",
    "authority": "<gov module account>",
    "params": {
      "code_upload_access": { "permission": "AnyOfAddresses", "addresses": ["uptick1…"] },
      "instantiate_default_permission": "Everybody"
    }
  }],
  "metadata": "ipfs://…",
  "deposit": "…auptick",
  "title": "wasm: tighten code upload access",
  "summary": "…"
}
```

投票参数以链上 `gov` params 为准（7777 联调链可能已调低门槛）。**当前 Q-W1=A，本流程仅作日后方案 B 预留。**

---

## 7. C1–C4 接口契约确认

| # | 确认 |
|---|---|
| **C1** | **确认。** wasmd `CustomContractEventPrefix = "wasm-"`；`Event::new("uge.qbt_reported")` → 链上 `wasm-uge.qbt_reported`。本版本前缀规则不变。UGE 摄取端兼容裸名属客户端策略，链侧只保证带前缀形态。 |
| **C2** | **确认。** 自定义事件 attribute 不被 wasmd 重命名；`_contract_address` 由 wasmd 附加在合约事件上（UGE 白名单依赖，必然出现于成功执行路径）。 |
| **C3** | **确认。** `_contract_address` 为 bech32 `uptick1…`，32 字节合约地址载荷（约 52 字符），与 wasmd v0.61.14 一致。 |
| **C4** | **确认。** wasm QBT 接入**不隐含** `x/appmining` / proto 演进；publish/approve/claim 与 VM 无关。 |

另悉：UGE 使用 `add_event` 挂业务属性（非仅 `Response::attributes`）——与 wasmd 约束一致，链侧无异议。

---

## 8. 链侧行动项

| 项 | 状态 |
|---|---|
| 改码 / 升级 / 改参 | **无** |
| 对外文档标注 `/cosmwasm/wasm/v1/codes/params` 等路径 | 接受建议；附录 A 可转载至链文档（非阻塞） |
| 将 UGE emitter wasm 纳入 `wasm_vm_compat_test` | 可选增强；UGE 提供二进制后可加（非阻塞） |

UGE 可按原文 §7「下一步」直接上传 `uge_qbt_emitter.wasm` 并走全链路。
