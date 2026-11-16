package streampush

// StreamPush 推流任务（领域实体）。
type StreamPush struct {
	ID               int
	App              string
	Stream           string
	MediaServerID    string
	ServerID         string
	PushTime         string
	Status           bool
	UpdateTime       string
	CreateTime       string
	Pushing          bool
	StartOfflinePush bool
}

// View 推流列表视图。
type View struct {
	ID               int    `json:"id"`
	App              string `json:"app"`
	Stream           string `json:"stream"`
	MediaServerID    string `json:"mediaServerId"`
	ServerID         string `json:"serverId"`
	PushTime         string `json:"pushTime"`
	CreateTime       string `json:"createTime"`
	UpdateTime       string `json:"updateTime"`
	Pushing          bool   `json:"pushing"`
	StartOfflinePush bool   `json:"startOfflinePush"`
	GbID             int    `json:"gbId"`
	GbDeviceID       string `json:"gbDeviceId"`
	GbName           string `json:"gbName"`
	Name             string `json:"name"`
	DataType         int    `json:"dataType"`
}

type Repository interface {
	List(page, count int, query string, pushing *bool, mediaServerID string) ([]View, int64, error)
	Create(m *StreamPush) error
	Update(m *StreamPush) error
	Delete(id int) error
	GetByID(id int) (*StreamPush, error)
	GetByAppStream(app, stream string) (*StreamPush, error)
	UpdatePushing(id int, pushing bool, mediaServerID string) error
	UpsertGBChannel(pushID int, gbDeviceID, gbName string) error
	RemoveGBChannel(pushID int) error
}
