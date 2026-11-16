package deviceaccess

import "context"

// gbAdapter 国标 GB28181（被动接入）适配器。
type gbAdapter struct{ s *Service }

func (a gbAdapter) Protocol() string                { return ProtocolGB28181 }
func (a gbAdapter) Supports(accessMode string) bool { return accessMode == AccessPassive }

func (a gbAdapter) List(_ context.Context, q ListQuery) ([]*DeviceView, error) {
	var online *bool
	switch q.Status {
	case StatusOnline:
		v := true
		online = &v
	case StatusOffline, StatusPending:
		v := false
		online = &v
	}
	list, _, err := a.s.gb.List(1, 5000, q.Query, online)
	if err != nil {
		return nil, err
	}
	views := make([]*DeviceView, 0, len(list))
	for _, d := range list {
		v := viewFromGB(d)
		if matchStatus(v, q.Status) {
			views = append(views, v)
		}
	}
	return views, nil
}

func (a gbAdapter) GetByInternalCode(_ context.Context, code string) (*DeviceView, bool, error) {
	if d, err := a.s.gb.GetByInternalCode(code); err == nil && d != nil {
		return viewFromGB(d), true, nil
	}
	return nil, false, nil
}

func (a gbAdapter) Create(_ context.Context, req CreateRequest) (*DeviceView, error) {
	return a.s.createGB(req)
}

func (a gbAdapter) Update(_ context.Context, rawID string, req UpdateRequest) (*DeviceView, error) {
	return a.s.updateGB(rawID, req)
}

func (a gbAdapter) Delete(_ context.Context, rawID string) error {
	return a.s.gb.Delete(rawID)
}

func (a gbAdapter) Sync(_ context.Context, rawID string) error {
	return a.s.gb.SyncCatalog(rawID)
}
