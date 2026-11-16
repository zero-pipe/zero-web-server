//go:build hik && cgo

// Command hik-ipc-plugin is a Hikvision HCNetSDK sidecar skeleton.
// It registers with the platform, logs in devices through HCNetSDK, exposes
// channels/PTZ, and returns a Hikvision RTSP URL as the live source.
//
// Build requires HCNetSDK headers/libs under ./sdk (see README.md).
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	platformURL string
	endpoint    string
	token       string
	listen      string
	channelNum  int
	rtspTmpl    string
}

type session struct {
	UserID  int64
	IP      string
	Port    int
	User    string
	Pass    string
	Channel int
}

var (
	sdk       HikSDK
	sessionMu sync.Mutex
	sessions  = map[string]*session{}
)

func main() {
	cfg := config{
		platformURL: env("PLATFORM_URL", "http://127.0.0.1:18080"),
		endpoint:    env("PLUGIN_ENDPOINT", "http://127.0.0.1:19091"),
		token:       env("PLUGIN_TOKEN", ""),
		listen:      env("PLUGIN_LISTEN", ":19091"),
		channelNum:  envInt("HIK_CHANNEL_COUNT", 1),
		rtspTmpl:    env("HIK_RTSP_TEMPLATE", "rtsp://%s:%s@%s:554/Streaming/Channels/%d"),
	}
	if err := sdk.Init(); err != nil {
		log.Fatalf("HCNetSDK init: %v", err)
	}
	defer sdk.Cleanup()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/login", handleLogin)
	mux.HandleFunc("/v1/logout", handleLogout)
	mux.HandleFunc("/v1/channels", handleChannels(cfg.channelNum))
	mux.HandleFunc("/v1/live/start", handleLiveStart(cfg))
	mux.HandleFunc("/v1/live/stop", okHandler)
	mux.HandleFunc("/v1/ptz", handlePTZ)

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
	channel := paramInt(req.Params, "channel", 1)
	if ip == "" || user == "" {
		writeErr(w, "INVALID_ARGUMENT", "missing ip/username")
		return
	}
	uid, err := sdk.Login(ip, port, user, pass)
	if err != nil {
		writeErr(w, "AUTH_FAILED", err.Error())
		return
	}
	sessionMu.Lock()
	sessions[req.DeviceUID] = &session{UserID: uid, IP: ip, Port: port, User: user, Pass: pass, Channel: channel}
	sessionMu.Unlock()
	log.Printf("hik login device=%s ip=%s uid=%d", req.DeviceUID, ip, uid)
	writeOK(w, map[string]any{
		"manufacturer": "Hikvision",
		"model":        "IPC",
		"firmware":     "",
		"serialNumber": req.DeviceUID,
	})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceUID string `json:"device_uid"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	sessionMu.Lock()
	if s, ok := sessions[req.DeviceUID]; ok {
		sdk.Logout(s.UserID)
		delete(sessions, req.DeviceUID)
	}
	sessionMu.Unlock()
	writeOK(w, nil)
}

func handleChannels(count int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		// Skeleton: fixed channel count. Replace with NET_DVR_GetDVRConfig
		// (NET_DVR_GET_IPPARACFG_V40) for real channel enumeration.
		if count <= 0 {
			count = 1
		}
		list := make([]map[string]any, 0, count)
		for i := 1; i <= count; i++ {
			list = append(list, map[string]any{
				"channelId": strconv.Itoa(i),
				"name":      fmt.Sprintf("通道 %d", i),
				"status":    "ON",
				"ptzType":   3,
				"hasAudio":  true,
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
		ch := s.Channel
		if n, err := strconv.Atoi(req.ChannelID); err == nil && n > 0 {
			ch = n
		}
		// Hikvision RTSP channel mapping: CH1 -> 101, CH2 -> 102 ...
		rtsp := fmt.Sprintf(cfg.rtspTmpl, s.User, s.Pass, s.IP, ch*100+1)
		writeOK(w, map[string]any{"transport": "rtsp_url", "rtspUrl": rtsp})
	}
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
	ch, _ := strconv.Atoi(req.ChannelID)
	if ch <= 0 {
		ch = s.Channel
	}
	if req.Stop {
		_ = sdk.PTZ(s.UserID, ch, cmd, true)
		writeOK(w, nil)
		return
	}
	if err := sdk.PTZ(s.UserID, ch, cmd, false); err != nil {
		writeErr(w, "INTERNAL", err.Error())
		return
	}
	// Continuous move: stop shortly after; callers can send explicit stop.
	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = sdk.PTZ(s.UserID, ch, cmd, true)
	}()
	writeOK(w, nil)
}

func register(cfg config) {
	body := map[string]any{
		"id":               pluginInstanceID,
		"pluginId":         pluginID,
		"vendor":           "hikvision",
		"version":          "0.1.0",
		"protocols":        []string{protocol},
		"capabilities":     []string{"live", "ptz"},
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
