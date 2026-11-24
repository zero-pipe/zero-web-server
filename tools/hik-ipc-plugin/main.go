//go:build hik && cgo

// Command hik-ipc-plugin is a Hikvision HCNetSDK sidecar.
//
// It uses HCNetSDK for login / device info / channel enumeration / PTZ, and
// returns a Hikvision RTSP URL as the live stream source (ZMS pulls it).
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var configSchema = `{
  "type": "object",
  "required": ["ip", "port", "username", "password"],
  "properties": {
    "ip":       { "type": "string",  "title": "设备 IP" },
    "port":     { "type": "integer", "title": "SDK 端口", "default": 8000 },
    "username": { "type": "string",  "title": "用户名" },
    "password": { "type": "string",  "title": "密码", "format": "password" },
    "channel":  { "type": "integer", "title": "默认通道", "default": 1 }
  }
}`

type config struct {
	listen        string
	rtspPort      int
	rtspTmpl      string
	ptzType       int
	hasAudio      bool
	isapiPort     int
	isapiHTTPS    bool
	isapiInsecure bool
}

var hikConfig config

type session struct {
	UserID   int64
	IP       string
	SDKPort  int
	User     string
	Pass     string
	Info     *DeviceInfo
	Channels map[string]int // display channel ID -> SDK channel number
	Order    []string

	AlarmHandle    int64
	Armed          bool
	SubscriptionID string
	InternalCode   string
	EventTypes     []string
	Pipeline       *pipeline
}

var (
	sdk       HikSDK
	sessionMu sync.Mutex
	sessions  = map[string]*session{}
	// lastPTZ 记住每个 设备|通道 最后一次下发的云台动作码，
	// 因为海康的"停止"必须复用同一个动作码再发一次 stop=1。
	lastPTZ sync.Map
)

func main() {
	cfg := config{
		listen:        env("PLUGIN_LISTEN", ":19091"),
		rtspPort:      envInt("HIK_RTSP_PORT", 554),
		rtspTmpl:      env("HIK_RTSP_TEMPLATE", ""),
		ptzType:       envInt("HIK_PTZ_TYPE", 3),
		hasAudio:      env("HIK_HAS_AUDIO", "false") == "true",
		isapiPort:     envInt("HIK_ISAPI_PORT", 80),
		isapiHTTPS:    env("HIK_ISAPI_HTTPS", "false") == "true",
		isapiInsecure: env("HIK_ISAPI_INSECURE", "true") == "true",
	}
	hikConfig = cfg
	if err := sdk.Init(); err != nil {
		log.Fatalf("HCNetSDK init: %v", err)
	}
	defer sdk.Cleanup()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", okHandler)
	mux.HandleFunc("/v1/login", handleLogin)
	mux.HandleFunc("/v1/logout", handleLogout)
	mux.HandleFunc("/v1/channels", handleChannels(cfg))
	mux.HandleFunc("/v1/live/start", handleLiveStart(cfg))
	mux.HandleFunc("/v1/live/stop", okHandler)
	mux.HandleFunc("/v1/ptz", handlePTZ)
	mux.HandleFunc("/v1/arm", handleArm)
	mux.HandleFunc("/v1/disarm", handleDisarm)
	mux.HandleFunc("/v1/record/query", handleRecordQuery)
	mux.HandleFunc("/v1/playback/start", handlePlaybackStart(cfg))
	mux.HandleFunc("/v1/playback/stop", handlePlaybackStop)
	mux.HandleFunc("/v1/playback/control", handlePlaybackControl)
	mux.HandleFunc("/v1/command", handleCommand(cfg))
	mux.HandleFunc("/v1/snapshot", handleSnapshot)

	ln, err := net.Listen("tcp", cfg.listen)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	// 先绑定监听，Agent 健康检查通过后再认为插件就绪。
	go func() {
		log.Printf("hik plugin listening on %s", cfg.listen)
		if err := http.Serve(ln, mux); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	select {}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceUID string         `json:"device_uid"`
		Params    map[string]any `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, "INVALID_ARGUMENT", err.Error())
		return
	}
	ip := paramString(req.Params, "ip")
	user := paramString(req.Params, "username")
	pass := paramString(req.Params, "password")
	port := paramInt(req.Params, "port", 8000)
	if ip == "" || user == "" {
		writeErr(w, "INVALID_ARGUMENT", "missing ip/username")
		return
	}
	uid, info, err := sdk.Login(ip, port, user, pass)
	if err != nil {
		writeErr(w, "AUTH_FAILED", err.Error())
		return
	}
	s := &session{
		UserID: uid, IP: ip, SDKPort: port, User: user, Pass: pass, Info: info,
		Channels: map[string]int{},
	}
	s.loadChannels()
	sessionMu.Lock()
	old := sessions[req.DeviceUID]
	sessions[req.DeviceUID] = s
	sessionMu.Unlock()
	if old != nil {
		if old.Armed && old.AlarmHandle >= 0 {
			_ = sdk.CloseAlarm(old.AlarmHandle)
		}
		old.Pipeline.stop()
		sdk.Logout(old.UserID)
	}
	log.Printf("hik login device=%s ip=%s uid=%d channels=%d serial=%s model=%s",
		req.DeviceUID, ip, uid, len(s.Order), info.Serial, info.Model)
	writeOK(w, map[string]any{
		"manufacturer": "Hikvision",
		"model":        firstNonEmpty(info.Model, "IPC"),
		"firmware":     "",
		"serialNumber": firstNonEmpty(info.Serial, req.DeviceUID),
	})
}

func (s *session) loadChannels() {
	ids := sdk.ListChannels(s.UserID, s.Info.StartChan, 64)
	if len(ids) == 0 && s.Info.AnalogChans > 0 {
		for i := 1; i <= s.Info.AnalogChans; i++ {
			ids = append(ids, i)
		}
	}
	if len(ids) == 0 {
		ids = []int{1}
	}
	for i, ch := range ids {
		display := strconv.Itoa(i + 1)
		s.Channels[display] = ch
		s.Order = append(s.Order, display)
	}
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceUID string `json:"device_uid"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	sessionMu.Lock()
	if s, ok := sessions[req.DeviceUID]; ok {
		if s.Armed && s.AlarmHandle >= 0 {
			_ = sdk.CloseAlarm(s.AlarmHandle)
		}
		s.Pipeline.stop()
		sdk.Logout(s.UserID)
		delete(sessions, req.DeviceUID)
	}
	sessionMu.Unlock()
	writeOK(w, nil)
}

func handleChannels(cfg config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			DeviceUID string `json:"device_uid"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		sessionMu.Lock()
		s := sessions[req.DeviceUID]
		sessionMu.Unlock()
		if s == nil {
			writeErr(w, "DEVICE_UNREACHABLE", "device not logged in")
			return
		}
		list := make([]map[string]any, 0, len(s.Order))
		for _, id := range s.Order {
			list = append(list, map[string]any{
				"channelId": id,
				"name":      fmt.Sprintf("通道 %s", id),
				"status":    "ON",
				"ptzType":   cfg.ptzType,
				"hasAudio":  cfg.hasAudio,
			})
		}
		writeOK(w, map[string]any{"channels": list})
	}
}

func handleLiveStart(cfg config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			DeviceUID string `json:"device_uid"`
			ChannelID string `json:"channel_id"`
			Stream    string `json:"stream"` // main（默认）| sub
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		sessionMu.Lock()
		s := sessions[req.DeviceUID]
		sessionMu.Unlock()
		if s == nil {
			writeErr(w, "DEVICE_UNREACHABLE", "device not logged in")
			return
		}
		display := 1
		if n, err := strconv.Atoi(req.ChannelID); err == nil && n > 0 {
			display = n
		}
		rtsp := buildRTSPURL(cfg, s, display, streamNumber(req.Stream))
		writeOK(w, map[string]any{"transport": "rtsp_url", "rtspUrl": rtsp})
	}
}

func handleArm(w http.ResponseWriter, r *http.Request) {
	var req armRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, "INVALID_ARGUMENT", err.Error())
		return
	}
	if req.DeviceUID == "" || req.DeviceInternalCode == "" {
		writeErr(w, "INVALID_ARGUMENT", "missing device_uid/device_internal_code")
		return
	}
	sessionMu.Lock()
	s := sessions[req.DeviceUID]
	sessionMu.Unlock()
	if s == nil {
		writeErr(w, "DEVICE_UNREACHABLE", "device not logged in")
		return
	}

	// Build and validate the whole event pipeline before touching the alarm channel.
	pipe, err := startPipeline(context.Background(), s, req)
	if err != nil {
		writeErr(w, "INTERNAL", err.Error())
		return
	}

	if s.Armed && s.AlarmHandle >= 0 {
		_ = sdk.CloseAlarm(s.AlarmHandle)
	}
	s.Pipeline.stop()
	handle, err := sdk.SetupAlarm(s.UserID)
	if err != nil {
		pipe.stop()
		writeErr(w, "INTERNAL", err.Error())
		return
	}
	subID := fmt.Sprintf("hik-%s-%d", req.DeviceUID, time.Now().UnixNano())
	sessionMu.Lock()
	s.AlarmHandle = handle
	s.Armed = true
	s.SubscriptionID = subID
	s.InternalCode = req.DeviceInternalCode
	s.EventTypes = req.EventTypes
	s.Pipeline = pipe
	sessionMu.Unlock()
	log.Printf("arm device=%s internal=%s events=%v subscription=%s alarmHandle=%d topic=%s",
		req.DeviceUID, req.DeviceInternalCode, req.EventTypes, subID, handle, req.MQTT.Topic)
	writeOK(w, map[string]any{"armed": true, "subscriptionId": subID})
}

func handleDisarm(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceUID      string `json:"device_uid"`
		SubscriptionID string `json:"subscription_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	sessionMu.Lock()
	s := sessions[req.DeviceUID]
	if s != nil && s.Armed {
		if s.AlarmHandle >= 0 {
			_ = sdk.CloseAlarm(s.AlarmHandle)
		}
		s.Pipeline.stop()
		s.Armed = false
		s.AlarmHandle = 0
		s.SubscriptionID = ""
		s.Pipeline = nil
	}
	sessionMu.Unlock()
	log.Printf("disarm device=%s subscription=%s", req.DeviceUID, req.SubscriptionID)
	writeOK(w, nil)
}

// recordQueryRequest 是录像检索请求。时间接受 Unix 秒、"2006-01-02 15:04:05" 或 RFC3339。
type recordQueryRequest struct {
	DeviceUID  string `json:"device_uid"`
	ChannelID  string `json:"channel_id"`
	StartTime  any    `json:"start_time"`
	EndTime    any    `json:"end_time"`
	StreamType int    `json:"stream_type"` // 0 任选(主>子>三) 1 子 2 三 3 主 254 双
}

// playbackStartRequest 是按时间回放请求。
type playbackStartRequest struct {
	DeviceUID string `json:"device_uid"`
	ChannelID string `json:"channel_id"`
	StartTime any    `json:"start_time"`
	EndTime   any    `json:"end_time"`
	Stream    string `json:"stream"` // main（默认）| sub
}

func handleRecordQuery(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeErr(w, "INVALID_ARGUMENT", err.Error())
		return
	}
	var req recordQueryRequest
	if err := json.Unmarshal(normalizeBody(body), &req); err != nil {
		writeErr(w, "INVALID_ARGUMENT", err.Error())
		return
	}
	doRecordQuery(w, req)
}

func handlePlaybackStart(cfg config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeErr(w, "INVALID_ARGUMENT", err.Error())
			return
		}
		var req playbackStartRequest
		if err := json.Unmarshal(normalizeBody(body), &req); err != nil {
			writeErr(w, "INVALID_ARGUMENT", err.Error())
			return
		}
		doPlaybackStart(cfg, w, req)
	}
}

// handlePlaybackStop 回放是"把 RTSP 地址交给 ZMS 去拉"，插件这边没有会话要拆，
// ZMS 断开 RTSP 连接回放即停止，所以这里是幂等的空操作。
func handlePlaybackStop(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]any{"stopped": true})
}

// handlePlaybackControl 暂停/倍速/拖放需要设备侧的控制通道。
// 当前是"交给 ZMS 拉 RTSP"的传输模式，没有可用的控制面，明确回不支持，
// 让平台把回放控制按钮置灰，而不是假装成功。
func handlePlaybackControl(w http.ResponseWriter, _ *http.Request) {
	writeErr(w, "UNSUPPORTED", "RTSP 回放取流不支持暂停/倍速/拖放控制")
}

// camelToSnake 是文档字段名与代码字段名的对应关系。
// docs/插件化设备接入.md 的 RPC 契约用 camelCase（deviceUid/startTime），
// 早期实现用的是 snake_case（device_uid/start_time），两种写法都接。
var camelToSnake = map[string]string{
	"deviceUid":       "device_uid",
	"channelId":       "channel_id",
	"streamKey":       "stream_key",
	"startTime":       "start_time",
	"endTime":         "end_time",
	"preferTransport": "prefer_transport",
	"streamType":      "stream_type",
}

// normalizeBody 在 snake_case 缺失时用 camelCase 补上，其余字段原样保留。
func normalizeBody(body []byte) []byte {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return body
	}
	changed := false
	for camel, snake := range camelToSnake {
		if _, ok := m[snake]; ok {
			continue
		}
		if v, ok := m[camel]; ok {
			m[snake] = v
			changed = true
		}
	}
	if !changed {
		return body
	}
	out, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return out
}

// handleCommand 是通用命令透传入口（平台 IPCClient.Command 打到这里），
// 用于还没定型的自定义能力，平台不解析 params 语义。
func handleCommand(cfg config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			DeviceUID string          `json:"device_uid"`
			ChannelID string          `json:"channel_id"`
			Cmd       string          `json:"cmd"`
			Params    json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, "INVALID_ARGUMENT", err.Error())
			return
		}
		switch req.Cmd {
		case "record.query":
			var q recordQueryRequest
			_ = json.Unmarshal(req.Params, &q)
			q.DeviceUID = req.DeviceUID
			if q.ChannelID == "" {
				q.ChannelID = req.ChannelID
			}
			doRecordQuery(w, q)
		case "playback.start":
			var p playbackStartRequest
			_ = json.Unmarshal(req.Params, &p)
			p.DeviceUID = req.DeviceUID
			if p.ChannelID == "" {
				p.ChannelID = req.ChannelID
			}
			doPlaybackStart(cfg, w, p)
		case "snapshot":
			var q struct {
				Quality int `json:"quality"`
			}
			_ = json.Unmarshal(req.Params, &q)
			doSnapshot(w, req.DeviceUID, req.ChannelID, q.Quality)
		default:
			writeErr(w, "UNSUPPORTED", "unknown command: "+req.Cmd)
		}
	}
}

func doRecordQuery(w http.ResponseWriter, req recordQueryRequest) {
	sessionMu.Lock()
	s := sessions[req.DeviceUID]
	sessionMu.Unlock()
	if s == nil {
		writeErr(w, "DEVICE_UNREACHABLE", "device not logged in")
		return
	}
	start, ok := parseFlexibleTime(req.StartTime)
	if !ok {
		writeErr(w, "INVALID_ARGUMENT", "start_time 无法解析")
		return
	}
	end, ok := parseFlexibleTime(req.EndTime)
	if !ok {
		writeErr(w, "INVALID_ARGUMENT", "end_time 无法解析")
		return
	}
	if end <= start {
		writeErr(w, "INVALID_ARGUMENT", "end_time 必须晚于 start_time")
		return
	}
	streamType := req.StreamType
	if streamType != 254 && (streamType < 0 || streamType > 3) {
		streamType = 0
	}
	records, err := sdk.FindRecords(s.UserID, s.channelOf(req.ChannelID), start, end, streamType)
	if err != nil {
		writeErr(w, "INTERNAL", err.Error())
		return
	}
	// 设备没开录像时 records 为空且 err 为 nil —— 这是正常结果，不是错误。
	writeOK(w, map[string]any{"records": records, "total": len(records)})
}

func doPlaybackStart(cfg config, w http.ResponseWriter, req playbackStartRequest) {
	sessionMu.Lock()
	s := sessions[req.DeviceUID]
	sessionMu.Unlock()
	if s == nil {
		writeErr(w, "DEVICE_UNREACHABLE", "device not logged in")
		return
	}
	start, ok := parseFlexibleTime(req.StartTime)
	if !ok {
		writeErr(w, "INVALID_ARGUMENT", "start_time 无法解析")
		return
	}
	end, ok := parseFlexibleTime(req.EndTime)
	if !ok {
		writeErr(w, "INVALID_ARGUMENT", "end_time 无法解析")
		return
	}
	if end <= start {
		writeErr(w, "INVALID_ARGUMENT", "end_time 必须晚于 start_time")
		return
	}
	// 回放取流走海康的按时间回放 RTSP 地址，由 ZMS 拉流；
	// 暂停/倍速/拖放需要 SIP INFO 或 SDK 控制通道，当前传输模式下不支持。
	rtsp := buildPlaybackRTSPURL(cfg, s, displayChannelOf(req.ChannelID), streamNumber(req.Stream), start, end)
	writeOK(w, map[string]any{"transport": "rtsp_url", "rtspUrl": rtsp})
}

func displayChannelOf(channelID string) int {
	if n, err := strconv.Atoi(channelID); err == nil && n > 0 {
		return n
	}
	return 1
}

// parseFlexibleTime 兼容 Unix 秒、"2006-01-02 15:04:05" 与 RFC3339 三种写法。
func parseFlexibleTime(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return n, true
		}
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, false
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n, true
		}
		for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339, "2006-01-02T15:04:05"} {
			if ts, err := time.ParseInLocation(layout, s, time.Local); err == nil {
				return ts.Unix(), true
			}
		}
	}
	return 0, false
}

// streamNumber 把码流名换算成海康 RTSP 路径里的码流号：1=主码流，2=子码流。
// 注意 hikvision 的通道号与码流号是两个维度，子码流不是"通道 2"。
func streamNumber(stream string) int {
	switch strings.ToLower(strings.TrimSpace(stream)) {
	case "sub", "2", "substream", "子码流":
		return 2
	default:
		return 1
	}
}

// buildRTSPURL 拼实时预览 RTSP 地址：/Streaming/Channels/<通道号*100+码流号>。
func buildRTSPURL(cfg config, s *session, displayChannel, stream int) string {
	if stream != 2 {
		stream = 1
	}
	pathChannel := displayChannel*100 + stream
	if cfg.rtspTmpl != "" {
		return fmt.Sprintf(cfg.rtspTmpl, s.User, s.Pass, s.IP, pathChannel)
	}
	u := &url.URL{
		Scheme: "rtsp",
		User:   url.UserPassword(s.User, s.Pass),
		Host:   net.JoinHostPort(s.IP, strconv.Itoa(cfg.rtspPort)),
		Path:   fmt.Sprintf("/Streaming/Channels/%d", pathChannel),
	}
	return u.String()
}

// hikLocalTimeLayout 是海康回放/下载地址里的时间格式：20260101T120000Z。
// 设备按自己的本地时间解释这个串（Z 只是格式约定），所以这里不做 UTC 换算，
// 直接把"设备本地时间"的墙上时间写进去，避免时区偏移导致回放起点差几个小时。
const hikLocalTimeLayout = "20060102T150405Z"

// buildPlaybackRTSPURL 拼按时间回放的 RTSP 地址：
// /Streaming/tracks/<通道号*100+码流号>?starttime=...&endtime=...
func buildPlaybackRTSPURL(cfg config, s *session, displayChannel, stream int, start, end int64) string {
	if stream != 2 {
		stream = 1
	}
	pathChannel := displayChannel*100 + stream
	u := &url.URL{
		Scheme: "rtsp",
		User:   url.UserPassword(s.User, s.Pass),
		Host:   net.JoinHostPort(s.IP, strconv.Itoa(cfg.rtspPort)),
		Path:   fmt.Sprintf("/Streaming/tracks/%d", pathChannel),
	}
	q := url.Values{}
	q.Set("starttime", time.Unix(start, 0).Format(hikLocalTimeLayout))
	q.Set("endtime", time.Unix(end, 0).Format(hikLocalTimeLayout))
	u.RawQuery = q.Encode()
	return u.String()
}

// channelOf 把协议侧通道号（展示用，从 1 开始）换成 SDK 通道号。
func (s *session) channelOf(display string) int {
	if n, err := strconv.Atoi(display); err == nil && n > 0 {
		if sdkCh, ok := s.Channels[display]; ok {
			return sdkCh
		}
		return n
	}
	if sdkCh, ok := s.Channels["1"]; ok {
		return sdkCh
	}
	return 1
}

var hikPTZCmd = map[string]int{
	"up": 21, "down": 22, "left": 23, "right": 24,
	// 斜向是 SDK 自己的复合命令（25~28），不是两次单方向命令的拼接。
	"upleft": 25, "upright": 26, "downleft": 27, "downright": 28,
	"zoomin": 11, "zoomout": 12,
	"focusnear": 13, "focusfar": 14,
	"irisnear": 15, "irisfar": 16,
}

// normalizePTZDirection 把平台的方向取值统一成小写无分隔符，兼容
// upleft / UP_LEFT / up-left 几种写法。
func normalizePTZDirection(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("_", "", "-", "", " ", "").Replace(s)
	return s
}

func handlePTZ(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceUID string `json:"device_uid"`
		ChannelID string `json:"channel_id"`
		Direction string `json:"direction"`
		H         int    `json:"h"`
		V         int    `json:"v"`
		Z         int    `json:"z"`
		Stop      bool   `json:"stop"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, "INVALID_ARGUMENT", err.Error())
		return
	}
	sessionMu.Lock()
	s := sessions[req.DeviceUID]
	sessionMu.Unlock()
	if s == nil {
		writeErr(w, "DEVICE_UNREACHABLE", "device not logged in")
		return
	}
	dir := normalizePTZDirection(req.Direction)
	isStop := req.Stop || dir == "stop"
	cmd, known := hikPTZCmd[dir]
	lastKey := req.DeviceUID + "|" + req.ChannelID
	if isStop {
		// 海康没有"停止"这条独立命令，必须拿同一个动作再发一次 stop=1，
		// 所以按 设备+通道 记住最后一次动的方向。没动过时直接回 OK（停是幂等的）。
		last, ok := lastPTZ.Load(lastKey)
		if !ok {
			writeOK(w, nil)
			return
		}
		cmd = last.(int)
	} else if !known {
		writeErr(w, "INVALID_ARGUMENT", "unknown direction")
		return
	}
	sdkCh, ok := s.Channels[req.ChannelID]
	if !ok {
		if n, err := strconv.Atoi(req.ChannelID); err == nil {
			sdkCh = n
		} else {
			sdkCh = 1
		}
	}
	speed := req.Z
	if speed <= 0 {
		speed = req.H
	}
	if speed <= 0 {
		speed = 4
	}
	if isStop {
		_ = sdk.PTZ(s.UserID, sdkCh, cmd, true, speed)
		lastPTZ.Delete(lastKey)
		writeOK(w, nil)
		return
	}
	if err := sdk.PTZ(s.UserID, sdkCh, cmd, false, speed); err != nil {
		writeErr(w, "INTERNAL", err.Error())
		return
	}
	lastPTZ.Store(lastKey, cmd)
	// Continuous move: stop shortly after; callers may also send explicit stop.
	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = sdk.PTZ(s.UserID, sdkCh, cmd, true, speed)
	}()
	writeOK(w, nil)
}

func okHandler(w http.ResponseWriter, _ *http.Request) { writeOK(w, nil) }

// handleSnapshot 抓一张 JPEG，以 base64 放在 data.image 里回给平台。
//
// 走 SDK 抓图接口而不是从直播流截帧：截帧要求有人正在播，抓图不要求。
func handleSnapshot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceUID string `json:"device_uid"`
		ChannelID string `json:"channel_id"`
		Quality   int    `json:"quality"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, "INVALID_ARGUMENT", err.Error())
		return
	}
	doSnapshot(w, req.DeviceUID, req.ChannelID, req.Quality)
}

func writeSnapshot(w http.ResponseWriter, img []byte) {
	writeOK(w, map[string]any{
		"format":   "jpeg",
		"encoding": "base64",
		"image":    base64.StdEncoding.EncodeToString(img),
		"size":     len(img),
	})
}

// doSnapshot 供 /v1/command 的分发复用。
func doSnapshot(w http.ResponseWriter, deviceUID, channelID string, quality int) {
	sessionMu.Lock()
	s := sessions[deviceUID]
	sessionMu.Unlock()
	if s == nil {
		writeErr(w, "DEVICE_UNREACHABLE", "device not logged in")
		return
	}
	img, err := sdk.CaptureJPEG(s.UserID, s.channelOf(channelID), quality)
	if err != nil {
		writeErr(w, "INTERNAL", err.Error())
		return
	}
	writeSnapshot(w, img)
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, map[string]any{"code": "OK", "message": "", "data": data})
}

func writeErr(w http.ResponseWriter, code, msg string) {
	writeJSON(w, map[string]any{"code": code, "message": msg, "data": nil})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func paramString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprint(v))
	}
	return ""
}

func paramInt(m map[string]any, key string, def int) int {
	if m == nil {
		return def
	}
	if v, ok := m[key]; ok && v != nil {
		if n, err := strconv.Atoi(fmt.Sprint(v)); err == nil {
			return n
		}
	}
	return def
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
