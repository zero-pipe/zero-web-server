package sipinfra

import (
	"errors"
	"sync"
	"time"

	domainptz "zero-web-kit/internal/domain/ptz"
)

var ErrPresetTimeout = errors.New("预置位查询超时，设备未返回")

type PresetManager struct {
	pending sync.Map // sn -> *presetWaiter
}

type presetWaiter struct {
	ch     chan []domainptz.Preset
	items  []domainptz.Preset
	sumNum int
	sn     string
}

func NewPresetManager() *PresetManager {
	return &PresetManager{}
}

func (m *PresetManager) Register(sn string) <-chan []domainptz.Preset {
	w := &presetWaiter{ch: make(chan []domainptz.Preset, 1), sn: sn}
	m.pending.Store(sn, w)
	return w.ch
}

func (m *PresetManager) Cancel(sn string) {
	if v, ok := m.pending.LoadAndDelete(sn); ok {
		close(v.(*presetWaiter).ch)
	}
}

func (m *PresetManager) HandlePresetQuery(sn string, sumNum int, items []domainptz.Preset) {
	v, ok := m.pending.Load(sn)
	if !ok {
		return
	}
	w := v.(*presetWaiter)
	if sumNum > 0 {
		w.sumNum = sumNum
	}
	w.items = append(w.items, items...)
	target := w.sumNum
	if target == 0 && len(items) == 0 {
		// Device reported no presets.
		select {
		case w.ch <- []domainptz.Preset{}:
		default:
		}
		m.pending.Delete(sn)
		return
	}
	if target == 0 || len(w.items) >= target {
		out := append([]domainptz.Preset(nil), w.items...)
		select {
		case w.ch <- out:
		default:
		}
		m.pending.Delete(sn)
	}
}

func (m *PresetManager) Wait(sn string, timeout time.Duration) ([]domainptz.Preset, error) {
	v, ok := m.pending.Load(sn)
	if !ok {
		return nil, ErrPresetTimeout
	}
	w := v.(*presetWaiter)
	select {
	case info := <-w.ch:
		m.pending.Delete(sn)
		if info == nil {
			return []domainptz.Preset{}, nil
		}
		return info, nil
	case <-time.After(timeout):
		m.Cancel(sn)
		return nil, ErrPresetTimeout
	}
}
