// Package storagekit 是平台访问 zstorage（云存储节点）的 HTTP 客户端。
//
// zstorage 是 ZMS 的 fork，API 形态一致：A 族控制面 /index/api/<path>?secret=…，
// 统一响应 {code,msg,data}，code==0 为成功。本包只封装旁路录制相关的 zstorage/ 分组接口。
package storagekit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	secret  string
	http    *http.Client
}

func NewClient(ip string, httpPort int, secret string) *Client {
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
}

// RecordStart 让 zstorage 从 url 拉流并按 channel_key 录制到槽位。
// channel 即平台流名（<设备内码>_<通道内码>），url 为 ZMS 的 RTSP 拉流地址。
func (c *Client) RecordStart(ctx context.Context, channel, pullURL string) (*APIResponse, error) {
	params := url.Values{"channel": {channel}, "url": {pullURL}}
	return c.get(ctx, "zstorage/record/start", params)
}

// RecordStop 停止某通道的旁路录制（会封口当前槽位）。
func (c *Client) RecordStop(ctx context.Context, channel string) (*APIResponse, error) {
	return c.get(ctx, "zstorage/record/stop", url.Values{"channel": {channel}})
}

// RecordList 列出存储节点当前录制任务。
func (c *Client) RecordList(ctx context.Context) (*APIResponse, error) {
	return c.get(ctx, "zstorage/record/list", nil)
}

// RecordQuery 查询某通道（或全部）在 [startMs, endMs] 的已封口分段。
func (c *Client) RecordQuery(ctx context.Context, channel string, startMs, endMs int64) (*APIResponse, error) {
	params := url.Values{"start": {fmt.Sprintf("%d", startMs)}, "end": {fmt.Sprintf("%d", endMs)}}
	if channel != "" {
		params.Set("channel", channel)
	}
	return c.get(ctx, "zstorage/record/query", params)
}

// RecordDelete 按槽位号删除录像（CSV）。
func (c *Client) RecordDelete(ctx context.Context, slots []int64) (*APIResponse, error) {
	parts := make([]string, 0, len(slots))
	for _, s := range slots {
		parts = append(parts, fmt.Sprintf("%d", s))
	}
	return c.get(ctx, "zstorage/record/delete", url.Values{"slots": {strings.Join(parts, ",")}})
}

// NodeStatus 查询存储节点容量与录制任务统计。
func (c *Client) NodeStatus(ctx context.Context) (*APIResponse, error) {
	return c.get(ctx, "zstorage/node/status", nil)
}

// PlaybackStart 在存储节点打开某槽位回放，返回 {app,stream,slot,start,duration}。
func (c *Client) PlaybackStart(ctx context.Context, slot int64, channel string, startMs int64) (*APIResponse, error) {
	params := url.Values{
		"slot":    {fmt.Sprintf("%d", slot)},
		"channel": {channel},
		"start":   {fmt.Sprintf("%d", startMs)},
	}
	return c.get(ctx, "zstorage/playback/start", params)
}

// PlaybackStop 关闭回放会话。
func (c *Client) PlaybackStop(ctx context.Context, app, stream string) (*APIResponse, error) {
	return c.get(ctx, "zstorage/playback/stop", url.Values{"app": {app}, "stream": {stream}})
}

// BuildMediaURLs 拼出回放出流地址（与 ZMS 同构，前端零改动）。
// httpPort 用于 FLV/WS-FLV，rtspPort 用于 RTSP（zstorage 与 ZMS 端口已分离，不能混用）。
func (c *Client) BuildMediaURLs(app, stream string, httpPort, rtspPort int) map[string]string {
	host := strings.TrimPrefix(c.baseURL, "http://") // 形如 1.2.3.4:<httpPort>
	ip := host
	if i := strings.LastIndex(host, ":"); i >= 0 {
		ip = host[:i]
	}
	if httpPort <= 0 {
		if i := strings.LastIndex(host, ":"); i >= 0 {
			httpPort = atoiSafe(host[i+1:])
		}
	}
	httpHost := fmt.Sprintf("%s:%d", ip, httpPort)
	res := map[string]string{
		"flv":    "http://" + httpHost + "/" + app + "/" + stream + ".live.flv",
		"ws_flv": "ws://" + httpHost + "/" + app + "/" + stream + ".live.flv",
	}
	if rtspPort > 0 {
		res["rtsp"] = fmt.Sprintf("rtsp://%s:%d/%s/%s", ip, rtspPort, app, stream)
	}
	return res
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return n
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func (c *Client) get(ctx context.Context, api string, params url.Values) (*APIResponse, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("secret", c.secret)
	u := c.baseURL + "/index/api/" + api + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
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
	var r APIResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("zstorage %s 响应解析失败: %w (%s)", api, err, string(body))
	}
	return &r, nil
}
