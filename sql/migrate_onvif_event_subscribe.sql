-- ONVIF 设备新增「事件订阅开关」。
-- 背景：设备接入页「订阅」列统一为「勾选框 + 事件」，ONVIF 由平台真开关控制 PullPoint 订阅。
-- 说明：zws 启动时 GORM AutoMigrate 会自动补列；本脚本供手工/离线升级时执行。
-- MySQL 8.0 的 ADD COLUMN 不支持 IF NOT EXISTS，重复执行会报「列已存在」，
-- 属预期，可忽略报错。

ALTER TABLE zws_onvif_device
    ADD COLUMN event_subscribe TINYINT(1) DEFAULT 1
        COMMENT '事件订阅开关（1=开启 PullPoint）';
