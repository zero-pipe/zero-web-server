package eventingest

import (
	"errors"
	"testing"

	domainasset "zero-web-server/internal/domain/asset"
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
func (f *fakeRepo) UpdateAlarmID(id string, alarmID int) error { return nil }

func TestHandleMessageStats(t *testing.T) {
	repo := &fakeRepo{}
	s := NewService(repo, nil, nil, nil)

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
