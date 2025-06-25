package alarm

type Alarm struct {
	ID          int     `json:"id"`
	ChannelID   int     `json:"channelId"`
	Description string  `json:"description"`
	SnapPath    string  `json:"snapPath"`
	RecordPath  string  `json:"recordPath"`
	Longitude   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
	AlarmType   int     `json:"alarmType"`
	AlarmTime   int64   `json:"alarmTime"`
}

type Repository interface {
	Create(alarm *Alarm) error
	List(page, count int, alarmType *int, beginTime, endTime int64) ([]*Alarm, int64, error)
	Delete(ids []int) error
	Clear(alarmType *int, beginTime, endTime int64) error
}
