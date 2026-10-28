# Uptick 链侧 T1 发版部署记录 0.1

**日期：** 2026-10-28  
**目标链：** `uptick_7777-1`（GCP test01，`34.177.82.11`）  
**指令：** `UPTICK-CHAIN-T1-EXECUTION-DIRECTIVE-0.1` S-6

## 四元一致

| 项 | 值 |
|---|---|
| tag | `v0.5.1-t1`（annotated） |
| commit | `66c9f7873dae87958883f92fa0470ad940c241b7` |
| 二进制 SHA-256 | `201c3d97cbf30ee9f8c91a31b4a50c90496e4995aa2ecd74c40ee4c2dca29750` |
| 在线版本 | LCD `application_version.version=v0.5.1-t1`，`git_commit=66c9f787`；`web3_clientVersion=Version v0.5.1-t1 (66c9f787)` |

## 构建

- 主机：test01 原生 `go1.26.9 linux/amd64`
- 命令：`make build VERSION=v0.5.1-t1 COMMIT=66c9f787 LEDGER_ENABLED=false`
- 源：`git archive 66c9f787` → `~/uptick-src-t1`
- 产物：`/home/ubuntu/bin/uptickd`（BuildDate `2026-10-28T08:20:00Z`）

## 部署

- 替换二进制后 `systemctl restart uptickd`
- 取证高度：`378741`（catching_up=false）
- 端点：8545 / 8546 / 26657 / 1317 均可达

## 备注

首轮失败二进制（手写追加 gRPC methods、无 FileDescriptor）已回滚；`66c9f78` 起 `tx.proto` 正式含 AssessmentSigner RPC 并重新生成 `tx.pb.go`。
