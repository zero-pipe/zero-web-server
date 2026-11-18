package eventingest

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	objectstoreapp "zero-web-server/internal/application/objectstore"
	domainasset "zero-web-server/internal/domain/asset"
	domainevent "zero-web-server/internal/domain/event"
	applog "zero-web-server/pkg/log"
)

type Service struct {
	repo        domainevent.Repository
	assets      domainasset.Repository
	objectStore *objectstoreapp.Service

	received  atomic.Uint64
	stored    atomic.Uint64
	duplicate atomic.Uint64
	invalid   atomic.Uint64
	failed    atomic.Uint64
	unmapped  atomic.Uint64
}

func NewService(repo domainevent.Repository, assets domainasset.Repository, objectStore *objectstoreapp.Service) *Service {
	return &Service{repo: repo, assets: assets, objectStore: objectStore}
}

type payload struct {
	Schema             string `json:"schema"`
	EventID            string `json:"eventId"`
	DeviceInternalCode string `json:"deviceInternalCode"`
	DeviceUID          string `json:"deviceUid"`
	ChannelID          string `json:"channelId"`
	EventType          string `json:"eventType"`
	Severity           string `json:"severity"`
	OccurredAt         int64  `json:"occurredAt"`
	ReceivedAt         int64  `json:"receivedAt"`
	Image              *struct {
		Key       string `json:"key"`
		URL       string `json:"url"`
		ExpiresAt int64  `json:"expiresAt"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	} `json:"image"`
	BBox json.RawMessage `json:"bbox"`
	Raw  json.RawMessage `json:"raw"`
}

// HandleMessage 消费 MQTT 事件并持久化。
func (s *Service) HandleMessage(topic string, body []byte) error {
	s.received.Add(1)
	var p payload
	if err := json.Unmarshal(body, &p); err != nil {
		s.invalid.Add(1)
		applog.Warn("event payload invalid", "topic", topic, "err", err)
		return fmt.Errorf("invalid event payload: %w", err)
	}
	p.EventID = strings.TrimSpace(p.EventID)
	p.DeviceInternalCode = strings.TrimSpace(p.DeviceInternalCode)
	p.ChannelID = strings.TrimSpace(p.ChannelID)
	p.EventType = strings.ToLower(strings.TrimSpace(p.EventType))
	if p.EventID == "" || p.DeviceInternalCode == "" || p.EventType == "" {
		s.invalid.Add(1)
		applog.Warn("event payload missing required fields", "topic", topic)
		return fmt.Errorf("event payload missing required fields")
	}
	exists, err := s.repo.Exists(p.EventID)
	if err != nil {
		s.failed.Add(1)
		return fmt.Errorf("check event duplicate: %w", err)
	}
	if exists {
		s.duplicate.Add(1)
		applog.Debugf("event duplicate ignored event=%s", p.EventID)
		return nil
	}

	e := &domainevent.Event{
		EventID: p.EventID, SchemaVersion: p.Schema, DeviceInternalCode: p.DeviceInternalCode,
		DeviceUID: p.DeviceUID, ChannelExternalKey: p.ChannelID, EventType: p.EventType,
		Severity:   firstNonEmpty(p.Severity, "warning"),
		OccurredAt: p.OccurredAt, ReceivedAt: p.ReceivedAt,
		BBoxJSON: string(p.BBox), RawJSON: string(body), Status: "unmapped",
	}
	if e.OccurredAt <= 0 {
		e.OccurredAt = time.Now().UnixMilli()
	}
	if e.ReceivedAt <= 0 {
		e.ReceivedAt = time.Now().UnixMilli()
	}
	if p.Image != nil {
		e.ImageKey = p.Image.Key
		e.ImageURL = p.Image.URL
		e.ImageExpiresAt = p.Image.ExpiresAt
	}

	// deviceInternalCode → 物理设备 → 资源 / 事件源
	if s.assets != nil {
		if dev, err := s.assets.GetDeviceByInternalCode(p.DeviceInternalCode); err == nil && dev != nil {
			e.DeviceID = dev.ID
			e.Status = "mapped"
			if binding, err := s.assets.GetBindingByDevice(dev.ID); err == nil && binding != nil {
				e.BindingID = binding.ID
			}
			if resources, err := s.assets.ListResources(dev.ID); err == nil {
				e.ResourceID = matchResource(resources, p.ChannelID)
			}
			if sources, err := s.assets.ListEventSources(dev.ID); err == nil {
				for _, src := range sources {
					if strings.EqualFold(src.Kind, p.EventType) {
						e.SourceID = src.ID
						break
					}
				}
			}
		}
	}

	if err := s.repo.Create(e); err != nil {
		s.failed.Add(1)
		applog.Warn("event persist failed", "event", p.EventID, "err", err)
		return fmt.Errorf("persist event: %w", err)
	}
	s.stored.Add(1)
	if e.Status == "unmapped" {
		s.unmapped.Add(1)
	}
	applog.Info("event stored", "event", p.EventID, "type", p.EventType,
		"device", e.DeviceID, "internal", e.DeviceInternalCode, "status", e.Status)
	return nil
}

func (s *Service) Stats() map[string]uint64 {
	return map[string]uint64{
		"received": s.received.Load(), "stored": s.stored.Load(),
		"duplicate": s.duplicate.Load(), "invalid": s.invalid.Load(),
		"failed": s.failed.Load(), "unmapped": s.unmapped.Load(),
	}
}

func (s *Service) ClearBefore(ctx context.Context, before int64) (int64, error) {
	if before <= 0 {
		return 0, fmt.Errorf("before 时间不能为空")
	}
	return s.repo.DeleteBefore(before)
}

func (s *Service) List(ctx context.Context, page, count int, f domainevent.ListFilter) ([]*domainevent.Event, int64, error) {
	list, total, err := s.repo.List(page, count, f)
	if err != nil {
		return nil, 0, err
	}
	for _, e := range list {
		s.refreshImageURL(ctx, e)
	}
	return list, total, nil
}

func (s *Service) Get(ctx context.Context, id string) (*domainevent.Event, error) {
	e, err := s.repo.Get(id)
	if err != nil {
		return nil, fmt.Errorf("事件不存在")
	}
	s.refreshImageURL(ctx, e)
	return e, nil
}

func (s *Service) refreshImageURL(ctx context.Context, e *domainevent.Event) {
	if e == nil || e.ImageKey == "" || s.objectStore == nil {
		return
	}
	cfg, err := s.objectStore.GetOrEmpty()
	if err != nil || cfg == nil {
		return
	}
	if cfg.PublicBase != "" {
		e.ImageURL = strings.TrimRight(cfg.PublicBase, "/") + "/" + strings.TrimLeft(e.ImageKey, "/")
		return
	}
	c, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if u, perr := s.objectStore.Store().PresignGet(c, e.ImageKey, time.Hour); perr == nil && u != "" {
		e.ImageURL = u
	}
}

// matchResource 兼顾插件通道号(1)与 ISAPI 通道号(101)两种写法。
func matchResource(resources []*domainasset.Resource, channel string) string {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return ""
	}
	for _, r := range resources {
		if strings.TrimSpace(r.ExternalKey) == channel {
			return r.ID
		}
	}
	candidates := map[string]bool{}
	if n, err := strconv.Atoi(channel); err == nil {
		candidates[strconv.Itoa(n*100+1)] = true
		if n > 100 && (n-1)%100 == 0 {
			candidates[strconv.Itoa((n-1)/100)] = true
		}
	}
	for _, r := range resources {
		key := strings.TrimSpace(r.ExternalKey)
		if candidates[key] {
			return r.ID
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
