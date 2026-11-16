# hik-ipc-plugin（海康 HCNetSDK sidecar 骨架）

通过 HCNetSDK 登录海康设备、控制 PTZ，并以 **RTSP URL** 作为直播流源返回给平台。
平台侧只需把它当作一个 IPC 插件（`protocol = hik.private.v5`）。

> 这是骨架实现：通道枚举固定为 `HIK_CHANNEL_COUNT` 个，取流走设备自带 RTSP；
> 真正的 SDK 取流/回放/报警请在此基础上扩展。

## 1. 准备 SDK

从海康开放平台下载 **HCNetSDK（设备网络 SDK）**，放到本目录：

```text
tools/hik-ipc-plugin/sdk/
├── include/        # HCNetSDK.h 等头文件
└── lib/            # libhnetsdk.so / HCNetSDK.lib / HCNetSDK.dll 等
```

目录内容不入库（见 `sdk/.gitignore`）。运行时需要把动态库放在可执行文件同级或系统库路径：

- Linux：`libhnetsdk.so`、`libhpr.so`、`libHCCore.so`
- Windows：`HCNetSDK.dll`、`HCCore.dll`

## 2. 构建

Linux：

```bash
CGO_ENABLED=1 go build -tags hik -o bin/hik-ipc-plugin ./tools/hik-ipc-plugin
```

Windows（需 MinGW/MSVC 工具链）：

```powershell
$env:CGO_ENABLED=1
go build -tags hik -o bin/hik-ipc-plugin.exe ./tools/hik-ipc-plugin
```

未加 `-tags hik` 时会编译 `main_stub.go`，只打印提示，不影响主仓库 `go build ./...`。

## 3. 运行

```bash
PLATFORM_URL=http://127.0.0.1:18080 \
PLUGIN_ENDPOINT=http://127.0.0.1:19091 \
PLUGIN_LISTEN=:19091 \
HIK_CHANNEL_COUNT=4 \
./bin/hik-ipc-plugin
```

环境变量：

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `PLATFORM_URL` | `http://127.0.0.1:18080` | 平台地址 |
| `PLUGIN_ENDPOINT` | `http://127.0.0.1:19091` | 平台回调本插件的地址（须平台可达） |
| `PLUGIN_LISTEN` | `:19091` | 插件监听地址 |
| `PLUGIN_TOKEN` | 空 | 与平台 `ZWS_PLUGIN_TOKEN` 一致 |
| `HIK_CHANNEL_COUNT` | `1` | 通道数（骨架用） |
| `HIK_RTSP_TEMPLATE` | `rtsp://%s:%s@%s:554/Streaming/Channels/%d` | RTSP 模板，参数顺序 user/pass/ip/channel*100+1 |

## 4. 平台侧使用

1. 启动平台（`make run`），再启动本插件。
2. “接入管理 → 插件接入”应出现 `hik-ipc-01`。
3. 添加设备：IP、SDK 端口（默认 8000）、用户名、密码、默认通道。
4. 添加成功后同步出通道；播放时插件返回海康 RTSP，平台交给 ZMS 拉流。

## 5. 后续扩展点

- `handleChannels`：改用 `NET_DVR_GetDVRConfig` + `NET_DVR_GET_IPPARACFG_V40` 获取真实 IP 通道。
- `handleLiveStart`：改为 `NET_DVR_RealPlay_V40` 回调收 PS 流，按 `rtp_push` 推给平台。
- 报警订阅：`NET_DVR_SetDVRMessageCallBack_V50` → 回调平台 `/api/v1/...`（待平台事件接口）。
- 回放：`NET_DVR_FindFile_V40` / `NET_DVR_PlayBackByTime_V40`。
