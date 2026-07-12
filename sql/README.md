# 数据库 SQL

## 基本原则

| 场景 | 用什么 |
|------|--------|
| **新安装** | 只执行全量 SQL：`init_zws_mysql.sql`（已含 ONVIF / 国标 SIP / 云录像 `play_url` 等） |
| **升级已有库** | 依赖进程启动时的 GORM **AutoMigrate** 补缺表/缺列；必要时再跑增量脚本（如旧库缺 ONVIF 时用 `add_onvif_tables.sql`） |

**不要把 AutoMigrate 当成新安装建表手段。** 新环境必须先有完整 SQL，保证不启后端也能核对表结构。

表名统一 `zws_*`。默认管理员：`admin` / `admin`。

## 文件

| 文件 | 用途 |
|------|------|
| `init_zws_mysql.sql` | **新安装全量建表**（唯一必跑脚本） |
| `add_onvif_tables.sql` | 旧库升级补 ONVIF 表（`CREATE IF NOT EXISTS`，新装可跳过） |
| `schema_manifest.yaml` | 表清单说明 |

## 新安装

```bash
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS zws CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -u root -p zws < sql/init_zws_mysql.sql
```

PowerShell：

```powershell
$mysql = "mysql"   # 或完整路径
& $mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS zws CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
cmd /c "`"$mysql`" -u root -p zws < sql\init_zws_mysql.sql"
```

验证：

```sql
SHOW TABLES LIKE 'zws_%';
SELECT username FROM zws_user WHERE username='admin';
DESCRIBE zws_cloud_record;   -- 应有 play_url
DESCRIBE zws_gb_sip_config;  -- 国标 SIP
```

## 升级（已有库）

1. 部署新版本后端并启动 → AutoMigrate 补列（如历史库缺 `play_url`）。
2. 若库很旧且完全没有 ONVIF 表：`mysql ... < sql/add_onvif_tables.sql`。
