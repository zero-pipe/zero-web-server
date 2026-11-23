package channeldrv

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domainasset "zero-web-server/internal/domain/asset"
	"zero-web-server/internal/dto"
)

func TestRegistry(t *testing.T) {
	r := New(Deps{})
	for _, proto := range []string{"gb28181", "onvif", "plugin"} {
		d, ok := r.Get(proto)
		if !ok || d.Protocol() != proto {
			t.Fatalf("driver %s missing or mismatched", proto)
		}
	}
	if _, ok := r.Get("unknown"); ok {
		t.Fatal("unexpected driver for unknown protocol")
	}
}

func TestClamp01(t *testing.T) {
	cases := map[float64]float64{-1: 0, 0: 0, 0.5: 0.5, 1: 1, 2: 1}
	for in, want := range cases {
		if got := clamp01(in); got != want {
			t.Errorf("clamp01(%v)=%v want %v", in, got, want)
		}
	}
}

type fakeDriver struct{ delay time.Duration }

func (f *fakeDriver) Protocol() string { return "fake" }
func (f *fakeDriver) StartLive(ctx context.Context, _ *domainasset.Resource, _ *domainasset.AccessBinding,
	_ *domainasset.StreamProfile) (*dto.StreamContent, *Session, error) {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}
	return &dto.StreamContent{}, nil, nil
}
func (f *fakeDriver) StopLive(context.Context, *domainasset.Resource, *domainasset.AccessBinding, *Session) error {
	return nil
}
func (f *fakeDriver) Sync(context.Context, *domainasset.AccessBinding) error { return nil }
func (f *fakeDriver) PTZ(context.Context, *domainasset.Resource, *domainasset.AccessBinding, PTZCommand) error {
	return nil
}

func (f *fakeDriver) Command(context.Context, *domainasset.Resource, *domainasset.AccessBinding, string, json.RawMessage) (json.RawMessage, error) {
	return nil, nil
}

func TestLimitedTimeout(t *testing.T) {
	d := newLimited(&fakeDriver{delay: time.Second})
	d.timeout = 20 * time.Millisecond
	if _, _, err := d.StartLive(context.Background(), nil, nil, nil); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestLimitedConcurrency(t *testing.T) {
	d := newLimited(&fakeDriver{delay: 50 * time.Millisecond})
	d.sem = make(chan struct{}, 1)
	if _, _, err := d.begin(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, _, err := d.begin(ctx); err == nil {
		t.Fatal("expected busy error when semaphore is full")
	}
	<-d.sem // release slot acquired above
}
