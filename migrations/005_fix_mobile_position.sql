-- 修正 zws_mobile_position.channel_id 类型，与 zero-web-kit 代码对齐
-- 代码写入的是 zws_device_channel.id（INT），非国标编码字符串
-- 已有库从 001/旧版导入后执行；004 全新安装无需本脚本

SET NAMES utf8mb4;

DELIMITER //
DROP PROCEDURE IF EXISTS `zwk_fix_mobile_position_channel_id`//
CREATE PROCEDURE `zwk_fix_mobile_position_channel_id`()
proc: BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = DATABASE() AND table_name = 'zws_mobile_position'
    ) THEN
        LEAVE proc;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = DATABASE()
          AND table_name = 'zws_mobile_position'
          AND column_name = 'channel_id'
          AND data_type IN ('varchar', 'char', 'text', 'mediumtext', 'longtext')
    ) THEN
        TRUNCATE TABLE zws_mobile_position;
        ALTER TABLE zws_mobile_position
            MODIFY channel_id INT NOT NULL COMMENT '通道数据库主键ID';
    END IF;
END//
DELIMITER ;

CALL zwk_fix_mobile_position_channel_id();
DROP PROCEDURE IF EXISTS zwk_fix_mobile_position_channel_id;
