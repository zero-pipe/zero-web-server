package sipinfra

import (
	"github.com/zero-pipe/gb28181-go/manscdp"
	"github.com/zero-pipe/gb28181-go/mansrtsp"
	"github.com/zero-pipe/gb28181-go/ptz"
	"github.com/zero-pipe/gb28181-go/sdp"

	domainptz "zero-web-kit/internal/domain/ptz"
)

type CatalogItem = manscdp.CatalogItem
type RecordItem = manscdp.RecordItem
type AlarmNotify = manscdp.AlarmNotify
type MobilePositionNotify = manscdp.MobilePositionNotify

// GBMessage mirrors manscdp.Message but maps presets to domainptz.Preset
// so existing preset managers keep working unchanged.
type GBMessage struct {
	Root        string
	CmdType     string
	SN          string
	DeviceID    string
	SumNum      int
	Raw         []byte
	Items       []CatalogItem
	RecordItems []RecordItem
	PresetItems []domainptz.Preset
	Alarm       *AlarmNotify
	Position    *MobilePositionNotify
}

func ParseGBXML(body []byte) (*GBMessage, error) {
	msg, err := manscdp.Parse(body)
	if err != nil {
		return nil, err
	}
	out := &GBMessage{
		Root:        msg.Root,
		CmdType:     msg.CmdType,
		SN:          msg.SN,
		DeviceID:    msg.DeviceID,
		SumNum:      msg.SumNum,
		Raw:         msg.Raw,
		Items:       msg.Items,
		RecordItems: msg.RecordItems,
		Alarm:       msg.Alarm,
		Position:    msg.Position,
	}
	if len(msg.PresetItems) > 0 {
		out.PresetItems = make([]domainptz.Preset, len(msg.PresetItems))
		for i, p := range msg.PresetItems {
			out.PresetItems[i] = domainptz.Preset{PresetID: p.PresetID, PresetName: p.PresetName}
		}
	}
	return out, nil
}

func BuildCatalogQuery(deviceID, platformID, sn string) string {
	_ = platformID
	return manscdp.BuildCatalogQuery(deviceID, sn)
}

func BuildDeviceInfoQuery(deviceID, sn string) string {
	return manscdp.BuildDeviceInfoQuery(deviceID, sn)
}

func BuildDeviceControlPTZ(deviceID, channelID, ptzCmd string) string {
	_ = deviceID
	return manscdp.BuildDeviceControlPTZ(channelID, "1", ptzCmd)
}

func BuildPresetQuery(channelID, sn string) string {
	return manscdp.BuildPresetQuery(channelID, sn)
}

func FrontEndCmdString(cmdCode, parameter1, parameter2, combineCode2 int) string {
	return ptz.FrontEndCmd(cmdCode, parameter1, parameter2, combineCode2)
}

func BuildDeviceControlFrontEnd(channelID, sn, ptzCmd string) string {
	return manscdp.BuildDeviceControlFrontEnd(channelID, sn, ptzCmd)
}

func BuildRecordInfoQuery(channelID, sn, startTime, endTime string) string {
	return manscdp.BuildRecordInfoQuery(channelID, sn, startTime, endTime)
}

func BuildPlaybackSDP(deviceID, channelID, sdpIP string, port int, ssrc, streamMode, startTime, endTime string) string {
	return sdp.BuildPlayback(deviceID, channelID, sdpIP, port, ssrc, streamMode, startTime, endTime)
}

func BuildPlatformKeepalive(platformID, sn string) string {
	return manscdp.BuildPlatformKeepalive(platformID, sn)
}

func BuildDownloadSDP(deviceID, channelID, sdpIP string, port int, ssrc, streamMode, startTime, endTime string, downloadSpeed int) string {
	return sdp.BuildDownload(deviceID, channelID, sdpIP, port, ssrc, streamMode, startTime, endTime, downloadSpeed)
}

func BuildPlaybackPause(cseq int) string {
	return mansrtsp.Pause(cseq)
}

func BuildPlaybackResume(cseq int) string {
	return mansrtsp.Resume(cseq)
}

func BuildPlaybackSpeed(cseq int, speed float64) string {
	return mansrtsp.Speed(cseq, speed)
}

func BuildPlaybackSeek(cseq int, seekTime int64) string {
	return mansrtsp.Seek(cseq, seekTime)
}

func BuildPlaybackSeekSpeed(cseq int, seekTime int64, speed float64) string {
	return mansrtsp.SeekSpeed(cseq, seekTime, speed)
}

func BuildCatalogNotify(platformDeviceID, sn string, items []CatalogItem) string {
	return manscdp.BuildCatalogNotify(platformDeviceID, sn, items)
}

func BuildBroadcastNotify(sourceID, targetChannelID, sn string) string {
	return manscdp.BuildBroadcastNotify(sourceID, targetChannelID, sn)
}

func BuildPlaySDP(deviceID, sdpIP string, port int, ssrc, streamMode, streamIdentification string) string {
	return sdp.BuildPlay(deviceID, sdpIP, port, ssrc, streamMode, streamIdentification)
}

func PTZCommand(direction string, horizonSpeed, verticalSpeed, zoomSpeed int) string {
	return ptz.Command(direction, horizonSpeed, verticalSpeed, zoomSpeed)
}

// ParseInviteAnswerMedia 从 INVITE 200 OK 的 SDP 解析摄像机媒体地址（TCP-ACTIVE 用）。
func ParseInviteAnswerMedia(sdpBody string) (host string, port int, err error) {
	return sdp.ParseAnswerMedia(sdpBody)
}

func extractSDPPort(sdpBody string) int {
	return sdp.ExtractVideoPort(sdpBody)
}

func extractTag(body []byte, tag string) string {
	return manscdp.ExtractTag(body, tag)
}