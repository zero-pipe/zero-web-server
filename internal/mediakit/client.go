package mediakit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"zero-web-server/internal/infrastructure/config"
	applog "zero-web-server/pkg/log"
)

type Client struct {
	baseURL string
	secret  string
	http    *http.Client
}

func NewClient(cfg config.MediaConfig) *Client {
	return NewClientAddr(cfg.IP, cfg.HTTPPort, cfg.Secret)
}

func NewClientAddr(ip string, httpPort int, secret string) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%d", ip, httpPort),
		secret:  secret,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

type APIResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
	Port int             `json:"port"` // openRtpServer returns port at root on some media server versions
}

func (c *Client) GetMediaList(ctx context.Context) (*APIResponse, error) {
	return c.get(ctx, "media/list", nil)
}

func (c *Client) GetServerConfig(ctx context.Context) (*APIResponse, error) {
	return c.get(ctx, "system/config", nil)
}

func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.GetServerConfig(ctx)
	if err == nil && resp != nil && resp.Code == 0 {
		return nil
	}
	resp, err = c.get(ctx, "system/version", nil)
	if err != nil {
		return err
	}
	if resp == nil || resp.Code != 0 {
		msg := "unknown error"
		if resp != nil {
			msg = resp.Msg
		}
		return fmt.Errorf("media server ping failed: %s", msg)
	}
	return nil
}

func (c *Client) AddStreamProxy(ctx context.Context, vhost, app, stream, streamURL, rtpType string, enableHLS, enableMP4, autoClose bool) (*APIResponse, error) {
	params := url.Values{
		"vhost":          {vhost},
		"app":            {app},
		"stream":         {stream},
		"url":            {streamURL},
		"rtp_type":       {rtpType},
		"enable_hls":     {boolStr(enableHLS)},
		"enable_mp4":     {boolStr(enableMP4)},
		"enable_audio":   {"1"},
		"add_mute_audio": {"0"},
		"auto_close":     {boolStr(autoClose)},
	}
	return c.get(ctx, "proxy/add", params)
}

func (c *Client) DelStreamProxy(ctx context.Context, key string) (*APIResponse, error) {
	params := url.Values{"key": {key}}
	return c.get(ctx, "proxy/del", params)
}

func (c *Client) LoadMP4File(ctx context.Context, app, stream, filePath string) (*APIResponse, error) {
	params := url.Values{
		"vhost":       {"__defaultVhost__"},
		"app":         {app},
		"stream":      {stream},
		"file_path":   {filePath},
		"file_repeat": {"0"},
	}
	// ZMS webapi 目前只解析 URL query；与 openRtpServer 一样走 GET
	return c.get(ctx, "record/load", params)
}

func (c *Client) DeleteRecordFile(ctx context.Context, filePath string) (*APIResponse, error) {
	params := url.Values{
		"file_path": {filePath},
	}
	return c.get(ctx, "record/delete", params)
}

func (c *Client) SeekRecordStamp(ctx context.Context, app, stream string, stamp float64, schema string) (*APIResponse, error) {
	if schema == "" {
		schema = "ts"
	}
	params := url.Values{
		"vhost":  {"__defaultVhost__"},
		"app":    {app},
		"stream": {stream},
		"stamp":  {strconv.FormatFloat(stamp, 'f', -1, 64)},
		"schema": {schema},
	}
	return c.get(ctx, "record/seek", params)
}

func (c *Client) SetRecordSpeed(ctx context.Context, app, stream string, speed int, schema string) (*APIResponse, error) {
	if schema == "" {
		schema = "ts"
	}
	params := url.Values{
		"vhost":  {"__defaultVhost__"},
		"app":    {app},
		"stream": {stream},
		"speed":  {strconv.Itoa(speed)},
		"schema": {schema},
	}
	return c.get(ctx, "record/speed", params)
}

func (c *Client) CloseStreams(ctx context.Context, vhost, app, stream string) (*APIResponse, error) {
	params := url.Values{
		"vhost":  {vhost},
		"app":    {app},
		"stream": {stream},
		"force":  {"1"},
	}
	return c.get(ctx, "media/close_all", params)
}

type OpenRtpResult struct {
	Port int `json:"port"`
}

func (c *Client) OpenRtpServer(ctx context.Context, app, stream string, port, tcpMode int) (*OpenRtpResult, error) {
	params := url.Values{
		"app":       {app},
		"stream_id": {stream},
		"port":      {strconv.Itoa(port)},
		"tcp_mode":  {strconv.Itoa(tcpMode)},
	}
	resp, err := c.get(ctx, "rtp/open", params)
	if err != nil {
		applog.Warnf("[mediakit-api] openRtpServer HTTP error app=%s stream=%s url=%s err=%v",
			app, stream, c.baseURL+"/index/api/rtp/open", err)
		return nil, err
	}
	if resp.Code != 0 {
		applog.Warnf("[mediakit-api] openRtpServer code=%d msg=%s app=%s stream=%s", resp.Code, resp.Msg, app, stream)
		return nil, fmt.Errorf("openRtpServer failed: %s", resp.Msg)
	}
	var result OpenRtpResult
	if len(resp.Data) > 0 {
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			var alt struct {
				Port int `json:"port"`
			}
			if err2 := json.Unmarshal(resp.Data, &alt); err2 == nil && alt.Port > 0 {
				result.Port = alt.Port
			} else {
				return nil, fmt.Errorf("parse openRtpServer response: %w", err)
			}
		}
	}
	if result.Port == 0 && resp.Port > 0 {
		result.Port = resp.Port
	}
	if result.Port == 0 {
		return nil, fmt.Errorf("openRtpServer: missing port in response")
	}
	return &result, nil
}

func (c *Client) ConnectRtpServer(ctx context.Context, app, stream, host string, port int) error {
	params := url.Values{
		"app":       {app},
		"stream_id": {stream},
		"dst_url":   {host},
		"dst_port":  {strconv.Itoa(port)},
	}
	resp, err := c.post(ctx, "rtp/connect", params)
	if err != nil {
		applog.Warnf("[mediakit-api] connectRtpServer HTTP error app=%s stream=%s dst=%s:%d err=%v",
			app, stream, host, port, err)
		return err
	}
	if resp.Code != 0 {
		applog.Warnf("[mediakit-api] connectRtpServer code=%d msg=%s app=%s stream=%s dst=%s:%d",
			resp.Code, resp.Msg, app, stream, host, port)
		return fmt.Errorf("connectRtpServer failed: %s", resp.Msg)
	}
	applog.Debugf("[mediakit-api] connectRtpServer OK app=%s stream=%s -> %s:%d", app, stream, host, port)
	return nil
}

// SetAuthTokens 把当前启用中的接入 token 同步到 ZMS（本地验签用）。
// tokens 为空表示清除（ZMS 侧随之关闭推拉流鉴权）。
func (c *Client) SetAuthTokens(ctx context.Context, tokens []string) (*APIResponse, error) {
	params := url.Values{"tokens": {strings.Join(tokens, ",")}}
	return c.get(ctx, "auth/tokens", params)
}

func (c *Client) post(ctx context.Context, api string, params url.Values) (*APIResponse, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("secret", c.secret)
	reqURL := fmt.Sprintf("%s/index/api/%s", strings.TrimRight(c.baseURL, "/"), api)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result APIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse mediakit response: %w, body=%s", err, string(body))
	}
	return &result, nil
}

// StreamMediaInfo zero-media-server getMediaInfo 响应。
type StreamMediaInfo struct {
	Code          int     `json:"code"`
	Msg           string  `json:"msg"`
	Schema        string  `json:"schema"`
	App           string  `json:"app"`
	Stream        string  `json:"stream"`
	VideoCodec    string  `json:"videoCodec"`
	AudioCodec    string  `json:"audioCodec"`
	Video         bool    `json:"video"`
	Audio         bool    `json:"audio"`
	ReaderCount   int     `json:"readerCount"`
	AliveSecond   int64   `json:"aliveSecond"`
	BytesSpeed    int64   `json:"bytesSpeed"`
	VideoFps      uint    `json:"videoFps"` // 统计估算帧率
	Fps           float64 `json:"fps"`      // SPS/元数据帧率
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	SampleRate    int     `json:"sampleRate"`
	Channels      int     `json:"channels"`
	OriginType    int     `json:"originType"`
	OriginTypeStr string  `json:"originTypeStr"`
}

func (c *Client) GetMediaInfo(ctx context.Context, schema, app, stream string) (*StreamMediaInfo, error) {
	params := url.Values{
		"schema": {schema},
		"app":    {app},
		"stream": {stream},
	}
	params.Set("secret", c.secret)
	reqURL := fmt.Sprintf("%s/index/api/media/info?%s", strings.TrimRight(c.baseURL, "/"), params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var info StreamMediaInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parse getMediaInfo response: %w, body=%s", err, string(body))
	}
	if info.Code != 0 {
		return nil, fmt.Errorf("getMediaInfo failed: %s", info.Msg)
	}
	return &info, nil
}

// LookupStreamMediaInfo 按常见 schema 顺序查询在线流编码信息。
func (c *Client) LookupStreamMediaInfo(ctx context.Context, app, stream string) *StreamMediaInfo {
	for _, schema := range []string{"rtp-ps", "rtmp", "rtsp"} {
		info, err := c.GetMediaInfo(ctx, schema, app, stream)
		if err == nil && info != nil {
			return info
		}
	}
	return nil
}

func (c *Client) get(ctx context.Context, api string, params url.Values) (*APIResponse, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("secret", c.secret)

	reqURL := fmt.Sprintf("%s/index/api/%s?%s", strings.TrimRight(c.baseURL, "/"), api, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result APIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse mediakit response: %w, body=%s", err, string(body))
	}
	return &result, nil
}

func boolStr(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func BuildPlayURLs(mediaIP string, httpPort int, app, stream string) map[string]string {
	return BuildPlayURLsForBackend("zms", mediaIP, httpPort, app, stream)
}

func BuildPlayURLsForBackend(backend, mediaIP string, httpPort int, app, stream string) map[string]string {
	base := fmt.Sprintf("http://%s:%d", mediaIP, httpPort)
	if backend == "zms" || backend == "zeromediakit" || backend == "zeromediaserver" || backend == "zero-media-server" {
		return map[string]string{
			"flv":  fmt.Sprintf("%s/%s/%s.flv", base, app, stream),
			"hls":  fmt.Sprintf("%s/%s/%s.m3u8", base, app, stream),
			"ws":   fmt.Sprintf("ws://%s:%d/%s/%s.flv", mediaIP, httpPort, app, stream),
			"rtsp": fmt.Sprintf("rtsp://%s:8554/%s/%s", mediaIP, app, stream),
			"rtmp": fmt.Sprintf("rtmp://%s:1935/%s/%s", mediaIP, app, stream),
		}
	}
	return map[string]string{
		"flv":  fmt.Sprintf("%s/%s/%s.live.flv", base, app, stream),
		"hls":  fmt.Sprintf("%s/%s/%s/hls.m3u8", base, app, stream),
		"ws":   fmt.Sprintf("ws://%s:%d/%s/%s.live.flv", mediaIP, httpPort, app, stream),
		"rtsp": fmt.Sprintf("rtsp://%s:554/%s/%s", mediaIP, app, stream),
		"rtmp": fmt.Sprintf("rtmp://%s:1935/%s/%s", mediaIP, app, stream),
	}
}

func BuildPlayURLsFromConfig(cfg config.MediaConfig, app, stream string) map[string]string {
	return BuildPlayURLsForMedia(cfg, app, stream)
}

// BuildPlayURLsForMedia 依据节点上报的端口与协议能力拼装播放地址：
//   - 端口取节点实际上报值（RTSP/RTMP 未上报时回退 8554/1935）；
//   - 未上报能力的旧节点视为全开；
//   - 已上报且关闭的协议不生成对应 URL（如未开 HLS / RTSP 则不返回该键）。
func BuildPlayURLsForMedia(cfg config.MediaConfig, app, stream string) map[string]string {
	ip := cfg.IP
	httpPort := cfg.HTTPPort
	base := fmt.Sprintf("http://%s:%d", ip, httpPort)
	out := make(map[string]string, 5)
	if cfg.BackendType() == "zms" {
		out["flv"] = fmt.Sprintf("%s/%s/%s.flv", base, app, stream)
		out["ws"] = fmt.Sprintf("ws://%s:%d/%s/%s.flv", ip, httpPort, app, stream)
		if cfg.HLSAvailable() {
			out["hls"] = fmt.Sprintf("%s/%s/%s.m3u8", base, app, stream)
		}
		if cfg.RTSPAvailable() {
			out["rtsp"] = fmt.Sprintf("rtsp://%s:%d/%s/%s", ip, cfg.RTSPPlayPort(), app, stream)
		}
		if cfg.RTMPAvailable() {
			out["rtmp"] = fmt.Sprintf("rtmp://%s:%d/%s/%s", ip, cfg.RTMPPlayPort(), app, stream)
		}
		return out
	}
	// 非 zms 后端：保留历史路径形态，端口仍尽量用上报值。
	out["flv"] = fmt.Sprintf("%s/%s/%s.live.flv", base, app, stream)
	if cfg.HLSAvailable() {
		out["hls"] = fmt.Sprintf("%s/%s/%s/hls.m3u8", base, app, stream)
	}
	out["ws"] = fmt.Sprintf("ws://%s:%d/%s/%s.live.flv", ip, httpPort, app, stream)
	if cfg.RTSPAvailable() {
		rtspPort := cfg.RTSPPort
		if rtspPort <= 0 {
			rtspPort = 554
		}
		out["rtsp"] = fmt.Sprintf("rtsp://%s:%d/%s/%s", ip, rtspPort, app, stream)
	}
	if cfg.RTMPAvailable() {
		out["rtmp"] = fmt.Sprintf("rtmp://%s:%d/%s/%s", ip, cfg.RTMPPlayPort(), app, stream)
	}
	return out
}

// StreamPlayURLs 平台层播放地址（对齐 ZWS StreamContent 常用字段）。
type StreamPlayURLs struct {
	Flv, WsFlv, Hls, Rtmp, Rtsp, Rtc, Rtcs string
}

// BuildStreamPlayURLs 生成拉流 URL；webrtcPush=false 为点播播放；signalingPort>0 时 WebRTC 走平台代理。
// WebRTC 的 host 使用媒体节点 streamIp（cfg.IP），切勿配置成 127.0.0.1，否则浏览器 ICE/画面会异常。
func BuildStreamPlayURLs(cfg config.MediaConfig, app, stream string, webrtcPush bool, signalingPort int) StreamPlayURLs {
	urls := BuildPlayURLsFromConfig(cfg, app, stream)
	var rtc, rtcs string
	if cfg.WebRTCAvailable() {
		rtcBase := cfg.SignalingBaseURL(signalingPort)
		if config.IsLoopbackHost(cfg.IP) {
			applog.Warnf("[webrtc] media streamIp=%q is loopback; set 媒体节点「流IP」to LAN IP, and ZMS [webrtc] advertise_host", cfg.IP)
		}
		rtc, rtcs = BuildWebRTCURLs(rtcBase, app, stream, webrtcPush)
	}
	return StreamPlayURLs{
		Flv:   urls["flv"],
		WsFlv: urls["ws"],
		Hls:   urls["hls"],
		Rtmp:  urls["rtmp"],
		Rtsp:  urls["rtsp"],
		Rtc:   rtc,
		Rtcs:  rtcs,
	}
}

// BuildGB28181PushURL 国标收流地址摘要（UDP 或 TCP 被动 listen 端口）。
func BuildGB28181PushURL(sdpIP string, port int, tcpMode int) string {
	switch tcpMode {
	case 1:
		return fmt.Sprintf("tcp://%s:%d (RTP/PS RFC4571, camera -> zero-media-server listen)", sdpIP, port)
	case 2:
		return fmt.Sprintf("tcp://camera:port (RTP/PS RFC4571, zero-media-server -> camera after 200 OK)")
	default:
		return fmt.Sprintf("udp://%s:%d (RTP/PS, camera -> zero-media-server)", sdpIP, port)
	}
}

// LogPlayStreamPaths logs push/pull URLs at debug level for troubleshooting.
func LogPlayStreamPaths(tag, app, stream, pushURL string, urls map[string]string) {
	applog.Debugf("[%s] app=%s stream=%s", tag, app, stream)
	if pushURL != "" {
		applog.Debugf("[%s] push  %s", tag, pushURL)
	}
	if urls == nil {
		return
	}
	if v := urls["flv"]; v != "" {
		applog.Debugf("[%s] pull  http-flv : %s", tag, v)
	}
	if v := urls["ws"]; v != "" {
		applog.Debugf("[%s] pull  ws-flv  : %s", tag, v)
	}
	if v := urls["hls"]; v != "" {
		applog.Debugf("[%s] pull  hls     : %s", tag, v)
	}
	if v := urls["rtsp"]; v != "" {
		applog.Debugf("[%s] pull  rtsp    : %s", tag, v)
	}
	if v := urls["rtmp"]; v != "" {
		applog.Debugf("[%s] pull  rtmp    : %s", tag, v)
	}
	rtc, rtcs := BuildWebRTCURLs("", app, stream, false)
	if base := urls["flv"]; base != "" {
		if i := strings.Index(base, "://"); i > 0 {
			if j := strings.Index(base[i+3:], "/"); j >= 0 {
				rtcBase := base[:i+3+j]
				rtc, rtcs = BuildWebRTCURLs(rtcBase, app, stream, false)
			}
		}
	}
	if rtc != "" {
		applog.Debugf("[%s] pull  webrtc  : %s", tag, rtc)
	}
	if rtcs != "" && rtcs != rtc {
		applog.Debugf("[%s] pull  webrtcs : %s", tag, rtcs)
	}
}

func BuildWebRTCURLs(baseURL, app, stream string, push bool) (rtc, rtcs string) {
	typ := "play"
	if push {
		typ = "push"
	}
	q := fmt.Sprintf("app=%s&stream=%s&type=%s", app, stream, typ)
	rtc = fmt.Sprintf("%s/index/api/webrtc?%s", strings.TrimRight(baseURL, "/"), q)
	rtcs = strings.Replace(rtc, "http://", "https://", 1)
	return rtc, rtcs
}
