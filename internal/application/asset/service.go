package asset

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"

	"zero-web-server/internal/application/channeldrv"
	domainasset "zero-web-server/internal/domain/asset"
	domainptz "zero-web-server/internal/domain/ptz"
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
	drivers   channeldrv.Registry

	mu       sync.Mutex
	sessions map[string]*channeldrv.Session
	// sf 合并同一资源/码流的并发起流：多人同时点播只发一次信令，避免在 ZMS 互相踢流。
	sf singleflight.Group
}

func NewService(repo domainasset.Repository, projector Projector, drivers channeldrv.Registry) *Service {
	return &Service{
		repo: repo, projector: projector, drivers: drivers,
		sessions: map[string]*channeldrv.Session{},
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
	list, total, err := s.repo.ListChannels(page, count, query, protocol, deviceInternalCode)
	if err != nil {
		return nil, 0, err
	}
	for _, v := range list {
		v.Caps = s.channelCaps(v)
	}
	return list, total, nil
}

// channelCaps 按协议驱动自报的能力 ∩ 通道事实，得出该通道真实可用的能力集。
//
// 驱动 Capabilities() 是"这个协议实现了哪些操作"的唯一真相；
// 再叠加通道自身事实（云台要 ptzType>0、音频要 hasAudio）收窄。
// 前端「能力集」列据此点亮图标，扩展能力只需改驱动 + 前端模板两处。
func (s *Service) channelCaps(v *domainasset.ChannelView) []string {
	if v == nil {
		return nil
	}
	var base []string
	if drv, ok := s.drivers.Get(v.Protocol); ok && drv != nil {
		base = drv.Capabilities()
	}
	out := make([]string, 0, len(base))
	for _, op := range base {
		switch op {
		case "ptz":
			if v.PTZType > 0 {
				out = append(out, op)
			}
		default:
			out = append(out, op)
		}
	}
	return out
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
	ext, _ := s.repo.GetDeviceExt(deviceID)
	return &domainasset.DeviceDetail{
		Device: dev, Ext: ext, Binding: binding, Resources: resources,
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
	drv, ok := s.drivers.Get(binding.Protocol)
	if !ok {
		return fmt.Errorf("不支持的接入协议: %s", binding.Protocol)
	}
	if err := drv.Sync(ctx, binding); err != nil {
		return err
	}
	if s.projector != nil {
		return s.projector.Backfill()
	}
	return nil
}

// StartLive 按统一资源与码流选择路由到对应协议的播放实现。
// 并发同一资源(同码流)的点播用 singleflight 合并：只发一次信令，其余直接共享结果，
// 避免多人同时点播时在 ZMS 反复 AddStreamProxy 互相踢流(kick_prev)。
func (s *Service) StartLive(ctx context.Context, resourceID, profileID string) (*dto.StreamContent, error) {
	key := resourceID + "|" + profileID
	v, err, _ := s.sf.Do(key, func() (any, error) {
		return s.startLiveOnce(ctx, resourceID, profileID)
	})
	if err != nil {
		return nil, err
	}
	content, _ := v.(*dto.StreamContent)
	return content, nil
}

func (s *Service) startLiveOnce(ctx context.Context, resourceID, profileID string) (*dto.StreamContent, error) {
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

	// 能力声明 + 运行时熔断：明确不支持的直接拒绝；调用失败则把可用性置 false。
	if supported, exists := s.capabilitySupported("resource", resourceID, "play"); exists && !supported {
		return nil, fmt.Errorf("该通道不支持实时播放")
	}
	content, err := s.startLiveByProtocol(ctx, res, binding, profile)
	if err != nil {
		s.markCapability("resource", resourceID, "play", false)
		return nil, err
	}
	s.markCapability("resource", resourceID, "play", true)
	return content, nil
}

func (s *Service) startLiveByProtocol(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, profile *domainasset.StreamProfile) (*dto.StreamContent, error) {
	drv, ok := s.drivers.Get(binding.Protocol)
	if !ok {
		return nil, fmt.Errorf("不支持的接入协议: %s", binding.Protocol)
	}
	content, sess, err := drv.StartLive(ctx, res, binding, profile)
	if err != nil {
		return nil, err
	}
	if sess != nil {
		s.setSession(res.ID, sess)
	}
	return content, nil
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
	drv, ok := s.drivers.Get(protocol)
	if !ok {
		return fmt.Errorf("不支持的接入协议")
	}
	if protocol == "gb28181" && binding == nil {
		return fmt.Errorf("设备缺少接入绑定")
	}
	return drv.StopLive(ctx, res, binding, sess)
}

// StartLiveByChannelInternalCode 按通道内码起直播（任意协议）。
// 用于兼容旧客户端：客户端只有通道内码（非国标设备没有国标码）时也能取流。
func (s *Service) StartLiveByChannelInternalCode(ctx context.Context, internalCode string) (*dto.StreamContent, error) {
	res, err := s.repo.GetResourceByInternalCode(strings.TrimSpace(internalCode))
	if err != nil || res == nil {
		return nil, fmt.Errorf("通道不存在")
	}
	return s.StartLive(ctx, res.ID, "")
}

// StopLiveByChannelInternalCode 按通道内码停直播（任意协议）。
func (s *Service) StopLiveByChannelInternalCode(ctx context.Context, internalCode string) error {
	res, err := s.repo.GetResourceByInternalCode(strings.TrimSpace(internalCode))
	if err != nil || res == nil {
		return fmt.Errorf("通道不存在")
	}
	return s.StopLive(ctx, res.ID)
}

// ---- 按时间回放（统一资源） ----
//
// 回放只在驱动实现了 channeldrv.PlaybackDriver 时可用。目前只有插件（私有 SDK）
// 接入做到位：国标回放在旧的 playback 服务里走 SIP INVITE，ONVIF 缺 Profile G。
// 这里不做能力行判定，直接按驱动实现与否拒绝，避免给前端点亮无实际功能的按钮。

// QueryRecords 检索资源对应通道的设备录像。
func (s *Service) QueryRecords(ctx context.Context, resourceID, startTime, endTime string, streamType int) (json.RawMessage, error) {
	res, binding, drv, err := s.playbackTarget(resourceID)
	if err != nil {
		return nil, err
	}
	pd, ok := drv.(channeldrv.PlaybackDriver)
	if !ok {
		return nil, errPlaybackUnsupported
	}
	return pd.QueryRecords(ctx, res, binding, startTime, endTime, streamType)
}

// StartPlayback 起一路回放流。streamKey 是码流选择（main/sub），空则用设备默认。
func (s *Service) StartPlayback(ctx context.Context, resourceID, startTime, endTime, streamKey string) (*dto.StreamContent, error) {
	res, binding, drv, err := s.playbackTarget(resourceID)
	if err != nil {
		return nil, err
	}
	pd, ok := drv.(channeldrv.PlaybackDriver)
	if !ok {
		return nil, errPlaybackUnsupported
	}
	content, sess, err := pd.StartPlayback(ctx, res, binding, startTime, endTime, streamKey)
	if err != nil {
		return nil, err
	}
	if sess != nil {
		// 回放会话单独记账，避免覆盖同一资源的直播会话。
		s.mu.Lock()
		s.sessions[resourceID+"#playback"] = sess
		s.mu.Unlock()
	}
	return content, nil
}

// StopPlayback 关闭回放流。
func (s *Service) StopPlayback(ctx context.Context, resourceID string) error {
	res, err := s.repo.GetResource(resourceID)
	if err != nil {
		return fmt.Errorf("资源不存在")
	}
	binding, _ := s.repo.GetBindingByDevice(res.DeviceID)
	if binding == nil {
		return fmt.Errorf("设备缺少接入绑定")
	}
	drv, ok := s.drivers.Get(binding.Protocol)
	if !ok {
		return fmt.Errorf("不支持的接入协议: %s", binding.Protocol)
	}
	pd, ok := drv.(channeldrv.PlaybackDriver)
	if !ok {
		return errPlaybackUnsupported
	}
	return pd.StopPlayback(ctx, res, binding)
}

var errPlaybackUnsupported = fmt.Errorf("该接入方式暂不支持回放")

// ptzTarget 解析云台/预置位所需的资源、接入绑定与驱动（空能力校验，能力由调用方按需做）。
func (s *Service) ptzTarget(resourceID string) (*domainasset.Resource, *domainasset.AccessBinding, channeldrv.ChannelDriver, error) {
	res, err := s.repo.GetResource(resourceID)
	if err != nil || res == nil {
		return nil, nil, nil, fmt.Errorf("资源不存在")
	}
	binding, err := s.repo.GetBindingByDevice(res.DeviceID)
	if err != nil || binding == nil {
		return nil, nil, nil, fmt.Errorf("设备缺少接入绑定")
	}
	drv, ok := s.drivers.Get(binding.Protocol)
	if !ok {
		return nil, nil, nil, fmt.Errorf("不支持的接入协议: %s", binding.Protocol)
	}
	return res, binding, drv, nil
}

// normalize01 把整数速度归一到 [0,1]（速度上限 max）。
func normalize01(v, max int) float64 {
	if v <= 0 || max <= 0 {
		return 0
	}
	f := float64(v) / float64(max)
	if f > 1 {
		f = 1
	}
	return f
}

// PTZ 统一云台控制：按资源 id 路由到协议驱动。
// 速度沿用平台既有取值：水平/垂直上限 255，变倍上限 16；command 为方向/变倍指令。
func (s *Service) PTZ(ctx context.Context, resourceID, command string, horizonSpeed, verticalSpeed, zoomSpeed int) error {
	res, binding, drv, err := s.ptzTarget(resourceID)
	if err != nil {
		return err
	}
	if !s.capabilityAllowed("resource", res.ID, "ptz") {
		return fmt.Errorf("该通道不支持云台控制")
	}
	maxSpeed := horizonSpeed
	if verticalSpeed > maxSpeed {
		maxSpeed = verticalSpeed
	}
	cmd := channeldrv.PTZCommand{
		Direction: command,
		Speed:     normalize01(maxSpeed, 255),
		Zoom:      normalize01(zoomSpeed, 16),
	}
	if err := drv.PTZ(ctx, res, binding, cmd); err != nil {
		s.markCapability("resource", res.ID, "ptz", false)
		return err
	}
	s.markCapability("resource", res.ID, "ptz", true)
	return nil
}

// presetTarget 解析预置位所需的资源、接入绑定与预设驱动。
// QueryPresets 查询某资源的预置位列表。
func (s *Service) QueryPresets(ctx context.Context, resourceID string) ([]domainptz.Preset, error) {
	res, binding, drv, err := s.ptzTarget(resourceID)
	if err != nil {
		return nil, err
	}
	pd, ok := drv.(channeldrv.PresetDriver)
	if !ok {
		return nil, channeldrv.ErrPresetUnsupported
	}
	return pd.QueryPresets(ctx, res, binding)
}

// CallPreset 调用指定预置位。
func (s *Service) CallPreset(ctx context.Context, resourceID, presetID string) error {
	res, binding, drv, err := s.ptzTarget(resourceID)
	if err != nil {
		return err
	}
	pd, ok := drv.(channeldrv.PresetDriver)
	if !ok {
		return channeldrv.ErrPresetUnsupported
	}
	return pd.CallPreset(ctx, res, binding, presetID)
}

// AddPreset 设置指定预置位。
func (s *Service) AddPreset(ctx context.Context, resourceID, presetID string) error {
	res, binding, drv, err := s.ptzTarget(resourceID)
	if err != nil {
		return err
	}
	pd, ok := drv.(channeldrv.PresetDriver)
	if !ok {
		return channeldrv.ErrPresetUnsupported
	}
	return pd.AddPreset(ctx, res, binding, presetID)
}

// DeletePreset 删除指定预置位。
func (s *Service) DeletePreset(ctx context.Context, resourceID, presetID string) error {
	res, binding, drv, err := s.ptzTarget(resourceID)
	if err != nil {
		return err
	}
	pd, ok := drv.(channeldrv.PresetDriver)
	if !ok {
		return channeldrv.ErrPresetUnsupported
	}
	return pd.DeletePreset(ctx, res, binding, presetID)
}

// Snapshot 让协议侧抓一张 JPEG。
//
// 抓图走协议自己的接口（ONVIF 抓图地址 / SDK 抓图），不依赖是否有人在看直播：
// 从流里截帧在无人观看时拿不到，而且要先起流，代价大得多。
func (s *Service) Snapshot(ctx context.Context, resourceID string) ([]byte, error) {
	res, err := s.repo.GetResource(resourceID)
	if err != nil || res == nil {
		return nil, fmt.Errorf("资源不存在")
	}
	if res.Kind != domainasset.ResourceVideoSensor {
		return nil, fmt.Errorf("该资源类型不支持抓图")
	}
	binding, err := s.repo.GetBindingByDevice(res.DeviceID)
	if err != nil || binding == nil {
		return nil, fmt.Errorf("设备缺少接入绑定")
	}
	drv, ok := s.drivers.Get(binding.Protocol)
	if !ok {
		return nil, fmt.Errorf("不支持的接入协议: %s", binding.Protocol)
	}
	sd, ok := drv.(channeldrv.SnapshotDriver)
	if !ok {
		return nil, fmt.Errorf("该接入方式暂不支持抓图")
	}
	return sd.Snapshot(ctx, res, binding)
}

// playbackTarget 解析回放所需的资源、接入绑定与驱动。
func (s *Service) playbackTarget(resourceID string) (*domainasset.Resource, *domainasset.AccessBinding, channeldrv.ChannelDriver, error) {
	res, err := s.repo.GetResource(resourceID)
	if err != nil || res == nil {
		return nil, nil, nil, fmt.Errorf("资源不存在")
	}
	if res.Kind != domainasset.ResourceVideoSensor {
		return nil, nil, nil, fmt.Errorf("该资源类型不支持回放")
	}
	binding, err := s.repo.GetBindingByDevice(res.DeviceID)
	if err != nil || binding == nil {
		return nil, nil, nil, fmt.Errorf("设备缺少接入绑定")
	}
	drv, ok := s.drivers.Get(binding.Protocol)
	if !ok {
		return nil, nil, nil, fmt.Errorf("不支持的接入协议: %s", binding.Protocol)
	}
	return res, binding, drv, nil
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

// Command 通用命令透传（自定义能力）：校验声明/熔断后转发到协议驱动；平台不解析语义。
func (s *Service) Command(ctx context.Context, resourceID, cmd string, params json.RawMessage) (json.RawMessage, error) {
	res, err := s.repo.GetResource(resourceID)
	if err != nil || res == nil {
		return nil, fmt.Errorf("资源不存在")
	}
	binding, err := s.repo.GetBindingByDevice(res.DeviceID)
	if err != nil || binding == nil {
		return nil, fmt.Errorf("设备缺少接入绑定")
	}
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil, fmt.Errorf("命令不能为空")
	}
	if supported, exists := s.capabilitySupported("device", res.DeviceID, cmd); exists && !supported {
		return nil, fmt.Errorf("该设备不支持能力 %s", cmd)
	}
	drv, ok := s.drivers.Get(binding.Protocol)
	if !ok {
		return nil, fmt.Errorf("不支持的接入协议: %s", binding.Protocol)
	}
	out, err := drv.Command(ctx, res, binding, cmd, params)
	if err != nil {
		s.markCapability("device", res.DeviceID, cmd, false)
		return nil, err
	}
	s.markCapability("device", res.DeviceID, cmd, true)
	return out, nil
}

func (s *Service) setSession(resourceID string, sess *channeldrv.Session) {
	if sess == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[resourceID] = sess
}

func (s *Service) getSession(resourceID string) *channeldrv.Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.sessions[resourceID]
	if sess != nil {
		delete(s.sessions, resourceID)
	}
	return sess
}

// capabilitySupported 查某能力是否声明支持；exists=false 表示没有声明（不拦截）。
func (s *Service) capabilitySupported(scopeType, scopeID, operation string) (supported bool, exists bool) {
	caps, err := s.repo.ListCapabilities(scopeType, scopeID)
	if err != nil {
		return true, false
	}
	for _, c := range caps {
		if c.Operation == operation {
			return c.Supported, true
		}
	}
	return true, false
}

// markCapability 运行时更新能力可用性（首次失败降级，恢复后置回 true）。
func (s *Service) markCapability(scopeType, scopeID, operation string, available bool) {
	if err := s.repo.SetCapabilityAvailable(scopeType, scopeID, operation, available); err != nil {
		log.Printf("[asset] 更新能力状态失败 scope=%s/%s op=%s available=%v err=%v",
			scopeType, scopeID, operation, available, err)
	}
}

// capabilityAllowed 判断某能力当前是否可用：未声明放行；声明不支持或已降级则拒绝。
func (s *Service) capabilityAllowed(scopeType, scopeID, operation string) bool {
	caps, err := s.repo.ListCapabilities(scopeType, scopeID)
	if err != nil {
		return true
	}
	for _, c := range caps {
		if c.Operation != operation {
			continue
		}
		if !c.Supported {
			return false
		}
		if !c.Available {
			return false
		}
	}
	return true
}
