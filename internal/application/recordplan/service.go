package recordplan

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	onvifapp "zero-web-server/internal/application/onvif"
	playapp "zero-web-server/internal/application/play"
	publishauth "zero-web-server/internal/application/publishauth"
	storagenodeapp "zero-web-server/internal/application/storagenode"
	domainrecordplan "zero-web-server/internal/domain/recordplan"
	"zero-web-server/internal/domain/shared"
	"zero-web-server/internal/mediaurl"
	"zero-web-server/internal/storagekit"
	applog "zero-web-server/pkg/log"
)

type PlanView struct {
	domainrecordplan.Plan
	PlanItemList []domainrecordplan.Item `json:"planItemList"`
	ChannelCount int64                   `json:"channelCount"`
}

type LinkParam struct {
	PlanID      int   `json:"planId"`
	ChannelIds  []int `json:"channelIds"`
	DeviceDbIds []int `json:"deviceDbIds"`
	AllLink     *bool `json:"allLink"`
}

type Service struct {
	repo        domainrecordplan.Repository
	play        *playapp.Service
	publish     *publishauth.PublishRegistry
	storageNode *storagenodeapp.Service
	onvif       *onvifapp.Service
	serverID    string
	active      sync.Map // channelID -> streamKey "app/stream"
	meta        sync.Map // channelID -> activeMeta
	stopCh      chan struct{}
}

type activeMeta struct {
	DeviceID   string
	ChannelID  string
	App        string
	Stream     string
	Sidecar    bool   // 旁路录制（zstorage）
	ChannelKey string // 旁路：zstorage 的 channel_key（内码_内码）
	StorageID  string // 旁路：存储节点 node_id
	StartedZMS bool   // 旁路：是否已让 ZMS 出流（GB 才需要，停时要 StopPlay）
}

func NewService(repo domainrecordplan.Repository, play *playapp.Service, publish *publishauth.PublishRegistry, storageNode *storagenodeapp.Service, onvif *onvifapp.Service, serverID string) *Service {
	return &Service{
		repo: repo, play: play, publish: publish, storageNode: storageNode, onvif: onvif, serverID: serverID,
		stopCh: make(chan struct{}),
	}
}

func (s *Service) Start() {
	go s.loop()
}

func (s *Service) Stop() { close(s.stopCh) }

func (s *Service) loop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	s.tick()
	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Service) tick() {
	now := time.Now()
	weekDay := int(now.Weekday())
	if weekDay == 0 {
		weekDay = 7
	}
	index := now.Hour()*60 + now.Minute()
	channels, err := s.repo.QueryRecordingChannels(weekDay, index)
	if err != nil {
		applog.Warnf("[record-plan] query recording channels: %v", err)
		return
	}
	need := make(map[int]domainrecordplan.ChannelRecord, len(channels))
	for _, ch := range channels {
		need[ch.ID] = ch
	}
	// 窗口外：主动停流
	s.active.Range(func(k, v any) bool {
		chID := k.(int)
		if _, ok := need[chID]; ok {
			return true
		}
		s.stopRecord(chID)
		return true
	})
	for _, ch := range need {
		if _, ok := s.active.Load(ch.ID); ok {
			continue
		}
		if !s.canRecord(ch) {
			continue
		}
		go s.startRecord(ch)
	}
}

// canRecord 判定该通道在当前计划下是否应起录。
// 旁路(有存储节点)：GB28181 需设备/GB 编号齐全；ONVIF 只需统一通道 ID（直连相机 RTSP）。
// 本地 MP4：沿用原 GB28181 条件，不改变既有行为。
func (s *Service) canRecord(ch domainrecordplan.ChannelRecord) bool {
	if ch.StorageNodeID != "" && ch.Protocol == "onvif" {
		return ch.ID > 0
	}
	return ch.DataType == shared.ChannelDataTypeGB28181 && ch.DeviceID != "" && ch.GBDeviceID != ""
}

func (s *Service) startRecord(ch domainrecordplan.ChannelRecord) {
	app := mediaurl.LiveApp
	stream := mediaurl.LiveStream(ch.DeviceInternalCode, ch.InternalCode)
	if stream == "" {
		applog.Warnf("[record-plan] skip channel=%d: device/channel internal code missing", ch.ID)
		return
	}
	// 旁路录制（zstorage）：通道绑定 storage_node_id 时走 PS 槽位录制，不再启用 MP4。
	if ch.StorageNodeID != "" && s.storageNode != nil {
		s.startSidecarRecording(ch, app, stream)
		return
	}
	s.publish.EnableMP4(app, stream)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	content, err := s.play.StartPlay(ctx, ch.DeviceID, ch.GBDeviceID)
	if err != nil {
		s.publish.DisableMP4(app, stream)
		applog.Warnf("[record-plan] start play failed channel=%d %s/%s: %v", ch.ID, ch.DeviceID, ch.GBDeviceID, err)
		return
	}
	keyApp, keyStream := app, stream
	if content != nil && content.App != "" && content.Stream != "" {
		keyApp, keyStream = content.App, content.Stream
	}
	s.active.Store(ch.ID, keyApp+"/"+keyStream)
	s.meta.Store(ch.ID, activeMeta{
		DeviceID: ch.DeviceID, ChannelID: ch.GBDeviceID,
		App: keyApp, Stream: keyStream,
	})
	applog.Infof("[record-plan] recording start channel=%d stream=%s/%s", ch.ID, keyApp, keyStream)
}

// startSidecarRecording 在存储节点上按 channel_key 拉流旁路录 PS。
// GB28181：先 StartPlay 让 ZMS 出流（StartedZMS=true），拉 ZMS 的 RTSP；
// ONVIF：直接取相机原生 RTSP 直连录制，不经 ZMS。
func (s *Service) startSidecarRecording(ch domainrecordplan.ChannelRecord, app, stream string) {
	node, err := s.storageNode.Resolve(ch.StorageNodeID)
	if err != nil {
		applog.Warnf("[record-plan] sidecar channel=%d resolve storage node %q: %v", ch.ID, ch.StorageNodeID, err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pullURL string
	startedZMS := false
	if ch.Protocol == "onvif" {
		if s.onvif == nil {
			applog.Warnf("[record-plan] sidecar channel=%d: onvif 服务未注入，跳过", ch.ID)
			return
		}
		url, perr := s.onvif.PullURL(ctx, int64(ch.ID))
		if perr != nil || url == "" {
			applog.Warnf("[record-plan] sidecar channel=%d onvif 取流地址失败: %v", ch.ID, perr)
			return
		}
		pullURL = url
	} else {
		content, perr := s.play.StartPlay(ctx, ch.DeviceID, ch.GBDeviceID)
		if perr != nil {
			applog.Warnf("[record-plan] sidecar channel=%d start play %s/%s: %v", ch.ID, ch.DeviceID, ch.GBDeviceID, perr)
			return
		}
		startedZMS = true
		if content != nil {
			pullURL = content.Rtsp
		}
		if pullURL == "" {
			_ = s.play.StopPlay(ch.DeviceID, ch.GBDeviceID)
			applog.Warnf("[record-plan] sidecar channel=%d: ZMS 未返回 RTSP 拉流地址", ch.ID)
			return
		}
	}

	cli := storagekit.NewClient(node.IP, node.HTTPPort, node.Secret)
	resp, rerr := cli.RecordStart(ctx, stream, pullURL)
	if rerr != nil || resp.Code != 0 {
		if startedZMS {
			_ = s.play.StopPlay(ch.DeviceID, ch.GBDeviceID)
		}
		msg := ""
		if resp != nil {
			msg = resp.Msg
		}
		applog.Warnf("[record-plan] sidecar channel=%d record/start node=%s err=%v msg=%s", ch.ID, node.NodeID, rerr, msg)
		return
	}
	s.active.Store(ch.ID, app+"/"+stream)
	s.meta.Store(ch.ID, activeMeta{
		DeviceID: ch.DeviceID, ChannelID: ch.GBDeviceID,
		App: app, Stream: stream,
		Sidecar: true, ChannelKey: stream, StorageID: node.NodeID, StartedZMS: startedZMS,
	})
	applog.Infof("[record-plan] sidecar recording start channel=%d proto=%s node=%s channel_key=%s", ch.ID, ch.Protocol, node.NodeID, stream)
}

func (s *Service) stopRecord(channelID int) {
	v, ok := s.meta.Load(channelID)
	s.active.Delete(channelID)
	s.meta.Delete(channelID)
	if !ok {
		return
	}
	m := v.(activeMeta)
	if m.Sidecar {
		if s.storageNode != nil {
			if node, err := s.storageNode.Resolve(m.StorageID); err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_, _ = storagekit.NewClient(node.IP, node.HTTPPort, node.Secret).RecordStop(ctx, m.ChannelKey)
				cancel()
			}
		}
		if m.StartedZMS {
			_ = s.play.StopPlay(m.DeviceID, m.ChannelID)
		}
		applog.Infof("[record-plan] sidecar recording stop channel=%d channel_key=%s", channelID, m.ChannelKey)
		return
	}
	s.publish.DisableMP4(m.App, m.Stream)
	if err := s.play.StopPlay(m.DeviceID, m.ChannelID); err != nil {
		applog.Warnf("[record-plan] stop play failed channel=%d %s/%s: %v", channelID, m.DeviceID, m.ChannelID, err)
	} else {
		applog.Infof("[record-plan] recording stop channel=%d stream=%s/%s", channelID, m.App, m.Stream)
	}
}

func (s *Service) Add(name, storageNodeID string, items []domainrecordplan.Item) error {
	if len(items) == 0 {
		return fmt.Errorf("录制计划不可为空")
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	plan := &domainrecordplan.Plan{Name: name, StorageNodeID: strings.TrimSpace(storageNodeID), CreateTime: now, UpdateTime: now}
	return s.repo.Create(plan, items)
}

func (s *Service) Update(id int, name, storageNodeID string, items []domainrecordplan.Item) error {
	plan, _, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	plan.Name = name
	plan.StorageNodeID = strings.TrimSpace(storageNodeID)
	plan.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	return s.repo.Update(plan, items)
}

func (s *Service) Delete(id int) error {
	return s.repo.Delete(id)
}

func (s *Service) Get(id int) (*PlanView, error) {
	plan, items, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return &PlanView{
		Plan:         *plan,
		PlanItemList: items,
		ChannelCount: s.repo.CountChannels(id),
	}, nil
}

func (s *Service) Query(page, count int, query string) ([]PlanView, int64, error) {
	rows, total, err := s.repo.List(page, count, query)
	if err != nil {
		return nil, 0, err
	}
	out := make([]PlanView, len(rows))
	for i, p := range rows {
		_, items, _ := s.repo.GetByID(p.ID)
		out[i] = PlanView{
			Plan:         p,
			PlanItemList: items,
			ChannelCount: s.repo.CountChannels(p.ID),
		}
	}
	return out, total, nil
}

func (s *Service) Link(param LinkParam) error {
	var err error
	if param.AllLink != nil {
		if *param.AllLink {
			err = s.repo.LinkAll(param.PlanID)
		} else {
			err = s.repo.UnlinkAll(param.PlanID)
		}
	} else if len(param.DeviceDbIds) > 0 {
		if param.PlanID == 0 {
			err = s.repo.UnlinkChannelsByDevice(param.DeviceDbIds)
		} else {
			err = s.repo.LinkChannelsByDevice(param.PlanID, param.DeviceDbIds)
		}
	} else {
		ids := param.ChannelIds
		if len(ids) == 0 {
			return fmt.Errorf("通道编号必须存在")
		}
		if param.PlanID == 0 {
			err = s.repo.UnlinkChannels(ids)
		} else {
			err = s.repo.LinkChannels(param.PlanID, ids)
		}
	}
	if err != nil {
		return err
	}
	go s.tick()
	return nil
}

func (s *Service) ChannelList(page, count, planID int, query string, hasLink *bool, online, channelType string) ([]domainrecordplan.ChannelRecord, int64, error) {
	return s.repo.ChannelList(page, count, planID, query, hasLink, online, channelType)
}

func (s *Service) Recording(app, stream string) bool {
	key := app + "/" + stream
	found := false
	s.active.Range(func(_, v any) bool {
		if v.(string) == key {
			found = true
			return false
		}
		return true
	})
	return found
}

func (s *Service) OnStreamDeparture(app, stream string) {
	key := app + "/" + stream
	var channelID int
	s.active.Range(func(k, v any) bool {
		if v.(string) == key {
			channelID = k.(int)
			return false
		}
		return true
	})
	if channelID == 0 {
		return
	}
	now := time.Now()
	weekDay := int(now.Weekday())
	if weekDay == 0 {
		weekDay = 7
	}
	index := now.Hour()*60 + now.Minute()
	channels, err := s.repo.QueryRecordingChannels(weekDay, index)
	if err != nil {
		s.stopRecord(channelID)
		return
	}
	var ch *domainrecordplan.ChannelRecord
	for i := range channels {
		if channels[i].ID == channelID {
			ch = &channels[i]
			break
		}
	}
	s.active.Delete(channelID)
	s.meta.Delete(channelID)
	s.publish.DisableMP4(app, stream)
	if ch == nil {
		return
	}
	go s.startRecord(*ch)
}
