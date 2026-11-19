# ZWS 平台架构约定（定型稿）

> 状态：设计约定，待评审后冻结。本文只定义目标框架、命名、边界与迁移顺序，不代表已实现。
> 关联：[ARCHITECTURE.md](ARCHITECTURE.md)、[UNIFIED_IDENTITY_DESIGN.md](UNIFIED_IDENTITY_DESIGN.md)、[EVENT_PIPELINE_DESIGN.md](EVENT_PIPELINE_DESIGN.md)、[IPC_PLUGIN_DESIGN.md](IPC_PLUGIN_DESIGN.md)。

## 0. 一句话

ZWS 只做**平台与业务**，协议全部通过**标准接入适配器**或**zero-agent 托管的插件进程**接入；设备、通道、事件、报警使用**一套稳定内码与一套统一模型**，页面与 API 不再按协议分叉。

## 1. 范围与非目标

### 目标

- 接入面收敛为：GB28181、ONVIF、插件（企标/私有协议）。
- 设备/通道/资源/能力/事件/报警有唯一概念模型。
- 插件安装、启动、启停、观测由 zero-agent 统一托管。
- 视频播放路径、事件关联、报警联动有稳定约定。
- MySQL / Redis / MQTT / MinIO 作为固定基础设施。

### 非目标（本期）

- 不实现地标/部标（JT/T 1078 等），只保留扩展位。
- 不做多协议同时接入同一物理设备（此前已确定 1 设备 1 接入）。
- 不做插件市场、在线计费、跨租户隔离。
- 不要求一次替换旧表/旧 API；采用兼容迁移。

## 2. 当前状态盘点（要点）

| 领域 | 现状 | 问题 |
| --- | --- | --- |
| 标准接入 | GB28181 + ONVIF 完整链路；设备门面聚合；`AccessMode` 区分主动/被动 | 页面/API 仍按 protocol 分支，`id/rawId/dbId/gbId/internalCode` 混用 |
| 插件接入 | JSON-RPC sidecar；mock/海康插件可注册、布防、发事件；插件自报能力 | 插件手工启动；无 Agent 托管/插件目录/清单/启停/日志 |
| 内码 | `pkg/idcode`：`D`/`C` + 16 hex；GB/ONVIF/插件设备与通道都有内码；重同步按协议键保留 | `Resource` 内码来自 legacy 复制；ONVIF/插件内码为空时流名回退协议标识 |
| 实时流名 | **已统一** `live/<deviceInternalCode>_<channelInternalCode>` | ONVIF/插件保留协议标识兜底流名 |
| 回放/下载流名 | **未统一**：`DeviceID_GBDeviceID_playback` / `_download` | 与实时流规则冲突；Stop 缺省同样用协议标识 |
| 资产模型 | 统一表已建并幂等投影；legacy 六表是事实源，统一表是 read model；改名保留 | 在线状态是快照；`EventSource` 仅 GB；Capability 有三套表示 |
| 能力 | 资产 Capability（device/resource）、deviceaccess `[]string`、插件 Hello `[]string` | 三套语义；profile 级未落数据；available 仅在 backfill 时计算 |
| 事件/报警 | MQTT→事件→报警投影；事件有 bbox/图片；报警页有回放 | 事件类型靠代码硬映射；GB 直报告警不产生事件；报警无 bbox/资源引用；事件无回放联动 |
| 前端 | 标准接入/插件/设备资产/通道列表四套并列表格；已有统一通道面板 | 无设备→通道树；通道页内伪下钻；事件/报警两页未打通 |
| 基础设施 | MySQL/Redis/MQTT/MinIO 配置页面已统一 | 无需新增基础设施类型 |

## 3. 接入约定

### 3.1 三类接入

| 类型 | 范围 | 实现方式 |
| --- | --- | --- |
| 标准接入 | GB28181、ONVIF | 平台内协议适配器，遵守标准协议 |
| 国标扩展 | 地标/部标 | 暂不实现，只保留 `protocol` 扩展位 |
| 插件接入 | 企标、私有 SDK、厂商私有协议 | zero-agent 托管独立进程，通过 RPC 与平台交互 |

原则：**协议细节不进入业务层**。标准协议通过端口/适配器隔离，私有协议通过进程隔离。

```text
GB28181/ONVIF ── 平台内适配器 ──┐
                                ├── 统一端口/统一资产模型 ── 应用/领域
私有协议 ── 插件进程 ── RPC ── zero-agent ── 平台
```

### 3.2 zero-agent

#### 定位

zero-agent 是安装在设备侧/边缘侧的**常驻后台服务**，负责：

- 扫描并加载 `plugins/` 目录；
- 启动/停止/重启/守护插件进程；
- 向平台注册 Agent 与插件清单；
- 接收平台下发的启用/禁用、配置、日志、升级命令；
- 默认作为插件的反向代理，使平台只和 Agent 通信。

#### 进程与目录

```text
zero-agent/
├── bin/zero-agent(.exe)
├── config/agent.yaml
├── plugins/
│   └── hik-ipc/
│       ├── plugin.yaml          # 清单
│       ├── hik-ipc-plugin(.exe)
│       ├── schema.json          # 设备动态表单
│       └── LICENSE
├── logs/
└── data/
```

#### 插件清单 `plugin.yaml`

```yaml
id: hik-ipc                 # 插件类型，平台内唯一
name: 海康 IPC
vendor: hikvision
version: 0.2.0
protocols: [hik.private.v5]
executable:
  linux: hik-ipc-plugin
  windows: hik-ipc-plugin.exe
args: []
listen: "127.0.0.1:19091"
health: /health
capabilities: [live, playback, ptz, preset, snapshot, alarm, arming, mqtt]
stream_transports: [rtsp_url, rtp_push]
config_schema: schema.json
event_types: [motion, line_crossing, intrusion, video_loss, tamper]
checksum: sha256:...
```

- `id` 是插件**类型**，不是实例；实例 ID 由 Agent 生成：`{agentId}-{pluginId}`。
- 清单变更（版本、能力、schema）由 Agent 上报平台，平台展示“待重启/待启用”。
- 清单必须可校验（checksum/签名），禁止执行未登记二进制。

#### 控制面与数据面

```text
平台 zws  ──HTTPS 注册/心跳（Agent 主动外连）──  zero-agent
                                                          ├─ plugin A（127.0.0.1:port）
                                                          └─ plugin B（127.0.0.1:port）
平台调用 Agent：HTTPS（启停/状态）
平台调用插件：统一经 Agent 反代 /plugins/{pluginId}/{rpcPath}
```

**只支持 Agent 托管 + 反代**：

- 平台不需要知道插件端口，插件只监听 localhost；
- Agent 的 `endpoint` 必须平台可达；Agent 主动向平台注册/心跳；
- 启停、日志、健康统一由 Agent 负责；
- 不提供插件直连平台模式，插件不得自注册到平台。

#### 平台地址与 token

- Agent 配置 `platform_url` 与 `token`；安装时由部署包/注册码写入，插件不保存平台地址。
- Agent 注册/心跳响应返回插件 `desiredState`，Agent 据此对账本地进程；
- 插件启动参数只由 Agent 注入：`PLUGIN_LISTEN`、`PLUGIN_ID`、`PLUGIN_INSTANCE_ID`、`PLUGIN_TOKEN`；
- 插件不硬编码平台地址、不直接访问平台 API。

#### Agent 生命周期

1. 启动 → 读配置 → 扫描 `plugins/*/plugin.yaml`；
2. 校验清单与二进制 → 上报 `discovered`；
3. 平台下发 desired state（enabled/disabled）；
4. Agent 启动 enabled 插件，健康检查，进入 `running`；
5. 插件异常退出 → 指数退避重启，上报 `error` + 日志；
6. 平台禁用 → Agent 先停流/撤防（由平台业务侧协调），再停进程，状态 `disabled`；
7. Agent 自身断线 → 平台标记 Agent 离线，其插件/设备进入离线态；
8. 恢复后重新对账（插件状态、版本、能力）。

#### Agent 与平台 API（草案）

```text
POST /api/v1/agents/register            # agent_id/host/endpoint/plugins
POST /api/v1/agents/heartbeat           # 状态、插件状态、desired 对账
管理员：
GET  /api/v1/agents
GET  /api/v1/agents/{id}/plugins
POST /api/v1/agents/{id}/plugins/{pluginId}/enable
POST /api/v1/agents/{id}/plugins/{pluginId}/disable
POST /api/v1/agents/{id}/plugins/{pluginId}/restart
```

#### 数据模型

- `zws_agent`：id、hostname、ip、version、status、last_seen、token_hash。
- `zws_agent_plugin`：agent_id、plugin_id、manifest_json、enabled、desired_state、actual_state、version、last_error、last_seen。
- `zws_plugin` 保留为运行时实例（id、agent_id、plugin_type、endpoint、online、capabilities），由 Agent 上报。

#### 部署形态

- Linux：systemd 服务 `zero-agent.service`，开机自启、异常重启；配置文件与插件目录固定路径。
- Windows：Windows Service，开机自启；日志/数据目录可配置。
- Agent 每台接入主机必装一套；一台 Agent 可托管多个插件、多个设备。
- 插件升级：新版本目录 + 清单校验 → 平台确认 → Agent 停旧起新 → 失败自动回滚。
- 离线安装：随包提供插件目录与离线依赖，运行时不依赖公网。

#### 安全

- Agent 使用独立 token；推荐 mTLS。
- 插件进程使用独立系统账号/低权限，限制文件与网络。
- Plugin manifest 校验 checksum/签名；插件目录只允许 Agent 写入。
- 平台下发的凭据（设备密码、MQTT、MinIO）只在插件内存使用，不落盘、不写日志。
- 反代只暴露白名单 RPC 路径，禁止任意路径透传。

#### 关键边界

- Agent **只做控制面与 RPC 反代**，不转发视频流、不做媒体转码；视频仍由插件/媒体节点直出。
- Agent 不保存业务数据；设备、通道、能力、事件以平台为准。
- 插件禁用后，其设备标记离线但数据保留；重新启用后走一次同步对账。
- Agent 离线时，平台拒绝新的插件操作并给出明确错误；已有录像/事件按本地策略继续或停止，由插件决定并上报。

### 3.3 接入与模型边界

- GB28181/ONVIF 适配器只负责协议交互与协议快照，统一模型由应用层投影。
- 插件只负责设备交互与流源，不维护业务数据；设备/通道/能力/事件归属平台。
- 所有接入最终产出统一的 `PhysicalDevice / AccessBinding / Resource / StreamProfile / Capability / Event`。

新增协议时的判定：

| 情况 | 做法 |
| --- | --- |
| 国标/行业标准，平台需深度参与信令/媒体 | 新增标准适配器 + 端口实现 |
| 厂商私有 SDK/企标，进程隔离可解决 | 新增插件 + manifest；平台业务代码不改 |
| 仅数据上报/第三方系统 | 走 MQTT/HTTP 事件源，不造设备模型 |

禁止事项：

- 禁止在 application/domain 里按 `protocol` 写业务分支。
- 禁止插件直接连平台数据库或写业务表。
- 禁止把协议 token/自增 ID 作为跨模块主键。

## 4. 标识与编码约定

### 4.1 四种码

| 字段 | 含义 | 稳定性 | 示例 |
| --- | --- | --- | --- |
| `deviceInternalCode` | 平台设备内码 | 永不变，全局唯一 | `Dbbe16078a0bc386b` |
| `deviceExternalCode` | 协议侧设备标识 | 协议可变更 | GB 20 位 DeviceID / ONVIF endpoint / 插件 UID |
| `channelInternalCode` | 平台通道/资源内码 | 永不变，全局唯一 | `C7f3a...` |
| `channelExternalCode` | 协议侧通道标识 | 协议可变更 | GB 通道号 / ONVIF VideoSource / 插件 channelId |

规则：

- 内码格式沿用 `pkg/idcode`：设备 `D` + 16 hex，通道/资源 `C` + 16 hex；创建时生成，永不变。
- 所有设备、通道/资源**必须有内码**；导入/回填时缺失则生成。
- 外码只用于协议交互和排障，不进入播放流名、事件关联、前端主键。
- 外码变化（GB 重新注册、ONVIF token 变化、插件 channelId 变化）只更新绑定/资源外码，内码不变；按外码对账，禁止重建内码。
- 内码不复用、不回收；删除用 tombstone/状态，不做物理重用。
- 统一资产 UUID 只做数据库主键/内部关联，不作为 API 标识；`LegacyRef` 负责旧表↔统一 ID 映射。
- 旧 `InternalCode` 继续沿用，不重新生成；旧数字主键通过 `LegacyRef` 映射。

### 4.2 播放路径

统一逻辑路径：

```text
播放目标 = { deviceInternalCode, channelInternalCode, profileRole }
媒体流 key = {deviceInternalCode}_{channelInternalCode}
```

- `profileRole = main | sub | third | custom`。
- 主码流保持 `dev_ch`；非主码流建议 `dev_ch_sub` / `dev_ch_third`，避免同一通道多码流并发冲突。
- 对外 API 不暴露 GB 设备号/ONVIF token/插件实例 ID 作为播放主键。
- 旧流名（GB `DeviceID_GBDeviceID`、ONVIF `deviceID_profile`、插件 `instance_UID_channel`）通过 `LegacyRef` 映射；迁移期允许双读，停止/清理需按会话记录回原流名。

**现状**：

- 实时流已按 `live/<deviceInternalCode>_<channelInternalCode>` 实现；ONVIF/插件在内码缺失时仍会回退协议标识流名，须在 Phase A 消除。
- 回放/下载仍是 `DeviceID_GBDeviceID_playback|_download`，是本次统一的重点技术债；Stop 缺省路径也必须一起改，否则会停错流。
- 主/子码流目前不改变流 key：GB 只改 SDP `a=` 属性，ONVIF 改 ProfileToken 的 StreamURI，插件无主/子概念；因此 `dev_ch_sub` 后缀是新增约定，需确认是否本期落地。

### 4.3 API 标识

- 新 API 只返回 `deviceInternalCode`、`channelInternalCode`、`resourceId`、`profileId`。
- `rawId/dbId/gbId/platformId` 只保留在兼容 API，标注 deprecated。
- 前端路由/URL 使用内码，不用数据库自增 ID。

## 5. 资产与通道模型

### 5.1 关系

```text
PhysicalDevice 1 ── 1 AccessBinding
PhysicalDevice 1 ── N Resource
Resource(video_sensor) 1 ── N StreamProfile
Resource / Device / Profile 1 ── N Capability
PhysicalDevice 1 ── N EventSource
```

- 一个物理设备可以没有视频通道（纯报警/音频设备），也可以有多个视频通道（双光谱、双镜头 NVR 子设备）。
- NVR 本身是物理设备，其下通道作为 Resource；不因每个 GB 通道自动创建物理摄像机。
- `Resource.kind`：`video_sensor`、`audio_input`、`audio_output`、`alarm_input`、`relay_output`、`metadata_source`。
- 只有 `video_sensor` 才有 `StreamProfile` 和播放入口。

### 5.2 通道字段（统一）

- `channelInternalCode` 就是 `Resource.InternalCode`；视频/音频/告警/继电器/元数据资源都有通道内码，`resourceId`（UUID）只在内部使用。
- 资源类型与页面通道类型映射：`video_sensor`=视频通道、`audio_input/audio_output`=音频通道、`alarm_input`=报警输入、`relay_output`=继电器、`metadata_source`=数据通道。
- 内码/外码、名称、kind、状态、厂商/型号、经纬度、地图层级、PTZ 类型、音频、标签、启用状态。
- 视频通道额外：默认 profile、可选 profiles、编码/分辨率。
- 事件源：设备级/通道级，kind、external_key、enabled、config。

### 5.3 设备/通道目录树

树节点示例：

```text
[设备] 1号门岗摄像机  D...  在线 / GB28181
├─ [视频通道] 通道1  C...  枪机 / 在线 / 主码流
│  ├─ [码流] 主码流 1080P
│  └─ [码流] 子码流 720P
├─ [报警输入] 报警输入1  C...
└─ [继电器] 道闸  C...
```

- 树节点字段：`nodeType(device|resource|profile)`、`internalCode`、`kind`、`name`、`online`、`protocol`、`capabilities[]`、`children`。
- 设备节点可展开资源；资源节点按类型分组；视频资源可展开码流。
- 通道列表演示全量资源，支持按设备、类型、在线状态过滤。
- 设备可编辑：名称、厂商、型号、序列号、标签。
- 通道可编辑：名称、类型、经纬度、地图层级、PTZ 类型、音频、默认码流、标签。
- 页面不再是“标准接入/插件接入”两套设备列表；插件页面只做 **Agent/插件生命周期管理**。

### 5.4 事实源与写回

目标事实源分层：

| 数据 | 目标事实源 | 说明 |
| --- | --- | --- |
| 设备/通道身份、名称、经纬度、类型、标签、启用 | 统一资产表 | 设备管理页直接编辑 |
| 接入协议、外码、endpoint、插件实例、协议配置 | `AccessBinding` + legacy 协议表 | 协议适配器维护 |
| 在线/运行态 | Redis + 统一设备状态 | 适配器/Agent 实时更新 |
| 能力 | 统一 `Capability` | 协议/插件声明 + 平台计算 |
| 事件/报警 | 统一事件/报警表 | 平台唯一写入 |

迁移期策略：

- 统一资产表逐步成为业务事实源；legacy 六表降级为“协议配置 + 兼容投影”。
- 设备/通道编辑写统一表；为兼容旧页面，可同步回写 legacy 名称字段，但不反向覆盖统一表的经纬度/标签。
- 协议同步只更新外码、协议配置、在线状态和资源存在性，不覆盖用户编辑字段。
- 删除设备：统一表 tombstone + legacy 停用；不做物理级联删除。
- 事件/报警只写统一表，legacy 报警表在迁移期继续可读。

### 5.5 页面字段归并

| 统一字段 | GB28181 | ONVIF | 插件 |
| --- | --- | --- | --- |
| 设备名称 | `zws_device.name` | `zws_onvif_device.name` | `zws_plugin_device.name` |
| 设备内码 | `internal_code` | `internal_code` | `internal_code` |
| 设备外码 | `device_id` | `ip:port` | `device_uid` |
| 接入方式 | `passive/gb28181` | `active/onvif` | `plugin` |
| 接入地址 | SIP 域/ID | IP:port | Agent/插件实例 |
| 通道名称 | `gb_name` | channel name | channel name |
| 通道内码 | `internal_code` | `internal_code` | `internal_code` |
| 通道外码 | `gb_device_id` | `video_source/profile_token` | `channel_id` |
| 经纬度 | channel 经纬度 | channel 经纬度 | schema 字段 |
| 能力 | 适配器声明 | 适配器声明 | manifest/Hello 声明 |

页面只读统一字段；协议差异收进“接入绑定/原始配置”折叠区，不再作为列表主列。

## 6. 能力模型

- `scopeType = device | resource | profile`
- `operation = play | playback | download | ptz | preset | snapshot | arm | disarm | config | audio | talk | record | event`
- 状态：`supported`（协议/插件声明）、`configured`（平台配置）、`available`（在线/权限/运行态）。
- 有效能力 = supported ∧ configured ∧ available。
- 前端按钮由能力驱动，不再按 `protocol` 分支；缺能力时 disabled 并展示原因。
- 插件上报能力只是输入，平台须与端口实现取交集。

**归一规则**：

- 标准协议：由适配器声明 `supported`；配置/在线状态决定 `configured/available`。
- 插件：`plugin.yaml` 与 Hello 的能力合并为 `supported`；布防/对象存储/MQTT 等前置条件由平台计算 `configured`。
- 统一表 `zws_capability` 是唯一对外能力源；`deviceaccess.DeviceView.Capabilities []string` 与插件裸 `[]string` 只作为输入，不直接给前端。
- `profile` 级能力仅用于码流差异（如子码流不支持抓图）；没有差异时回落到 resource 级。
- 能力重建必须幂等，不得因为一次 backfill 把运行时 `available` 覆盖成 false；`available` 由在线/会话状态实时计算。

## 7. 事件与报警

### 7.1 分类

| 类别 | 定义 | 示例 |
| --- | --- | --- |
| `alarm` | 需要人工关注/处置 | 入侵、越界、破坏、视频丢失 |
| `status` | 设备/通道状态变化 | 上下线、录像异常 |
| `operation` | 人为操作 | 布防、撤防、PTZ、配置修改 |
| `ai` | AI 识别结果 | 人脸、车牌、徘徊、人群聚集 |
| `metadata` | 附属数据 | 目标框、轨迹、结构化数据 |

**事件是全集；报警是事件的子集**。报警必须关联 `eventId`，事件不必投影为报警。

### 7.2 事件类型注册表

新增 `zws_event_type`（DB 种子 + 代码常量兜底）：

```text
code, name, category, severity_default, alarm_eligible,
requires_video, bbox_capable, vendor_mappings_json, enabled
```

内置规范码（首版）：

| code | 名称 | category | 报警 | bbox | 说明 |
| --- | --- | --- | --- | --- | --- |
| `motion` | 移动侦测 | alarm | 是 | 可 | 通用移动事件 |
| `line_crossing` | 越界侦测 | alarm | 是 | 是 | 带目标框 |
| `intrusion` | 区域入侵 | alarm | 是 | 是 | 带目标框 |
| `loitering` | 徘徊 | alarm | 是 | 是 | 带目标框 |
| `video_loss` | 视频丢失 | alarm | 是 | 否 | 通道级 |
| `tamper` | 遮挡/破坏 | alarm | 是 | 可 | 通道级 |
| `audio_abnormal` | 音频异常 | alarm | 是 | 否 | 音频通道或视频通道 |
| `scene_change` | 场景变更 | alarm | 是 | 否 | 通道级 |
| `face` | 人脸 | ai | 可 | 是 | 结构化 |
| `plate` | 车牌 | ai | 可 | 是 | 结构化 |
| `alarm_input` | 报警输入 | alarm | 是 | 否 | 开关量 |
| `device_online` / `device_offline` | 设备上下线 | status | 否 | 否 | 设备级 |
| `record_abnormal` | 录像异常 | status | 是 | 否 | 通道级 |
| `ptz` / `config_change` | 云台/配置操作 | operation | 否 | 否 | 审计 |
| `metadata` | 附属数据 | metadata | 否 | 可 | 轨迹/结构化 |
| `other` | 其他 | alarm | 可 | 可 | 未登记类型兜底 |

- 厂商/协议映射：
  - GB 报警类型码 → 规范码（兼容现有 `1/2/7/10/11/15/22/25/26/99` 映射）；
  - ONVIF topic → 规范码；
  - 海康 ISAPI eventType → 规范码；
  - 插件事件类型 → 规范码（未登记则 `other` + `originalType`）。
- `alarmCode` 使用规范码字符串，不再用数字当平台主码；数字只作为 `vendor_mappings` 里的协议码。
- 新增事件类型只加注册表数据，不改业务分支；前端筛选/标签从注册表读取。

### 7.3 事件字段约定

统一事件载荷（MQTT `zws.event.v1`）：

```json
{
  "schema": "zws.event.v1",
  "eventId": "hik-01-1720000000-0001",
  "type": "line_crossing",
  "category": "alarm",
  "severity": "major",
  "occurredAt": 1720000000000,
  "deviceInternalCode": "Dxxxxxxxxxxxxxxxx",
  "channelInternalCode": "Cxxxxxxxxxxxxxxxx",
  "channelExternalCode": "1",
  "image": { "key": "events/...jpg", "url": null, "expiresAt": 0 },
  "bbox": { "x": 0.12, "y": 0.20, "w": 0.15, "h": 0.30, "coord": "normalized", "frameTs": 1720000000000, "confidence": 0.93, "trackId": "t-1" },
  "originalType": "VMD",
  "raw": {}
}
```

字段规则：

- `eventId` 全局幂等键；重复直接丢弃。
- `type` 必须是注册表规范码；未登记写 `other`，原始值放 `originalType`。
- `deviceInternalCode` 必填；`channelInternalCode` 在有通道时必填；外码只用于排障。
- 事件与通道关联即代表与设备关联：只给通道内码时，平台反查设备并补全；查不到则标记 `unmapped`，不得阻塞入库。
- `occurredAt` 毫秒时间戳；`receivedAt` 由平台补。
- `image`、`bbox` 可空；bbox 坐标默认 0~1 归一化，必须带 `coord`。
- `raw` 原样保存，不参与业务判断。
- `alarmId` 由平台投影后回写，插件不填。

### 7.4 报警字段约定

- `alarmId`、`eventId`（唯一）、`alarmCode`、`severity`、`status`（unhandled/handled/ignored）。
- `deviceInternalCode`、`channelInternalCode`、`channelLegacyId`（回放兼容）。
- `imageKey`、`bbox`、`occurredAt`。
- `handledBy/handledAt/remark`。
- 报警详情必须可一键回放：`playbackContext = {device, channel, profile, start=occurredAt-10s, end=+10s}`。
- **所有报警必须由事件产生**：GB 直报告警先构造 synthetic event（`type` 由 GB 报警码映射，`eventId` 用协议流水/SR 生成），再投影报警；不再允许报警表独立写入。

### 7.5 AI 与视频联动

- AI 事件必须关联视频资源和 profile；bbox 坐标必须注明坐标系与帧时间。
- 事件详情：原图 + bbox 叠加 + “查看回放”按钮。
- 回放能力通过 `playback` 能力判断；不支持时显示不可用原因。
- 联动上下文由事件推导，不要求插件提供：

```json
{
  "deviceInternalCode": "D...",
  "channelInternalCode": "C...",
  "profileRole": "main",
  "startAt": 1720000000000,
  "endAt": 1720000020000,
  "preferred": "playback",
  "fallback": "snapshot"
}
```

- 默认窗口 `occurredAt ± 10s`；前端可调。
- 回放不可用时：优先显示事件原图；若 `download` 可用则提供下载；否则明确提示“该通道不支持回放”。
- bbox 只存原始坐标系与数值；前端按播放器画面尺寸换算，不在后端做像素换算。

## 8. 基础设施

固定为：

- MySQL：业务事实源。
- Redis：设备在线/心跳/过期/会话缓存。
- MQTT：事件总线（插件 → 平台），共享订阅 + 死信。
- MinIO/S3：事件图片/录像对象存储。

MQTT 主题约定：

```text
事件主题      {prefix}/events
共享订阅      $share/zws/{prefix}/#
死信          {prefix}-dlq        # 必须在共享订阅范围之外
事件 schema   zws.event.v1
```

- 插件只发布事件，不发布控制消息；布防/撤防走 RPC。
- 平台按 `eventId` 幂等消费；失败重试后进死信，死信不自动回灌。
- 图片由插件先传 MinIO，再发事件；事件只带 key/URL。

配置统一在「存储管理 → 基础设施」：MySQL/Redis 改 YAML 需重启；MQTT/对象存储热重载。

## 9. API 约定

目标资源：

```text
/api/v1/devices                    物理设备（含绑定/资源摘要）
/api/v1/devices/{code}             设备详情/编辑
/api/v1/devices/{code}/channels    通道列表
/api/v1/assets/tree                设备→通道/资源树
/api/v1/channels/{code}            通道详情/编辑
/api/v1/channels/{code}/live       实时播放
/api/v1/channels/{code}/playback   回放
/api/v1/channels/{code}/capabilities
/api/v1/events                     事件查询
/api/v1/events/types               事件类型注册表
/api/v1/alarms                     报警查询/处置
/api/v1/agents                     Agent/插件管理
```

- 新 API 统一返回 `deviceInternalCode` / `channelInternalCode`，不使用协议字段做主键。
- 编辑接口只接受统一字段（名称/经纬度/类型/标签/启用），协议字段通过接入绑定单独维护。
- 旧接口保留并标注 deprecated，前端迁移完成后下线。
- 分页统一 `{list,total,pageNum,pageSize}`；错误统一 `AppError`。

## 10. 前端统一

| 页面 | 目标 |
| --- | --- |
| 设备资产 | 左树（设备→通道/资源）+ 右详情/表格；标准/插件设备同一入口 |
| 通道列表 | 全量资源表，类型/在线/设备过滤，可编辑 |
| 插件接入 | 改为 Agent/插件管理：Agent 状态、插件清单、启停、日志、版本 |
| 事件中心 | 事件全集：类型/设备/通道/时间过滤、图片/bbox/回放 |
| 报警管理 | 重点报警：处置状态、确认/忽略、回放联动 |
| 设备详情 | 概览/绑定/通道/能力/事件/原始配置 |

复用组件：`DeviceTree`、`RegionTree`、`jessibuca`、`playerTabs`、`ptzControls`，但树节点增加 `resourceKind / internalCode / capabilities`。

**现状（前端）**：

- 标准接入：`views/devices/list.vue` 固定 GB/ONVIF 表单与操作列；通道通过 `views/devices/index.vue` 的 `channelCtx` 页内切换，不是树。
- 插件接入：`views/plugins/index.vue` 同时是“插件实例 + 插件设备”列表，动态 JSON Schema 表单、`ipc/*` API、布尔布防。
- 设备资产：`views/assets/index.vue` 是分页表格 + 详情抽屉，已能展示 resources/profiles/capabilities 并播放，但没有设备树，也没有通道编辑。
- 通道列表：`views/channel/index.vue` 是 GB 全局通道表；`views/channel/unified-panel.vue` 已跨协议，但只嵌在插件页。
- 现有 `DeviceTree/RegionTree/GroupTree` 是行政区划/业务分组组织树，不是设备拓扑；`queryDeviceTree/queryChannelTree` 后端接口已存在但前端未接线。
- 事件页有图片 + bbox JSON，无回放；报警页有回放，无 bbox；两页未打通。

## 11. 迁移路线

### Phase A：标识与流名统一

- 固化 `Resource.InternalCode`（已存在，来自 legacy）：校验唯一、非空；禁止投影时因映射丢失重生 UUID。
- 统一外码字段：设备/通道 external code 落到 binding/resource；新 API 只暴露内码。
- 实时流消除 ONVIF/插件协议标识兜底；回放/下载改为 `dev_ch_playback|download`，Stop 路径同步改。
- 播放会话记录 `protocol + legacy identifiers + stream key`，保证清理按原流名执行。
- 验收：新旧流名都能播；停止/清理不串流；回放/下载与实时一致。

### Phase B：zero-agent 控制面

- Agent 注册/心跳/清单上报/desired state/启停命令。
- 插件清单规范；mock/海康插件改为 Agent 托管，删除插件直连注册路径。
- 平台 Agent/插件管理页；插件 enable/disable。
- 验收：平台启停插件进程；插件重启自动恢复；Agent 断线设备离线。

### Phase C：统一设备页面

- 设备资产树 + 通道全量表 + 编辑 + 能力按钮。
- 标准/插件设备合并；插件页面只保留 Agent/插件运维。
- 验收：同一设备不同接入方式页面字段一致；通道经纬度/类型可编辑。

### Phase D：事件/报警定型

- 事件类型注册表、报警投影、bbox、回放联动。
- 事件中心/报警中心统一筛选与详情。
- 验收：GB/ONVIF/插件事件都能归一到同一类型码并关联通道。

### Phase E：旧接口/旧表收敛

- 旧 API 标记 deprecated；旧表只保留协议配置/兼容读。
- 流名/事件/报警全部走内码。
- 验收：前端无旧 ID 依赖；回归测试通过。

## 12. 现状盘点明细（实现依据）

### 12.1 标识

- `pkg/idcode`：`Device()` = `D`+16 hex，`Channel()` = `C`+16 hex；创建时生成，重同步按协议键保留。
- 统一资产主键是 UUID 字符串；`zws_unified_ref(legacy_table, legacy_id, entity_type, unified_id)` 负责旧表↔统一 ID 映射；映射丢失时 `uuid.NewString()` 会重生 UUID（需在 Phase A 禁用/修复）。
- 设备 DTO 同时暴露 `ID`（`gb:<DeviceID>` / `onvif:<id>`）、`RawID`、`DBID`、`InternalCode` 四套标识。
- `dto.ChannelView` 的 `DeviceID` 实际是国标通道编码，`ParentDeviceID` 是国标设备编码，语义与命名相反。
- 播放/回放路由中的 `deviceId/channelId` 是国标编码；ONVIF 是 legacy 自增 ID；插件设备是 legacy 自增 ID，通道是外码字符串。
- `position_handler` 同一 `channelId` 参数在不同分支分别当 legacy ID 和国标通道编码。

### 12.2 流名

- 实时：`mediaurl.LiveStream(deviceInternalCode, channelInternalCode)`，GB/ONVIF/插件已接入。
- 回放：`fmt.Sprintf("%s_%s_playback", device.DeviceID, channel.GBDeviceID)`；下载同规则 `_download`；Stop 缺省同。
- ONVIF/插件在内码为空时回退 `deviceID_profileToken` / `pluginID_deviceUID_channelID`。
- 码流选择：GB 只影响 SDP，ONVIF 只影响 ProfileToken/StreamURI，插件无主/子；当前不改变流 key。

### 12.3 能力

- 统一 `Capability` 仅由 backfill 生成：`source=backfill`，`available=online`；`profile` 级没有数据。
- 设备级操作：GB `play,playback,ptz,snapshot,alarm,subscribe,guard,transport`；ONVIF `play,ptz,snapshot,sync`；插件 `play,ptz,sync`。
- 资源级：`play,snapshot` + PTZ 类型>0 时 `ptz` + 有音频时 `audio`。
- 另有 `deviceaccess.DeviceView.Capabilities []string` 和插件 Hello `Plugin.Capabilities []string`，三套并存。

### 12.4 资产与事实源

- legacy 六表是事实源：`zws_device`、`zws_device_channel`、`zws_onvif_device`、`zws_onvif_channel`、`zws_plugin_device`、`zws_plugin_channel`。
- 统一表只做单向投影，不写回 legacy；设备/资源改名在统一侧保留。
- `EventSource` 当前只有 GB 设备一条 `kind=alarm`；ONVIF/插件未生成。
- 在线状态是 backfill 快照，两次投影之间可能滞后；legacy 实时状态才是准的。

### 12.5 事件/报警

- 事件字段：`EventID` 去重、`DeviceInternalCode` 关联设备、`ChannelInternalCode/ChannelExternalKey` 关联资源、`BBoxJSON`、`ImageKey/URL`、`AlarmID`。
- 设备关联：`deviceInternalCode → ListResources`；先精确匹配 `ChannelInternalCode`，再按外码匹配，并兼容 `1`/`101` 通道号写法。
- 事件→报警：仅当匹配到 `LegacyChannelID>0` 才投影；`AlarmType` 由硬编码映射（motion=7、line_crossing=10、intrusion=11、video_loss=1…默认 99），并回写 `Event.AlarmID`。
- GB 直接报警上报：`gbsip.AlarmNotify` 直接建 Alarm，不产生 Event，`AlarmType` 是设备侧原始码。
- 报警字段仍是 legacy 形状：`ChannelID`（自增 ID）、`AlarmType int`、`SnapPath`、`AlarmTime int64`；没有 `eventId/bbox/resource/profile`。
- 事件无回放关联字段；报警有 `channelId + alarmTime` 可回放，但无 bbox。

### 12.6 页面

- 标准接入/插件接入/设备资产/通道列表是四套并列表格；组件、API、字段均不同。
- 树组件用于行政区划/业务分组，不支持设备→通道拓扑。
- 事件页有 bbox 无回放；报警页有回放无 bbox。

## 13. 已定决策

1. zero-agent 只做 **托管 + 反代**，不提供直连模式。
2. 插件由 Agent 代注册，插件不自注册；插件只通过 Agent RPC 与平台交互。
3. 非主码流 stream key 采用 `_sub/_third` 后缀，主码流保持 `dev_ch`。
4. 事件类型注册表采用 DB 种子 + 代码常量兜底。
5. `PhysicalDevice/AccessBinding/Resource` 作为业务事实源，旧表降级为协议配置/兼容投影。
