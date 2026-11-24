# zero-web-server 数据模型 ER（权威设计）

> 本文是**数据库设计的唯一权威**。改表先改本文，再改 `sql/init_zws_mysql.sql` 与 `model/entity.go`。
> 目标物模型：**设备 → 通道（多） → 码流 + 能力集合（制约操作）**；扩展信息解耦；目录树多挂载；事件按设备/通道关联。

---

## 0. 设计原则

1. **一张表一件事**：稳定接入信息与易变扩展信息分表（`*_ext`），接入后基本不改的主表不被扩展字段污染。
2. **主键**：全部 `BIGINT AUTO_INCREMENT`（InnoDB 聚簇索引最优）。
3. **业务唯一键**：`internal_code`（内码 D/C…）、`gb_code`（国标码，可空）各自 `UNIQUE`，不拿主键当业务标识。
4. **外键**：真实 `FOREIGN KEY`；从属数据 `ON DELETE CASCADE`；可选引用 `SET NULL`。
5. **删设备 = 级联删其通道及通道关联的一切**（码流/能力/扩展/目录挂载/事件/位置…）。
6. **时间**：本轮沿用 `VARCHAR(50)`（`2006-01-02 15:04:05`）。改 `DATETIME` 会连带把全库 Go 字段改 `time.Time`、JSON 输出改 RFC3339，破坏前端日期展示/排序契约，属全局性改动，**单独立项**再做。结构重建不受影响。
7. **长尾字段进 JSON**：不进查询的协议原始属性（原 `gb_*` 40 列、级联 `custom_*` 30 列）一律 JSON。
8. **多态仅限目录挂载**（target=device|channel），其余一律显式外键。

---

## 1. 全局关系总览

```
【物模型·稳定】
 zws_device ─1:1─ zws_device_ext
     │1:N  (ON DELETE CASCADE)
 zws_channel ─1:1─ zws_channel_ext
     │1:N                └─(record_plan_id)─►zws_record_plan
     ├─ zws_stream_profile
     └─ zws_capability

【目录树·展示】
 zws_region(自关联)  zws_group(自关联)
        └──────┬──────────┘
           zws_catalog  (多挂载：一通道可挂多个区划/分组节点)

【事件】
 zws_device ◄─device_id─ zws_event ─channel_id─► zws_channel
                              │ event_type
                              ▼
                        zws_event_type
                              └─1:0..1─► zws_alarm
 zws_event_source(→device/channel)   zws_event_subscription(→device)

【媒体/录像/位置】
 zws_media_server   zws_cloud_record(→channel)   zws_record_plan/item
 zws_stream_proxy   zws_stream_push             zws_mobile_position(→channel)

【级联】 zws_upper_platform  zws_subordinate_platform─catalog/snapshot
 zws_platform_channel(→channel,overrides_json)  zws_platform_group_share
 zws_region_code_mapping  zws_gb_code_seq

【接入/插件】 zws_agent ─ zws_agent_plugin ─ zws_plugin
 zws_plugin_package ─ zws_plugin_install

【用户/配置/审计】 zws_user─role/api_key  zws_access_token
 zws_gb_sip_config  zws_object_store_config  zws_mqtt_config  zws_audit_log
```

---

## 2. 物模型（6 张）

### 2.1 `zws_device` 设备（稳定）
| 字段 | 类型 | 约束/说明 |
|---|---|---|
| id | BIGINT | PK AUTO_INCREMENT |
| internal_code | VARCHAR(32) | **UNIQUE** 内码 D…（对外身份/流名/事件） |
| gb_code | VARCHAR(20) | **UNIQUE**(可空) 国标设备码（国标接入/级联；纯本地其它协议可空） |
| name | VARCHAR(255) | |
| manufacturer / model / firmware / serial_number | VARCHAR | |
| online | TINYINT | 运行时在线 |
| status | VARCHAR(32) | 注册状态 |
| create_time / update_time | DATETIME | |

> 合并了旧 `zws_physical_device` + `zws_onvif_device` + `zws_plugin_device`。接入信息（协议/地址/凭据）在独立的 `zws_access_binding`。

### 2.1.1 `zws_access_binding` 接入绑定（1:1，独立表）
| 字段 | 类型 | 约束 |
|---|---|---|
| id | BIGINT | PK |
| device_id | BIGINT | **UNIQUE + FK→zws_device CASCADE**（一设备一接入） |
| protocol | VARCHAR(16) | gb28181 / onvif / plugin |
| mode | VARCHAR(16) | passive / active |
| external_id | VARCHAR(255) | 协议侧设备号（GB设备码/ONVIF地址/插件uid） |
| endpoint | VARCHAR(255) | ip:port |
| plugin_instance_id | VARCHAR(128) | 插件实例（plugin 协议） |
| config_json | JSON | 协议凭据/参数 |
| status | VARCHAR(32) | 接入状态 |
| last_seen | DATETIME | |
| **UNIQUE(protocol, external_id)** | | 防同设备重复接入 |

### 2.2 `zws_device_ext` 设备扩展（易变，1:1）
| 字段 | 类型 | 约束 |
|---|---|---|
| id | BIGINT | PK |
| device_id | BIGINT | **UNIQUE + FK→zws_device CASCADE** |
| longitude / latitude | DECIMAL(10,7) | 上图 |
| address | VARCHAR(255) | |
| geo_coord_sys | VARCHAR(50) | |
| remark / tags | VARCHAR | |
| extra_json | JSON | 长尾 |

### 2.3 `zws_channel` 通道（稳定）
| 字段 | 类型 | 约束 |
|---|---|---|
| id | BIGINT | PK |
| device_id | BIGINT | **FK→zws_device CASCADE**, INDEX |
| internal_code | VARCHAR(32) | **UNIQUE** 内码 C… |
| gb_code | VARCHAR(20) | **UNIQUE**(可空) 国标通道码 |
| external_key | VARCHAR(255) | ONVIF token / 插件通道号（GB 通道亦存此便于统一） |
| kind | VARCHAR(32) | video_sensor / audio_input / alarm_input / metadata_source |
| name | VARCHAR(255) | |
| online / status | TINYINT / VARCHAR | 运行时 |

> 合并旧 `zws_device_channel` + `zws_onvif_channel` + `zws_plugin_channel` + `zws_resource`。

### 2.4 `zws_channel_ext` 通道扩展（易变，1:1）
| 字段 | 类型 | 约束 |
|---|---|---|
| id | BIGINT | PK |
| channel_id | BIGINT | **UNIQUE + FK→zws_channel CASCADE** |
| longitude / latitude | DECIMAL(10,7) | 上图 |
| map_level | INT | |
| ptz_type | INT | |
| has_audio | TINYINT | |
| enabled | TINYINT | |
| tags | VARCHAR(255) | |
| record_plan_id | INT | **FK→zws_record_plan SET NULL**（可空） |
| gb_json | JSON | 原 gb_* 40 列（国标原始属性存档） |
| extra_json | JSON | 协议长尾（profileToken/resolution/codec…） |

### 2.5 `zws_stream_profile` 码流
| 字段 | 约束 |
|---|---|
| id PK | |
| channel_id | **FK→zws_channel CASCADE** |
| role | main / sub / third |
| name / codec / resolution / selector_json / is_default | |
| **UNIQUE(channel_id, role)** | |

### 2.6 `zws_capability` 能力集合（三态制约操作）
| 字段 | 约束 |
|---|---|
| id PK | |
| channel_id | **FK→zws_channel CASCADE**, INDEX |
| operation | play / playback / ptz / record / snapshot / event |
| supported / configured / available | TINYINT（驱动支持∩已配置∩当前可用） |
| params_json / source | |
| **UNIQUE(channel_id, operation)** | |

---

## 3. 目录树（3 张）

### 3.1 `zws_region` 行政区划
| 字段 | 约束 |
|---|---|
| id PK | |
| parent_id | FK→zws_region SET NULL（自关联） |
| code | VARCHAR(20) **UNIQUE** 区划码 |
| name / sort | |

### 3.2 `zws_group` 业务分组
| 字段 | 约束 |
|---|---|
| id PK | |
| parent_id | FK→zws_group SET NULL（自关联） |
| code | VARCHAR(20) **UNIQUE** 业务分组码（原 215/216 国标节点码） |
| name / sort | |

### 3.3 `zws_catalog` 目录挂载（多挂载）
| 字段 | 约束 |
|---|---|
| id PK | |
| region_id | FK→zws_region CASCADE（可空） |
| group_id | FK→zws_group CASCADE（可空） |
| target_type | device / channel |
| target_id | BIGINT（多态，应用层保证存在） |
| sort | |
| **UNIQUE(region_id, target_type, target_id)** | 同一区划节点不重复挂同一对象 |
| **UNIQUE(group_id, target_type, target_id)** | 同一分组节点不重复挂同一对象 |

> 一条通道可有多行：挂在不同区划/分组节点下（"交界处大家都关注"）。两独立唯一键防同节点重复。

---

## 4. 事件（5 张）

### 4.1 `zws_event_type` 事件类型注册表
`code VARCHAR(64) PK / name / category / severity / alarm_eligible / bbox_capable / vendor_mappings JSON / enabled`

### 4.2 `zws_event` 事件
| 字段 | 约束 |
|---|---|
| id PK | |
| event_id | VARCHAR(64) **UNIQUE**（外部事件号） |
| device_id | **FK→zws_device CASCADE** |
| channel_id | **FK→zws_channel CASCADE**（可空；删通道事件一起删。通道关联即间接关联设备） |
| event_type | VARCHAR(64)（FK 语义指向 event_type.code；未知类型也落此，不再单独 custom 表） |
| severity / status | |
| occurred_at / received_at | DATETIME，occurred_at INDEX |
| bbox_json / image_url / raw_json | |
| **INDEX(device_id, occurred_at) / (event_type, occurred_at)** | |

### 4.3 `zws_alarm` 报警处置投影（1:1 事件）
| 字段 | 约束 |
|---|---|
| id PK | |
| event_id | VARCHAR(64) **UNIQUE + FK→zws_event(event_id) CASCADE** |
| status / handed_by / handled_at / snap_path / record_path / remark | |

### 4.4 `zws_event_source` 事件来源
`id PK / device_id FK CASCADE / channel_id FK SET NULL / kind / external_key / enabled / config_json`

### 4.5 `zws_event_subscription` 事件订阅/布防（插件设备）
`id PK / device_id FK CASCADE / channel_id FK SET NULL / event_types / armed / status / last_error / last_armed_at` —— **UNIQUE(device_id, channel_id)**

---

## 5. 媒体 / 录像 / 位置（7 张）

- **`zws_media_server`**：id PK / name / ip / http_port / rtmp_port / rtsp_port / srt_port / webrtc_port_min·max / secret / enable_rtsp·rtmp·hls·srt·webrtc / last_seen / ...（端口能力齐全）
- **`zws_record_plan`**：id PK / name / snap
- **`zws_record_plan_item`**：id PK / plan_id FK CASCADE / week_day / start_min / stop_min
- **`zws_cloud_record`**：id PK / **channel_id FK CASCADE** / app / stream / media_server_id / start_time / end_time / file_name / file_path / file_size / time_len
- **`zws_stream_proxy`**：id PK / app / stream / src_url / enable / ... —— UNIQUE(app,stream)
- **`zws_stream_push`**：id PK / app / stream / media_server_id / status —— UNIQUE(app,stream)
- **`zws_mobile_position`**：id PK / **channel_id FK CASCADE INDEX** / timestamp / longitude / latitude / altitude / speed / direction

---

## 6. 级联（8 张）

- **`zws_upper_platform`**（原 zws_platform）上级平台：id PK / server_gb_id **UNIQUE** / server_ip·port / device_gb_id / transport / status / catalog_with_platform·group·region / auto_push_channel / ...
- **`zws_subordinate_platform`** 下级平台：id PK / device_gb_id **UNIQUE** / password / status / ip·port
- **`zws_subordinate_catalog`** 下级目录：id PK / platform_id FK CASCADE / device_id 国标 / parent_id / type_code / item_type / orphan / raw_json —— UNIQUE(platform_id, device_id)
- **`zws_platform_catalog_snapshot`** 推送快照：id PK / platform_id FK / device_id / name / parent_id / hash —— UNIQUE(platform_id, device_id)
- **`zws_platform_group_share`** 分组共享：id PK / platform_id FK / group_id FK→zws_group —— UNIQUE(platform_id, group_id)
- **`zws_platform_channel`** 通道对外映射：id PK / platform_id FK / **channel_id FK CASCADE** / custom_gb_code / **overrides_json**（原 30 个 custom_* 列收进 JSON） —— UNIQUE(platform_id, channel_id)
- **`zws_region_code_mapping`** 区划码映射：region_id PK→zws_region CASCADE / gb_code
- **`zws_gb_code_seq`** 国标码序列：prefix VARCHAR(14) PK / next BIGINT

---

## 7. 接入 / 插件（5 张）

- **`zws_agent`**：id VARCHAR PK / hostname / ip / version / endpoint / status / last_seen
- **`zws_plugin`** 插件实例：id VARCHAR PK / plugin_id / vendor / version / endpoint / online / capabilities JSON / config_schema
- **`zws_agent_plugin`** Agent 托管插件：id VARCHAR PK / agent_id FK / plugin_id / desired_state / actual_state / last_seen
- **`zws_plugin_package`** 插件包：id VARCHAR PK / plugin_id / version / name / object_key / sha256 / manifest_json —— UNIQUE(plugin_id, version)
- **`zws_plugin_install`** 安装任务：id VARCHAR PK / agent_id / package_id / plugin_id / status / last_error

> 插件设备/通道不再单独建表 → 统一进 `zws_device`/`zws_channel`（protocol=plugin），凭据进 `config_json`。

---

## 8. 用户 / 权限（4 张）

- **`zws_user`**：id PK / username **UNIQUE** / password(MD5) / role_id FK SET NULL
- **`zws_user_role`**：id PK / name / authority JSON 菜单码
- **`zws_user_api_key`**：id PK / user_id FK CASCADE / app / api_key / expired_at / enable
- **`zws_access_token`** 对外接入密钥：id PK / name / token **UNIQUE** / enabled / remark

> 数据级权限（用户↔设备/通道）本轮留位，后续按需加 `zws_user_data_scope(user_id, target_type, target_id)`。

---

## 9. 系统配置（3 张）

- **`zws_gb_sip_config`**：id=1 固定 / ip / port / domain / device_id / password / alarm / require_pre_register / transport / stream_mode
- **`zws_object_store_config`**：id=1 固定 / enabled / provider / endpoint / region / bucket / access_key / secret_key / use_ssl / path_style / public_base
- **`zws_mqtt_config`**：id=1 固定 / enabled / broker / client_id_prefix / username / password / topic_prefix / qos / keepalive_sec / tls_*

## 10. 审计（1 张，新增）

- **`zws_audit_log`**：id PK / user_id / action / target_type / target_id / detail JSON / ip / create_time（INDEX create_time, user_id）

---

## 11. 删除的旧表（推倒重来）

| 旧表 | 去向 |
|---|---|
| zws_physical_device | 并入 `zws_device` |
| zws_onvif_device / zws_plugin_device | 并入 `zws_device`（protocol 区分） |
| zws_device_channel / zws_onvif_channel / zws_plugin_channel / zws_resource | 并入 `zws_channel` |
| zws_jt_terminal / zws_jt_channel | 删除（死表，无实现） |
| zws_custom_event | 并入 `zws_event`（event_type 未注册即自定义） |
| zws_unified_ref | 删除（无新旧双轨） |
| zws_platform | 改名 `zws_upper_platform` |
| zws_platform_group / zws_platform_region | 并入 `zws_catalog` |
| zws_common_region / zws_common_group | 改名 `zws_region` / `zws_group` |

---

## 12. 表清单（44 张）

物模型 7 · 目录 3 · 事件 6 · 媒体 7 · 级联 8 · 接入 5 · 用户 4 · 配置 3 · 审计 1 = **44**

> 事件族含 `zws_custom_event`（厂商自定义事件 raw 原样保存，与统一事件 `zws_event` 隔离）。
> 落地时对部分表做了“查询字段冗余”补充（在注释中已标）：`zws_alarm` 增加 channel_id/description/
> alarm_type/alarm_code/alarm_time/device_internal_code/channel_internal_code/longitude/latitude/bbox_json
> 便于保留旧报警查询；`zws_device` 增加 enabled；`zws_stream_push`/`zws_stream_proxy` 增加 gb_code/gb_name/
> server_id 等对外字段（替代原“无主通道”伪行）；`zws_cloud_record` 增加 play_url/server_id；`zws_media_server`
> 增加 server_id。均为兼容现有接口契约的最小子集，不改变主模型分层。

## 13. 落地与验证进度

**持久层已全量迁移到本模型并运行时验证（真库 `zws` 重置后跑通）：**

- 构建：`go build ./...` 全绿；`go vet` 干净。
- 建库：`sql/init_zws_mysql.sql` 在空库一次建 44 表 + 种子（admin/角色/SIP/对象存储）。
- 启动：平台 `:18080` + SIP `:5060` 正常起，full routes 注册，`mysql ready`。
- 读路径：`/api/v1/devices`、`/channels`、`/assets/tree`、`/region/tree/list`、`/group/tree/list`
  从统一表正确回填（协议/vendor/经纬度/PTZ/有音频/gb_json 字段、嵌套 binding+resources 均正确）。
- 写路径：`/region/add`（根/子级 parentId 自关联）、`/group/add`（国标 215 业务分组）写入并在库里读回。
- 级联：删设备 → 绑定/通道/扩展/码流/能力/位置/事件/报警 级联归零；`zws_catalog` 因多态 target 保留（应用层清理）。

**尚未合并的内部重构（不影响运行，作为后续独立批）：**

- `domain/asset`（统一模型）与 `domain/device|channel|onvif|plugin`（扁平旧模型）两套 domain 并存，
  当前都读写同一组统一表；合并为一套是纯内部重构，不改变 `/api/v1` 契约。
- 通道隶属（区划 civilCode / 分组 parentId）过渡期存 `channel_ext.gb_json`，与 `zws_catalog` 多挂载并存；
  可后续决定是否把 catalog 作为唯一隶属来源。

**写路径扩展验证（118 个 GET 全表 + 写接口 create/read-back）：**

- 读面：对全部 118 个 GET `/api/v1` 路由无参冒烟，无 500/panic/查询失败；非 0 返回均为
  预期的业务校验（缺参 400、无记录 record not found、服务未启用、无媒体节点）。
- 写面（创建→读回，均落到统一表）：接入 token→`zws_access_token`；拉流代理→`zws_stream_proxy`；
  上级平台→`zws_upper_platform`（`/platform/query` 正常回填）；录像计划+时段→`zws_record_plan`/`_item`；
  录像计划关联通道→`channel_ext.record_plan_id`（`hasLink=true` 过滤正确）。
- 媒体节点保存需真实 ZMS 可达（按预期拒绝并提示连通），非迁移问题。
