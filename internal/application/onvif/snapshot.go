package onvifapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	domainonvif "zero-web-server/internal/domain/onvif"
)

// snapshotMaxBytes 限制单张抓图大小，避免异常设备把内存打满。
const snapshotMaxBytes = 8 << 20

// snapshotHTTPTimeout 抓图是一次同步 HTTP 请求，卡住会直接挂住接口。
const snapshotHTTPTimeout = 10 * time.Second

// Snapshot 从设备取一张 JPEG。
//
// 走 ONVIF 的 GetSnapshotURI 拿到设备自述的抓图地址再 HTTP 拉取：
// 这是规范给的正路，比从直播流里截帧可靠得多（不依赖 ZMS 是否在出流，
// 也不会因为播放器没在播就抓不到）。profileToken 为空时用通道当前码流。
func (s *Service) Snapshot(ctx context.Context, channelID int64, profileToken string) ([]byte, error) {
	ch, err := s.channels.GetByID(ctx, channelID)
	if err != nil || ch == nil {
		return nil, fmt.Errorf("通道不存在")
	}
	device, err := s.devices.GetByID(ctx, ch.DeviceID)
	if err != nil || device == nil {
		return nil, fmt.Errorf("设备不存在")
	}
	if strings.TrimSpace(profileToken) == "" {
		profileToken = ch.ProfileToken
	}
	if strings.TrimSpace(profileToken) == "" {
		return nil, fmt.Errorf("通道缺少 ProfileToken，无法抓图")
	}
	client, err := s.factory.Initialize(ctx, device.DeviceURI, device.Username, device.Password)
	if err != nil {
		return nil, fmt.Errorf("连接设备失败: %w", err)
	}
	media, err := client.GetSnapshotURI(ctx, profileToken)
	if err != nil {
		return nil, fmt.Errorf("设备未提供抓图地址: %w", err)
	}
	if media == nil || strings.TrimSpace(media.URI) == "" {
		return nil, fmt.Errorf("设备未提供抓图地址")
	}
	return fetchSnapshot(ctx, media.URI, device.Username, device.Password)
}

// fetchSnapshot 拉取抓图地址。
//
// 认证分两步：URL 里塞用户名密码（Basic）先试一次；401 时用同一个 URL 再带
// Authorization 头试一次——有的设备只在 header 里认，有的要求 URL 形式。
// Digest 认证不支持：Go 标准库没有，设备也极少在抓图接口上强制 Digest。
func fetchSnapshot(ctx context.Context, rawURL, username, password string) ([]byte, error) {
	hc := &http.Client{Timeout: snapshotHTTPTimeout}
	do := func(withAuth bool) ([]byte, int, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, 0, err
		}
		if withAuth && username != "" {
			token := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
			req.Header.Set("Authorization", "Basic "+token)
		}
		resp, err := hc.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			return nil, resp.StatusCode, nil
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, snapshotMaxBytes))
		if err != nil {
			return nil, resp.StatusCode, err
		}
		return body, resp.StatusCode, nil
	}

	body, status, err := do(false)
	if err != nil {
		return nil, fmt.Errorf("抓图请求失败: %w", err)
	}
	if status == http.StatusUnauthorized && username != "" {
		body, status, err = do(true)
		if err != nil {
			return nil, fmt.Errorf("抓图请求失败: %w", err)
		}
	}
	if status >= 300 {
		return nil, fmt.Errorf("设备抓图返回 HTTP %d", status)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("设备返回空图")
	}
	if !looksLikeJPEG(body) {
		return nil, fmt.Errorf("设备返回的不是 JPEG 图片")
	}
	return body, nil
}

func looksLikeJPEG(b []byte) bool {
	return len(b) > 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF
}

// ChannelSnapshot 按设备直接抓图（不经过平台通道表），供驱动层复用。
func (s *Service) ChannelSnapshot(ctx context.Context, ch *domainonvif.Channel, device *domainonvif.Device) ([]byte, error) {
	if ch == nil || device == nil {
		return nil, fmt.Errorf("通道或设备不存在")
	}
	return s.Snapshot(ctx, ch.ID, ch.ProfileToken)
}
