// Package ipcplugin implements the JSON-RPC client used by the platform to call
// IPC sidecar plugins. JSON over HTTP is the documented fallback transport.
package ipcplugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"zero-web-server/internal/port"
)

type Client struct {
	endpoint string
	token    string
	http     *http.Client
}

func NewClient(endpoint, token string) *Client {
	return &Client{
		endpoint: strings.TrimRight(strings.TrimSpace(endpoint), "/"),
		token:    strings.TrimSpace(token),
		http:     &http.Client{Timeout: 15 * time.Second},
	}
}

type envelope struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) post(ctx context.Context, path string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("X-Plugin-Token", c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ipc plugin %s: %w", path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("ipc plugin %s: http %d: %s", path, resp.StatusCode, string(raw))
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("ipc plugin %s: bad response: %w", path, err)
	}
	if env.Code != "" && env.Code != "OK" {
		return fmt.Errorf("ipc plugin %s: %s: %s", path, env.Code, env.Message)
	}
	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("ipc plugin %s: decode data: %w", path, err)
		}
	}
	return nil
}

func (c *Client) Login(ctx context.Context, deviceUID string, params map[string]any) (*port.IPCDeviceInfo, error) {
	var out port.IPCDeviceInfo
	if err := c.post(ctx, "/v1/login", map[string]any{
		"device_uid": deviceUID, "params": params,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Logout(ctx context.Context, deviceUID string) error {
	return c.post(ctx, "/v1/logout", map[string]any{"device_uid": deviceUID}, nil)
}

func (c *Client) ListChannels(ctx context.Context, deviceUID string) ([]port.IPCChannel, error) {
	var out struct {
		Channels []port.IPCChannel `json:"channels"`
	}
	if err := c.post(ctx, "/v1/channels", map[string]any{"device_uid": deviceUID}, &out); err != nil {
		return nil, err
	}
	return out.Channels, nil
}

func (c *Client) StartLive(ctx context.Context, deviceUID, channelID, streamKey, preferTransport string, push *port.IPCPushParams) (*port.IPCStreamSource, error) {
	body := map[string]any{
		"device_uid": deviceUID, "channel_id": channelID, "stream_key": streamKey,
		"prefer_transport": preferTransport,
	}
	if push != nil {
		body["rtp"] = map[string]any{
			"sdp_ip": push.SDPIP, "rtp_port": push.RTPPort, "mode": push.Mode,
		}
	}
	var out port.IPCStreamSource
	if err := c.post(ctx, "/v1/live/start", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) StopLive(ctx context.Context, deviceUID, channelID, streamKey string) error {
	return c.post(ctx, "/v1/live/stop", map[string]any{
		"device_uid": deviceUID, "channel_id": channelID, "stream_key": streamKey,
	}, nil)
}

func (c *Client) PTZ(ctx context.Context, deviceUID, channelID, direction string, h, v, z int) error {
	return c.post(ctx, "/v1/ptz", map[string]any{
		"device_uid": deviceUID, "channel_id": channelID,
		"direction": direction, "h": h, "v": v, "z": z,
	}, nil)
}

var _ port.IPCClient = (*Client)(nil)

type Factory struct{ token string }

func NewFactory(token string) Factory { return Factory{token: token} }

func (f Factory) For(endpoint string) port.IPCClient { return NewClient(endpoint, f.token) }

var _ port.IPCClientFactory = Factory{}
