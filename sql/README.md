# 数据库 SQL

新环境表结构以本目录脚本为准（表名 `zws_*`）。  
后端启动时也会用 GORM AutoMigrate 自动建表/补列，多数情况只需 MySQL 可连即可。

## 文件

| 文件 | 用途 |
|------|------|
| `init_zws_mysql.sql` | 新库全量建表（推荐；Docker 初始化用） |
| `add_onvif_tables.sql` | ONVIF 设备/通道表 |
| `schema_manifest.yaml` | 表清单说明 |

默认管理员：`admin` / `admin`（脚本或启动种子写入）

## 手工初始化（可选）

```bash
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS zws CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -u root -p zws < sql/init_zws_mysql.sql
mysql -u root -p zws < sql/add_onvif_tables.sql
```

PowerShell：

```powershell
$mysql = "mysql"   # 或完整路径
& $mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS zws CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
cmd /c "`"$mysql`" -u root -p zws < sql\init_zws_mysql.sql"
cmd /c "`"$mysql`" -u root -p zws < sql\add_onvif_tables.sql"
```
