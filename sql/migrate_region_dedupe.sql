-- 行政区划重复数据清理（zws_common_region 同名同编号多行）。
-- 背景：早期由 AutoMigrate 建出的 zws_common_region 缺少 device_id 唯一索引，
-- 因此可写入重复编号/名称的区划。清理后重启服务会自动补唯一索引
-- （模型已声明 uk_common_region_device_id）；也可手工执行下面第 3 步。
-- 执行前请备份 zws_common_region。

-- 1) 把重复行下的子区划改挂到保留行（每组 device_id 保留最小 id）。
UPDATE zws_common_region AS child
JOIN zws_common_region AS dup ON child.parent_id = dup.id
JOIN (
    SELECT device_id, MIN(id) AS keep_id
    FROM zws_common_region
    GROUP BY device_id
    HAVING COUNT(*) > 1
) AS k ON k.device_id = dup.device_id
SET child.parent_id = k.keep_id
WHERE dup.id <> k.keep_id;

-- 2) 删除重复行，仅保留每组最小 id。
DELETE dup
FROM zws_common_region AS dup
JOIN (
    SELECT device_id, MIN(id) AS keep_id
    FROM zws_common_region
    GROUP BY device_id
    HAVING COUNT(*) > 1
) AS k ON k.device_id = dup.device_id
WHERE dup.id <> k.keep_id;

-- 3) 补唯一索引（索引已存在会报 Duplicate key name，属预期，可忽略）。
ALTER TABLE zws_common_region ADD UNIQUE KEY uk_common_region_device_id (device_id);
