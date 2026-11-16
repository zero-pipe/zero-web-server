# 贡献指南（Contributing）

感谢参与 zero-web-server（ZWS）！提交前请阅读以下约定。

## 环境准备

- Go 1.25.x（`go.mod` 要求 1.25.4）
- Node.js 18+（前端 `web/`）
- MySQL 8、Redis 7（可用 `make docker-up` 启动）
- 可选：zero-media-server（ZMS）用于流媒体

拉取代码时务必带子模块：

```bash
git clone --recursive https://github.com/zero-pipe/zero-web-server.git
# 已克隆的仓库：
git submodule update --init --recursive
```

## 本地开发

```bash
cp configs/config.yaml configs/config.local.yaml   # 本地覆盖（含密钥），该文件不入库
# 按需修改 config.local.yaml 中的 MySQL / Redis / SIP 等配置
make docker-up      # 启动 MySQL + Redis（可选）
make run            # 启动后端
make frontend-dev   # 启动前端
```

数据库初始化脚本：`sql/init_zws_mysql.sql`。

## 提交规范

- 使用 [Conventional Commits](https://www.conventionalcommits.org/)：
  `feat: ...` / `fix: ...` / `refactor: ...` / `docs: ...` / `chore: ...`
- 一个提交只做一件事，避免混入无关格式化。
- 提交前确保：`go build ./...`、`go test ./...` 通过；前端 `npm run build` 通过。

## 分支与 PR

1. 从 `master` 切出 `feature/<name>` 或 `fix/<name>`。
2. PR 描述写清：背景、改动点、验证方式、影响范围。
3. 涉及配置/接口变更时同步更新 `docs/` 与 `README.md`。

## 代码风格

- Go：`gofmt`；应用层不得直接依赖 `internal/infrastructure`（`config` 除外），依赖通过 `internal/port` 注入。
- Vue/JS：遵循仓库内 ESLint 配置。

## 安全

请勿提交真实密码、私钥等敏感信息；使用 `configs/config.local.yaml` 或环境变量。安全漏洞请按 [SECURITY.md](SECURITY.md) 私下上报。
