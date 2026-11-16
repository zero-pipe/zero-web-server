# 架构说明（Architecture）

ZWS 后端采用分层架构，依赖方向严格向内：`interfaces → application → domain`，基础设施通过端口实现。流媒体本身不在此项目内实现，而是对接 [zero-media-server](https://github.com/zero-pipe/zero-media-server)（ZMS）。

## 分层与目录

```
cmd/server/           程序入口，手工依赖装配（main.go）
internal/
  domain/             领域实体与仓储接口（无框架依赖）
  application/        应用服务（用例编排）
  port/               端口接口：媒体集群、对象存储、SIP 会话、设备缓存、上级平台客户端
  adapter/            端口适配器（媒体集群、对象存储）
  infrastructure/     MySQL(GORM)、Redis、SIP 栈封装、ONVIF、配置加载
  interfaces/         HTTP API、媒体 Hook 回调
  gbsip/              GB28181 协议纯函数与类型别名（application 可安全依赖）
  dto/                跨层数据传输对象
  apperr/             带业务码的应用错误
  mediakit/           对 ZMS HTTP API 的客户端
  mediaurl/           播放/推流地址拼装
  onvifcli/           ONVIF 发现/端点解析
  civilcode/          行政区划 CSV 加载
pkg/                  jwt、log、response、idcode 等通用库
web/                  Vue 2 管理台
3rdpart/              第三方子模块/本地依赖（gb28181-go、onvif-go）
sql/                  建表脚本
```

## 关键约束

- **应用层依赖规则**：`internal/application/**` 只依赖 `domain`、`port`、`dto`、`gbsip` 与少量中立工具包；共享值对象 `internal/infrastructure/config` 例外。禁止直接依赖 `infrastructure/{sip,persistence,redis,onvif}`。
- **端口注入**：SIP 能力通过 `port.GBSession`（按角色拆分：Inviter/Playbacker/PTZNode/DeviceControl）；设备缓存通过 `port.DeviceCache`（Redis 不可用时用 `port.NoopDeviceCache`）；媒体通过 `port.MediaCluster`。
- **回调方向**：SIP 栈通过 `port.DeviceLifecycle` 回调设备应用服务，基础设施不反向依赖具体服务类型。

## 设备接入

- `deviceaccess` 是统一设备门面，内部用 `ProtocolAdapter` 注册表派发：
  - `gbAdapter`：国标被动接入（注册 / 目录 / 订阅 / 控制）。
  - `onvifAdapter`：ONVIF 主动接入（发现 / 通道 / 拉流 / PTZ）。
  - `jt1078Adapter`：部标占位，接入时替换实现即可。
- 新增协议：实现 `ProtocolAdapter` 并在 `NewService` 注册，无需改动 HTTP 层。

## 流媒体与 Hook

ZMS 通过 `/index/hook/*` 回调平台（`on_publish`、`on_stream_changed`、`on_record_mp4` 等）。`interfaces/hook` 将事件分发给 `StreamNotifier` 列表（点播/回放/云录/推流/代理/录像计划）。

## 错误处理

应用服务返回 `internal/apperr.AppError{Code,Msg}`，handler 统一用 `response.FromError` 映射。码段：`10xxx` 通用、`11xxx` 设备、`12xxx` 媒体、`13xxx` 流、`14xxx` 存储、`15xxx` 用户、`16xxx` 级联、`17xxx` 分组；`401/403` 语义保持不变。

## HTTP API

- 前缀 `/api/v1`；健康检查 `/health`、实时日志 `/channel/log`、媒体 Hook `/index/hook/*`、WebRTC 反代 `/index/api/*` 不在 `/api/v1` 下。
- 统一响应：`{ "code": 0, "msg": "success", "data": ... }`。
