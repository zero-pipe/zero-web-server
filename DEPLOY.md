# 部署指南（Deploy）

## 1. 依赖

| 组件 | 版本 | 说明 |
| --- | --- | --- |
| Go | 1.25.x（go.mod 要求 1.25.4） | 编译后端 |
| Node.js | 18+ | 编译前端 |
| MySQL | 8.x | 库名 `zws` |
| Redis | 7.x | 设备在线状态 / 心跳时间戳 |
| zero-media-server | 兼容版本 | 流媒体，HTTP API + Hook |

## 2. 获取代码

```bash
git clone --recursive https://github.com/zero-pipe/zero-web-server.git
cd zero-web-server
# 已克隆但未拉子模块：
git submodule update --init --recursive
```

`3rdpart/gb28181-go` 为 Git 子模块；`3rdpart/onvif-go` 为本地 vendored 依赖（`go.mod` 中 `replace`）。

## 3. 初始化数据库

```bash
mysql -uroot -p -e "CREATE DATABASE IF NOT EXISTS zws DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -uroot -p zws < sql/init_zws_mysql.sql
```

> 说明：服务启动时也会通过 GORM AutoMigrate 补建缺失的表与基础列（见 `internal/infrastructure/persistence/mysql/mysql.go`），但**生产环境建议以 `sql/init_zws_mysql.sql` 为准**（包含完整索引/默认值），AutoMigrate 仅作为开发期兜底。

或使用 Docker：

```bash
make docker-up   # MySQL:3306 / Redis:6379，自动执行 init SQL
```

## 4. 配置

- 默认配置：`configs/config.yaml`
- 本地/生产覆盖：`configs/config.local.yaml`（已在 `.gitignore`，建议把密码放这里）

```bash
cp configs/config.yaml configs/config.local.yaml
# 修改 MySQL / Redis / SIP / server_id 等
```

关键项：`server.port`（默认 18080）、`mysql.*`、`redis.*`、`sip.*`、`user_settings.server_id`、`user_settings.jwk_file`（不存在会自动生成）。

配置优先级：`config.local.yaml` 覆盖 `config.yaml`；支持环境变量覆盖（键路径中的 `.` 替换为 `_`，如 `MYSQL_PASSWORD`）。详见 [CONFIG.md](CONFIG.md)。

## 5. 编译与运行

```bash
make build          # 输出 bin/zero-web-server
./bin/zero-web-server -config configs/config.yaml

# 或直接运行
make run
```

首次启动会自动创建 `configs/jwk.json`（JWT 签名密钥）与日志文件。默认账号 `admin` / `admin`，请登录后立即修改。

## 6. 前端

```bash
cd web
npm install
npm run build       # 产物在 web/dist，交由 Nginx 等静态服务器托管
```

前端通过 `VUE_APP_BASE_API` 与后端 `/api/v1` 通信；开发态可用 `npm run dev`（见 `web/vue.config.js` 代理）。

## 7. Docker

```bash
docker build -t zero-web-server .
docker run -d --name zws \
  -p 18080:18080 -p 5060:5060/udp -p 5060:5060/tcp \
  -v $PWD/logs:/app/logs \
  -v $PWD/configs/config.local.yaml:/app/configs/config.local.yaml:ro \
  zero-web-server
```

> 容器内 `127.0.0.1` 指向容器自身；MySQL/Redis 请使用可达的主机地址（可在 `config.local.yaml` 覆盖）。

## 8. 端口

| 端口 | 协议 | 用途 |
| --- | --- | --- |
| 18080 | TCP | HTTP API / 静态代理 |
| 5060 | UDP/TCP | GB28181 SIP 信令 |
| 30000+ | UDP/TCP | 媒体 RTP（由 ZMS 管理） |

## 9. 升级

1. 备份数据库。
2. 拉取代码并更新子模块：`git pull && git submodule update --init --recursive`。
3. 执行 `sql/` 下的增量脚本（如有）。
4. 重新编译并重启服务。
