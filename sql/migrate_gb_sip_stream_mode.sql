-- 本级国标配置新增「媒体流传输模式」。
-- 背景：标准接入页不再逐设备配置流传输模式，改为「国标配置 → 本级身份」统一设置。
-- 说明：zws 启动时 GORM AutoMigrate 会自动补列；本脚本供手工/离线升级时执行。
-- MySQL 8.0 的 ADD COLUMN 不支持 IF NOT EXISTS，重复执行会报「列已存在」，
-- 属预期，可忽略报错。

ALTER TABLE zws_gb_sip_config
    ADD COLUMN stream_mode VARCHAR(16) NOT NULL DEFAULT 'UDP'
        COMMENT '媒体流传输模式(UDP/TCP-ACTIVE/TCP-PASSIVE)';
