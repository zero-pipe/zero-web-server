package channel

type Channel struct {
	ID           int     `json:"id"`
	DeviceID     string  `json:"deviceId"`
	DataType     int     `json:"dataType"`
	DataDeviceID int     `json:"dataDeviceId"`
	GBDeviceID   string  `json:"gbDeviceId"`
	Name         string  `json:"name"`
	Manufacturer string  `json:"manufacturer"`
	Model        string  `json:"model"`
	Status       string  `json:"status"`
	PTZType      int     `json:"ptzType"`
	Parental     int     `json:"parental"`
	ParentID     string  `json:"parentId"`
	Longitude    float64 `json:"longitude"`
	Latitude     float64 `json:"latitude"`
	HasAudio     bool    `json:"hasAudio"`
	SubCount     int     `json:"subCount"`
	CreateTime   string  `json:"createTime"`
	UpdateTime   string  `json:"updateTime"`
}

type Repository interface {
	GetOne(deviceID, channelDeviceID string) (*Channel, error)
	GetByID(id int) (*Channel, error)
	ListByDevice(deviceID string, page, count int, query string, online *bool) ([]*Channel, int64, error)
	ResetByDevice(deviceID string, dataDeviceID int, channels []*Channel) error
	DeleteByDevice(deviceID string) error
	ChangeAudio(channelID int, audio bool) error
}
