package cloudrecord

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"zero-web-kit/internal/infrastructure/persistence"
	"zero-web-kit/internal/infrastructure/persistence/model"
	"zero-web-kit/internal/interfaces/http/dto"
	"zero-web-kit/internal/port"
)

type Service struct {
	repo     *persistence.CloudRecordRepository
	media    port.MediaCluster
	serverID string
	sessions sync.Map
}

func NewService(repo *persistence.CloudRecordRepository, media port.MediaCluster, serverID string) *Service {
	return &Service{repo: repo, media: media, serverID: serverID}
}

func (s *Service) OnRecordMp4(param RecordHookParam) error {
	mediaID := param.MediaServerID
	if mediaID == "" {
		if node, err := s.media.SelectMinimumLoad(context.Background()); err == nil {
			mediaID = node.ID()
		}
	}
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

func normalizeRecordTimes(startSecOrMs int64, timeLenSecOrMs float64) (startMs int64, timeLenMs float64) {
	startMs = startSecOrMs
	if startMs > 0 && startMs < 1_000_000_000_000 {
		startMs *= 1000
	}
	timeLenMs = timeLenSecOrMs
	if timeLenMs > 0 && timeLenMs < 100_000 {
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
	URL           string
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
		node, resolveErr := s.resolveForRecord(rec.MediaServerID)
		if resolveErr != nil || node == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		_ = node.DeleteRecordFile(ctx, rec.FilePath)
		cancel()
	}
	return s.repo.Delete(ids)
}

// resolveForRecord 点播/拼 URL：优先在线节点，Ping 不通时仍用已登记配置（不阻断回放）。
func (s *Service) resolveForRecord(preferID string) (port.MediaEndpoint, error) {
	ctx := context.Background()
	if node, err := s.media.Resolve(ctx, preferID); err == nil {
		return node, nil
	}
	if node, err := s.media.SelectMinimumLoad(ctx); err == nil {
		return node, nil
	}
	if node, err := s.media.Lookup(ctx, preferID); err == nil {
		return node, nil
	}
	return s.media.Lookup(ctx, "")
}

func (s *Service) GetPlayPath(recordID int) (map[string]string, error) {
	rec, err := s.repo.GetByID(recordID)
	if err != nil {
		return nil, fmt.Errorf("录像不存在")
	}
	s.ensurePlayURL(rec)
	out := map[string]string{
		"filePath": rec.FilePath,
	}
	if rec.PlayURL != "" {
		out["playUrl"] = rec.PlayURL
		out["mp4"] = rec.PlayURL
	}
	node, err := s.resolveForRecord(rec.MediaServerID)
	if err != nil {
		if rec.PlayURL != "" {
			out["httpPath"] = rec.PlayURL
			out["httpsPath"] = rec.PlayURL
			out["download"] = rec.PlayURL
			return out, nil
		}
		return nil, err
	}
	base := node.BaseURL()
	download := fmt.Sprintf("%s/index/api/downloadFile?secret=%s&file_path=%s",
		base, url.QueryEscape(node.Secret()), url.QueryEscape(rec.FilePath))
	out["httpPath"] = download
	out["httpsPath"] = download
	out["download"] = download
	return out, nil
}

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
	out := &dto.StreamContent{
		App:           app,
		Stream:        stream,
		Mp4:           rec.PlayURL,
		Flv:           rec.PlayURL,
		MediaServerID: rec.MediaServerID,
		ServerID:      s.serverID,
		Progress:      rec.TimeLen,
		Duration:      rec.TimeLen,
	}
	if node, err := s.resolveForRecord(rec.MediaServerID); err == nil && node != nil {
		out.IP = node.StreamIP()
		out.MediaServerID = node.ID()
	} else if u, parseErr := url.Parse(rec.PlayURL); parseErr == nil {
		out.IP = u.Hostname()
	}
	return out, nil
}

func (s *Service) Seek(app, stream, mediaServerID string, seek float64, schema string) error {
	_, _, _, _, _ = app, stream, mediaServerID, seek, schema
	return nil
}

func (s *Service) Speed(app, stream, mediaServerID string, speed int, schema string) error {
	_, _, _, _, _ = app, stream, mediaServerID, speed, schema
	return nil
}

func (s *Service) AddTask(app, stream, mediaServerID, startTime, endTime string) (string, error) {
	return "", fmt.Errorf("暂不支持云端录像合并任务")
}

func (s *Service) QueryTaskList(mediaServerID string, isEnd *bool) ([]any, error) {
	return []any{}, nil
}

func (s *Service) OnStreamStarted(app, stream string) {
	key := streamKey(app, stream)
	if v, ok := s.sessions.Load(key); ok {
		if ch, ok := v.(chan *dto.StreamContent); ok {
			node, _ := s.media.ResolveForStream(context.Background(), app, stream, "auto")
			select {
			case ch <- s.buildStreamContent(app, stream, node):
			default:
			}
		}
	}
}

func (s *Service) buildStreamContent(app, stream string, node port.MediaEndpoint) *dto.StreamContent {
	if node == nil {
		return &dto.StreamContent{App: app, Stream: stream, ServerID: s.serverID}
	}
	urls := node.PlayURLs(app, stream)
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
	node, err := s.resolveForRecord(mediaServerID)
	if err != nil || node == nil {
		return ""
	}
	return strings.TrimRight(node.BaseURL(), "/") + "/" + rel
}

func (s *Service) rewritePlayURLHost(mediaServerID, playURL string) string {
	u, err := url.Parse(playURL)
	if err != nil || u.Path == "" {
		return playURL
	}
	node, err := s.resolveForRecord(mediaServerID)
	if err != nil || node == nil {
		return playURL
	}
	base := strings.TrimRight(node.BaseURL(), "/")
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
