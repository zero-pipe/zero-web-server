package sipinfra

import (
	"time"

	domainptz "zero-web-server/internal/domain/ptz"
	domainrecord "zero-web-server/internal/domain/record"

	"github.com/zero-pipe/gb28181-go/manscdp"
	"github.com/zero-pipe/gb28181-go/session"
)

var (
	ErrRecordTimeout   = session.ErrRecordTimeout
	ErrPresetTimeout   = session.ErrPresetTimeout
	ErrSessionNotFound = session.ErrSessionNotFound
)

type SessionType = session.SessionType

const (
	SessionPlay     = session.SessionPlay
	SessionPlayback = session.SessionPlayback
	SessionDownload = session.SessionDownload
)

type RecordManager struct {
	inner *session.RecordWaiter
}

func (m *RecordManager) Wait(sn string, timeout time.Duration) (*domainrecord.RecordInfo, error) {
	info, err := m.inner.Wait(sn, timeout)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return &domainrecord.RecordInfo{}, nil
	}
	return toDomainRecordInfo(info), nil
}

func (m *RecordManager) Cancel(sn string) { m.inner.Cancel(sn) }

type PresetManager struct {
	inner *session.PresetWaiter
}

func (m *PresetManager) Wait(sn string, timeout time.Duration) ([]domainptz.Preset, error) {
	items, err := m.inner.Wait(sn, timeout)
	if err != nil {
		return nil, err
	}
	return toDomainPresets(items), nil
}

func (m *PresetManager) Cancel(sn string) { m.inner.Cancel(sn) }

type InviteManager struct {
	inner *session.InviteManager
}

type InviteSession struct {
	inner         *session.InviteSession
	DownloadSpeed int
}

func (s *InviteSession) Progress() float64 {
	if s == nil || s.inner == nil {
		return 0
	}
	return s.inner.Progress()
}

func (m *InviteManager) Get(stream string) (*InviteSession, bool) {
	sess, ok := m.inner.Get(stream)
	if !ok {
		return nil, false
	}
	return &InviteSession{inner: sess, DownloadSpeed: sess.DownloadSpeed}, true
}

func (m *InviteManager) Remove(stream string) { m.inner.Remove(stream) }

// Keep Handle methods available for tests that inject responses directly.
func (m *RecordManager) HandleRecordInfo(deviceID, channelID, sn string, sumNum int, items []manscdp.RecordItem) {
	m.inner.Handle(deviceID, channelID, sn, sumNum, items)
}

func (m *PresetManager) HandlePresetQuery(sn string, sumNum int, items []domainptz.Preset) {
	libItems := make([]manscdp.Preset, len(items))
	for i, p := range items {
		libItems[i] = manscdp.Preset{PresetID: p.PresetID, PresetName: p.PresetName}
	}
	m.inner.Handle(sn, sumNum, libItems)
}
