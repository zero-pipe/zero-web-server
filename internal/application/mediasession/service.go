// Package mediasession 提供"媒体会话"只读运维视图：合并各在线媒体节点的实时流，
// 用媒体流名 live/<设备内码>_<通道内码> 反查设备/通道，标注取流来源（GB推流/ONVIF拉流/插件/手动）。
// 流是"点播"的产物，本服务只观测/停止，不负责建立（点播走 play/start）。
package mediasession

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	mediaserverapp "zero-web-server/internal/application/mediaserver"
	domainasset "zero-web-server/internal/domain/asset"
	"zero-web-server/internal/mediakit"
	"zero-web-server/internal/port"
)

// LiveStopper 设备侧断开能力（由 asset.Service 实现）：
// GB28181 发 BYE、ONVIF 停拉流、插件停推流；内部通常也会关闭该路 ZMS 流。
type LiveStopper interface {
	StopLive(ctx context.Context, resourceID string) error
}

type Service struct {
	nodes      *mediaserverapp.Service
	assets     domainasset.Repository
	stopper    LiveStopper
	cluster    port.MediaCluster
	serverPort int
}

func NewService(nodes *mediaserverapp.Service, assets domainasset.Repository, stopper LiveStopper, cluster port.MediaCluster, serverPort int) *Service {
	return &Service{nodes: nodes, assets: assets, stopper: stopper, cluster: cluster, serverPort: serverPort}
}

// StreamAddress 一路流的直链地址（协议 + 人读标签 + URL）。
type StreamAddress struct {
	Protocol string `json:"protocol"`
	Label    string `json:"label"`
	URL      string `json:"url"`
}

// View 是一路媒体会话的展示视图。
type View struct {
	App                 string          `json:"app"`
	Stream              string          `json:"stream"`
	ResourceID          string          `json:"resourceId"`
	DeviceInternalCode  string          `json:"deviceInternalCode"`
	ChannelInternalCode string          `json:"channelInternalCode"`
	DeviceName          string          `json:"deviceName"`
	ChannelName         string          `json:"channelName"`
	GBDeviceCode        string          `json:"gbDeviceCode"`  // 设备国标编号（展示用，不参与路由）
	GBChannelCode       string          `json:"gbChannelCode"` // 通道国标编号（展示用，不参与路由）
	PTZType             int             `json:"ptzType"`
	PTZTypeText         string          `json:"ptzTypeText"`
	HasAudio            bool            `json:"hasAudio"`
	SourceType          string          `json:"sourceType"`  // gb28181 | onvif | plugin | manual
	SourceLabel         string          `json:"sourceLabel"` // 人读来源
	MediaServerID       string          `json:"mediaServerId"`
	Video               bool            `json:"video"`
	Audio               bool            `json:"audio"`
	VideoCodec          string          `json:"videoCodec"`
	AudioCodec          string          `json:"audioCodec"`
	Width               int             `json:"width"`
	Height              int             `json:"height"`
	VideoFps            uint            `json:"videoFps"`
	BytesSpeed          int64           `json:"bytesSpeed"`
	ReaderCount         int             `json:"readerCount"`
	AliveSecond         int64           `json:"aliveSecond"`
	OriginTypeStr       string          `json:"originTypeStr"`
	Addresses           []StreamAddress `json:"addresses"`
}

// mediaItem 对应 ZMS media/list 的单条流（字段见 media_json.c）。
type mediaItem struct {
	App              string  `json:"app"`
	Stream           string  `json:"stream"`
	Schema           string  `json:"schema"`
	Video            bool    `json:"video"`
	Audio            bool    `json:"audio"`
	VideoCodec       string  `json:"videoCodec"`
	AudioCodec       string  `json:"audioCodec"`
	VideoFps         uint    `json:"videoFps"`
	Fps              float64 `json:"fps"`
	Width            int     `json:"width"`
	Height           int     `json:"height"`
	ReaderCount      int     `json:"readerCount"`
	TotalReaderCount int     `json:"totalReaderCount"`
	AliveSecond      int64   `json:"aliveSecond"`
	BytesSpeed       int64   `json:"bytesSpeed"`
	OriginType       int     `json:"originType"`
	OriginTypeStr    string  `json:"originTypeStr"`
	OriginURL        string  `json:"originUrl"`
}

// List 汇总所有在线节点的实时流。
func (s *Service) List(ctx context.Context) ([]*View, error) {
	if s.nodes == nil {
		return nil, fmt.Errorf("媒体节点服务未初始化")
	}
	onlineNodes, err := s.nodes.ListOnline()
	if err != nil {
		return nil, err
	}
	var out []*View
	for _, node := range onlineNodes {
		client, cerr := s.nodes.ClientForID(node.ID)
		if cerr != nil {
			continue
		}
		resp, gerr := client.GetMediaList(ctx)
		if gerr != nil || resp == nil || resp.Code != 0 || len(resp.Data) == 0 {
			continue
		}
		var items []mediaItem
		if json.Unmarshal(resp.Data, &items) != nil {
			continue
		}
		for i := range items {
			out = append(out, s.toView(ctx, node.ID, &items[i]))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].MediaServerID != out[j].MediaServerID {
			return out[i].MediaServerID < out[j].MediaServerID
		}
		return out[i].Stream < out[j].Stream
	})
	return out, nil
}

// GBReceivingResult 国标收流列表查询结果。
type GBReceivingResult struct {
	List  []*View `json:"list"`
	Total int     `json:"total"`
}

// GBReceivingChannels 返回 ZMS 上正在收流的国标通道（按流名反查资源）。
// 只做查询，不建立流；provider 只区分来源，播放仍走 /resources/{id}/live/start。
func (s *Service) GBReceivingChannels(ctx context.Context, page, count int, query string) (*GBReceivingResult, error) {
	all, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*View, 0)
	q := strings.ToLower(strings.TrimSpace(query))
	for _, v := range all {
		if v.SourceType != "gb28181" || v.ResourceID == "" {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(v.ChannelName), q) &&
			!strings.Contains(strings.ToLower(v.DeviceName), q) &&
			!strings.Contains(strings.ToLower(v.GBChannelCode), q) &&
			!strings.Contains(strings.ToLower(v.GBDeviceCode), q) {
			continue
		}
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Stream < out[j].Stream })
	total := len(out)
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	start := (page - 1) * count
	if start >= total {
		return &GBReceivingResult{List: []*View{}, Total: total}, nil
	}
	end := start + count
	if end > total {
		end = total
	}
	return &GBReceivingResult{List: out[start:end], Total: total}, nil
}

func (s *Service) toView(ctx context.Context, nodeID string, it *mediaItem) *View {
	readers := it.TotalReaderCount
	if readers == 0 {
		readers = it.ReaderCount
	}
	v := &View{
		App: it.App, Stream: it.Stream, MediaServerID: nodeID,
		Video: it.Video, Audio: it.Audio,
		VideoCodec: it.VideoCodec, AudioCodec: it.AudioCodec,
		Width: it.Width, Height: it.Height, VideoFps: it.VideoFps,
		BytesSpeed: it.BytesSpeed, ReaderCount: readers, AliveSecond: it.AliveSecond,
		OriginTypeStr: it.OriginTypeStr,
	}
	s.enrich(v)
	v.Addresses = s.addresses(ctx, nodeID, v.App, v.Stream)
	return v
}

// addresses 复用点播同源的 URL 合成（协议直链），供「地址/分享」入口展示。
func (s *Service) addresses(ctx context.Context, nodeID, app, stream string) []StreamAddress {
	if s.cluster == nil {
		return nil
	}
	ep, err := s.cluster.ResolveForStream(ctx, app, stream, nodeID)
	if err != nil || ep == nil {
		return nil
	}
	u := ep.StreamPlayURLs(app, stream, false, s.serverPort)
	defs := []struct{ protocol, label, url string }{
		{"flv", "FLV", u.Flv},
		{"ws_flv", "FLV (WS)", u.WsFlv},
		{"hls", "HLS", u.Hls},
		{"rtmp", "RTMP", u.Rtmp},
		{"rtsp", "RTSP", u.Rtsp},
		{"rtc", "WebRTC", u.Rtc},
		{"rtcs", "WebRTC (TLS)", u.Rtcs},
	}
	var out []StreamAddress
	for _, d := range defs {
		if d.url != "" {
			out = append(out, StreamAddress{Protocol: d.protocol, Label: d.label, URL: d.url})
		}
	}
	return out
}

// enrich 从流名 live/<D>_<C> 反查设备/通道并判定来源。
func (s *Service) enrich(v *View) {
	v.SourceType = "manual"
	v.SourceLabel = "手动代理/其他"
	if v.App != "live" || s.assets == nil {
		return
	}
	devCode, chCode := splitStream(v.Stream)
	if devCode == "" {
		return
	}
	v.DeviceInternalCode = devCode
	v.ChannelInternalCode = chCode
	dev, err := s.assets.GetDeviceByInternalCode(devCode)
	if err != nil || dev == nil {
		return
	}
	v.DeviceName = dev.Name
	if chCode != "" {
		if res, rerr := s.assets.GetResourceByInternalCode(chCode); rerr == nil && res != nil {
			v.ResourceID = res.ID
			v.ChannelName = res.Name
			v.PTZType = res.PTZType
			v.PTZTypeText = ptzTypeText(res.PTZType)
			v.HasAudio = res.HasAudio
			v.GBChannelCode = res.ExternalKey
		}
	}
	if binding, berr := s.assets.GetBindingByDevice(dev.ID); berr == nil && binding != nil {
		v.GBDeviceCode = binding.ExternalID
		switch binding.Protocol {
		case "gb28181":
			v.SourceType, v.SourceLabel = "gb28181", "GB28181 · RTP/PS 推流"
		case "onvif":
			v.SourceType, v.SourceLabel = "onvif", "ONVIF · RTSP 拉流"
		case "plugin":
			v.SourceType, v.SourceLabel = "plugin", "插件 · "+pluginTransportLabel(v.OriginTypeStr)
		default:
			v.SourceType, v.SourceLabel = binding.Protocol, binding.Protocol
		}
	}
}

// splitStream 解析 live 流名 <设备内码>_<通道内码>。
func splitStream(stream string) (devCode, chCode string) {
	stream = strings.TrimSuffix(stream, "_playback")
	stream = strings.TrimSuffix(stream, "_download")
	parts := strings.SplitN(stream, "_", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

func ptzTypeText(t int) string {
	texts := map[int]string{
		0: "未知", 1: "球机", 2: "半球", 3: "固定枪机",
		4: "遥控枪机", 5: "遥控半球", 6: "全景通道", 7: "分割通道",
	}
	if s := texts[t]; s != "" {
		return s
	}
	return texts[0]
}

func pluginTransportLabel(origin string) string {
	switch {
	case strings.Contains(origin, "rtp"):
		return "RTP 推流"
	case strings.Contains(origin, "rtsp"):
		return "RTSP 拉流"
	case strings.Contains(origin, "rtmp"):
		return "RTMP"
	case origin != "":
		return origin
	default:
		return "取流中"
	}
}

// Stop 关闭一路流。设备绑定的实时流（live/<D>_<C> 且能反查到资源）走"设备感知断开"：
// GB28181 发 BYE、ONVIF 停拉流、插件停推流——这些路径通常已同时关闭 ZMS 流；
// 手动代理/无法反查的流只做节点级 close_stream。
func (s *Service) Stop(ctx context.Context, nodeID, app, stream string) error {
	if nodeID == "" || app == "" || stream == "" {
		return fmt.Errorf("参数不完整")
	}
	client, err := s.nodes.ClientForID(nodeID)
	if err != nil {
		return fmt.Errorf("媒体节点不存在")
	}

	// 1) 设备感知断开
	if app == "live" && s.assets != nil && s.stopper != nil {
		_, chCode := splitStream(stream)
		if chCode != "" {
			if res, rerr := s.assets.GetResourceByInternalCode(chCode); rerr == nil && res != nil {
				if terr := s.stopper.StopLive(ctx, res.ID); terr != nil {
					// 设备断开失败：仍尽力关闭节点流，避免残留，并把原因带回。
					_, _ = client.CloseStreams(ctx, "__defaultVhost__", app, stream)
					return fmt.Errorf("已尝试关流，但断开设备失败: %w", terr)
				}
				// 设备侧已断开，节点流通常随之结束；兜底关一次并忽略"已不存在"。
				_, _ = client.CloseStreams(ctx, "__defaultVhost__", app, stream)
				return nil
			}
		}
	}

	// 2) 手动代理 / 无法反查资源：直接关节点流
	if _, err := client.CloseStreams(ctx, "__defaultVhost__", app, stream); err != nil {
		return fmt.Errorf("关闭流失败: %w", err)
	}
	return nil
}

// AddProxy 临时拉流代理（保留的手动运维能力）：把外部 URL 拉进 ZMS。
func (s *Service) AddProxy(ctx context.Context, nodeID, app, stream, srcURL string) error {
	if strings.TrimSpace(srcURL) == "" {
		return fmt.Errorf("拉流地址不能为空")
	}
	if app == "" {
		app = "proxy"
	}
	if stream == "" {
		stream = fmt.Sprintf("proxy_%d", stableSuffix())
	}
	var client *mediakit.Client
	if nodeID != "" {
		c, err := s.nodes.ClientForID(nodeID)
		if err != nil {
			return fmt.Errorf("媒体节点不存在")
		}
		client = c
	} else {
		nodes, err := s.nodes.ListOnline()
		if err != nil || len(nodes) == 0 {
			return fmt.Errorf("无在线媒体节点")
		}
		c, err := s.nodes.ClientForID(nodes[0].ID)
		if err != nil {
			return err
		}
		client = c
	}
	if _, err := client.AddStreamProxy(ctx, "__defaultVhost__", app, stream, srcURL, "0", false, false, true); err != nil {
		return fmt.Errorf("创建拉流代理失败: %w", err)
	}
	return nil
}

var suffixCounter int64

func stableSuffix() int64 {
	suffixCounter++
	return suffixCounter
}
