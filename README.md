# zero-web-kit（ZWS · Zero Web Server）

国标 GB28181 + ONVIF 设备管理平台。后端 Go 重写，前端 Vue 2 管理台，流媒体对接 **[zero-media-server](https://github.com/zero-pipe/zero-media-server)**（ZMS）。

## 技术栈

| 层级 | 技术 |
|------|------|
| Web | Gin |
| 架构 | DDD 分层 |
| 数据库 | MySQL（库名 `zws`，表前缀 `zws_*`） |
| 缓存 | Redis |
| 流媒体 | [zero-media-server](https://github.com/zero-pipe/zero-media-server)（ZMS，HTTP API + Hook） |
| ONVIF | `3rdpart/onvif-go`（本地 vendored） |
| 前端 | Vue 2 + Element UI |

## 目录结构

```
zero-web-kit/
├── 3rdpart/                 # 第三方源码（vendored）
│   └── onvif-go/            # ONVIF 客户端/发现库
├── cmd/server/              # 程序入口
├── configs/                 # config.yaml 主配置；config.local.yaml 可选本地覆盖（不入库）
├── config/                  # 运行时密钥（jwk.json，不入库）
├── docker/                  # MySQL + Redis Compose
├── docs/                    # 补充文档（含 DEPLOY.md 部署指南）
├── internal/
│   ├── application/         # 应用服务（用例编排）
│   ├── domain/              # 领域模型与仓储接口
│   ├── infrastructure/      # DB / Redis / SIP / zero-media-server 客户端 / ONVIF
│   └── interfaces/          # HTTP API、Hook 回调
├── migrations/              # SQL 迁移（表名 zws_*）
├── pkg/
│   ├── jwt/                 # JWT 签发
│   ├── log/                 # 结构化日志 + 文件轮转
│   └── response/            # 统一 JSON 响应（兼容原前端）
├── resources/               # 静态资源（如 civilCode.csv）
├── tools/                   # 开发脚本（dev.ps1 / dev.sh 一键启停）
├── web/                     # Vue 前端
├── logs/                    # 运行时日志（gitignore）
├── Makefile
└── README.md
```

## 快速开始

完整 **Windows / Linux、有无 Docker** 的编译、部署、运行、验证说明见：

**[docs/DEPLOY.md](docs/DEPLOY.md)**

### 最简流程（Linux + Docker + 本机 zero-media-server）

```bash
# 编辑 configs/config.yaml；密码可放 configs/config.local.yaml（不入库）
# 若用 compose，mysql.password 改为 root
make docker-up
make tidy && make build && make run
# 另开终端
make frontend-install && make frontend-dev
```

默认账号：`admin` / `admin`

## 流媒体（zero-media-server / ZMS）

> **说明**：`zero-media-kit` 是 ZMS 内部的容器/协议库；zero-web-kit 对接的是 **zero-media-server** 产品（HTTP API + Hook），不是 media-kit 库本身。

- 源码目录：与 zero-web-kit 同级的 **`zms/`**，编译产物为 `demo_media_server`
- 联调配置：`zms/conf/config.zero-web-kit.ini`（Hook → `:18080`，HTTP → `:8080`）
- 平台配置：`configs/config.yaml` 中 `media.type: zms`（兼容 `zeromediakit` / `zlm` 等旧值）
- 编译与启动详见 [docs/DEPLOY.md](docs/DEPLOY.md) 第九节；ZMS 日志级别见 [zms/README.md](../zms/README.md)
- WebRTC 信令可由平台反向代理：`/index/api/webrtc`

媒体节点管理页使用 zero-media-server 品牌图标，已移除 ZLMediaKit 选项。

## GB28181

`sip` 段需与设备侧一致：域、平台 ID、注册密码、监听端口。

## 开发

### 一键启动（推荐）

**默认不需要 Docker**——本机已有 MySQL（:3306）和 Redis（:6379）时，直接：

```powershell
# Windows
.\tools\dev.ps1 start

# 先检查依赖是否就绪
.\tools\dev.ps1 check
```

```bash
# Linux / macOS
./tools/dev.sh start
./tools/dev.sh check
```

`config.yaml` 里 `mysql.password` / `redis.password` 需与本机服务一致；也可用 `configs/config.local.yaml` 覆盖敏感项。

**有 Docker 时**（自动拉起 MySQL + Redis 容器）：

```powershell
.\tools\dev.ps1 start -Docker
```

```powershell
.\tools\dev.ps1 stop
```

```bash
./tools/dev.sh start --docker
./tools/dev.sh stop
```

未装 Docker 却加了 `-Docker` / `--docker` 会明确报错并提示改用本机数据库。

**流媒体联调**（可选）：`-Media` / `--media`（需先编译 `../zms`）

```powershell
.\tools\dev.ps1 start -Media          # 本机 DB
.\tools\dev.ps1 start -Docker -Media  # Docker DB + ZMS
.\tools\dev.ps1 start -Detached       # 后台，不占终端
.\tools\dev.ps1 stop | status
```

| 地址 | 说明 |
|------|------|
| http://localhost:9528 | 前端（dev 代理 `/dev-api` → 后端） |
| http://localhost:18080 | Go API |
| http://localhost:8080 | zero-media-server（`-Media` / `--media`） |

日志：`.dev/logs/`。首次会自动 `npm install`；请确保已有 `configs/config.yaml`。

### 分开启动（传统）

```bash
make test
make build
make docker-up    # MySQL + Redis
make run          # 终端 1：后端
make frontend-dev # 终端 2：前端
```

### 提交前检查

```bash
make tidy
make test
make build
git status
```

## 版本

当前后端版本：`0.8.0`（`cmd/server/main.go`）

## 许可证

见各 `3rdpart/` 子项目及根目录 LICENSE（如有）。
