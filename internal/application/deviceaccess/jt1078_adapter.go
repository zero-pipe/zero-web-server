package deviceaccess

import "context"

// jt1078Adapter 部标 JT1078 占位适配器：接入实现后只需替换本文件即可。
type jt1078Adapter struct{}

func (jt1078Adapter) Protocol() string                { return ProtocolJT1078 }
func (jt1078Adapter) Supports(accessMode string) bool { return accessMode == ProtocolJT1078 }

func (jt1078Adapter) List(context.Context, ListQuery) ([]*DeviceView, error) {
	return []*DeviceView{}, nil
}
func (jt1078Adapter) GetByInternalCode(context.Context, string) (*DeviceView, bool, error) {
	return nil, false, nil
}
func (jt1078Adapter) Create(context.Context, CreateRequest) (*DeviceView, error) {
	return nil, ErrProtocolNotImplemented
}
func (jt1078Adapter) Update(context.Context, string, UpdateRequest) (*DeviceView, error) {
	return nil, ErrProtocolNotImplemented
}
func (jt1078Adapter) Delete(context.Context, string) error { return ErrProtocolNotImplemented }
func (jt1078Adapter) Sync(context.Context, string) error   { return ErrProtocolNotImplemented }
