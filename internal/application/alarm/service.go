package alarmapp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	domainalarm "zero-web-server/internal/domain/alarm"
	domainasset "zero-web-server/internal/domain/asset"
	domainchannel "zero-web-server/internal/domain/channel"
	domaindevice "zero-web-server/internal/domain/device"
	domainevent "zero-web-server/internal/domain/event"
	"zero-web-server/internal/domain/shared"
	"zero-web-server/internal/eventtype"
	"zero-web-server/internal/gbsip"
)

type Service struct {
	alarms   domainalarm.Repository
	channels domainchannel.Repository
	devices  domaindevice.Repository
	events   domainevent.Repository
	assets   domainasset.Repository
}

func NewService(alarms domainalarm.Repository, channels domainchannel.Repository,
	devices domaindevice.Repository, events domainevent.Repository, assets domainasset.Repository) *Service {
	return &Service{alarms: alarms, channels: channels, devices: devices, events: events, assets: assets}
}

func (s *Service) HandleNotify(deviceID, channelGBID string, n *gbsip.AlarmNotify) error {
	if n == nil {
		return nil
	}
	ch, err := s.channels.GetOne(deviceID, channelGBID)
	if err != nil {
		// try channelGBID as device channel id from notify DeviceID field
		return nil
	}
	alarmTime := time.Now().UnixMilli()
	if n.AlarmTime != "" {
		if t, err := time.ParseInLocation("2006-01-02T15:04:05", n.AlarmTime, time.Local); err == nil {
			alarmTime = t.UnixMilli()
		}
	}
	code := eventtype.FromGBAlarmType(n.AlarmType)
	var eventID, devInternal string
	if s.devices != nil {
		if dev, derr := s.devices.GetByDeviceID(deviceID); derr == nil && dev != nil {
			devInternal = dev.InternalCode
		}
	}
	if s.events != nil {
		eventID = fmt.Sprintf("gb-alarm-%s-%d-%d", deviceID, ch.ID, time.Now().UnixNano())
		raw, _ := json.Marshal(map[string]any{
			"alarmType": n.AlarmType, "alarmDescription": n.AlarmDescription,
			"alarmTime": n.AlarmTime, "channelId": channelGBID,
		})
		e := &domainevent.Event{
			EventID: eventID, SchemaVersion: "zws.event.v1",
			DeviceInternalCode: devInternal, DeviceName: deviceID,
			ChannelInternalCode: ch.InternalCode, ChannelExternalKey: ch.GBDeviceID,
			EventType: code, Category: eventtype.CategoryAlarm, Severity: "major",
			OccurredAt: alarmTime, ReceivedAt: time.Now().UnixMilli(),
			RawJSON: string(raw), Status: "mapped",
		}
		if devInternal == "" {
			e.Status = "unmapped"
		}
		if err := s.events.Create(e); err != nil {
			eventID = ""
		}
	}
	alarm := &domainalarm.Alarm{
		ChannelID:          ch.ID,
		Description:        n.AlarmDescription,
		Longitude:          n.Longitude,
		Latitude:           n.Latitude,
		AlarmType:          n.AlarmType,
		AlarmCode:          code,
		AlarmTime:          alarmTime,
		EventID:            eventID,
		DeviceInternalCode: devInternal, ChannelInternalCode: ch.InternalCode,
		Severity: "major", Status: "unhandled",
	}
	if err := s.alarms.Create(alarm); err != nil {
		return err
	}
	if eventID != "" && alarm.ID > 0 {
		// 事件表用 UUID 主键，这里按 EventID 反查更新；
		if e, err := s.events.Get(eventID); err == nil && e != nil {
			_ = s.events.UpdateAlarmID(e.ID, alarm.ID)
		}
	}
	return nil
}

func (s *Service) Get(id int) (*domainalarm.Alarm, error) {
	a, err := s.alarms.GetByID(id)
	if err != nil {
		return nil, err
	}
	s.enrich(a)
	return a, nil
}

func (s *Service) List(page, count int, alarmType *int, beginTime, endTime string) ([]*domainalarm.Alarm, int64, error) {
	list, total, err := s.alarms.List(page, count, alarmType, parseTimeMs(beginTime), parseTimeMs(endTime))
	if err != nil {
		return nil, 0, err
	}
	for _, a := range list {
		s.enrich(a)
	}
	return list, total, nil
}

// enrich 补齐旧页面依赖的通道名称与回放上下文。
func (s *Service) enrich(a *domainalarm.Alarm) {
	if a == nil {
		return
	}
	// 优先用统一资产（跨 GB/ONVIF/插件），旧通道表只作兜底。
	if s.assets != nil && a.ChannelInternalCode != "" {
		if dev, err := s.assets.GetDeviceByInternalCode(a.DeviceInternalCode); err == nil && dev != nil {
			a.DeviceName = dev.Name
		}
		if res, err := s.assets.GetResourceByInternalCode(a.ChannelInternalCode); err == nil && res != nil {
			a.ChannelName = res.Name
			a.ChannelDeviceID = res.ExternalKey
			supported := false
			protocol := ""
			if binding, berr := s.assets.GetBindingByDevice(res.DeviceID); berr == nil && binding != nil {
				protocol = binding.Protocol
				supported = protocol == "gb28181"
			}
			a.Playback = map[string]any{
				"supported":       supported,
				"protocol":        protocol,
				"channelLegacyId": res.LegacyChannelID,
				"startAt":         a.AlarmTime - 10_000,
				"endAt":           a.AlarmTime + 10_000,
				"reason":          map[bool]string{true: "", false: protocol + " 回放尚未接入"}[supported],
			}
			return
		}
	}
	if a.ChannelID <= 0 || s.channels == nil {
		return
	}
	ch, err := s.channels.GetByID(a.ChannelID)
	if err != nil || ch == nil {
		return
	}
	a.ChannelName = ch.Name
	a.ChannelDeviceID = ch.GBDeviceID
	if s.devices != nil {
		if dev, derr := s.devices.GetByDeviceID(ch.DeviceID); derr == nil && dev != nil {
			a.DeviceName = dev.Name
		}
	}
	a.Playback = map[string]any{
		"supported":       ch.DataType == shared.ChannelDataTypeGB28181,
		"protocol":        map[bool]string{true: "gb28181", false: "other"}[ch.DataType == shared.ChannelDataTypeGB28181],
		"channelLegacyId": a.ChannelID,
		"startAt":         a.AlarmTime - 10_000,
		"endAt":           a.AlarmTime + 10_000,
	}
}

func (s *Service) Delete(ids []int) error { return s.alarms.Delete(ids) }

func (s *Service) SetStatus(id int, status, handledBy, remark string) error {
	switch status {
	case "unhandled", "handled", "ignored":
	default:
		return fmt.Errorf("不支持的报警状态: %s", status)
	}
	return s.alarms.UpdateStatus(id, status, handledBy, remark, time.Now().UnixMilli())
}

func (s *Service) Clear(alarmType *int, beginTime, endTime string) error {
	return s.alarms.Clear(alarmType, parseTimeMs(beginTime), parseTimeMs(endTime))
}

func parseTimeMs(v string) int64 {
	if v == "" {
		return 0
	}
	if ms, err := strconv.ParseInt(v, 10, 64); err == nil {
		return ms
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", v, time.Local)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}
