-- 云存储旁路录制（zstorage）：已有库增量迁移。
-- 新库直接以 sql/init_zws_mysql.sql 为准，本文件仅用于升级已存在的库。
-- 幂等性说明：MySQL 不支持 ADD COLUMN IF NOT EXISTS，重复执行会报错可忽略；按需分次执行。

-- 表：云存储节点（若由 AutoMigrate 已建可跳过）
CREATE TABLE IF NOT EXISTS zws_storage_node
(
    id           BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    node_id      VARCHAR(191) NOT NULL COMMENT 'zstorage 上报 media_server_id',
    name         VARCHAR(128) NOT NULL DEFAULT '',
    ip           VARCHAR(50) DEFAULT NULL,
    http_port    INT DEFAULT NULL COMMENT 'zstorage HTTP/API/FLV 端口',
    rtsp_port    INT DEFAULT NULL,
    secret       VARCHAR(50) DEFAULT NULL COMMENT '与 zstorage [api] secret 一致',
    version      VARCHAR(50) DEFAULT NULL,
    capacity_gb  INT NOT NULL DEFAULT 0,
    used_gb      INT NOT NULL DEFAULT 0,
    online       TINYINT(1) NOT NULL DEFAULT 0,
    last_seen    BIGINT NOT NULL DEFAULT 0,
    default_node TINYINT(1) NOT NULL DEFAULT 0,
    create_time  VARCHAR(50) NOT NULL DEFAULT '',
    update_time  VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE KEY uk_storage_node (node_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='云存储节点（旁路录制）';

-- 通道 → 存储节点绑定（空=不旁路录）
ALTER TABLE zws_channel_ext
    ADD COLUMN storage_node_id VARCHAR(128) DEFAULT NULL COMMENT '旁路录制存储节点 node_id（空=不旁路录）' AFTER record_plan_id;

-- 录像表增列：区分 MP4 / PS 槽位来源
ALTER TABLE zws_cloud_record
    ADD COLUMN storage_node_id VARCHAR(128) DEFAULT NULL COMMENT 'PS 录制的存储节点 node_id' AFTER collect,
    ADD COLUMN container_format VARCHAR(16) NOT NULL DEFAULT 'mp4' COMMENT 'mp4|ps' AFTER storage_node_id,
    ADD COLUMN slot_id BIGINT NOT NULL DEFAULT 0 COMMENT 'PS 槽位号（mp4 为 0）' AFTER container_format,
    ADD COLUMN status INT NOT NULL DEFAULT 2 COMMENT '1 录制中/2 正常/3 残片恢复/4 已清理' AFTER slot_id,
    ADD KEY idx_cloud_node_slot (storage_node_id, slot_id);
