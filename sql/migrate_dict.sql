-- 数据字典子系统：建表 + 内置种子（幂等，可重复执行）
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

INSERT IGNORE INTO zws_dict_type (code, name, builtin, sort, create_time, update_time) VALUES
 ('device_type', '设备类型', 1, 1, NOW(), NOW()),
 ('channel_type', '通道类型', 1, 2, NOW(), NOW()),
 ('coordinate_system', '坐标系', 1, 3, NOW(), NOW()),
 ('manufacturer', '厂商', 1, 4, NOW(), NOW());

INSERT IGNORE INTO zws_dict_item (type_code, value, label, sort, enabled, builtin, create_time, update_time) VALUES
 ('device_type', 'bullet', '枪机', 1, 1, 1, NOW(), NOW()),
 ('device_type', 'dome', '半球', 2, 1, 1, NOW(), NOW()),
 ('device_type', 'ptz', '球机', 3, 1, 1, NOW(), NOW()),
 ('device_type', 'remote_ptz', '遥控枪机', 4, 1, 1, NOW(), NOW()),
 ('device_type', 'remote_dome', '遥控半球', 5, 1, 1, NOW(), NOW()),
 ('device_type', 'multi_panorama', '多目全景', 6, 1, 1, NOW(), NOW()),
 ('device_type', 'multi_split', '多目分割', 7, 1, 1, NOW(), NOW()),
 ('device_type', 'other', '其他', 99, 1, 1, NOW(), NOW()),
 ('channel_type', 'visible', '可见光', 1, 1, 1, NOW(), NOW()),
 ('channel_type', 'thermal', '热成像', 2, 1, 1, NOW(), NOW()),
 ('channel_type', 'infrared', '红外', 3, 1, 1, NOW(), NOW()),
 ('channel_type', 'temperature', '测温', 4, 1, 1, NOW(), NOW()),
 ('channel_type', 'other', '其他', 99, 1, 1, NOW(), NOW()),
 ('coordinate_system', 'WGS84', 'WGS84', 1, 1, 1, NOW(), NOW()),
 ('coordinate_system', 'CGCS2000', 'CGCS2000', 2, 1, 1, NOW(), NOW()),
 ('coordinate_system', 'GCJ02', 'GCJ02', 3, 1, 1, NOW(), NOW()),
 ('coordinate_system', 'BD09', 'BD09', 4, 1, 1, NOW(), NOW()),
 ('manufacturer', '海康', '海康', 1, 1, 1, NOW(), NOW()),
 ('manufacturer', '大华', '大华', 2, 1, 1, NOW(), NOW()),
 ('manufacturer', '宇视', '宇视', 3, 1, 1, NOW(), NOW()),
 ('manufacturer', '华为', '华为', 4, 1, 1, NOW(), NOW()),
 ('manufacturer', '其他', '其他', 99, 1, 1, NOW(), NOW());
