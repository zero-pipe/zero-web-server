package deviceaccess

import (
	"context"
	"fmt"
)

// ProtocolJT1078 部标 JT1078（占位，待接入）。
const ProtocolJT1078 = "jt1078"

// ErrProtocolNotImplemented 协议适配器尚未实现。
var ErrProtocolNotImplemented = fmt.Errorf("协议适配器尚未实现")

// ProtocolAdapter 设备协议适配器：每种接入协议实现一份，注册后由门面统一派发。
type ProtocolAdapter interface {
	Protocol() string
	// Supports 该适配器是否处理给定的接入模式（passive/active/...）。
	Supports(accessMode string) bool
	List(ctx context.Context, q ListQuery) ([]*DeviceView, error)
	GetByInternalCode(ctx context.Context, code string) (*DeviceView, bool, error)
	Create(ctx context.Context, req CreateRequest) (*DeviceView, error)
	Update(ctx context.Context, rawID string, req UpdateRequest) (*DeviceView, error)
	Delete(ctx context.Context, rawID string) error
	Sync(ctx context.Context, rawID string) error
}

type protocolRegistry struct {
	adapters []ProtocolAdapter
}

func newProtocolRegistry(adapters ...ProtocolAdapter) *protocolRegistry {
	return &protocolRegistry{adapters: adapters}
}

func (r *protocolRegistry) all() []ProtocolAdapter { return r.adapters }

func (r *protocolRegistry) byProtocol(protocol string) (ProtocolAdapter, bool) {
	for _, a := range r.adapters {
		if a.Protocol() == protocol {
			return a, true
		}
	}
	return nil, false
}

func (r *protocolRegistry) forCreate(accessMode, protocol string) (ProtocolAdapter, error) {
	if a, ok := r.byProtocol(protocol); ok && a.Supports(accessMode) {
		return a, nil
	}
	return nil, fmt.Errorf("不支持的接入组合: mode=%s protocol=%s", accessMode, protocol)
}
