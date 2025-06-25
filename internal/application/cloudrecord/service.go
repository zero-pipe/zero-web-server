package cloudrecord

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"zero-web-kit/internal/infrastructure/config"
	"zero-web-kit/internal/infrastructure/media/mediakit"
	"zero-web-kit/internal/infrastructure/persistence"
	"zero-web-kit/internal/infrastructure/persistence/model"
	"zero-web-kit/internal/interfaces/http/dto"
)

const loadMP4App = "mp4_record"

type Service struct {
	repo     *persistence.CloudRecordRepository
	zlm      *mediakit.Client
	mediaCfg config.MediaConfig
	serverID string
	sessions sync.Map // streamKey -> chan *dto.StreamContent
}

func NewService(repo *persistence.CloudRecordRepository, zlmClient *mediakit.Client, mediaCfg config.MediaConfig, serverID string) *Service {
	return &Service{repo: repo, zlm: zlmClient, mediaCfg: mediaCfg, serverID: serverID}
}

func (s *Service) OnRecordMp4(param RecordHookParam) error {
	rec := &model.CloudRecord{
		App:           param.App,
		Stream:        param.Stream,
		StartTime:     param.StartTime,
		EndTime:       param.StartTime + int64(param.TimeLen),
		MediaServerID: s.mediaCfg.ID,
		ServerID:      s.serverID,
		FileName:      param.FileName,
		Folder:        param.Folder,
		FilePath:      param.FilePath,
		FileSize:      param.FileSize,
		TimeLen:       param.TimeLen,
	}
	if param.CallID != "" {
		rec.CallID = param.CallID
	}
	return s.repo.Create(rec)
}

type RecordHookParam struct {
	App       string
	Stream    string
	FileName  string
	FilePath  string
	FileSize  int64
	Folder    string
	StartTime int64
	TimeLen   float64
	CallID    string
}

func (s *Service) List(page, count int, app, stream, query, callID, mediaServerID string, startTime, endTime int64, asc bool) ([]model.CloudRecord, int64, error) {
	return s.repo.List(page, count, app, stream, query, callID, mediaServerID, startTime, endTime, asc)
}

func (s *Service) DateList(app, stream, mediaServerID string, year, month int) ([]string, error) {
	return s.repo.DateList(app, stream, mediaServerID, year, month)
}

func (s *Service) Delete(ids []int) error {
	return s.repo.Delete(ids)
}

func (s *Service) GetPlayPath(recordID int) (map[string]string, error) {
	rec, err := s.repo.GetByID(recordID)
	if err != nil {
		return nil, fmt.Errorf("录像不存在")
	}
	base := fmt.Sprintf("http://%s:%d", s.mediaCfg.IP, s.mediaCfg.HTTPPort)
	return map[string]string{
		"httpPath": rec.FilePath,
		"download": fmt.Sprintf("%s/index/api/downloadFile?secret=%s&file_path=%s", base, s.mediaCfg.Secret, rec.FilePath),
		"filePath": rec.FilePath,
	}, nil
}

func (s *Service) LoadRecord(ctx context.Context, app, stream string, cloudRecordID int) (*dto.StreamContent, error) {
	rec, err := s.repo.GetByID(cloudRecordID)
	if err != nil {
		return nil, fmt.Errorf("录像不存在")
	}
	filePath := rec.FilePath
	if filePath == "" {
		return nil, fmt.Errorf("录像文件路径为空")
	}
	name := strings.TrimSuffix(rec.FileName, filepath.Ext(rec.FileName))
	buildStream := fmt.Sprintf("%s_%s_%s_%s", app, stream, name, randomSuffix())
	done := make(chan *dto.StreamContent, 1)
	key := streamKey(loadMP4App, buildStream)
	s.sessions.Store(key, done)
	defer s.sessions.Delete(key)

	resp, err := s.zlm.LoadMP4File(ctx, loadMP4App, buildStream, filePath)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("loadMP4File: %s", resp.Msg)
	}

	select {
	case content := <-done:
		content.Progress = rec.TimeLen
		return content, nil
	case <-time.After(15 * time.Second):
		return s.buildStreamContent(loadMP4App, buildStream), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Service) Seek(app, stream, mediaServerID string, seek float64, schema string) error {
	_, err := s.zlm.SeekRecordStamp(context.Background(), app, stream, seek, schema)
	return err
}

func (s *Service) Speed(app, stream, mediaServerID string, speed int, schema string) error {
	_, err := s.zlm.SetRecordSpeed(context.Background(), app, stream, speed, schema)
	return err
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
			select {
			case ch <- s.buildStreamContent(app, stream):
			default:
			}
		}
	}
}

func (s *Service) buildStreamContent(app, stream string) *dto.StreamContent {
	urls := mediakit.BuildPlayURLsFromConfig(s.mediaCfg, app, stream)
	return &dto.StreamContent{
		App: app, Stream: stream, IP: s.mediaCfg.IP,
		Flv: urls["flv"], WsFlv: urls["ws"], Hls: urls["hls"],
		Rtmp: urls["rtmp"], Rtsp: urls["rtsp"],
		MediaServerID: s.mediaCfg.ID, ServerID: s.serverID,
	}
}

func streamKey(app, stream string) string { return app + "/" + stream }

func randomSuffix() string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
