# zero-web-server（ZWS · Zero Web Server）

[![GitHub](https://img.shields.io/badge/GitHub-zero--pipe%2Fzero--web--server-blue?logo=github)](https://github.com/zero-pipe/zero-web-server)

国标 **GB28181** + **ONVIF** 设备管理平台。后端 Go，前端 Vue 2 管理台，流媒体对接 **[zero-media-server](https://github.com/zero-pipe/zero-media-server)**（ZMS）。

| 项 | 说明 |
|----|------|
| 仓库 | https://github.com/zero-pipe/zero-web-server |
| Go module | `zero-web-server` |
| 默认账号 | `admin` / `admin` |
| 后端版本 | `1.0.0`（`cmd/server/main.go`） |

## 技术栈

| 层级 | 技术 |
|------|------|
| Web | Gin |
| 架构 | DDD 分层 |
| 数据库 | MySQL（库名 `zws`，表前缀 `zws_*`） |
| 缓存 | Redis |
| 流媒体 | [zero-media-server](https://github.com/zero-pipe/zero-media-server)（ZMS，HTTP API + Hook） |
| GB28181 | `3rdpart/gb28181-go` |
| ONVIF | `3rdpart/onvif-go`（本地 vendored） |
| 前端 | Vue 2 + Element UI |
| 日志 | `logs/zero-web-server.log`（text / json，可轮转） |

## 目录结构

```
zero-web-server/
├── 3rdpart/
│   ├── gb28181-go/          # 国标 SIP 栈
│   └── onvif-go/            # ONVIF 客户端 / 发现
├── cmd/server/              # 程序入口
├── configs/                 # config.yaml；jwk.json；config.local.yaml（可选，不入库）
├── docker/                  # MySQL + Redis Compose
├── docs/                    # 部署与设计文档（含 DEPLOY.md）
├── internal/
│   ├── application/         # 应用服务（用例编排）
│   ├── domain/              # 领域模型与仓储接口
│   ├── infrastructure/      # DB / Redis / SIP / ZMS 客户端 / ONVIF
│   └── interfaces/          # HTTP API、Hook 回调
├── sql/                     # 建表 SQL（表名 zws_*）
├── pkg/
│   ├── jwt/
│   ├── log/                 # 结构化日志 + 文件轮转 + 实时推送
│   └── response/            # 统一 JSON 响应 {code,msg,data}
├── resources/               # 静态资源（如 civilCode.csv）
├── tools/                   # dev.ps1 / dev.sh 一键启停
├── web/                     # Vue 管理台
├── logs/                    # 运行时日志（gitignore），默认 zero-web-server.log
├── Makefile
└── README.md
```

## 快速开始

完整 **Windows / Linux、有无 Docker** 说明见 **[docs/DEPLOY.md](docs/DEPLOY.md)**。

### 最简流程（Linux + Docker + 本机 ZMS）

```bash
git clone https://github.com/zero-pipe/zero-web-server.git
cd zero-web-server

# 编辑 configs/config.yaml；密码可放 configs/config.local.yaml（不入库）
# 若用 compose，mysql.password 改为 root
make docker-up
make tidy && make build && make run
# 另开终端
make frontend-install && make frontend-dev
```

| 地址 | 说明 |
|------|------|
| http://localhost:9528 | 前端（dev 代理 → 后端） |
| http://localhost:18080 | Go API / Hook |
| http://localhost:8080 | zero-media-server（可选） |

## 流媒体（zero-media-server / ZMS）

> **说明**：`zero-media-kit` 是 ZMS 内部协议库；本仓库对接的是 **zero-media-server** 产品（HTTP API + Hook），不是 media-kit 本身。

- 源码：与本仓库同级的 **`zms/`**，编译产物一般为 `demo_media_server`
- 联调配置：优先 `zms/conf/config.zero-web-server.ini`（若存在），否则用 `zms/conf/config.ini`（Hook → `:18080`，HTTP → `:8080`）
- 平台侧：`configs/config.yaml` 中媒体节点以库表为准；类型为 `zms`
- 编译与启动见 [docs/DEPLOY.md](docs/DEPLOY.md)；ZMS 说明见 [zms/README.md](../zms/README.md)
- WebRTC 信令可由平台反代：`/index/api/webrtc`

## GB28181

`configs/config.yaml` 的 `sip` 段需与设备侧一致：域、平台 ID、注册密码、监听端口（默认 `8116`）。

## 开发

### 一键启动（推荐）

本机已有 MySQL（:3306）和 Redis（:6379）时，无需 Docker：

```powershell
# Windows
.\tools\dev.ps1 start
.\tools\dev.ps1 check
```

```bash
# Linux / macOS
./tools/dev.sh start
./tools/dev.sh check
```

敏感项可用 `configs/config.local.yaml` 覆盖。

**Docker 拉起 MySQL + Redis：**

```powershell
.\tools\dev.ps1 start -Docker
.\tools\dev.ps1 stop
```

```bash
./tools/dev.sh start --docker
./tools/dev.sh stop
```

**附带 ZMS 联调**（需先编译 `../zms`）：

```powershell
.\tools\dev.ps1 start -Media
.\tools\dev.ps1 start -Docker -Media
.\tools\dev.ps1 start -Detached
.\tools\dev.ps1 stop
.\tools\dev.ps1 status
```

开发日志也可看 `.dev/logs/`；进程正式日志默认写入 `logs/zero-web-server.log`。

### 分开启动

```bash
make test
make build
make docker-up      # MySQL + Redis
make run            # 终端 1：后端 → bin/zero-web-server
make frontend-dev   # 终端 2：前端
```

### 提交前检查

```bash
make tidy
make test
make build
git status
```

## 日志约定

| 配置项 | 说明 |
|--------|------|
| `log.level` | `debug` / `info` / `warn` / `error` |
| `log.format` | `text`（人读）或 `json`（采集） |
| `log.file.path` | 默认 `logs/zero-web-server.log` |

text 示例：

```text
2026-07-14 17:09:21.165 INFO  zero-web-server starting version=1.0.0 http=:18080
```

管理台「实时日志 / 历史日志」读取同一套日志文件。

## 许可证

见各 `3rdpart/` 子项目及根目录 LICENSE（如有）。
