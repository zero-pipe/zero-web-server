package port

import (
	domainchannel "zero-web-server/internal/domain/channel"
	domaindevice "zero-web-server/internal/domain/device"
	domainptz "zero-web-server/internal/domain/ptz"
	domainrecord "zero-web-server/internal/domain/record"
	"zero-web-server/internal/gbsip"
)

// GBInviteSession 是一个进行中的 INVITE 会话视图。
type GBInviteSession interface {
	Progress() float64
	DownloadSpeed() int
}

// GBInviteManager 管理按 stream 索引的 INVITE 会话。
type GBInviteManager interface {
	Get(stream string) (GBInviteSession, bool)
}

// GBInviter 实时点播 / 广播 / 设备控制（级联入站）用到的 SIP 能力。
type GBInviter interface {
	Domain() string
	InviteManager() GBInviteManager
	CloseInviteSession(stream string) error
	SendInvitePlay(device *domaindevice.Device, channel *domainchannel.Channel, sdpBody, ssrc, stream, streamMode string, tcpConnect func(host string, port int) error) error
	SendAudioBroadcast(device *domaindevice.Device, channelGBID string) error
	SendDeviceControl(device *domaindevice.Device, channelID, xmlBody string) error
}

// GBPlaybacker 录像回放 / 下载用到的 SIP 能力。
type GBPlaybacker interface {
	Domain() string
	InviteManager() GBInviteManager
	CloseInviteSession(stream string) error
	SendRecordInfoQuery(device *domaindevice.Device, channelID, startTime, endTime string) (string, <-chan *domainrecord.RecordInfo)
	CancelRecordQuery(sn string)
	SendInviteSession(device *domaindevice.Device, channel *domainchannel.Channel, sdpBody, ssrc, stream string, sessionType gbsip.SessionType, startTime, endTime string, downloadSpeed int) error
	SendPlaybackControl(stream, content string) error
	NextInfoCSeq() int
}

// GBPTZNode PTZ / 预置位 / 前端控制用到的 SIP 能力。
type GBPTZNode interface {
	SendPTZ(device *domaindevice.Device, channelID, direction string, h, v, z int) error
	SendPresetQuery(device *domaindevice.Device, channelID string) (string, <-chan []domainptz.Preset)
	CancelPresetQuery(sn string)
	SendFrontEndCmd(device *domaindevice.Device, channelID string, cmdCode, parameter1, parameter2, combineCode2 int) error
}

// GBDeviceControl 注册后设备控制 / 订阅用到的 SIP 能力。
type GBDeviceControl interface {
	SendCatalogQuery(device *domaindevice.Device) error
	SendDeviceInfoQuery(device *domaindevice.Device) error
	SendDeviceControl(device *domaindevice.Device, channelID, xmlBody string) error
	SendGuardCmd(device *domaindevice.Device, channelID, guardCmd string) error
	SendRecordCmd(device *domaindevice.Device, channelID, recordCmd string) error
	SendDragZoom(device *domaindevice.Device, channelID string, zoomIn bool, length, width, midX, midY, lengthX, lengthY int) error
	SendHomePosition(device *domaindevice.Device, channelID string, enabled, resetTime, presetIndex int) error
	SendBasicParamConfig(device *domaindevice.Device, name string, expiration, heartBeatInterval, heartBeatCount, positionCapability int) error
	QueryConfigDownload(device *domaindevice.Device, configType string) (string, <-chan gbsip.BasicParam)
	QueryHomePosition(device *domaindevice.Device, channelID string) (string, <-chan gbsip.HomePosition)
	QueryCruiseTrackList(device *domaindevice.Device, channelID string) (string, <-chan gbsip.CruiseTrackList)
	QueryCruiseTrack(device *domaindevice.Device, channelID string, number int) (string, <-chan gbsip.CruiseTrack)
	QueryPTZPosition(device *domaindevice.Device, channelID string) (string, <-chan gbsip.PTZPosition)
	QuerySDCardStatus(device *domaindevice.Device, channelID string) (string, <-chan gbsip.SDCardStatus)
	CancelQuery(sn string)
	SendSnapShotConfig(device *domaindevice.Device, channelID string, snapNum, interval int, uploadURL, sessionID string) error
	SendPTZPrecise(device *domaindevice.Device, channelID string, pan, tilt, zoom float64) error
	SendFormatSDCard(device *domaindevice.Device, channelID string) error
	SendTargetTrack(device *domaindevice.Device, channelID, trackType string) error
	SendAlarmCmd(device *domaindevice.Device, channelID, alarmCmd string) error
	SendTeleBoot(device *domaindevice.Device, channelID string) error
	SendIFrameCmd(device *domaindevice.Device, channelID string) error
	SendDeviceStatusQuery(device *domaindevice.Device) (string, <-chan GBDeviceStatus)
	CancelDeviceStatusQuery(sn string)
	SendSubscribe(device *domaindevice.Device, eventType, body string, expiresSec int) error
	SendSubscribeCancel(device *domaindevice.Device, eventType, body string) error
}

// GBDeviceStatus 是 DeviceStatus 查询应答（9.3 / A.2.4.2）。
type GBDeviceStatus struct {
	Online     string `json:"online"`
	Status     string `json:"status"`
	Encode     string `json:"encode"`
	Record     string `json:"record"`
	DeviceTime string `json:"deviceTime"`
}

// GBSession 是 GB28181 SIP 能力的组合端口。
type GBSession interface {
	GBInviter
	GBPlaybacker
	GBPTZNode
	GBDeviceControl
}
