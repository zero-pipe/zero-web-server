-- 修复：gb_code 唯一索引把空串 '' 也当成唯一值，导致第二台 ONVIF/插件设备
-- （无 GB 编码、gb_code=''）插入报 Duplicate entry '' for key uk_*_gb_code。
-- 改为仅对非空 gb_code 生效的函数索引：NULLIF(gb_code,'') → 空串变 NULL，
-- MySQL 唯一索引允许多个 NULL，真实国标码仍唯一。MySQL 8.0.13+ 支持函数索引。
ALTER TABLE zws_device
    DROP INDEX uk_device_gb_code,
    ADD UNIQUE KEY uk_device_gb_code ((NULLIF(gb_code, '')));

ALTER TABLE zws_channel
    DROP INDEX uk_channel_gb_code,
    ADD UNIQUE KEY uk_channel_gb_code ((NULLIF(gb_code, '')));
