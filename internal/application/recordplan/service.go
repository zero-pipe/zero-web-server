package recordplan

import (
	"context"
	"fmt"
	"sync"
	"time"

	mediaapp "zero-web-kit/internal/application/media"
	playapp "zero-web-kit/internal/application/play"
	"zero-web-kit/internal/domain/shared"
	"zero-web-kit/internal/infrastructure/persistence"
	"zero-web-kit/internal/infrastructure/persistence/model"
)

type PlanView struct {
	model.RecordPlan
	PlanItemList []model.RecordPlanItem `json:"planItemList"`
	ChannelCount int64                  `json:"channelCount"`
}

type LinkParam struct {
	PlanID      int     `json:"planId"`
	ChannelIds  []int   `json:"channelIds"`
	DeviceDbIds []int   `json:"deviceDbIds"`
	AllLink     *bool   `json:"allLink"`
}

type Service struct {
	repo     *persistence.RecordPlanRepository
	play     *playapp.Service
	publish  *mediaapp.PublishRegistry
	serverID string
	active   sync.Map // channelID -> streamKey
	stopCh   chan struct{}
}

func NewService(repo *persistence.RecordPlanRepository, play *playapp.Service, publish *mediaapp.PublishRegistry, serverID string) *Service {
	return &Service{
		repo: repo, play: play, publish: publish, serverID: serverID,
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
		return
	}
	need := make(map[int]model.GBDeviceChannel, len(channels))
	for _, ch := range channels {
		need[ch.ID] = ch
	}
	s.active.Range(func(k, v any) bool {
		chID := k.(int)
		if _, ok := need[chID]; !ok {
			s.active.Delete(chID)
		}
		return true
	})
	for _, ch := range need {
		if _, ok := s.active.Load(ch.ID); ok {
			continue
		}
		if ch.DataType != shared.ChannelDataTypeGB28181 || ch.DeviceID == "" {
			continue
		}
		go s.startRecord(ch)
	}
}

func (s *Service) startRecord(ch model.GBDeviceChannel) {
	app := "rtp"
	stream := fmt.Sprintf("%s_%s", ch.DeviceID, ch.GBDeviceID)
	s.publish.EnableMP4(app, stream)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	content, err := s.play.StartPlay(ctx, ch.DeviceID, ch.GBDeviceID)
	if err != nil {
		s.publish.DisableMP4(app, stream)
		return
	}
	s.active.Store(ch.ID, content.App+"/"+content.Stream)
}

func (s *Service) Add(name string, items []model.RecordPlanItem) error {
	if len(items) == 0 {
		return fmt.Errorf("录制计划不可为空")
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	plan := &model.RecordPlan{Name: name, CreateTime: now, UpdateTime: now}
	return s.repo.Create(plan, items)
}

func (s *Service) Update(id int, name string, items []model.RecordPlanItem) error {
	plan, _, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	plan.Name = name
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
		RecordPlan:   *plan,
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
			RecordPlan:   p,
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

func (s *Service) ChannelList(page, count, planID int, query string, hasLink *bool) ([]model.GBDeviceChannel, int64, error) {
	return s.repo.ChannelList(page, count, planID, query, hasLink)
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
		s.active.Delete(channelID)
		return
	}
	var ch *model.GBDeviceChannel
	for i := range channels {
		if channels[i].ID == channelID {
			ch = &channels[i]
			break
		}
	}
	s.active.Delete(channelID)
	if ch == nil {
		s.publish.DisableMP4(app, stream)
		return
	}
	go s.startRecord(*ch)
}
