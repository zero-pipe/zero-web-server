package alarm

type Alarm struct {
	ID                  int     `json:"id"`
	ChannelID           int     `json:"channelId"`
	Description         string  `json:"description"`
	SnapPath            string  `json:"snapPath"`
	RecordPath          string  `json:"recordPath"`
	Longitude           float64 `json:"longitude"`
	Latitude            float64 `json:"latitude"`
	AlarmType           int     `json:"alarmType"`
	AlarmCode           string  `json:"alarmCode"`
	AlarmTime           int64   `json:"alarmTime"`
	EventID             string  `json:"eventId"`
	DeviceInternalCode  string  `json:"deviceInternalCode"`
	ChannelInternalCode string  `json:"channelInternalCode"`
	BBoxJSON            string  `json:"bboxJson"`
	Severity            string  `json:"severity"`
	Status              string  `json:"status"`
	HandledBy           string  `json:"handledBy"`
	HandledAt           int64   `json:"handledAt"`
	Remark              string  `json:"remark"`
	ChannelName         string  `json:"channelName,omitempty"`
	ChannelDeviceID     string  `json:"channelDeviceId,omitempty"`
	DeviceName          string  `json:"deviceName,omitempty"`
	Playback            any     `json:"playback,omitempty"`
}

type Repository interface {
	Create(alarm *Alarm) error
	GetByID(id int) (*Alarm, error)
	List(page, count int, alarmType *int, beginTime, endTime int64) ([]*Alarm, int64, error)
	Delete(ids []int) error
	Clear(alarmType *int, beginTime, endTime int64) error
	UpdateStatus(id int, status, handledBy, remark string, handledAt int64) error
}
