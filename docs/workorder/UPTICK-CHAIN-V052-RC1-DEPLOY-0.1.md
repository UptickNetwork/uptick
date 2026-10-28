# v0.5.2-rc1 发版部署记录（DEC-10）

**日期：** 2026-10-28  
**指令：** `UPTICK-CHAIN-TASK-HANDOFF-2026-10-28-0.1` TASK-1

## 四元一致

| 项 | 值 |
|---|---|
| tag | `v0.5.2-rc1` |
| commit | `85295086c3ccbb5c0a1f589528e8ccf5afa2e8ce`（含 `bde3cb6` DEC-10 + golden/T4 确认） |
| md5 | `1be4033530b0c5ad762fa367987c8f43` |
| 在线版本 | LCD `application_version.version=v0.5.2-rc1`，`git_commit=85295086` |
| VCS | `vcs.revision=85295086…`，`vcs.modified=false` |

## 验证

- `register-role-key --help` 含 `--key-type secp256k1-evm`
- 节点 `catching_up=false`，height≈385339
- 旧备份：`/home/ubuntu/bin/uptickd.v0.5.1-t1.1.bak`

## TASK-4 就绪

发布链 8545/8546/26657/1317 可用；UGE 可开始 DEC-10 真网 e2e。
