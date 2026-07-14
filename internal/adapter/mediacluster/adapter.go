package mediacluster

import (
	"context"
	"encoding/json"

	"zero-web-kit/internal/application/mediaserver"
	"zero-web-kit/internal/infrastructure/media/mediakit"
	"zero-web-kit/internal/port"
)

// Adapter 将现有 mediaserver.Service 暴露为 port.MediaCluster。
type Adapter struct {
	svc *mediaserver.Service
}

func New(svc *mediaserver.Service) *Adapter {
	return &Adapter{svc: svc}
}

func (a *Adapter) List(ctx context.Context) ([]port.MediaNode, error) {
	_ = ctx
	views, err := a.svc.List()
	if err != nil {
		return nil, err
	}
	out := make([]port.MediaNode, 0, len(views))
	for _, v := range views {
		out = append(out, port.MediaNode{
			ID: v.ID, IP: v.IP, HTTPPort: v.HTTPPort, Secret: v.Secret,
			Type: v.Type, Online: v.Online, Load: v.Load,
			SDPIP: v.SDPIP, StreamIP: v.StreamIP,
		})
	}
	return out, nil
}

func (a *Adapter) Lookup(ctx context.Context, id string) (port.MediaEndpoint, error) {
	_ = ctx
	n, err := a.svc.Lookup(id)
	if err != nil {
		return nil, err
	}
	return wrap(n), nil
}

func (a *Adapter) Resolve(ctx context.Context, preferID string) (port.MediaEndpoint, error) {
	_ = ctx
	n, err := a.svc.Resolve(preferID)
	if err != nil {
		return nil, err
	}
	return wrap(n), nil
}

func (a *Adapter) ResolveForStream(ctx context.Context, app, stream, preferID string) (port.MediaEndpoint, error) {
	_ = ctx
	n, err := a.svc.ResolveForStream(app, stream, preferID)
	if err != nil {
		return nil, err
	}
	return wrap(n), nil
}

func (a *Adapter) SelectMinimumLoad(ctx context.Context) (port.MediaEndpoint, error) {
	_ = ctx
	n, err := a.svc.SelectMinimumLoad()
	if err != nil {
		return nil, err
	}
	return wrap(n), nil
}

func (a *Adapter) BindStream(nodeID, app, stream string) {
	// mediaserver.Service 签名为 (app, stream, nodeID)
	a.svc.BindStream(app, stream, nodeID)
}

func (a *Adapter) UnbindStream(app, stream string) {
	a.svc.UnbindStream(app, stream)
}

func (a *Adapter) StreamNodeID(app, stream string) (string, bool) {
	return a.svc.StreamNodeID(app, stream)
}

type endpoint struct {
	node *mediaserver.Node
}

func wrap(n *mediaserver.Node) port.MediaEndpoint {
	if n == nil {
		return nil
	}
	return &endpoint{node: n}
}

func (e *endpoint) ID() string       { return e.node.ID() }
func (e *endpoint) SDPIP() string    { return e.node.SDPIP() }
func (e *endpoint) StreamIP() string { return e.node.StreamIP() }
func (e *endpoint) Secret() string   { return e.node.Model.Secret }

func (e *endpoint) BaseURL() string {
	return e.node.MediaConfig().BaseURL()
}

func (e *endpoint) SignalingBaseURL(serverPort int) string {
	return e.node.MediaConfig().SignalingBaseURL(serverPort)
}

func (e *endpoint) PlayURLs(app, stream string) map[string]string {
	return mediakit.BuildPlayURLsFromConfig(e.node.MediaConfig(), app, stream)
}

func (e *endpoint) StreamPlayURLs(app, stream string, webrtcPush bool, signalingPort int) port.StreamPlayURLSet {
	u := mediakit.BuildStreamPlayURLs(e.node.MediaConfig(), app, stream, webrtcPush, signalingPort)
	return port.StreamPlayURLSet{
		Flv: u.Flv, WsFlv: u.WsFlv, Hls: u.Hls, Rtmp: u.Rtmp,
		Rtsp: u.Rtsp, Rtc: u.Rtc, Rtcs: u.Rtcs,
	}
}

func (e *endpoint) OpenRtpServer(ctx context.Context, app, stream string, port, tcpMode int) (int, error) {
	resp, err := e.node.Client.OpenRtpServer(ctx, app, stream, port, tcpMode)
	if err != nil {
		return 0, err
	}
	return resp.Port, nil
}

func (e *endpoint) ConnectRtpServer(ctx context.Context, app, stream, host string, port int) error {
	return e.node.Client.ConnectRtpServer(ctx, app, stream, host, port)
}

func (e *endpoint) CloseStreams(ctx context.Context, vhost, app, stream string) error {
	_, err := e.node.Client.CloseStreams(ctx, vhost, app, stream)
	return err
}

func (e *endpoint) LookupStream(ctx context.Context, app, stream string) *port.StreamProbe {
	info := e.node.Client.LookupStreamMediaInfo(ctx, app, stream)
	if info == nil {
		return nil
	}
	return &port.StreamProbe{
		VideoCodec: info.VideoCodec, AudioCodec: info.AudioCodec,
		Video: info.Video, Audio: info.Audio, ReaderCount: info.ReaderCount,
		AliveSecond: info.AliveSecond, BytesSpeed: info.BytesSpeed,
		VideoFps: info.VideoFps, Width: info.Width, Height: info.Height,
	}
}

func (e *endpoint) AddStreamProxy(ctx context.Context, vhost, app, stream, streamURL, rtpType string, enableHLS, enableMP4, autoClose bool) (*port.ProxyResult, error) {
	resp, err := e.node.Client.AddStreamProxy(ctx, vhost, app, stream, streamURL, rtpType, enableHLS, enableMP4, autoClose)
	if err != nil {
		return nil, err
	}
	out := &port.ProxyResult{Code: resp.Code, Msg: resp.Msg}
	if len(resp.Data) > 0 {
		var proxyData struct {
			Key string `json:"key"`
		}
		if json.Unmarshal(resp.Data, &proxyData) == nil {
			out.Key = proxyData.Key
		}
	}
	return out, nil
}

func (e *endpoint) DelStreamProxy(ctx context.Context, key string) error {
	_, err := e.node.Client.DelStreamProxy(ctx, key)
	return err
}

func (e *endpoint) DeleteRecordFile(ctx context.Context, filePath string) error {
	_, err := e.node.Client.DeleteRecordFile(ctx, filePath)
	return err
}

func (e *endpoint) LoadMP4File(ctx context.Context, app, stream, filePath string) error {
	_, err := e.node.Client.LoadMP4File(ctx, app, stream, filePath)
	return err
}

var _ port.MediaCluster = (*Adapter)(nil)
var _ port.MediaEndpoint = (*endpoint)(nil)
