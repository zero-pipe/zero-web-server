package sipinfra

import (
	"context"
	"strconv"
	"time"

	domaindevice "zero-web-kit/internal/domain/device"
	"zero-web-kit/internal/infrastructure/config"
	redisinfra "zero-web-kit/internal/infrastructure/redis"

	"github.com/zero-pipe/gb28181-go/manscdp"
	gbserver "github.com/zero-pipe/gb28181-go/server"
)

// bridge adapts ZWS device/alarm/position/redis into gb28181-go handlers.
type bridge struct {
	deviceSvc DeviceService
	alarm     AlarmHandler
	position  PositionHandler
	redis     *redisinfra.Client
	password  string
	serverID  string
}

func (b *bridge) ResolvePassword(deviceID string) (string, bool, error) {
	d, err := b.deviceSvc.GetByDeviceID(deviceID)
	if err != nil || d == nil {
		return b.password, false, nil
	}
	if d.Password != "" {
		return d.Password, true, nil
	}
	return b.password, true, nil
}

func (b *bridge) OnRegister(ctx context.Context, ev gbserver.RegisterEvent) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	var device *domaindevice.Device
	if existing, err := b.deviceSvc.GetByDeviceID(ev.DeviceID); err == nil && existing != nil {
		device = existing
	} else {
		device = &domaindevice.Device{
			DeviceID:          ev.DeviceID,
			Name:              ev.DeviceID,
			StreamMode:        "UDP",
			Charset:           "GB2312",
			MediaServerID:     "auto",
			HeartBeatInterval: 60,
			HeartBeatCount:    3,
			ServerID:          b.serverID,
			CreateTime:        now,
		}
	}
	device.IP = ev.IP
	device.Port = ev.Port
	device.HostAddress = ev.IP + ":" + strconv.Itoa(ev.Port)
	device.Expires = ev.Expires
	device.Transport = ev.Transport
	device.UpdateTime = now
	device.RegisterCallID = ev.CallID
	if ev.LocalIP != "" {
		device.LocalIP = ev.LocalIP
	}
	if b.serverID != "" {
		device.ServerID = b.serverID
	}
	saved, err := b.deviceSvc.SaveRegister(device)
	if err != nil {
		return err
	}
	_ = b.deviceSvc.Online(saved)
	b.deviceSvc.OnDeviceOnline(saved)
	return nil
}

func (b *bridge) OnUnregister(_ context.Context, deviceID string) error {
	if device, err := b.deviceSvc.GetByDeviceID(deviceID); err == nil && device != nil {
		_ = b.deviceSvc.Offline(device)
	}
	return nil
}

func (b *bridge) DeviceKnown(deviceID string) bool {
	d, err := b.deviceSvc.GetByDeviceID(deviceID)
	return err == nil && d != nil
}

func (b *bridge) OnKeepalive(_ context.Context, deviceID, ip string, port int) error {
	return b.deviceSvc.HandleKeepalive(deviceID, ip, port)
}

func (b *bridge) OnCatalog(_ context.Context, deviceID string, items []manscdp.CatalogItem) error {
	return b.deviceSvc.HandleCatalog(deviceID, items)
}

func (b *bridge) OnDeviceInfo(_ context.Context, deviceID, name, manufacturer, model, firmware string) error {
	return b.deviceSvc.HandleDeviceInfo(deviceID, name, manufacturer, model, firmware)
}

func (b *bridge) OnDeviceStatus(_ context.Context, deviceID string, status *manscdp.DeviceStatus) error {
	_ = deviceID
	_ = status
	return nil
}

func (b *bridge) OnMediaStatus(_ context.Context, deviceID string, status *manscdp.MediaStatusNotify) error {
	_ = deviceID
	_ = status
	return nil
}

func (b *bridge) OnDeviceControlResult(_ context.Context, deviceID, sn, result string) error {
	_ = deviceID
	_ = sn
	_ = result
	return nil
}

func (b *bridge) OnAlarm(_ context.Context, deviceID, channelID string, alarm *manscdp.AlarmNotify) error {
	if b.alarm == nil {
		return nil
	}
	return b.alarm.HandleNotify(deviceID, channelID, alarm)
}

func (b *bridge) OnMobilePosition(_ context.Context, deviceID, channelID string, pos *manscdp.MobilePositionNotify) error {
	if b.position == nil {
		return nil
	}
	return b.position.HandleNotify(deviceID, channelID, pos)
}

type telemetryBridge struct{ b *bridge }

func (t telemetryBridge) OnRegister(deviceID string, ts time.Time) {
	if t.b.redis != nil {
		_ = t.b.redis.PushRegister(context.Background(), deviceID, ts.UnixMilli())
	}
}

func (t telemetryBridge) OnKeepalive(deviceID string, ts time.Time) {
	if t.b.redis != nil {
		_ = t.b.redis.PushKeepalive(context.Background(), deviceID, ts.UnixMilli())
	}
}

func toLibConfig(cfg config.SIPConfig, serverID string, requirePreRegister bool) gbserver.Config {
	return gbserver.Config{
		ID:                 cfg.ID,
		Domain:             cfg.Domain,
		Password:           cfg.Password,
		IP:                 cfg.IP,
		Port:               cfg.Port,
		RequirePreRegister: requirePreRegister,
		UserAgent:          cfg.ID,
		ServerID:           serverID,
	}
}

func toPeer(d *domaindevice.Device) gbserver.Peer {
	if d == nil {
		return gbserver.Peer{}
	}
	return gbserver.Peer{
		DeviceID:       d.DeviceID,
		IP:             d.IP,
		Port:           d.Port,
		Password:       d.Password,
		LocalIP:        d.LocalIP,
		SDPIP:          d.SDPIP,
		Transport:      d.Transport,
		HostAddress:    d.HostAddress,
		Expires:        d.Expires,
		RegisterCallID: d.RegisterCallID,
	}
}
