package deviceapp

import (
	"context"
	"errors"
	"sync"
	"testing"

	domainchannel "zero-web-server/internal/domain/channel"
	domaindevice "zero-web-server/internal/domain/device"
	"zero-web-server/internal/gbsip"
)

// --- fakes ---

type fakeDeviceRepo struct {
	mu      sync.Mutex
	rows    map[string]*domaindevice.Device
	fields  map[string]map[string]interface{}
	online  map[string]bool
	created []string
}

func newFakeDeviceRepo() *fakeDeviceRepo {
	return &fakeDeviceRepo{
		rows:   map[string]*domaindevice.Device{},
		fields: map[string]map[string]interface{}{},
		online: map[string]bool{},
	}
}

func (f *fakeDeviceRepo) clone(d *domaindevice.Device) *domaindevice.Device {
	if d == nil {
		return nil
	}
	c := *d
	return &c
}

func (f *fakeDeviceRepo) GetByDeviceID(id string) (*domaindevice.Device, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.rows[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return f.clone(d), nil
}

func (f *fakeDeviceRepo) GetByID(id int) (*domaindevice.Device, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, d := range f.rows {
		if d.ID == id {
			return f.clone(d), nil
		}
	}
	return nil, errors.New("not found")
}

func (f *fakeDeviceRepo) Create(d *domaindevice.Device) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if d.ID == 0 {
		d.ID = len(f.rows) + 1
	}
	f.rows[d.DeviceID] = f.clone(d)
	f.created = append(f.created, d.DeviceID)
	return nil
}

func (f *fakeDeviceRepo) Update(d *domaindevice.Device) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[d.DeviceID] = f.clone(d)
	f.fields[d.DeviceID] = map[string]interface{}{"__full_update__": true}
	return nil
}

func (f *fakeDeviceRepo) UpdateFields(id string, fields map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fields[id] = fields
	if d, ok := f.rows[id]; ok {
		for k, v := range fields {
			switch k {
			case "ip":
				d.IP, _ = v.(string)
			case "port":
				d.Port, _ = v.(int)
			case "manufacturer":
				d.Manufacturer, _ = v.(string)
			case "model":
				d.Model, _ = v.(string)
			}
		}
	}
	return nil
}

func (f *fakeDeviceRepo) UpdateOnline(id string, online bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.online[id] = online
	if d, ok := f.rows[id]; ok {
		d.OnLine = online
	}
	return nil
}

func (f *fakeDeviceRepo) DeleteByDeviceID(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.rows, id)
	return nil
}

func (f *fakeDeviceRepo) List(page, count int, query string, online *bool) ([]*domaindevice.Device, int64, error) {
	return nil, 0, nil
}

func (f *fakeDeviceRepo) ListOnline() ([]*domaindevice.Device, error) { return nil, nil }

func (f *fakeDeviceRepo) GetByInternalCode(code string) (*domaindevice.Device, error) {
	return nil, errors.New("not found")
}

type fakeChannelRepo struct {
	mu   sync.Mutex
	rows map[string][]*domainchannel.Channel
}

func newFakeChannelRepo() *fakeChannelRepo {
	return &fakeChannelRepo{rows: map[string][]*domainchannel.Channel{}}
}

func (f *fakeChannelRepo) GetOne(deviceID, gbid string) (*domainchannel.Channel, error) {
	return nil, errors.New("not found")
}
func (f *fakeChannelRepo) GetByID(id int) (*domainchannel.Channel, error) {
	return nil, errors.New("not found")
}
func (f *fakeChannelRepo) GetByInternalCode(code string) (*domainchannel.Channel, error) {
	return nil, errors.New("not found")
}
func (f *fakeChannelRepo) ListByDevice(deviceID string, page, count int, query string, online *bool) ([]*domainchannel.Channel, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rows := f.rows[deviceID]
	return rows, int64(len(rows)), nil
}
func (f *fakeChannelRepo) ResetByDevice(deviceID string, dataDeviceID int, channels []*domainchannel.Channel) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, ch := range channels {
		if ch.ID == 0 {
			ch.ID = i + 1
		}
		ch.DeviceID = deviceID
		ch.DataDeviceID = dataDeviceID
	}
	f.rows[deviceID] = channels
	return nil
}
func (f *fakeChannelRepo) DeleteByDevice(deviceID string) error   { delete(f.rows, deviceID); return nil }
func (f *fakeChannelRepo) ChangeAudio(id int, audio bool) error   { return nil }
func (f *fakeChannelRepo) Update(ch *domainchannel.Channel) error { return nil }
func (f *fakeChannelRepo) UpdateStreamIdentification(id int, deviceID, value string) error {
	return nil
}
func (f *fakeChannelRepo) ListCommon(page, count int, query string, channelType *int, online *bool, hasRecordPlan *bool) ([]*domainchannel.Channel, int64, error) {
	return nil, 0, nil
}
func (f *fakeChannelRepo) ListAll(page, count int, query string) ([]*domainchannel.Channel, int64, error) {
	return nil, 0, nil
}
func (f *fakeChannelRepo) ListByCivilCode(page, count int, query string, channelType *int, online *bool, civilCode string) ([]*domainchannel.Channel, int64, error) {
	return nil, 0, nil
}
func (f *fakeChannelRepo) ListByGroupParent(page, count int, query string, channelType *int, online *bool, groupDeviceID string) ([]*domainchannel.Channel, int64, error) {
	return nil, 0, nil
}
func (f *fakeChannelRepo) ListIDsByDevice(deviceID string) ([]int, error) { return nil, nil }
func (f *fakeChannelRepo) CountCommonByIDs(ids []int) (int64, error)      { return 0, nil }
func (f *fakeChannelRepo) CountCommonByDataDeviceIDs(ids []int) (int64, error) {
	return 0, nil
}
func (f *fakeChannelRepo) SetCivilCode(civilCode string, ids []int) error { return nil }
func (f *fakeChannelRepo) ClearCivilCode(ids []int) error                 { return nil }
func (f *fakeChannelRepo) SetCivilCodeByDataDeviceIDs(civilCode string, ids []int) error {
	return nil
}
func (f *fakeChannelRepo) ClearCivilCodeByDataDeviceIDs(ids []int) error { return nil }
func (f *fakeChannelRepo) SetGroup(parentID, businessGroup string, ids []int) error {
	return nil
}
func (f *fakeChannelRepo) SetGroupByDataDeviceIDs(parentID, businessGroup string, ids []int) error {
	return nil
}
func (f *fakeChannelRepo) ClearGroupParent(ids []int) error          { return nil }
func (f *fakeChannelRepo) ClearGroupByDataDeviceIDs(ids []int) error { return nil }

type fakeDeviceCache struct {
	mu   sync.Mutex
	data map[string]*domaindevice.Device
}

func newFakeDeviceCache() *fakeDeviceCache {
	return &fakeDeviceCache{data: map[string]*domaindevice.Device{}}
}

func (f *fakeDeviceCache) UpdateDevice(_ context.Context, d *domaindevice.Device) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := *d
	f.data[d.DeviceID] = &c
	return nil
}
func (f *fakeDeviceCache) GetDevice(_ context.Context, id string) (*domaindevice.Device, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.data[id]
	if !ok {
		return nil, errors.New("cache miss")
	}
	c := *d
	return &c, nil
}
func (f *fakeDeviceCache) RemoveDevice(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, id)
	return nil
}
func (f *fakeDeviceCache) RemoveDeviceExpiry(context.Context, string, string) error { return nil }
func (f *fakeDeviceCache) SetDeviceExpiry(context.Context, string, string, int64) error {
	return nil
}
func (f *fakeDeviceCache) ListExpiredDevices(context.Context, string) ([]string, error) {
	return nil, nil
}
func (f *fakeDeviceCache) RemoveExpiredDevices(context.Context, string, []string) error { return nil }
func (f *fakeDeviceCache) GetKeepaliveTimeStamps(context.Context, string, int) ([]int64, error) {
	return nil, nil
}
func (f *fakeDeviceCache) GetRegisterTimeStamps(context.Context, string, int) ([]int64, error) {
	return nil, nil
}

func newTestService() (*Service, *fakeDeviceRepo, *fakeChannelRepo, *fakeDeviceCache) {
	devs := newFakeDeviceRepo()
	chans := newFakeChannelRepo()
	cache := newFakeDeviceCache()
	svc := NewService(devs, chans, cache)
	return svc, devs, chans, cache
}

// --- tests ---

func TestSaveRegisterCreatesNewWhenAllowed(t *testing.T) {
	svc, devs, _, _ := newTestService()
	dev := &domaindevice.Device{DeviceID: "34020000001320000001", Name: "cam"}
	if _, err := svc.SaveRegister(dev); err != nil {
		t.Fatalf("SaveRegister: %v", err)
	}
	if len(devs.created) != 1 {
		t.Fatalf("expected 1 create, got %d", len(devs.created))
	}
}

func TestSaveRegisterRejectsWhenPreRegisterRequired(t *testing.T) {
	svc, _, _, _ := newTestService()
	svc.SetRequirePreRegister(true)
	_, err := svc.SaveRegister(&domaindevice.Device{DeviceID: "x"})
	if !errors.Is(err, ErrDeviceNotPreRegistered) {
		t.Fatalf("expected ErrDeviceNotPreRegistered, got %v", err)
	}
}

func TestSaveRegisterExistingWritesOnlySignalingFields(t *testing.T) {
	svc, devs, _, _ := newTestService()
	devs.rows["dev1"] = &domaindevice.Device{ID: 7, DeviceID: "dev1", Name: "old", Manufacturer: "Acme"}

	dev := &domaindevice.Device{DeviceID: "dev1", IP: "10.0.0.9", Port: 5060, Transport: "UDP", Expires: 3600}
	saved, err := svc.SaveRegister(dev)
	if err != nil {
		t.Fatalf("SaveRegister: %v", err)
	}
	if saved.ID != 7 {
		t.Fatalf("expected ID 7, got %d", saved.ID)
	}
	if saved.Manufacturer != "Acme" {
		t.Fatalf("identity should be merged from DB, got %q", saved.Manufacturer)
	}
	fields := devs.fields["dev1"]
	if fields["__full_update__"] == true {
		t.Fatalf("must not full-row Save on REGISTER")
	}
	if fields["ip"] != "10.0.0.9" || fields["transport"] != "UDP" {
		t.Fatalf("unexpected targeted fields: %#v", fields)
	}
}

func TestGetByDeviceIDMergesCacheIdentityFromDB(t *testing.T) {
	svc, devs, _, cache := newTestService()
	devs.rows["d1"] = &domaindevice.Device{ID: 3, DeviceID: "d1", Name: "db-name", Manufacturer: "Hik"}
	_ = cache.UpdateDevice(context.Background(), &domaindevice.Device{DeviceID: "d1", Name: "d1"})

	got, err := svc.GetByDeviceID("d1")
	if err != nil {
		t.Fatalf("GetByDeviceID: %v", err)
	}
	if got.ID != 3 || got.Name != "db-name" || got.Manufacturer != "Hik" {
		t.Fatalf("identity not merged: %#v", got)
	}
}

func TestGetByDeviceIDRemovesCacheWhenDBMissing(t *testing.T) {
	svc, _, _, cache := newTestService()
	_ = cache.UpdateDevice(context.Background(), &domaindevice.Device{DeviceID: "gone"})
	if _, err := svc.GetByDeviceID("gone"); err == nil {
		t.Fatal("expected error for missing device")
	}
	if _, err := cache.GetDevice(context.Background(), "gone"); err == nil {
		t.Fatal("expected cache entry removed")
	}
}

func TestHandleCatalogCreatesChannelsAndBackfills(t *testing.T) {
	svc, devs, chans, _ := newTestService()
	devs.rows["dev"] = &domaindevice.Device{ID: 1, DeviceID: "dev", Name: "cam"}

	items := []gbsip.CatalogItem{
		{DeviceID: "ch1", Manufacturer: "Acme", Model: "M1", Status: "ON"},
		{DeviceID: "ch2", Status: "OFF"},
	}
	if err := svc.HandleCatalog("dev", items); err != nil {
		t.Fatalf("HandleCatalog: %v", err)
	}
	rows := chans.rows["dev"]
	if len(rows) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(rows))
	}
	if rows[0].Name != "cam_channel_1" {
		t.Fatalf("unexpected channel name %q", rows[0].Name)
	}
	if devs.rows["dev"].Manufacturer != "Acme" || devs.rows["dev"].Model != "M1" {
		t.Fatalf("device identity not backfilled: %#v", devs.rows["dev"])
	}
}

func TestHandleCatalogEmptyIPCDeviceCreatesDefaultChannel(t *testing.T) {
	svc, devs, chans, _ := newTestService()
	ipc := "34020000001320000001"
	devs.rows[ipc] = &domaindevice.Device{ID: 1, DeviceID: ipc, Name: "cam"}

	if err := svc.HandleCatalog(ipc, nil); err != nil {
		t.Fatalf("HandleCatalog: %v", err)
	}
	rows := chans.rows[ipc]
	if len(rows) != 1 {
		t.Fatalf("expected default channel, got %d", len(rows))
	}
	if rows[0].GBDeviceID != defaultIPCChannelGBID(ipc) {
		t.Fatalf("unexpected default channel id %q", rows[0].GBDeviceID)
	}
}

func TestOfflineUsesTargetedOnlineUpdate(t *testing.T) {
	svc, devs, _, _ := newTestService()
	devs.rows["d"] = &domaindevice.Device{ID: 1, DeviceID: "d", OnLine: true}
	if err := svc.Offline(&domaindevice.Device{DeviceID: "d", OnLine: true}); err != nil {
		t.Fatalf("Offline: %v", err)
	}
	if devs.online["d"] != false {
		t.Fatalf("expected targeted offline update")
	}
}
