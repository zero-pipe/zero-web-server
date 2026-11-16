package port

import (
	"context"
	"errors"

	domaindevice "zero-web-server/internal/domain/device"
)

// ErrCacheMiss 表示缓存中没有该设备。
var ErrCacheMiss = errors.New("device cache miss")

// DeviceCache 是设备注册/心跳所需的缓存与过期记账端口（Redis 实现）。
type DeviceCache interface {
	UpdateDevice(ctx context.Context, device *domaindevice.Device) error
	GetDevice(ctx context.Context, deviceID string) (*domaindevice.Device, error)
	RemoveDevice(ctx context.Context, deviceID string) error
	RemoveDeviceExpiry(ctx context.Context, serverID, deviceID string) error
	SetDeviceExpiry(ctx context.Context, serverID, deviceID string, expireAtMs int64) error
	ListExpiredDevices(ctx context.Context, serverID string) ([]string, error)
	RemoveExpiredDevices(ctx context.Context, serverID string, ids []string) error
	GetKeepaliveTimeStamps(ctx context.Context, deviceID string, count int) ([]int64, error)
	GetRegisterTimeStamps(ctx context.Context, deviceID string, count int) ([]int64, error)
}

// NoopDeviceCache 在 Redis 不可用时使用，行为为空且不影响主流程。
type NoopDeviceCache struct{}

func (NoopDeviceCache) UpdateDevice(context.Context, *domaindevice.Device) error { return nil }
func (NoopDeviceCache) GetDevice(context.Context, string) (*domaindevice.Device, error) {
	return nil, ErrCacheMiss
}
func (NoopDeviceCache) RemoveDevice(context.Context, string) error               { return nil }
func (NoopDeviceCache) RemoveDeviceExpiry(context.Context, string, string) error { return nil }
func (NoopDeviceCache) SetDeviceExpiry(context.Context, string, string, int64) error {
	return nil
}
func (NoopDeviceCache) ListExpiredDevices(context.Context, string) ([]string, error) {
	return nil, nil
}
func (NoopDeviceCache) RemoveExpiredDevices(context.Context, string, []string) error {
	return nil
}
func (NoopDeviceCache) GetKeepaliveTimeStamps(context.Context, string, int) ([]int64, error) {
	return nil, nil
}
func (NoopDeviceCache) GetRegisterTimeStamps(context.Context, string, int) ([]int64, error) {
	return nil, nil
}
