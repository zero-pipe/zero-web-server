package event

// Event 是平台持久化的事件记录。
type Event struct {
	ID                  string           `json:"id"`
	EventID             string           `json:"eventId"`
	SchemaVersion       string           `json:"schemaVersion"`
	SourceID            string           `json:"sourceId"`
	DeviceID            string           `json:"deviceId"`
	DeviceName          string           `json:"deviceName"`
	ChannelName         string           `json:"channelName"`
	BindingID           string           `json:"bindingId"`
	ResourceID          string           `json:"resourceId"`
	DeviceInternalCode  string           `json:"deviceInternalCode"`
	DeviceUID           string           `json:"deviceUid"`
	ChannelExternalKey  string           `json:"channelExternalKey"`
	ChannelInternalCode string           `json:"channelInternalCode"`
	EventType           string           `json:"eventType"`
	Severity            string           `json:"severity"`
	OccurredAt          int64            `json:"occurredAt"`
	ReceivedAt          int64            `json:"receivedAt"`
	ImageKey            string           `json:"imageKey"`
	ImageURL            string           `json:"imageUrl"`
	ImageExpiresAt      int64            `json:"imageExpiresAt"`
	BBoxJSON            string           `json:"bboxJson"`
	RawJSON             string           `json:"rawJson"`
	Status              string           `json:"status"` // mapped | unmapped | invalid
	AlarmID             int              `json:"alarmId"`
	Category            string           `json:"category,omitempty"`
	Playback            *PlaybackContext `json:"playback,omitempty"`
	CreateTime          string           `json:"createTime"`
}

// PlaybackContext 由事件关联的设备/通道推导，供报警回放联动。
type PlaybackContext struct {
	Supported       bool   `json:"supported"`
	Protocol        string `json:"protocol"`
	DeviceExternal  string `json:"deviceExternal,omitempty"`
	ChannelExternal string `json:"channelExternal,omitempty"`
	ChannelLegacyID int    `json:"channelLegacyId,omitempty"`
	StartAt         int64  `json:"startAt"`
	EndAt           int64  `json:"endAt"`
	Reason          string `json:"reason,omitempty"`
}

type ListFilter struct {
	DeviceID           string
	DeviceInternalCode string
	DeviceName         string
	EventType          string
	Status             string
	Begin              int64
	End                int64
}

type Repository interface {
	Create(e *Event) error
	Exists(eventID string) (bool, error)
	Get(id string) (*Event, error)
	List(page, count int, f ListFilter) ([]*Event, int64, error)
	DeleteBefore(before int64) (int64, error)
	// DeleteRange 按时间范围删除；all=true 时忽略范围删除全部。
	DeleteRange(begin, end int64, all bool) (int64, error)
	UpdateAlarmID(id string, alarmID int) error
}
