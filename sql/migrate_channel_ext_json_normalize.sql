-- channel_ext.gb_json / extra_json 里空串 '' 或非法 JSON 会让 JSON_EXTRACT 报
-- Error 3141（The document is empty）。把空串/非法值归一为 '{}'（NULL 保持不动，
-- JSON_EXTRACT(NULL,..) 返回 NULL 是安全的）。
UPDATE zws_channel_ext
SET gb_json = '{}'
WHERE gb_json = '' OR (gb_json IS NOT NULL AND JSON_VALID(gb_json) = 0);

UPDATE zws_channel_ext
SET extra_json = '{}'
WHERE extra_json = '' OR (extra_json IS NOT NULL AND JSON_VALID(extra_json) = 0);
