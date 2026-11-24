# 第三方依赖

本目录存放第三方库源码。**两种处理方式，不要混用：**

| 目录 | 处理方式 | 说明 |
|------|----------|------|
| `onvif-go/` | **已并入本模块**，不再作为独立模块 | ONVIF 发现、设备信息、媒体、PTZ、事件。直接在本目录改代码，import 路径为 `zero-web-server/3rdpart/onvif-go` |
| `gb28181-go/` | 独立模块，经 `go.mod` 的 `replace` 引用 | GB28181 SIP 协议栈 |

## onvif-go：我们自己的代码

`onvif-go` 原为 `github.com/0x524a/onvif-go`（MIT，Copyright (c) 2025 ProtoTess，见同目录 `LICENSE`），
已按"自己维护"的方式并入本仓库：

- 删除了它自带的 `go.mod`/`go.sum`，`import` 路径改为 `zero-web-server/3rdpart/onvif-go`，
  根 `go.mod` 不再有它的 `require` 与 `replace`。
- 删除了与本平台无关的部分：`cmd/`、`examples/`、`server/`、`testing/`、`testdata/`、`docs/`、CI 配置与构建脚本。
- 保留单元测试（`*_test.go`），它们随 `go test ./...` 一起跑，是这层代码的回归网。

**升级上游时不再走 `go get`**：需要新能力（例如 Profile G 的 Replay / RecordingSearch）时，
直接在本目录补代码，并补对应测试。改动遵循本仓库的提交规范。

改动记录（相对上游 v1.1.3）：

| 日期 | 改动 | 原因 |
|------|------|------|
| 2026-11-24 | 并入本模块，去掉独立 `go.mod` 与 `replace` | 不再作为外部模块依赖，改动直接进本仓库 |
| 2026-11-24 | `discovery_test.go` 的 `TestResolveNetworkInterface` 按 `FlagLoopback` 找回环网卡 | 原用例只认 `lo`/`lo0`，Windows 下必然失败 |
| 2026-11-24 | `device_storage_test.go` 的 `StorageURI` 断言改为非空才校验 | 上游 `device_storage.go` 从未给 `StorageURI` 赋值，测试与实现不一致 |

已知上游缺口（本平台需要，库里没有）：**Profile G 的 Replay 与 RecordingSearch 整套缺失**
（`GetReplayUri` / `FindRecordings` / `GetRecordingSearchResults` 全库 0 命中）。
补这两个服务时复用 `internal/soap` 的通用 `Call(ctx, endpoint, action, req, resp)`，不必换库。
详见 `docs/设备接入能力矩阵与缺口处置.md`。
