package asset

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	deviceapp "zero-web-server/internal/application/device"
	ipcdevice "zero-web-server/internal/application/ipcdevice"
	onvifapp "zero-web-server/internal/application/onvif"
	playapp "zero-web-server/internal/application/play"
	domainasset "zero-web-server/internal/domain/asset"
	"zero-web-server/internal/dto"
)

// Projector 触发统一资产投影（由持久化层实现）。
type Projector interface {
	Backfill() error
	// Prune 清理 legacy 已删除、统一资产仍残留的孤儿数据。
	Prune() error
}

type Service struct {
	repo      domainasset.Repository
	projector Projector
	play      *playapp.Service
	onvif     *onvifapp.Service
	ipc       *ipcdevice.Service
	device    *deviceapp.Service

	mu       sync.Mutex
	sessions map[string]*liveSession
}

type liveSession struct {
	Protocol        string
	DeviceExternal  string
	ChannelKey      string
	LegacyDeviceID  int
	LegacyChannelID int64
	ProfileToken    string
}

func NewService(repo domainasset.Repository, projector Projector, play *playapp.Service,
	onvif *onvifapp.Service, ipc *ipcdevice.Service, device *deviceapp.Service) *Service {
	return &Service{
		repo: repo, projector: projector, play: play, onvif: onvif,
		ipc: ipc, device: device, sessions: map[string]*liveSession{},
	}
}

// Prune 清理 legacy 已删除、统一资产仍残留的孤儿数据（删除设备后调用）。
func (s *Service) Prune() error {
	if s == nil || s.projector == nil {
		return nil
	}
	return s.projector.Prune()
}

// Backfill 触发一次全量资产投影（插件/ONVIF/GB 设备增删改后调用，幂等）。
func (s *Service) Backfill() error {
	if s == nil || s.projector == nil {
		return nil
	}
	return s.projector.Backfill()
}

func (s *Service) List(page, count int, query, protocol, status string) ([]*domainasset.DeviceSummary, int64, error) {
	if s.repo == nil {
		return nil, 0, fmt.Errorf("资产仓储未初始化")
	}
	return s.repo.ListDevices(page, count, query, protocol, status)
}

// ListChannels 跨协议统一通道列表（以通道内码为对外身份）。
func (s *Service) ListChannels(page, count int, query, protocol, deviceInternalCode string) ([]*domainasset.ChannelView, int64, error) {
	if s.repo == nil {
		return nil, 0, fmt.Errorf("资产仓储未初始化")
	}
	return s.repo.ListChannels(page, count, query, protocol, deviceInternalCode)
}

func (s *Service) GetDetail(deviceID string) (*domainasset.DeviceDetail, error) {
	dev, err := s.repo.GetDevice(deviceID)
	if err != nil {
		return nil, fmt.Errorf("设备不存在")
	}
	binding, _ := s.repo.GetBindingByDevice(deviceID)
	resources, err := s.repo.ListResources(deviceID)
	if err != nil {
		return nil, err
	}
	caps, _ := s.repo.ListCapabilities("device", deviceID)
	events, _ := s.repo.ListEventSources(deviceID)
	return &domainasset.DeviceDetail{
		Device: dev, Binding: binding, Resources: resources,
		Capabilities: caps, EventSources: events,
	}, nil
}

func (s *Service) ResourceDetail(resourceID string) (*domainasset.ResourceDetail, error) {
	res, err := s.repo.GetResource(resourceID)
	if err != nil {
		return nil, fmt.Errorf("资源不存在")
	}
	profiles, err := s.repo.ListProfiles(resourceID)
	if err != nil {
		return nil, err
	}
	caps, _ := s.repo.ListCapabilities("resource", resourceID)
	return &domainasset.ResourceDetail{Resource: res, Profiles: profiles, Capabilities: caps}, nil
}

// Tree 返回设备→通道/资源树（全量、已按协议/状态过滤）。
func (s *Service) Tree(query, protocol, status string) ([]*domainasset.DeviceNode, error) {
	list, _, err := s.repo.ListDevices(1, 500, query, protocol, status)
	if err != nil {
		return nil, err
	}
	out := make([]*domainasset.DeviceNode, 0, len(list))
	for _, item := range list {
		if item.Device == nil {
			continue
		}
		resources, err := s.repo.ListResources(item.Device.ID)
		if err != nil {
			return nil, err
		}
		nodes := make([]*domainasset.ResourceNode, 0, len(resources))
		for _, res := range resources {
			profiles, _ := s.repo.ListProfiles(res.ID)
			caps, _ := s.repo.ListCapabilities("resource", res.ID)
			nodes = append(nodes, &domainasset.ResourceNode{
				Resource: res, Profiles: profiles, Capabilities: caps,
			})
		}
		out = append(out, &domainasset.DeviceNode{
			Device: item.Device, Binding: item.Binding, Resources: nodes,
		})
	}
	return out, nil
}

func (s *Service) UpdateDevice(deviceID string, patch domainasset.DevicePatch) (*domainasset.PhysicalDevice, error) {
	if patch.Name != nil && strings.TrimSpace(*patch.Name) == "" {
		return nil, fmt.Errorf("设备名称不能为空")
	}
	if _, err := s.repo.GetDevice(deviceID); err != nil {
		return nil, fmt.Errorf("设备不存在")
	}
	if err := s.repo.UpdateDeviceFields(deviceID, patch); err != nil {
		return nil, err
	}
	return s.repo.GetDevice(deviceID)
}

func (s *Service) UpdateChannel(internalCode string, patch domainasset.ResourcePatch) (*domainasset.Resource, error) {
	if patch.Name != nil && strings.TrimSpace(*patch.Name) == "" {
		return nil, fmt.Errorf("通道名称不能为空")
	}
	res, err := s.repo.GetResourceByInternalCode(internalCode)
	if err != nil {
		return nil, fmt.Errorf("通道不存在")
	}
	if err := s.repo.UpdateResourceFields(res.ID, patch); err != nil {
		return nil, err
	}
	return s.repo.GetResource(res.ID)
}

// Sync 调用对应协议的同步，然后重新投影统一资产。
func (s *Service) Sync(ctx context.Context, deviceID string) error {
	dev, err := s.repo.GetDevice(deviceID)
	if err != nil {
		return fmt.Errorf("设备不存在")
	}
	_ = dev
	binding, err := s.repo.GetBindingByDevice(deviceID)
	if err != nil || binding == nil {
		return fmt.Errorf("设备缺少接入绑定")
	}
	switch binding.Protocol {
	case "gb28181":
		if err := s.device.SyncCatalog(binding.ExternalID); err != nil {
			return err
		}
	case "onvif":
		if _, err := s.onvif.SyncChannels(ctx, int64(binding.LegacyDeviceID)); err != nil {
			return err
		}
	case "plugin":
		if _, err := s.ipc.SyncChannels(ctx, binding.LegacyDeviceID); err != nil {
			return err
		}
	default:
		return fmt.Errorf("不支持的接入协议: %s", binding.Protocol)
	}
	if s.projector != nil {
		return s.projector.Backfill()
	}
	return nil
}

// StartLive 按统一资源与码流选择路由到对应协议的播放实现。
func (s *Service) StartLive(ctx context.Context, resourceID, profileID string) (*dto.StreamContent, error) {
	res, err := s.repo.GetResource(resourceID)
	if err != nil {
		return nil, fmt.Errorf("资源不存在")
	}
	if res.Kind != domainasset.ResourceVideoSensor {
		return nil, fmt.Errorf("该资源类型不支持播放")
	}
	binding, err := s.repo.GetBindingByDevice(res.DeviceID)
	if err != nil || binding == nil {
		return nil, fmt.Errorf("设备缺少接入绑定")
	}
	profile, _ := s.pickProfile(resourceID, profileID)

	switch binding.Protocol {
	case "gb28181":
		if s.play == nil {
			return nil, fmt.Errorf("GB 播放服务未启用")
		}
		return s.play.StartPlay(ctx, binding.ExternalID, res.ExternalKey)
	case "onvif":
		token := profileToken(profile)
		content, err := s.onvif.StartPlay(ctx, int64(res.LegacyChannelID), token)
		if err != nil {
			return nil, err
		}
		s.setSession(resourceID, &liveSession{
			Protocol: binding.Protocol, LegacyChannelID: int64(res.LegacyChannelID), ProfileToken: token,
		})
		return &dto.StreamContent{
			App: content.App, Stream: content.Stream, MediaServerID: content.MediaServerID,
			Flv: content.URLs["flv"], WsFlv: content.URLs["ws"], Hls: content.URLs["hls"],
			Rtmp: content.URLs["rtmp"], Rtsp: content.URLs["rtsp"], Rtc: content.URLs["rtc"], Rtcs: content.URLs["rtcs"],
			VideoCodec: content.VideoCodec, AudioCodec: content.AudioCodec,
		}, nil
	case "plugin":
		if s.ipc == nil {
			return nil, fmt.Errorf("插件服务未启用")
		}
		content, err := s.ipc.StartLive(ctx, binding.LegacyDeviceID, res.ExternalKey)
		if err != nil {
			return nil, err
		}
		s.setSession(resourceID, &liveSession{
			Protocol: binding.Protocol, LegacyDeviceID: binding.LegacyDeviceID, ChannelKey: res.ExternalKey,
		})
		return content, nil
	default:
		return nil, fmt.Errorf("不支持的接入协议: %s", binding.Protocol)
	}
}

func (s *Service) StopLive(ctx context.Context, resourceID string) error {
	res, err := s.repo.GetResource(resourceID)
	if err != nil {
		return fmt.Errorf("资源不存在")
	}
	sess := s.getSession(resourceID)
	binding, _ := s.repo.GetBindingByDevice(res.DeviceID)
	protocol := ""
	if sess != nil {
		protocol = sess.Protocol
	} else if binding != nil {
		protocol = binding.Protocol
	}
	switch protocol {
	case "gb28181":
		if binding == nil {
			return fmt.Errorf("设备缺少接入绑定")
		}
		return s.play.StopPlay(binding.ExternalID, res.ExternalKey)
	case "onvif":
		token := ""
		if sess != nil {
			token = sess.ProfileToken
		}
		return s.onvif.StopPlay(ctx, int64(res.LegacyChannelID), token)
	case "plugin":
		devID := 0
		if binding != nil {
			devID = binding.LegacyDeviceID
		}
		if sess != nil {
			devID = sess.LegacyDeviceID
		}
		return s.ipc.StopLive(ctx, devID, res.ExternalKey)
	default:
		return fmt.Errorf("不支持的接入协议")
	}
}

func (s *Service) pickProfile(resourceID, profileID string) (*domainasset.StreamProfile, error) {
	if strings.TrimSpace(profileID) != "" {
		return s.repo.GetProfile(profileID)
	}
	list, err := s.repo.ListProfiles(resourceID)
	if err != nil || len(list) == 0 {
		return nil, fmt.Errorf("资源没有可用码流")
	}
	for _, p := range list {
		if p.IsDefault {
			return p, nil
		}
	}
	return list[0], nil
}

func profileToken(p *domainasset.StreamProfile) string {
	if p == nil || strings.TrimSpace(p.SelectorJSON) == "" {
		return ""
	}
	var sel struct {
		ProfileToken string `json:"profileToken"`
	}
	if err := json.Unmarshal([]byte(p.SelectorJSON), &sel); err != nil {
		return ""
	}
	return sel.ProfileToken
}

func (s *Service) setSession(resourceID string, sess *liveSession) {
	if sess == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[resourceID] = sess
}

func (s *Service) getSession(resourceID string) *liveSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.sessions[resourceID]
	if sess != nil {
		delete(s.sessions, resourceID)
	}
	return sess
}
