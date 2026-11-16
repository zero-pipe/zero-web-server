// Command mock-ipc-plugin is a minimal IPC sidecar used to validate the
// platform <-> plugin RPC loop without a vendor SDK.
//
// It registers itself with the platform, serves the /v1/* JSON-RPC endpoints,
// and returns a configurable RTSP URL as the live stream source.
//
// Environment:
//
//	PLATFORM_URL    platform API base (default http://127.0.0.1:18080)
//	PLUGIN_ENDPOINT this plugin's reachable endpoint (default http://127.0.0.1:19090)
//	PLUGIN_TOKEN    shared token, must match ZWS_PLUGIN_TOKEN on the platform
//	MOCK_RTSP_URL   RTSP source returned by /v1/live/start
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	pluginInstanceID = "mock-ipc-01"
	pluginID         = "mock-ipc"
	protocol         = "mock.ipc.v1"
)

var configSchema = `{
  "type": "object",
  "required": ["ip", "port", "username", "password"],
  "properties": {
    "ip":       { "type": "string",  "title": "设备 IP" },
    "port":     { "type": "integer", "title": "端口", "default": 8000 },
    "username": { "type": "string",  "title": "用户名" },
    "password": { "type": "string",  "title": "密码", "format": "password" },
    "channel":  { "type": "integer", "title": "默认通道", "default": 101 }
  }
}`

type pluginConfig struct {
	platformURL string
	endpoint    string
	token       string
	rtspURL     string
}

func main() {
	cfg := pluginConfig{
		platformURL: env("PLATFORM_URL", "http://127.0.0.1:18080"),
		endpoint:    env("PLUGIN_ENDPOINT", "http://127.0.0.1:19090"),
		token:       env("PLUGIN_TOKEN", ""),
		rtspURL:     env("MOCK_RTSP_URL", "rtsp://127.0.0.1:8554/live/test"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/login", handleLogin)
	mux.HandleFunc("/v1/logout", okHandler)
	mux.HandleFunc("/v1/channels", handleChannels)
	mux.HandleFunc("/v1/live/start", handleLiveStart(cfg.rtspURL))
	mux.HandleFunc("/v1/live/stop", okHandler)
	mux.HandleFunc("/v1/ptz", okHandler)

	go func() {
		addr := ":19090"
		if u := os.Getenv("PLUGIN_LISTEN"); u != "" {
			addr = u
		}
		log.Printf("mock plugin listening on %s, endpoint=%s", addr, cfg.endpoint)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()

	register(cfg)

	// Block forever; HTTP server and heartbeat run in goroutines.
	select {}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func register(cfg pluginConfig) {
	body := map[string]any{
		"id":               pluginInstanceID,
		"pluginId":         pluginID,
		"vendor":           "mock",
		"version":          "0.1.0",
		"protocols":        []string{protocol},
		"capabilities":     []string{"live", "ptz"},
		"streamTransports": transportsFromEnv(),
		"configSchema":     configSchema,
		"endpoint":         cfg.endpoint,
	}
	for {
		if err := post(cfg, "/api/v1/plugins/hello", body, nil); err != nil {
			log.Printf("register failed (retry in 3s): %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		log.Printf("registered with platform: %s", pluginInstanceID)
		break
	}
	go heartbeatLoop(cfg)
}

func heartbeatLoop(cfg pluginConfig) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := post(cfg, "/api/v1/plugins/heartbeat", map[string]any{"id": pluginInstanceID}, nil); err != nil {
			log.Printf("heartbeat failed: %v", err)
		}
	}
}

var (
	sessionMu sync.Mutex
	sessions  = map[string]bool{}
)

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceUID string         `json:"device_uid"`
		Params    map[string]any `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, "INVALID_ARGUMENT", err.Error())
		return
	}
	ip, _ := req.Params["ip"].(string)
	if ip == "" {
		writeErr(w, "INVALID_ARGUMENT", "missing params.ip")
		return
	}
	sessionMu.Lock()
	sessions[req.DeviceUID] = true
	sessionMu.Unlock()
	log.Printf("login device=%s ip=%s", req.DeviceUID, ip)
	writeOK(w, map[string]any{
		"manufacturer": "Mock",
		"model":        "IPC-SIM",
		"firmware":     "0.1.0",
		"serialNumber": req.DeviceUID,
	})
}

func handleChannels(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]any{
		"channels": []map[string]any{
			{"channelId": "101", "name": "通道 1", "status": "ON", "ptzType": 3, "hasAudio": true},
			{"channelId": "102", "name": "通道 2", "status": "ON", "ptzType": 3, "hasAudio": true},
		},
	})
}

func handleLiveStart(rtspURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			DeviceUID       string `json:"device_uid"`
			ChannelID       string `json:"channel_id"`
			StreamKey       string `json:"stream_key"`
			PreferTransport string `json:"prefer_transport"`
			RTP             *struct {
				SDPIP   string `json:"sdp_ip"`
				RTPPort int    `json:"rtp_port"`
				Mode    string `json:"mode"`
			} `json:"rtp"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		log.Printf("live start device=%s channel=%s stream=%s prefer=%s rtp=%v",
			req.DeviceUID, req.ChannelID, req.StreamKey, req.PreferTransport, req.RTP)
		if req.PreferTransport == "rtp_push" || req.RTP != nil {
			writeOK(w, map[string]any{"transport": "rtp_push"})
			return
		}
		writeOK(w, map[string]any{"transport": "rtsp_url", "rtspUrl": rtspURL})
	}
}

func transportsFromEnv() []string {
	raw := env("MOCK_TRANSPORTS", "rtsp_url")
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = []string{"rtsp_url"}
	}
	return out
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

func post(cfg pluginConfig, path string, in any, out any) error {
	raw, _ := json.Marshal(in)
	req, err := http.NewRequest(http.MethodPost, cfg.platformURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.token != "" {
		req.Header.Set("X-Plugin-Token", cfg.token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
