package server

import (
	"context"
	"time"

	"github.com/zero-pipe/gb28181-go/manscdp"
)

// AuthResolver supplies Digest credentials and pre-register policy.
type AuthResolver interface {
	// ResolvePassword returns the Digest password for deviceID.
	// known=false means the device is not in the host registry.
	ResolvePassword(deviceID string) (password string, known bool, err error)
}

// RegisterHandler handles REGISTER success / unregister.
type RegisterHandler interface {
	OnRegister(ctx context.Context, ev RegisterEvent) error
	OnUnregister(ctx context.Context, deviceID string) error
}

// MessageHandler handles inbound MANSCDP Notify/Response (except RecordInfo/PresetQuery waiters).
type MessageHandler interface {
	// DeviceKnown reports whether MESSAGE from deviceID should be accepted (else 404).
	DeviceKnown(deviceID string) bool
	OnKeepalive(ctx context.Context, deviceID, ip string, port int) error
	OnCatalog(ctx context.Context, deviceID string, items []manscdp.CatalogItem) error
	OnDeviceInfo(ctx context.Context, deviceID, name, manufacturer, model, firmware string) error
	OnAlarm(ctx context.Context, deviceID, channelID string, alarm *manscdp.AlarmNotify) error
	OnMobilePosition(ctx context.Context, deviceID, channelID string, pos *manscdp.MobilePositionNotify) error
}

// TelemetryHook is optional side-effect hooks (e.g. Redis timestamps).
type TelemetryHook interface {
	OnRegister(deviceID string, ts time.Time)
	OnKeepalive(deviceID string, ts time.Time)
}

// Handlers bundles callbacks for the SIP server.
type Handlers struct {
	Auth      AuthResolver
	Register  RegisterHandler
	Message   MessageHandler
	Telemetry TelemetryHook
}

// RegisterEvent carries REGISTER-derived fields for the host to persist.
type RegisterEvent struct {
	DeviceID       string
	IP             string
	Port           int
	Transport      string
	Expires        int
	CallID         string
	LocalIP        string
	ServerID       string
	IsNewDevice    bool // known=false before register (auto-create candidate)
}
