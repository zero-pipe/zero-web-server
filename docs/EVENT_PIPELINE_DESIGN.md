# 事件管道设计（布防 / 抓图 / MQTT / 事件入库）

> 状态：设计稿。目标是把「插件布防 → 抓图上传 → MQTT 事件 → 平台消费入库 → 资产关联」做成一条可扩展的标准链路。
> 关联：[ARCHITECTURE.md](ARCHITECTURE.md)、[IPC_PLUGIN_DESIGN.md](IPC_PLUGIN_DESIGN.md)。

## 1. 目标与原则

1. **布防即开通事件管道**：布防时平台把「MQTT 地址 + 图像存储地址 + 设备身份」一次性下发给插件。
2. **插件负责采集**：抓图、上传图片、组装事件、发布 MQTT；插件不做业务持久化。
3. **平台负责治理**：配置校验、布防状态、事件消费、持久化、资产关联、查询展示。
4. **设备内码是关联主线**：事件里必须带 `deviceInternalCode`，平台据此关联物理设备、资源、图片和告警。
5. **前置条件明确**：对象存储与 MQTT 未配置/不可用，布防按钮不可用，并给出明确提示。
6. **可重入、可重放**：MQTT QoS1 + `eventId` 唯一键，消费者幂等；插件断线重试，平台可补偿。

## 2. 总体链路

```text
Web 布防
  │ 1. 校验 object_store + mqtt 配置与连通性
  ▼
平台 ipc/arm ──2. 下发 MQTT + ObjectStore + deviceInternalCode──▶ 插件
                                                                  │
                                                                  │ 3. 打开报警通道/事件订阅
                                                                  │ 4. 事件触发 → 抓图 → 上传对象存储
                                                                  │ 5. 发布事件 JSON 到 MQTT
                                                                  ▼
                                                            MQTT Broker
                                                                  │ 6. 平台消费者订阅
                                                                  ▼
平台事件接入 ──7. 解析/去重/映射设备内码与资源──▶ zws_event
                                                   │
                                                   ├─ 可选：投影为 Alarm（报警管理）
                                                   └─ 事件中心/设备详情展示图片与 AI 框
```

## 3. 配置管理

### 3.0 统一基础设施配置（存储管理）

把 MySQL、Redis、MQTT、对象存储收敛到「存储管理 → 基础设施」一个页面，避免配置分散。

| 项 | 配置来源 | 在线状态 | 保存后 |
| --- | --- | --- | --- |
| MySQL | `configs/config.yaml`（启动引导） | `db.Ping()` | 写回 YAML + 备份，**需重启** |
| Redis | `configs/config.yaml`（启动引导） | `PING` | 写回 YAML + 备份，**需重启** |
| MQTT | `zws_mqtt_config`（库） | Broker CONNECT 状态 | 热重载消费者 |
| 对象存储 | `zws_object_store_config`（库） | `ObjectStore.Health()` | 热重载存储适配器 |

要点：

- MySQL/Redis 是**启动引导配置**，运行中改了只能写回 YAML，页面标记 `restartRequired=true`，提示手动重启；不在库里维护第二份，避免双源。
- 写 YAML 用 `yaml.Node` 保留注释，临时文件 + 原子 rename，保留 `.bak`，权限 `0600`；配置文件只读（容器）时返回明确错误。
- 密码字段读取时脱敏；保存时收到 `******` 表示“不修改原密码”。
- MySQL 挂掉时页面本身不可用（登录依赖库），此时直接改 YAML；这是可接受的边界。
- 页面统一展示四项状态；MySQL/Redis 使用当前运行中的连接探测，MQTT/对象存储使用各自健康检查。
- 旧「对象存储」入口保留兼容，内容并入「基础设施」页签。


### 3.1 复用对象存储配置

现有 `zws_object_store_config`（MinIO/S3）直接复用，字段：

```text
enabled, provider(minio|s3), endpoint, region, bucket,
accessKey, secretKey, useSSL, pathStyle, publicBase
```

布防时下发给插件的 `object_store` 与 `port.ObjectStoreConfig` 保持一致。

### 3.2 新增 MQTT 配置

新增单行配置表 `zws_mqtt_config`（与对象存储一致，全局一份）：

```sql
CREATE TABLE zws_mqtt_config (
  id INT PRIMARY KEY,
  enabled TINYINT NOT NULL DEFAULT 0,
  broker VARCHAR(255) NOT NULL,          -- tcp://host:1883 / ssl://host:8883
  client_id_prefix VARCHAR(64),          -- zws / zws-plugin
  username VARCHAR(128),
  password VARCHAR(256),
  topic_prefix VARCHAR(128) DEFAULT 'zws/events',
  qos TINYINT DEFAULT 1,
  keepalive_sec INT DEFAULT 30,
  tls_enabled TINYINT DEFAULT 0,
  ca_cert TEXT, client_cert TEXT, client_key TEXT,
  create_time VARCHAR(50), update_time VARCHAR(50)
);
```

平台 API（与对象存储风格一致）：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/mqtt_config` | 读取配置（密码脱敏） |
| POST | `/api/v1/mqtt_config/save` | 保存并热重载消费者 |
| GET | `/api/v1/mqtt_config/health` | 连接探测 |

前端：`存储管理 → 基础设施 → 消息队列`，包含 Broker、Topic 前缀、QoS、TLS、连通性测试。

### 3.3 事件管道就绪状态

新增只读接口：

```http
GET /api/v1/event_pipeline/status
```

返回：

```json
{
  "ready": true,
  "objectStore": { "enabled": true, "provider": "minio", "healthy": true, "error": "" },
  "mqtt": { "enabled": true, "broker": "tcp://...", "connected": true, "error": "" }
}
```

布防按钮据此启用/禁用；**平台在真正布防时仍会再校验一次**，不能只依赖前端。

## 4. 布防 / 撤防

### 4.1 平台 API

```http
POST /api/v1/ipc/devices/arm
{ "deviceId": 12, "eventTypes": ["motion", "line_crossing", "intrusion"] }

POST /api/v1/ipc/devices/disarm
{ "deviceId": 12 }
```

布防前置校验：

1. 插件在线、设备已登录；
2. `object_store_config.enabled = true` 且 `Health()` 通过；
3. `mqtt_config.enabled = true` 且平台消费者连接正常；
4. 插件声明支持 `alarm` / `arming` 能力。

任一不满足返回明确错误，例如：

```json
{ "code": 13001, "msg": "请先在「存储管理 → 对象存储」完成配置并测试连通性" }
{ "code": 13002, "msg": "请先在「存储管理 → 消息队列」完成配置并测试连接" }
```

### 4.2 RPC：Arm / Disarm

```json
// Arm
{
  "device_uid": "192.168.1.64-8000",
  "device_internal_code": "Dbbe16078a0bc386b",
  "event_types": ["motion", "line_crossing", "intrusion"],
  "mqtt": {
    "broker": "tcp://192.168.1.10:1883",
    "client_id": "hik-ipc-01-Dbbe16078a0bc386b",
    "username": "...", "password": "...",
    "topic": "zws/events/Dbbe16078a0bc386b",
    "qos": 1, "tls": false
  },
  "object_store": {
    "provider": "minio", "endpoint": "192.168.1.10:9000",
    "bucket": "zws", "access_key": "...", "secret_key": "...",
    "use_ssl": false, "path_style": true,
    "public_base": "https://img.example.com/zws"
  }
}
```

插件布防步骤：

1. 校验 MQTT 连接（CONNECT）；
2. 校验对象存储可写（小对象 PUT + DELETE）；
3. 打开报警通道 / 事件订阅；
4. 返回 `{ "armed": true, "subscription_id": "..." }`。

Disarm：关闭报警通道、清理事件缓冲，返回 `{ "armed": false }`。

### 4.3 布防状态持久化

新增 `zws_event_subscription`：

```text
id, device_id(统一资产), binding_id, plugin_instance_id,
device_uid, device_internal_code,
event_types_json, armed, status(armed|disarmed|error),
last_armed_at, last_error, create_time, update_time
```

插件重启/重新 Hello 后，平台按该表自动补布防（幂等），避免“平台以为已布防、插件实际没开”。

## 5. 事件模型

### 5.1 MQTT Topic

```text
{topic_prefix}/{deviceInternalCode}     # 按设备分 topic，便于 ACL
```

消费端订阅 `{topic_prefix}/#`；多实例部署使用共享订阅：

```text
$share/zws/{topic_prefix}/#
```

### 5.2 事件 JSON（zws.event.v1）

```json
{
  "schema": "zws.event.v1",
  "eventId": "uuid",
  "deviceInternalCode": "Dbbe16078a0bc386b",
  "deviceUid": "192.168.1.64-8000",
  "channelId": "1",
  "eventType": "motion",
  "severity": "warning",
  "occurredAt": 1710000000000,
  "image": {
    "key": "events/Dbbe16078a0bc386b/2026/11/18/<eventId>.jpg",
    "url": "https://.../....jpg",
    "expiresAt": 1710086400000,
    "width": 1920,
    "height": 1080
  },
  "bbox": { "x": 0.12, "y": 0.20, "w": 0.30, "h": 0.40, "coord": "normalized" },
  "raw": { "vendor": "hikvision", "sdkType": 0 }
}
```

约定：

- `image.key` 必须存在；`image.url` 可为 `publicBase` 直链或预签名 URL，平台入库以 `key` 为准，URL 可重新生成。
- 图片 key 规范：`events/{deviceInternalCode}/{yyyy}/{MM}/{dd}/{eventId}.jpg`，避免覆盖。
- `eventId` 由插件生成并全局唯一；消费者按 unique 索引幂等。
- `occurredAt` 用设备/插件时间；`receivedAt` 由平台记录，二者都保留。
- `bbox` 仅在智能事件有坐标时出现，`coord=normalized` 表示 0~1 相对画面坐标。

### 5.3 平台事件表

新增 `zws_event`：

```sql
CREATE TABLE zws_event (
  id VARCHAR(36) PRIMARY KEY,
  event_id VARCHAR(64) NOT NULL,          -- 唯一，幂等
  schema_version VARCHAR(16),
  source_id VARCHAR(36),                  -- zws_event_source.id
  device_id VARCHAR(36),                  -- 统一资产
  binding_id VARCHAR(36),
  resource_id VARCHAR(36),
  device_internal_code VARCHAR(64),
  device_uid VARCHAR(128),
  channel_external_key VARCHAR(128),
  event_type VARCHAR(64),
  severity VARCHAR(16),
  occurred_at BIGINT,
  received_at BIGINT,
  image_key VARCHAR(512),
  image_url VARCHAR(1024),
  image_expires_at BIGINT,
  bbox_json TEXT,
  raw_json LONGTEXT,
  status VARCHAR(16),                     -- mapped | unmapped | invalid
  alarm_id INT DEFAULT 0,                 -- 可选投影
  create_time VARCHAR(50),
  UNIQUE KEY uk_event_id (event_id),
  KEY idx_event_device_time (device_id, occurred_at),
  KEY idx_event_type_time (event_type, occurred_at)
);
```

消费映射顺序：

```text
deviceInternalCode → zws_physical_device.internal_code
  → access_binding → resource (external_key = channelId)
  → event_source (kind = eventType)
  → zws_event
  → alarm（可选）
```

找不到设备/资源时仍入库，`status=unmapped`，保留原始字段供后续修复。

## 6. 插件实现要点（海康）

### 6.1 事件来源

优先 **ISAPI**（跨平台、字段稳定）：

```text
GET /ISAPI/Event/notification/alertStream   (multipart 长连接)
```

SDK 作为增强：

```text
NET_DVR_SetDVRMessageCallBack_V50
NET_DVR_SetupAlarmChan_V41 / NET_DVR_CloseAlarmChan_V30
```

### 6.2 抓图

- SDK：`NET_DVR_CaptureJPEGPicture`
- 或 ISAPI：`GET /ISAPI/Streaming/channels/<id>/picture`

抓图失败时：事件仍可发布，`image` 为空并标记 `imageError`；平台入库不阻塞。

### 6.3 上传

插件内置 S3/MinIO 客户端（复用 `minio-go`），按 Arm 下发的配置直传；不落本地磁盘（或仅临时文件）。上传失败重试 3 次，仍失败则发布无图事件。

### 6.4 MQTT 发布

- QoS 默认 1；`client_id` 由平台下发，保证设备级唯一。
- 断线重连；发送失败进入内存队列（容量可配，默认 1000），按指数退避重发。
- 后续可加磁盘 spool（`HIK_SPOOL_DIR`）保证进程重启不丢事件。

### 6.5 插件能力声明

Hello 增加：

```json
"capabilities": ["live","ptz","alarm","arming","snapshot","mqtt"]
```

平台据此开放布防入口；未声明的插件不显示布防按钮。

## 7. 平台消费者

新增 `internal/application/event`（或 `eventingest`）：

- 启动时按 `mqtt_config` 建立订阅；配置保存后热重载。
- 收到消息：解析 → 校验 → 幂等去重 → 映射资产/资源 → 入库 → 可选 Alarm 投影。
- 失败消息进入死信 topic：`{topic_prefix}/dlq`，并记录错误。
- 指标：接收数、成功数、重复数、未映射数、失败数、消费延迟。

依赖引入：`github.com/eclipse/paho.mqtt.golang`（平台消费 + 插件发布共用）。

## 8. 前端改造

| 页面 | 改造 |
| --- | --- |
| `存储管理 → 基础设施` | MySQL/Redis 状态与配置（改完需重启）；MQTT/对象存储配置与连通性测试 |
| `存储管理 → 对象存储` | 保持现状，增加“事件图片”说明 |
| `设备资产 / 插件接入` | 「布防 / 撤防」按钮；未就绪时禁用并提示缺少配置 |
| `应用管理 → 报警管理` | 兼容原有报警；新增事件详情/图片/bbox |
| 新增 `事件中心`（可选） | 按设备/类型/时间检索事件，查看原图与原始 payload |

## 9. 分步实施

### Step 1：MQTT 配置与消费者骨架
- 表、领域、仓储、API、前端页面、健康检查。
- 平台消费者启动/热重载、订阅日志。
- 验收：保存配置后可连接；未配置时管道状态为 not ready。

### Step 2：布防/撤防与状态
- `event_subscription` 表、Arm/Disarm RPC、平台 API、前置校验。
- Mock 插件实现 Arm/Disarm；Hik 插件打开/关闭报警通道（先不产生事件）。
- 验收：缺对象存储或 MQTT 时布防被拒；配置齐全时插件返回 armed；重启插件后平台自动补布防。

### Step 3：插件抓图 + 上传 + MQTT 发布
- Hik 插件接入 ISAPI alertStream（或 SDK 回调）、抓图、MinIO/S3 上传、MQTT 发布。
- Mock 插件增加“模拟事件”按钮/定时事件。
- 验收：插件发布的事件能在 MQTT 客户端看到，图片对象可访问。

### Step 4：平台消费与资产关联
- `zws_event` 入库、幂等、设备内码映射、资源映射、未映射状态。
- 事件查询 API + 事件中心页面；设备详情展示最近事件。
- 验收：MQTT 事件入库后，能按设备内码/事件类型/时间查询到图片与 bbox。

### Step 5：可靠性、安全与运维
- 断线重试/内存队列/死信；共享订阅；指标。
- MQTT TLS/ACL；插件不落盘密钥；Arm 配置脱敏日志。
- 自动补布防、配置变更重下发。

### Step 6：事件类型扩展
- 运动、遮挡、视频丢失、越界、区域入侵、人脸、车牌等映射表。
- 告警投影规则；AI 框叠加；保留策略与清理。

## 10. 已确认的设计选择

1. **配置位置**：`存储管理` 统一管理 MySQL、Redis、MQTT、对象存储；MySQL/Redis 改完需重启，MQTT/对象存储热重载。
2. **事件图片 URL**：使用 `publicBase` 直链；无 `publicBase` 时回退预签名，平台入库以 `image.key` 为准。
3. **图片上传**：平台把对象存储 AK/SK 下发给插件，插件直传；视为可信接入网络。
4. **智能事件来源**：ISAPI `alertStream` 优先，HCNetSDK 报警回调作为增强。
5. **多实例消费**：启用 MQTT 共享订阅（`$share/zws/...`）。
