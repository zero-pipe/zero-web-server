package persistence

import (
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/google/uuid"
)

// ===========================================================================
// 新 schema 通用桥接：对外（domain）用字符串 ID / 内码，库内用 BIGINT 自增主键。
// 所有 repository 在边界处用这里的函数互转，保持 /api/v1 契约稳定。
// ===========================================================================

// newBusinessID 生成一个对外业务标识（事件等无内码概念时使用）。
func newBusinessID(prefix string) string {
	return prefix + strings.ReplaceAll(uuid.NewString(), "-", "")
}

// parseID 把对外的字符串 ID 解析为库内 BIGINT 主键；解析失败返回 ok=false。
func parseID(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// formatID 把库内 BIGINT 主键格式化为对外字符串 ID。
func formatID(v int64) string { return strconv.FormatInt(v, 10) }

// int64Ptr 便于给可空 BIGINT 字段赋指针值。
func int64Ptr(v int64) *int64 { return &v }

// firstNonEmptyStr 返回第一个去空白后非空的字符串。
func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// deviceIDByInternalCode 用设备内码（D…）解析库内 BIGINT 主键；无匹配返回 0。
func deviceIDByInternalCode(db *gorm.DB, code string) (int64, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return 0, nil
	}
	var id int64
	err := db.Table("zws_device").Select("id").Where("internal_code = ?", code).Scan(&id).Error
	return id, err
}

// channelIDByInternalCode 用通道内码（C…）解析库内 BIGINT 主键；无匹配返回 0。
func channelIDByInternalCode(db *gorm.DB, code string) (int64, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return 0, nil
	}
	var id int64
	err := db.Table("zws_channel").Select("id").Where("internal_code = ?", code).Scan(&id).Error
	return id, err
}
