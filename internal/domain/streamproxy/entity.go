package streamproxy

// StreamProxy 拉流代理（领域实体）。
type StreamProxy struct {
	ID                      int
	Type                    string
	App                     string
	Stream                  string
	SrcURL                  string
	Timeout                 int
	FFmpegCmdKey            string
	RTSPType                string
	MediaServerID           string
	EnableAudio             bool
	EnableMP4               bool
	Pulling                 bool
	Enable                  bool
	CreateTime              string
	Name                    string
	UpdateTime              string
	StreamKey               string
	ServerID                string
	EnableDisableNoneReader bool
	RelatesMediaServerID    string
}

type Repository interface {
	List(page, count int, query string, pulling *bool, mediaServerID string) ([]StreamProxy, int64, error)
	Create(m *StreamProxy) error
	Update(m *StreamProxy) error
	Delete(id int) error
	GetByID(id int) (*StreamProxy, error)
	GetByAppStream(app, stream string) (*StreamProxy, error)
	UpsertGBChannel(proxyID int, gbDeviceID, gbName, app, stream string) error
}
