# 物联设备统一接入模型

本文是平台唯一的接入模型文档：GB/T 28181、ONVIF、私有 SDK（插件）三类接入在平台内部使用同一套资产模型、标识体系与能力声明；协议差异只允许存在于适配器内部。

## 1. 目标与边界

**目标**
- 一份设备/通道数据：接入协议不同，平台内的设备、通道、事件、级联只认一套模型。
- 一套身份：设备内码、通道内码对外统一，协议码只用于协议交互。
- 一套能力：能力显式声明 + 运行时熔断，页面与接口据此展示/拒绝。
- 一个出口：播放、回放、云台、级联推送都走统一入口，不在业务层按协议分支堆补丁。

**非目标（当前阶段）**
- 不做同一设备的多协议合并（一个物理设备只允许一条接入绑定）。
- 不实现 SOS/Web 服务端发现以外的第三方协议；新增协议通过适配器扩展。
- 语音广播/对讲、软件升级等能力暂不实现（见 GB 场景文档）。

## 2. 分层与依赖规则

```
interfaces（HTTP/Hook） → application → domain
                              ↓
                            port（媒体、对象存储、SIP 会话、设备缓存、平台客户端）
                              ↑
                     adapter / infrastructure（MySQL、Redis、SIP、ONVIF、插件 RPC）
```

- `internal/application/**` 只依赖 `domain`、`port`、`dto`、`gbsip` 与中立工具包（`config` 例外）。
- 基础设施通过 `port` 接口回调应用层（如 SIP 注册/目录回调 `port.DeviceLifecycle`），不得反向依赖具体服务类型。
- 媒体服务（ZMS）不内嵌，通过 HTTP API + Hook 对接。

## 3. 三类接入

| 接入 | 方式 | 设备标识 | 通道标识 | 国标码 |
|---|---|---|---|---|
| GB28181 | 被动：设备注册上来 | 20 位国标码 | 通道国标码 | 设备/通道自带 |
| ONVIF | 主动：平台发现/连接 | endpoint（ip:port） | VideoSource / ProfileToken | 无（挂国标目录时分配） |
| 插件（私有 SDK） | zero-agent 托管 sidecar | 插件 device_uid | 插件通道号 | 无（挂国标目录时分配） |

- 新增协议：实现设备适配器并在 `deviceaccess` 注册表登记，HTTP 层不改。
- 私有 SDK 不直连平台：zero-agent 扫描 `plugins/*/plugin.yaml`、启停进程、反向代理 RPC，平台管 Agent/插件。

## 4. 统一资产模型

```
PhysicalDevice（zws_physical_device）        物理设备：名称/厂商/型号/在线
  └─ AccessBinding（zws_access_binding）     唯一接入方式（1:1）
       protocol: gb28181 | onvif | plugin
       external_id / endpoint / plugin_instance_id / config_json
  └─ Resource（zws_resource）                子资源（即"通道"）
       kind: video_sensor | audio_input | audio_output
             | alarm_input | relay_output | metadata_source
       external_key: 协议侧标识（GB 通道码 / ONVIF token / 插件通道号）
       internal_code: 通道内码
       attrs_json: 经纬度、PTZ 类型、标签、启用状态等（用户编辑会被保留）
       └─ StreamProfile（zws_stream_profile）  role: main | sub | third | custom
              selector_json（协议侧码流标识）、codec、resolution、是否默认
Capability（zws_capability）                 scope= device|resource|profile 的能力事实
EventSource（zws_event_source）              事件/数据来源（GB 报警、AI 元数据等）
UnifiedRef（zws_unified_ref）                旧表主键 → 统一 UUID 的可重入映射
```

- 旧表（GB/ONVIF/插件）保留，统一资产是**增量投影**，启动时幂等回填。
- 录像计划/分组/级联当前仍引用旧通道数字 ID，通过 `UnifiedRef` 可追溯；后续阶段切换。

## 5. 标识与编码

| 标识 | 形式 | 用途 |
|---|---|---|
| 设备内码 | `D` + 16 hex | 对外设备身份、流名、事件关联 |
| 通道内码 | `C` + 16 hex | 对外通道身份、流名、事件关联 |
| 设备/通道国标码 | 20 位数字 | 仅协议交互（注册、目录、级联、点播） |
| 统一内部主键 | UUID | 仅数据库外键，不对外暴露 |

- 直播流名：`live/<设备内码>_<通道内码>`；回放 `_playback`、下载 `_download`。
- 页面不展示内码、UUID、数据库 ID；国标编号只在新增/编辑表单出现。
- 级联对外的编码由「级联编码层」负责（详见 GB 场景文档），与内部内码解耦。

## 6. 能力模型（Caps）

能力是**声明 + 运行时事实**，分三层作用域：设备、资源、码流。

- 常见操作：`play`（实时）、`playback`（回放）、`ptz`、`snapshot`、`audio`、`alarm`、`subscribe`、`guard`、`transport`、`sync`。
- `Supported` 表示协议/设备声明支持；`Configured` 表示平台已配置；`Available` 表示最近一次调用是否成功。
- **运行时熔断**：调用前若声明 `Supported=false` 直接拒绝；调用失败把 `Available` 置 false 并记录日志，恢复后置回 true。页面据此置灰按钮，避免反复调用烂固件。
- 当前已接入熔断的入口：资源实时播放（`resources/:id/live/start`）。回放、PTZ、抓拍的熔断按同一方式扩展。

## 7. 事件与报警

- 事件与报警分离：`EventSource` 描述来源，`zws_event` 存事件，`zws_alarm` 是事件的告警投影。
- 事件类型走注册表（`zws_event_type`），GB 数字告警码映射到规范事件码；AI 事件带 bbox 等扩展字段。
- 事件原始报文/扩展字段只在页面「高级信息（排障用）」折叠展示。

## 8. 基础设施

| 依赖 | 来源 | 生效 |
|---|---|---|
| MySQL / Redis | `configs/config.yaml` | 页面写回 YAML，重启生效 |
| MQTT | `zws_mqtt_config` | 保存后热重载消费者（共享订阅） |
| 对象存储 | `zws_object_store_config` | 保存后热重载（MinIO/S3） |
| 媒体节点 | ZMS 自注册（Hook） | 幂等 upsert，无需手工添加 |

## 9. API 与页面约定

- 统一前缀 `/api/v1`，响应 `{code,msg,data}`；错误用 `apperr` 业务码。
- 资产接口：`/assets`、`/assets/tree`、`/assets/:id`、`/assets/:id/sync`、`/resources/:id`、`/resources/:id/live/start|stop`、`/channels/:code`。
- 页面：设备资产（设备→资源树）、通道列表（跨协议）、接入管理（GB/插件/ONVIF）、事件中心、报警管理。
- 页面展示规则：只展示名称/IP/协议/状态等可读字段；原始 JSON、国标编号、内码不作为主展示。

## 10. 与"成熟接入框架"建议的对照（现状与差距）

| 建议 | 现状 | 结论 |
|---|---|---|
| 统一资产模型 | PhysicalDevice/AccessBinding/Resource/StreamProfile/Capability 已落地并回填 | ✅ 已具备 |
| 通道作为最小调度单元 | 通道 = Resource，播放/云台/事件均按资源定位 | ✅ 基本具备 |
| 能力声明 + 运行时降级 | 能力表已建；实时播放、回放/下载、云台控制已接入熔断 | ✅ 主要操作已具备，抓拍待接 |
| 统一驱动接口（ChannelDriver） | 已抽象：`asset/driver.go` 中 gb/onvif/plugin 三个驱动统一实现实时/停止/同步，业务层不再按协议分支 | ✅ 已具备 |
| 云台速度归一化 [-1,1] | 各协议各自量纲（GB 速度档位、ONVIF 浮点） | ⚠️ 待做（P2） |
| 流源统一抽象（StreamSource） | 统一返回 `dto.StreamContent`（含各协议播放地址） | ✅ 基本具备 |
| 超时与并发保护 | SIP 等待 5s、ONVIF 客户端 30s；无全局并发限制 | ⚠️ 部分 |
| 级联编码映射与反查 | 通道国标码全局唯一、落库；级联按通道国标码反查 | ✅ 已具备（见 GB 场景文档） |

**后续路线（按优先级）**
1. 能力熔断扩展到抓拍/音频。
2. 云台速度归一化 [-1,1]，在各驱动内换算。
3. 驱动接口扩展到回放/PTZ/抓拍（当前驱动只覆盖实时/停止/同步）。
4. 外部设备调用的统一超时/并发信号量。

## 11. 已定决策

- 一个物理设备一条接入绑定，不做多协议合并。
- 内码即对外身份；国标码只用于协议；UUID 不对外。
- 插件只经 zero-agent 接入，不允许插件直连注册。
- 业务目录（215/216）是对外级联的来源；业务分类不占国标目录。
