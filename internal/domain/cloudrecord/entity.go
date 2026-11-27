package cloudrecord

// Record 云端录像记录（领域实体）。
type Record struct {
	ID              int     `json:"id"`
	App             string  `json:"app"`
	Stream          string  `json:"stream"`
	CallID          string  `json:"callId"`
	StartTime       int64   `json:"startTime"`
	EndTime         int64   `json:"endTime"`
	MediaServerID   string  `json:"mediaServerId"`
	ServerID        string  `json:"serverId"`
	FileName        string  `json:"fileName"`
	Folder          string  `json:"folder"`
	FilePath        string  `json:"filePath"`
	PlayURL         string  `json:"playUrl"`
	Collect         bool    `json:"collect"`
	FileSize        int64   `json:"fileSize"`
	TimeLen         float64 `json:"timeLen"`
	StorageNodeID   string  `json:"storageNodeId"`
	ContainerFormat string  `json:"containerFormat"` // mp4 | ps
	SlotID          int64   `json:"slotId"`
	Status          int     `json:"status"` // 1 录制中/2 正常/3 残片恢复/4 已清理
}

type Repository interface {
	Create(rec *Record) error
	UpsertPs(rec *Record) error
	List(page, count int, app, stream, query, callID, mediaServerID string, startTime, endTime int64, asc bool) ([]Record, int64, error)
	ListSegments(stream string, startMs, endMs int64) ([]Record, error)
	DateList(app, stream, mediaServerID string, year, month int) ([]string, error)
	ListByIDs(ids []int) ([]Record, error)
	GetByID(id int) (*Record, error)
	Delete(ids []int) error
}
