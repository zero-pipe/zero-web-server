-- 媒体节点自注册能力补齐（对应 zws_media_server 新增列）。
-- 说明：zws 启动时 GORM AutoMigrate 会自动补列；本脚本供手工/离线升级时执行。
-- MySQL 8.0 的 ADD COLUMN 不支持 IF NOT EXISTS，重复执行会报「列已存在」，
-- 属预期，可忽略报错。逐条执行更安全。

ALTER TABLE zws_media_server ADD COLUMN srt_port       INT          COMMENT 'SRT端口';
ALTER TABLE zws_media_server ADD COLUMN webrtc_port_min INT         COMMENT 'WebRTC ICE UDP 起始端口';
ALTER TABLE zws_media_server ADD COLUMN webrtc_port_max INT         COMMENT 'WebRTC ICE UDP 结束端口';
ALTER TABLE zws_media_server ADD COLUMN capabilities_reported TINYINT(1) DEFAULT 0 COMMENT '端口与能力是否由节点自注册上报';
ALTER TABLE zws_media_server ADD COLUMN enable_rtsp    TINYINT(1)   DEFAULT 0 COMMENT '是否开启RTSP';
ALTER TABLE zws_media_server ADD COLUMN enable_rtmp    TINYINT(1)   DEFAULT 0 COMMENT '是否开启RTMP';
ALTER TABLE zws_media_server ADD COLUMN enable_hls     TINYINT(1)   DEFAULT 0 COMMENT '是否开启HLS';
ALTER TABLE zws_media_server ADD COLUMN enable_srt     TINYINT(1)   DEFAULT 0 COMMENT '是否开启SRT';
ALTER TABLE zws_media_server ADD COLUMN enable_webrtc  TINYINT(1)   DEFAULT 0 COMMENT '是否开启WebRTC';
ALTER TABLE zws_media_server ADD COLUMN version        VARCHAR(50)  COMMENT '节点版本';
ALTER TABLE zws_media_server ADD COLUMN last_seen      BIGINT       DEFAULT 0 COMMENT '最近一次注册/心跳时间(unix ms)';

-- 旧表可能缺失、但模型已引用的端口列（如已存在可忽略）：
ALTER TABLE zws_media_server ADD COLUMN rtmp_ssl_port  INT          COMMENT 'RTMPS端口';
ALTER TABLE zws_media_server ADD COLUMN rtsp_ssl_port  INT          COMMENT 'RTSPS端口';
ALTER TABLE zws_media_server ADD COLUMN http_ssl_port  INT          COMMENT 'HTTPS端口';
