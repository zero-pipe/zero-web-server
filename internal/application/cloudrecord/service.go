package cloudrecord

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	mediaserverapp "zero-web-kit/internal/application/mediaserver"
	"zero-web-kit/internal/infrastructure/media/mediakit"
	"zero-web-kit/internal/infrastructure/persistence"
	"zero-web-kit/internal/infrastructure/persistence/model"
	"zero-web-kit/internal/interfaces/http/dto"
)

type Service struct {
	repo         *persistence.CloudRecordRepository
	mediaServers *mediaserverapp.Service
	serverID     string
	sessions     sync.Map // 保留：兼容旧 FLV load 会话（现已不走）
}

func NewService(repo *persistence.CloudRecordRepository, mediaServers *mediaserverapp.Service, serverID string) *Service {
	return &Service{repo: repo, mediaServers: mediaServers, serverID: serverID}
}

func (s *Service) OnRecordMp4(param RecordHookParam) error {
	mediaID := param.MediaServerID
	if mediaID == "" {
		if node, err := s.mediaServers.SelectMinimumLoad(); err == nil {
			mediaID = node.ID()
		}
	}
	// ZMS/ZLM hook: start_time 为秒，time_len 为秒；前端按毫秒展示/播放
	startMs, timeLenMs := normalizeRecordTimes(param.StartTime, param.TimeLen)
	playURL := strings.TrimSpace(param.URL)
	if playURL == "" {
		playURL = s.synthesizePlayURL(mediaID, param.FilePath)
	} else {
		playURL = s.rewritePlayURLHost(mediaID, playURL)
	}
	rec := &model.CloudRecord{
		App:           param.App,
		Stream:        param.Stream,
		StartTime:     startMs,
		EndTime:       startMs + int64(timeLenMs),
		MediaServerID: mediaID,
		ServerID:      s.serverID,
		FileName:      param.FileName,
		Folder:        param.Folder,
		FilePath:      param.FilePath,
		PlayURL:       playURL,
		FileSize:      param.FileSize,
		TimeLen:       timeLenMs,
	}
	if param.CallID != "" {
		rec.CallID = param.CallID
	}
	return s.repo.Create(rec)
}

// normalizeRecordTimes 统一为毫秒；兼容库内已按秒写入的旧数据（列表展示时也会再规范化）。
func normalizeRecordTimes(startSecOrMs int64, timeLenSecOrMs float64) (startMs int64, timeLenMs float64) {
	startMs = startSecOrMs
	if startMs > 0 && startMs < 1_000_000_000_000 { // < 2001-09-09 in ms → 按秒
		startMs *= 1000
	}
	timeLenMs = timeLenSecOrMs
	if timeLenMs > 0 && timeLenMs < 100_000 { // 切片通常 < 1 天(秒)；已是毫秒则 >= 100000
		timeLenMs *= 1000
	}
	return startMs, timeLenMs
}

func normalizeCloudRecord(rec *model.CloudRecord) {
	if rec == nil {
		return
	}
	startMs, timeLenMs := normalizeRecordTimes(rec.StartTime, rec.TimeLen)
	rec.StartTime = startMs
	rec.TimeLen = timeLenMs
	if rec.EndTime > 0 && rec.EndTime < 1_000_000_000_000 {
		rec.EndTime *= 1000
	}
	if rec.EndTime <= 0 && startMs > 0 {
		rec.EndTime = startMs + int64(timeLenMs)
	}
}

type RecordHookParam struct {
	App           string
	Stream        string
	FileName      string
	FilePath      string
	URL           string // HTTP-MP4 点播地址（ZMS on_record_mp4.url）
	FileSize      int64
	Folder        string
	StartTime     int64
	TimeLen       float64
	CallID        string
	MediaServerID string
}

func (s *Service) List(page, count int, app, stream, query, callID, mediaServerID string, startTime, endTime int64, asc bool) ([]model.CloudRecord, int64, error) {
	rows, total, err := s.repo.List(page, count, app, stream, query, callID, mediaServerID, startTime, endTime, asc)
	if err != nil {
		return nil, 0, err
	}
	for i := range rows {
		normalizeCloudRecord(&rows[i])
		s.ensurePlayURL(&rows[i])
	}
	return rows, total, nil
}

func (s *Service) DateList(app, stream, mediaServerID string, year, month int) ([]string, error) {
	return s.repo.DateList(app, stream, mediaServerID, year, month)
}

func (s *Service) Delete(ids []int) error {
	rows, err := s.repo.ListByIDs(ids)
	if err != nil {
		return err
	}
	for _, rec := range rows {
		if rec.FilePath == "" {
			continue
		}
		node, resolveErr := s.mediaServers.Resolve(rec.MediaServerID)
		if resolveErr != nil {
			node, resolveErr = s.mediaServers.SelectMinimumLoad()
		}
		if resolveErr != nil || node == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		_, _ = node.Client.DeleteRecordFile(ctx, rec.FilePath)
		cancel()
	}
	return s.repo.Delete(ids)
}

func (s *Service) GetPlayPath(recordID int) (map[string]string, error) {
	rec, err := s.repo.GetByID(recordID)
	if err != nil {
		return nil, fmt.Errorf("录像不存在")
	}
	node, err := s.mediaServers.Resolve(rec.MediaServerID)
	if err != nil {
		node, err = s.mediaServers.SelectMinimumLoad()
		if err != nil {
			return nil, err
		}
	}
	cfg := node.MediaConfig()
	base := cfg.BaseURL()
	download := fmt.Sprintf("%s/index/api/downloadFile?secret=%s&file_path=%s",
		base, url.QueryEscape(cfg.Secret), url.QueryEscape(rec.FilePath))
	s.ensurePlayURL(rec)
	out := map[string]string{
		"httpPath":  download,
		"httpsPath": download,
		"download":  download,
		"filePath":  rec.FilePath,
	}
	if rec.PlayURL != "" {
		out["playUrl"] = rec.PlayURL
		out["mp4"] = rec.PlayURL
	}
	return out, nil
}

// LoadRecord 云录像点播：直接返回已落库的 HTTP-MP4 URL，不再 loadMP4File/转 FLV。
func (s *Service) LoadRecord(ctx context.Context, app, stream string, cloudRecordID int) (*dto.StreamContent, error) {
	_ = ctx
	rec, err := s.repo.GetByID(cloudRecordID)
	if err != nil {
		return nil, fmt.Errorf("录像不存在")
	}
	normalizeCloudRecord(rec)
	s.ensurePlayURL(rec)
	if rec.PlayURL == "" {
		return nil, fmt.Errorf("录像播放地址为空")
	}
	if app == "" {
		app = rec.App
	}
	if stream == "" {
		stream = rec.Stream
	}
	node, err := s.mediaServers.Resolve(rec.MediaServerID)
	if err != nil {
		node, err = s.mediaServers.SelectMinimumLoad()
		if err != nil {
			return nil, err
		}
	}
	content := &dto.StreamContent{
		App:           app,
		Stream:        stream,
		IP:            node.StreamIP(),
		Mp4:           rec.PlayURL,
		Flv:           rec.PlayURL, // 兼容旧播放器字段；前端优先 mp4 + 原生 video
		MediaServerID: node.ID(),
		ServerID:      s.serverID,
		Progress:      rec.TimeLen,
		Duration:      rec.TimeLen,
	}
	return content, nil
}

func (s *Service) Seek(app, stream, mediaServerID string, seek float64, schema string) error {
	// HTTP-MP4 由浏览器 Range/currentTime 本地 seek，无需 ZMS demux seek
	_ = app
	_ = stream
	_ = mediaServerID
	_ = seek
	_ = schema
	return nil
}

func (s *Service) Speed(app, stream, mediaServerID string, speed int, schema string) error {
	_ = app
	_ = stream
	_ = mediaServerID
	_ = speed
	_ = schema
	return nil
}

func (s *Service) AddTask(app, stream, mediaServerID, startTime, endTime string) (string, error) {
	return "", fmt.Errorf("未配置 RecordAssist 服务，暂不支持云端录像合并任务")
}

func (s *Service) QueryTaskList(mediaServerID string, isEnd *bool) ([]any, error) {
	return []any{}, nil
}

func (s *Service) OnStreamStarted(app, stream string) {
	key := streamKey(app, stream)
	if v, ok := s.sessions.Load(key); ok {
		if ch, ok := v.(chan *dto.StreamContent); ok {
			node, _ := s.mediaServers.ResolveForStream(app, stream, "auto")
			select {
			case ch <- s.buildStreamContent(app, stream, node):
			default:
			}
		}
	}
}

func (s *Service) buildStreamContent(app, stream string, node *mediaserverapp.Node) *dto.StreamContent {
	if node == nil {
		return &dto.StreamContent{App: app, Stream: stream, ServerID: s.serverID}
	}
	urls := mediakit.BuildPlayURLsFromConfig(node.MediaConfig(), app, stream)
	return &dto.StreamContent{
		App: app, Stream: stream, IP: node.StreamIP(),
		Flv: urls["flv"], WsFlv: urls["ws"], Hls: urls["hls"],
		Rtmp: urls["rtmp"], Rtsp: urls["rtsp"],
		MediaServerID: node.ID(), ServerID: s.serverID,
	}
}

func (s *Service) ensurePlayURL(rec *model.CloudRecord) {
	if rec == nil || rec.PlayURL != "" {
		return
	}
	rec.PlayURL = s.synthesizePlayURL(rec.MediaServerID, rec.FilePath)
}

func (s *Service) synthesizePlayURL(mediaServerID, filePath string) string {
	rel := filePathToRecordHTTPRel(filePath)
	if rel == "" {
		return ""
	}
	node, err := s.mediaServers.Resolve(mediaServerID)
	if err != nil {
		node, err = s.mediaServers.SelectMinimumLoad()
		if err != nil || node == nil {
			return ""
		}
	}
	return strings.TrimRight(node.MediaConfig().BaseURL(), "/") + "/" + rel
}

// rewritePlayURLHost 用平台媒体节点流 IP 替换 ZMS 回调里的 host（避免 127.0.0.1）。
func (s *Service) rewritePlayURLHost(mediaServerID, playURL string) string {
	u, err := url.Parse(playURL)
	if err != nil || u.Path == "" {
		return playURL
	}
	node, err := s.mediaServers.Resolve(mediaServerID)
	if err != nil {
		node, err = s.mediaServers.SelectMinimumLoad()
		if err != nil || node == nil {
			return playURL
		}
	}
	base := strings.TrimRight(node.MediaConfig().BaseURL(), "/")
	return base + u.Path
}

func filePathToRecordHTTPRel(filePath string) string {
	if filePath == "" {
		return ""
	}
	norm := strings.ReplaceAll(filePath, "\\", "/")
	idx := strings.Index(norm, "/record/")
	if idx >= 0 {
		return strings.TrimPrefix(norm[idx:], "/")
	}
	if strings.HasPrefix(norm, "record/") {
		return norm
	}
	if strings.HasPrefix(norm, "./record/") {
		return norm[2:]
	}
	return ""
}

func streamKey(app, stream string) string { return app + "/" + stream }
