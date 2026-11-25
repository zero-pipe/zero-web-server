-- 1) 插件设备的 access_binding.protocol 之前写成厂商协议（如 hik.private.v5），
--    违反 protocol 枚举 (gb28181|onvif|plugin)，导致 /channels?protocol=plugin 过滤不到通道。
--    归一为 'plugin'，并把厂商协议原值保存到 config_json.protocol 以不丢失。
UPDATE zws_access_binding
SET config_json = JSON_SET(COALESCE(config_json, JSON_OBJECT()), '$.protocol', protocol),
    protocol    = 'plugin'
WHERE plugin_instance_id IS NOT NULL
  AND plugin_instance_id <> ''
  AND protocol NOT IN ('gb28181', 'onvif', 'plugin');

-- 2) ONVIF 事件订阅默认应为关闭（由用户勾选开启）。旧数据里默认写成了 true，重置。
UPDATE zws_access_binding
SET config_json = JSON_SET(COALESCE(config_json, JSON_OBJECT()), '$.eventSubscribe', FALSE)
WHERE protocol = 'onvif'
  AND JSON_EXTRACT(config_json, '$.eventSubscribe') = TRUE;
