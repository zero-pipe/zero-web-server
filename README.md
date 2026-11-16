# zero-web-server（ZWS · Zero Web Server）

[![GitHub](https://img.shields.io/badge/GitHub-zero--pipe%2Fzero--web--server-blue?logo=github)](https://github.com/zero-pipe/zero-web-server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](go.mod)

国标 **GB28181** + **ONVIF** 设备管理平台。后端 Go（Gin + DDD 分层），前端 Vue 2 管理台，流媒体对接 **[zero-media-server](https://github.com/zero-pipe/zero-media-server)**（ZMS）。

| 项 | 说明 |
| --- | --- |
| 仓库 | <https://github.com/zero-pipe/zero-web-server> |
| Go module | `zero-web-server` |
| 默认账号 | `admin` / `admin`（首次登录后请修改） |
| 后端版本 | `1.0.0`（`cmd/server/main.go`） |
| API 前缀 | `/api/v1` |

## 功能

- 国标 GB28181：注册、心跳、目录、设备信息、订阅、布防、录像控制、PTZ、移动位置、报警。
- ONVIF：设备发现、主动接入、通道同步、拉流、PTZ、预置位。
- 流媒体：实时点播、录像回放/下载、云录像、录像计划、推流、拉流代理、媒体节点集群。
- 级联：上级平台注册/目录推送；下级平台接入。
- 平台能力：用户/角色/菜单权限、操作日志、实时日志 WebSocket、设备分组、行政区划、地图。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| Web | Gin |
| 架构 | DDD 分层 + 端口适配 |
| 数据库 | MySQL 8（库名 `zws`，表前缀 `zws_*`） |
| 缓存 | Redis 7 |
| 流媒体 | [zero-media-server](https://github.com/zero-pipe/zero-media-server)（HTTP API + Hook） |
| GB28181 | `3rdpart/gb28181-go`（Git 子模块） |
| ONVIF | `3rdpart/onvif-go`（本地 vendored） |
| 前端 | Vue 2 + Element UI |
| 日志 | `logs/zero-web-server.log`（text / json，可轮转） |

## 快速开始

前置：Go 1.25.x、Node.js 18+、MySQL 8、Redis 7。

```bash
# 1. 克隆（含子模块）
git clone --recursive https://github.com/zero-pipe/zero-web-server.git
cd zero-web-server

# 2. 启动 MySQL + Redis（可选，使用 Docker）
make docker-up

# 3. 初始化数据库
mysql -uroot -p -e "CREATE DATABASE IF NOT EXISTS zws DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -uroot -p zws < sql/init_zws_mysql.sql

# 4. 本地配置（放密码等，不入库）
cp configs/config.yaml configs/config.local.yaml
# 编辑 config.local.yaml：MySQL / Redis / SIP 等

# 5. 运行后端
make run            # 或 make build && ./bin/zero-web-server -config configs/config.yaml

# 6. 运行前端
cd web && npm install && npm run dev
```

浏览器访问前端开发地址，使用 `admin` / `admin` 登录。

更多：

- 部署（Docker / 端口 / 升级）：[docs/DEPLOY.md](docs/DEPLOY.md)
- 配置项说明：[docs/CONFIG.md](docs/CONFIG.md)
- 架构与依赖规则：[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- IPC 私有 SDK 插件化接入设计：[docs/IPC_PLUGIN_DESIGN.md](docs/IPC_PLUGIN_DESIGN.md)

## 目录结构

```text
zero-web-server/
├── 3rdpart/
│   ├── gb28181-go/          # 国标 SIP 栈（Git 子模块）
│   └── onvif-go/            # ONVIF 客户端 / 发现（vendored）
├── cmd/server/              # 程序入口与依赖装配
├── configs/                 # config.yaml、civilCode.csv；config.local.yaml（本地覆盖，不入库）
├── docker/                  # MySQL + Redis Compose
├── docs/                    # 部署 / 配置 / 架构文档
├── internal/
│   ├── adapter/             # 端口适配器（媒体集群、对象存储）
│   ├── apperr/              # 带业务码的应用错误
│   ├── application/         # 应用服务（用例编排）
│   ├── domain/              # 领域实体与仓储接口
│   ├── dto/                 # 跨层 DTO
│   ├── gbsip/               # GB28181 协议纯函数
│   ├── infrastructure/      # MySQL / Redis / SIP / ONVIF / 配置
│   ├── interfaces/          # HTTP API、媒体 Hook
│   ├── port/                # 端口接口（SIP / 媒体 / 缓存 / 存储 / 上级平台）
│   ├── mediakit/ mediaurl/ onvifcli/ civilcode/  # 中立工具包
├── pkg/                     # jwt / log / response / idcode
├── sql/                     # 建表 SQL（表名 zws_*）
├── tools/                   # dev.ps1 / dev.sh 一键启停
├── web/                     # Vue 管理台
├── Dockerfile
├── Makefile
└── README.md
```

## 常用命令

```bash
make run            # 运行后端
make build          # 编译到 bin/zero-web-server
make test           # go test ./...
make verify-local   # 本机 Go 版本低于 go.mod 要求时的本地构建/测试
make docker-up      # 启动 MySQL + Redis
make frontend-dev   # 前端开发模式
make frontend-build # 前端构建
```

## 贡献

欢迎提交 Issue / PR。请先阅读 [CONTRIBUTING.md](CONTRIBUTING.md) 与 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。安全问题请按 [SECURITY.md](SECURITY.md) 私下报告。

## 许可证

[MIT](LICENSE)
