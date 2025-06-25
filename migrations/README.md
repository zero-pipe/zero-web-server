# 数据库迁移

脚本已收拢到本目录，注释统一为 MySQL 标准 `--` 行注释。  
表清单见 **`schema_manifest.yaml`**（必需 / 可选 / 废弃）。

## 优先级与文件说明

| 优先级 | 文件 | 用途 |
|--------|------|------|
| P0 | `schema_manifest.yaml` | 表清单单一真相源 |
| P1 | `004_init_wvp_mysql_native.sql` | **新环境推荐**：纯 MySQL 8 全量建表 |
| P1 | `002_add_onvif_tables.sql` | ONVIF 扩展表 |
| P2 | `003_update_wvp_mysql.sql` | 从旧 WVP 2.7.4 脚本升级时的增量补丁 |
| P2 | `005_fix_mobile_position.sql` | 修正 `channel_id` 为 INT（已有库） |
| P3 | `006_drop_legacy_tables.sql` | 删除 WVP 遗留表（可选） |
| 遗留 | `001_init_wvp_mysql.sql` | WVP 兼容语法，仅作归档参考 |

默认管理员：`admin` / `admin`

---

## 全新安装（推荐）

```bash
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS wvp CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -u root -p wvp < migrations/004_init_wvp_mysql_native.sql
mysql -u root -p wvp < migrations/002_add_onvif_tables.sql
```

PowerShell：

```powershell
cd E:\13_zero\gb28181\zero-web-kit
$mysql = "D:\Program Files\mysql-8.0.34-winx64\bin\mysql.exe"
& $mysql -u root --password=你的密码 -e "CREATE DATABASE IF NOT EXISTS wvp CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
cmd /c "`"$mysql`" -u root --password=你的密码 wvp < migrations\004_init_wvp_mysql_native.sql"
cmd /c "`"$mysql`" -u root --password=你的密码 wvp < migrations\002_add_onvif_tables.sql"
```

---

## 已有库（从 WVP / 001 导入过）

```bash
mysql -u root -p wvp < migrations/003_update_wvp_mysql.sql
mysql -u root -p wvp < migrations/002_add_onvif_tables.sql
mysql -u root -p wvp < migrations/005_fix_mobile_position.sql
# 可选：清理遗留表
mysql -u root -p wvp < migrations/006_drop_legacy_tables.sql
```

---

## 验证

```sql
USE wvp;
SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'wvp';
SHOW TABLES LIKE 'wvp_onvif%';
SELECT username FROM wvp_user WHERE username = 'admin';
SHOW COLUMNS FROM wvp_mobile_position LIKE 'channel_id';
```

GORM AutoMigrate 仍会在启动时确保 `wvp_onvif_device`、`wvp_onvif_channel` 存在。
