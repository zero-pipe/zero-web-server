-- 清理旧版 WVP 遗留表（zero-web-kit 不读写）
-- 执行前请确认无业务依赖；详见 schema_manifest.yaml deprecated 段

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS wvp_device_alarm;
DROP TABLE IF EXISTS wvp_device_mobile_position;
DROP TABLE IF EXISTS wvp_gb_stream;
DROP TABLE IF EXISTS wvp_log;
DROP TABLE IF EXISTS wvp_platform_catalog;
DROP TABLE IF EXISTS wvp_platform_gb_channel;
DROP TABLE IF EXISTS wvp_platform_gb_stream;
DROP TABLE IF EXISTS wvp_resources_tree;

SET FOREIGN_KEY_CHECKS = 1;
