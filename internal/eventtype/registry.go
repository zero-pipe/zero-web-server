// Package eventtype 定义平台统一事件类型注册表。
package eventtype

import "strings"

// Type 是一个规范事件类型。报警是事件的子集：AlarmEligible=true 才会投影报警。
type Type struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	Category       string `json:"category"` // alarm | status | operation | ai | metadata
	Severity       string `json:"severity"`
	AlarmEligible  bool   `json:"alarmEligible"`
	RequiresVideo  bool   `json:"requiresVideo"`
	BBoxCapable    bool   `json:"bboxCapable"`
	VendorMappings string `json:"vendorMappings"`
	Enabled        bool   `json:"enabled"`
}

// 分类。
const (
	CategoryAlarm     = "alarm"
	CategoryStatus    = "status"
	CategoryOperation = "operation"
	CategoryAI        = "ai"
	CategoryMetadata  = "metadata"
)

// Builtins 是首版内置规范码，与 docs/PLATFORM_CONVENTIONS.md 保持一致。
func Builtins() []Type {
	return []Type{
		{Code: "motion", Name: "移动侦测", Category: CategoryAlarm, Severity: "warning", AlarmEligible: true, BBoxCapable: true, Enabled: true},
		{Code: "line_crossing", Name: "越界侦测", Category: CategoryAlarm, Severity: "major", AlarmEligible: true, RequiresVideo: true, BBoxCapable: true, Enabled: true},
		{Code: "intrusion", Name: "区域入侵", Category: CategoryAlarm, Severity: "major", AlarmEligible: true, RequiresVideo: true, BBoxCapable: true, Enabled: true},
		{Code: "loitering", Name: "徘徊", Category: CategoryAlarm, Severity: "major", AlarmEligible: true, RequiresVideo: true, BBoxCapable: true, Enabled: true},
		{Code: "video_loss", Name: "视频丢失", Category: CategoryAlarm, Severity: "major", AlarmEligible: true, RequiresVideo: true, Enabled: true},
		{Code: "tamper", Name: "遮挡/破坏", Category: CategoryAlarm, Severity: "major", AlarmEligible: true, RequiresVideo: true, BBoxCapable: true, Enabled: true},
		{Code: "audio_abnormal", Name: "音频异常", Category: CategoryAlarm, Severity: "warning", AlarmEligible: true, Enabled: true},
		{Code: "scene_change", Name: "场景变更", Category: CategoryAlarm, Severity: "warning", AlarmEligible: true, RequiresVideo: true, Enabled: true},
		{Code: "face", Name: "人脸", Category: CategoryAI, Severity: "info", AlarmEligible: true, RequiresVideo: true, BBoxCapable: true, Enabled: true},
		{Code: "plate", Name: "车牌", Category: CategoryAI, Severity: "info", AlarmEligible: true, RequiresVideo: true, BBoxCapable: true, Enabled: true},
		{Code: "alarm_input", Name: "报警输入", Category: CategoryAlarm, Severity: "major", AlarmEligible: true, Enabled: true},
		{Code: "device_online", Name: "设备上线", Category: CategoryStatus, Severity: "info", Enabled: true},
		{Code: "device_offline", Name: "设备离线", Category: CategoryStatus, Severity: "warning", Enabled: true},
		{Code: "record_abnormal", Name: "录像异常", Category: CategoryStatus, Severity: "warning", AlarmEligible: true, Enabled: true},
		{Code: "ptz", Name: "云台操作", Category: CategoryOperation, Severity: "info", Enabled: true},
		{Code: "config_change", Name: "配置变更", Category: CategoryOperation, Severity: "info", Enabled: true},
		{Code: "metadata", Name: "附属数据", Category: CategoryMetadata, Severity: "info", BBoxCapable: true, Enabled: true},
		{Code: "other", Name: "其他", Category: CategoryAlarm, Severity: "warning", AlarmEligible: true, BBoxCapable: true, Enabled: true},
	}
}

// IsKnown 判断是否为平台已登记的规范事件类型（用于事件分流：已知→zws_event，未知→自定义事件）。
func IsKnown(code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	for _, t := range Builtins() {
		if t.Code == code {
			return true
		}
	}
	return false
}

// Normalize 返回规范类型；未登记类型回落 other。
func Normalize(code string) Type {
	code = strings.ToLower(strings.TrimSpace(code))
	for _, t := range Builtins() {
		if t.Code == code {
			return t
		}
	}
	t := Type{
		Code: "other", Name: "其他", Category: CategoryAlarm, Severity: "warning",
		AlarmEligible: true, BBoxCapable: true, Enabled: true,
	}
	return t
}

// IsAlarmEligible 判断事件是否应投影为报警。
func IsAlarmEligible(code string) bool {
	return Normalize(code).AlarmEligible
}

// LegacyAlarmType 返回旧版 zws_alarm.alarm_type 数字码（兼容旧页面）。
func LegacyAlarmType(code string) int {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "video_loss":
		return 1
	case "tamper":
		return 2
	case "motion":
		return 7
	case "line_crossing":
		return 10
	case "intrusion":
		return 11
	case "loitering":
		return 15
	case "audio_abnormal":
		return 22
	case "defocus":
		return 25
	case "scene_change":
		return 26
	case "face":
		return 27
	case "plate":
		return 28
	case "alarm_input":
		return 30
	default:
		return 99
	}
}

// FromGBAlarmType 把 GB 报警类型码映射为规范码。
func FromGBAlarmType(v int) string {
	switch v {
	case 1:
		return "video_loss"
	case 2:
		return "tamper"
	case 7:
		return "motion"
	case 10:
		return "line_crossing"
	case 11:
		return "intrusion"
	case 15:
		return "loitering"
	case 22:
		return "audio_abnormal"
	case 25:
		return "defocus"
	case 26:
		return "scene_change"
	default:
		return "other"
	}
}
