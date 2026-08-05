package position

type MobilePosition struct {
	ID        int     `json:"id"`
	ChannelID int     `json:"channelId"`
	Timestamp int64   `json:"timestamp"`
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Altitude  float64 `json:"altitude"`
	Speed     float64 `json:"speed"`
	Direction float64 `json:"direction"`
	CreateTime string `json:"createTime"`
}

type Repository interface {
	Create(pos *MobilePosition) error
	BatchCreate(list []*MobilePosition) error
	ListByChannel(channelID int, start, end int64) ([]*MobilePosition, error)
	Latest(channelID int) (*MobilePosition, error)
}
