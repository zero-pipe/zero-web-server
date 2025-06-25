package onvif

import "context"

type Device struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	IP            string `json:"ip"`
	Port          int    `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Manufacturer  string `json:"manufacturer"`
	Model         string `json:"model"`
	Firmware      string `json:"firmware"`
	SerialNumber  string `json:"serialNumber"`
	HardwareID    string `json:"hardwareId"`
	DeviceURI     string `json:"deviceUri"`
	MediaURI      string `json:"mediaUri"`
	PTZURI        string `json:"ptzUri"`
	OnLine        bool   `json:"onLine"`
	DiscoveryMode int    `json:"discoveryMode"`
	MediaServerID string `json:"mediaServerId"`
	CustomName    string `json:"customName"`
	ServerID      string `json:"serverId"`
	CreateTime    string `json:"createTime"`
	UpdateTime    string `json:"updateTime"`
}

type Channel struct {
	ID            int64  `json:"id"`
	DeviceID      int64  `json:"deviceId"`
	ProfileToken  string `json:"profileToken"`
	Name          string `json:"name"`
	VideoSource   string `json:"videoSource"`
	EncoderToken  string `json:"encoderToken"`
	Resolution    string `json:"resolution"`
	Codec         string `json:"codec"`
	ConfigCodec   string `json:"configCodec"`
	StreamChannel string `json:"streamChannel"`
	StreamType    string `json:"streamType"`
	HasAudio      bool   `json:"hasAudio"`
	HasPTZ        bool   `json:"hasPtz"`
	StreamURI     string `json:"streamUri"`
	Status        string `json:"status"`
	CreateTime    string `json:"createTime"`
	UpdateTime    string `json:"updateTime"`
}

type DiscoveredDevice struct {
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Endpoint string `json:"endpoint"`
	Location string `json:"location"`
}

type DeviceRepository interface {
	Create(ctx context.Context, device *Device) error
	Update(ctx context.Context, device *Device) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*Device, error)
	List(ctx context.Context, page, count int, keyword string) ([]*Device, int64, error)
	ExistsByIPPort(ctx context.Context, ip string, port int) (bool, error)
	UpdateOnlineStatus(ctx context.Context, id int64, online bool) error
}

type ChannelRepository interface {
	DeleteByDeviceID(ctx context.Context, deviceID int64) error
	BatchCreate(ctx context.Context, channels []*Channel) error
	ListByDeviceID(ctx context.Context, deviceID int64) ([]*Channel, error)
	GetByID(ctx context.Context, id int64) (*Channel, error)
	List(ctx context.Context, page, count int, deviceID int64) ([]*Channel, int64, error)
}
