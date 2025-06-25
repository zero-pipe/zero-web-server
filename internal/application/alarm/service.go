package alarmapp

import (
	"strconv"
	"time"

	domainalarm "zero-web-kit/internal/domain/alarm"
	domainchannel "zero-web-kit/internal/domain/channel"
	sipinfra "zero-web-kit/internal/infrastructure/sip"
)

type Service struct {
	alarms   domainalarm.Repository
	channels domainchannel.Repository
}

func NewService(alarms domainalarm.Repository, channels domainchannel.Repository) *Service {
	return &Service{alarms: alarms, channels: channels}
}

func (s *Service) HandleNotify(deviceID, channelGBID string, n *sipinfra.AlarmNotify) error {
	if n == nil {
		return nil
	}
	ch, err := s.channels.GetOne(deviceID, channelGBID)
	if err != nil {
		// try channelGBID as device channel id from notify DeviceID field
		return nil
	}
	alarmTime := time.Now().Unix()
	if n.AlarmTime != "" {
		if t, err := time.ParseInLocation("2006-01-02T15:04:05", n.AlarmTime, time.Local); err == nil {
			alarmTime = t.Unix()
		}
	}
	return s.alarms.Create(&domainalarm.Alarm{
		ChannelID:   ch.ID,
		Description: n.AlarmDescription,
		Longitude:   n.Longitude,
		Latitude:    n.Latitude,
		AlarmType:   n.AlarmType,
		AlarmTime:   alarmTime,
	})
}

func (s *Service) List(page, count int, alarmType *int, beginTime, endTime string) ([]*domainalarm.Alarm, int64, error) {
	return s.alarms.List(page, count, alarmType, parseTimeMs(beginTime), parseTimeMs(endTime))
}

func (s *Service) Delete(ids []int) error { return s.alarms.Delete(ids) }

func (s *Service) Clear(alarmType *int, beginTime, endTime string) error {
	return s.alarms.Clear(alarmType, parseTimeMs(beginTime), parseTimeMs(endTime))
}

func parseTimeMs(v string) int64 {
	if v == "" {
		return 0
	}
	if ms, err := strconv.ParseInt(v, 10, 64); err == nil {
		return ms
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", v, time.Local)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}
