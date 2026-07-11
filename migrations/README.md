# 数据库迁移

脚本已收拢到本目录，注释统一为 MySQL 标准 `--` 行注释。  
表清单见 **`schema_manifest.yaml`**（必需 / 可选 / 废弃）。

## 优先级与文件说明

| 优先级 | 文件 | 用途 |
|--------|------|------|
| P0 | `schema_manifest.yaml` | 表清单单一真相源 |
| P1 | `004_init_zws_mysql_native.sql` | **新环境推荐**：纯 MySQL 8 全量建表 |
| P1 | `002_add_onvif_tables.sql` | ONVIF 扩展表 |
| P2 | `003_update_zws_mysql.sql` | 历史库升级时的增量补丁（归档） |
| P2 | `005_fix_mobile_position.sql` | 修正 `channel_id` 为 INT（已有库） |
| P3 | `006_drop_legacy_tables.sql` | 删除历史遗留表（可选） |
| 遗留 | `001_init_zws_mysql.sql` | 历史兼容语法，仅作归档参考 |

默认管理员：`admin` / `admin`

---

## 全新安装（推荐）

后端启动时会自动：

1. `CREATE DATABASE IF NOT EXISTS <配置中的库名>`
2. GORM `AutoMigrate` 创建缺失表 / 补缺列（**不删已有数据**）
3. 若用户表为空，写入默认管理员 `admin` / `admin`

因此多数环境只需保证 MySQL 可连、`configs/config.yaml` 中 mysql 配置正确即可，不必再手工跑建表脚本。

仍可手工初始化（可选）：

```bash
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS zws CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -u root -p zws < migrations/004_init_zws_mysql_native.sql
mysql -u root -p zws < migrations/002_add_onvif_tables.sql
```

PowerShell：

```powershell
cd E:\13_zero\gb28181\zero-web-kit
$mysql = "D:\Program Files\mysql-8.0.34-winx64\bin\mysql.exe"
& $mysql -u root --password=你的密码 -e "CREATE DATABASE IF NOT EXISTS zws CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
cmd /c "`"$mysql`" -u root --password=你的密码 zws < migrations\004_init_zws_mysql_native.sql"
cmd /c "`"$mysql`" -u root --password=你的密码 zws < migrations\002_add_onvif_tables.sql"
```

---

## 已有库（从旧版 / 001 导入过）

```bash
mysql -u root -p zws < migrations/003_update_zws_mysql.sql
mysql -u root -p zws < migrations/002_add_onvif_tables.sql
mysql -u root -p zws < migrations/005_fix_mobile_position.sql
# 可选：清理遗留表
mysql -u root -p zws < migrations/006_drop_legacy_tables.sql
```

---

## 验证

```sql
USE zws;
SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'zws';
SHOW TABLES LIKE 'zws_onvif%';
SELECT username FROM zws_user WHERE username = 'admin';
SHOW COLUMNS FROM zws_mobile_position LIKE 'channel_id';
```

GORM AutoMigrate 在启动时确保业务所需表存在（含 `zws_media_server`、`zws_onvif_*`、用户与国标表等）；已有表只补缺列，不删数据。
