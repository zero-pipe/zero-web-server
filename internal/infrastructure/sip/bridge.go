package sipinfra

import (
	"context"
	"fmt"
	"strconv"
	"time"

	domaindevice "zero-web-server/internal/domain/device"
	"zero-web-server/internal/infrastructure/config"
	redisinfra "zero-web-server/internal/infrastructure/redis"

	"github.com/zero-pipe/gb28181-go/manscdp"
	gbserver "github.com/zero-pipe/gb28181-go/server"
)

// SubordinateHandler routes REGISTER/MESSAGE for downstream platforms.
type SubordinateHandler interface {
	ResolvePassword(gbID string) (password string, known bool, err error)
	Known(gbID string) bool
	Exists(gbID string) bool
	OnRegister(ev gbserver.RegisterEvent) error
	OnUnregister(gbID string) error
	OnKeepalive(gbID, ip string, port int) error
}

// bridge adapts ZWS device/alarm/position/redis into gb28181-go handlers.
type bridge struct {
	deviceSvc   DeviceService
	subordinate SubordinateHandler
	cascade     gbserver.CascadeInboundHandler
	alarm       AlarmHandler
	position    PositionHandler
	redis       *redisinfra.Client
	password    string
	serverID    string
}

func (b *bridge) ResolvePassword(deviceID string) (string, bool, error) {
	if b.subordinate != nil {
		if p, known, err := b.subordinate.ResolvePassword(deviceID); err != nil {
			return "", false, err
		} else if known {
			return p, true, nil
		}
	}
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
	if b.subordinate != nil && b.subordinate.Known(ev.DeviceID) {
		return b.subordinate.OnRegister(ev)
	}
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
	if b.subordinate != nil && b.subordinate.Exists(deviceID) {
		return b.subordinate.OnUnregister(deviceID)
	}
	if device, err := b.deviceSvc.GetByDeviceID(deviceID); err == nil && device != nil {
		_ = b.deviceSvc.Offline(device)
	}
	return nil
}

func (b *bridge) DeviceKnown(deviceID string) bool {
	if b.subordinate != nil && b.subordinate.Known(deviceID) {
		return true
	}
	d, err := b.deviceSvc.GetByDeviceID(deviceID)
	return err == nil && d != nil
}

func (b *bridge) UpstreamKnown(upstreamGBID string) bool {
	if b.cascade == nil {
		return false
	}
	return b.cascade.UpstreamKnown(upstreamGBID)
}

func (b *bridge) OnDeviceControl(ctx context.Context, ev gbserver.InboundControlEvent) error {
	if b.cascade == nil {
		return nil
	}
	return b.cascade.OnDeviceControl(ctx, ev)
}

func (b *bridge) OnInvite(ctx context.Context, ev gbserver.InboundInviteEvent) ([]byte, error) {
	if b.cascade == nil {
		return nil, fmt.Errorf("cascade inbound not configured")
	}
	return b.cascade.OnInvite(ctx, ev)
}

func (b *bridge) OnInviteEnd(ctx context.Context, callID string) error {
	if b.cascade == nil {
		return nil
	}
	return b.cascade.OnInviteEnd(ctx, callID)
}

func (b *bridge) OnKeepalive(_ context.Context, deviceID, ip string, port int) error {
	if b.subordinate != nil && b.subordinate.Known(deviceID) {
		return b.subordinate.OnKeepalive(deviceID, ip, port)
	}
	return b.deviceSvc.HandleKeepalive(deviceID, ip, port)
}

func (b *bridge) OnCatalog(_ context.Context, deviceID string, items []manscdp.CatalogItem) error {
	if b.subordinate != nil && b.subordinate.Known(deviceID) {
		// 下级目录入库留给后续通道映射波次；此处先保活在线即可
		_ = b.subordinate.OnKeepalive(deviceID, "", 0)
		return nil
	}
	return b.deviceSvc.HandleCatalog(deviceID, items)
}

func (b *bridge) OnDeviceInfo(_ context.Context, deviceID, name, manufacturer, model, firmware string) error {
	if b.subordinate != nil && b.subordinate.Known(deviceID) {
		return nil
	}
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
