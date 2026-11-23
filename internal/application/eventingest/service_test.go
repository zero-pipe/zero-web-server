package eventingest

import (
	"errors"
	"testing"

	domainasset "zero-web-server/internal/domain/asset"
	domaincustomevent "zero-web-server/internal/domain/customevent"
	domainevent "zero-web-server/internal/domain/event"
)

type fakeRepo struct {
	events    map[string]*domainevent.Event
	createErr error
}

func (f *fakeRepo) Create(e *domainevent.Event) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.events == nil {
		f.events = map[string]*domainevent.Event{}
	}
	f.events[e.EventID] = e
	return nil
}
func (f *fakeRepo) Exists(eventID string) (bool, error) {
	_, ok := f.events[eventID]
	return ok, nil
}
func (f *fakeRepo) Get(id string) (*domainevent.Event, error) { return nil, errors.New("not found") }
func (f *fakeRepo) List(page, count int, fl domainevent.ListFilter) ([]*domainevent.Event, int64, error) {
	return nil, 0, nil
}
func (f *fakeRepo) DeleteBefore(before int64) (int64, error)   { return 0, nil }
func (f *fakeRepo) DeleteRange(begin, end int64, all bool) (int64, error) { return 0, nil }
func (f *fakeRepo) UpdateAlarmID(id string, alarmID int) error { return nil }

func TestHandleMessageStats(t *testing.T) {
	repo := &fakeRepo{}
	s := NewService(repo, nil, nil, nil, nil)

	if err := s.HandleMessage("t", []byte("not-json")); err == nil {
		t.Fatal("expected invalid payload error")
	}
	ok := []byte(`{"schema":"zws.event.v1","eventId":"e1","deviceInternalCode":"D1","eventType":"motion","occurredAt":1}`)
	if err := s.HandleMessage("t", ok); err != nil {
		t.Fatal(err)
	}
	if err := s.HandleMessage("t", ok); err != nil {
		t.Fatal(err)
	}
	st := s.Stats()
	if st["received"] != 3 || st["stored"] != 1 || st["duplicate"] != 1 || st["invalid"] != 1 {
		t.Fatalf("unexpected stats: %#v", st)
	}
	if st["unmapped"] != 1 {
		t.Fatalf("expected unmapped=1: %#v", st)
	}
}

func TestMatchResourceChannelForms(t *testing.T) {
	resources := []*domainasset.Resource{{ID: "r101", ExternalKey: "101"}, {ID: "r1", ExternalKey: "1"}}
	if got := findResource(resources, "1"); got == nil || got.ID != "r1" {
		t.Fatalf("got %#v", got)
	}
	if got := findResource(resources, "101"); got == nil || got.ID != "r101" {
		t.Fatalf("got %#v", got)
	}
}

type fakeCustomRepo struct {
	events map[string]*domaincustomevent.CustomEvent
}

func (f *fakeCustomRepo) Create(e *domaincustomevent.CustomEvent) error {
	if f.events == nil {
		f.events = map[string]*domaincustomevent.CustomEvent{}
	}
	f.events[e.EventID] = e
	return nil
}
func (f *fakeCustomRepo) Exists(id string) (bool, error) { _, ok := f.events[id]; return ok, nil }
func (f *fakeCustomRepo) Get(id string) (*domaincustomevent.CustomEvent, error) {
	return nil, errors.New("not found")
}
func (f *fakeCustomRepo) List(page, count int, fl domaincustomevent.ListFilter) ([]*domaincustomevent.CustomEvent, int64, error) {
	return nil, 0, nil
}
func (f *fakeCustomRepo) DistinctTypes(deviceInternalCode string) ([]domaincustomevent.TypeCount, error) {
	return nil, nil
}

func (f *fakeCustomRepo) DeleteRange(begin, end int64, all bool) (int64, error) { return 0, nil }

func TestCustomEventRouting(t *testing.T) {
	er := &fakeRepo{}
	cr := &fakeCustomRepo{}
	s := NewService(er, nil, nil, nil, cr)

	body := []byte(`{"eventId":"c1","deviceInternalCode":"D1","eventType":"access.card.open","majorType":"access","minorType":"card.open","occurredAt":1000,"raw":{"cardNo":"A1"}}`)
	if err := s.HandleMessage("t", body); err != nil {
		t.Fatal(err)
	}
	if len(er.events) != 0 {
		t.Fatalf("未知类型不应进 zws_event，实得 %d 条", len(er.events))
	}
	e := cr.events["c1"]
	if e == nil {
		t.Fatal("未知类型应进自定义事件仓库")
	}
	if e.MajorType != "access" || e.MinorType != "card.open" {
		t.Fatalf("大类/小类解析错误: %#v", e)
	}
	if e.DeviceInternalCode != "D1" || e.RawJSON == "" {
		t.Fatalf("基础字段缺失: %#v", e)
	}
}
