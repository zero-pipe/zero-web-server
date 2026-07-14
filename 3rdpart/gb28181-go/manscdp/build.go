package manscdp

import (
	"fmt"
	"strings"
)

// BuildCatalogQuery builds a Catalog query MESSAGE body.
func BuildCatalogQuery(deviceID, sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>Catalog</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
</Query>`, sn, deviceID)
}

// BuildDeviceInfoQuery builds a DeviceInfo query MESSAGE body.
func BuildDeviceInfoQuery(deviceID, sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>DeviceInfo</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
</Query>`, sn, deviceID)
}

// BuildDeviceControlPTZ builds a DeviceControl body with PTZCmd.
func BuildDeviceControlPTZ(channelID, ptzCmd string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Control>
<CmdType>DeviceControl</CmdType>
<SN>%d</SN>
<DeviceID>%s</DeviceID>
<PTZCmd>%s</PTZCmd>
</Control>`, 1, channelID, ptzCmd)
}

// BuildPresetQuery builds a PresetQuery MESSAGE body.
func BuildPresetQuery(channelID, sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>PresetQuery</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
</Query>`, sn, channelID)
}

// BuildDeviceControlFrontEnd builds a DeviceControl body for front-end commands.
func BuildDeviceControlFrontEnd(channelID, sn, ptzCmd string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Control>
<CmdType>DeviceControl</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
<PTZCmd>%s</PTZCmd>
</Control>`, sn, channelID, ptzCmd)
}

// BuildRecordInfoQuery builds a RecordInfo query MESSAGE body.
func BuildRecordInfoQuery(channelID, sn, startTime, endTime string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>RecordInfo</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
<StartTime>%s</StartTime>
<EndTime>%s</EndTime>
<Secrecy>0</Secrecy>
<Type>all</Type>
</Query>`, sn, channelID, startTime, endTime)
}

// BuildPlatformKeepalive builds an upstream platform keepalive notify.
func BuildPlatformKeepalive(platformID, sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Notify>
<CmdType>Keepalive</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
<Status>OK</Status>
</Notify>`, sn, platformID)
}

// BuildCatalogNotify builds a Catalog notify for cascade.
func BuildCatalogNotify(platformDeviceID, sn string, items []CatalogItem) string {
	var buf strings.Builder
	buf.WriteString(`<?xml version="1.0" encoding="GB2312"?>` + "\r\n")
	buf.WriteString("<Notify>\r\n")
	buf.WriteString("<CmdType>Catalog</CmdType>\r\n")
	buf.WriteString(fmt.Sprintf("<SN>%s</SN>\r\n", sn))
	buf.WriteString(fmt.Sprintf("<DeviceID>%s</DeviceID>\r\n", platformDeviceID))
	buf.WriteString(fmt.Sprintf("<SumNum>%d</SumNum>\r\n", len(items)))
	buf.WriteString(fmt.Sprintf("<DeviceList Num=\"%d\">\r\n", len(items)))
	for _, it := range items {
		buf.WriteString("<Item>\r\n")
		buf.WriteString(fmt.Sprintf("<DeviceID>%s</DeviceID>\r\n", it.DeviceID))
		buf.WriteString(fmt.Sprintf("<Name>%s</Name>\r\n", it.Name))
		buf.WriteString(fmt.Sprintf("<Status>%s</Status>\r\n", it.Status))
		buf.WriteString("</Item>\r\n")
	}
	buf.WriteString("</DeviceList>\r\n</Notify>\r\n")
	return buf.String()
}

// BuildBroadcastNotify builds an audio broadcast notify.
func BuildBroadcastNotify(sourceID, targetChannelID, sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>`+"\r\n"+
		"<Notify>\r\n"+
		"<CmdType>Broadcast</CmdType>\r\n"+
		"<SN>%s</SN>\r\n"+
		"<SourceID>%s</SourceID>\r\n"+
		"<TargetID>%s</TargetID>\r\n"+
		"</Notify>\r\n", sn, sourceID, targetChannelID)
}

// BuildSubscribeCatalog builds a Catalog subscribe query body.
func BuildSubscribeCatalog(deviceID, sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>Catalog</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
</Query>`, sn, deviceID)
}

// BuildSubscribeAlarm builds an Alarm subscribe query body.
func BuildSubscribeAlarm(deviceID, sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>Alarm</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
</Query>`, sn, deviceID)
}

// BuildSubscribeMobilePosition builds a MobilePosition subscribe query body.
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
