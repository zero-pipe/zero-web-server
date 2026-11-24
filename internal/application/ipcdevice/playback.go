package ipcdevice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"zero-web-server/internal/dto"
	"zero-web-server/internal/mediaurl"
	domainplugin "zero-web-server/internal/domain/plugin"
	applog "zero-web-server/pkg/log"
)

// 回放相关的命令名，与插件 RPC 契约（docs/插件化设备接入.md）一致。
const (
	cmdRecordQuery   = "record.query"
	cmdPlaybackStart = "playback.start"
	cmdPlaybackStop  = "playback.stop"
)

// playbackStartResult 是插件回放取流的返回：目前只有 RTSP 拉流一种传输方式。
type playbackStartResult struct {
	Transport string `json:"transport"`
	RTSPURL   string `json:"rtspUrl"`
}

// QueryRecord 检索设备侧录像。
//
// 返回的是插件原始结构（records/total），平台不解析语义：
// 各家的录像条目字段差异太大，归一化交给前端按需处理。
// 设备没开录像时插件返回空数组而不是错误，这里原样透传。
func (s *Service) QueryRecord(ctx context.Context, deviceID int, channelID, startTime, endTime string, streamType int) (json.RawMessage, error) {
	dev, _, client, err := s.deviceClient(deviceID)
	if err != nil {
		return nil, err
	}
	if !dev.Enabled {
		return nil, fmt.Errorf("设备已禁用")
	}
	params, err := json.Marshal(map[string]any{
		"startTime":  startTime,
		"endTime":    endTime,
		"streamType": streamType,
	})
	if err != nil {
		return nil, err
	}
	// 不在这里 MarkOffline：参数非法、设备没开录像等都会走到这里，
	// 把插件标离线会让整个插件下的设备一起掉线。插件心跳负责真实在线状态。
	return client.Command(ctx, dev.DeviceUID, channelID, cmdRecordQuery, params)
}

// StartPlayback 让插件给出按时间回放的 RTSP 地址，再由 ZMS 拉成
// live/<设备内码>_<通道内码>_playback 供播放器使用。
//
// streamKey 是码流选择（main/sub），空则跟随插件默认。
func (s *Service) StartPlayback(ctx context.Context, deviceID int, channelID, startTime, endTime, streamKey string) (*dto.StreamContent, error) {
	dev, _, client, err := s.deviceClient(deviceID)
	if err != nil {
		return nil, err
	}
	if !dev.Enabled {
		return nil, fmt.Errorf("设备已禁用")
	}
	stream, err := s.playbackStreamName(dev, channelID)
	if err != nil {
		return nil, err
	}
	params, err := json.Marshal(map[string]any{
		"startTime": startTime,
		"endTime":   endTime,
		"stream":    streamKey,
	})
	if err != nil {
		return nil, err
	}
	raw, err := client.Command(ctx, dev.DeviceUID, channelID, cmdPlaybackStart, params)
	if err != nil {
		return nil, fmt.Errorf("插件回放取流失败: %w", err)
	}
	var res playbackStartResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("插件回放返回无法解析: %w", err)
	}
	if strings.TrimSpace(res.RTSPURL) == "" {
		return nil, fmt.Errorf("插件未返回可用的回放流源")
	}

	node, err := s.media.ResolveForStream(ctx, streamApp, stream, "")
	if err != nil {
		return nil, err
	}
	// 回放每次请求对应一个时间段，不能像直播那样复用旧拉流：时间段变了流内容就变了。
	// 先清掉可能残留的上一段，避免 AddStreamProxy 冲突。
	if probe := node.LookupStream(ctx, streamApp, stream); probe != nil {
		_ = node.CloseStreams(ctx, "__defaultVhost__", streamApp, stream)
	}
	if _, err := node.AddStreamProxy(ctx, "__defaultVhost__", streamApp, stream, res.RTSPURL, "1", false, false, true); err != nil {
		return nil, fmt.Errorf("创建回放拉流代理失败: %w", err)
	}
	s.media.BindStream(node.ID(), streamApp, stream)
	s.setActive(stream, &activeStream{
		App: streamApp, Stream: stream, NodeID: node.ID(),
		PluginID: dev.PluginID, DeviceUID: dev.DeviceUID, ChannelID: channelID,
	})
	applog.Debugf("[ipc] 回放起流: stream=%s %s~%s src=%s", stream, startTime, endTime, res.RTSPURL)
	return s.buildContent(ctx, stream)
}

// StopPlayback 关闭回放流。设备侧的 RTSP 会话由 ZMS 断开，插件无需额外动作，
// 这里仍通知一次以保持两端状态对称（失败不影响关流）。
func (s *Service) StopPlayback(ctx context.Context, deviceID int, channelID string) error {
	dev, err := s.repo.GetDevice(deviceID)
	if err != nil {
		return err
	}
	stream, err := s.playbackStreamName(dev, channelID)
	if err != nil {
		return err
	}
	act := s.getActive(stream)
	if act == nil {
		return nil
	}
	if p, perr := s.repo.GetPlugin(dev.PluginID); perr == nil {
		if _, cerr := s.factory.For(p.Endpoint).Command(ctx, dev.DeviceUID, channelID, cmdPlaybackStop, nil); cerr != nil {
			applog.Debugf("[ipc] 插件回放停止返回错误（忽略）: %v", cerr)
		}
	}
	if node, nerr := s.media.Resolve(ctx, act.NodeID); nerr == nil {
		_ = node.CloseStreams(ctx, "__defaultVhost__", act.App, act.Stream)
	}
	s.media.UnbindStream(act.App, act.Stream)
	s.setActive(stream, nil)
	return nil
}

// playbackStreamName 回放流标识：<设备内码>_<通道内码>_playback，落在 ZMS 的 live app 下。
func (s *Service) playbackStreamName(dev *domainplugin.Device, channelID string) (string, error) {
	if dev == nil {
		return "", fmt.Errorf("设备不存在")
	}
	name := mediaurl.PlaybackStream(dev.InternalCode, s.channelInternalCode(dev, channelID))
	if name == "" {
		return "", fmt.Errorf("设备或通道内码缺失")
	}
	return name, nil
}

// snapshotResult 是插件抓图的返回：base64 编码的 JPEG。
type snapshotResult struct {
	Format   string `json:"format"`
	Encoding string `json:"encoding"`
	Image    string `json:"image"`
}

// Snapshot 让插件抓一张 JPEG。
//
// 走通用命令通道（cmd=snapshot）而不是新增 RPC 方法：抓图是纯透传，
// 塞进 IPCClient 接口只会让所有插件实现都被迫加一个空方法。
func (s *Service) Snapshot(ctx context.Context, deviceID int, channelID string) ([]byte, error) {
	dev, _, client, err := s.deviceClient(deviceID)
	if err != nil {
		return nil, err
	}
	if !dev.Enabled {
		return nil, fmt.Errorf("设备已禁用")
	}
	params, err := json.Marshal(map[string]any{"quality": 1})
	if err != nil {
		return nil, err
	}
	raw, err := client.Command(ctx, dev.DeviceUID, channelID, "snapshot", params)
	if err != nil {
		return nil, fmt.Errorf("插件抓图失败: %w", err)
	}
	var res snapshotResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("插件抓图返回无法解析: %w", err)
	}
	if !strings.EqualFold(res.Encoding, "base64") || strings.TrimSpace(res.Image) == "" {
		return nil, fmt.Errorf("插件未返回图片数据")
	}
	img, err := base64.StdEncoding.DecodeString(res.Image)
	if err != nil {
		return nil, fmt.Errorf("插件返回的图片数据损坏: %w", err)
	}
	if len(img) < 3 || img[0] != 0xFF || img[1] != 0xD8 {
		return nil, fmt.Errorf("插件返回的不是 JPEG 图片")
	}
	return img, nil
}
