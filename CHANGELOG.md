# 变更记录（Changelog）

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 与 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### Changed

- **架构重构**：应用层依赖改为端口（`internal/port`）注入，不再直接依赖 `internal/infrastructure`（`config` 共享值对象除外）。
- 拆分 `device` 应用服务为注册 / 目录 / 控制 / 订阅 / 统计等内聚组件。
- 设备接入门面 `deviceaccess` 引入 `ProtocolAdapter` 注册表，新增协议只需实现适配器。
- HTTP 路由按模块拆分；API 前缀统一为 `/api/v1`（前端已同步）。
- 引入 `internal/apperr` 统一业务错误码，handler 使用 `response.FromError` 映射。
- 设备注册/心跳改为分列更新，避免全行覆盖与并发写冲突；SSRC 序列原子化。

### Added

- 应用层单元测试（设备注册/目录同步、SSRC 并发、回放参数）。
- `make verify-local`：本机 Go 版本低于锁定工具链时的本地构建/测试工作流。

## [1.0.0] - 2026-11

### Added

- 国标 GB28181 设备接入：注册、心跳、目录、设备信息、订阅、布防、录像控制、PTZ。
- ONVIF 设备主动接入：发现、通道同步、拉流、PTZ、预置位。
- 实时点播、录像回放、下载、云录像与录像计划。
- 推流 / 拉流代理，媒体节点集群管理。
- 上级平台级联与下级平台接入。
- 用户 / 角色 / 菜单权限，操作日志与实时日志 WebSocket。
- Vue 2 管理台。
