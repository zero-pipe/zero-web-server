package deviceapp

import (
	"fmt"

	domainchannel "zero-web-server/internal/domain/channel"
)

// checkChannelCapByID 按通道主键校验资源能力（无声明不拦截；声明不支持或上次失败则拒绝）。
func (s *Service) checkChannelCapByID(channelID int, op string) error {
	ch, err := s.channels.GetByID(channelID)
	if err != nil || ch == nil {
		return nil
	}
	return s.checkChannelCap(ch, op)
}

// markChannelCapByID 运行时更新能力可用性。
func (s *Service) markChannelCapByID(channelID int, op string, ok bool) {
	ch, err := s.channels.GetByID(channelID)
	if err != nil || ch == nil {
		return
	}
	s.markChannelCap(ch, op, ok)
}

// checkChannelCapByGB 按设备/通道国标码校验资源能力。
func (s *Service) checkChannelCapByGB(deviceID, channelDeviceID, op string) error {
	if s.caps == nil || channelDeviceID == "" {
		return nil
	}
	ch, err := s.channels.GetOne(deviceID, channelDeviceID)
	if err != nil || ch == nil {
		return nil
	}
	return s.checkChannelCap(ch, op)
}

// markChannelCapByGB 按设备/通道国标码更新能力可用性。
func (s *Service) markChannelCapByGB(deviceID, channelDeviceID, op string, ok bool) {
	if s.caps == nil || channelDeviceID == "" {
		return
	}
	ch, err := s.channels.GetOne(deviceID, channelDeviceID)
	if err != nil || ch == nil {
		return
	}
	s.markChannelCap(ch, op, ok)
}

func (s *Service) checkChannelCap(ch *domainchannel.Channel, op string) error {
	if s.caps == nil || ch == nil {
		return nil
	}
	res, err := s.caps.GetResourceByInternalCode(ch.InternalCode)
	if err != nil || res == nil {
		return nil
	}
	caps, err := s.caps.ListCapabilities("resource", res.ID)
	if err != nil {
		return nil
	}
	for _, c := range caps {
		if c.Operation != op {
			continue
		}
		if !c.Supported {
			return fmt.Errorf("该通道不支持%s", capText(op))
		}
		if !c.Available {
			return fmt.Errorf("该通道%s能力当前不可用（设备离线或上次调用失败）", capText(op))
		}
	}
	return nil
}

func (s *Service) markChannelCap(ch *domainchannel.Channel, op string, ok bool) {
	if s.caps == nil || ch == nil {
		return
	}
	res, err := s.caps.GetResourceByInternalCode(ch.InternalCode)
	if err != nil || res == nil {
		return
	}
	_ = s.caps.SetCapabilityAvailable("resource", res.ID, op, ok)
}

func capText(op string) string {
	switch op {
	case "snapshot":
		return "抓拍"
	case "audio":
		return "音频"
	case "ptz":
		return "云台控制"
	case "playback":
		return "回放"
	default:
		return op
	}
}
