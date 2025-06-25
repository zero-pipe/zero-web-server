package sipinfra

import (
	"errors"
	"sync"
	"time"

	domainrecord "zero-web-kit/internal/domain/record"
)

type RecordManager struct {
	pending sync.Map // sn -> *recordWaiter
}

type recordWaiter struct {
	ch     chan *domainrecord.RecordInfo
	items  []RecordItem
	sumNum int
	sn     string
}

func NewRecordManager() *RecordManager {
	return &RecordManager{}
}

func (m *RecordManager) Register(sn string) <-chan *domainrecord.RecordInfo {
	w := &recordWaiter{ch: make(chan *domainrecord.RecordInfo, 1), sn: sn}
	m.pending.Store(sn, w)
	return w.ch
}

func (m *RecordManager) Cancel(sn string) {
	if v, ok := m.pending.LoadAndDelete(sn); ok {
		close(v.(*recordWaiter).ch)
	}
}

func (m *RecordManager) Wait(sn string, timeout time.Duration) (*domainrecord.RecordInfo, error) {
	v, ok := m.pending.Load(sn)
	if !ok {
		ch := m.Register(sn)
		select {
		case info := <-ch:
			m.pending.Delete(sn)
			return info, nil
		case <-time.After(timeout):
			m.Cancel(sn)
			return nil, ErrRecordTimeout
		}
	}
	w := v.(*recordWaiter)
	select {
	case info := <-w.ch:
		m.pending.Delete(sn)
		return info, nil
	case <-time.After(timeout):
		m.Cancel(sn)
		return nil, ErrRecordTimeout
	}
}

func (m *RecordManager) HandleRecordInfo(deviceID, channelID, sn string, sumNum int, items []RecordItem) {
	v, ok := m.pending.Load(sn)
	if !ok {
		return
	}
	w := v.(*recordWaiter)
	w.sumNum = sumNum
	w.items = append(w.items, items...)
	if sumNum == 0 || len(w.items) >= sumNum {
		info := &domainrecord.RecordInfo{
			DeviceID: deviceID, ChannelID: channelID, SN: sn,
			SumNum: sumNum, Count: len(w.items),
			RecordList: toDomainRecordItems(w.items),
		}
		select {
		case w.ch <- info:
		default:
		}
		m.pending.Delete(sn)
	}
}

func toDomainRecordItems(items []RecordItem) []domainrecord.RecordItem {
	out := make([]domainrecord.RecordItem, 0, len(items))
	for _, it := range items {
		out = append(out, domainrecord.RecordItem{
			DeviceID: it.DeviceID, Name: it.Name, FilePath: it.FilePath,
			FileSize: it.FileSize, StartTime: it.StartTime, EndTime: it.EndTime,
			Secrecy: it.Secrecy, Type: it.Type, RecorderID: it.RecorderID,
		})
	}
	return out
}

var ErrRecordTimeout = errors.New("record query timeout")
