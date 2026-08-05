package device

type Device struct {
	ID                              int    `json:"id"`
	InternalCode                    string `json:"internalCode"`
	DeviceID                        string `json:"deviceId"`
	Name                            string `json:"name"`
	Manufacturer                    string `json:"manufacturer"`
	Model                           string `json:"model"`
	Firmware                        string `json:"firmware"`
	Transport                       string `json:"transport"`
	StreamMode                      string `json:"streamMode"`
	IP                              string `json:"ip"`
	Port                            int    `json:"port"`
	HostAddress                     string `json:"hostAddress"`
	OnLine                          bool   `json:"onLine"`
	Charset                         string `json:"charset"`
	Expires                         int    `json:"expires"`
	Password                        string `json:"password,omitempty"`
	MediaServerID                   string `json:"mediaServerId"`
	CustomName                      string `json:"customName"`
	SDPIP                           string `json:"sdpIp"`
	LocalIP                         string `json:"localIp"`
	ServerID                        string `json:"serverId"`
	HeartBeatInterval               int    `json:"heartBeatInterval"`
	HeartBeatCount                  int    `json:"heartBeatCount"`
	SubscribeCycleForCatalog        int    `json:"subscribeCycleForCatalog"`
	SubscribeCycleForMobilePosition int    `json:"subscribeCycleForMobilePosition"`
	MobilePositionSubmissionInterval int   `json:"mobilePositionSubmissionInterval"`
	SubscribeCycleForAlarm          int    `json:"subscribeCycleForAlarm"`
	CreateTime                      string `json:"createTime"`
	UpdateTime                      string `json:"updateTime"`
	ChannelCount                    int    `json:"channelCount"`
	RegisterCallID                  string `json:"-"`
}

type TimeStatistics struct {
	Time     string `json:"time"`
	TimeDiff int64  `json:"timeDiff"`
}

type Repository interface {
	GetByDeviceID(deviceID string) (*Device, error)
	GetByID(id int) (*Device, error)
	Create(device *Device) error
	Update(device *Device) error
	UpdateOnline(deviceID string, online bool) error
	DeleteByDeviceID(deviceID string) error
	List(page, count int, query string, online *bool) ([]*Device, int64, error)
	ListOnline() ([]*Device, error)
	GetByInternalCode(code string) (*Device, error)
}
