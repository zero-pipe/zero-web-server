package deviceapp

import (
	"sync"
	"time"
	"zero-web-server/internal/apperr"

	domainchannel "zero-web-server/internal/domain/channel"
	domaindevice "zero-web-server/internal/domain/device"
	"zero-web-server/internal/port"
)

var (
	ErrDeviceNotFound         = apperr.New(apperr.CodeDevice+1, "设备不存在")
	ErrDeviceNotPreRegistered = apperr.New(apperr.CodeDevice+2, "设备未预添加，拒绝注册")
)

type Service struct {
	devices            domaindevice.Repository
	channels           domainchannel.Repository
	redis              port.DeviceCache
	sip                port.GBDeviceControl
	requirePreRegister bool
	syncStatus         sync.Map // deviceID -> *SyncStatus
	renewal            *subscribeRenewal
}

// SetRequirePreRegister 未预添加是否禁止 SaveRegister 自动建档

func (s *Service) SetRequirePreRegister(v bool) {
	s.requirePreRegister = v
}

type SyncStatus struct {
	Total    int
	Current  int
	SyncIng  bool
	ErrorMsg string
	Started  time.Time
}

func NewService(devices domaindevice.Repository, channels domainchannel.Repository, redis port.DeviceCache) *Service {
	return &Service{
		devices:  devices,
		channels: channels,
		redis:    redis,
		renewal:  &subscribeRenewal{},
	}
}

func (s *Service) SetSIP(sipServer port.GBDeviceControl) {
	s.sip = sipServer
}
