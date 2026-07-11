-- ZWS MySQL patch (legacy upgrade)
-- Legacy upgrade script (archived)
-- Run after 001_init_zws_mysql.sql on existing deployments

SET NAMES utf8mb4;
create table IF NOT EXISTS zws_jt_terminal (
                                 id serial primary key,
                                 phone_number character varying(50),
                                 terminal_id character varying(50),
                                 province_id character varying(50),
                                 province_text character varying(100),
                                 city_id character varying(50),
                                 city_text character varying(100),
                                 maker_id character varying(50),
                                 model character varying(50),
                                 plate_color character varying(50),
                                 plate_no character varying(50),
                                 longitude double precision,
                                 latitude double precision,
                                 status bool default false,
                                 register_time character varying(50) default null,
                                 update_time character varying(50) not null,
                                 create_time character varying(50) not null,
                                 geo_coord_sys character varying(50),
                                 media_server_id character varying(50) default 'auto',
                                 sdp_ip character varying(50),
                                 constraint uk_jt_device_id_device_id unique (id, phone_number)
);

create table IF NOT EXISTS zws_jt_channel (
                               id serial primary key,
                               terminal_db_id integer,
                               channel_id integer,
                               has_audio bool default false,
                               name character varying(255),
                               update_time character varying(50) not null,
                               create_time character varying(50) not null,
                               constraint uk_jt_channel_id_device_id unique (terminal_db_id, channel_id)
);


DELIMITER //  -- 重定义分隔符避免分号冲突
DROP PROCEDURE IF EXISTS `zws_20250708`//
CREATE PROCEDURE `zws_20250708`()
BEGIN
    IF NOT EXISTS (SELECT column_name FROM information_schema.columns
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_media_server' and column_name = 'jtt_proxy_port')
    THEN
        ALTER TABLE zws_media_server ADD jtt_proxy_port  integer;
    END IF;
END; //
DELIMITER ;
call zws_20250708();
DROP PROCEDURE zws_20250708;

DELIMITER //  -- 重定义分隔符避免分号冲突
DROP PROCEDURE IF EXISTS `zws_20250917`//
CREATE PROCEDURE `zws_20250917`()
BEGIN
    IF NOT EXISTS (SELECT column_name FROM information_schema.columns
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_media_server' and column_name = 'mp4_port')
    THEN
        ALTER TABLE zws_media_server ADD mp4_port integer;
    END IF;

    IF NOT EXISTS (SELECT column_name FROM information_schema.columns
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_media_server' and column_name = 'mp4_ssl_port')
    THEN
        ALTER TABLE zws_media_server ADD mp4_ssl_port integer;
    END IF;
END; //
DELIMITER ;
call zws_20250917();
DROP PROCEDURE zws_20250917;


DELIMITER //  -- 重定义分隔符避免分号冲突
DROP PROCEDURE IF EXISTS `zws_20250924`//
CREATE PROCEDURE `zws_20250924`()
BEGIN
    IF NOT EXISTS (SELECT column_name FROM information_schema.columns
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_device_channel' and column_name = 'enable_broadcast')
    THEN
        ALTER TABLE zws_device_channel ADD enable_broadcast integer default 0;
    END IF;

    IF NOT EXISTS (SELECT column_name FROM information_schema.columns
                       WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_device_channel' and column_name = 'map_level')
    THEN
        ALTER TABLE zws_device_channel ADD map_level integer default 0;
    END IF;

    IF NOT EXISTS (SELECT column_name FROM information_schema.columns
                       WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_common_group' and column_name = 'alias')
    THEN
        ALTER TABLE zws_common_group ADD alias varchar(255) default null;
    END IF;
END; //
DELIMITER ;
call zws_20250924();
DROP PROCEDURE zws_20250924;

DELIMITER //  -- 重定义分隔符避免分号冲突
DROP PROCEDURE IF EXISTS `zws_20251027`//
CREATE PROCEDURE `zws_20251027`()
BEGIN
    IF EXISTS (SELECT column_name FROM information_schema.columns
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_stream_proxy' and column_name = 'enable_remove_none_reader')
    THEN
        ALTER TABLE zws_stream_proxy DROP enable_remove_none_reader;
    END IF;
END; //
DELIMITER ;
call zws_20251027();
DROP PROCEDURE zws_20251027;


DELIMITER //  -- 重定义分隔符避免分号冲突
DROP PROCEDURE IF EXISTS `zws_20251101`//
CREATE PROCEDURE `zws_20251101`()
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.STATISTICS
               WHERE TABLE_SCHEMA = (SELECT DATABASE())
                 AND TABLE_NAME = 'zws_media_server'
                 AND INDEX_NAME = 'uk_media_server_unique_ip_http_port')
    THEN
        drop index uk_media_server_unique_ip_http_port on zws_media_server;
    END IF;
END; //
DELIMITER ;
call zws_20251101();
DROP PROCEDURE zws_20251101;

DELIMITER //  -- 重定义分隔符避免分号冲突
DROP PROCEDURE IF EXISTS `zws_202601025`//
CREATE PROCEDURE `zws_202601025`()
BEGIN
    IF EXISTS (SELECT column_name FROM information_schema.columns
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_device' and column_name = 'register_time')
    THEN
        ALTER TABLE zws_device DROP register_time;
    END IF;
    IF EXISTS (SELECT column_name FROM information_schema.columns
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_device' and column_name = 'keepalive_time')
    THEN
        ALTER TABLE zws_device DROP keepalive_time;
    END IF;
END; //
DELIMITER ;
call zws_202601025();
DROP PROCEDURE zws_202601025;

create table IF NOT EXISTS zws_alarm (
    id serial primary key COMMENT '主键ID',
    channel_id integer COMMENT '关联通道的数据库id',
    description character varying(255) COMMENT '报警描述',
    snap_path character varying(255) COMMENT '报警快照路径',
    record_path character varying(255) COMMENT '报警录像路径',
    longitude double precision COMMENT '报警附带的经度',
    latitude double precision COMMENT '报警附带的纬度',
    alarm_type integer COMMENT '报警类别',
    alarm_time bigint COMMENT '报警时间'
);



DELIMITER //  -- 重定义分隔符避免分号冲突
DROP PROCEDURE IF EXISTS `zws_20260417`//
CREATE PROCEDURE `zws_20260417`()
BEGIN
 IF EXISTS (SELECT table_name FROM information_schema.tables
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and table_name = 'zws_device_mobile_position')
 THEN
 IF NOT EXISTS (SELECT column_name FROM information_schema.columns
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_device_mobile_position' and column_name = 'timestamp')
    THEN
    ALTER TABLE zws_device_mobile_position ADD timestamp BIGINT COMMENT '上报时间';
END IF;
IF EXISTS (SELECT column_name FROM information_schema.columns
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_device_mobile_position' and column_name = 'time')
    THEN
    UPDATE zws_device_mobile_position SET timestamp = UNIX_TIMESTAMP(time) * 1000;
    ALTER TABLE zws_device_mobile_position DROP time;
END IF;
IF EXISTS (SELECT column_name FROM information_schema.columns
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_device_mobile_position' and column_name = 'device_id')
    THEN
    ALTER TABLE zws_device_mobile_position DROP device_id;
END IF;
IF EXISTS (SELECT column_name FROM information_schema.columns
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_device_mobile_position' and column_name = 'device_name')
    THEN
    ALTER TABLE zws_device_mobile_position DROP device_name;
END IF;
IF EXISTS (SELECT column_name FROM information_schema.columns
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and  table_name = 'zws_device_mobile_position' and column_name = 'report_source')
    THEN
ALTER TABLE zws_device_mobile_position DROP report_source;
END IF;
-- 修改表名（目标表已存在时跳过，避免重复执行失败）
IF NOT EXISTS (SELECT table_name FROM information_schema.tables
                   WHERE TABLE_SCHEMA = (SELECT DATABASE()) and table_name = 'zws_mobile_position')
    THEN
ALTER TABLE zws_device_mobile_position RENAME TO zws_mobile_position;
END IF;
END IF;

END; //
DELIMITER ;
call zws_20260417();
DROP PROCEDURE zws_20260417;

DELIMITER //
DROP PROCEDURE IF EXISTS `zws_20260521`//
CREATE PROCEDURE `zws_20260521`()
BEGIN
    IF NOT EXISTS (SELECT 1
                   FROM information_schema.STATISTICS
                   WHERE TABLE_SCHEMA = (SELECT DATABASE())
                     AND TABLE_NAME = 'zws_device_channel'
                     AND INDEX_NAME = 'uk_device_channel_source')
    THEN
        -- 用 GROUP BY + LEFT JOIN 替代自连接 DELETE
        DELETE t1
        FROM zws_device_channel t1
                 LEFT JOIN (SELECT MAX(id) as id
                            FROM zws_device_channel
                            GROUP BY data_device_id, device_id) t2 ON t1.id = t2.id
        WHERE t2.id IS NULL;

ALTER TABLE zws_device_channel
    ADD UNIQUE INDEX uk_device_channel_source (data_device_id, device_id);
END IF;
END; //
DELIMITER ;
call zws_20260521();
DROP PROCEDURE zws_20260521;






