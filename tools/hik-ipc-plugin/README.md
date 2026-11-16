# hik-ipc-plugin（海康 HCNetSDK sidecar）

通过 HCNetSDK 登录海康设备、读取设备信息、枚举通道、控制 PTZ，并以 **RTSP URL** 作为直播流源返回给平台。

## 1. SDK

使用海康设备网络 SDK Linux64 包（例如 `HCNetSDKV6.1.11.30_build20260805_linux64`），目录包含：

```text
HCNetSDK_linux64/
├── 头文件/              # HCNetSDK.h 等
└── 库文件/              # libhcnetsdk.so / libHCCore.so / libhpr.so / HCNetSDKCom/
```

> SDK 不入库。构建脚本会把头文件/动态库复制到 `tools/hik-ipc-plugin/sdk/`（已在 `.gitignore`）。

## 2. 构建（Linux + CGO）

必须在 Linux 环境构建（cgo 不能从 Windows 交叉编译到 Linux）：

```bash
./tools/hik-ipc-plugin/build-linux.sh /path/to/HCNetSDKV6.1.11.30_build20260805_linux64
```

产物：

- `bin/hik-ipc-plugin`
- `bin/hik-lib/`（运行时动态库）

未加 `-tags hik` 时编译 `main_stub.go`，不影响主仓库 `go build ./...`。

## 3. 运行

```bash
PLATFORM_URL=http://127.0.0.1:18080 \
PLUGIN_ENDPOINT=http://192.168.1.10:19091 \
PLUGIN_LISTEN=:19091 \
HIK_SDK_LIB_PATH=$PWD/bin/hik-lib \
LD_LIBRARY_PATH=$PWD/bin/hik-lib \
./bin/hik-ipc-plugin
```

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `PLATFORM_URL` | `http://127.0.0.1:18080` | 平台地址 |
| `PLUGIN_ENDPOINT` | `http://127.0.0.1:19091` | 平台回调地址（须平台可达） |
| `PLUGIN_LISTEN` | `:19091` | 插件监听 |
| `PLUGIN_TOKEN` | 空 | 与平台 `ZWS_PLUGIN_TOKEN` 一致 |
| `HIK_RTSP_PORT` | `554` | 设备 RTSP 端口 |
| `HIK_SDK_LIB_PATH` | 空 | 动态库目录（含 `HCNetSDKCom`、`libcrypto.so.3`、`libssl.so.3`）；不设时依赖 `LD_LIBRARY_PATH` |
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

## 5. 已知限制

- 直播走设备 RTSP，不走 SDK 私有码流；NVR 下挂通道若 RTSP 路径/凭据不同，需用 `HIK_RTSP_TEMPLATE` 调整。
- 通道枚举依赖 `NET_DVR_GET_IPPARACFG_V40`；部分设备/权限下会回退为 1..N。
- 回放、报警订阅、抓图尚未接入插件 RPC。
- PTZ 速度取 `z` 或 `h`（1..7），方向映射见 `main.go`。

## 6. 后续扩展点

- `NET_DVR_RealPlay_V40` 回调收 PS 流，按 `rtp_push` 推给平台；
- `NET_DVR_FindFile_V40` / `NET_DVR_PlayBackByTime_V40` 做回放；
- `NET_DVR_SetDVRMessageCallBack_V50` 上报报警/AI 事件；
- 抓图 `NET_DVR_CaptureJPEGPicture`。
