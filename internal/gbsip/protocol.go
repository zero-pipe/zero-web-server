// Package gbsip holds GB28181 protocol primitives (pure helpers and type aliases)
// that the application layer may use without importing the SIP infrastructure.
package gbsip

import (
	"github.com/zero-pipe/gb28181-go/manscdp"
	"github.com/zero-pipe/gb28181-go/mansrtsp"
	"github.com/zero-pipe/gb28181-go/sdp"
	gbserver "github.com/zero-pipe/gb28181-go/server"
	"github.com/zero-pipe/gb28181-go/session"
	"github.com/zero-pipe/gb28181-go/ssrc"
	"github.com/zero-pipe/gb28181-go/transport"
)

// 协议消息类型别名（与 gb28181-go 保持一致）。
type CatalogItem = manscdp.CatalogItem
type RecordItem = manscdp.RecordItem
type AlarmNotify = manscdp.AlarmNotify
type MobilePositionNotify = manscdp.MobilePositionNotify
type BasicParam = manscdp.BasicParam
type HomePosition = manscdp.HomePosition
type CruiseTrack = manscdp.CruiseTrack
type CruiseTrackList = manscdp.CruiseTrackList
type PTZPosition = manscdp.PTZPosition
type SDCardStatus = manscdp.SDCardStatus

// 会话类型与错误。
type SessionType = session.SessionType

const (
	SessionPlay     = session.SessionPlay
	SessionPlayback = session.SessionPlayback
	SessionDownload = session.SessionDownload
)

var (
	ErrRecordTimeout = session.ErrRecordTimeout
	ErrPresetTimeout = session.ErrPresetTimeout
	ErrQueryTimeout  = session.ErrQueryTimeout
)

// NormalizeStreamMode 返回 INVITE SDP 使用的 RTP 媒体传输方式；空/未知回退 UDP。
func NormalizeStreamMode(mode string) string {
	return transport.Normalize(mode)
}

// BuildPTZPositionQuery 构造 PTZ 精准位置查询/订阅消息体。
func BuildPTZPositionQuery(channelID, sn string) string {
	return manscdp.BuildPTZPositionQuery(channelID, sn)
}

// PlaySSRC 生成 GB28181 播放 SSRC（0 + domain[3:8] + 4 位序号）。
func PlaySSRC(domain string, seq int) string {
	return ssrc.Play(domain, seq)
}

// PlaybackSSRC 生成历史回放/下载 SSRC（首位 1，区别于实时点播）。
func PlaybackSSRC(domain string, seq int) string {
	return ssrc.Playback(domain, seq)
}

// SDP 构造。
func BuildPlaySDP(deviceID, sdpIP string, port int, ssrc, streamMode, streamIdentification string) string {
	return sdp.BuildPlay(deviceID, sdpIP, port, ssrc, streamMode, streamIdentification)
}

func BuildPlaybackSDP(deviceID, channelID, sdpIP string, port int, ssrc, streamMode, startTime, endTime string) string {
	return sdp.BuildPlayback(deviceID, channelID, sdpIP, port, ssrc, streamMode, startTime, endTime)
}

func BuildDownloadSDP(deviceID, channelID, sdpIP string, port int, ssrc, streamMode, startTime, endTime string, downloadSpeed int) string {
	return sdp.BuildDownload(deviceID, channelID, sdpIP, port, ssrc, streamMode, startTime, endTime, downloadSpeed)
}

// 回放控制（MANSRTSP）。
func BuildPlaybackPause(cseq int) string                { return mansrtsp.Pause(cseq) }
func BuildPlaybackResume(cseq int) string               { return mansrtsp.Resume(cseq) }
func BuildPlaybackSpeed(cseq int, speed float64) string { return mansrtsp.Speed(cseq, speed) }
func BuildPlaybackSeek(cseq int, seekTime int64) string { return mansrtsp.Seek(cseq, seekTime) }

// 订阅消息构造。
func BuildSubscribeCatalog(deviceID, sn string) string {
	return manscdp.BuildSubscribeCatalog(deviceID, sn)
}

func BuildSubscribeAlarm(deviceID, sn string) string {
	return manscdp.BuildSubscribeAlarm(deviceID, sn)
}

func BuildSubscribeMobilePosition(deviceID, sn string, interval int) string {
	return manscdp.BuildSubscribeMobilePosition(deviceID, sn, interval)
}

// GuessLocalIP 猜测本机对外 SIP IP。
func GuessLocalIP() string { return gbserver.GuessLocalIP() }
