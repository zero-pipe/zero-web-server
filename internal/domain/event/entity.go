package event

// Event 是平台持久化的事件记录。
type Event struct {
	ID                 string `json:"id"`
	EventID            string `json:"eventId"`
	SchemaVersion      string `json:"schemaVersion"`
	SourceID           string `json:"sourceId"`
	DeviceID           string `json:"deviceId"`
	DeviceName         string `json:"deviceName"`
	BindingID          string `json:"bindingId"`
	ResourceID         string `json:"resourceId"`
	DeviceInternalCode string `json:"deviceInternalCode"`
	DeviceUID          string `json:"deviceUid"`
	ChannelExternalKey string `json:"channelExternalKey"`
	EventType          string `json:"eventType"`
	Severity           string `json:"severity"`
	OccurredAt         int64  `json:"occurredAt"`
	ReceivedAt         int64  `json:"receivedAt"`
	ImageKey           string `json:"imageKey"`
	ImageURL           string `json:"imageUrl"`
	ImageExpiresAt     int64  `json:"imageExpiresAt"`
	BBoxJSON           string `json:"bboxJson"`
	RawJSON            string `json:"rawJson"`
	Status             string `json:"status"` // mapped | unmapped | invalid
	AlarmID            int    `json:"alarmId"`
	CreateTime         string `json:"createTime"`
}

type ListFilter struct {
	DeviceID           string
	DeviceInternalCode string
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
}
