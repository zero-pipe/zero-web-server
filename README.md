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
├── configs/                 # config.yaml、jwk.json、civilCode.csv；config.local.yaml（可选，不入库）
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
├── tools/                   # dev.ps1 / dev.sh 一键启停
├── web/                     # Vue 管理台
├── logs/                    # 运行时日志（gitignore），默认 zero-web-server.log
├── Makefile
└── README.md
```

