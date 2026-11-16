package deviceaccess

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// onvifAdapter ONVIF（主动接入）适配器。
type onvifAdapter struct{ s *Service }

func (a onvifAdapter) Protocol() string                { return ProtocolONVIF }
func (a onvifAdapter) Supports(accessMode string) bool { return accessMode == AccessActive }

func (a onvifAdapter) List(ctx context.Context, q ListQuery) ([]*DeviceView, error) {
	list, _, err := a.s.onvif.ListDevices(ctx, 1, 5000, q.Query)
	if err != nil {
		return nil, err
	}
	views := make([]*DeviceView, 0, len(list))
	for _, d := range list {
		v := viewFromONVIF(d)
		if n, err := a.s.onvif.ChannelCount(ctx, d.ID); err == nil {
			v.ChannelCount = n
		}
		if matchStatus(v, q.Status) {
			views = append(views, v)
		}
	}
	return views, nil
}

func (a onvifAdapter) GetByInternalCode(ctx context.Context, code string) (*DeviceView, bool, error) {
	list, _, err := a.s.onvif.ListDevices(ctx, 1, 5000, code)
	if err != nil {
		return nil, false, err
	}
	for _, d := range list {
		if d != nil && strings.EqualFold(strings.TrimSpace(d.InternalCode), code) {
			v := viewFromONVIF(d)
			if n, err := a.s.onvif.ChannelCount(ctx, d.ID); err == nil {
				v.ChannelCount = n
			}
			return v, true, nil
		}
	}
	return nil, false, nil
}

func (a onvifAdapter) Create(ctx context.Context, req CreateRequest) (*DeviceView, error) {
	return a.s.createONVIF(ctx, req)
}

func (a onvifAdapter) Update(ctx context.Context, rawID string, req UpdateRequest) (*DeviceView, error) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("无效的 ONVIF 设备ID")
	}
	return a.s.updateONVIF(ctx, id, req)
}

func (a onvifAdapter) Delete(ctx context.Context, rawID string) error {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return fmt.Errorf("无效的 ONVIF 设备ID")
	}
	return a.s.onvif.DeleteDevice(ctx, id)
}

func (a onvifAdapter) Sync(ctx context.Context, rawID string) error {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return fmt.Errorf("无效的 ONVIF 设备ID")
	}
	_, err = a.s.onvif.SyncChannels(ctx, id)
	return err
}
