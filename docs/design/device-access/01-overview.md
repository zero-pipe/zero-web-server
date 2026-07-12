# 设备接入统一 · 概要设计

> 范围：**本期仅统一国标 GB28181 + ONVIF**。目标是一套设备菜单、一套列表、一套添加向导（主动/被动），为后续协议插件铺路。  
> UI 设计稿：[`docs/ui/device-management/mockup-unified-device.html`](../../ui/device-management/mockup-unified-device.html)

---

## 1. 背景与目标

### 现状问题

| 问题 | 表现 |
|------|------|
| 双菜单双栈 | 「国标设备」「ONVIF设备」分列，用户心智割裂 |
| 被动不安全 | 国标设备未预置即可 REGISTER 落库在线 |
| 模型分裂 | `zws_device` / `zws_onvif_device` 两套主表；通道未完全汇入 `zws_device_channel` |
| 扩展成本高 | 新协议要再开菜单 + app 包 + 路由，缺少统一入口 |

### 本期目标

1. **一个设备列表**：按「接入模式 + 协议」筛选，融合展示国标与 ONVIF。  
2. **添加设备向导**：先选被动/主动 → 再选协议 → 填协议表单 → 入库。  
3. **先添加再接入**：被动设备预置后才允许注册成功。  
4. **通道统一命名与入口**：设备下钻通道；ONVIF 通道进入统一通道体系（`data_type=4`）。  
5. **贴合现有框架**：不重写 SIP/ONVIF/ZMS，只在之上加统一门面与薄适配层。

### 非目标（本期不做）

- 部标 JT1078、推流、拉流代理并入统一设备主流程  
- 私有 TCP / 厂商 SDK 插件  
- 统一事件总线、预置位联动、扩展属性表（杆件/朝向等）——仅预留扩展位  

---

## 2. 设计原则

1. **接入模式优先于「有没有固定 IP」**  
   - 被动（Device → Platform）：国标  
   - 主动（Platform → Device）：ONVIF  
2. **核心管生命周期，协议插件管方言**  
   设备 CRUD、状态机、通道归属在核心；SIP/ONVIF 逻辑留在现有 `device` / `onvif` / `sip` 包。  
3. **流媒体不重做**  
   继续走 ZMS；统一播放入口按协议路由到现有 `playapp` / `onvifapp`。  
4. **渐进迁移**  
   一期可「逻辑统一 + 物理表兼容」（外观一表，底层映射两表），二期再物理合并。

---

## 3. 总体架构

```text
┌─────────────────────────────────────────────────────────┐
│  Web：设备列表 / 添加向导 / 设备通道                      │
│  menu: 设备列表（替换 国标设备+ONVIF设备）                 │
└───────────────────────────┬─────────────────────────────┘
                            │ HTTP API（统一 /api/devices）
┌───────────────────────────▼─────────────────────────────┐
│  application/deviceaccess（新门面，薄）                    │
│  - Create / List / Get / Update / Delete                 │
│  - SyncChannels / Probe                                  │
│  - 按 protocol 分发到既有 Service                         │
└─────────────┬─────────────────────────────┬─────────────┘
              │                             │
   ┌──────────▼──────────┐       ┌──────────▼──────────┐
   │ device + sip（国标）  │       │ onvif（ONVIF）       │
   │ 预注册校验加强         │       │ 探测/同步通道        │
   └──────────┬──────────┘       └──────────┬──────────┘
              │                             │
              └──────────┬──────────────────┘
                         ▼
              zws_device_channel（统一通道）
                         │
                         ▼
                    ZMS 流媒体
```

**协议适配约定（本期硬编码两个实现，接口形状为后续插件预留）：**

```text
ProtocolAdapter
  AccessMode()          // passive | active
  Protocol()            // gb28181 | onvif
  ValidateCreate(spec)
  Create(spec) -> Device
  OnPassiveRegister(...) // 仅被动；校验预置
  SyncChannels(deviceId)
  Capabilities(deviceId) // play/ptz/alarm/...
```

---

## 4. 领域概念

| 概念 | 说明 |
|------|------|
| 设备 Device | 接入主体；有 `access_mode`、`protocol`、`vendor`、业务状态 |
| 通道 Channel | 属于设备；统一落 `zws_device_channel`；默认名 `设备名_通道N` |
| 接入模式 | `passive` / `active` |
| 协议 | 本期 `gb28181` / `onvif`；与现有 `data_type` 1 / 4 对应 |
| 状态 | `pending`（待上线/已预置）→ `online` / `offline`；主动探测失败可为 `offline` |

---

## 5. 关键流程

### 5.1 被动 · 国标

```text
运维：添加设备（编号+密码等） → 状态 pending
设备：SIP REGISTER → 平台查预置记录 + Digest → 成功 Online + Catalog
未预置：拒绝（403）或进隔离区（配置开关，默认拒绝）
```

### 5.2 主动 · ONVIF

```text
运维：添加（IP/端口/账号）→ 可选校验连通性 → 入库
平台：GetProfiles → 写通道（data_type=4）→ 周期性 Probe
播放：仍走 addStreamProxy / 现有 onvif 播放链
```

---

## 6. 前端信息架构

| 菜单（现） | 菜单（目标） |
|-----------|-------------|
| 国标设备 `/device` | **设备列表** `/devices`（统一） |
| ONVIF设备 `/onvifDevice` | 合并；旧路由兼容跳转 |
| 通道列表 `/channel` | 保留；补 ONVIF 类型标签 |
| 部标 / 推流 / 代理 | 不动 |

添加向导：模式卡片 → 协议下拉（随模式过滤）→ 协议表单（复用现有字段集）。

列表列：名称、设备标识、模式、协议、地址/注册信息、厂商、通道数、状态、操作（按能力显隐）。

---

## 7. 与现有代码的贴合点

| 层 | 路径 | 改造方式 |
|----|------|----------|
| 菜单 | `web/src/layout/menu.js` | 合并两项为「设备列表」 |
| GB UI | `web/src/views/device/*` | 收敛进统一页或作协议子表单 |
| ONVIF UI | `web/src/views/onvifDevice/*` | 同上 |
| 通道类型色 | `web/src/main.js` `$channelTypeList` | 增加 `4: ONVIF` |
| GB 注册 | `application/device` + `infrastructure/sip` | 增加预置校验 |
| ONVIF | `application/onvif` | 经门面创建；通道 Upsert 到 `zws_device_channel` |
| 公共通道 | `application/commonchannel` | Play/PTZ 按 `data_type` 路由补全 ONVIF |
| 媒体 | ZMS + `media.PublishAuth` | 本期不变 |

---

## 8. 迁移策略（概要）

**推荐一期：外观统一、存储兼容**

- API / UI 统一为 Device 视图模型  
- 国标仍写 `zws_device`；ONVIF 仍写 `zws_onvif_*`，同时保证通道进 `zws_device_channel`  
- List 接口 UNION / 门面聚合两表  

**二期（可选）：物理统一主表**

- `zws_access_device` + `protocol_config` JSON/扩展表  
- 数据迁移脚本；旧表只读一段时间  

详见 [02-详细设计](./02-detailed.md)。

---

## 9. 风险与对策

| 风险 | 对策 |
|------|------|
| 预注册破坏现网「即插即用」习惯 | 配置项 `gb.require_pre_register`，默认新装开启、升级可关一版 |
| ONVIF 通道未进统一表导致播放分裂 | 本期强制 Sync 时 Upsert `data_type=4` |
| 列表聚合性能 | 分页在门面侧；索引按 protocol + status；量级不够再物化视图 |
| 旧书签路由 | `/device`、`/onvifDevice` 兼容重定向 |

---

## 10. 交付切片

| 切片 | 内容 | 验收 |
|------|------|------|
| A. UI | 设计稿确认 → 统一列表 + 添加向导页面 | 菜单一个入口，两种模式可添加 |
| B. API 门面 | `/api/devices` 聚合读写 | 列表可见两类设备 |
| C. 国标预注册 | REGISTER 校验预置 | 未添加设备无法上线 |
| D. ONVIF 通道统一 | 同步写入 `zws_device_channel` | 通道列表可见 ONVIF |
| E. 播放路由 | commonchannel 支持 ONVIF play | 统一通道页可播 |

A → B → C/D 可并行，E 收尾。

---

## 11. 结论

在现有 GB + ONVIF + ZMS 底座上做**统一门面 + 预注册 + 通道汇聚**，是可完成且收益最大的一步。  
把这两种模式跑通后，后续 ISUP / 私有协议只需：字典项 + Adapter 实现 + 表单 schema，不必再拆菜单。
