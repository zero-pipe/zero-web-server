package model

import "gorm.io/gorm"

// BeforeSave 保证能参与 JSON 查询的列始终是合法 JSON。
// 空串 '' 不是合法 JSON，会让 JSON_EXTRACT 报 Error 3141（The document is empty），
// 例如按 civilCode/parentId 检索区域通道时。统一把空值落成 '{}'。
func (c *ChannelExt) BeforeSave(tx *gorm.DB) error {
	if c.GBJSON == "" {
		c.GBJSON = "{}"
	}
	if c.ExtraJSON == "" {
		c.ExtraJSON = "{}"
	}
	return nil
}
