package commonchannel

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainchannel "zero-web-kit/internal/domain/channel"
	playapp "zero-web-kit/internal/application/play"
	playbackapp "zero-web-kit/internal/application/playback"
	ptzapp "zero-web-kit/internal/application/ptz"
	"zero-web-kit/internal/infrastructure/persistence"
	"zero-web-kit/internal/infrastructure/persistence/model"
	"zero-web-kit/internal/interfaces/http/dto"
)

type View struct {
	GbID         int     `json:"gbId"`
	GbDeviceId   string  `json:"gbDeviceId"`
	GbName       string  `json:"gbName"`
	DeviceId     string  `json:"deviceId"`
	DataType     int     `json:"dataType"`
	DataDeviceId int     `json:"dataDeviceId"`
	Name         string  `json:"name"`
	Status       string  `json:"gbStatus"`
	PTZType      int     `json:"ptzType"`
	Longitude    float64 `json:"longitude"`
	GbLongitude  float64 `json:"gbLongitude"`
	Latitude     float64 `json:"gbLatitude"`
	HasAudio     bool    `json:"hasAudio"`
	RecordPlanId int     `json:"recordPlanId"`
}

type Service struct {
	channels *persistence.ChannelRepository
	groups   *persistence.GroupRegionRepository
	play     *playapp.Service
	playback *playbackapp.Service
	ptz      *ptzapp.Service
}

func NewService(ch *persistence.ChannelRepository, groups *persistence.GroupRegionRepository, play *playapp.Service, playback *playbackapp.Service, ptz *ptzapp.Service) *Service {
	return &Service{channels: ch, groups: groups, play: play, playback: playback, ptz: ptz}
}

func toView(ch *domainchannel.Channel) View {
	return View{
		GbID: ch.ID, GbDeviceId: ch.GBDeviceID, GbName: ch.Name, DeviceId: ch.DeviceID,
		DataType: ch.DataType, DataDeviceId: ch.DataDeviceID, Name: ch.Name,
		Status: ch.Status, PTZType: ch.PTZType,
		Longitude: ch.Longitude, GbLongitude: ch.Longitude,
		Latitude: ch.Latitude, HasAudio: ch.HasAudio,
	}
}

func (s *Service) GetOne(id int) (*View, error) {
	ch, err := s.channels.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("通道不存在")
	}
	v := toView(ch)
	return &v, nil
}

func (s *Service) List(page, count int, query string, channelType *int, online, hasRecordPlan *bool) ([]View, int64, error) {
	return s.ListFiltered(page, count, query, channelType, online, hasRecordPlan, "", "")
}

func (s *Service) ListFiltered(page, count int, query string, channelType *int, online, hasRecordPlan *bool, civilCode, parentDeviceID string) ([]View, int64, error) {
	var rows []*domainchannel.Channel
	var total int64
	var err error
	switch {
	case civilCode != "":
		rows, total, err = s.channels.ListByCivilCode(page, count, query, channelType, online, civilCode)
	case parentDeviceID != "":
		rows, total, err = s.channels.ListByGroupParent(page, count, query, channelType, online, parentDeviceID)
	default:
		rows, total, err = s.channels.ListCommon(page, count, query, channelType, online, hasRecordPlan)
	}
	if err != nil {
		return nil, 0, err
	}
	out := make([]View, len(rows))
	for i, ch := range rows {
		out[i] = toView(ch)
	}
	return out, total, nil
}

func (s *Service) Update(v *View) error {
	ch, err := s.channels.GetByID(v.GbID)
	if err != nil {
		return err
	}
	if v.GbName != "" {
		ch.Name = v.GbName
	}
	if v.GbDeviceId != "" {
		ch.GBDeviceID = v.GbDeviceId
	}
	ch.Longitude = v.GbLongitude
	ch.Latitude = v.Latitude
	ch.PTZType = v.PTZType
	return s.channels.Update(&model.GBDeviceChannel{
		ID: ch.ID, DeviceID: ch.DeviceID, GBDeviceID: ch.GBDeviceID, Name: ch.Name,
		Longitude: ch.Longitude, Latitude: ch.Latitude, PTZType: ch.PTZType,
		DataType: ch.DataType, DataDeviceID: ch.DataDeviceID, Status: ch.Status,
		CreateTime: ch.CreateTime,
	})
}

func (s *Service) Play(channelID int) (*dto.StreamContent, error) {
	ch, err := s.channels.GetByID(channelID)
	if err != nil {
		return nil, fmt.Errorf("通道不存在")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.play.StartPlay(ctx, ch.DeviceID, ch.GBDeviceID)
}

func (s *Service) StopPlay(channelID int) error {
	ch, err := s.channels.GetByID(channelID)
	if err != nil {
		return err
	}
	return s.play.StopPlay(ch.DeviceID, ch.GBDeviceID)
}

func (s *Service) Broadcast(channelID int, broadcastMode bool) (*dto.AudioBroadcastResult, error) {
	ch, err := s.channels.GetByID(channelID)
	if err != nil {
		return nil, fmt.Errorf("通道不存在")
	}
	return s.play.AudioBroadcast(ch.DeviceID, ch.GBDeviceID, broadcastMode)
}

func (s *Service) BroadcastStop(channelID int) error {
	ch, err := s.channels.GetByID(channelID)
	if err != nil {
		return err
	}
	return s.play.StopAudioBroadcast(ch.DeviceID, ch.GBDeviceID)
}

func (s *Service) PTZ(channelID int, command string, panSpeed, tiltSpeed, zoomSpeed int) error {
	ch, err := s.channels.GetByID(channelID)
	if err != nil {
		return err
	}
	return s.ptz.Control(ch.DeviceID, ch.GBDeviceID, command, panSpeed, tiltSpeed, zoomSpeed)
}

func (s *Service) PlaybackQuery(channelID int, startTime, endTime string) (any, error) {
	ch, err := s.channels.GetByID(channelID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.playback.QueryRecord(ctx, ch.DeviceID, ch.GBDeviceID, startTime, endTime)
}

func (s *Service) PlaybackStart(channelID int, startTime, endTime string) (*dto.StreamContent, error) {
	ch, err := s.channels.GetByID(channelID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.playback.StartPlayback(ctx, ch.DeviceID, ch.GBDeviceID, startTime, endTime)
}

func (s *Service) PlaybackStop(channelID int, stream string) error {
	ch, err := s.channels.GetByID(channelID)
	if err != nil {
		return err
	}
	return s.playback.StopPlayback(ch.DeviceID, ch.GBDeviceID, stream)
}

func (s *Service) PlaybackPause(stream string) error  { return s.playback.PausePlayback(stream) }
func (s *Service) PlaybackResume(stream string) error { return s.playback.ResumePlayback(stream) }
func (s *Service) PlaybackSpeed(stream string, speed float64) error {
	return s.playback.SpeedPlayback(stream, speed)
}
func (s *Service) PlaybackSeek(stream string, seekTime int64) error {
	return s.playback.SeekPlayback(stream, seekTime)
}

func (s *Service) MapList(query string, channelType *int, online, hasRecordPlan *bool) ([]View, error) {
	rows, _, err := s.channels.ListCommon(1, 10000, query, channelType, online, hasRecordPlan)
	if err != nil {
		return nil, err
	}
	out := make([]View, len(rows))
	for i, ch := range rows {
		out[i] = toView(ch)
	}
	return out, nil
}

func (s *Service) ListByCivilCode(page, count int, query string, channelType *int, online *bool, civilCode string) ([]View, int64, error) {
	rows, total, err := s.channels.ListByCivilCode(page, count, query, channelType, online, civilCode)
	if err != nil {
		return nil, 0, err
	}
	out := make([]View, len(rows))
	for i, ch := range rows {
		out[i] = toView(ch)
	}
	return out, total, nil
}

func (s *Service) ListByGroupParent(page, count int, query string, channelType *int, online *bool, groupDeviceID string) ([]View, int64, error) {
	rows, total, err := s.channels.ListByGroupParent(page, count, query, channelType, online, groupDeviceID)
	if err != nil {
		return nil, 0, err
	}
	out := make([]View, len(rows))
	for i, ch := range rows {
		out[i] = toView(ch)
	}
	return out, total, nil
}

func (s *Service) AddToRegion(civilCode string, channelIDs []int) error {
	if civilCode == "" {
		return fmt.Errorf("未添加行政区划")
	}
	cnt, err := s.channels.CountCommonByIDs(channelIDs)
	if err != nil {
		return err
	}
	if cnt == 0 {
		return fmt.Errorf("所有通道Id不存在")
	}
	return s.channels.SetCivilCode(civilCode, channelIDs)
}

func (s *Service) DeleteFromRegion(channelIDs []int) error {
	if len(channelIDs) == 0 {
		return fmt.Errorf("参数异常")
	}
	cnt, err := s.channels.CountCommonByIDs(channelIDs)
	if err != nil {
		return err
	}
	if cnt == 0 {
		return fmt.Errorf("所有通道Id不存在")
	}
	return s.channels.ClearCivilCode(channelIDs)
}

func (s *Service) AddToGroup(parentID, businessGroup string, channelIDs []int) error {
	parentID = strings.TrimSpace(parentID)
	businessGroup = strings.TrimSpace(businessGroup)
	if parentID == "" {
		return fmt.Errorf("请选择虚拟组织节点，不要选择业务分组根节点")
	}
	if businessGroup == "" {
		resolved, err := s.groups.ResolveBusinessGroup(parentID)
		if err != nil {
			return fmt.Errorf("未找到所属业务分组，请重新选择虚拟组织节点")
		}
		businessGroup = resolved
	}
	if gbGroupTypeCode(parentID) == "215" {
		return fmt.Errorf("请选择虚拟组织节点（216），通道不能挂在业务分组根节点上")
	}
	cnt, err := s.channels.CountCommonByIDs(channelIDs)
	if err != nil {
		return err
	}
	if cnt == 0 {
		return fmt.Errorf("所有通道Id不存在")
	}
	return s.channels.SetGroup(parentID, businessGroup, channelIDs)
}

func gbGroupTypeCode(deviceID string) string {
	if len(deviceID) < 13 {
		return ""
	}
	code := deviceID[10:13]
	if code == "215" || code == "216" {
		return code
	}
	return ""
}

func (s *Service) DeleteFromGroup(channelIDs []int) error {
	if len(channelIDs) == 0 {
		return fmt.Errorf("参数异常")
	}
	cnt, err := s.channels.CountCommonByIDs(channelIDs)
	if err != nil {
		return err
	}
	if cnt == 0 {
		return fmt.Errorf("所有通道Id不存在")
	}
	return s.channels.ClearGroupParent(channelIDs)
}
