# zero-web-server MySQL 8 schema（全量建表，唯一权威）
-- 设计权威文档：docs/数据模型ER.md
-- 开发期：空库执行本脚本即可；表结构变更只改本文件，不维护增量合并 SQL。
-- 设计原则：BIGINT 自增主键；稳定/易变分表（*_ext）；真实外键 + CASCADE；
--           internal_code / gb_code 业务唯一；长尾字段进 JSON。
-- 时间沿用 VARCHAR(50)，与全库 Go 字段/JSON 契约一致（改 DATETIME 属全局改动，单独立项）。

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ===========================================================================
-- A. 物模型：设备 / 接入绑定 / 通道 / 扩展 / 码流 / 能力
-- ===========================================================================

DROP TABLE IF EXISTS zws_device;
CREATE TABLE IF NOT EXISTS zws_device
(
    id             BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    internal_code  VARCHAR(32)  NOT NULL COMMENT '平台内码 D…（对外身份/流名/事件）',
    gb_code        VARCHAR(20)  DEFAULT NULL COMMENT '国标设备码（国标接入/级联；纯本地其它协议可空）',
    name           VARCHAR(255) NOT NULL DEFAULT '',
    manufacturer   VARCHAR(255),
    model          VARCHAR(255),
    firmware       VARCHAR(255),
    serial_number  VARCHAR(255),
    online         TINYINT(1) NOT NULL DEFAULT 0 COMMENT '运行时在线',
    enabled        TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用（停用不参与调度）',
    status         VARCHAR(32)  NOT NULL DEFAULT 'OFF' COMMENT '注册状态',
    create_time    VARCHAR(50)  NOT NULL DEFAULT '',
    update_time    VARCHAR(50)  NOT NULL DEFAULT '',
    UNIQUE KEY uk_device_internal_code (internal_code),
    UNIQUE KEY uk_device_gb_code ((NULLIF(gb_code, '')))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='设备（物理资产，稳定信息）';

DROP TABLE IF EXISTS zws_access_binding;
CREATE TABLE IF NOT EXISTS zws_access_binding
(
    id                 BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    device_id          BIGINT NOT NULL COMMENT '所属设备',
    protocol           VARCHAR(16) NOT NULL COMMENT 'gb28181 | onvif | plugin',
    mode               VARCHAR(16) NOT NULL DEFAULT '' COMMENT 'passive | active',
    external_id        VARCHAR(255) NOT NULL DEFAULT '' COMMENT '协议侧设备号（GB码/ONVIF地址/插件uid）',
    endpoint           VARCHAR(255) DEFAULT NULL COMMENT 'ip:port',
    plugin_instance_id VARCHAR(128) DEFAULT NULL,
    config_json        LONGTEXT COMMENT '协议凭据/参数',
    status             VARCHAR(32) NOT NULL DEFAULT '' COMMENT '接入状态',
    last_seen          BIGINT NOT NULL DEFAULT 0 COMMENT '最近心跳(unix ms)',
    create_time        VARCHAR(50) NOT NULL DEFAULT '',
    update_time        VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_binding_device (device_id),
    UNIQUE KEY uk_binding_external (protocol, external_id),
    CONSTRAINT fk_binding_device FOREIGN KEY (device_id) REFERENCES zws_device (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='设备接入绑定(1:1)';

DROP TABLE IF EXISTS zws_channel;
CREATE TABLE IF NOT EXISTS zws_channel
(
    id            BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    device_id     BIGINT NOT NULL COMMENT '所属设备',
    internal_code VARCHAR(32) NOT NULL COMMENT '平台内码 C…（对外身份/流名/事件）',
    gb_code       VARCHAR(20) DEFAULT NULL COMMENT '国标通道码（国标接入；其它协议可空）',
    external_key  VARCHAR(255) NOT NULL DEFAULT '' COMMENT '协议侧通道标识（GB通道码/ONVIF token/插件通道号）',
    kind          VARCHAR(32) NOT NULL DEFAULT 'video_sensor' COMMENT 'video_sensor|audio_input|alarm_input|metadata_source',
    name          VARCHAR(255) NOT NULL DEFAULT '',
    online        TINYINT(1) NOT NULL DEFAULT 0,
    status        VARCHAR(32) NOT NULL DEFAULT 'OFF',
    create_time   VARCHAR(50) NOT NULL DEFAULT '',
    update_time   VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_channel_internal_code (internal_code),
    UNIQUE KEY uk_channel_gb_code ((NULLIF(gb_code, ''))),
    KEY idx_channel_device (device_id),
    CONSTRAINT fk_channel_device FOREIGN KEY (device_id) REFERENCES zws_device (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通道（业务唯一事实来源）';

DROP TABLE IF EXISTS zws_device_ext;
CREATE TABLE IF NOT EXISTS zws_device_ext
(
    id            BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    device_id     BIGINT NOT NULL,
    longitude     DECIMAL(10, 7) DEFAULT NULL,
    latitude      DECIMAL(10, 7) DEFAULT NULL,
    address       VARCHAR(255) DEFAULT NULL,
    geo_coord_sys VARCHAR(50) DEFAULT NULL,
    remark        VARCHAR(255) DEFAULT NULL,
    tags          VARCHAR(255) DEFAULT NULL,
    extra_json    LONGTEXT,
    create_time   VARCHAR(50) NOT NULL DEFAULT '',
    update_time   VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_device_ext_device (device_id),
    CONSTRAINT fk_device_ext_device FOREIGN KEY (device_id) REFERENCES zws_device (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='设备扩展信息（易变）';

DROP TABLE IF EXISTS zws_channel_ext;
CREATE TABLE IF NOT EXISTS zws_channel_ext
(
    id             BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    channel_id     BIGINT NOT NULL,
    longitude      DECIMAL(10, 7) DEFAULT NULL,
    latitude       DECIMAL(10, 7) DEFAULT NULL,
    map_level      INT NOT NULL DEFAULT 0,
    ptz_type       INT NOT NULL DEFAULT 0,
    has_audio      TINYINT(1) NOT NULL DEFAULT 0,
    enabled        TINYINT(1) NOT NULL DEFAULT 1,
    tags           VARCHAR(255) DEFAULT NULL,
    record_plan_id BIGINT DEFAULT NULL COMMENT '绑定的录像计划',
    gb_json        LONGTEXT COMMENT '国标原始属性（原 gb_* 列存档）',
    extra_json     LONGTEXT COMMENT '协议长尾（profileToken/resolution/codec…）',
    create_time    VARCHAR(50) NOT NULL DEFAULT '',
    update_time    VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_channel_ext_channel (channel_id),
    KEY idx_channel_ext_geo (longitude, latitude),
    KEY idx_channel_ext_plan (record_plan_id),
    CONSTRAINT fk_channel_ext_channel FOREIGN KEY (channel_id) REFERENCES zws_channel (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通道扩展信息（易变：经纬度/PTZ/标签/录像计划）';

DROP TABLE IF EXISTS zws_stream_profile;
CREATE TABLE IF NOT EXISTS zws_stream_profile
(
    id            BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    channel_id    BIGINT NOT NULL,
    role          VARCHAR(16) NOT NULL DEFAULT 'main' COMMENT 'main|sub|third',
    name          VARCHAR(128) NOT NULL DEFAULT '',
    selector_json TEXT,
    codec         VARCHAR(64) DEFAULT NULL,
    resolution    VARCHAR(64) DEFAULT NULL,
    is_default    TINYINT(1) NOT NULL DEFAULT 0,
    create_time   VARCHAR(50) NOT NULL DEFAULT '',
    update_time   VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_profile_channel_role (channel_id, role),
    CONSTRAINT fk_profile_channel FOREIGN KEY (channel_id) REFERENCES zws_channel (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频通道码流';

DROP TABLE IF EXISTS zws_capability;
CREATE TABLE IF NOT EXISTS zws_capability
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    channel_id  BIGINT NOT NULL,
    operation   VARCHAR(64) NOT NULL COMMENT 'play|playback|ptz|record|snapshot|event',
    supported   TINYINT(1) NOT NULL DEFAULT 1 COMMENT '驱动声明支持',
    configured  TINYINT(1) NOT NULL DEFAULT 1 COMMENT '平台已配置',
    available   TINYINT(1) NOT NULL DEFAULT 0 COMMENT '当前可用',
    params_json TEXT,
    source      VARCHAR(32) DEFAULT NULL,
    update_time VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_cap_channel_op (channel_id, operation),
    CONSTRAINT fk_cap_channel FOREIGN KEY (channel_id) REFERENCES zws_channel (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通道能力集合（三态制约操作）';

-- ===========================================================================
-- B. 目录树：行政区划 / 业务分组 / 多挂载
-- ===========================================================================

DROP TABLE IF EXISTS zws_region;
CREATE TABLE IF NOT EXISTS zws_region
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    parent_id   BIGINT DEFAULT NULL,
    code        VARCHAR(20) NOT NULL COMMENT '区划码（国标 215 节点码/行政区划码）',
    name        VARCHAR(255) NOT NULL DEFAULT '',
    alias       VARCHAR(255) DEFAULT NULL COMMENT '别名',
    sort        INT NOT NULL DEFAULT 0,
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_region_code (code),
    KEY idx_region_parent (parent_id),
    CONSTRAINT fk_region_parent FOREIGN KEY (parent_id) REFERENCES zws_region (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='行政区划目录';

DROP TABLE IF EXISTS zws_group;
CREATE TABLE IF NOT EXISTS zws_group
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    parent_id   BIGINT DEFAULT NULL,
    code        VARCHAR(20) NOT NULL COMMENT '业务分组码（国标 215/216 节点码）',
    name        VARCHAR(255) NOT NULL DEFAULT '',
    business_group VARCHAR(255) DEFAULT NULL COMMENT '所属业务分组码（解析用）',
    civil_code  VARCHAR(255) DEFAULT NULL COMMENT '关联区划码',
    alias       VARCHAR(255) DEFAULT NULL COMMENT '别名',
    sort        INT NOT NULL DEFAULT 0,
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_group_code (code),
    KEY idx_group_parent (parent_id),
    CONSTRAINT fk_group_parent FOREIGN KEY (parent_id) REFERENCES zws_group (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='业务分组目录';

-- 目录挂载：一通道/设备可挂多个区划/分组节点；靠 (节点,对象) 唯一防重复。
DROP TABLE IF EXISTS zws_catalog;
CREATE TABLE IF NOT EXISTS zws_catalog
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    region_id   BIGINT DEFAULT NULL COMMENT '挂在行政区划节点',
    group_id    BIGINT DEFAULT NULL COMMENT '挂在业务分组节点',
    target_type VARCHAR(16) NOT NULL COMMENT 'device | channel',
    target_id   BIGINT NOT NULL COMMENT '设备/通道主键',
    sort        INT NOT NULL DEFAULT 0,
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_catalog_region (region_id, target_type, target_id),
    UNIQUE KEY uk_catalog_group (group_id, target_type, target_id),
    KEY idx_catalog_target (target_type, target_id),
    CONSTRAINT fk_catalog_region FOREIGN KEY (region_id) REFERENCES zws_region (id) ON DELETE CASCADE,
    CONSTRAINT fk_catalog_group FOREIGN KEY (group_id) REFERENCES zws_group (id) ON DELETE CASCADE,
    CONSTRAINT ck_catalog_one_parent CHECK ((region_id IS NULL) <> (group_id IS NULL))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='目录挂载（多对多，可多挂）';

-- ===========================================================================
-- C. 事件
-- ===========================================================================

DROP TABLE IF EXISTS zws_event_type;
CREATE TABLE IF NOT EXISTS zws_event_type
(
    code            VARCHAR(64) NOT NULL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL DEFAULT '',
    category        VARCHAR(32) DEFAULT NULL,
    severity        VARCHAR(32) DEFAULT NULL,
    alarm_eligible  TINYINT(1) NOT NULL DEFAULT 0,
    requires_video  TINYINT(1) NOT NULL DEFAULT 0,
    bbox_capable    TINYINT(1) NOT NULL DEFAULT 0,
    vendor_mappings TEXT,
    enabled         TINYINT(1) NOT NULL DEFAULT 1,
    update_time     VARCHAR(50) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事件类型注册表';

DROP TABLE IF EXISTS zws_event;
CREATE TABLE IF NOT EXISTS zws_event
(
    id                   BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    event_id             VARCHAR(64) NOT NULL COMMENT '外部事件号（幂等去重）',
    device_id            BIGINT NOT NULL,
    channel_id           BIGINT DEFAULT NULL COMMENT '可空；通道关联即间接关联设备',
    event_type           VARCHAR(64) NOT NULL DEFAULT '' COMMENT '指向 zws_event_type.code；未注册即自定义',
    severity             VARCHAR(16) NOT NULL DEFAULT '',
    status               VARCHAR(16) NOT NULL DEFAULT '',
    occurred_at          BIGINT NOT NULL DEFAULT 0 COMMENT '发生时间(unix ms)',
    received_at          BIGINT NOT NULL DEFAULT 0 COMMENT '接收时间(unix ms)',
    device_internal_code VARCHAR(64) NOT NULL DEFAULT '' COMMENT '冗余便于检索',
    channel_internal_code VARCHAR(64) NOT NULL DEFAULT '',
    bbox_json            TEXT,
    image_url            VARCHAR(1024) DEFAULT NULL,
    raw_json             LONGTEXT,
    create_time          VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_event_id (event_id),
    KEY idx_event_device_time (device_id, occurred_at),
    KEY idx_event_channel_time (channel_id, occurred_at),
    KEY idx_event_type_time (event_type, occurred_at),
    CONSTRAINT fk_event_device FOREIGN KEY (device_id) REFERENCES zws_device (id) ON DELETE CASCADE,
    CONSTRAINT fk_event_channel FOREIGN KEY (channel_id) REFERENCES zws_channel (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事件（统一信封）';

DROP TABLE IF EXISTS zws_alarm;
CREATE TABLE IF NOT EXISTS zws_alarm
(
    id           BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    event_id     VARCHAR(64) NOT NULL COMMENT '关联事件',
    channel_id   BIGINT DEFAULT NULL COMMENT '关联通道',
    description  VARCHAR(255) DEFAULT NULL,
    alarm_type   INT NOT NULL DEFAULT 0 COMMENT '报警大类（旧）',
    alarm_code   VARCHAR(64) DEFAULT NULL COMMENT '事件类型码',
    alarm_time   BIGINT NOT NULL DEFAULT 0 COMMENT '发生时间(unix ms)',
    device_internal_code  VARCHAR(64) NOT NULL DEFAULT '',
    channel_internal_code VARCHAR(64) NOT NULL DEFAULT '',
    longitude    DOUBLE DEFAULT NULL,
    latitude     DOUBLE DEFAULT NULL,
    bbox_json    TEXT,
    status       VARCHAR(32) NOT NULL DEFAULT 'unhandled' COMMENT '处置状态',
    severity     VARCHAR(32) DEFAULT NULL,
    handed_by    VARCHAR(128) DEFAULT NULL,
    handled_at   BIGINT DEFAULT NULL,
    snap_path    VARCHAR(255) DEFAULT NULL,
    record_path  VARCHAR(255) DEFAULT NULL,
    remark       VARCHAR(255) DEFAULT NULL,
    create_time  VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_alarm_event (event_id),
    KEY idx_alarm_time (alarm_time),
    KEY idx_alarm_type (alarm_type),
    CONSTRAINT fk_alarm_event FOREIGN KEY (event_id) REFERENCES zws_event (event_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='报警（处置投影 + 查询字段，1:1 事件）';

DROP TABLE IF EXISTS zws_event_source;
CREATE TABLE IF NOT EXISTS zws_event_source
(
    id           BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    device_id    BIGINT NOT NULL,
    channel_id   BIGINT DEFAULT NULL,
    kind         VARCHAR(32) DEFAULT NULL,
    external_key VARCHAR(255) DEFAULT NULL,
    enabled      TINYINT(1) NOT NULL DEFAULT 0,
    config_json  TEXT,
    CONSTRAINT fk_evsrc_device FOREIGN KEY (device_id) REFERENCES zws_device (id) ON DELETE CASCADE,
    CONSTRAINT fk_evsrc_channel FOREIGN KEY (channel_id) REFERENCES zws_channel (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事件来源';

DROP TABLE IF EXISTS zws_event_subscription;
CREATE TABLE IF NOT EXISTS zws_event_subscription
(
    id                   BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    device_id            BIGINT NOT NULL,
    channel_id           BIGINT DEFAULT NULL,
    device_internal_code VARCHAR(64) NOT NULL DEFAULT '',
    plugin_instance_id   VARCHAR(128) NOT NULL DEFAULT '' COMMENT '插件实例ID（插件托管设备订阅用）',
    device_uid           VARCHAR(255) NOT NULL DEFAULT '' COMMENT '插件侧设备唯一号',
    event_types          TEXT,
    armed                TINYINT(1) NOT NULL DEFAULT 0,
    status               VARCHAR(16) DEFAULT NULL,
    last_error           VARCHAR(500) DEFAULT NULL,
    remote_id            VARCHAR(128) DEFAULT NULL,
    last_armed_at        BIGINT NOT NULL DEFAULT 0,
    create_time          VARCHAR(50) NOT NULL DEFAULT '',
    update_time          VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_event_sub (device_id, channel_id),
    CONSTRAINT fk_evsub_device FOREIGN KEY (device_id) REFERENCES zws_device (id) ON DELETE CASCADE,
    CONSTRAINT fk_evsub_channel FOREIGN KEY (channel_id) REFERENCES zws_channel (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事件订阅/布防';

-- 自定义事件：平台不理解语义，仅按基础信封 + raw 原样保存（与统一事件隔离）。
DROP TABLE IF EXISTS zws_custom_event;
CREATE TABLE IF NOT EXISTS zws_custom_event
(
    id                    BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    event_id              VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '外部事件号（幂等）',
    device_internal_code  VARCHAR(64)  NOT NULL DEFAULT '',
    channel_internal_code VARCHAR(64)  NOT NULL DEFAULT '',
    device_id             VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '上报设备标识（插件侧）',
    resource_id           VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '上报通道标识（插件侧）',
    major_type            VARCHAR(64)  NOT NULL DEFAULT '',
    minor_type            VARCHAR(64)  NOT NULL DEFAULT '',
    occur_at              BIGINT       NOT NULL DEFAULT 0,
    level                 VARCHAR(32)  DEFAULT NULL,
    image_key             VARCHAR(255) DEFAULT NULL,
    image_url             VARCHAR(1024) DEFAULT NULL,
    raw_json              LONGTEXT,
    create_time           VARCHAR(50)  NOT NULL DEFAULT '',
    KEY idx_cevent_device (device_internal_code, occur_at),
    KEY idx_cevent_event (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='自定义事件（raw 原样保存）';

-- ===========================================================================
-- D. 媒体 / 录像 / 位置
-- ===========================================================================

DROP TABLE IF EXISTS zws_media_server;
CREATE TABLE IF NOT EXISTS zws_media_server
(
    id                    VARCHAR(255) NOT NULL PRIMARY KEY COMMENT '节点ID',
    name                  VARCHAR(128) NOT NULL DEFAULT '',
    ip                    VARCHAR(50) DEFAULT NULL,
    hook_ip               VARCHAR(50) DEFAULT NULL,
    sdp_ip                VARCHAR(50) DEFAULT NULL,
    stream_ip             VARCHAR(50) DEFAULT NULL,
    http_port             INT DEFAULT NULL,
    http_ssl_port         INT DEFAULT NULL,
    rtmp_port             INT DEFAULT NULL,
    rtmp_ssl_port         INT DEFAULT NULL,
    rtp_proxy_port        INT DEFAULT NULL,
    rtsp_port             INT DEFAULT NULL,
    rtsp_ssl_port         INT DEFAULT NULL,
    flv_port              INT DEFAULT NULL,
    flv_ssl_port          INT DEFAULT NULL,
    mp4_port              INT DEFAULT NULL,
    mp4_ssl_port          INT DEFAULT NULL,
    ws_flv_port           INT DEFAULT NULL,
    ws_flv_ssl_port       INT DEFAULT NULL,
    jtt_proxy_port        INT DEFAULT NULL,
    srt_port              INT DEFAULT NULL,
    webrtc_port_min       INT DEFAULT NULL,
    webrtc_port_max       INT DEFAULT NULL,
    rtp_enable            TINYINT(1) NOT NULL DEFAULT 0,
    rtp_port_range        VARCHAR(50) DEFAULT NULL,
    send_rtp_port_range   VARCHAR(50) DEFAULT NULL,
    enable_rtsp           TINYINT(1) NOT NULL DEFAULT 0,
    enable_rtmp           TINYINT(1) NOT NULL DEFAULT 0,
    enable_hls            TINYINT(1) NOT NULL DEFAULT 0,
    enable_srt            TINYINT(1) NOT NULL DEFAULT 0,
    enable_webrtc         TINYINT(1) NOT NULL DEFAULT 0,
    secret                VARCHAR(50) DEFAULT NULL COMMENT 'API校验密钥',
    type                  VARCHAR(32) NOT NULL DEFAULT 'zms',
    default_server        TINYINT(1) NOT NULL DEFAULT 0,
    auto_config           TINYINT(1) NOT NULL DEFAULT 0,
    capabilities_reported TINYINT(1) NOT NULL DEFAULT 0,
    record_path           VARCHAR(255) DEFAULT NULL,
    record_day            INT NOT NULL DEFAULT 7,
    transcode_suffix      VARCHAR(255) DEFAULT NULL,
    hook_alive_interval   INT DEFAULT NULL,
    version               VARCHAR(50) DEFAULT NULL,
    server_id             VARCHAR(50) DEFAULT NULL,
    last_seen             BIGINT NOT NULL DEFAULT 0,
    create_time           VARCHAR(50) NOT NULL DEFAULT '',
    update_time           VARCHAR(50) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='媒体服务器节点';

DROP TABLE IF EXISTS zws_record_plan;
CREATE TABLE IF NOT EXISTS zws_record_plan
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name        VARCHAR(255) NOT NULL DEFAULT '',
    snap        TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否抓图计划',
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='录像计划';

DROP TABLE IF EXISTS zws_record_plan_item;
CREATE TABLE IF NOT EXISTS zws_record_plan_item
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    plan_id     BIGINT NOT NULL,
    week_day    INT NOT NULL DEFAULT 0 COMMENT '0-6',
    start_min   INT NOT NULL DEFAULT 0 COMMENT '开始（当天分钟）',
    stop_min    INT NOT NULL DEFAULT 0 COMMENT '结束（当天分钟）',
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT '',
    KEY idx_plan_item_plan (plan_id),
    CONSTRAINT fk_plan_item_plan FOREIGN KEY (plan_id) REFERENCES zws_record_plan (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='录像计划时段';

DROP TABLE IF EXISTS zws_cloud_record;
CREATE TABLE IF NOT EXISTS zws_cloud_record
(
    id              BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    channel_id      BIGINT DEFAULT NULL COMMENT '关联通道（按流名回填）',
    app             VARCHAR(64) NOT NULL DEFAULT '',
    stream          VARCHAR(255) NOT NULL DEFAULT '',
    call_id         VARCHAR(255) DEFAULT NULL,
    media_server_id VARCHAR(255) DEFAULT NULL,
    start_time      BIGINT DEFAULT 0,
    end_time        BIGINT DEFAULT 0,
    file_name       VARCHAR(255) DEFAULT NULL,
    folder          VARCHAR(255) DEFAULT NULL,
    file_path       VARCHAR(512) DEFAULT NULL,
    file_size       BIGINT DEFAULT 0,
    time_len        DOUBLE DEFAULT 0,
    play_url        VARCHAR(512) DEFAULT NULL,
    server_id       VARCHAR(50) DEFAULT NULL,
    collect         TINYINT(1) NOT NULL DEFAULT 0,
    KEY idx_cloud_channel_time (channel_id, start_time),
    KEY idx_cloud_app_stream (app, stream),
    CONSTRAINT fk_cloud_channel FOREIGN KEY (channel_id) REFERENCES zws_channel (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='云端录像';

DROP TABLE IF EXISTS zws_stream_proxy;
CREATE TABLE IF NOT EXISTS zws_stream_proxy
(
    id             BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name           VARCHAR(255) DEFAULT NULL,
    type           VARCHAR(50) DEFAULT NULL,
    app            VARCHAR(64) NOT NULL DEFAULT '',
    stream         VARCHAR(255) NOT NULL DEFAULT '',
    src_url        VARCHAR(1024) DEFAULT NULL,
    timeout        INT DEFAULT NULL,
    ffmpeg_cmd_key VARCHAR(255) DEFAULT NULL,
    rtsp_type      VARCHAR(50) DEFAULT NULL,
    media_server_id VARCHAR(255) DEFAULT NULL,
    enable_audio   TINYINT(1) NOT NULL DEFAULT 0,
    enable_mp4     TINYINT(1) NOT NULL DEFAULT 0,
    enable         TINYINT(1) NOT NULL DEFAULT 0,
    pulling        TINYINT(1) NOT NULL DEFAULT 0,
    stream_key     VARCHAR(255) DEFAULT NULL,
    server_id      VARCHAR(50) DEFAULT NULL,
    enable_disable_none_reader TINYINT(1) NOT NULL DEFAULT 0,
    relates_media_server_id VARCHAR(255) DEFAULT NULL,
    gb_code        VARCHAR(50) DEFAULT NULL COMMENT '对外国标通道码（可选）',
    gb_name        VARCHAR(255) DEFAULT NULL COMMENT '对外国标通道名（可选）',
    create_time    VARCHAR(50) NOT NULL DEFAULT '',
    update_time    VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_proxy_app_stream (app, stream)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='拉流代理';

DROP TABLE IF EXISTS zws_stream_push;
CREATE TABLE IF NOT EXISTS zws_stream_push
(
    id              BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    app             VARCHAR(64) NOT NULL DEFAULT '',
    stream          VARCHAR(255) NOT NULL DEFAULT '',
    media_server_id VARCHAR(255) DEFAULT NULL,
    status          TINYINT(1) NOT NULL DEFAULT 0,
    pushing         TINYINT(1) NOT NULL DEFAULT 0,
    self            TINYINT(1) NOT NULL DEFAULT 0,
    push_time       VARCHAR(50) DEFAULT NULL,
    server_id       VARCHAR(50) DEFAULT NULL,
    start_offline_push TINYINT(1) NOT NULL DEFAULT 0,
    gb_code         VARCHAR(50) DEFAULT NULL COMMENT '对外国标通道码（可选）',
    gb_name         VARCHAR(255) DEFAULT NULL COMMENT '对外国标通道名（可选）',
    create_time     VARCHAR(50) NOT NULL DEFAULT '',
    update_time     VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_push_app_stream (app, stream)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='推流会话';

DROP TABLE IF EXISTS zws_mobile_position;
CREATE TABLE IF NOT EXISTS zws_mobile_position
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    channel_id  BIGINT NOT NULL,
    timestamp   BIGINT DEFAULT NULL,
    longitude   DOUBLE DEFAULT NULL,
    latitude    DOUBLE DEFAULT NULL,
    altitude    DOUBLE DEFAULT NULL,
    speed       DOUBLE DEFAULT NULL,
    direction   DOUBLE DEFAULT NULL,
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    KEY idx_pos_channel_time (channel_id, timestamp),
    CONSTRAINT fk_pos_channel FOREIGN KEY (channel_id) REFERENCES zws_channel (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='移动位置上报';

-- ===========================================================================
-- E. 级联
-- ===========================================================================

DROP TABLE IF EXISTS zws_upper_platform;
CREATE TABLE IF NOT EXISTS zws_upper_platform
(
    id                    BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    enable                TINYINT(1) NOT NULL DEFAULT 0,
    name                  VARCHAR(255) DEFAULT NULL,
    server_gb_id          VARCHAR(50) NOT NULL COMMENT '上级平台国标编码',
    server_gb_domain      VARCHAR(50) DEFAULT NULL,
    server_ip             VARCHAR(50) DEFAULT NULL,
    server_port           INT DEFAULT NULL,
    device_gb_id          VARCHAR(50) DEFAULT NULL COMMENT '本级向上注册的国标编码',
    device_ip             VARCHAR(50) DEFAULT NULL,
    device_port           VARCHAR(50) DEFAULT NULL,
    username              VARCHAR(255) DEFAULT NULL,
    password              VARCHAR(255) DEFAULT NULL,
    expires               VARCHAR(50) DEFAULT NULL,
    keep_timeout          VARCHAR(50) DEFAULT NULL,
    transport             VARCHAR(50) DEFAULT NULL,
    civil_code            VARCHAR(50) DEFAULT NULL,
    manufacturer          VARCHAR(255) DEFAULT NULL,
    model                 VARCHAR(255) DEFAULT NULL,
    address               VARCHAR(255) DEFAULT NULL,
    character_set         VARCHAR(50) DEFAULT NULL,
    ptz                   TINYINT(1) NOT NULL DEFAULT 0,
    rtcp                  TINYINT(1) NOT NULL DEFAULT 0,
    status                TINYINT(1) NOT NULL DEFAULT 0,
    catalog_group         INT DEFAULT NULL,
    register_way          INT DEFAULT NULL,
    secrecy               INT DEFAULT NULL,
    as_message_channel    TINYINT(1) NOT NULL DEFAULT 0,
    catalog_with_platform INT NOT NULL DEFAULT 1,
    catalog_with_group    INT NOT NULL DEFAULT 1,
    catalog_with_region   INT NOT NULL DEFAULT 1,
    auto_push_channel     TINYINT(1) NOT NULL DEFAULT 1,
    send_stream_ip        VARCHAR(50) DEFAULT NULL,
    server_id             VARCHAR(50) DEFAULT NULL,
    create_time           VARCHAR(50) NOT NULL DEFAULT '',
    update_time           VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_upper_server_gb (server_gb_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='上级国标平台';

DROP TABLE IF EXISTS zws_subordinate_platform;
CREATE TABLE IF NOT EXISTS zws_subordinate_platform
(
    id               BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    enable           TINYINT(1) NOT NULL DEFAULT 1,
    name             VARCHAR(255) DEFAULT NULL,
    device_gb_id     VARCHAR(50) NOT NULL COMMENT '下级向上注册的国标编号',
    password         VARCHAR(64) NOT NULL DEFAULT '',
    transport        VARCHAR(16) NOT NULL DEFAULT 'UDP',
    status           TINYINT(1) NOT NULL DEFAULT 0,
    ip               VARCHAR(64) DEFAULT NULL,
    port             INT DEFAULT NULL,
    host_address     VARCHAR(128) DEFAULT NULL,
    expires          INT DEFAULT NULL,
    register_call_id VARCHAR(128) DEFAULT NULL,
    server_id        VARCHAR(50) DEFAULT NULL,
    create_time      VARCHAR(50) NOT NULL DEFAULT '',
    update_time      VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_sub_gb_id (device_gb_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='下级国标平台';

DROP TABLE IF EXISTS zws_subordinate_catalog;
CREATE TABLE IF NOT EXISTS zws_subordinate_catalog
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    platform_id BIGINT NOT NULL,
    device_id   VARCHAR(50) NOT NULL COMMENT '目录项国标编号',
    name        VARCHAR(255) DEFAULT NULL,
    parent_id   VARCHAR(50) DEFAULT NULL,
    type_code   VARCHAR(8) DEFAULT NULL,
    item_type   VARCHAR(16) DEFAULT NULL COMMENT 'node|channel',
    status      VARCHAR(16) DEFAULT NULL,
    civil_code  VARCHAR(16) DEFAULT NULL,
    manufacturer VARCHAR(128) DEFAULT NULL,
    model       VARCHAR(128) DEFAULT NULL,
    orphan      TINYINT(1) NOT NULL DEFAULT 0,
    raw_json    TEXT,
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_sub_catalog (platform_id, device_id),
    KEY idx_sub_catalog_parent (parent_id),
    CONSTRAINT fk_subcat_platform FOREIGN KEY (platform_id) REFERENCES zws_subordinate_platform (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='下级平台推送的目录';

DROP TABLE IF EXISTS zws_platform_catalog_snapshot;
CREATE TABLE IF NOT EXISTS zws_platform_catalog_snapshot
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    platform_id BIGINT NOT NULL,
    device_id   VARCHAR(50) NOT NULL,
    name        VARCHAR(255) DEFAULT NULL,
    parent_id   VARCHAR(50) DEFAULT NULL,
    item_type   VARCHAR(16) DEFAULT NULL,
    status      VARCHAR(16) DEFAULT NULL,
    hash        VARCHAR(64) DEFAULT NULL,
    UNIQUE KEY uk_plat_catalog_snap (platform_id, device_id),
    CONSTRAINT fk_snap_platform FOREIGN KEY (platform_id) REFERENCES zws_upper_platform (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='向上级推送目录快照';

DROP TABLE IF EXISTS zws_platform_group_share;
CREATE TABLE IF NOT EXISTS zws_platform_group_share
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    platform_id BIGINT NOT NULL,
    group_id    BIGINT NOT NULL COMMENT '共享的业务分组节点',
    UNIQUE KEY uk_plat_group (platform_id, group_id),
    CONSTRAINT fk_share_platform FOREIGN KEY (platform_id) REFERENCES zws_upper_platform (id) ON DELETE CASCADE,
    CONSTRAINT fk_share_group FOREIGN KEY (group_id) REFERENCES zws_group (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='上级平台按业务分组共享';

DROP TABLE IF EXISTS zws_platform_channel;
CREATE TABLE IF NOT EXISTS zws_platform_channel
(
    id             BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    platform_id    BIGINT NOT NULL,
    channel_id     BIGINT NOT NULL,
    custom_gb_code VARCHAR(50) DEFAULT NULL COMMENT '对外国标通道码',
    overrides_json LONGTEXT COMMENT '对外字段覆盖（名称/经纬度/PTZ…原 custom_* 收进 JSON）',
    create_time    VARCHAR(50) NOT NULL DEFAULT '',
    update_time    VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_plat_channel (platform_id, channel_id),
    CONSTRAINT fk_pch_platform FOREIGN KEY (platform_id) REFERENCES zws_upper_platform (id) ON DELETE CASCADE,
    CONSTRAINT fk_pch_channel FOREIGN KEY (channel_id) REFERENCES zws_channel (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通道对外映射（级联 overrides）';

DROP TABLE IF EXISTS zws_region_code_mapping;
CREATE TABLE IF NOT EXISTS zws_region_code_mapping
(
    region_id   BIGINT NOT NULL PRIMARY KEY COMMENT '区划节点',
    gb_code     VARCHAR(20) NOT NULL COMMENT '对外国标码',
    update_time VARCHAR(50) NOT NULL DEFAULT '',
    CONSTRAINT fk_recmap_region FOREIGN KEY (region_id) REFERENCES zws_region (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='区划节点对外编码映射';

DROP TABLE IF EXISTS zws_gb_code_seq;
CREATE TABLE IF NOT EXISTS zws_gb_code_seq
(
    prefix VARCHAR(14) NOT NULL PRIMARY KEY COMMENT '中心8+行业2+类型3+网络1',
    next   BIGINT NOT NULL DEFAULT 0 COMMENT '已分配的最大序号'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='国标码序号分配';

-- ===========================================================================
-- F. 接入 / 插件
-- ===========================================================================

DROP TABLE IF EXISTS zws_agent;
CREATE TABLE IF NOT EXISTS zws_agent
(
    id          VARCHAR(128) NOT NULL PRIMARY KEY COMMENT 'Agent ID',
    hostname    VARCHAR(255) DEFAULT NULL,
    ip          VARCHAR(64) DEFAULT NULL,
    version     VARCHAR(64) DEFAULT NULL,
    endpoint    VARCHAR(255) DEFAULT NULL COMMENT '平台可达地址',
    status      VARCHAR(32) NOT NULL DEFAULT 'offline',
    last_seen   BIGINT NOT NULL DEFAULT 0,
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='zero-agent';

DROP TABLE IF EXISTS zws_plugin;
CREATE TABLE IF NOT EXISTS zws_plugin
(
    id                VARCHAR(128) NOT NULL PRIMARY KEY COMMENT '插件实例ID',
    plugin_id         VARCHAR(128) DEFAULT NULL,
    vendor            VARCHAR(128) DEFAULT NULL,
    version           VARCHAR(64) DEFAULT NULL,
    protocols         TEXT,
    capabilities      TEXT,
    stream_transports TEXT,
    config_schema     LONGTEXT,
    endpoint          VARCHAR(255) DEFAULT NULL,
    online            TINYINT(1) NOT NULL DEFAULT 0,
    last_seen         BIGINT NOT NULL DEFAULT 0,
    create_time       VARCHAR(50) NOT NULL DEFAULT '',
    update_time       VARCHAR(50) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='IPC 插件实例';

DROP TABLE IF EXISTS zws_agent_plugin;
CREATE TABLE IF NOT EXISTS zws_agent_plugin
(
    id                VARCHAR(160) NOT NULL PRIMARY KEY COMMENT '运行时插件实例ID',
    agent_id          VARCHAR(128) NOT NULL,
    plugin_id         VARCHAR(128) NOT NULL,
    name              VARCHAR(255) DEFAULT NULL,
    vendor            VARCHAR(128) DEFAULT NULL,
    version           VARCHAR(64) DEFAULT NULL,
    protocols         TEXT,
    capabilities      TEXT,
    stream_transports TEXT,
    event_types       TEXT,
    config_schema     LONGTEXT,
    enabled           TINYINT(1) NOT NULL DEFAULT 1,
    desired_state     VARCHAR(32) NOT NULL DEFAULT 'enabled',
    actual_state      VARCHAR(32) NOT NULL DEFAULT 'discovered',
    source            VARCHAR(32) DEFAULT NULL,
    last_error        TEXT,
    last_seen         BIGINT NOT NULL DEFAULT 0,
    create_time       VARCHAR(50) NOT NULL DEFAULT '',
    update_time       VARCHAR(50) NOT NULL DEFAULT '',
    KEY idx_agent_plugin_agent (agent_id),
    CONSTRAINT fk_ap_agent FOREIGN KEY (agent_id) REFERENCES zws_agent (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Agent 托管的插件';

DROP TABLE IF EXISTS zws_plugin_package;
CREATE TABLE IF NOT EXISTS zws_plugin_package
(
    id                VARCHAR(64) NOT NULL PRIMARY KEY,
    plugin_id         VARCHAR(64) NOT NULL,
    version           VARCHAR(64) NOT NULL,
    name              VARCHAR(255) DEFAULT NULL,
    vendor            VARCHAR(128) DEFAULT NULL,
    object_key        VARCHAR(512) DEFAULT NULL,
    sha256            VARCHAR(64) DEFAULT NULL,
    size_bytes        BIGINT NOT NULL DEFAULT 0,
    manifest_json     LONGTEXT,
    config_schema     LONGTEXT,
    protocols         TEXT,
    capabilities      TEXT,
    stream_transports TEXT,
    event_types       TEXT,
    architectures     TEXT,
    status            VARCHAR(32) NOT NULL DEFAULT 'valid',
    error_msg         TEXT,
    create_time       VARCHAR(50) NOT NULL DEFAULT '',
    update_time       VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_plugin_pkg_version (plugin_id, version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='插件安装包';

DROP TABLE IF EXISTS zws_plugin_install;
CREATE TABLE IF NOT EXISTS zws_plugin_install
(
    id          VARCHAR(64) NOT NULL PRIMARY KEY,
    agent_id    VARCHAR(128) NOT NULL,
    package_id  VARCHAR(64) NOT NULL,
    plugin_id   VARCHAR(64) NOT NULL,
    version     VARCHAR(64) DEFAULT NULL,
    status      VARCHAR(32) NOT NULL DEFAULT 'pending',
    last_error  TEXT,
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT '',
    KEY idx_plugin_install_agent (agent_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='插件包安装任务';

-- ===========================================================================
-- G. 用户 / 权限
-- ===========================================================================

DROP TABLE IF EXISTS zws_user_role;
CREATE TABLE IF NOT EXISTS zws_user_role
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name        VARCHAR(50) NOT NULL DEFAULT '',
    authority   VARCHAR(512) NOT NULL DEFAULT '' COMMENT '菜单权限：* 或 JSON 数组',
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户角色';

DROP TABLE IF EXISTS zws_user;
CREATE TABLE IF NOT EXISTS zws_user
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    username    VARCHAR(255) NOT NULL,
    password    VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'MD5',
    role_id     BIGINT DEFAULT NULL,
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_user_username (username),
    CONSTRAINT fk_user_role FOREIGN KEY (role_id) REFERENCES zws_user_role (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='平台用户';

DROP TABLE IF EXISTS zws_user_api_key;
CREATE TABLE IF NOT EXISTS zws_user_api_key
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id     BIGINT DEFAULT NULL,
    app         VARCHAR(255) DEFAULT NULL,
    api_key     TEXT,
    expired_at  BIGINT DEFAULT NULL,
    remark      VARCHAR(255) DEFAULT NULL,
    enable      TINYINT(1) NOT NULL DEFAULT 1,
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT '',
    CONSTRAINT fk_apikey_user FOREIGN KEY (user_id) REFERENCES zws_user (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户 API Key';

CREATE TABLE IF NOT EXISTS zws_access_token
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name        VARCHAR(128) NOT NULL DEFAULT '' COMMENT '集成方名称',
    token       VARCHAR(128) NOT NULL DEFAULT '' COMMENT '密钥串',
    enabled     TINYINT(1) NOT NULL DEFAULT 1,
    remark      VARCHAR(255) NOT NULL DEFAULT '',
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_access_token_token (token),
    KEY idx_access_token_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='对外接入密钥';

-- ===========================================================================
-- H. 系统配置
-- ===========================================================================

DROP TABLE IF EXISTS zws_gb_sip_config;
CREATE TABLE IF NOT EXISTS zws_gb_sip_config
(
    id                   BIGINT NOT NULL PRIMARY KEY COMMENT '固定为1',
    ip                   VARCHAR(64)  NOT NULL,
    port                 INT          NOT NULL,
    domain               VARCHAR(32)  NOT NULL,
    device_id            VARCHAR(32)  NOT NULL COMMENT '本平台国标编号',
    password             VARCHAR(64)  NOT NULL DEFAULT '',
    alarm                TINYINT(1)   NOT NULL DEFAULT 0,
    require_pre_register TINYINT(1)   NOT NULL DEFAULT 1,
    transport            VARCHAR(16)  NOT NULL DEFAULT 'UDP',
    stream_mode          VARCHAR(16)  NOT NULL DEFAULT 'UDP',
    create_time          VARCHAR(50) NOT NULL DEFAULT '',
    update_time          VARCHAR(50) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='国标SIP配置';

DROP TABLE IF EXISTS zws_object_store_config;
CREATE TABLE IF NOT EXISTS zws_object_store_config
(
    id          BIGINT NOT NULL PRIMARY KEY COMMENT '固定为1',
    enabled     TINYINT(1) NOT NULL DEFAULT 0,
    provider    VARCHAR(32) NOT NULL DEFAULT 'noop',
    endpoint    VARCHAR(255) DEFAULT NULL,
    region      VARCHAR(64) DEFAULT NULL,
    bucket      VARCHAR(128) DEFAULT NULL,
    access_key  VARCHAR(128) DEFAULT NULL,
    secret_key  VARCHAR(256) DEFAULT NULL,
    use_ssl     TINYINT(1) NOT NULL DEFAULT 0,
    path_style  TINYINT(1) NOT NULL DEFAULT 1,
    public_base VARCHAR(255) DEFAULT NULL,
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='对象存储对接';

DROP TABLE IF EXISTS zws_mqtt_config;
CREATE TABLE IF NOT EXISTS zws_mqtt_config
(
    id               BIGINT NOT NULL PRIMARY KEY,
    enabled          TINYINT(1) NOT NULL DEFAULT 0,
    broker           VARCHAR(255) DEFAULT NULL,
    client_id_prefix VARCHAR(64) NOT NULL DEFAULT 'zws',
    username         VARCHAR(128) DEFAULT NULL,
    password         VARCHAR(256) DEFAULT NULL,
    topic_prefix     VARCHAR(128) NOT NULL DEFAULT 'zws/events',
    qos              TINYINT(1) NOT NULL DEFAULT 1,
    keepalive_sec    INT NOT NULL DEFAULT 30,
    tls_enabled      TINYINT(1) NOT NULL DEFAULT 0,
    ca_cert          TEXT,
    client_cert      TEXT,
    client_key       TEXT,
    create_time      VARCHAR(50) NOT NULL DEFAULT '',
    update_time      VARCHAR(50) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='MQTT消息队列配置';

-- ===========================================================================
-- I. 审计
-- ===========================================================================

DROP TABLE IF EXISTS zws_audit_log;
CREATE TABLE IF NOT EXISTS zws_audit_log
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id     BIGINT DEFAULT NULL,
    username    VARCHAR(255) NOT NULL DEFAULT '',
    action      VARCHAR(64) NOT NULL DEFAULT '',
    target_type VARCHAR(32) NOT NULL DEFAULT '',
    target_id   VARCHAR(64) NOT NULL DEFAULT '',
    detail      LONGTEXT,
    ip          VARCHAR(64) NOT NULL DEFAULT '',
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    KEY idx_audit_user_time (user_id, create_time),
    KEY idx_audit_target (target_type, target_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='操作审计日志';

-- ===========================================================================
-- 数据字典（系统配置可维护：设备类型/通道类型/坐标系/厂商…）
-- ===========================================================================
DROP TABLE IF EXISTS zws_dict_type;
CREATE TABLE IF NOT EXISTS zws_dict_type
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    code        VARCHAR(64) NOT NULL COMMENT '类型码 device_type/channel_type/coordinate_system/manufacturer',
    name        VARCHAR(128) NOT NULL DEFAULT '' COMMENT '显示名',
    remark      VARCHAR(255) DEFAULT NULL,
    builtin     TINYINT(1) NOT NULL DEFAULT 0 COMMENT '内置类型不可删',
    sort        INT NOT NULL DEFAULT 0,
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_dict_type_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据字典类型';

DROP TABLE IF EXISTS zws_dict_item;
CREATE TABLE IF NOT EXISTS zws_dict_item
(
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    type_code   VARCHAR(64) NOT NULL COMMENT '关联 zws_dict_type.code',
    value       VARCHAR(64) NOT NULL COMMENT '存库值',
    label       VARCHAR(128) NOT NULL DEFAULT '' COMMENT '显示文本',
    sort        INT NOT NULL DEFAULT 0,
    enabled     TINYINT(1) NOT NULL DEFAULT 1,
    builtin     TINYINT(1) NOT NULL DEFAULT 0 COMMENT '内置项可停用不可删',
    remark      VARCHAR(255) DEFAULT NULL,
    create_time VARCHAR(50) NOT NULL DEFAULT '',
    update_time VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_dict_item (type_code, value),
    KEY idx_dict_item_type (type_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据字典项';

-- ===========================================================================
-- 初始数据
-- ===========================================================================

-- 初始化角色：管理员 / 运维 / 视频值班
INSERT INTO zws_user_role (id, name, authority, create_time, update_time)
VALUES (1, '管理员', '*', '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       (2, '运维', '["ops","access","media","storage"]', '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       (3, '视频值班', '["observe","media","access"]', '2021-04-13 14:14:57', '2021-04-13 14:14:57');

-- 初始化管理员账号 admin（密码 admin，MD5）
INSERT INTO zws_user (id, username, password, role_id, create_time, update_time)
VALUES (1, 'admin', '21232f297a57a5a743894a0e4a801fc3', 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57');

-- 默认国标 SIP（与 configs/config.yaml 对齐；新装可随后在页面改编码）
INSERT INTO zws_gb_sip_config (id, ip, port, domain, device_id, password, alarm, require_pre_register, transport, stream_mode, create_time, update_time)
VALUES (1, '127.0.0.1', 5060, '3402000000', '34020000002000000001', '12345678', 0, 1, 'UDP', 'UDP',
        '2021-04-13 14:14:57', '2021-04-13 14:14:57');

-- 对象存储默认 noop
INSERT INTO zws_object_store_config (id, enabled, provider, path_style, create_time, update_time)
VALUES (1, 0, 'noop', 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57');

-- 数据字典：类型 + 内置项（builtin=1 可停用不可删；用户可自行扩展）
INSERT INTO zws_dict_type (id, code, name, builtin, sort, create_time, update_time)
VALUES (1, 'device_type', '设备类型', 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       (2, 'channel_type', '通道类型', 1, 2, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       (3, 'coordinate_system', '坐标系', 1, 3, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       (4, 'manufacturer', '厂商', 1, 4, '2021-04-13 14:14:57', '2021-04-13 14:14:57');

INSERT INTO zws_dict_item (type_code, value, label, sort, enabled, builtin, create_time, update_time)
VALUES ('device_type', 'bullet', '枪机', 1, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('device_type', 'dome', '半球', 2, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('device_type', 'ptz', '球机', 3, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('device_type', 'remote_ptz', '遥控枪机', 4, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('device_type', 'remote_dome', '遥控半球', 5, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('device_type', 'multi_panorama', '多目全景', 6, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('device_type', 'multi_split', '多目分割', 7, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('device_type', 'other', '其他', 99, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('channel_type', 'visible', '可见光', 1, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('channel_type', 'thermal', '热成像', 2, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('channel_type', 'infrared', '红外', 3, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('channel_type', 'temperature', '测温', 4, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('channel_type', 'other', '其他', 99, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('coordinate_system', 'WGS84', 'WGS84', 1, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('coordinate_system', 'CGCS2000', 'CGCS2000', 2, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('coordinate_system', 'GCJ02', 'GCJ02', 3, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('coordinate_system', 'BD09', 'BD09', 4, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('manufacturer', '海康', '海康', 1, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('manufacturer', '大华', '大华', 2, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('manufacturer', '宇视', '宇视', 3, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('manufacturer', '华为', '华为', 4, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57'),
       ('manufacturer', '其他', '其他', 99, 1, 1, '2021-04-13 14:14:57', '2021-04-13 14:14:57');

SET FOREIGN_KEY_CHECKS = 1;
