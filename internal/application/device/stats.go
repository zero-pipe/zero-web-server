package deviceapp

import (
	"context"
	"time"
	domaindevice "zero-web-server/internal/domain/device"
)

func (s *Service) GetKeepaliveStatistics(deviceID string, count int) []domaindevice.TimeStatistics {
	return s.getTimeStatistics(deviceID, count, true)
}

func (s *Service) GetRegisterStatistics(deviceID string, count int) []domaindevice.TimeStatistics {
	return s.getTimeStatistics(deviceID, count, false)
}

// Guard 向国标设备发送布防/撤防（SetGuard / ResetGuard）。
func (s *Service) getTimeStatistics(deviceID string, count int, keepalive bool) []domaindevice.TimeStatistics {
	if s.redis == nil || deviceID == "" {
		return []domaindevice.TimeStatistics{}
	}
	var (
		stamps []int64
		err    error
	)
	if keepalive {
		stamps, err = s.redis.GetKeepaliveTimeStamps(context.Background(), deviceID, count)
	} else {
		stamps, err = s.redis.GetRegisterTimeStamps(context.Background(), deviceID, count)
	}
	if err != nil || len(stamps) == 0 {
		return []domaindevice.TimeStatistics{}
	}
	return formatTimeStatistics(stamps, count)
}

func formatTimeStatistics(timeStampList []int64, count int) []domaindevice.TimeStatistics {
	list := make([]domaindevice.TimeStatistics, 0, len(timeStampList))
	for i, ts := range timeStampList {
		item := domaindevice.TimeStatistics{
			Time: time.UnixMilli(ts).Format("2006-01-02 15:04:05"),
		}
		if i > 0 {
			item.TimeDiff = (ts - timeStampList[i-1]) / 1000
		}
		list = append(list, item)
	}
	if len(list) > 1 {
		list = list[1:]
	}
	if count > 0 && len(list) > count {
		list = list[len(list)-count:]
	}
	return list
}
