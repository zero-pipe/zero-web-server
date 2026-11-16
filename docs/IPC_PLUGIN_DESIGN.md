# IPC 私有 SDK 插件化接入设计（Sidecar + RPC）

> 状态：**设计稿（Draft）**，仅文档，未落地代码。
> 目标读者：后端开发、插件开发、联调测试。
> 关联：[ARCHITECTURE.md](ARCHITECTURE.md)、[CONFIG.md](CONFIG.md)。

## 1. 背景与目标

许多 IPC（网络摄像机）厂商只提供 **C/C++ 私有 SDK**（如海康 HCNetSDK、大华 NetSDK），调用存在位数、运行时、崩溃风险、闭源分发等约束。若用 cgo 直接嵌入平台：

- C 段错误会带走整个 Go 进程；
- 32 位 SDK 无法在 64 位进程中加载；
- 交叉编译、Docker、许可分发都变得复杂。

因此采用 **Sidecar 插件**：每个厂商 SDK 封装为独立进程（`ipc-plugin-<vendor>`），通过 **RPC** 与平台交互。平台不直接接触 C SDK。

**目标**

1. 平台可动态接入多种私有协议，新增厂商只新增插件进程，不改平台 HTTP/前端主体。
2. 设备配置以平台为唯一事实源，插件无状态（仅会话缓存）。
3. 取流统一由流媒体服务（ZMS）输出多协议播放地址，插件只提供“流源”。
4. 插件崩溃、离线、重启不影响平台主进程。

**非目标（本期）**

- 不实现具体厂商 SDK 封装；
- 不改造现有 GB28181 / ONVIF 链路；
- 不涉及插件市场、计费、多租户隔离。

## 2. 设计原则

1. **平台为事实源**：设备、通道、凭据、权限存平台；插件重启后可重新登录恢复。
2. **插件无业务数据**：插件不做持久化，不生成最终播放 URL。
3. **能力协商**：插件注册时上报协议、能力清单与设备配置 Schema；平台据此动态渲染表单与功能入口。
4. **流源与播放解耦**：插件返回 `rtsp_url` / `rtp_push` 等流源，播放地址由平台 + ZMS 生成。
5. **可降级**：RPC 超时、插件离线时，相关设备置离线、流清理，平台其余功能不受影响。

## 3. 总体架构

```text
┌────────────────────────────┐
│ Web / 第三方调用            │
└─────────────┬──────────────┘
              │ HTTP /api/v1
┌─────────────▼───────────────────────────────┐
│ zero-web-server (Go)                         │
│  ├─ interfaces/http       设备/插件管理 API   │
│  ├─ application/plugin    插件注册表/心跳     │
│  ├─ application/deviceaccess  ProtocolAdapter│
│  ├─ port/IPCPlugin        gRPC 客户端端口     │
│  ├─ adapter/ipcsidecar    gRPC 实现          │
│  └─ port/MediaCluster ───────────────┐       │
└─────────────┬────────────────────────┼───────┘
              │ gRPC (双向/服务端流)     │ HTTP API
┌─────────────▼──────────────┐   ┌─────▼──────────────┐
│ ipc-plugin-hik (sidecar)    │   │ zero-media-server  │
│  ├─ RPC server              │   │  拉流/转封装/录像   │
│  └─ cgo → HCNetSDK          │   └─────▲──────────────┘
└─────────────┬──────────────┘         │ RTSP pull / RTP push
              │ 私有协议               │
┌─────────────▼──────────────┐         │
│ 海康 IPC                    │─────────┘
└────────────────────────────┘
```

## 4. 角色与职责

| 角色 | 职责 | 不负责 |
| --- | --- | --- |
| 平台 `zero-web-server` | 插件注册表、设备/通道/凭据、权限、播放地址、录像计划、报警入库 | 不直接调用厂商 SDK |
| 插件 `ipc-plugin-*` | 加载 SDK、登录设备、取流、PTZ、事件回调、向平台上报状态 | 不做持久化、不生成最终播放 URL |
| 流媒体 `ZMS` | 拉流/收流、转封装、多协议输出、录像 | 不感知厂商协议 |
| 设备 | 提供视频流与控制能力 | — |

## 5. 插件注册与生命周期

### 5.1 注册（Plugin → Platform）

插件启动后调用平台 `PluginRegistry.Hello`：

```text
plugin_id:        hik-sidecar-01            # 实例唯一 ID（建议 hostname/进程名）
protocols:        ["hik.private.v5"]        # 支持的协议/版本命名空间
vendor:           "hikvision"
version:          "0.1.0"
capabilities:     ["live","ptz","preset","alarm","playback"]
stream_transports:["rtsp_url","rtp_push"]   # 支持的取流方式
config_schema:    { JSON Schema }           # 设备配置表单（ip/port/user/pass/channel...）
endpoint:         "10.0.0.5:50051"          # 插件 RPC 地址
heartbeat_sec:    10
```

平台返回 `plugin_instance_id` 与 `heartbeat_interval`。随后插件按间隔调用 `Heartbeat`。

### 5.2 状态与离线处理

- 平台记录 `zws_plugin`：`last_seen`、`status`、`version`。
- 连续 N 次心跳缺失 → 插件 `offline`：
  - 其名下设备置 `offline`；
  - 该插件发起的流标记为需清理（平台调用 ZMS 关流并通知插件 `StopLive`，失败则记录）。
- 插件重连后重新 `Hello`，平台下发其名下设备清单供重新登录（可选，推荐由平台在首次 `StartLive` 时按需登录）。

### 5.3 部署形态

- **独立部署（推荐）**：插件单独进程/容器，平台通过 endpoint 访问；便于多实例、灰度、隔离。
- **平台托管（可选）**：平台拉起插件进程并注入 endpoint/token，适合单机部署；需做进程守护与日志回收。

## 6. 设备配置：动态 Schema

平台“添加设备”表单由插件 `config_schema` 驱动，避免每接一家厂商就改前端。

- `driver_params`（JSON）保存厂商特有字段，不与公共字段混表。
- 公共字段：`name`、`protocol`、`plugin_id`、`enabled`、`driver_params`。
- 凭据（账号/密码）加密存储，接口返回时脱敏。

示例 `config_schema`（节选）：

```json
{
  "type": "object",
  "required": ["ip", "port", "username", "password"],
  "properties": {
    "ip":       { "type": "string", "title": "设备 IP" },
    "port":     { "type": "integer", "title": "SDK 端口", "default": 8000 },
    "username": { "type": "string", "title": "用户名" },
    "password": { "type": "string", "title": "密码", "format": "password" },
    "channel":  { "type": "integer", "title": "默认通道", "default": 1 }
  }
}
```

## 7. RPC 规范

### 7.1 传输与版本

- **推荐 gRPC + protobuf**；插件用 Go+cgo 时成本最低。
- 若插件为 C++ 进程且希望减少依赖，可退化为 **JSON-RPC over HTTP**（语义保持一致）。
- 同机部署可用 Unix Domain Socket。
- 所有请求带 `api_version`（如 `v1`），平台与插件做兼容校验。

### 7.2 公共约定

| 项 | 约定 |
| --- | --- |
| 请求 ID | 每个请求带 `request_id`，日志与错误回传关联 |
| 超时 | 平台侧默认 5s，`StartLive`/`Login` 可 15s |
| 幂等 | `StartLive` 对同一 `(device, channel, transport)` 返回同一流源 |
| 错误 | 统一 `code`（枚举）+ `message` + `retryable` |
| 大数据 | 仅传地址/参数，不传视频帧（视频走 RTSP/RTP） |

### 7.3 方法集（v1）

| 分类 | 方法 | 请求要点 | 响应要点 |
| --- | --- | --- | --- |
| 生命周期 | `Hello` | 见 5.1 | `plugin_instance_id`, `heartbeat_interval` |
| | `Heartbeat` | `plugin_instance_id` | `ok` |
| | `Capabilities` | — | 能力/Schema（可随 Hello 返回） |
| 设备 | `Login` | `device_uid`, `driver_params`, `credentials` | `session_id`, `device_info` |
| | `Logout` | `device_uid` | `ok` |
| | `GetDeviceInfo` | `device_uid` | 厂商/型号/固件/序列号 |
| | `ListChannels` | `device_uid` | `channels[]`（稳定 channel_id/名称/是否 PTZ） |
| 直播 | `StartLive` | `device_uid`, `channel_id`, `transport`, `stream_key` | `source`（见 8） |
| | `StopLive` | `device_uid`, `channel_id`, `stream_key` | `ok` |
| 回放 | `QueryRecord` | 时间范围 | `records[]` |
| | `StartPlayback` | 时间范围, `stream_key` | `source` |
| | `PlaybackControl` | `pause/resume/speed/seek` | `ok` |
| 控制 | `PTZ` | 方向/速度 | `ok` |
| | `Preset` | `query/set/goto/delete` | 结果 |
| | `Snapshot` | `channel_id` | 图片地址或二进制（建议返回可拉取 URL） |
| 事件 | `OnAlarm` | 平台回调，插件主动调用 | `ok` |
| | `OnDeviceStatus` | 上下线、通道变化 | `ok` |
| | `OnStreamState` | 流异常/中断 | `ok` |

### 7.4 流源（`source`）

```text
oneof source {
  RtspURL  rtsp_url   // 插件提供 RTSP 地址，ZMS 拉流
  RtpPush  rtp_push   // 平台开端口，插件推 RTP/PS
  RtmpPush rtmp_push  // 插件推 RTMP
}
```

**平台不接收插件生成的最终播放 URL。**

### 7.5 错误模型（节选）

| code | 含义 | 处理 |
| --- | --- | --- |
| `OK` | 成功 | — |
| `INVALID_ARGUMENT` | 参数错误 | 不回设备，直接返回 |
| `DEVICE_UNREACHABLE` | 网络/SDK 连接失败 | 设备置离线，可重试 |
| `AUTH_FAILED` | 用户名/密码错误 | 不重试，提示用户 |
| `CHANNEL_NOT_FOUND` | 通道不存在 | 同步通道 |
| `UNSUPPORTED` | 不支持的能力/传输方式 | 平台按能力降级 |
| `INTERNAL` | 插件内部错误（含 SDK 崩溃前） | 记录 request_id，告警 |

## 8. 取流与播放

### 8.1 优先级

1. **插件能给 RTSP URL** → 平台调用 `MediaCluster.AddStreamProxy`，由 ZMS 拉流（复用现有 ONVIF 播放链路）。
2. 只能出裸流 → 插件声明 `rtp_push`：平台 `OpenRtpServer` 获取端口，插件推 RTP/PS，复用 GB28181 收流能力。
3. 其他（RTMP 等）按需扩展。

### 8.2 播放地址

平台基于 `MediaEndpoint.StreamPlayURLs(app, stream, ...)` 生成 `flv / ws-flv / hls / rtsp / webrtc`，并按 `ResolveForStream` 做节点粘滞与负载。

### 8.3 流标识约定

```
app    = "ipc"
stream = "<plugin_id>_<device_uid>_<channel_id>"
```

- 同一 `app/stream` 始终落到同一媒体节点；
- 平台维护 `stream_key -> (device, channel, source)` 映射，用于停止与清理。

### 8.4 生命周期

| 事件 | 处理 |
| --- | --- |
| 用户点播 | `StartLive` → `AddStreamProxy` → 返回播放地址 |
| 无人观看（`on_stream_none_reader`） | 默认调 `StopLive` 并关流；可在配置中关闭 |
| 插件心跳丢失 | 标记流待清理，调 ZMS 关流 |
| 设备离线事件 | 停止该设备所有流 |
| 平台重启 | 以 ZMS 实际流为准做对账，清理孤儿流 |

### 8.5 录像

- 云录像：由 ZMS `enable_mp4` 录制，平台 `on_record_mp4` 入库（现有链路）。
- 录像计划：需把插件通道纳入 `zws_device_channel`（新增 `data_type=5`），复用现有计划/云录能力。

## 9. 平台侧模块设计

### 9.1 新增/调整

| 层 | 模块 | 说明 |
| --- | --- | --- |
| domain | `domain/plugin` | 插件实例、设备、通道实体与仓储接口 |
| port | `port/ipcplugin.go` | `IPCPlugin` RPC 客户端端口（Hello/Heartbeat/Login/StartLive...） |
| adapter | `adapter/ipcsidecar` | gRPC/JSON-RPC 实现 |
| application | `application/plugin` | 注册表、心跳巡检、插件/设备编排 |
| application | `deviceaccess` | 新增 `ipcAdapter`；扩展能力接口 |
| interfaces | HTTP | 插件管理、设备 CRUD、动态 Schema、播放/PTZ |
| infrastructure | persistence | 插件/设备/通道仓储；凭据加解密 |
| shared | `ChannelDataTypeIPCPlugin = 5` | 复用 `zws_device_channel` |

### 9.2 `ProtocolAdapter` 能力扩展（设计）

现有 `ProtocolAdapter` 只有 List/Get/Create/Update/Delete/Sync。建议增加可选能力接口，`deviceaccess` 按能力断言：

```go
type LivePlayer interface {
    StartLive(ctx context.Context, deviceUID, channelID string) (*StreamSource, error)
    StopLive(ctx context.Context, deviceUID, channelID string) error
}

type Playbacker interface { QueryRecord(...); StartPlayback(...); PlaybackControl(...) }
type PTZController interface { PTZ(...); Preset(...) }
type EventSource interface { /* 由插件回调，平台注册 */ }
```

GB/ONVIF 适配器可按需实现，未实现的协议前端隐藏对应入口。

### 9.3 数据表（草案）

```sql
CREATE TABLE zws_plugin (
  id                VARCHAR(64) PRIMARY KEY,  -- plugin_instance_id
  plugin_id         VARCHAR(64) NOT NULL,
  vendor            VARCHAR(64),
  version           VARCHAR(32),
  protocols_json    TEXT,
  capabilities_json TEXT,
  config_schema     TEXT,
  endpoint          VARCHAR(255),
  status            TINYINT DEFAULT 0,         -- 0 offline / 1 online
  last_seen         BIGINT DEFAULT 0,
  create_time       VARCHAR(50),
  update_time       VARCHAR(50)
);

CREATE TABLE zws_plugin_device (
  id            BIGINT AUTO_INCREMENT PRIMARY KEY,
  internal_code VARCHAR(32) UNIQUE,
  plugin_id     VARCHAR(64) NOT NULL,
  protocol      VARCHAR(64) NOT NULL,
  device_uid    VARCHAR(128) NOT NULL,         -- 插件侧稳定设备 ID
  name          VARCHAR(255),
  enabled       TINYINT DEFAULT 1,
  online        TINYINT DEFAULT 0,
  driver_params TEXT,                          -- 厂商字段 JSON
  credentials   TEXT,                          -- 加密后的账号密码
  create_time   VARCHAR(50),
  update_time   VARCHAR(50),
  UNIQUE KEY uk_plugin_device (plugin_id, device_uid)
);

CREATE TABLE zws_plugin_channel (
  id             BIGINT AUTO_INCREMENT PRIMARY KEY,
  internal_code  VARCHAR(32) UNIQUE,
  device_db_id   BIGINT NOT NULL,
  channel_id     VARCHAR(64) NOT NULL,         -- 插件侧稳定通道 ID
  name           VARCHAR(255),
  status         VARCHAR(32),
  ptz_type       INT DEFAULT 0,
  has_audio      TINYINT DEFAULT 0,
  create_time    VARCHAR(50),
  update_time    VARCHAR(50),
  UNIQUE KEY uk_plugin_channel (device_db_id, channel_id)
);
```

> 也可先复用 `zws_device_channel`（`data_type=5`、`data_device_id` 指向插件设备），以减少录像计划改造量；两种方案在落地前二选一。

## 10. 设备与通道模型

- `device_uid`：插件保证稳定（建议用厂商序列号/设备 ID，而非 IP）。
- 通道 `channel_id` 必须稳定；变更通过 `OnDeviceStatus`/`ListChannels` 同步，平台按 `channel_id` 对齐，避免录像计划错位。
- 状态机：`pending(待登录) → online → offline → disabled`；`disabled` 不参与自动登录与报警。

## 11. 安全

- 插件注册与 RPC 使用 **mTLS** 或共享 Token；平台只接受白名单插件。
- 凭据加密存储（KMS/本地主密钥），RPC 传输走 TLS，日志脱敏。
- 限制 RPC 请求大小、并发与超时；对插件做熔断。
- 平台不向插件下发无关设备/凭据（按需最小化）。
- 插件进程以最小权限运行，限制可访问网段。

## 12. 可观测性与运维

- 每个 RPC 带 `request_id`，平台与插件日志可关联。
- 指标：插件在线数、RPC 成功率/延迟、登录失败、取流成功率、活跃流数、插件重启次数。
- 插件崩溃：由进程守护（systemd/Docker restart）拉起；平台侧标记离线并清理流。
- 版本兼容：`api_version` + 能力协商；新旧插件可并存。

## 13. 多插件与多节点

- 同一协议可有多个插件实例（多机器/多网段）；设备与实例做**亲和**（设备所在网段可达哪个插件）。
- 选实例策略：显式绑定优先 → 同网段 → 负载最低 → 随机。
- 平台记录 `device -> plugin_instance`，会话失效时优先在原实例重登。

## 14. 海康验证计划（MVP）

> 用真实海康摄像机验证“平台链路”，先绕开 SDK 取流复杂度，再补 SDK 能力。

### 阶段 A：RTSP 直连（1～2 天）
1. 不写插件：平台“拉流代理”填海康 RTSP（如 `rtsp://user:pass@ip:554/Streaming/Channels/101`），验证 ZMS 拉流与前端播放。
2. 验证点：flv/ws-flv/webrtc 播放、无人观看关流。

### 阶段 B：最小插件（3～5 天）
1. 插件 `Hello` 上报 `hik.private.v5` + capabilities + config_schema。
2. 平台插件管理可见在线；添加设备表单由 schema 渲染。
3. `Login`（可用 SDK `NET_DVR_Login_V40`）→ `ListChannels`。
4. `StartLive` 返回 `rtsp_url`（可先由插件拼海康 RTSP，后续再换 SDK 取流）。
5. 平台 `AddStreamProxy` → 返回平台生成的播放地址，前端播放成功。
6. `StopLive`、心跳丢失 → 设备离线/流清理。

### 阶段 C：SDK 能力补齐
1. PTZ / 预置位（HCNetSDK）。
2. 报警事件 `OnAlarm`（移动侦测/遮挡等）→ 平台报警入库。
3. 可选：回放 `QueryRecord`/`StartPlayback`。
4. 断线重连、SDK 崩溃后插件重启恢复。

### 验收标准
- 添加/删除海康设备，通道同步正确；
- 点播成功率、首帧时间满足要求；
- 插件重启/断网后平台状态正确、无孤儿流；
- 录像计划/云录可用（若纳入本期）。

## 15. 开放问题（落地前需定）

1. 插件部署形态：平台托管 vs 独立部署（影响进程守护与配置下发）。
2. RPC 选型：gRPC 还是 JSON-RPC（取决于插件实现语言与团队偏好）。
3. 通道表：新建 `zws_plugin_channel` 还是复用 `zws_device_channel`（影响录像计划改造量）。
4. 凭据加密方案与密钥管理。
5. 回放/录像是否纳入首期。
6. 多插件实例的亲和与调度策略。

## 16. 变更记录

| 日期 | 说明 |
| --- | --- |
| 2026-11-16 | 初稿：sidecar + RPC 架构、方法集、海康 MVP |
