package sipinfra

import (
	domainchannel "zero-web-server/internal/domain/channel"
	domaindevice "zero-web-server/internal/domain/device"
	"zero-web-server/internal/port"
)

// SessionAdapter 把 *Server 适配成 port.GBSession，供应用层依赖接口而非具体实现。
// 除 SendInvitePlay（去掉 sipgo 的 onOK 回调参数）与 InviteManager（返回端口视图）外，
// 其余方法由内嵌的 *Server 直接提升满足。
type SessionAdapter struct {
	*Server
}

// NewSessionAdapter 构造会话端口适配器。
func NewSessionAdapter(s *Server) SessionAdapter { return SessionAdapter{Server: s} }

func (a SessionAdapter) SendInvitePlay(device *domaindevice.Device, channel *domainchannel.Channel, sdpBody, ssrc, stream, streamMode string, tcpConnect func(host string, port int) error) error {
	return a.Server.SendInvitePlay(device, channel, sdpBody, ssrc, stream, streamMode, tcpConnect, nil)
}

func (a SessionAdapter) InviteManager() port.GBInviteManager {
	return inviteManagerPort{m: a.Server.InviteManager()}
}

type inviteManagerPort struct{ m *InviteManager }

func (p inviteManagerPort) Get(stream string) (port.GBInviteSession, bool) {
	sess, ok := p.m.Get(stream)
	if !ok || sess == nil {
		return nil, false
	}
	return inviteSessionPort{s: sess}, true
}

type inviteSessionPort struct{ s *InviteSession }

func (p inviteSessionPort) Progress() float64  { return p.s.Progress() }
func (p inviteSessionPort) DownloadSpeed() int { return p.s.DownloadSpeed }

var _ port.GBSession = SessionAdapter{}
