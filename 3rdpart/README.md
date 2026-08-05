# 第三方依赖

本目录存放以源码形式 vendored 的第三方库，通过 `go.mod` 的 `replace` 指令引用。

| 目录 | 模块 | 说明 |
|------|------|------|
| `onvif-go/` | `github.com/0x524a/onvif-go` | ONVIF 发现、设备信息、PTZ 等 |

升级时请在本目录内修改后执行 `go mod tidy` 与回归测试。
