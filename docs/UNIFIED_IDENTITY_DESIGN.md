# 统一接入归一化设计（设备内码 / 通道内码）

> 状态：**已实施（v1）**。落地记录见 §9。关联：[ARCHITECTURE.md](ARCHITECTURE.md)、[EVENT_PIPELINE_DESIGN.md](EVENT_PIPELINE_DESIGN.md)、[IPC_PLUGIN_DESIGN.md](IPC_PLUGIN_DESIGN.md)。

## 1. 设计目标（来自需求）

1. 每个设备生成**设备内码**；每个设备下生成一个或多个**通道内码**（视频/音频/未来其他类型可扩展），通道内码归属设备内码。
2. ONVIF、插件、GB28181 三种接入在平台内部**统一用设备内码 + 通道内码**表示。
3. GB28181 接入额外持有**设备国标码 / 通道国标码**；ONVIF、插件接入的国标码为空。
4. 推流/播放地址统一：`live/<设备内码>/<通道内码>`。
5. 前端通道展示统一：插件接入与标准接入一样进入**通道列表页**，而不是弹窗。
6. 设备内码、通道内码、事件码三者关联，同时表达「事件 ↔ 音视频 ↔ 设备从属关系」。

## 2. 现状盘点（代码事实）

| 层 | 设备内码 | 通道内码 | 国标码 |
| --- | --- | --- | --- |
| GB（`zws_device` / `zws_device_channel`） | ✅ `internal_code` | ✅ `internal_code` | ✅ `device_id` / 通道 `gb_device_id` |
| ONVIF（`zws_onvif_device` / `zws_onvif_channel`） | ✅ | ✅ | 通道 `gb_code` 可选 |
| 插件（`zws_plugin_device` / `zws_plugin_channel`） | ✅ | ✅ | 无 |
| **统一资产 `zws_physical_device`** | ✅ `internal_code` | — | 无 |
| **统一资产 `zws_resource`（通道）** | — | **❌ 没有** | 无 |

问题点：

1. **通道内码没有进入统一资产**。`zws_resource` 只有 uuid 主键 + `external_key`（协议侧通道号/GB 编/ProfileToken），业务层拿不到稳定的通道内码。
2. **流命名三套并存**：
   - 插件：`app=ipc`，`stream=<pluginID>_<deviceUID>_<channelID>`（含 IP 与端口，改名/换址即断）
   - GB28181：`app=live`，`stream=<设备国标码>_<通道国标码>`
   - ONVIF：基于 `profileToken`
3. **事件只带外部键**：`zws_event` 有 `device_internal_code` + `channel_external_key`，没有通道内码；资源映射靠 `external_key` 猜（`1` ↔ `101` 的兼容 hack 就在 `findResource`）。
4. **前端不一致**：插件接入的「通道」是弹窗；标准接入有独立通道页。
5. **插件设备/通道没有进入统一资产**（本次已修，见 §7 已完成项），否则设备资产页没有资源、事件全部 `unmapped`。

## 3. 目标领域模型

```text
PhysicalDevice（统一设备）
├─ id            内部 uuid（对外不暴露）
├─ internal_code 设备内码 D+16hex（唯一，对外主键）
├─ gb_code       设备国标码（GB 有，ONVIF/插件为空）
└─ channels: Resource[]（1..N，可扩展类型）

Resource（统一通道）
├─ id            内部 uuid
├─ internal_code 通道内码 C+16hex（唯一，对外主键）
├─ device_id     → PhysicalDevice.id（从属关系）
├─ kind          video_sensor | audio | io | ...（可扩展）
├─ external_key  协议侧标识（插件通道号 / GB 通道编码 / ONVIF ProfileToken）
├─ gb_code       通道国标码（GB 有，其余为空）
└─ profiles: StreamProfile[]

AccessBinding（接入绑定，与设备一对一）
├─ protocol      gb28181 | onvif | plugin
├─ external_id   协议侧设备标识（GB 设备编码 / ONVIF endpoint / 插件 device_uid）
└─ config_json   协议参数
```

约束：

- **内码即对外身份**：API、流地址、事件、前端路由都用内码；uuid 仅内部外键。
- **external_key 只用于回连协议侧**，不参与对外表示；`findResource` 的兼容 hack 可以被通道内码取代。
- **通道类型可扩展**：`kind` 字段 + 能力（`zws_capability`）表达，未来加音频/报警输入通道不需要改主表。
- GB 只是多两个字段（设备国标码、通道国标码），接入路径没有特殊分支。

## 4. 流地址统一

目标：

```text
RTSP   rtsp://<zms>:8554/live/<设备内码>_<通道内码>
FLV    http://<zms>:8080/live/<设备内码>_<通道内码>.flv
WS-FLV ws://<zms>:8080/live/<设备内码>_<通道内码>.flv
HLS    http://<zms>:8080/live/<设备内码>_<通道内码>/hls.m3u8
```

> ZMS 地址是 `/{app}/{stream}` 两段，因此落地为 **app=live，stream=`<设备内码>_<通道内码>`**（决策见 §8）。逻辑上仍是「设备/通道」两段身份，只是用 `_` 合并到一个 stream 段，避免 ZMS 侧出现大量 app 命名空间。

实现要点：

- 新增 `pkg/streamname`（或 `internal/mediaurl` 内）统一生成，三种协议共用，不再各拼各的。
- GB28181 的 INVITE/SIP 流标识、录像计划、云录制、推流鉴权都读同一函数，避免路径各写一套。
- 兼容期：`StopLive` 同时尝试新旧两套 stream 名，ZMS 侧旧 proxy 按 `close_streams` 清理。
- 推流（`zws_stream_push`）与播放鉴权（`publishauth`）的 app 也需要同步，否则会漏鉴权。

## 5. 事件归一化

事件载荷（插件 → MQTT → 平台）：

```json
{
  "schema": "zws.event.v1",
  "eventId": "<设备内码>-<ts>-<rand>",       // 事件码：全局唯一
  "deviceInternalCode": "D...",
  "channelInternalCode": "C...",              // 新增：事件归属通道
  "eventType": "intrusion",
  "occurredAt": 1795012000000,
  "raw": { "vendor": "hikvision", "alertXml": "..." }  // 原始报文，保留协议细节
}
```

平台侧：

- `zws_event` 增加 `channel_internal_code`；映射优先用内码，external_key 仅兜底。
- 事件 → 告警投影用的 `legacy_channel_id` 改为内码直连资源，去掉 `findResource` 猜测逻辑。
- 插件侧需要知道通道内码：布防（arm）请求里下发 `channels: [{channelId, internalCode}]`，插件在事件里回填。

## 6. 前端统一

- 「插件接入」的设备行 `通道` 按钮从弹窗改为跳转**通道列表页**（`/channel?deviceInternalCode=...`），与管理台现有通道页复用同一套组件，按协议只读/可操作能力渲染。
- 设备资产、事件中心、地图等页面统一以设备内码/通道内码定位，不再混用协议标识。
- 播放统一走 `/resources/<通道内码>/live/start` 一类接口（见 §8 决策）。

## 7. 已完成（本次联调修复，先落地、不可回退的前提）

1. 插件设备/通道在新增、同步、插件重启补布防后**投影到统一资产**（`PluginHandler` → `asset.Service.Backfill()`）；否则设备资产无资源、播放 404、事件全部 `unmapped`。
2. 插件 Hello 后补布防增加重试（插件监听端口就绪时序），避免一次失败后长期不布防。
3. 插件事件保留完整原始 XML（`raw.alertXml`），区域入侵等 AI 事件的规则/区域/目标字段不再丢失；事件类型过滤对配置项做归一化。
4. 海康插件 Windows 构建/运行验证通过；播放链路（资产 → 插件 RTSP → ZMS 拉流 → FLV）已验证。

## 8. 已定决策

1. **流命名**：`app=live`，`stream=<设备内码>_<通道内码>`（`internal/mediaurl.LiveStream`）。
2. **通道内码存放**：`zws_resource.internal_code` 为统一资产字段；**legacy 通道表（GB/ONVIF/插件）的内码是来源**，投影时复制并保持一致（legacy 内码按协议侧通道稳定保留）。
3. **迁移范围**：GB28181、ONVIF、插件三协议一次到位；`StopPlay` 兼容期同时清理旧命名流。

## 9. 实施记录（v1）

| 阶段 | 内容 | 状态 |
| --- | --- | --- |
| P1 | `zws_resource.internal_code`（AutoMigrate/init SQL + 启动回填，优先继承 legacy 内码）；投影 `upsertResource` 携带内码；`zws_event.channel_internal_code`，事件按通道内码映射（external key 兜底） | ✅ |
| P2 | `internal/mediaurl.LiveStream` 统一流名；插件 `StartLive/StopLive` 切内码（旧名清理） | ✅ |
| P3 | GB（`play`/`recordplan`/`cascade`）、ONVIF、插件全部切 `app=live`；StopPlay 兼容清理旧命名；录像计划查询 SQL 适配 only_full_group_by | ✅ |
| P4 | 新增 `GET /api/v1/channels` 统一通道 API；通道列表抽成可复用组件 `channel/unified-panel.vue`（搜索/协议筛选/播放/跳设备），**插件接入的「通道」按钮在页内下钻**（与标准接入一致：页头「返回」+「通道列表」，不再跳到独立菜单页） | ✅ |
| 事件链 | 布防请求下发 `channels[{channelId, internalCode}]`；海康插件事件回填 `channelInternalCode`；事件保留原始 XML | ✅ |

验证记录（2026-11-19）：插件设备 1080P 流经 `live/<设备内码>_<通道内码>` 由 ZMS 拉流并 FLV 输出；GB28181 同命名播放通过；带 `channelInternalCode` 的事件落库后 `status=mapped` 且生成告警。

已知后续项：旧事件（切换前入库）`channel_internal_code` 为空，可按需一次性回填；ONVIF 未在当前环境做设备级回归。
