package positionapp

import (
	"time"

	domainchannel "zero-web-kit/internal/domain/channel"
	domainposition "zero-web-kit/internal/domain/position"
	sipinfra "zero-web-kit/internal/infrastructure/sip"
)

type Service struct {
	positions domainposition.Repository
	channels  domainchannel.Repository
}

func NewService(positions domainposition.Repository, channels domainchannel.Repository) *Service {
	return &Service{positions: positions, channels: channels}
}

func (s *Service) HandleNotify(deviceID, channelGBID string, n *sipinfra.MobilePositionNotify) error {
	if n == nil {
		return nil
	}
	ch, err := s.channels.GetOne(deviceID, channelGBID)
	if err != nil {
		return nil
	}
	ts := time.Now().UnixMilli()
	if n.Time != "" {
		if t, err := time.ParseInLocation("2006-01-02T15:04:05", n.Time, time.Local); err == nil {
			ts = t.UnixMilli()
		}
	}
	return s.positions.Create(&domainposition.MobilePosition{
		ChannelID: ch.ID, Timestamp: ts,
		Longitude: n.Longitude, Latitude: n.Latitude,
		Altitude: n.Altitude, Speed: n.Speed, Direction: n.Direction,
		CreateTime: time.Now().Format("2006-01-02 15:04:05"),
	})
}

func (s *Service) HistoryByChannelDBID(channelDBID int, start, end int64) ([]*domainposition.MobilePosition, error) {
	return s.positions.ListByChannel(channelDBID, start, end)
}

func (s *Service) History(deviceID string, channelGBID string, start, end int64) ([]*domainposition.MobilePosition, error) {
	ch, err := s.channels.GetOne(deviceID, channelGBID)
	if err != nil {
		return nil, err
	}
	return s.positions.ListByChannel(ch.ID, start, end)
}

func (s *Service) Latest(channelDBID int) (*domainposition.MobilePosition, error) {
	return s.positions.Latest(channelDBID)
}
