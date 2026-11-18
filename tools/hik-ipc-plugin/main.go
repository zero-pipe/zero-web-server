//go:build hik && cgo

// Command hik-ipc-plugin is a Hikvision HCNetSDK sidecar.
//
// It uses HCNetSDK for login / device info / channel enumeration / PTZ, and
// returns a Hikvision RTSP URL as the live stream source (ZMS pulls it).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

const (
	pluginInstanceID = "hik-ipc-01"
	pluginID         = "hik-ipc"
	protocol         = "hik.private.v5"
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
	platformURL   string
	endpoint      string
	token         string
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
)

func main() {
	cfg := config{
		platformURL:   env("PLATFORM_URL", "http://127.0.0.1:18080"),
		endpoint:      env("PLUGIN_ENDPOINT", "http://127.0.0.1:19091"),
		token:         env("PLUGIN_TOKEN", ""),
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
	mux.HandleFunc("/v1/login", handleLogin)
	mux.HandleFunc("/v1/logout", handleLogout)
	mux.HandleFunc("/v1/channels", handleChannels(cfg))
	mux.HandleFunc("/v1/live/start", handleLiveStart(cfg))
	mux.HandleFunc("/v1/live/stop", okHandler)
	mux.HandleFunc("/v1/ptz", handlePTZ)
	mux.HandleFunc("/v1/arm", handleArm)
	mux.HandleFunc("/v1/disarm", handleDisarm)

	go func() {
		log.Printf("hik plugin listening on %s, endpoint=%s", cfg.listen, cfg.endpoint)
		if err := http.ListenAndServe(cfg.listen, mux); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()

	register(cfg)
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
		rtsp := buildRTSPURL(cfg, s, display)
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

func buildRTSPURL(cfg config, s *session, displayChannel int) string {
	// Hikvision channel mapping: CH1 -> 101, CH2 -> 102 ...
	pathChannel := displayChannel*100 + 1
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

var hikPTZCmd = map[string]int{
	"up": 21, "down": 22, "left": 23, "right": 24,
	"zoomIn": 11, "zoomOut": 12,
	"focusNear": 13, "focusFar": 14,
	"irisNear": 15, "irisFar": 16,
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
	cmd, ok := hikPTZCmd[req.Direction]
	if !ok {
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
	if req.Stop {
		_ = sdk.PTZ(s.UserID, sdkCh, cmd, true, speed)
		writeOK(w, nil)
		return
	}
	if err := sdk.PTZ(s.UserID, sdkCh, cmd, false, speed); err != nil {
		writeErr(w, "INTERNAL", err.Error())
		return
	}
	// Continuous move: stop shortly after; callers may also send explicit stop.
	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = sdk.PTZ(s.UserID, sdkCh, cmd, true, speed)
	}()
	writeOK(w, nil)
}

func register(cfg config) {
	body := map[string]any{
		"id":               pluginInstanceID,
		"pluginId":         pluginID,
		"vendor":           "hikvision",
		"version":          "0.2.0",
		"protocols":        []string{protocol},
		"capabilities":     []string{"live", "ptz", "alarm", "arming", "snapshot", "mqtt"},
		"streamTransports": []string{"rtsp_url"},
		"configSchema":     configSchema,
		"endpoint":         cfg.endpoint,
	}
	for {
		if err := post(cfg, "/api/v1/plugins/hello", body); err != nil {
			log.Printf("register failed (retry in 3s): %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		log.Printf("registered with platform: %s", pluginInstanceID)
		break
	}
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			if err := post(cfg, "/api/v1/plugins/heartbeat", map[string]any{"id": pluginInstanceID}); err != nil {
				log.Printf("heartbeat failed: %v", err)
			}
		}
	}()
}

func okHandler(w http.ResponseWriter, _ *http.Request) { writeOK(w, nil) }

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

func post(cfg config, path string, in any) error {
	raw, _ := json.Marshal(in)
	req, err := http.NewRequest(http.MethodPost, cfg.platformURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.token != "" {
		req.Header.Set("X-Plugin-Token", cfg.token)
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return nil
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
