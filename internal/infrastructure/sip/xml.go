package sipinfra

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

type GBMessage struct {
	Root        string
	CmdType     string
	SN          string
	DeviceID    string
	SumNum      int
	Raw         []byte
	Items       []CatalogItem
	RecordItems []RecordItem
	Alarm       *AlarmNotify
	Position    *MobilePositionNotify
}

type RecordItem struct {
	DeviceID   string
	Name       string
	FilePath   string
	FileSize   string
	StartTime  string
	EndTime    string
	Secrecy    int
	Type       string
	RecorderID string
}

type AlarmNotify struct {
	AlarmPriority    string
	AlarmMethod      string
	AlarmTime        string
	AlarmDescription string
	Longitude        float64
	Latitude         float64
	AlarmType        int
}

type MobilePositionNotify struct {
	Longitude float64
	Latitude  float64
	Speed     float64
	Direction float64
	Altitude  float64
	Time      string
}

type CatalogItem struct {
	DeviceID     string
	Name         string
	Manufacturer string
	Model        string
	Owner        string
	CivilCode    string
	Address      string
	Parental     int
	ParentID     string
	Status       string
	Longitude    float64
	Latitude     float64
	PTZType      int
}

func ParseGBXML(body []byte) (*GBMessage, error) {
	dec := xml.NewDecoder(bytes.NewReader(body))
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}

	msg := &GBMessage{Raw: body}
	var current *CatalogItem
	inDeviceList := false

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			name := localName(el.Name.Local)
			if msg.Root == "" && isRootTag(name) {
				msg.Root = name
			}
			if name == "DeviceList" {
				inDeviceList = true
			}
			if inDeviceList && name == "Item" {
				current = &CatalogItem{}
			}
		case xml.EndElement:
			name := localName(el.Name.Local)
			if name == "Item" && current != nil {
				msg.Items = append(msg.Items, *current)
				current = nil
			}
			if name == "DeviceList" {
				inDeviceList = false
			}
		case xml.CharData:
			text := strings.TrimSpace(string(el))
			if text == "" {
				continue
			}
			if current != nil {
				assignCatalogField(current, dec, text)
				continue
			}
			// read last start element name from stack - simplified approach
			switch {
			case strings.Contains(string(body), "<CmdType>"+text+"</CmdType>"):
				if msg.CmdType == "" {
					msg.CmdType = text
				}
			}
		}
	}

	// fallback field extraction
	msg.CmdType = extractTag(body, "CmdType")
	msg.SN = extractTag(body, "SN")
	msg.DeviceID = extractTag(body, "DeviceID")
	fmt.Sscanf(extractTag(body, "SumNum"), "%d", &msg.SumNum)

	if len(msg.Items) == 0 {
		msg.Items = parseCatalogItemsFallback(body)
	} else {
		nonEmpty := 0
		for _, it := range msg.Items {
			if it.DeviceID != "" {
				nonEmpty++
			}
		}
		if nonEmpty == 0 {
			msg.Items = parseCatalogItemsFallback(body)
		}
	}
	if msg.CmdType == "Catalog" && len(msg.Items) == 0 {
		msg.Items = parseCatalogItemsFallback(body)
	}
	if msg.CmdType == "RecordInfo" {
		msg.RecordItems = parseRecordItemsFallback(body)
	}
	if msg.CmdType == "Alarm" {
		msg.Alarm = parseAlarmNotify(body)
	}
	if msg.CmdType == "MobilePosition" {
		msg.Position = parseMobilePositionNotify(body)
	}
	if msg.Root == "" {
		for _, tag := range []string{"Notify", "Response", "Query", "Control"} {
			if strings.Contains(string(body), "<"+tag+">") {
				msg.Root = tag
				break
			}
		}
	}
	return msg, nil
}

func isRootTag(name string) bool {
	switch name {
	case "Notify", "Response", "Query", "Control":
		return true
	default:
		return false
	}
}

func localName(name string) string {
	if idx := strings.Index(name, ":"); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

func extractTag(body []byte, tag string) string {
	return extractTagFold(string(body), tag)
}

func extractTagFold(s, tag string) string {
	lower := strings.ToLower(s)
	tagLower := strings.ToLower(tag)
	startTag := "<" + tagLower + ">"
	endTag := "</" + tagLower + ">"
	i := strings.Index(lower, startTag)
	if i < 0 {
		return ""
	}
	i += len(startTag)
	j := strings.Index(lower[i:], endTag)
	if j < 0 {
		return ""
	}
	return strings.TrimSpace(s[i : i+j])
}

func parseCatalogItemsFallback(body []byte) []CatalogItem {
	s := string(body)
	lower := strings.ToLower(s)
	items := make([]CatalogItem, 0)
	for {
		i := strings.Index(lower, "<item")
		if i < 0 {
			break
		}
		closeIdx := strings.Index(lower[i:], "</item>")
		if closeIdx < 0 {
			break
		}
		end := i + closeIdx + len("</item>")
		chunk := s[i:end]
		item := CatalogItem{
			DeviceID:     extractTagFold(chunk, "DeviceID"),
			Name:         extractTagFold(chunk, "Name"),
			Manufacturer: extractTagFold(chunk, "Manufacturer"),
			Model:        extractTagFold(chunk, "Model"),
			Owner:        extractTagFold(chunk, "Owner"),
			CivilCode:    extractTagFold(chunk, "CivilCode"),
			Address:      extractTagFold(chunk, "Address"),
			ParentID:     extractTagFold(chunk, "ParentID"),
			Status:       extractTagFold(chunk, "Status"),
		}
		fmt.Sscanf(extractTagFold(chunk, "Parental"), "%d", &item.Parental)
		fmt.Sscanf(extractTagFold(chunk, "Longitude"), "%f", &item.Longitude)
		fmt.Sscanf(extractTagFold(chunk, "Latitude"), "%f", &item.Latitude)
		if info := extractInfoPTZType(chunk); info >= 0 {
			item.PTZType = info
		}
		items = append(items, item)
		s = s[end:]
		lower = strings.ToLower(s)
	}
	return items
}

func extractInfoPTZType(chunk string) int {
	i := strings.Index(strings.ToLower(chunk), "<info>")
	if i < 0 {
		return -1
	}
	sub := chunk[i:]
	val := extractTagFold(sub, "PTZType")
	var ptz int
	fmt.Sscanf(val, "%d", &ptz)
	return ptz
}

func assignCatalogField(item *CatalogItem, dec *xml.Decoder, text string) {
	// unused in fallback mode
}

func BuildCatalogQuery(deviceID, platformID, sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>Catalog</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
</Query>`, sn, deviceID)
}

func BuildDeviceInfoQuery(deviceID, sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>DeviceInfo</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
</Query>`, sn, deviceID)
}

func BuildDeviceControlPTZ(deviceID, channelID, ptzCmd string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Control>
<CmdType>DeviceControl</CmdType>
<SN>%d</SN>
<DeviceID>%s</DeviceID>
<PTZCmd>%s</PTZCmd>
</Control>`, 1, channelID, ptzCmd)
}

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

func BuildPlaybackSDP(deviceID, channelID, sdpIP string, port int, ssrc, streamMode, startTime, endTime string) string {
	mediaProto := "RTP/AVP"
	setup := ""
	switch streamMode {
	case "TCP-ACTIVE":
		mediaProto = "TCP/RTP/AVP"
		setup = "a=setup:active\r\na=connection:new\r\n"
	case "TCP-PASSIVE":
		mediaProto = "TCP/RTP/AVP"
		setup = "a=setup:passive\r\na=connection:new\r\n"
	}
	tStart, tEnd := parseTimeRange(startTime, endTime)
	return fmt.Sprintf(
		"v=0\r\no=%s 0 0 IN IP4 %s\r\ns=Playback\r\nu=%s:0\r\nc=IN IP4 %s\r\nt=%d %d\r\nm=video %d %s 96 98 97 99\r\na=recvonly\r\na=rtpmap:96 PS/90000\r\na=rtpmap:98 H264/90000\r\na=rtpmap:97 MPEG4/90000\r\na=rtpmap:99 H265/90000\r\n%sy=%s\r\n",
		deviceID, sdpIP, channelID, sdpIP, tStart, tEnd, port, mediaProto, setup, ssrc,
	)
}

func parseTimeRange(start, end string) (int64, int64) {
	const layout = "2006-01-02 15:04:05"
	ts, _ := time.ParseInLocation(layout, start, time.Local)
	te, _ := time.ParseInLocation(layout, end, time.Local)
	return ts.Unix(), te.Unix()
}

func parseRecordItemsFallback(body []byte) []RecordItem {
	s := string(body)
	items := make([]RecordItem, 0)
	for {
		i := strings.Index(s, "<Item")
		if i < 0 {
			break
		}
		j := strings.Index(s[i:], "</Item>")
		if j < 0 {
			break
		}
		chunk := s[i : i+j+7]
		item := RecordItem{
			DeviceID: extractTag([]byte(chunk), "DeviceID"),
			Name:     extractTag([]byte(chunk), "Name"),
			FilePath: extractTag([]byte(chunk), "FilePath"),
			FileSize: extractTag([]byte(chunk), "FileSize"),
			StartTime: extractTag([]byte(chunk), "StartTime"),
			EndTime:   extractTag([]byte(chunk), "EndTime"),
			Type:      extractTag([]byte(chunk), "Type"),
			RecorderID: extractTag([]byte(chunk), "RecorderID"),
		}
		fmt.Sscanf(extractTag([]byte(chunk), "Secrecy"), "%d", &item.Secrecy)
		items = append(items, item)
		s = s[i+j+7:]
	}
	return items
}

func parseAlarmNotify(body []byte) *AlarmNotify {
	a := &AlarmNotify{
		AlarmPriority:    extractTag(body, "AlarmPriority"),
		AlarmMethod:      extractTag(body, "AlarmMethod"),
		AlarmTime:        extractTag(body, "AlarmTime"),
		AlarmDescription: extractTag(body, "AlarmDescription"),
	}
	fmt.Sscanf(extractTag(body, "Longitude"), "%f", &a.Longitude)
	fmt.Sscanf(extractTag(body, "Latitude"), "%f", &a.Latitude)
	fmt.Sscanf(extractTag(body, "AlarmType"), "%d", &a.AlarmType)
	return a
}

func parseMobilePositionNotify(body []byte) *MobilePositionNotify {
	p := &MobilePositionNotify{Time: extractTag(body, "Time")}
	fmt.Sscanf(extractTag(body, "Longitude"), "%f", &p.Longitude)
	fmt.Sscanf(extractTag(body, "Latitude"), "%f", &p.Latitude)
	fmt.Sscanf(extractTag(body, "Speed"), "%f", &p.Speed)
	fmt.Sscanf(extractTag(body, "Direction"), "%f", &p.Direction)
	fmt.Sscanf(extractTag(body, "Altitude"), "%f", &p.Altitude)
	return p
}

func BuildPlatformKeepalive(platformID, sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Notify>
<CmdType>Keepalive</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
<Status>OK</Status>
</Notify>`, sn, platformID)
}

func BuildDownloadSDP(deviceID, channelID, sdpIP string, port int, ssrc, streamMode, startTime, endTime string, downloadSpeed int) string {
	mediaProto := "RTP/AVP"
	setup := ""
	switch streamMode {
	case "TCP-ACTIVE":
		mediaProto = "TCP/RTP/AVP"
		setup = "a=setup:active\r\na=connection:new\r\n"
	case "TCP-PASSIVE":
		mediaProto = "TCP/RTP/AVP"
		setup = "a=setup:passive\r\na=connection:new\r\n"
	}
	if downloadSpeed <= 0 {
		downloadSpeed = 4
	}
	tStart, tEnd := parseTimeRange(startTime, endTime)
	return fmt.Sprintf(
		"v=0\r\no=%s 0 0 IN IP4 %s\r\ns=Download\r\nu=%s:0\r\nc=IN IP4 %s\r\nt=%d %d\r\nm=video %d %s 96 98 97 99\r\na=recvonly\r\na=rtpmap:96 PS/90000\r\na=rtpmap:98 H264/90000\r\na=rtpmap:97 MPEG4/90000\r\na=rtpmap:99 H265/90000\r\n%sa=downloadspeed:%d\r\ny=%s\r\n",
		deviceID, sdpIP, channelID, sdpIP, tStart, tEnd, port, mediaProto, setup, downloadSpeed, ssrc,
	)
}

func BuildPlaybackPause(cseq int) string {
	return fmt.Sprintf("PAUSE RTSP/1.0\r\nCSeq: %d\r\nPauseTime: now\r\n", cseq)
}

func BuildPlaybackResume(cseq int) string {
	return fmt.Sprintf("PLAY RTSP/1.0\r\nCSeq: %d\r\nRange: npt=now-\r\n", cseq)
}

func BuildPlaybackSpeed(cseq int, speed float64) string {
	return fmt.Sprintf("PLAY RTSP/1.0\r\nCSeq: %d\r\nScale: %.6f\r\n", cseq, speed)
}

func BuildPlaybackSeek(cseq int, seekTime int64) string {
	return fmt.Sprintf("PLAY RTSP/1.0\r\nCSeq: %d\r\nRange: npt=%d-\r\n", cseq, seekTime)
}

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

func BuildBroadcastNotify(sourceID, targetChannelID, sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>`+"\r\n"+
		"<Notify>\r\n"+
		"<CmdType>Broadcast</CmdType>\r\n"+
		"<SN>%s</SN>\r\n"+
		"<SourceID>%s</SourceID>\r\n"+
		"<TargetID>%s</TargetID>\r\n"+
		"</Notify>\r\n", sn, sourceID, targetChannelID)
}

func BuildPlaySDP(deviceID, sdpIP string, port int, ssrc, streamMode, streamIdentification string) string {
	mediaProto := "RTP/AVP"
	setup := ""
	switch streamMode {
	case "TCP-ACTIVE":
		mediaProto = "TCP/RTP/AVP"
		setup = "a=setup:active\r\na=connection:new\r\n"
	case "TCP-PASSIVE":
		mediaProto = "TCP/RTP/AVP"
		setup = "a=setup:passive\r\na=connection:new\r\n"
	}
	streamAttr := ""
	if streamIdentification != "" {
		streamAttr = fmt.Sprintf("a=%s\r\n", streamIdentification)
	}
	return fmt.Sprintf(
		"v=0\r\no=%s 0 0 IN IP4 %s\r\ns=Play\r\nc=IN IP4 %s\r\nt=0 0\r\nm=video %d %s 96 98 97 99\r\na=recvonly\r\na=rtpmap:96 PS/90000\r\na=rtpmap:98 H264/90000\r\na=rtpmap:97 MPEG4/90000\r\na=rtpmap:99 H265/90000\r\n%s%sy=%s\r\n",
		deviceID, sdpIP, sdpIP, port, mediaProto, streamAttr, setup, ssrc,
	)
}

func PTZCommand(direction string, horizonSpeed, verticalSpeed, zoomSpeed int) string {
	if horizonSpeed <= 0 {
		horizonSpeed = 100
	}
	if verticalSpeed <= 0 {
		verticalSpeed = 100
	}
	if zoomSpeed <= 0 {
		zoomSpeed = 16
	}
	var cmd byte = 0x00
	switch strings.ToLower(direction) {
	case "left":
		cmd = 0x01
	case "right":
		cmd = 0x02
	case "up":
		cmd = 0x03
	case "down":
		cmd = 0x04
	case "upleft":
		cmd = 0x05
	case "upright":
		cmd = 0x06
	case "downleft":
		cmd = 0x07
	case "downright":
		cmd = 0x08
	case "zoomin":
		cmd = 0x09
	case "zoomout":
		cmd = 0x0a
	case "stop":
		cmd = 0x00
	default:
		cmd = 0x00
	}
	check := (0xA5 + 0x0F + 0x01 + int(cmd) + horizonSpeed + verticalSpeed + zoomSpeed) % 256
	return fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X",
		0xA5, 0x0F, 0x01, cmd, byte(horizonSpeed), byte(verticalSpeed), byte(zoomSpeed), byte(check))
}
