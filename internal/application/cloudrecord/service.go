package cloudrecord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	storagenodeapp "zero-web-server/internal/application/storagenode"
	domaincloudrecord "zero-web-server/internal/domain/cloudrecord"
	"zero-web-server/internal/dto"
	"zero-web-server/internal/mediaurl"
	"zero-web-server/internal/port"
	"zero-web-server/internal/storagekit"
)

type Service struct {
	repo        domaincloudrecord.Repository
	media       port.MediaCluster
	storageNode *storagenodeapp.Service
	serverID    string
	sessions    sync.Map
}

func NewService(repo domaincloudrecord.Repository, media port.MediaCluster, storageNode *storagenodeapp.Service, serverID string) *Service {
	return &Service{repo: repo, media: media, storageNode: storageNode, serverID: serverID}
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
	rec := &domaincloudrecord.Record{
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

func normalizeCloudRecord(rec *domaincloudrecord.Record) {
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

// RecordPsParam 是 zstorage on_record_ps 封口回调。
type RecordPsParam struct {
	StorageNodeID string
	Channel       string // channel_key = <设备内码>_<通道内码>
	Slot          int64
	StartTime     int64 // UTC ms
	EndTime       int64 // UTC ms
	DurationMs    int64
	FileSize      int64
	FilePath      string
	Status        int
}

// OnRecordPs 把 zstorage 封口的分段按 (node, slot) 幂等写入 zws_cloud_record。
func (s *Service) OnRecordPs(p RecordPsParam) error {
	if p.Channel == "" || p.Slot <= 0 {
		return fmt.Errorf("on_record_ps 缺 channel/slot")
	}
	startMs := p.StartTime
	if startMs > 0 && startMs < 1_000_000_000_000 {
		startMs *= 1000
	}
	endMs := p.EndTime
	if endMs > 0 && endMs < 1_000_000_000_000 {
		endMs *= 1000
	}
	if endMs <= 0 && p.DurationMs > 0 {
		endMs = startMs + p.DurationMs
	}
	timeLen := float64(endMs - startMs)
	if timeLen < 0 {
		timeLen = 0
	}
	rec := &domaincloudrecord.Record{
		App:             "live",
		Stream:          p.Channel,
		StartTime:       startMs,
		EndTime:         endMs,
		TimeLen:         timeLen,
		FileSize:        p.FileSize,
		FilePath:        p.FilePath,
		MediaServerID:   p.StorageNodeID,
		StorageNodeID:   p.StorageNodeID,
		ContainerFormat: "ps",
		SlotID:          p.Slot,
		Status:          p.Status,
		ServerID:        s.serverID,
	}
	if rec.Status == 0 {
		rec.Status = 2
	}
	return s.repo.UpsertPs(rec)
}

func (s *Service) List(page, count int, app, stream, query, callID, mediaServerID string, startTime, endTime int64, asc bool) ([]domaincloudrecord.Record, int64, error) {
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
	download := fmt.Sprintf("%s/index/api/record/download?secret=%s&file_path=%s",
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
	// PS 旁路录像：走存储节点回放（zstorage/playback/start）。
	if rec.StorageNodeID != "" && rec.SlotID > 0 {
		return s.loadPsRecord(rec, 0)
	}
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
	_ = schema
	if s.storageNode == nil || mediaServerID == "" {
		return nil
	}
	node, err := s.storageNode.Resolve(mediaServerID)
	if err != nil || node == nil {
		return nil // 非存储节点，交给播放端本地处理
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := storagekit.NewClient(node.IP, node.HTTPPort, node.Secret).
		PlaybackSeek(ctx, app, stream, int64(seek))
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("存储节点回放拖动失败: %s", resp.Msg)
	}
	return nil
}

// loadPsRecord 打开存储节点上的槽位回放，返回与 ZMS 同构的出流地址。
// offsetMs 为段内相对毫秒（= 绝对起播时刻 - 段 StartTime）。
func (s *Service) loadPsRecord(rec *domaincloudrecord.Record, offsetMs int64) (*dto.StreamContent, error) {
	if s.storageNode == nil {
		return nil, fmt.Errorf("未配置存储节点服务")
	}
	node, err := s.storageNode.Resolve(rec.StorageNodeID)
	if err != nil || node == nil {
		return nil, fmt.Errorf("存储节点不可用: %s", rec.StorageNodeID)
	}
	if offsetMs < 0 {
		offsetMs = 0
	}
	client := storagekit.NewClient(node.IP, node.HTTPPort, node.Secret)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.PlaybackStart(ctx, rec.SlotID, rec.Stream, offsetMs)
	if err != nil {
		return nil, fmt.Errorf("存储节点回放失败: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("存储节点回放失败: %s", resp.Msg)
	}
	var data struct {
		App    string `json:"app"`
		Stream string `json:"stream"`
	}
	_ = json.Unmarshal(resp.Data, &data)
	app, stream := data.App, data.Stream
	if app == "" {
		app = "playback"
	}
	if stream == "" {
		stream = rec.Stream
	}
	urls := client.BuildMediaURLs(app, stream, node.HTTPPort, node.RTSPPort)
	return &dto.StreamContent{
		App: app, Stream: stream, IP: node.IP,
		Flv: urls["flv"], WsFlv: urls["ws_flv"], Rtsp: urls["rtsp"],
		MediaServerID: node.NodeID, ServerID: s.serverID,
		Progress: float64(offsetMs), Duration: rec.TimeLen,
	}, nil
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

func (s *Service) ensurePlayURL(rec *domaincloudrecord.Record) {
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

// ---- 播放器回放链路：按内码查片段 + 绝对时间起播/拖动 ----

// SegmentView 是播放器回放时间轴上的一个云端片段（PS 或 MP4）。
type SegmentView struct {
	RecordID   int    `json:"recordId"`
	Stream     string `json:"stream"`
	StartTime  int64  `json:"startTime"` // 绝对 ms
	EndTime    int64  `json:"endTime"`   // 绝对 ms
	DurationMs int64  `json:"durationMs"`
	Container  string `json:"container"` // ps | mp4
	Size       int64  `json:"size"`
	Online     bool   `json:"online"`
}

// Segments 按通道内码查云端片段（PS+MP4，按开始时间升序）。空=无录像。
func (s *Service) Segments(devCode, chCode string, startMs, endMs int64) ([]SegmentView, error) {
	stream := mediaurl.LiveStream(devCode, chCode)
	if stream == "" {
		return nil, fmt.Errorf("设备/通道内码缺失")
	}
	rows, err := s.repo.ListSegments(stream, startMs, endMs)
	if err != nil {
		return nil, err
	}
	out := make([]SegmentView, 0, len(rows))
	for i := range rows {
		r := rows[i]
		normalizeCloudRecord(&r)
		container := r.ContainerFormat
		if container == "" {
			if r.SlotID > 0 {
				container = "ps"
			} else {
				container = "mp4"
			}
		}
		out = append(out, SegmentView{
			RecordID: r.ID, Stream: r.Stream, StartTime: r.StartTime, EndTime: r.EndTime,
			DurationMs: r.EndTime - r.StartTime, Container: container, Size: r.FileSize, Online: true,
		})
	}
	return out, nil
}

// getRecord 取记录并按需归一化时间。
func (s *Service) getRecord(id int) (*domaincloudrecord.Record, error) {
	rec, err := s.repo.GetByID(id)
	if err != nil || rec == nil {
		return nil, fmt.Errorf("录像不存在")
	}
	normalizeCloudRecord(rec)
	return rec, nil
}

// playbackSessionKey 返回 zstorage 回放会话的 (app, stream)：与 loadPsRecord 起流时一致。
func psSessionKey(rec *domaincloudrecord.Record) (string, string) {
	return "playback", fmt.Sprintf("%s_%d", rec.Stream, rec.SlotID)
}

// StartByRecord 播放器起播：PS 走 zstorage（offset=绝对时刻-段起始），MP4 走原 ZMS 路径。
func (s *Service) StartByRecord(recordID int, absStartMs int64) (*dto.StreamContent, error) {
	rec, err := s.getRecord(recordID)
	if err != nil {
		return nil, err
	}
	if rec.StorageNodeID != "" && rec.SlotID > 0 {
		off := int64(0)
		if absStartMs > 0 {
			off = absStartMs - rec.StartTime
		}
		return s.loadPsRecord(rec, off)
	}
	return s.LoadRecord(context.Background(), rec.App, rec.Stream, recordID)
}

// SeekByRecord 播放器拖动：绝对时间 → 段内相对偏移再下沉。
func (s *Service) SeekByRecord(recordID int, absSeekMs int64) error {
	rec, err := s.getRecord(recordID)
	if err != nil {
		return err
	}
	if rec.StorageNodeID == "" || rec.SlotID <= 0 {
		return nil // MP4 暂由播放端/ZMS 处理
	}
	off := absSeekMs - rec.StartTime
	if off < 0 {
		off = 0
	}
	node, err := s.storageNode.Resolve(rec.StorageNodeID)
	if err != nil || node == nil {
		return fmt.Errorf("存储节点不可用: %s", rec.StorageNodeID)
	}
	app, stream := psSessionKey(rec)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := storagekit.NewClient(node.IP, node.HTTPPort, node.Secret).PlaybackSeek(ctx, app, stream, off)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("回放拖动失败: %s", resp.Msg)
	}
	return nil
}

// StopByRecord 关闭回放（PS 关闭 zstorage 会话）。
func (s *Service) StopByRecord(recordID int) error {
	rec, err := s.getRecord(recordID)
	if err != nil {
		return err
	}
	if rec.StorageNodeID == "" || rec.SlotID <= 0 {
		return nil
	}
	node, err := s.storageNode.Resolve(rec.StorageNodeID)
	if err != nil || node == nil {
		return nil
	}
	app, stream := psSessionKey(rec)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = storagekit.NewClient(node.IP, node.HTTPPort, node.Secret).PlaybackStop(ctx, app, stream)
	return nil
}
