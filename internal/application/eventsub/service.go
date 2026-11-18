package eventsub

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	infraapp "zero-web-server/internal/application/infraconfig"
	ipcdevice "zero-web-server/internal/application/ipcdevice"
	objectstoreapp "zero-web-server/internal/application/objectstore"
	domainevent "zero-web-server/internal/domain/eventsub"
	domainplugin "zero-web-server/internal/domain/plugin"
	"zero-web-server/internal/port"
	applog "zero-web-server/pkg/log"
)

type Service struct {
	repo        domainevent.Repository
	ipc         *ipcdevice.Service
	plugins     domainplugin.Repository
	objectStore *objectstoreapp.Service
	infra       *infraapp.Service
}

func NewService(
	repo domainevent.Repository,
	ipc *ipcdevice.Service,
	plugins domainplugin.Repository,
	objectStore *objectstoreapp.Service,
	infra *infraapp.Service,
) *Service {
	return &Service{repo: repo, ipc: ipc, plugins: plugins, objectStore: objectStore, infra: infra}
}

func (s *Service) Get(deviceID int) (*domainevent.Subscription, error) {
	sub, err := s.repo.GetByDevice(deviceID)
	if err != nil {
		return nil, nil
	}
	return sub, nil
}

// Arm 校验基础设施、下发 MQTT/对象存储配置，并打开插件报警通道。
func (s *Service) Arm(ctx context.Context, deviceID int, eventTypes []string) (*domainevent.Subscription, error) {
	dev, err := s.plugins.GetDevice(deviceID)
	if err != nil || dev == nil {
		return nil, fmt.Errorf("设备不存在")
	}
	p, err := s.plugins.GetPlugin(dev.PluginID)
	if err != nil || p == nil {
		return nil, fmt.Errorf("插件不存在")
	}
	if !p.Online {
		return nil, fmt.Errorf("插件离线，无法布防")
	}
	if !supportsAlarm(p) {
		return nil, fmt.Errorf("插件未声明报警/布防能力")
	}
	if strings.TrimSpace(dev.InternalCode) == "" {
		return nil, fmt.Errorf("设备缺少内码，无法关联事件")
	}
	if !dev.Enabled {
		return nil, fmt.Errorf("设备已禁用")
	}

	objCfg, err := s.objectStore.GetOrEmpty()
	if err != nil {
		return nil, fmt.Errorf("读取对象存储配置失败: %w", err)
	}
	if objCfg == nil || !objCfg.Enabled {
		return nil, fmt.Errorf("请先在「存储管理 → 对象存储」启用并配置对象存储")
	}
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = s.objectStore.Health(healthCtx)
	cancel()
	if err != nil {
		return nil, fmt.Errorf("对象存储不可用: %w", err)
	}

	mqttCfg, err := s.infra.MQTTConfigRaw()
	if err != nil || mqttCfg == nil {
		return nil, fmt.Errorf("读取 MQTT 配置失败")
	}
	if !mqttCfg.Enabled {
		return nil, fmt.Errorf("请先在「存储管理 → 基础设施 → 消息队列」启用 MQTT")
	}
	if connected, msg := s.infra.MQTTStatus(); !connected {
		if msg == "" {
			msg = "未连接"
		}
		return nil, fmt.Errorf("MQTT 平台消费者未连接: %s", msg)
	}

	if len(eventTypes) == 0 {
		eventTypes = []string{"motion"}
	}
	topicPrefix := strings.Trim(strings.TrimSpace(mqttCfg.TopicPrefix), "/")
	if topicPrefix == "" {
		topicPrefix = "zws/events"
	}
	cfg := port.IPCArmConfig{
		DeviceUID:          dev.DeviceUID,
		DeviceInternalCode: dev.InternalCode,
		EventTypes:         eventTypes,
		MQTT: port.IPCArmMQTT{
			Broker: mqttCfg.Broker, ClientID: publisherClientID(p.ID, dev.DeviceUID),
			Username: mqttCfg.Username, Password: mqttCfg.Password,
			Topic: topicPrefix + "/" + dev.InternalCode, QoS: mqttCfg.QoS,
			KeepAlive: mqttCfg.KeepAliveSec, TLSEnabled: mqttCfg.TLSEnabled,
			CACert: mqttCfg.CACert, ClientCert: mqttCfg.ClientCert, ClientKey: mqttCfg.ClientKey,
		},
		ObjectStore: port.IPCArmObjectStore{
			Provider: objCfg.Provider, Endpoint: objCfg.Endpoint, Region: objCfg.Region,
			Bucket: objCfg.Bucket, AccessKey: objCfg.AccessKey, SecretKey: objCfg.SecretKey,
			UseSSL: objCfg.UseSSL, PathStyle: objCfg.PathStyle, PublicBase: objCfg.PublicBase,
		},
	}

	sub := &domainevent.Subscription{
		DeviceID: deviceID, PluginID: dev.PluginID, DeviceUID: dev.DeviceUID,
		DeviceInternalCode: dev.InternalCode, EventTypes: eventTypes,
	}
	res, armErr := s.ipc.Arm(ctx, deviceID, cfg)
	if armErr != nil {
		sub.Armed = false
		sub.Status = "error"
		sub.LastError = armErr.Error()
		_ = s.repo.Save(sub)
		return nil, fmt.Errorf("插件布防失败: %w", armErr)
	}
	if res == nil || !res.Armed {
		sub.Armed = false
		sub.Status = "error"
		sub.LastError = "插件未返回 armed"
		_ = s.repo.Save(sub)
		return nil, fmt.Errorf("插件未确认布防")
	}
	sub.Armed = true
	sub.Status = "armed"
	sub.LastError = ""
	sub.RemoteID = res.SubscriptionID
	sub.LastArmedAt = time.Now().UnixMilli()
	if err := s.repo.Save(sub); err != nil {
		return nil, err
	}
	applog.Info("event subscription armed", "device", deviceID, "uid", dev.DeviceUID, "internal", dev.InternalCode)
	return sub, nil
}

func (s *Service) Disarm(ctx context.Context, deviceID int) error {
	sub, err := s.repo.GetByDevice(deviceID)
	if err != nil || sub == nil {
		return fmt.Errorf("设备未布防")
	}
	disarmErr := s.ipc.Disarm(ctx, deviceID, sub.RemoteID)
	sub.Armed = false
	sub.Status = "disarmed"
	if disarmErr != nil {
		sub.Status = "error"
		sub.LastError = disarmErr.Error()
	} else {
		sub.LastError = ""
	}
	_ = s.repo.Save(sub)
	if disarmErr != nil {
		return fmt.Errorf("插件撤防失败: %w", disarmErr)
	}
	applog.Info("event subscription disarmed", "device", deviceID)
	return nil
}

// RearmForPlugin 插件重启/重连后，自动补登录并补布防。
func (s *Service) RearmForPlugin(ctx context.Context, pluginID string) {
	list, err := s.repo.ListArmedByPlugin(pluginID)
	if err != nil || len(list) == 0 {
		return
	}
	for _, sub := range list {
		if err := s.ipc.EnsureLogin(ctx, sub.DeviceID); err != nil {
			applog.Warn("rearm ensure login failed", "device", sub.DeviceID, "err", err)
			continue
		}
		if _, err := s.Arm(ctx, sub.DeviceID, sub.EventTypes); err != nil {
			applog.Warn("rearm failed", "device", sub.DeviceID, "err", err)
			continue
		}
		applog.Info("rearmed after plugin hello", "device", sub.DeviceID, "plugin", pluginID)
	}
}

func supportsAlarm(p *domainplugin.Plugin) bool {
	for _, c := range p.Capabilities {
		switch strings.ToLower(strings.TrimSpace(c)) {
		case "alarm", "arming":
			return true
		}
	}
	return false
}

var unsafeID = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

func publisherClientID(pluginID, deviceUID string) string {
	raw := fmt.Sprintf("%s-%s-publisher", pluginID, deviceUID)
	return unsafeID.ReplaceAllString(raw, "-")
}
