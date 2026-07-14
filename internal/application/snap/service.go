package snapapp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	objectstoreapp "zero-web-kit/internal/application/objectstore"
)

// Service 抓拍图归档到 ObjectStore（固定 key：最新一帧可覆盖）。
type Service struct {
	store *objectstoreapp.Service
}

func NewService(store *objectstoreapp.Service) *Service {
	return &Service{store: store}
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
