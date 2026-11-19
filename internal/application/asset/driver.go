package asset

import (
	"context"
	"fmt"

	deviceapp "zero-web-server/internal/application/device"
	ipcdevice "zero-web-server/internal/application/ipcdevice"
	onvifapp "zero-web-server/internal/application/onvif"
	playapp "zero-web-server/internal/application/play"
	domainasset "zero-web-server/internal/domain/asset"
	"zero-web-server/internal/dto"
)

// ChannelDriver 封装一种接入协议的通道操作，协议差异只能在驱动内部出现。
// 业务层（设备资产、播放入口、级联）只依赖本接口，不再按 binding.Protocol 分支。
type ChannelDriver interface {
	Protocol() string
	// StartLive 打开实时流；返回的 session 用于停止时恢复协议侧上下文（可为 nil）。
	StartLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
		profile *domainasset.StreamProfile) (*dto.StreamContent, *liveSession, error)
	StopLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, sess *liveSession) error
	// Sync 触发协议侧同步（目录/通道），由驱动决定具体实现。
	Sync(ctx context.Context, binding *domainasset.AccessBinding) error
}

// newDrivers 构建协议驱动表；新增协议在此注册，业务层无需改动。
func newDrivers(play *playapp.Service, onvif *onvifapp.Service, ipc *ipcdevice.Service,
	device *deviceapp.Service) map[string]ChannelDriver {
	return map[string]ChannelDriver{
		"gb28181": &gbDriver{play: play, device: device},
		"onvif":   &onvifDriver{onvif: onvif},
		"plugin":  &pluginDriver{ipc: ipc},
	}
}

// ---- GB28181：被动接入，点播经 SIP INVITE 拉流 ----

type gbDriver struct {
	play   *playapp.Service
	device *deviceapp.Service
}

func (d *gbDriver) Protocol() string { return "gb28181" }

func (d *gbDriver) StartLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	profile *domainasset.StreamProfile) (*dto.StreamContent, *liveSession, error) {
	if d.play == nil {
		return nil, nil, fmt.Errorf("GB 播放服务未启用")
	}
	content, err := d.play.StartPlay(ctx, binding.ExternalID, res.ExternalKey)
	if err != nil {
		return nil, nil, err
	}
	return content, nil, nil
}

func (d *gbDriver) StopLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, sess *liveSession) error {
	if d.play == nil {
		return fmt.Errorf("GB 播放服务未启用")
	}
	return d.play.StopPlay(binding.ExternalID, res.ExternalKey)
}

func (d *gbDriver) Sync(ctx context.Context, binding *domainasset.AccessBinding) error {
	if d.device == nil {
		return fmt.Errorf("设备服务未启用")
	}
	return d.device.SyncCatalog(binding.ExternalID)
}

// ---- ONVIF：主动接入，按 ProfileToken 取流 ----

type onvifDriver struct {
	onvif *onvifapp.Service
}

func (d *onvifDriver) Protocol() string { return "onvif" }

func (d *onvifDriver) StartLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	profile *domainasset.StreamProfile) (*dto.StreamContent, *liveSession, error) {
	if d.onvif == nil {
		return nil, nil, fmt.Errorf("ONVIF 服务未启用")
	}
	token := profileToken(profile)
	content, err := d.onvif.StartPlay(ctx, int64(res.LegacyChannelID), token)
	if err != nil {
		return nil, nil, err
	}
	sess := &liveSession{
		Protocol: d.Protocol(), LegacyChannelID: int64(res.LegacyChannelID), ProfileToken: token,
	}
	return &dto.StreamContent{
		App: content.App, Stream: content.Stream, MediaServerID: content.MediaServerID,
		Flv: content.URLs["flv"], WsFlv: content.URLs["ws"], Hls: content.URLs["hls"],
		Rtmp: content.URLs["rtmp"], Rtsp: content.URLs["rtsp"], Rtc: content.URLs["rtc"], Rtcs: content.URLs["rtcs"],
		VideoCodec: content.VideoCodec, AudioCodec: content.AudioCodec,
	}, sess, nil
}

func (d *onvifDriver) StopLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, sess *liveSession) error {
	if d.onvif == nil {
		return fmt.Errorf("ONVIF 服务未启用")
	}
	token := ""
	if sess != nil {
		token = sess.ProfileToken
	}
	return d.onvif.StopPlay(ctx, int64(res.LegacyChannelID), token)
}

func (d *onvifDriver) Sync(ctx context.Context, binding *domainasset.AccessBinding) error {
	if d.onvif == nil {
		return fmt.Errorf("ONVIF 服务未启用")
	}
	_, err := d.onvif.SyncChannels(ctx, int64(binding.LegacyDeviceID))
	return err
}

// ---- 插件（私有 SDK）：zero-agent 托管 sidecar，按插件通道号取流 ----

type pluginDriver struct {
	ipc *ipcdevice.Service
}

func (d *pluginDriver) Protocol() string { return "plugin" }

func (d *pluginDriver) StartLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	profile *domainasset.StreamProfile) (*dto.StreamContent, *liveSession, error) {
	if d.ipc == nil {
		return nil, nil, fmt.Errorf("插件服务未启用")
	}
	content, err := d.ipc.StartLive(ctx, binding.LegacyDeviceID, res.ExternalKey)
	if err != nil {
		return nil, nil, err
	}
	sess := &liveSession{
		Protocol: d.Protocol(), LegacyDeviceID: binding.LegacyDeviceID, ChannelKey: res.ExternalKey,
	}
	return content, sess, nil
}

func (d *pluginDriver) StopLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, sess *liveSession) error {
	if d.ipc == nil {
		return fmt.Errorf("插件服务未启用")
	}
	devID := 0
	if binding != nil {
		devID = binding.LegacyDeviceID
	}
	if sess != nil {
		devID = sess.LegacyDeviceID
	}
	return d.ipc.StopLive(ctx, devID, res.ExternalKey)
}

func (d *pluginDriver) Sync(ctx context.Context, binding *domainasset.AccessBinding) error {
	if d.ipc == nil {
		return fmt.Errorf("插件服务未启用")
	}
	_, err := d.ipc.SyncChannels(ctx, binding.LegacyDeviceID)
	return err
}
