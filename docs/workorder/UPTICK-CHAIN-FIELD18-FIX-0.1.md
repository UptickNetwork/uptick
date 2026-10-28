# FIELD18 修复记录 0.1

**指令：** `UPTICK-CHAIN-FIELD18-FIX-DIRECTIVE-0.1`  
**日期：** 2026-10-28

## 根因

`x/appmining/types/types.pb.go` 中 **Go Unmarshal 已含 Params case 18**，但 **FileDescriptor 未同步再生**。  
CheckTx 走 `codec/unknownproto.RejectUnknownFieldsStrict`，按 FileDescriptor 判 TagNum 18 为 unknown → code 2。  
空 allowlist 不在 wire 上出现 tag 18，故空值对照必过。

## 修复

- `buf generate` 再生 `types.pb.go` / `query.pb.go`
- 删除手写 `assessment_pb.go`；`assessment.go` 仅保留 helper
- 回归：`TestParams_Field18AllowlistRoundTripAndDescriptor`

## 四元一致

| 项 | 值 |
|---|---|
| tag | `v0.5.1-t1.1` |
| commit | `e6b35e166df5d4097c8ea701b136bc307e076a99` |
| md5 | `c9537b8736abb6216cb691817e1c2d7d` |
| 在线版本 | `v0.5.1-t1.1` / git `e6b35e16` |
| `go version -m` | `vcs.revision=e6b35e16…`，`vcs.modified=false` |

## S-5

- `update-params` 带 `["uge-assessment-0.1"]` → **code 0**
- tx：`C4DD976D3BF76C6994D21D2960FD60BF8BBD7A6CA9D87672D02E1DB504B3E633`
- 查询回读：`['uge-assessment-0.1']`
- 旧二进制备份：`/home/ubuntu/bin/uptickd.v0.5.1-t1.bak`
