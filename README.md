# zero-web-kit

国标 GB28181 + ONVIF 设备管理平台。后端 Go 重写，前端 Vue 2 管理台，流媒体对接 **zero-media-kit**。

## 技术栈

| 层级 | 技术 |
|------|------|
| Web | Gin |
| 架构 | DDD 分层 |
| 数据库 | MySQL（兼容历史 `wvp_*` 表结构） |
| 缓存 | Redis |
| 流媒体 | [zero-media-kit](https://github.com)（HTTP API + Hook） |
| ONVIF | `3rdpart/onvif-go`（本地 vendored） |
| 前端 | Vue 2 + Element UI |

## 目录结构

```
zero-web-kit/
├── 3rdpart/                 # 第三方源码（vendored）
│   └── onvif-go/            # ONVIF 客户端/发现库
├── cmd/server/              # 程序入口
├── configs/                 # 配置模板（复制为 config.yaml 使用）
├── config/                  # 运行时密钥（jwk.json，不入库）
├── docker/                  # MySQL + Redis Compose
├── docs/                    # 补充文档（含 DEPLOY.md 部署指南）
├── internal/
│   ├── application/         # 应用服务（用例编排）
│   ├── domain/              # 领域模型与仓储接口
│   ├── infrastructure/      # DB / Redis / SIP / mediakit / ONVIF
│   └── interfaces/          # HTTP API、Hook 回调
├── migrations/              # SQL 迁移（历史表名 wvp_*）
├── pkg/
│   ├── jwt/                 # JWT 签发
│   ├── log/                 # 结构化日志 + 文件轮转
│   └── response/            # 统一 JSON 响应（兼容原前端）
├── resources/               # 静态资源（如 civilCode.csv）
├── web/                     # Vue 前端
├── logs/                    # 运行时日志（gitignore）
├── Makefile
└── README.md
```

## 快速开始

完整 **Windows / Linux、有无 Docker** 的编译、部署、运行、验证说明见：

**[docs/DEPLOY.md](docs/DEPLOY.md)**

### 最简流程（Linux + Docker + 本机 media-kit）

```bash
cp configs/config.example.yaml configs/config.yaml   # 改 mysql.password 为 root（若用 compose）
make docker-up
make tidy && make build && make run
# 另开终端
make frontend-install && make frontend-dev
```

默认账号：`admin` / `admin`

## 流媒体（zero-media-kit / ZMS）

- 源码目录：与 zero-web-kit 同级的 **`zms/`**，编译产物为 `demo_media_server`
- 联调配置：`zms/conf/config.zero-web-kit.ini`（Hook → `:18080`，HTTP → `:8080`）
- 平台配置：`configs/config.yaml` 中 `media.type: zeromediakit`（兼容 `zms` / `zlm`）
- 编译与启动详见 [docs/DEPLOY.md](docs/DEPLOY.md) 第九节；ZMS 日志级别见 [zms/README.md](../zms/README.md)
- WebRTC 信令可由平台反向代理：`/index/api/webrtc`

媒体节点管理页使用 zero-media-kit 品牌图标，已移除 ZLMediaKit 选项。

## GB28181

`sip` 段需与设备侧一致：域、平台 ID、注册密码、监听端口。

## 开发

```bash
make test
make build
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
