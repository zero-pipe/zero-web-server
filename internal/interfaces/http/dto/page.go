package dto

type PageInfo[T any] struct {
	Total    int64 `json:"total"`
	List     []T   `json:"list"`
	PageNum  int   `json:"pageNum"`
	PageSize int   `json:"pageSize"`
	Pages    int   `json:"pages"`
}

func NewPageInfo[T any](list []T, total int64, page, count int) PageInfo[T] {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	pages := int(total) / count
	if int(total)%count > 0 {
		pages++
	}
	return PageInfo[T]{
		Total:    total,
		List:     list,
		PageNum:  page,
		PageSize: count,
		Pages:    pages,
	}
}

type StreamContent struct {
	App           string  `json:"app"`
	Stream        string  `json:"stream"`
	IP            string  `json:"ip"`
	Flv           string  `json:"flv"`
	WsFlv         string  `json:"ws_flv"`
	Hls           string  `json:"hls"`
	Rtmp          string  `json:"rtmp"`
	Rtsp          string  `json:"rtsp"`
	Rtc           string  `json:"rtc"`
	Rtcs          string  `json:"rtcs"`
	VideoCodec    string  `json:"videoCodec,omitempty"`
	AudioCodec    string  `json:"audioCodec,omitempty"`
	MediaServerID string  `json:"mediaServerId"`
	ServerID      string  `json:"serverId"`
	Progress      float64 `json:"progress"`
	Duration      float64 `json:"duration"` // 云端录像点播总时长（毫秒），前端进度条用
	Mp4           string  `json:"mp4,omitempty"` // HTTP-MP4 直出（云录像优先）
}

type AudioBroadcastResult struct {
	StreamInfo     *StreamContent `json:"streamInfo"`
	PlayStreamInfo *StreamContent `json:"playStreamInfo,omitempty"`
	Codec          string         `json:"codec"`
	App            string         `json:"app"`
	Stream         string         `json:"stream"`
}

type SyncStatus struct {
	Total    int    `json:"total"`
	Current  int    `json:"current"`
	SyncIng  bool   `json:"syncIng"`
	ErrorMsg string `json:"errorMsg"`
}
