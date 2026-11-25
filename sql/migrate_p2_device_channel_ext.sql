-- P2：设备类型 / 通道类型 / 监视方位 加列（幂等，可重复执行）
DROP PROCEDURE IF EXISTS tmp_add_col;
DELIMITER //
CREATE PROCEDURE tmp_add_col(IN tbl VARCHAR(64), IN col VARCHAR(64), IN ddl VARCHAR(255))
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = DATABASE() AND table_name = tbl AND column_name = col) THEN
        SET @s = CONCAT('ALTER TABLE ', tbl, ' ADD COLUMN ', col, ' ', ddl);
        PREPARE st FROM @s;
        EXECUTE st;
        DEALLOCATE PREPARE st;
    END IF;
END//
DELIMITER ;

CALL tmp_add_col('zws_device', 'device_type', "VARCHAR(64) NOT NULL DEFAULT '' COMMENT '设备类型（字典 device_type）'");
CALL tmp_add_col('zws_channel', 'channel_type', "VARCHAR(32) NOT NULL DEFAULT '' COMMENT '通道类型（字典 channel_type）'");
CALL tmp_add_col('zws_channel_ext', 'direction', "INT NOT NULL DEFAULT 0 COMMENT '监视方位(GB方位枚举,0=未设)'");

DROP PROCEDURE IF EXISTS tmp_add_col;
