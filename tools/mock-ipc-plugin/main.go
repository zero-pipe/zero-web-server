// Command mock-ipc-plugin is a minimal IPC sidecar used to validate the
// platform <-> agent <-> plugin RPC loop without a vendor SDK.
//
// It is started by zero-agent, serves the /v1/* JSON-RPC endpoints, and
// returns a configurable RTSP URL as the live stream source.
//
// Environment:
//
//	PLUGIN_LISTEN this plugin's local listen address (default :19090)
//	PLUGIN_TOKEN  shared token injected by the agent
//	MOCK_RTSP_URL RTSP source returned by /v1/live/start
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
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
	rtspURL string
}

func main() {
	cfg := pluginConfig{
		rtspURL: env("MOCK_RTSP_URL", "rtsp://127.0.0.1:8554/live/test"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", okHandler)
	mux.HandleFunc("/v1/info", handleInfo)
	mux.HandleFunc("/v1/login", handleLogin)
	mux.HandleFunc("/v1/logout", okHandler)
	mux.HandleFunc("/v1/channels", handleChannels)
	mux.HandleFunc("/v1/live/start", handleLiveStart(cfg.rtspURL))
	mux.HandleFunc("/v1/live/stop", okHandler)
	mux.HandleFunc("/v1/ptz", okHandler)
	mux.HandleFunc("/v1/snapshot", okHandler)
	mux.HandleFunc("/v1/config/get", okHandler)
	mux.HandleFunc("/v1/config/set", okHandler)
	mux.HandleFunc("/v1/record/query", handleRecordQuery)
	mux.HandleFunc("/v1/playback/start", handleLiveStart(cfg.rtspURL))
	mux.HandleFunc("/v1/playback/stop", okHandler)
	mux.HandleFunc("/v1/playback/control", okHandler)
	mux.HandleFunc("/v1/arm", handleArm)
	mux.HandleFunc("/v1/disarm", handleDisarm)
	mux.HandleFunc("/v1/mock/event", handleMockEvent)

	go func() {
		addr := ":19090"
		if u := os.Getenv("PLUGIN_LISTEN"); u != "" {
			addr = u
		}
		log.Printf("mock plugin listening on %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Block forever; the HTTP server runs in a goroutine.
	select {}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type mockArmConfig struct {
	DeviceUID          string   `json:"device_uid"`
	DeviceInternalCode string   `json:"device_internal_code"`
	EventTypes         []string `json:"event_types"`
	MQTT               struct {
		Broker    string `json:"broker"`
		ClientID  string `json:"clientId"`
		Username  string `json:"username"`
		Password  string `json:"password"`
		Topic     string `json:"topic"`
		QoS       int    `json:"qos"`
		KeepAlive int    `json:"keepAlive"`
	} `json:"mqtt"`
}

var (
	sessionMu sync.Mutex
	sessions  = map[string]bool{}
	armed     = map[string]string{}
	armConfig = map[string]*mockArmConfig{}
	mqttConns = map[string]mqtt.Client{}
)

func handleArm(w http.ResponseWriter, r *http.Request) {
	var req mockArmConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, "INVALID_ARGUMENT", err.Error())
		return
	}
	if req.DeviceUID == "" || req.DeviceInternalCode == "" {
		writeErr(w, "INVALID_ARGUMENT", "missing device_uid/device_internal_code")
		return
	}
	if req.MQTT.Broker == "" || req.MQTT.Topic == "" {
		writeErr(w, "INVALID_ARGUMENT", "missing mqtt config")
		return
	}
	opts := mqtt.NewClientOptions().AddBroker(req.MQTT.Broker).SetClientID(req.MQTT.ClientID)
	if req.MQTT.Username != "" {
		opts.SetUsername(req.MQTT.Username)
	}
	if req.MQTT.Password != "" {
		opts.SetPassword(req.MQTT.Password)
	}
	ka := req.MQTT.KeepAlive
	if ka <= 0 {
		ka = 30
	}
	opts.SetKeepAlive(time.Duration(ka) * time.Second).SetAutoReconnect(true).SetConnectTimeout(6 * time.Second)
	cli := mqtt.NewClient(opts)
	token := cli.Connect()
	if !token.WaitTimeout(8*time.Second) || token.Error() != nil {
		writeErr(w, "INTERNAL", "mqtt connect failed")
		return
	}
	subID := fmt.Sprintf("mock-%s-%d", req.DeviceUID, time.Now().UnixNano())
	sessionMu.Lock()
	if old := mqttConns[req.DeviceUID]; old != nil && old.IsConnectionOpen() {
		old.Disconnect(250)
	}
	armed[req.DeviceUID] = subID
	armConfig[req.DeviceUID] = &req
	mqttConns[req.DeviceUID] = cli
	sessionMu.Unlock()
	log.Printf("arm device=%s internal=%s events=%v subscription=%s", req.DeviceUID, req.DeviceInternalCode, req.EventTypes, subID)
	writeOK(w, map[string]any{"armed": true, "subscriptionId": subID})
}

func handleMockEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DeviceUID string `json:"device_uid"`
		EventType string `json:"event_type"`
		ChannelID string `json:"channel_id"`
		ImageKey  string `json:"image_key"`
		ImageURL  string `json:"image_url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	sessionMu.Lock()
	cfg := armConfig[body.DeviceUID]
	cli := mqttConns[body.DeviceUID]
	sessionMu.Unlock()
	if cfg == nil || cli == nil {
		writeErr(w, "DEVICE_UNREACHABLE", "device not armed")
		return
	}
	if body.EventType == "" {
		body.EventType = "motion"
	}
	if body.ChannelID == "" {
		body.ChannelID = "1"
	}
	now := time.Now().UnixMilli()
	event := map[string]any{
		"schema": "zws.event.v1", "eventId": fmt.Sprintf("mock-%d", now),
		"deviceInternalCode": cfg.DeviceInternalCode, "deviceUid": cfg.DeviceUID,
		"channelId": body.ChannelID, "eventType": body.EventType, "severity": "warning",
		"occurredAt": now, "receivedAt": now,
		"raw": map[string]any{"vendor": "mock", "eventType": body.EventType},
	}
	if body.ImageKey != "" {
		event["image"] = map[string]any{"key": body.ImageKey, "url": body.ImageURL}
	}
	payload, _ := json.Marshal(event)
	token := cli.Publish(cfg.MQTT.Topic, byte(cfg.MQTT.QoS), false, payload)
	if !token.WaitTimeout(5*time.Second) || token.Error() != nil {
		writeErr(w, "INTERNAL", "mqtt publish failed")
		return
	}
	writeOK(w, map[string]any{"eventId": event["eventId"]})
}

func handleDisarm(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceUID      string `json:"device_uid"`
		SubscriptionID string `json:"subscription_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	sessionMu.Lock()
	delete(armed, req.DeviceUID)
	delete(armConfig, req.DeviceUID)
	if cli := mqttConns[req.DeviceUID]; cli != nil {
		cli.Disconnect(250)
		delete(mqttConns, req.DeviceUID)
	}
	sessionMu.Unlock()
	log.Printf("disarm device=%s subscription=%s", req.DeviceUID, req.SubscriptionID)
	writeOK(w, nil)
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

// handleInfo 返回清单摘要，按规范 5.2 提供 /v1/info。
func handleInfo(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, map[string]any{
		"id":           "mock-ipc",
		"version":      "0.1.0",
		"protocols":    []string{"mock"},
		"capabilities": []string{"login", "channels", "live", "ptz", "snapshot", "arm", "disarm", "playback", "record_query", "config"},
	})
}

// handleRecordQuery 返回固定两条录像，便于验证回放链路。
func handleRecordQuery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceUID string `json:"device_uid"`
		ChannelID string `json:"channel_id"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	log.Printf("record query device=%s channel=%s %s~%s", req.DeviceUID, req.ChannelID, req.StartTime, req.EndTime)
	start := req.StartTime
	if start == "" {
		start = "2026-01-01T00:00:00.000"
	}
	writeOK(w, map[string]any{
		"records": []map[string]any{
			{"streamId": "mock-" + req.ChannelID + "-1", "startTime": start, "endTime": start, "type": "timing", "storage": "device"},
		},
	})
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
