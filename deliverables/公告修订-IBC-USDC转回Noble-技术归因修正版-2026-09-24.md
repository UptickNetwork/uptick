# 公告修订：USDC 转回 Noble（技术归因修正版）

> 2026-09-24 修订。原公告把「旧 IBC token 不能用」归因于「EVM/IBC 迁移后旧通道不兼容」，与实测事实不符。
> 真实机制：IBC 协议层跨版本互通（G-7 实测 11/0），失效点在 **x/erc20 对旧 IBC-ERC20 pair 的支持被移除**。
> 修订原则：对外不暴露内部模块名细节（OWNER_MODULE 等），但归因方向、持有形态区分、版本号必须改对。

---

## 一、修订版英文公告（可直接发布）

**[Important Notice] Transfer USDC on Uptick Chain Back to Noble Before the Network Upgrade**

Dear Uptick Community,

Due to an upcoming upgrade of Uptick Network's underlying modules, we strongly recommend that all users holding USDC on the Uptick chain transfer their USDC back to the Noble chain via officially supported IBC channels **before the upgrade height**. Please act early to avoid any inability to recover your assets after the upgrade.

### Why is this needed?

This upgrade involves two underlying module migrations:

1. **EVM module migration** — Uptick previously used the evmos/evm module, which is no longer actively maintained upstream. Uptick is migrating to cosmos/evm.
2. **IBC module migration** — Uptick's previous IBC implementation was based on a legacy ibc-go v8 fork, and is now migrating to ibc-go v10.

**What actually changes for your USDC — please read carefully:**

- **IBC transfers themselves remain compatible across the upgrade.** Cross-version testing confirmed that IBC transfers work in both directions between the old and new versions, and USDC held as a **native IBC voucher on a Cosmos-style address (uptick1...)** remains a valid balance after the upgrade.
- **What is being discontinued is the legacy ERC20 representation of IBC tokens on EVM addresses.** After the upgrade, the new EVM/erc20 modules no longer recognize the legacy token-pair model used before the upgrade. Converting ERC20 back to a native IBC voucher will be **permanently disabled**. USDC held as an **ERC20 token on an EVM address (0x...)** would become stuck: it can no longer be converted, and therefore can no longer be transferred back to Noble.

In short: the risk is not "old channels stop working" — it is that **the ERC20 form of IBC assets loses its only exit once the upgrade is applied**.

### What do you need to do?

Check how you hold USDC on the Uptick chain:

- **USDC shown on an EVM address (0x...) as an ERC20 token** → this is the affected form. First convert it back to the native IBC voucher, then transfer it back to Noble via IBC. **This must be done before the upgrade.**
- **USDC held as a native IBC voucher on a Cosmos-style address (uptick1...)** → it will remain valid after the upgrade, but we still recommend transferring it back to Noble as soon as possible.

Ways to transfer back:

- Upward Wallet app
- Uptick IBC Transfer Tool: https://myassets.uptick.network/myasset

We recommend first conducting a small test transaction. After confirming receipt, transfer all remaining assets. Please act as soon as possible — **after the upgrade, ERC20-held IBC assets can no longer be recovered**.

### Security reminder

- Official team members will never proactively DM you to ask for your mnemonic phrase, private key, verification code, or request that you transfer funds to a personal address. Please be alert to scams.
- If you encounter issues, please contact support only through official channels. Do not trust unofficial customer service or third-party tools.

Please spread the word among community members. Thank you for your support and cooperation with Uptick Network.

Uptick Network Team

---

## 二、修订版中文公告（对照稿）

**【重要公告】请在网络升级前将 Uptick 链上的 USDC 转回 Noble 链**

亲爱的 Uptick 社区用户：

Uptick Network 即将进行底层模块升级，我们强烈建议所有在 Uptick 链上持有 USDC 的用户，**在升级高度之前**通过官方支持的 IBC 通道将 USDC 转回 Noble 链。请尽早操作，以免升级后资产无法取回。

### 为什么需要转回？

本次升级涉及两项底层模块迁移：

1. **EVM 模块迁移** —— 此前使用的 evmos/evm 模块上游已停止积极维护，Uptick 将迁移至 cosmos/evm；
2. **IBC 模块迁移** —— 此前的 IBC 实现基于旧版 ibc-go v8 分支，本次将迁移至 ibc-go v10。

**对您的 USDC 实际意味着什么——请仔细阅读：**

- **IBC 转账本身跨升级版本仍然互通。** 跨版本实测确认新旧版本之间 IBC 转账双向可用；以**原生 IBC 凭证（voucher）形式保存在 Cosmos 格式地址（uptick1...）上的 USDC，升级后仍是有效余额**。
- **被停用的是 IBC 资产在 EVM 地址上的 ERC20 表达形式。** 升级后，新的 EVM/erc20 模块不再识别旧版代币对模型，**ERC20 转回原生 IBC 凭证的能力将被永久关闭**。以 **ERC20 形式保存在 EVM 地址（0x...）上的 USDC 将无法转出**：既不能转换，也就无法再转回 Noble。

一句话概括：风险不是「旧通道停用」，而是 **ERC20 形式的 IBC 资产在升级后将失去唯一出口**。

### 您需要做什么？

请先确认您在 Uptick 链上的 USDC 持有形式：

- **USDC 显示在 EVM 地址（0x...）上、以 ERC20 代币形式存在** → 这是受影响的形式。请先将其转换回原生 IBC 凭证，再通过 IBC 转回 Noble。**必须在升级前完成。**
- **USDC 以原生 IBC 凭证形式保存在 Cosmos 格式地址（uptick1...）上** → 升级后仍然有效，但我们仍建议尽早转回 Noble。

转回方式：

- Upward Wallet 应用
- Uptick IBC 转账工具：https://myassets.uptick.network/myasset

建议先进行小额测试转账，确认到账后再转出全部剩余资产。请尽快操作——**升级完成后，ERC20 形式的 IBC 资产将无法再取回**。

### 安全提醒

- 官方团队成员绝不会主动私信索要您的助记词、私钥、验证码，或要求您向个人地址转账。请警惕诈骗。
- 如遇问题，请仅通过官方渠道联系支持。请勿轻信非官方客服或第三方工具。

请社区成员相互转告。感谢您对 Uptick Network 的支持与配合！

Uptick Network Team

---

## 三、修订说明（内部参考，不发布）

| # | 原公告表述 | 问题 | 修订后 |
|---|---|---|---|
| 1 | 「旧 EVM 和旧 IBC 通道将不再兼容」「无法通过旧通道转回」 | 与事实相反：跨版本 ICS-20 双向实测 11/0（G-7），旧 voucher 是纯 bank 余额，升级后经任意通道（含新建通道）都能回 Noble | 改为「IBC 转账跨版本互通；失效点是 ERC20 表达形式被停用」 |
| 2 | 归因于「EVM 迁移 + IBC 迁移导致不兼容」 | 真实根因：x/erc20 owner 模型变更（旧 pair 恒拒转换 + 无转换动作）+ 入站注册收紧 ⇒ 升级链 token_pairs=0（硬约束 9/10，活链取证） | 归因指向「ERC20 代币对模型不再被识别、转换能力永久关闭」，不诿过于 IBC |
| 3 | 「iris-v8」「cosmos/ibc-v10」 | 版本号对不上（实测 ibc 6→8、transfer 5→6） | 改为「旧 ibc-go v8 分支 → ibc-go v10」 |
| 4 | 未区分持有形态，全体用户一刀切 | 只有 ERC20 形态用户会锁死；voucher 用户被过度恐吓 | 增加持有形态判别（EVM 地址 0x... vs Cosmos 地址 uptick1...）与分路径操作指引 |
| 5 | 「old assets may not be migratable」 | 模糊，且对 ERC20 用户而言不是 "may not"，是必然锁死 | 改为明确「升级后 ERC20 形式的 IBC 资产无法再取回」 |

保留不动的部分：小额测试建议、防诈骗提醒、官方工具入口、行动紧迫性（原建议方向正确）。

注意事项：
- 发布前请与升级时间表对齐「before the upgrade height」的具体日期/高度。
- myassets 工具与监控需同步 F-25（denom-trace→denom 改名 + schema 变更，查询恒空 rc=0），避免升级当天巡检假绿。
