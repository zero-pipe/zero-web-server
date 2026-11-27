# hik-ipc-plugin（海康 HCNetSDK sidecar）

通过 HCNetSDK 登录海康设备、读取设备信息、枚举通道、控制 PTZ，并以 **RTSP URL** 作为直播流源返回给平台。

同一套代码同时支持 **Linux64 / Windows64**；差异只在构建时使用的 SDK 包与运行时动态库。

## 1. SDK

准备对应平台的设备网络 SDK：

```text
Linux:   HCNetSDKV6.1.11.30_build20260805_linux64/
Windows: HCNetSDKV6.1.11.30_build20260805_Win64_ZH/
├── 头文件/              # HCNetSDK.h 等
└── 库文件/              # Linux: libhcnetsdk.so ... / Windows: HCNetSDK.dll/.lib ... + HCNetSDKCom/
```

> SDK 不入库。构建脚本会把头文件/动态库复制到 `tools/hik-ipc-plugin/sdk/`（已在 `.gitignore`）。

## 2. 构建

### Linux64

必须在 Linux 环境构建（cgo 不能从 Windows 交叉编译到 Linux）：

```bash
./tools/hik-ipc-plugin/build-linux.sh /path/to/HCNetSDKV6.1.11.30_build20260805_linux64
```

产物：`bin/hik-ipc-plugin`、`bin/hik-lib/`

### Windows64

需要 Go amd64 + MinGW-w64 的 `gcc/g++`、`gendef`、`dlltool`（MinGW binutils）：

```bash
./tools/hik-ipc-plugin/build-windows.sh /path/to/HCNetSDKV6.1.11.30_build20260805_Win64_ZH
```

脚本会把官方 `HCNetSDK.lib/HCCore.lib` 转换成 MinGW 可链接的 `.a`，产物：

- `bin/hik-ipc-plugin.exe`
- `bin/hik-win/`（exe + 全部 DLL + `HCNetSDKCom/`）

> 未加 `-tags hik` 时编译 `main_stub.go`，不影响主仓库 `go build ./...`。

## 3. 运行

### Linux

```bash
PLATFORM_URL=http://127.0.0.1:18080 \
PLUGIN_ENDPOINT=http://192.168.1.10:19091 \
PLUGIN_LISTEN=:19091 \
HIK_SDK_LIB_PATH=$PWD/bin/hik-lib \
LD_LIBRARY_PATH=$PWD/bin/hik-lib \
./bin/hik-ipc-plugin
```

### Windows（Git Bash / cmd）

```bat
cd bin\hik-win
set PLATFORM_URL=http://127.0.0.1:18080
set PLUGIN_ENDPOINT=http://192.168.1.10:19091
set PLUGIN_LISTEN=:19091
hik-ipc-plugin.exe
```

DLL 与 `HCNetSDKCom` 必须和 exe 同目录（`bin/hik-win/` 已按此布局生成）。

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `PLATFORM_URL` | `http://127.0.0.1:18080` | 平台地址 |
| `PLUGIN_ENDPOINT` | `http://127.0.0.1:19091` | 平台回调地址（须平台可达） |
| `PLUGIN_LISTEN` | `:19091` | 插件监听 |
| `PLUGIN_TOKEN` | 空 | 与平台 `ZWS_PLUGIN_TOKEN` 一致 |
| `HIK_RTSP_PORT` | `554` | 设备 RTSP 端口 |
| `HIK_ISAPI_PORT` | `80` | ISAPI 端口（事件流/抓图，不是 SDK 8000） |
| `HIK_ISAPI_HTTPS` | `false` | ISAPI 是否使用 HTTPS |
| `HIK_ISAPI_INSECURE` | `true` | 跳过 ISAPI 证书校验（自签名设备） |
| `HIK_SDK_LIB_PATH` | 空 | 仅 Linux：动态库目录（含 `HCNetSDKCom`、`libcrypto.so.3`、`libssl.so.3`） |
| `HIK_RTSP_TEMPLATE` | 空 | 自定义 RTSP 模板：`rtsp://%s:%s@%s:554/Streaming/Channels/%d`（user/pass/ip/channelPath） |
| `HIK_PTZ_TYPE` | `3` | 通道 PTZ 类型（0 表示不显示云台） |
| `HIK_HAS_AUDIO` | `false` | 通道是否带音频 |

## 4. 平台侧验证

1. 启动平台与插件，进入「接入管理 → 插件接入」，应出现 `hik-ipc-01`（协议 `hik.private.v5`）。
2. 添加设备：IP、SDK 端口（默认 8000）、用户名、密码、默认通道。
3. 成功后同步出真实通道（SDK `NET_DVR_GET_IPPARACFG_V40`，失败时回退到设备通道数）。
4. 进入「设备资产」→ 对应设备 → 视频资源 → 播放：
   - 插件返回 `rtsp://user:pass@ip:554/Streaming/Channels/<ch*100+1>`；
   - 平台交给 ZMS 拉流并生成 flv/ws-flv 等地址；
   - **需要平台已配置在线 ZMS 媒体节点**。
5. PTZ：设备资产/插件页面调用 PTZ，插件通过 `NET_DVR_PTZControlWithSpeed_Other` 控制并自动停止。
6. 布防：插件页面点击「布防」，平台把 MQTT + 对象存储 + 设备内码下发。插件会：
   - 校验对象存储 Bucket 可访问、MQTT 可连接、ISAPI 可登录；
   - 打开 SDK 报警通道（`NET_DVR_SetupAlarmChan_V41`）；
   - 启动 ISAPI `alertStream` 事件流。
7. 事件：收到事件后抓图（`/ISAPI/Streaming/channels/<ch*100+1>/picture`），上传对象存储（`events/<内码>/YYYY/MM/DD/<eventId>.jpg`），再发布 MQTT 事件（`<topicPrefix>/<内码>`），平台消费者订阅并记录日志（Step 4 持久化）。

## 5. 直播取流方式（SDK 私有协议 + RTP/PS 推送）

直播默认走 **SDK 私有协议取流 → RTP/PS 推送**（`transport=rtp_push`），不再像 ONVIF 那样让 ZMS 拉 RTSP：

- 用 `NET_DVR_RealPlay_V40`（`byProtoType=0` 私有协议）向设备取流，回调拿到 **PS 流**；
- 插件把 PS 按 pack 头（`00 00 01 BA`）分帧、打成 RTP/PS，用 UDP 发到 ZMS `OpenRtpServer`
  分配的口（平台经 `/v1/live/start` 的 `rtp:{sdp_ip,rtp_port}` 下发）；
- 设备侧连的是私有协议端口（默认 8000），**不碰 RTSP 554**，因此不与 ONVIF 抢 RTSP 会话；
- ZMS 收流器（`src/protocol/rtp/rtp-ps-unpack.c`）忽略 RTP marker，按「载荷以 `00 00 01 BA`
  开头」分帧，所以每组 RTP 包首包载荷必须从 PS pack 头起始（实现已保证）。

若平台未分配 RTP 口（例如老 ZMS 或未声明 `rtp_push`），插件回退为返回 RTSP URL 由 ZMS 拉流
（`transport=rtsp_url`）。`plugin.yaml` 的 `stream_transports` 声明两种方式，平台优先 `rtp_push`。

## 6. 已知限制

- 直播默认走 SDK 私有协议取流（见第 5 节）；RTSP URL 仅作回退。
- 回放（`/v1/playback/*`）仍走海康按时间回放的 RTSP 地址，由 ZMS 拉流。
- 通道枚举依赖 `NET_DVR_GET_IPPARACFG_V40`；部分设备/权限下会回退为 1..N。
- 事件来源目前仅 ISAPI `alertStream`；HCNetSDK 报警回调作为后续增强。
- 事件只发布到 MQTT，平台尚未持久化（Step 4）。
- PTZ 速度取 `z` 或 `h`（1..7），方向映射见 `main.go`。

## 7. 后续扩展点

- 回放改 SDK 私有协议取流（`NET_DVR_PlayBackByTime_V40` + RTP/PS 推送），彻底摆脱 RTSP；
- `NET_DVR_SetDVRMessageCallBack_V50` 上报报警/AI 事件；
- 抓图可切换到 `NET_DVR_CaptureJPEGPicture`（当前走 ISAPI 抓图）。
