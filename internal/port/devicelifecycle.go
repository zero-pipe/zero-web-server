package port

import (
	domaindevice "zero-web-server/internal/domain/device"
	"zero-web-server/internal/gbsip"
)

// DeviceLifecycle 是 SIP 栈回调到应用层的设备生命周期端口（注册/心跳/目录/设备信息）。
// 由 device 应用服务实现，供 SIP 基础设施依赖，避免基础设施反向依赖具体服务类型。
type DeviceLifecycle interface {
	GetByDeviceID(deviceID string) (*domaindevice.Device, error)
	Online(device *domaindevice.Device) error
	Offline(device *domaindevice.Device) error
	SaveRegister(device *domaindevice.Device) (*domaindevice.Device, error)
	HandleKeepalive(deviceID, ip string, port int) error
	HandleCatalog(deviceID string, items []gbsip.CatalogItem) error
	HandleDeviceInfo(deviceID, name, manufacturer, model, firmware string) error
	OnDeviceOnline(device *domaindevice.Device)
}
