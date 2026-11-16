# 配置说明（Configuration）

配置文件：`configs/config.yaml`（仓库默认），可通过 `-config` 指定路径。
本地覆盖：与主配置同目录的 `configs/config.local.yaml`（不入库，推荐放密钥）。
环境变量：所有键支持环境变量覆盖，键路径中的 `.` 转为 `_`，例如 `MYSQL_PASSWORD`、`SERVER_PORT`。

## server

| 键 | 默认 | 说明 |
| --- | --- | --- |
| `port` | 18080 | HTTP 监听端口 |
| `mode` | debug | `debug` / `release`（Gin 模式） |

## mysql

| 键 | 说明 |
| --- | --- |
| `host` / `port` | MySQL 地址 |
| `user` / `password` | 账号密码 |
| `database` | 库名，默认 `zws` |
| `charset` | 默认 `utf8mb4` |
| `max_idle_conns` / `max_open_conns` | 连接池 |

## redis

| 键 | 说明 |
| --- | --- |
| `host` / `port` / `password` / `database` | 连接参数 |
| `pool_size` | 连接池大小 |

Redis 不可用时服务仍可启动（设备缓存退化为空实现）。

## sip（GB28181）

| 键 | 说明 |
| --- | --- |
| `ip` | 摄像机可达的平台 IP；空则自动探测/用媒体节点 IP 兜底 |
| `port` | SIP 监听端口，默认 5060 |
| `domain` | 国标域，通常为编号前 10 位 |
| `id` | 本平台 20 位国标编号 |
| `password` | 注册/鉴权密码 |
| `alarm` | 是否启用报警 |

> 启动时若库表 `zws_gb_sip_config` 为空，会用 `sip.*` 作为种子写入库；之后以库为准，页面修改可热更新（改端口需重启）。

## gb

| 键 | 说明 |
| --- | --- |
| `require_pre_register` | `true` 时必须先在平台预添加设备，否则拒绝其 REGISTER |

## log

| 键 | 说明 |
| --- | --- |
| `level` | `debug` / `info` / `warn` / `error` |
| `format` | `text` / `json` |
| `output` | `stdout` / `file` / `both` |
| `file.path` | 日志文件路径，默认 `logs/zero-web-server.log` |
| `file.max_size_mb` / `max_backups` / `max_age_days` / `compress` | 轮转策略 |

## user_settings

| 键 | 说明 |
| --- | --- |
| `play_timeout` | 点播等待超时（毫秒） |
| `record_info_timeout` | 录像查询超时（毫秒） |
| `stream_on_demand` | 无人观看是否按需关流 |
| `push_authority` / `record_push_live` | 推流鉴权 / 推流录像策略 |
| `server_id` | 平台实例标识，多实例部署时区分 |
| `login_timeout` | 登录有效期（分钟） |
| `jwk_file` | JWT 签名密钥文件路径；不存在会自动生成 |

## media（可选）

媒体节点以数据库为准（页面“媒体节点”动态添加）；如需兼容旧部署，可取消 `media:` 段注释并填写 `ip` + `http_port` 作为兜底节点。

## onvif

| 键 | 说明 |
| --- | --- |
| `discovery_timeout` | ONVIF 发现超时（秒） |
| `probe_interval` | 通道探测间隔（秒） |

## 安全建议

- 真实密码放在 `configs/config.local.yaml`，不要提交到仓库。
- 部署后修改默认账号 `admin` / `admin`。
- 生产环境为 Redis 设置密码并限制端口暴露。
