package snapapp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	objectstoreapp "zero-web-server/internal/application/objectstore"
)

// Service 抓拍图归档到 ObjectStore（固定 key：最新一帧可覆盖）。
type Service struct {
	store    *objectstoreapp.Service
	mu       sync.Mutex
	finished map[string]SnapShotRecord // sessionID -> 最近一次完成通知
}

// SnapShotRecord 记录设备上报的图像抓拍完成通知。
type SnapShotRecord struct {
	DeviceID  string   `json:"deviceId"`
	SessionID string   `json:"sessionId"`
	Files     []string `json:"files"`
	At        string   `json:"at"`
}

func NewService(store *objectstoreapp.Service) *Service {
	return &Service{store: store, finished: make(map[string]SnapShotRecord)}
}

// PutSnapshot 归档 GB28181 图像抓拍上传的 JPEG（2022 9.14）。
func (s *Service) PutSnapshot(ctx context.Context, deviceGBID, fileName string, data []byte) (key, url string, err error) {
	if len(data) == 0 {
		return "", "", fmt.Errorf("空图像")
	}
	name := strings.TrimSpace(fileName)
	if name == "" {
		name = fmt.Sprintf("%s_%d.jpg", sanitize(deviceGBID), time.Now().UnixMilli())
	}
	key = fmt.Sprintf("snapshot/%s/%s", sanitize(deviceGBID), sanitize(name))
	if err := s.store.Store().Put(ctx, key, bytes.NewReader(data), int64(len(data)), "image/jpeg"); err != nil {
		return "", "", err
	}
	url, _ = s.store.Store().PresignGet(ctx, key, time.Hour)
	return key, url, nil
}

// HandleFinished 记录设备上报的抓拍完成通知。
func (s *Service) HandleSnapShotFinished(deviceID, sessionID string, files []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.finished[sessionID] = SnapShotRecord{
		DeviceID: deviceID, SessionID: sessionID, Files: files,
		At: time.Now().Format("2006-01-02 15:04:05"),
	}
	return nil
}

// Finished 返回某会话的抓拍完成记录。
func (s *Service) Finished(sessionID string) (SnapShotRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.finished[sessionID]
	return rec, ok
}

func SnapKey(deviceID, channelID string) string {
	return fmt.Sprintf("snap/%s/%s.jpg", sanitize(deviceID), sanitize(channelID))
}

func (s *Service) PutJPEG(ctx context.Context, deviceID, channelID string, data []byte) (key string, url string, err error) {
	if len(data) == 0 {
		return "", "", fmt.Errorf("empty image")
	}
	key = SnapKey(deviceID, channelID)
	if err := s.store.Store().Put(ctx, key, bytes.NewReader(data), int64(len(data)), "image/jpeg"); err != nil {
		return "", "", err
	}
	url, _ = s.store.Store().PresignGet(ctx, key, time.Hour)
	return key, url, nil
}

// OpenLatest 优先 Presign 重定向；否则流式读出对象。
func (s *Service) OpenLatest(ctx context.Context, deviceID, channelID string) (body io.ReadCloser, contentType string, redirectURL string, err error) {
	key := SnapKey(deviceID, channelID)
	if url, perr := s.store.Store().PresignGet(ctx, key, time.Hour); perr == nil && url != "" {
		// publicBase 或预签名均可直接跳转
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			return nil, "", url, nil
		}
	}
	rc, gerr := s.store.Store().Get(ctx, key)
	if gerr != nil {
		return nil, "", "", gerr
	}
	return rc, "image/jpeg", "", nil
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	if s == "" {
		return "_"
	}
	return s
}
