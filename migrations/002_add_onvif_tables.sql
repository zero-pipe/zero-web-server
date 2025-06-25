-- ONVIF device extension tables for zero-web-kit
-- Run after WVP 2.7.4 base schema is initialized.

CREATE TABLE IF NOT EXISTS wvp_onvif_device (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    name            VARCHAR(255) COMMENT '设备名称',
    ip              VARCHAR(50) NOT NULL COMMENT '设备IP',
    port            INT DEFAULT 80 COMMENT 'ONVIF端口',
    username        VARCHAR(255) COMMENT '认证用户名',
    password        VARCHAR(255) COMMENT '认证密码',
    manufacturer    VARCHAR(255) COMMENT '厂商',
    model           VARCHAR(255) COMMENT '型号',
    firmware        VARCHAR(255) COMMENT '固件版本',
    serial_number   VARCHAR(255) COMMENT '序列号',
    hardware_id     VARCHAR(255) COMMENT '硬件ID',
    device_uri      VARCHAR(512) COMMENT 'Device Service URI',
    media_uri       VARCHAR(512) COMMENT 'Media Service URI',
    ptz_uri         VARCHAR(512) COMMENT 'PTZ Service URI',
    on_line         TINYINT(1) DEFAULT 0 COMMENT '在线状态',
    discovery_mode  TINYINT DEFAULT 0 COMMENT '0=手动 1=自动发现',
    media_server_id VARCHAR(50) DEFAULT 'auto',
    custom_name     VARCHAR(255),
    server_id       VARCHAR(50),
    create_time     VARCHAR(50),
    update_time     VARCHAR(50),
    UNIQUE KEY uk_onvif_ip_port (ip, port)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='ONVIF设备';

CREATE TABLE IF NOT EXISTS wvp_onvif_channel (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    device_id       BIGINT NOT NULL COMMENT '关联 wvp_onvif_device.id',
    profile_token   VARCHAR(255) NOT NULL COMMENT 'ONVIF Profile Token',
    name            VARCHAR(255) COMMENT '通道名称',
    video_source    VARCHAR(255) COMMENT 'VideoSource Token',
    encoder_token   VARCHAR(255) COMMENT 'VideoEncoder Token',
    resolution      VARCHAR(50) COMMENT '分辨率',
    codec           VARCHAR(50) COMMENT '编码格式',
    has_audio       TINYINT(1) DEFAULT 0,
    has_ptz         TINYINT(1) DEFAULT 0,
    stream_uri      VARCHAR(512) COMMENT 'RTSP地址',
    status          VARCHAR(50) DEFAULT 'OFF',
    create_time     VARCHAR(50),
    update_time     VARCHAR(50),
    UNIQUE KEY uk_device_profile (device_id, profile_token),
    KEY idx_device_id (device_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='ONVIF通道';
