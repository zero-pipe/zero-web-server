package sipinfra

import (
	"fmt"
	"strconv"

	domaindevice "zero-web-kit/internal/domain/device"

	"github.com/emiago/sipgo/sip"
)

func BuildSubscribeCatalog(deviceID, sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>Catalog</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
</Query>`, sn, deviceID)
}

func BuildSubscribeAlarm(deviceID, sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>Alarm</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
</Query>`, sn, deviceID)
}

func BuildSubscribeMobilePosition(deviceID, sn string, interval int) string {
	if interval <= 0 {
		interval = 5
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>MobilePosition</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
<Interval>%d</Interval>
</Query>`, sn, deviceID, interval)
}

func (s *Server) SendSubscribe(device *domaindevice.Device, eventType, body string, expiresSec int) error {
	if expiresSec > 0 && expiresSec < 30 {
		expiresSec = 30
	}
	recipient := sip.Uri{User: device.DeviceID, Host: device.IP, Port: device.Port}
	req := sip.NewRequest(sip.SUBSCRIBE, recipient)
	req.SetBody([]byte(body))
	req.AppendHeader(sip.NewHeader("Content-Type", "Application/MANSCDP+xml"))
	req.AppendHeader(sip.NewHeader("Event", eventType))
	req.AppendHeader(sip.NewHeader("Expires", strconv.Itoa(expiresSec)))
	from := &sip.FromHeader{Address: sip.Uri{User: s.cfg.ID, Host: s.cfg.Domain}}
	to := &sip.FromHeader{Address: recipient}
	req.AppendHeader(from)
	req.AppendHeader(to)
	return s.client.WriteRequest(req)
}

func (s *Server) SendSubscribeCancel(device *domaindevice.Device, eventType, body string) error {
	return s.SendSubscribe(device, eventType, body, 0)
}
