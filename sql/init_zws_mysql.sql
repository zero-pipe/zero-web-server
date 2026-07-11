-- zero-web-kit MySQL 8 native schema (canonical)
-- Pure MySQL syntax; use this for new installations instead of 001
-- See schema_manifest.yaml for required/optional/deprecated tables

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 存储国标设备的基础信息及在线状态
DROP TABLE IF EXISTS zws_device;
CREATE TABLE IF NOT EXISTS zws_device
(
    id                                  INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    device_id                           VARCHAR(50) not null COMMENT '国标设备编号',
    name                                VARCHAR(255) COMMENT '设备名称',
    manufacturer                        VARCHAR(255) COMMENT '设备厂商',
    model                               VARCHAR(255) COMMENT '设备型号',
    firmware                            VARCHAR(255) COMMENT '固件版本号',
    transport                           VARCHAR(50) COMMENT '信令传输协议（TCP/UDP）',
    stream_mode                         VARCHAR(50) COMMENT '拉流方式（主动/被动）',
    on_line                             TINYINT(1) DEFAULT 0 COMMENT '在线状态',
    ip                                  VARCHAR(50) COMMENT '设备IP地址',
    create_time                         VARCHAR(50) COMMENT '创建时间',
    update_time                         VARCHAR(50) COMMENT '更新时间',
    port                                INT COMMENT '信令端口',
    expires                             INT COMMENT '注册有效期',
    subscribe_cycle_for_catalog         INT DEFAULT 0 COMMENT '目录订阅周期',
    subscribe_cycle_for_mobile_position INT DEFAULT 0 COMMENT '移动位置订阅周期',
    mobile_position_submission_interval INT DEFAULT 5 COMMENT '移动位置上报间隔',
    subscribe_cycle_for_alarm           INT DEFAULT 0 COMMENT '报警订阅周期',
    host_address                        VARCHAR(50) COMMENT '设备域名/主机地址',
    charset                             VARCHAR(50) COMMENT '信令字符集',
    ssrc_check                          TINYINT(1) DEFAULT 0 COMMENT '是否校验SSRC',
    geo_coord_sys                       VARCHAR(50) COMMENT '坐标系类型',
    media_server_id                     VARCHAR(50) default 'auto' COMMENT '绑定的流媒体服务ID',
    custom_name                         VARCHAR(255) COMMENT '自定义显示名称',
    sdp_ip                              VARCHAR(50) COMMENT 'SDP中携带的IP',
    local_ip                            VARCHAR(50) COMMENT '本地局域网IP',
    password                            VARCHAR(255) COMMENT '设备鉴权密码',
    as_message_channel                  TINYINT(1) DEFAULT 0 COMMENT '是否作为消息通道',
    heart_beat_interval                 INT COMMENT '心跳间隔',
    heart_beat_count                    INT COMMENT '心跳失败次数',
    position_capability                 INT COMMENT '定位能力标识',
    broadcast_push_after_ack            TINYINT(1) DEFAULT 0 COMMENT 'ACK后是否自动推流',
    server_id                           VARCHAR(50) COMMENT '所属信令服务器ID',
    UNIQUE KEY uk_device_device (device_id)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 存储移动位置订阅上报的数据
DROP TABLE IF EXISTS zws_mobile_position;
CREATE TABLE IF NOT EXISTS zws_mobile_position
(
    id              INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    channel_id      INT NOT NULL COMMENT '通道数据库主键ID',
    timestamp       BIGINT COMMENT '上报时间',
    longitude       DOUBLE COMMENT '经度',
    latitude        DOUBLE COMMENT '纬度',
    altitude        DOUBLE COMMENT '海拔',
    speed           DOUBLE COMMENT '速度',
    direction       DOUBLE COMMENT '方向角',
    create_time     VARCHAR(50) COMMENT '入库时间'
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 保存设备下的通道信息以及扩展属性
DROP TABLE IF EXISTS zws_device_channel;
CREATE TABLE IF NOT EXISTS zws_device_channel
(
    id                           INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    device_id                    VARCHAR(50) COMMENT '所属设备ID',
    name                         VARCHAR(255) COMMENT '通道名称',
    manufacturer                 VARCHAR(50) COMMENT '厂商',
    model                        VARCHAR(50) COMMENT '型号',
    owner                        VARCHAR(50) COMMENT '归属单位',
    civil_code                   VARCHAR(50) COMMENT '行政区划代码',
    block                        VARCHAR(50) COMMENT '区域/小区编号',
    address                      VARCHAR(50) COMMENT '安装地址',
    parental                     INT COMMENT '是否有子节点',
    parent_id                    VARCHAR(50) COMMENT '父级通道ID',
    safety_way                   INT COMMENT '安全防范等级',
    register_way                 INT COMMENT '注册方式',
    cert_num                     VARCHAR(50) COMMENT '证书编号',
    certifiable                  INT COMMENT '是否可认证',
    err_code                     INT COMMENT '故障状态码',
    end_time                     VARCHAR(50) COMMENT '服务截止时间',
    secrecy                      INT COMMENT '保密级别',
    ip_address                   VARCHAR(50) COMMENT '设备IP地址',
    port                         INT COMMENT '设备端口',
    password                     VARCHAR(255) COMMENT '访问密码',
    status                       VARCHAR(50) COMMENT '在线状态',
    longitude                    DOUBLE COMMENT '经度',
    latitude                     DOUBLE COMMENT '纬度',
    ptz_type                     INT COMMENT '云台类型',
    position_type                INT COMMENT '点位类型',
    room_type                    INT COMMENT '房间类型',
    use_type                     INT COMMENT '使用性质',
    supply_light_type            INT COMMENT '补光方式',
    direction_type               INT COMMENT '朝向',
    resolution                   VARCHAR(255) COMMENT '分辨率',
    business_group_id            VARCHAR(255) COMMENT '业务分组ID',
    download_speed               VARCHAR(255) COMMENT '下载/码流速率',
    svc_space_support_mod        INT COMMENT '空域SVC能力',
    svc_time_support_mode        INT COMMENT '时域SVC能力',
    create_time                  VARCHAR(50) not null COMMENT '创建时间',
    update_time                  VARCHAR(50) not null COMMENT '更新时间',
    sub_count                    INT COMMENT '子节点数量',
    stream_id                    VARCHAR(255) COMMENT '绑定的流ID',
    has_audio                    TINYINT(1) DEFAULT 0 COMMENT '是否有音频',
    gps_time                     VARCHAR(50) COMMENT 'GPS定位时间',
    stream_identification        VARCHAR(50) COMMENT '流标识',
    channel_type                 int  default 0 not null COMMENT '通道类型',
    map_level                    int  default 0 COMMENT '地图层级',
    gb_device_id                 VARCHAR(50) COMMENT 'GB内的设备ID',
    gb_name                      VARCHAR(255) COMMENT 'GB上报的名称',
    gb_manufacturer              VARCHAR(255) COMMENT 'GB厂商',
    gb_model                     VARCHAR(255) COMMENT 'GB型号',
    gb_owner                     VARCHAR(255) COMMENT 'GB归属',
    gb_civil_code                VARCHAR(255) COMMENT 'GB行政区划',
    gb_block                     VARCHAR(255) COMMENT 'GB区域',
    gb_address                   VARCHAR(255) COMMENT 'GB地址',
    gb_parental                  INT COMMENT 'GB子节点标识',
    gb_parent_id                 VARCHAR(255) COMMENT 'GB父通道',
    gb_safety_way                INT COMMENT 'GB安全防范',
    gb_register_way              INT COMMENT 'GB注册方式',
    gb_cert_num                  VARCHAR(50) COMMENT 'GB证书编号',
    gb_certifiable               INT COMMENT 'GB认证标志',
    gb_err_code                  INT COMMENT 'GB错误码',
    gb_end_time                  VARCHAR(50) COMMENT 'GB截止时间',
    gb_secrecy                   INT COMMENT 'GB保密级别',
    gb_ip_address                VARCHAR(50) COMMENT 'GB IP',
    gb_port                      INT COMMENT 'GB端口',
    gb_password                  VARCHAR(50) COMMENT 'GB接入密码',
    gb_status                    VARCHAR(50) COMMENT 'GB状态',
    gb_longitude                 double COMMENT 'GB经度',
    gb_latitude                  double COMMENT 'GB纬度',
    gb_business_group_id         VARCHAR(50) COMMENT 'GB业务分组',
    gb_ptz_type                  INT COMMENT 'GB云台类型',
    gb_position_type             INT COMMENT 'GB点位类型',
    gb_room_type                 INT COMMENT 'GB房间类型',
    gb_use_type                  INT COMMENT 'GB用途',
    gb_supply_light_type         INT COMMENT 'GB补光',
    gb_direction_type            INT COMMENT 'GB朝向',
    gb_resolution                VARCHAR(255) COMMENT 'GB分辨率',
    gb_download_speed            VARCHAR(255) COMMENT 'GB码流速率',
    gb_svc_space_support_mod     INT COMMENT 'GB空域SVC',
    gb_svc_time_support_mode     INT COMMENT 'GB时域SVC',
    record_plan_id               INT COMMENT '绑定的录像计划ID',
    data_type                    INT not null COMMENT '数据类型标识',
    data_device_id               INT not null COMMENT '数据来源设备主键',
    gps_speed                    DOUBLE COMMENT 'GPS速度',
    gps_altitude                 DOUBLE COMMENT 'GPS海拔',
    gps_direction                DOUBLE COMMENT 'GPS方向',
    enable_broadcast             INT default 0 COMMENT '是否支持广播',
    KEY idx_data_type (data_type),
    KEY idx_data_device_id (data_device_id),
    UNIQUE KEY uk_zws_unique_channel (gb_device_id),
    UNIQUE KEY uk_device_channel_source (data_device_id, device_id)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 媒体服务器（如 ZLM）节点信息
DROP TABLE IF EXISTS zws_media_server;
CREATE TABLE IF NOT EXISTS zws_media_server
(
    id                  VARCHAR(255) PRIMARY KEY COMMENT '媒体服务器ID',
    ip                  VARCHAR(50) COMMENT '服务器IP',
    hook_ip             VARCHAR(50) COMMENT 'hook回调IP',
    sdp_ip              VARCHAR(50) COMMENT 'SDP中使用的IP',
    stream_ip           VARCHAR(50) COMMENT '推流使用的IP',
    http_port           INT COMMENT 'HTTP端口',
    http_ssl_port       INT COMMENT 'HTTPS端口',
    rtmp_port           INT COMMENT 'RTMP端口',
    rtmp_ssl_port       INT COMMENT 'RTMPS端口',
    rtp_proxy_port      INT COMMENT 'RTP代理端口',
    rtsp_port           INT COMMENT 'RTSP端口',
    rtsp_ssl_port       INT COMMENT 'RTSPS端口',
    flv_port            INT COMMENT 'FLV端口',
    flv_ssl_port        INT COMMENT 'FLV HTTPS端口',
    mp4_port            INT COMMENT 'MP4点播端口',
    mp4_ssl_port        INT COMMENT 'MP4 HTTPS端口',
    ws_flv_port         INT COMMENT 'WS-FLV端口',
    ws_flv_ssl_port     INT COMMENT 'WS-FLV HTTPS端口',
    jtt_proxy_port      INT COMMENT 'JT/T代理端口',
    auto_config         TINYINT(1) DEFAULT 0 COMMENT '是否自动配置',
    secret              VARCHAR(50) COMMENT 'ZLM校验密钥',
    type                VARCHAR(50) default 'zlm' COMMENT '节点类型',
    rtp_enable          TINYINT(1) DEFAULT 0 COMMENT '是否开启RTP',
    rtp_port_range      VARCHAR(50) COMMENT 'RTP端口范围',
    send_rtp_port_range VARCHAR(50) COMMENT '发送RTP端口范围',
    record_assist_port  INT COMMENT '录像辅助端口',
    default_server      TINYINT(1) DEFAULT 0 COMMENT '是否默认节点',
    create_time         VARCHAR(50) COMMENT '创建时间',
    update_time         VARCHAR(50) COMMENT '更新时间',
    hook_alive_interval INT COMMENT 'hook心跳间隔',
    record_path         VARCHAR(255) COMMENT '录像目录',
    record_day          INT               default 7 COMMENT '录像保留天数',
    transcode_suffix    VARCHAR(255) COMMENT '转码指令后缀',
    server_id           VARCHAR(50) COMMENT '对应信令服务器ID'
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 上级国标平台注册信息
DROP TABLE IF EXISTS zws_platform;
CREATE TABLE IF NOT EXISTS zws_platform
(
    id                    INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    enable                TINYINT(1) DEFAULT 0 COMMENT '是否启用该平台注册',
    name                  VARCHAR(255) COMMENT '平台名称',
    server_gb_id          VARCHAR(50) COMMENT '上级平台国标编码',
    server_gb_domain      VARCHAR(50) COMMENT '上级平台域编码',
    server_ip             VARCHAR(50) COMMENT '上级平台IP',
    server_port           INT COMMENT '上级平台注册端口',
    device_gb_id          VARCHAR(50) COMMENT '本平台向上注册的国标编码',
    device_ip             VARCHAR(50) COMMENT '本平台信令IP',
    device_port           VARCHAR(50) COMMENT '本平台信令端口',
    username              VARCHAR(255) COMMENT '注册用户名',
    password              VARCHAR(50) COMMENT '注册密码',
    expires               VARCHAR(50) COMMENT '注册有效期',
    keep_timeout          VARCHAR(50) COMMENT '心跳超时时间',
    transport             VARCHAR(50) COMMENT '传输协议（UDP/TCP）',
    civil_code            VARCHAR(50) COMMENT '行政区划代码',
    manufacturer          VARCHAR(255) COMMENT '厂商',
    model                 VARCHAR(255) COMMENT '型号',
    address               VARCHAR(255) COMMENT '地址',
    character_set         VARCHAR(50) COMMENT '字符集',
    ptz                   TINYINT(1) DEFAULT 0 COMMENT '是否支持PTZ',
    rtcp                  TINYINT(1) DEFAULT 0 COMMENT '是否开启RTCP',
    status                TINYINT(1) DEFAULT 0 COMMENT '注册状态',
    catalog_group         INT COMMENT '目录分组方式',
    register_way          INT COMMENT '注册方式',
    secrecy               INT COMMENT '保密级别',
    create_time           VARCHAR(50) COMMENT '创建时间',
    update_time           VARCHAR(50) COMMENT '更新时间',
    as_message_channel    TINYINT(1) DEFAULT 0 COMMENT '是否作为消息通道',
    catalog_with_platform INT default 1 COMMENT '是否推送平台目录',
    catalog_with_group    INT default 1 COMMENT '是否推送分组目录',
    catalog_with_region   INT default 1 COMMENT '是否推送区域目录',
    auto_push_channel     TINYINT(1) DEFAULT 1 COMMENT '是否自动推送通道',
    send_stream_ip        VARCHAR(50) COMMENT '推流时使用的IP',
    server_id             VARCHAR(50) COMMENT '对应信令服务器ID',
    UNIQUE KEY uk_platform_unique_server_gb_id (server_gb_id)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 国标平台下发的通道映射关系
DROP TABLE IF EXISTS zws_platform_channel;
CREATE TABLE IF NOT EXISTS zws_platform_channel
(
    id                           INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    platform_id                  INT COMMENT '平台ID',
    device_channel_id            INT COMMENT '本地通道表主键',
    custom_device_id             VARCHAR(50) COMMENT '自定义国标编码',
    custom_name                  VARCHAR(255) COMMENT '自定义名称',
    custom_manufacturer          VARCHAR(50) COMMENT '自定义厂商',
    custom_model                 VARCHAR(50) COMMENT '自定义型号',
    custom_owner                 VARCHAR(50) COMMENT '自定义归属',
    custom_civil_code            VARCHAR(50) COMMENT '自定义行政区划',
    custom_block                 VARCHAR(50) COMMENT '自定义区域',
    custom_address               VARCHAR(50) COMMENT '自定义地址',
    custom_parental              INT COMMENT '自定义父/子标识',
    custom_parent_id             VARCHAR(50) COMMENT '自定义父节点',
    custom_safety_way            INT COMMENT '自定义安全防范',
    custom_register_way          INT COMMENT '自定义注册方式',
    custom_cert_num              VARCHAR(50) COMMENT '自定义证书编号',
    custom_certifiable           INT COMMENT '自定义可认证标志',
    custom_err_code              INT COMMENT '自定义错误码',
    custom_end_time              VARCHAR(50) COMMENT '自定义截止时间',
    custom_secrecy               INT COMMENT '自定义保密级别',
    custom_ip_address            VARCHAR(50) COMMENT '自定义IP',
    custom_port                  INT COMMENT '自定义端口',
    custom_password              VARCHAR(255) COMMENT '自定义密码',
    custom_status                VARCHAR(50) COMMENT '自定义状态',
    custom_longitude             DOUBLE COMMENT '自定义经度',
    custom_latitude              DOUBLE COMMENT '自定义纬度',
    custom_ptz_type              INT COMMENT '自定义云台类型',
    custom_position_type         INT COMMENT '自定义点位类型',
    custom_room_type             INT COMMENT '自定义房间类型',
    custom_use_type              INT COMMENT '自定义用途',
    custom_supply_light_type     INT COMMENT '自定义补光',
    custom_direction_type        INT COMMENT '自定义朝向',
    custom_resolution            VARCHAR(255) COMMENT '自定义分辨率',
    custom_business_group_id     VARCHAR(255) COMMENT '自定义业务分组',
    custom_download_speed        VARCHAR(255) COMMENT '自定义码流速率',
    custom_svc_space_support_mod INT COMMENT '自定义空域SVC',
    custom_svc_time_support_mode INT COMMENT '自定义时域SVC',
    UNIQUE KEY uk_platform_gb_channel_platform_id_catalog_id_device_channel_id (platform_id, device_channel_id),
    UNIQUE KEY uk_platform_gb_channel_device_id (custom_device_id)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 平台与分组（行政区划/组织）关系
DROP TABLE IF EXISTS zws_platform_group;
CREATE TABLE IF NOT EXISTS zws_platform_group
(
    id          INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    platform_id INT COMMENT '平台ID',
    group_id    INT COMMENT '分组ID',
    UNIQUE KEY uk_zws_platform_group_platform_id_group_id (platform_id, group_id)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 平台与区域关系
DROP TABLE IF EXISTS zws_platform_region;
CREATE TABLE IF NOT EXISTS zws_platform_region
(
    id          INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    platform_id INT COMMENT '平台ID',
    region_id   INT COMMENT '区域ID',
    UNIQUE KEY uk_zws_platform_region_platform_id_group_id (platform_id, region_id)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 拉流代理/转推配置
DROP TABLE IF EXISTS zws_stream_proxy;
CREATE TABLE IF NOT EXISTS zws_stream_proxy
(
    id                         INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    type                       VARCHAR(50) COMMENT '代理类型（拉流/推流）',
    app                        VARCHAR(255) COMMENT '应用名',
    stream                     VARCHAR(255) COMMENT '流ID',
    src_url                    VARCHAR(255) COMMENT '源地址',
    timeout                    INT COMMENT '拉流超时时间',
    ffmpeg_cmd_key             VARCHAR(255) COMMENT 'FFmpeg命令模板键',
    rtsp_type                  VARCHAR(50) COMMENT 'RTSP拉流方式',
    media_server_id            VARCHAR(50) COMMENT '指定媒体服务器ID',
    enable_audio               TINYINT(1) DEFAULT 0 COMMENT '是否启用音频',
    enable_mp4                 TINYINT(1) DEFAULT 0 COMMENT '是否录制MP4',
    pulling                    TINYINT(1) DEFAULT 0 COMMENT '当前是否在拉流',
    enable                     TINYINT(1) DEFAULT 0 COMMENT '是否启用该代理',
    create_time                VARCHAR(50) COMMENT '创建时间',
    name                       VARCHAR(255) COMMENT '代理名称',
    update_time                VARCHAR(50) COMMENT '更新时间',
    stream_key                 VARCHAR(255) COMMENT '唯一流标识',
    server_id                  VARCHAR(50) COMMENT '信令服务器ID',
    enable_disable_none_reader TINYINT(1) DEFAULT 0 COMMENT '是否无人观看时自动停流',
    relates_media_server_id    VARCHAR(50) COMMENT '关联的媒体服务器ID',
    UNIQUE KEY uk_stream_proxy_app_stream (app, stream)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 推流会话记录
DROP TABLE IF EXISTS zws_stream_push;
CREATE TABLE IF NOT EXISTS zws_stream_push
(
    id                 INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    app                VARCHAR(255) COMMENT '应用名',
    stream             VARCHAR(255) COMMENT '流ID',
    create_time        VARCHAR(50) COMMENT '创建时间',
    media_server_id    VARCHAR(50) COMMENT '推流所在媒体服务器',
    server_id          VARCHAR(50) COMMENT '信令服务器ID',
    push_time          VARCHAR(50) COMMENT '推流开始时间',
    status             TINYINT(1) DEFAULT 0 COMMENT '推流状态',
    update_time        VARCHAR(50) COMMENT '更新时间',
    pushing            TINYINT(1) DEFAULT 0 COMMENT '是否正在推流',
    self               TINYINT(1) DEFAULT 0 COMMENT '是否本地发起',
    start_offline_push TINYINT(1) DEFAULT 1 COMMENT '是否离线后自动重推',
    UNIQUE KEY uk_stream_push_app_stream (app, stream)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 云端录像记录
DROP TABLE IF EXISTS zws_cloud_record;
CREATE TABLE IF NOT EXISTS zws_cloud_record
(
    id              INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    app             VARCHAR(255) COMMENT '应用名',
    stream          VARCHAR(255) COMMENT '流ID',
    call_id         VARCHAR(255) COMMENT '会话ID',
    start_time      bigint COMMENT '录像开始时间',
    end_time        bigint COMMENT '录像结束时间',
    media_server_id VARCHAR(50) COMMENT '媒体服务器ID',
    server_id       VARCHAR(50) COMMENT '信令服务器ID',
    file_name       VARCHAR(255) COMMENT '文件名',
    folder          VARCHAR(500) COMMENT '目录',
    file_path       VARCHAR(500) COMMENT '完整路径',
    collect         TINYINT(1) DEFAULT 0 COMMENT '是否收藏',
    file_size       bigint COMMENT '文件大小',
    time_len        DOUBLE COMMENT '时长'
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 平台用户信息
DROP TABLE IF EXISTS zws_user;
CREATE TABLE IF NOT EXISTS zws_user
(
    id          INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    username    VARCHAR(255) COMMENT '用户名',
    password    VARCHAR(255) COMMENT '密码（MD5）',
    role_id     INT COMMENT '角色ID',
    create_time VARCHAR(50) COMMENT '创建时间',
    update_time VARCHAR(50) COMMENT '更新时间',
    push_key    VARCHAR(50) COMMENT '推送密钥',
    UNIQUE KEY uk_user_username (username)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 用户角色信息
DROP TABLE IF EXISTS zws_user_role;
CREATE TABLE IF NOT EXISTS zws_user_role
(
    id          INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    name        VARCHAR(50) COMMENT '角色名称',
    authority   VARCHAR(50) COMMENT '权限标识',
    create_time VARCHAR(50) COMMENT '创建时间',
    update_time VARCHAR(50) COMMENT '更新时间'
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


DROP TABLE IF EXISTS zws_user_api_key;
CREATE TABLE IF NOT EXISTS zws_user_api_key
(
    id          INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    user_id     bigint COMMENT '关联用户ID',
    app         VARCHAR(255) COMMENT '应用标识',
    api_key     text COMMENT 'API Key',
    expired_at  bigint COMMENT '过期时间戳',
    remark      VARCHAR(255) COMMENT '备注',
    enable      TINYINT(1) DEFAULT 1 COMMENT '是否启用',
    create_time VARCHAR(50) COMMENT '创建时间',
    update_time VARCHAR(50) COMMENT '更新时间'
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- 初始数据
-- 初始化管理员账号，账号admin 密码admin（MD5加密后）
INSERT INTO zws_user
VALUES (1, 'admin', '21232f297a57a5a743894a0e4a801fc3', 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57',
        '3e80d1762a324d5b0ff636e0bd16f1e3');
-- 初始化管理员角色
INSERT INTO zws_user_role
VALUES (1, 'admin', '0', '2021-04-13 14:14:57', '2021-04-13 14:14:57');

-- 通用分组表，存储行业或组织结构
DROP TABLE IF EXISTS zws_common_group;
CREATE TABLE IF NOT EXISTS zws_common_group
(
    id               INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    device_id        varchar(50)  NOT NULL COMMENT '分组对应的平台或设备ID',
    name             varchar(255) NOT NULL COMMENT '分组名称',
    parent_id        int COMMENT '父级分组ID',
    parent_device_id varchar(50) DEFAULT NULL COMMENT '父级分组对应的设备ID',
    business_group   varchar(50)  NOT NULL COMMENT '业务分组编码',
    create_time      varchar(50)  NOT NULL COMMENT '创建时间',
    update_time      varchar(50)  NOT NULL COMMENT '更新时间',
    civil_code       varchar(50) default null COMMENT '行政区划代码',
    alias            varchar(255) default null COMMENT '别名',
    UNIQUE KEY uk_common_group_device_platform (device_id)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 通用行政区域表
DROP TABLE IF EXISTS zws_common_region;
CREATE TABLE IF NOT EXISTS zws_common_region
(
    id               INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    device_id        varchar(50)  NOT NULL COMMENT '区域对应的平台或设备ID',
    name             varchar(255) NOT NULL COMMENT '区域名称',
    parent_id        int COMMENT '父级区域ID',
    parent_device_id varchar(50) DEFAULT NULL COMMENT '父级区域的设备ID',
    create_time      varchar(50)  NOT NULL COMMENT '创建时间',
    update_time      varchar(50)  NOT NULL COMMENT '更新时间',
    UNIQUE KEY uk_common_region_device_id (device_id)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 录像计划基础信息
DROP TABLE IF EXISTS zws_record_plan;
CREATE TABLE IF NOT EXISTS zws_record_plan
(
    id              INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    snap            TINYINT(1) DEFAULT 0 COMMENT '是否抓图计划',
    name            varchar(255) NOT NULL COMMENT '计划名称',
    create_time     VARCHAR(50) COMMENT '创建时间',
    update_time     VARCHAR(50) COMMENT '更新时间'
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 录像计划条目表
DROP TABLE IF EXISTS zws_record_plan_item;
CREATE TABLE IF NOT EXISTS zws_record_plan_item
(
    id              INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    start           int COMMENT '开始时间（分钟）',
    stop            int COMMENT '结束时间（分钟）',
    week_day        int COMMENT '星期（0-6）',
    plan_id         int COMMENT '所属录像计划ID',
    create_time     VARCHAR(50) COMMENT '创建时间',
    update_time     VARCHAR(50) COMMENT '更新时间'
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 交通部 JT/T 1076 终端信息
DROP TABLE IF EXISTS zws_jt_terminal;
CREATE TABLE IF NOT EXISTS zws_jt_terminal (
                                 id INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
                                 phone_number VARCHAR(50) COMMENT '终端SIM卡号',
                                 terminal_id VARCHAR(50) COMMENT '终端设备ID',
                                 province_id VARCHAR(50) COMMENT '所在省份ID',
                                 province_text VARCHAR(100) COMMENT '所在省份名称',
                                 city_id VARCHAR(50) COMMENT '所在城市ID',
                                 city_text VARCHAR(100) COMMENT '所在城市名称',
                                 maker_id VARCHAR(50) COMMENT '厂商ID',
                                 model VARCHAR(50) COMMENT '终端型号',
                                 plate_color VARCHAR(50) COMMENT '车牌颜色',
                                 plate_no VARCHAR(50) COMMENT '车牌号码',
                                 longitude DOUBLE COMMENT '经度',
                                 latitude DOUBLE COMMENT '纬度',
                                 status TINYINT(1) DEFAULT 0 COMMENT '在线状态',
                                 register_time VARCHAR(50) default null COMMENT '注册时间',
                                 update_time VARCHAR(50) not null COMMENT '更新时间',
                                 create_time VARCHAR(50) not null COMMENT '创建时间',
                                 geo_coord_sys VARCHAR(50) COMMENT '坐标系',
                                 media_server_id VARCHAR(50) default 'auto' COMMENT '媒体服务器ID',
                                 sdp_ip VARCHAR(50) COMMENT 'SDP IP',
                                 UNIQUE KEY uk_jt_device_id_device_id (id, phone_number)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 交通部 JT/T 1076 通道信息
DROP TABLE IF EXISTS zws_jt_channel;
CREATE TABLE IF NOT EXISTS zws_jt_channel (
                               id INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
                               terminal_db_id INT COMMENT '所属终端记录ID',
                               channel_id INT COMMENT '通道号',
                               has_audio TINYINT(1) DEFAULT 0 COMMENT '是否有音频',
                               name VARCHAR(255) COMMENT '通道名称',
                               update_time VARCHAR(50) not null COMMENT '更新时间',
                               create_time VARCHAR(50) not null COMMENT '创建时间',
                               UNIQUE KEY uk_jt_channel_id_device_id (terminal_db_id, channel_id)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 报警信息表，表结构参考alarm类
DROP TABLE IF EXISTS zws_alarm;
CREATE TABLE IF NOT EXISTS zws_alarm (
                          id INT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
                          channel_id INT COMMENT '关联通道的数据库id',
                          description VARCHAR(255) COMMENT '报警描述',
                          snap_path VARCHAR(255) COMMENT '报警快照路径',
                          record_path VARCHAR(255) COMMENT '报警录像路径',
                          longitude DOUBLE COMMENT '报警附带的经度',
                          latitude DOUBLE COMMENT '报警附带的纬度',
                          alarm_type INT COMMENT '报警类别',
                          alarm_time bigint COMMENT '报警时间'
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SET FOREIGN_KEY_CHECKS = 1;
