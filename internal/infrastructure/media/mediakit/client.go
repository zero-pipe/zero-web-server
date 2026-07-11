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

	"zero-web-kit/internal/infrastructure/config"
	applog "zero-web-kit/pkg/log"
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
	return c.get(ctx, "getMediaList", nil)
}

func (c *Client) GetServerConfig(ctx context.Context) (*APIResponse, error) {
	return c.get(ctx, "getServerConfig", nil)
}

func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.GetServerConfig(ctx)
	if err == nil && resp != nil && resp.Code == 0 {
		return nil
	}
	resp, err = c.get(ctx, "version", nil)
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
	return c.get(ctx, "addStreamProxy", params)
}

func (c *Client) DelStreamProxy(ctx context.Context, key string) (*APIResponse, error) {
	params := url.Values{"key": {key}}
	return c.get(ctx, "delStreamProxy", params)
}

func (c *Client) LoadMP4File(ctx context.Context, app, stream, filePath string) (*APIResponse, error) {
	params := url.Values{
		"vhost":       {"__defaultVhost__"},
		"app":         {app},
		"stream":      {stream},
		"file_path":   {filePath},
		"file_repeat": {"0"},
	}
	return c.post(ctx, "loadMP4File", params)
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
	return c.post(ctx, "seekRecordStamp", params)
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
	return c.post(ctx, "setRecordSpeed", params)
}

func (c *Client) CloseStreams(ctx context.Context, vhost, app, stream string) (*APIResponse, error) {
	params := url.Values{
		"vhost":  {vhost},
		"app":    {app},
		"stream": {stream},
		"force":  {"1"},
	}
	return c.get(ctx, "close_streams", params)
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
	resp, err := c.get(ctx, "openRtpServer", params)
	if err != nil {
		applog.Warnf("[mediakit-api] openRtpServer HTTP error app=%s stream=%s url=%s err=%v",
			app, stream, c.baseURL+"/index/api/openRtpServer", err)
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
	resp, err := c.post(ctx, "connectRtpServer", params)
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
	reqURL := fmt.Sprintf("%s/index/api/getMediaInfo?%s", strings.TrimRight(c.baseURL, "/"), params.Encode())
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
	return BuildPlayURLsForBackend(cfg.BackendType(), cfg.IP, cfg.HTTPPort, app, stream)
}

// StreamPlayURLs 平台层播放地址（对齐 ZWS StreamContent 常用字段）。
type StreamPlayURLs struct {
	Flv, WsFlv, Hls, Rtmp, Rtsp, Rtc, Rtcs string
}

// BuildStreamPlayURLs 生成拉流 URL；webrtcPush=false 为点播播放；signalingPort>0 时 WebRTC 走平台代理。
func BuildStreamPlayURLs(cfg config.MediaConfig, app, stream string, webrtcPush bool, signalingPort int) StreamPlayURLs {
	urls := BuildPlayURLsFromConfig(cfg, app, stream)
	rtc, rtcs := BuildWebRTCURLs(cfg.SignalingBaseURL(signalingPort), app, stream, webrtcPush)
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
