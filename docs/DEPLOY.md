# zero-web-kit 编译、部署、运行与验证

本文档覆盖 **Windows（无 Docker）**、**Windows（Docker）**、**Linux（无 Docker）**、**Linux（Docker）** 四种场景。

---

## 一、架构与端口

| 组件 | 默认端口 | 说明 |
|------|----------|------|
| zero-web-kit HTTP | **18080** | REST API + Hook 回调 |
| GB28181 SIP | **8116** | UDP/TCP，设备注册/信令 |
| MySQL | 3306 | 库名 `zws` |
| Redis | 6379 | 设备状态、心跳统计等 |
| zero-media-server | **8080** | 流媒体 HTTP API（需单独部署） |
| 前端开发服 | 9528 | 仅 `npm run dev` 时使用 |

**依赖关系：**

```
浏览器 → zero-web-kit(:18080) → MySQL / Redis
                ↓ SIP :8116
            国标 IPC/NVR
                ↓ RTP
         zero-media-server(:8080) ← Hook 回调到 :18080/index/hook/*
```

---

## 二、环境准备

### 2.1 通用（Windows / Linux）

| 工具 | 版本建议 | 用途 |
|------|----------|------|
| Go | **1.24+** | 编译后端 |
| Node.js | 16+（推荐 18 LTS） | 前端开发/打包 |
| npm | 随 Node | 前端依赖 |
| MySQL | 8.x | 业务库 |
| Redis | 6/7 | 缓存 |
| zero-media-server | 与平台联调版本 | 收流/转发/Hook |

检查命令：

```bash
go version
node -v
npm -v
```

Windows PowerShell 同上。

### 2.2 可选：Docker

仅用于快速拉起 **MySQL + Redis**（不包含 zero-media-server）。

- Windows： [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- Linux： `docker` + `docker compose`（或 `docker-compose`）

验证：

```bash
docker --version
docker compose version
```

---

## 三、数据库初始化（四种场景通用）

**全新安装**（推荐 SQL）：

```bash
# Linux / macOS / Git Bash
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS zws CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -u root -p zws < migrations/004_init_zws_mysql_native.sql
mysql -u root -p zws < migrations/002_add_onvif_tables.sql
```

**Windows PowerShell**（把 `$mysql` 换成你的 mysql.exe 路径）：

```powershell
cd E:\16_project\zero-web-kit
$mysql = "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe"
& $mysql -u root -p你的密码 -e "CREATE DATABASE IF NOT EXISTS zws CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
Get-Content migrations\004_init_zws_mysql_native.sql -Raw | & $mysql -u root -p你的密码 zws
Get-Content migrations\002_add_onvif_tables.sql -Raw | & $mysql -u root -p你的密码 zws
```

默认管理员：**admin / admin**

验证 SQL：

```sql
USE zws;
SELECT username FROM zws_user WHERE username = 'admin';
```

---

## 四、配置文件

主配置为 `configs/config.yaml`。敏感项（密码等）可写在 `configs/config.local.yaml`（不入库，启动时自动合并）。

**必须核对** `configs/config.yaml`（及可选的 `config.local.yaml`）：

```yaml
mysql:
  host: 127.0.0.1
  port: 3306
  user: root
  password: "你的MySQL密码"    # 与真实库一致
  database: zws

redis:
  host: 127.0.0.1
  port: 6379
  password: ""
  database: 7

media:
  id: zms-local
  type: zms
  ip: 127.0.0.1              # zero-media-server 所在 IP
  http_port: 8080            # zero-media-server HTTP 端口
  secret: "与流媒体一致的secret"

sip:
  port: 8116
  domain: "4101050000"       # 与设备/平台国标域一致
  id: "41010500002000000001"
  password: "12345678"       # 设备注册鉴权密码
```

> **注意**：若用本文 Docker Compose 起 MySQL，root 密码为 `root`，请把 `config.yaml` 里 `mysql.password` 改为 `"root"`，而不是 example 里的 `12345678`。

首次启动会自动生成 `config/jwk.json`（JWT 密钥，勿提交 git）。

---

## 五、依赖服务启动

### 5.1 方式 A：Docker（Windows / Linux 相同）

在项目根目录：

```bash
# Linux
make docker-up
# 或
cd docker && docker compose up -d
```

```powershell
# Windows（无 make 时）
cd E:\16_project\zero-web-kit\docker
docker compose up -d
```

停止：

```bash
make docker-down
# 或 cd docker && docker compose down
```

验证容器：

```bash
docker compose -f docker/docker-compose.yml ps
docker compose -f docker/docker-compose.yml logs mysql --tail 20
```

MySQL 首次启动会自动执行 `migrations/004_*.sql` 与 `002_*.sql`。

---

### 5.2 方式 B：Windows 无 Docker

1. **MySQL 8**  
   - 安装后创建库并导入 SQL（见第三节 PowerShell 命令）。  
   - 确认服务运行：`Get-Service MySQL*`

2. **Redis**（任选其一）  
   - [Memurai](https://www.memurai.com/)（Windows 原生）  
   - 或 WSL2 内 `sudo apt install redis-server && redis-server`  
   - 或使用远程 Linux 上 Redis，并修改 `config.yaml` 的 `redis.host`

3. **zero-media-server**  
   - 在本机或局域网机器启动，HTTP 默认 `:8080`  
   - 配置 Hook 指向：`http://<平台IP>:18080/index/hook/`

4. **防火墙**（SIP 必开）  
   PowerShell 管理员：

   ```powershell
   New-NetFirewallRule -DisplayName "GB28181 SIP" -Direction Inbound -Protocol UDP -LocalPort 8116 -Action Allow
   New-NetFirewallRule -DisplayName "GB28181 SIP TCP" -Direction Inbound -Protocol TCP -LocalPort 8116 -Action Allow
   ```

---

### 5.3 方式 C：Linux 无 Docker

```bash
# Ubuntu/Debian 示例
sudo apt update
sudo apt install -y mysql-server redis-server

sudo systemctl enable --now mysql redis-server

# 导入数据库（第三节）
mysql -u root -p zws < migrations/004_init_zws_mysql_native.sql
mysql -u root -p zws < migrations/002_add_onvif_tables.sql

# 防火墙（若启用 ufw）
sudo ufw allow 18080/tcp
sudo ufw allow 8116/tcp
sudo ufw allow 8116/udp
```

zero-media-server 单独编译/启动，与 Windows 相同，改 `media.ip` / `media.http_port` 即可。

---

## 六、编译后端

### 6.1 Windows

```powershell
cd E:\16_project\zero-web-kit
go mod tidy
go build -o bin\zero-web-kit.exe .\cmd\server\
```

### 6.2 Linux

```bash
cd /opt/zero-web-kit   # 你的路径
go mod tidy
go build -o bin/zero-web-kit ./cmd/server/
```

### 6.3 交叉编译（可选）

在 Windows 编 Linux 包：

```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"
go build -o bin/zero-web-kit ./cmd/server/
```

在 Linux 编 Windows 包：

```bash
GOOS=windows GOARCH=amd64 go build -o bin/zero-web-kit.exe ./cmd/server/
```

---

## 七、运行后端

### 7.1 开发运行（改代码即生效）

```bash
# Linux / make
make run
```

```powershell
# Windows
go run .\cmd\server\ -config configs\config.yaml
```

```bash
# Linux 无 make
go run ./cmd/server -config configs/config.yaml
```

### 7.2 生产运行

**Linux（systemd 示例）**

```bash
cd /opt/zero-web-kit
./bin/zero-web-kit -config configs/config.yaml
```

可写 unit 文件 `/etc/systemd/system/zero-web-kit.service`：

```ini
[Unit]
Description=zero-web-kit
After=network.target mysql.service redis.service

[Service]
Type=simple
WorkingDirectory=/opt/zero-web-kit
ExecStart=/opt/zero-web-kit/bin/zero-web-kit -config /opt/zero-web-kit/configs/config.yaml
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now zero-web-kit
sudo systemctl status zero-web-kit
journalctl -u zero-web-kit -f
```

**Windows**

```powershell
cd E:\16_project\zero-web-kit
.\bin\zero-web-kit.exe -config configs\config.yaml
```

日志：控制台 + `logs/zero-web-kit.log`（由 `log.output: both` 控制）。

---

## 八、前端

### 8.1 开发模式（联调推荐）

**一键启动（单窗口，推荐）：**

```powershell
# Windows — 本机 MySQL/Redis（无需 Docker）
cd E:\16_project\zero-web-kit
.\tools\dev.ps1 start
.\tools\dev.ps1 check    # 仅探测 :3306 / :6379 / Docker

# 或用 Docker 拉起 MySQL/Redis
.\tools\dev.ps1 start -Docker
# ZMS 联调加 -Media
```

```bash
# Linux
chmod +x tools/dev.sh
./tools/dev.sh start
./tools/dev.sh start --docker
```

浏览器：**http://localhost:9528**；API 代理到 `:18080`。停止：`dev.ps1 stop` / `dev.sh stop`。

**分开两个终端（传统）：**

```bash
make frontend-install && make frontend-dev   # 终端 2
make run                                     # 终端 1
```

```powershell
go run .\cmd\server\ -config configs\config.yaml
cd web && npm install && npm run dev
```

### 8.2 生产打包

```bash
make frontend-build
# 产物在 web/dist/
```

部署方式任选：

- Nginx 静态托管 `web/dist`，`/dev-api` 反代到 `:18080`  
- 或由网关/Ingress 统一转发

---

## 九、zero-media-server（ZMS）编译与对接

> `zero-media-kit` 是 ZMS 内部的容器/协议库；zero-web-kit 通过 HTTP API + Hook 对接 **zero-media-server**，不直接依赖 media-kit。

ZMS 源码在 **`../zms`**（与 zero-web-kit 同级目录）。完整功能说明见 [zms/README.md](../zms/README.md)。

### 9.1 编译 ZMS（Windows 示例）

```powershell
cd E:\16_project\zms
cmake -S . -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build --config Release
# 产物: build\examples\Release\demo_media_server.exe
```

Linux：

```bash
cd /opt/zms
cmake -S . -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -j
# 产物: build/examples/demo_media_server
```

### 9.2 配置与启动

```powershell
cd E:\16_project\zms
# 使用仓库内联调预设（改 externIP、Hook 地址为实际平台 IP）
.\build\examples\Release\demo_media_server.exe --config conf\config.zero-web-kit.ini
```

`conf/` **应提交**的文件：

| 文件 | 说明 |
|------|------|
| `config.ini.example` | 通用模板 |
| `config.server.ini.example` / `config.embedded.ini.example` | 预设 |
| `config.zero-web-kit.ini` | zero-web-kit 联调预设 |

**勿提交**：`conf/*.bak`、`conf/config.local.ini`、运行时生成的 `config.ini`、日志 `zms_media_server.log`。

日志：`log_level=info` 为默认；排查流问题时临时改 `debug`。

### 9.3 与 zero-web-kit 对齐

1. ZMS HTTP 默认 `:8080`，与 `configs/config.yaml` 中 `media.http_port` 一致  
2. `media.secret` 与 ZMS `[api] secret` 一致（本机联调可都留空）  
3. ZMS `[hook]` 指向 `http://<平台IP>:18080/index/hook/...`（预设为 `127.0.0.1`）  
4. `[general] externIP` 改为浏览器/设备可达的平台或 ZMS 出口 IP  
5. RTP 端口范围与 `media.rtp.port_range` 一致，防火墙放行 UDP  

验证 ZMS：

```powershell
Invoke-RestMethod "http://127.0.0.1:8080/index/api/getServerConfig"
```

---


## 十、验证清单

### 10.1 后端健康检查

**Windows PowerShell：**

```powershell
Invoke-RestMethod http://127.0.0.1:18080/health
Invoke-RestMethod http://127.0.0.1:18080/api/server/version
```

**Linux / curl：**

```bash
curl -s http://127.0.0.1:18080/health
curl -s http://127.0.0.1:18080/api/server/version
```

期望：`health` 返回 JSON 且 `code: 0`；日志出现 `zero-web-kit starting`。

### 10.2 登录

```bash
curl -s "http://127.0.0.1:18080/api/user/login?username=admin&password=admin"
```

或浏览器访问前端，使用 **admin / admin** 登录。

### 10.3 MySQL / Redis

```bash
# MySQL
mysql -u root -p -e "SELECT COUNT(*) FROM zws.zws_device;"

# Redis
redis-cli -n 7 PING
```

### 10.4 日志

```bash
# Linux
tail -f logs/zero-web-kit.log

# Windows
Get-Content logs\zero-web-kit.log -Wait -Tail 50
```

### 10.5 功能冒烟（浏览器）

| 步骤 | 路径 | 期望 |
|------|------|------|
| 登录 | 首页 | 进入控制台无 401 |
| 国标设备 | 设备列表 | 列表加载；刷新 IPC 不长时间卡在 0% |
| 通道 | 通道列表 | 「音频」开关不 404 |
| 分屏监控 | 业务分组树 | 展开根节点不 404 |
| 电子地图 | 地图页 | 底图/列表正常，无连续 404 |
| 点播 | 实时预览 | 有流时 WS-FLV/H265 可播（依赖 zero-media-server） |

### 10.6 GB28181（有设备时）

1. 设备 SIP 指向 `平台IP:8116`，密码与 `sip.password` 一致  
2. 平台日志出现 `GB28181 device registered`  
3. 设备列表显示在线，通道数 > 0  

---

## 十一、常见问题

| 现象 | 排查 |
|------|------|
| 启动报 MySQL 连接失败 | 检查 `config.yaml` 密码、MySQL 是否监听 3306 |
| Redis unavailable 警告 | Redis 未启动或 `database`/`host` 不对；无 Redis 部分功能受限 |
| 前端 404 / 网络错误 | 后端是否在 18080；开发模式是否 `npm run dev` |
| Docker MySQL 连不上 | 密码用 `root`；等待容器 healthy 后再启后端 |
| 设备注册不上 | 防火墙 8116 UDP/TCP；SIP 域/密码/平台 ID 是否与设备一致 |
| 有信令无画面 | zero-media-server 是否运行；Hook 是否指向 18080；RTP 端口是否通 |
| Windows 无 make | 直接用文档中的 `go` / `npm` / `docker compose` 命令 |

---

## 十二、命令速查

| 操作 | Linux | Windows (PowerShell) |
|------|-------|----------------------|
| 拉依赖容器 | `make docker-up` | `cd docker; docker compose up -d` |
| 编译 | `make build` | `go build -o bin\zero-web-kit.exe .\cmd\server\` |
| 运行 | `make run` | `go run .\cmd\server\ -config configs\config.yaml` |
| 前端开发 | `make frontend-dev` | `cd web; npm run dev` |
| 健康检查 | `curl localhost:18080/health` | `Invoke-RestMethod localhost:18080/health` |

更多表结构说明见 [migrations/README.md](../migrations/README.md)。
