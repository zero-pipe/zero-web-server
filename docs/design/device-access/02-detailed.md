# 设备接入统一 · 详细设计

> 配套：[概要设计](./01-overview.md) · [UI 设计稿](../../ui/device-management/mockup-unified-device.html)  
> 约束：贴合当前 DDD 分层与现有国标/ONVIF/ZMS 实现；本期不做物理大一统拆表重构（可选二期）。

---

## 1. 视图模型（API / 前端统一）

对外统一设备 DTO（门面聚合，不必一期等同物理表）：

```go
// 示意：internal/application/deviceaccess/dto.go
type AccessMode string // "passive" | "active"
type Protocol   string // "gb28181" | "onvif"
type DeviceStatus string // "pending" | "online" | "offline"

type DeviceView struct {
    ID           string       `json:"id"`           // 业务主键：国标=deviceId；ONVIF=内部 id 或 onvif-{ip}
    Name         string       `json:"name"`
    AccessMode   AccessMode   `json:"accessMode"`
    Protocol     Protocol     `json:"protocol"`
    Vendor       string       `json:"vendor"`       // 可选
    DeviceType   string       `json:"deviceType"`   // IPC/NVR/... 可选
    Status       DeviceStatus `json:"status"`
    ChannelCount int          `json:"channelCount"`
    // 展示用摘要
    AddressText  string       `json:"addressText"`  // "192.168.1.100:80" 或 "待注册" / "ip:sipPort"
    // 协议扩展（原样回传编辑表单）
    GB     *GBConfig     `json:"gb,omitempty"`
    Onvif  *OnvifConfig  `json:"onvif,omitempty"`
    Capabilities []string `json:"capabilities"` // play,ptz,alarm,sync,catalog...
}

type GBConfig struct {
    DeviceID              string `json:"deviceId"`
    Password              string `json:"password"`
    SdpIP                 string `json:"sdpIp"`
    MediaServerID         string `json:"mediaServerId"`
    Charset               string `json:"charset"`
    GeoCoordSys           string `json:"geoCoordSys"`
    SSRCCheck             bool   `json:"ssrcCheck"`
    AsMessageChannel      bool   `json:"asMessageChannel"`
    BroadcastPushAfterAck bool   `json:"broadcastPushAfterAck"`
}

type OnvifConfig struct {
    IP       string `json:"ip"`
    Port     int    `json:"port"`
    Username string `json:"username"`
    Password string `json:"password"`
}
```

创建请求：

```go
type CreateDeviceRequest struct {
    AccessMode AccessMode `json:"accessMode" binding:"required"`
    Protocol   Protocol   `json:"protocol" binding:"required"`
    Name       string     `json:"name"`
    Vendor     string     `json:"vendor"`
    DeviceType string     `json:"deviceType"`
    GB         *GBConfig  `json:"gb"`
    Onvif      *OnvifConfig `json:"onvif"`
    // 主动可选
    ProbeOnCreate   bool `json:"probeOnCreate"`
    SyncOnCreate    bool `json:"syncOnCreate"`
}
```

校验规则：

| 条件 | 规则 |
|------|------|
| `accessMode=passive` | `protocol` ∈ {gb28181}；`gb.deviceId` 必填 |
| `accessMode=active` | `protocol` ∈ {onvif}；`onvif.ip/port` 必填 |
| 模式与协议不匹配 | 400 |

---

## 2. 物理存储（一期兼容方案）

### 2.1 映射

| Protocol | 设备表 | 通道 |
|----------|--------|------|
| gb28181 | 现有 `zws_device` | 现有 Catalog → `zws_device_channel`（`data_type=1`） |
| onvif | 现有 `zws_onvif_device` | **新增**：同步时 Upsert `zws_device_channel`（`data_type=4`），并保留 `zws_onvif_channel` 作协议细节或逐步只读 |

### 2.2 国标表增量字段（建议）

在 `zws_device` 增加（若已有等同语义可复用）：

| 字段 | 类型 | 说明 |
|------|------|------|
| `pre_registered` | tinyint | 1=运维预置（默认新插入为 1） |
| `vendor` | varchar | 可选厂商 |
| `device_kind` | varchar | 可选设备类型 IPC/NVR |

历史已在线设备：迁移脚本置 `pre_registered=1`，避免被新校验误伤。

### 2.3 ONVIF → 统一通道 Upsert 规则

沿用 `commonchannel` / StreamPush 的 Upsert 思路：

| `zws_device_channel` 字段 | 取值 |
|---------------------------|------|
| `device_id` | ONVIF 设备业务 ID（与门面 ID 一致） |
| `gb_device_id` | 通道对外 ID：可用 `{deviceId}_{profileToken}` 或稳定 hash |
| `data_type` | `4`（已有常量） |
| `data_device_id` | `zws_onvif_channel` 主键 |
| `gb_name` | 默认 `{设备名}_通道{N}`；若有 Profile Name 则 `{设备名}_{profileName}` |
| 经纬度等 | 空，留给后续扩展表 |

### 2.4 二期物理统一（仅规划）

```text
zws_access_device (
  id, name, access_mode, protocol, vendor, device_kind,
  status, channel_count, created_at, updated_at
)
zws_access_device_config (
  device_id, config_json  -- GB/ONVIF 专有字段
)
```

本期不实施，避免拖慢统一 UI/预注册。

---

## 3. 状态机

### 3.1 被动（国标）

```text
[创建预置] → pending
     │
     │ REGISTER 鉴权成功且 deviceId 命中预置
     ▼
  online ←── Keepalive ──→（超时）offline
     │
     │ 运维删除
     ▼
  removed（行删除或软删）
```

- `pending`：未成功注册过，或主动置待上线。  
- 列表「地址」列：`pending` 显示「待注册」；`online/offline` 显示 SIP 来源地址。

### 3.2 主动（ONVIF）

```text
[创建] →（可选 probe）
   成功 → online + 可选 sync channels
   失败 → offline（仍入库，便于改密重试）
周期 Probe：刷新 online/offline
```

---

## 4. 国标预注册（安全改造）

### 4.1 现有路径

`infrastructure/sip` REGISTER → `deviceapp.SaveRegister` → 落库 Online。

### 4.2 目标行为

```text
On REGISTER:
  1. 解析 deviceId
  2. 查 zws_device
  3. 若不存在：
       - 若 gb.require_pre_register=true → 401/403，不建档
       - 若 false → 保持旧行为（兼容开关）
  4. 若存在：走现有 Digest；成功 → Online + 后续 Catalog
```

配置建议（YAML / 库表均可，与现有 gb sip config 并列）：

```yaml
gb:
  require_pre_register: true
```

### 4.3 与「添加设备」关系

UI/API 创建国标设备 = 插入 `zws_device`（`pre_registered=1`, 初始离线/pending），**不**等待设备在线。

---

## 5. ProtocolAdapter（包内接口）

位置建议：`internal/application/deviceaccess/adapter.go`

```go
type Adapter interface {
    Protocol() Protocol
    AccessMode() AccessMode
    Create(ctx context.Context, req CreateDeviceRequest) (*DeviceView, error)
    Update(ctx context.Context, id string, req CreateDeviceRequest) (*DeviceView, error)
    Delete(ctx context.Context, id string) error
    ToView(ctx context.Context, raw any) (*DeviceView, error)
    SyncChannels(ctx context.Context, id string) error
    Probe(ctx context.Context, id string) error
    Capabilities(ctx context.Context, id string) []string
}

type Registry struct {
    byProtocol map[Protocol]Adapter
}
```

实现：

| Adapter | 委托 |
|---------|------|
| `GBAdapter` | `deviceapp.Service` + 现有 repo |
| `OnvifAdapter` | `onvifapp.Service` + 通道 Upsert |

门面 `deviceaccess.Service`：

- `List`：并行查两库 → map ToView → 内存过滤排序分页（数据量通常可控；后续再下推 SQL）  
- `Create`：Registry 分发  

**不**在本期做 go-plugin 热加载；字典表可先用前端常量 + 后端枚举。

---

## 6. HTTP API

前缀建议：`/api/devices`（与现有 `/api/device`、`/api/onvif` 并存一版）。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/devices` | query: `q, accessMode, protocol, status, page, count` |
| POST | `/api/devices` | 创建（向导提交） |
| GET | `/api/devices/:id` | 详情；`id` 建议带协议前缀或 query `protocol=` |
| PUT | `/api/devices/:id` | 更新 |
| DELETE | `/api/devices/:id` | 删除 |
| POST | `/api/devices/:id/sync-channels` | ONVIF 同步；国标可映射为刷新目录 |
| POST | `/api/devices/:id/probe` | 主动探测 |
| GET | `/api/devices/:id/channels` | 统一通道列表（读 `zws_device_channel`） |

ID 约定（一期）：

- 国标：`gb:{deviceId}` 或直接 `deviceId`（20 位）+ 响应里带 `protocol`  
- ONVIF：`onvif:{dbId}`  

列表聚合必须返回 `protocol`，避免歧义。

旧接口保留：现有 GB/ONVIF handler 暂不删，门面内部调用，降低一次性改爆风险。

---

## 7. 播放 / 控制路由

`commonchannel.Service` 现状：PTZ 已有 `switch data_type`，Play 偏国标。

本期补全：

```text
Play(channel):
  data_type=1 → playapp（INVITE）
  data_type=4 → onvifapp.Play（RTSP proxy）
  其他保持原样

PTZ / Preset:
  已有分支则复用；缺 ONVIF 则接 onvifapp 现有云台
```

通道页、统一通道列表、设备下钻通道，全部只认 `zws_device_channel` 主键。

---

## 8. 前端详细设计

### 8.1 路由与菜单

`web/src/layout/menu.js`：

```js
children: [
  { title: '设备列表', path: '/devices', icon: 'device' },
  // 部标、通道、推流、代理保留
]
```

`router/index.js`：

- 新增 `/devices` → `views/devices/index.vue`  
- `/device`、`/onvifDevice` → redirect `/devices`（可带 query `protocol=`）

### 8.2 页面结构

```text
views/devices/
  index.vue          # 列表 ↔ 通道下钻（学现有 device/index.vue）
  list.vue           # 统一表格 + 筛选 + 工具按钮
  addWizard.vue      # 步骤1模式+协议 / 步骤2协议表单
  forms/
    GbForm.vue       # 从 device/edit.vue 抽离字段
    OnvifForm.vue    # 从 onvifDevice/list.vue 对话框抽离
  channel/index.vue  # 下钻；按 protocol 显示差异列，播放走统一 API
```

### 8.3 列表操作矩阵

| 操作 | 国标 | ONVIF |
|------|------|-------|
| 编辑 | ✓ | ✓ |
| 通道 | ✓ | ✓ |
| 删除 | ✓ | ✓ |
| 刷新/目录 | ✓ | — |
| 布防/撤防 | ✓（能力有则显示） | — |
| 同步通道 | — | ✓ |
| 探测 | — | ✓ |

工具栏全局：

- **接入信息**：原国标 `configInfo` 对话框  
- **局域网发现**：原 ONVIF discover → 回填主动添加表单  

### 8.4 通道类型色

`main.js` 增加：

```js
4: { id: 4, name: 'ONVIF设备', style: { color: '#7c3aed', borderColor: '#ddd6fe' } }
```

### 8.5 交互文案（与设计稿一致）

- 被动提示：「先添加，再接入；设备编号须与摄像机一致。」  
- 主动提示：可选「保存前校验连通性 / 成功后同步通道」。

---

## 9. 通道命名

| 场景 | 规则 |
|------|------|
| 首次生成 | `{device.Name}_通道{index}`，index 从 1 |
| 国标 Catalog 带回 Name | 优先用设备上报名称；若空则回退默认规则 |
| ONVIF Profile Name | `{device.Name}_{profileName}`；空则 `_通道{N}` |
| 用户在 CommonChannelEdit 改名 | 尊重用户覆盖，同步时不强制打回（可配置） |

---

## 10. 配置与开关

| 键 | 默认 | 含义 |
|----|------|------|
| `gb.require_pre_register` | `true`（新装） | 未预置拒绝注册 |
| `onvif.upsert_common_channel` | `true` | 同步时写统一通道表 |
| `deviceaccess.list_sources` | `gb,onvif` | 门面聚合来源，便于以后加协议 |

升级安装：若检测已有大量「非预置产生」的国标设备，安装说明中提示可将 `require_pre_register` 临时设为 `false`。

---

## 11. 测试要点

| 用例 | 期望 |
|------|------|
| 预置国标 → 设备用正确 ID/密码注册 | 上线成功，通道 Catalog 正常 |
| 未预置国标注册 | 拒绝；库中无新设备 |
| 添加 ONVIF 并校验 | 在线 + 通道出现在设备下钻与通道列表，`data_type=4` |
| 统一列表筛选被动/国标 | 仅国标行 |
| 统一通道播放 ONVIF | 出流成功 |
| 旧 URL `/device` | 进入统一列表 |
| 删除设备 | 协议侧设备删掉；通道清理策略与现网一致（级联删） |

---

## 12. 任务拆分（实现序）

1. **文档确认**：本详细设计 + HTML 设计稿评审  
2. **后端**：`deviceaccess` 门面 + List/Create；GB 预注册开关  
3. **后端**：ONVIF Sync → Upsert `zws_device_channel`  
4. **后端**：commonchannel Play 路由 `data_type=4`  
5. **前端**：`/devices` 列表 + 添加向导 + 菜单  
6. **前端**：通道类型色 + 下钻页  
7. **兼容**：旧路由 redirect；开关与迁移说明写入 `docs/DEPLOY.md` 片段  

---

## 13. 验收标准（Definition of Done）

- [ ] 菜单仅保留一个「设备列表」入口（部标等除外）  
- [ ] 可添加被动国标设备，未预置无法上线  
- [ ] 可添加主动 ONVIF 设备，通道进入统一通道表  
- [ ] 列表可按模式/协议/状态筛选，两类设备同表展示  
- [ ] 从统一通道或设备下钻可播放国标与 ONVIF  
- [ ] 设计稿主要交互与实现一致（步骤、字段、标签语义）  

---

## 14. 后续扩展怎么接（铺路说明）

新增协议（如 ISUP 被动 / 私有 TCP 主动）时：

1. 增加 `Protocol` 枚举与 Adapter 实现  
2. 前端 `forms/XxxForm.vue` + 向导协议下拉  
3. `data_type` 新值 + 通道 Upsert  
4. Play/控制在 commonchannel 加分支  
5. **不必**再新增一级菜单  

事件表、扩展属性表按概要设计非目标，待设备门面稳定后再立专项。
