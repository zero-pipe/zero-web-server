# zero-web-server · AI 指引

> 平台工程（Go + Gin + Vue2）。跨工程背景先读上级 [`../Claude.md`](../Claude.md)。
> 本文只回答：目录怎么分层、改哪里、怎么跑、踩过哪些坑。

## 1. 组成

- `cmd/server/main.go` — 平台主进程（HTTP + SIP + 后台任务装配）
- `cmd/zero-agent/main.go` — 现场 Agent：扫描 `plugins/*/plugin.yaml`、拉起插件进程、反向代理 RPC
- `internal/` — 后端（分层见下）
- `web/` — Vue2 管理端（vue-cli，dev 端口 9528，代理到 18080）
- `sql/init_zws_mysql.sql` — **权威建表脚本**；`sql/migrate_*.sql` — 手工迁移
- `docs/` — 接入模型/插件/国标场景文档（先读）

## 2. 后端分层（依赖方向不可反）

```
interfaces(HTTP/Hook) → application → domain
                         ↓
                       port（media/sip/objectstore/device-cache/platform-client）
                         ↑
              adapter / infrastructure（MySQL/Redis/SIP/ONVIF/插件RPC）
```

关键包：
| 包 | 职责 |
|---|---|
| `internal/application/deviceaccess` | 三类接入适配器 + 注册表（新增协议从这里扩展） |
| `internal/application/asset` + `infrastructure/persistence/asset_backfill.go` | 统一资产投影与孤儿清理 `Prune()` |
| `internal/application/play / playback / ptz / channel` | 取流、回放、云台、通道 |
| `internal/application/mediaserver` + `internal/mediakit` | 媒体节点管理与 ZMS HTTP 客户端 |
| `internal/application/accesstoken` | 对外接入 token 可视化配置 + 同步到 ZMS 本地白名单 |
| `internal/application/upstream / downstream / cascade` | 级联目录与上级/下级平台 |
| `internal/gbsip` | GB28181 SIP（注册、目录、级联） |
| `internal/mediaurl` | **流命名唯一来源**：`live/<设备内码>_<通道内码>` |
| `internal/civilcode` + `application/region / group` | 行政区划、业务分组 |
| `internal/application/plugin / ipcdevice / agent` | 插件与 Agent 链路 |

## 3. 数据与身份

- 统一资产：`zws_physical_device` → `zws_access_binding` → `zws_resource` → `zws_stream_profile`，能力 `zws_capability`，事件源 `zws_event_source`，映射 `zws_unified_ref`。
- 旧表（`zws_device` / `zws_device_channel` / `zws_onvif_*` / `zws_plugin_*`）仍存在，投影到统一资产。
- 内码 `D...`/`C...` 是平台身份；GB 码只用于协议交互。**不要用协议码拼流名**。
- 建表以 `sql/init_zws_mysql.sql` 为准；GORM `AutoMigrate` 只是启动兜底（补列/补缺失表）。**模型里的约束（如唯一索引）必须与 SQL 同步**，否则 AutoMigrate 建的库会缺约束（区划重复问题就是这么来的）。

## 4. 运行与验证

```bash
make run            # 后端 go run ./cmd/server -config configs/config.yaml
make frontend-dev   # 前端 dev（9528）
make dev / dev-stop # 一键起停（本地 MySQL/Redis）
make test
make agent-build    # zero-agent + mock 插件
make mock-plugin    # 单独起 mock 插件联调
```

依赖：MySQL（库 `zws`）、Redis（db 7）、一个可达的 ZMS 节点（页面「媒体节点」添加）。
平台不内嵌 ZMS；没有 ZMS 时设备能接、目录能出，但取流会失败。

## 5. 对接密钥与认证

### 5.1 平台 ↔ ZMS 媒体节点

- 页面「媒体节点」新增节点：IP / HTTP 端口 / **SECRET**；SECRET 必须与 ZMS `conf/config.ini` 的 `[api] secret` 完全一致。
- 落库 `zws_media_server.secret`；调用走 `internal/mediakit/client.go`（每个 API 带 `?secret=`）。
- 不一致的表现：平台里节点"不可达/鉴权失败"，ZMS 日志拒绝。改密钥两边同步。
- ZMS 回调平台：`/index/hook/*`（`internal/interfaces/hook/`），地址配在 ZMS `[hook]`，**不鉴权**，只应对可信内网。
- 详见 [`../zmedia-server/Claude.md`](../zmedia-server/Claude.md) 第 5 节。

### 5.1.1 对外接入 token（推/拉流）

- 页面「接入鉴权」（运维管理，`/accessToken`）：可视化配置外部集成密钥，`zws_access_token` 落库。
- 后端 `internal/application/accesstoken/` + `persistence/access_token_repository.go` + `routes_access_token.go`；外部携带 `?token=`。
- 启用中的 token 自动同步到在线 ZMS（`/index/api/auth/tokens`），节点注册/心跳时重推；默认关闭（无启用 token 即不鉴权）。
- 规范全文见 [`docs/接口与鉴权规范.md`](docs/接口与鉴权规范.md)。

### 5.2 平台 ↔ 客户端（zplayer）

- 登录：`GET /api/v1/user/login?username=&password=`，密码 **MD5 后**提交；响应 `data.accessToken` / `username` / `serverId`。
- 鉴权头：`access-token`（非 Bearer）；任意接口 401 = 登录过期，客户端回登录页。
- 统一响应 `{code,msg,data}`，`code != 0` 即业务错误。
- 目录 `GET /api/v1/group/tree/list`；取流 `GET /api/v1/play/start/{设备国标编号}/{通道国标编号}`、停止 `play/stop/...`（返回多个候选地址，客户端选传输方式）；云台/预置位 `front-end/ptz|preset/...`。
- 改接口时同步 `zplayer/src/platform/web_platform_api.cpp` 与 `platform_api.h`。

### 5.3 推拉流 token（可选）

- 平台签发逻辑 `internal/application/publishauth/`，与 ZMS `[auth]` 配合；内网联调保持关闭。

## 6. 前端要点

- API 层 `web/src/api/*.js`，状态 `web/src/store/modules/*.js`，页面 `web/src/views/<模块>/`。
- 统一响应拦截：`web/src/utils/request.js`；轮询类接口传 `silent:true` 避免弹全局错误。
- 播放：`views/common/playerTabs.vue` + `utils/playerStrategy.js`（标准/超低延迟/高清 的归一化选型）+ 全局播放模式 `store/modules/player.js`。
- 不要在任何页面硬编码引擎名（Jessibuca/WebRTC/H265web）给用户看。

## 7. 常见坑（已踩过）

- **区划重复/删不掉/加不了下级**：`zws_common_region` 缺唯一索引导致重复行；删父行后子行 `parent_id` 悬挂成孤儿（树上不可见却阻塞重建）。处理：`sql/migrate_region_dedupe.sql`，孤儿在根层显示可直接删。
- **设备删除后资产残留**：必须走统一删除（`/api/v1/devices/delete`），它会在删除后调用 `AssetProjector.Prune()`；旧的按协议删除接口只能删旧表。
- **播放弹"流不存在或已离线"**：流尚未在 ZMS 注册时的轮询噪声，界面轮询应静默。
- **gb.require_pre_register=true** 时未预登记设备注册会被 403，属预期。

## 8. 改动自检清单

- [ ] 新协议：只加适配器 + 注册表，不改 HTTP 层
- [ ] 加字段：同时改 `sql/init_zws_mysql.sql` 与 `model/entity.go`
- [ ] 流名/取流：只经 `mediaurl` + `mediakit`
- [ ] 前端错误提示：非用户主动触发的一律 `silent`
- [ ] 跨工程：改了 ZMS 调用点 → 同步确认 ZMS 侧接口
