# UPTICK 链侧任务交接包（UGE → 链 AI）2026-10-28 0.1

> 发起：UGE 侧
> 接收：Uptick 链侧 AI / 链团队
> 性质：**自包含任务书**——含全部背景、路径、验收标准，无需追问即可执行
> 关联仓库：链仓 `uptick`（分支 `uge/v1.1`，当前 HEAD `bde3cb6`）；UGE 仓（分支 `overhaul`，HEAD `3c68c4f`，已推 origin）

---

## 〇、背景速览（已完成的，勿重做）

| 事项 | 状态 | 证据 |
|---|---|---|
| DEC-10 需求（publisher MetaMask/secp256k1 直签） | 链侧已实现（`bde3cb6`）并 ACK 方案 1；UGE 已确认通过 | 链仓 `workorder/UPTICK-UGE-CHAIN-ACTION-REPLY-REQ-D-SECP256K1-0.1.md` |
| 跨实现互验（UGE Node SDK ↔ 链 Go 验签） | **3/3 PASS**（正路径 verified；high_s / sig_length 拒绝 reason 精确匹配） | `x/appmining/spec/appmining-v1-wire-secp256k1-golden-1.0.json`（已在链仓目录但**未 git 入库**）；UGE 侧 `docs/workorder/UPTICK-UGE-DEC10-INTEROP-CONFIRMATION-0.1.md` |
| UGE 侧配套（SDK ecrecover 复核 + console MetaMask 直签 UI + 错误码 82~85） | 已完成并**已发版上线**（uge.upticknet.com/console 可用） | UGE 仓提交 `6ad5fa7` |
| V1 publish→approve→claim 全生命周期（Ed25519 路径） | 真网 e2e 已全绿 | 2026-10-28 验收记录 |
| freeze 代发 | 真网 e2e 已全绿（FROZEN +72h） | tx AD39A31B… |

---

## TASK-1【P0】DEC-10 发版：把 bde3cb6 推上 origin 并部署发布链

### 要做什么

1. `git push origin uge/v1.1`（当前 `bde3cb6` 仅在本地，origin 停在 `174df34`）。
2. 打 **annotated tag**（建议 `v0.5.2-rc1` 或按链侧现行 tag 规则），并保证**四元一致**：tag = commit = 二进制 checksum = 在线版本字符串。
3. 构建发布链二进制并部署到**发布链 `uptick_7777-1`（34.177.82.11）**——链侧二进制按惯例由链团队自推。
4. 部署后更新节点可查的版本信息：在线版本字符串须与新 tag 一致（取证口径：LCD `/cosmos/base/tendermint/v1beta1/node_info` 顶层字段 / `abci_info` / `version --long`；cometbft `/status` 不携带 application_version，勿用）。

### 验收标准（DoD）

- [ ] origin/uge/v1.1 包含 `bde3cb6`；
- [ ] 发布链节点版本字符串 = 新 tag（UGE 侧会用上述取证口径核对）；
- [ ] 链上 `uptickd tx appmining register-role-key --help` 出现 `--key-type secp256k1-evm`。

### 为什么 P0

UGE 侧 console MetaMask 发布已上线但链上验签还不支持 secp256k1——**发布链不升级，DEC-10 端到端就无法真网验收**（TASK-4 阻塞在本项）。

---

## TASK-2【P0】T4 治理提案 3 项书面确认

提案包：UGE 仓 `docs/workorder/UPTICK-CHAIN-T4-GOV-PROPOSAL-PACKAGE-DRAFT-0.1.md`（链仓 workorder 有副本）；配套可提交 JSON：同目录 `gov-proposal-t4-publisher-binding-enforcement.json`。提案本体（2 条 Msg：先 MsgUpdateParams 强制凭证+押金制，后 MsgSetRoleKeyStatus 撤 pub-2）UGE 已定稿，**只差以下 3 条书面回复**即可升级 1.0 EXECUTABLE：

### 确认项 ①：押金对价条款

`challenge_open_access="deposit"` 后押金流向——

- 挑战被 **DISMISSED**（驳回）时：押金没收给谁？（候选答案：模块奖励池 `appmining_reward_pool` / 申损方）
- 挑战 **UPHELD**（成立）时：是否**全额退回** challenger？

请直接定条款（可在选项中选或另给），UGE 会把它写死进提案 metadata，避免执行后解释分歧。

### 确认项 ②：空串 = "open" 等价语义

链上 `challenge_open_access` 现值为**空串**（T1 实现，行为等同 open）。请确认：

- keeper 判断里空串与 `"open"` 等价；
- MsgUpdateParams 写入 `"deposit"` 后**无需重启/迁移**即生效。

### 确认项 ③：生产挑战窗口块数

当前 `challenge_window_blocks=500` 为**联调值**（本次 e2e 实测 deadline=publish+600 秒）。请按发布链实测块速给出**放量用建议值**（口径：≥ 7 天等价块数），并明确：

- 并入 T4 提案同票，还是前置独立提案（UGE 侧建议：独立提案先行，避免 T4 被窗口值争论拖住）。

### 回复方式

在链仓 `workorder/` 落一份回复文档（沿用现有 REPLY 命名惯例，如 `UPTICK-UGE-CHAIN-REPLY-T4-GOV-CONFIRMATIONS-0.1.md`），UGE 侧拿到后 24h 内升级提案包至 1.0。

---

## TASK-3【P1】互验向量与往来文档入库（仓库卫生）

以下文件已在链仓工作区但为 **untracked**，请按链团队入库节奏 commit 到 `uge/v1.1`：

1. `x/appmining/spec/appmining-v1-wire-secp256k1-golden-1.0.json` —— DEC-10 跨实现互验黄金向量（3 条：正路径 + high_s 负向量 + sig_length 负向量；与既有 `app-mining-v1-wire-golden-1.0.json` 并列，spec 目录家族一致性）；
2. `workorder/` 下本轮 UGE 往来文档副本（REQ-D 需求、REPLY、互验确认等）。

### 验收标准

- [ ] 上述文件 git tracked，`git status` 不再出现；
- [ ] 若链侧 CI 有 spec 校验，向量 JSON 通过。

---

## TASK-4【P1，依赖 TASK-1】真网 e2e 配合

UGE 侧将在发布链升级后跑 DEC-10 最终验收，链侧需配合的点：

1. 发布链保持可用（8545/8546/26657/1317 四端点）；
2. UGE 将执行：
   - `register-role-key publisher <key-id> <0xEVM地址> --key-type secp256k1-evm`（注册 secp256k1 publisher 键）；
   - console MetaMask `personal_sign` → `MsgPublishCandidate`（65B r,s,v / v∈{27,28} / low-s）全链路；
   - 故意发 high-s / 64B 签名各一次，确认链上分别回错误码（high-s → 85、长度 → 82 附近）；
3. 若中途出现 82~85 错误码，链侧协助核对 message 与注册行为一致。

---

## 执行顺序建议

```
TASK-1（push+tag+发布链部署）  ──┬──> TASK-4（UGE 跑 e2e）
TASK-2（3 项书面确认，可并行）  ──> UGE 升级提案 1.0 → 链侧提交治理
TASK-3（入库，随时）
```

## 材料索引

| 材料 | 位置 |
|---|---|
| DEC-10 需求原文 | UGE 仓 `docs/workorder/UPTICK-UGE-CHAIN-ACTION-REQUESTS-0.2.md` |
| 链侧 REPLY（ACK） | 链仓 `workorder/UPTICK-UGE-CHAIN-ACTION-REPLY-REQ-D-SECP256K1-0.1.md` |
| 互验确认文档 | UGE 仓 `docs/workorder/UPTICK-UGE-DEC10-INTEROP-CONFIRMATION-0.1.md` |
| 互验向量 | 链仓 `x/appmining/spec/appmining-v1-wire-secp256k1-golden-1.0.json` |
| T4 提案包草案 + JSON | UGE 仓 `docs/workorder/UPTICK-CHAIN-T4-GOV-PROPOSAL-PACKAGE-DRAFT-0.1.md` / `gov-proposal-t4-publisher-binding-enforcement.json` |
| UGE 侧错误码登记（82~85） | UGE 仓 `sdk/javascript/appmining-v1-errors.mjs` |
