package deviceaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	ipcdevice "zero-web-server/internal/application/ipcdevice"
	domainplugin "zero-web-server/internal/domain/plugin"
)

// pluginAdapter 插件（私有 SDK，经 zero-agent 托管）设备适配器。
// 复用 ipcdevice 服务，把插件设备投影成与 GB/ONVIF 对齐的统一设备视图。
type pluginAdapter struct {
	ipc *ipcdevice.Service
}

func (a pluginAdapter) Protocol() string                { return ProtocolPlugin }
func (a pluginAdapter) Supports(accessMode string) bool { return accessMode == AccessActive }

func (a pluginAdapter) List(ctx context.Context, q ListQuery) ([]*DeviceView, error) {
	list, err := a.ipc.ListDevices()
	if err != nil {
		return nil, err
	}
	views := make([]*DeviceView, 0, len(list))
	for _, d := range list {
		if q.Query != "" && !matchPluginQuery(d, q.Query) {
			continue
		}
		v := viewFromPlugin(d)
		if matchStatus(v, q.Status) {
			views = append(views, v)
		}
	}
	return views, nil
}

func (a pluginAdapter) GetByInternalCode(ctx context.Context, code string) (*DeviceView, bool, error) {
	list, err := a.ipc.ListDevices()
	if err != nil {
		return nil, false, err
	}
	for _, d := range list {
		if d != nil && strings.EqualFold(strings.TrimSpace(d.InternalCode), code) {
			return viewFromPlugin(d), true, nil
		}
	}
	return nil, false, nil
}

func (a pluginAdapter) Create(context.Context, CreateRequest) (*DeviceView, error) {
	// 插件设备走独立的添加流程（需选插件、按 configSchema 填参数），不经统一创建。
	return nil, fmt.Errorf("插件设备请在「设备接入」页添加")
}

func (a pluginAdapter) Update(ctx context.Context, rawID string, req UpdateRequest) (*DeviceView, error) {
	// 目前插件设备仅支持改名。
	id, err := strconv.Atoi(rawID)
	if err != nil {
		return nil, fmt.Errorf("无效的插件设备ID")
	}
	dev, err := a.ipc.GetDevice(id)
	if err != nil {
		return nil, fmt.Errorf("设备不存在")
	}
	dev.Name = strings.TrimSpace(req.Name)
	if err := a.ipc.UpdateDevice(dev); err != nil {
		return nil, err
	}
	return viewFromPlugin(dev), nil
}

func (a pluginAdapter) Delete(ctx context.Context, rawID string) error {
	id, err := strconv.Atoi(rawID)
	if err != nil {
		return fmt.Errorf("无效的插件设备ID")
	}
	return a.ipc.DeleteDevice(ctx, id)
}

func (a pluginAdapter) Sync(ctx context.Context, rawID string) error {
	id, err := strconv.Atoi(rawID)
	if err != nil {
		return fmt.Errorf("无效的插件设备ID")
	}
	_, err = a.ipc.SyncChannels(ctx, id)
	return err
}

func matchPluginQuery(d *domainplugin.Device, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	return strings.Contains(strings.ToLower(d.Name), q) ||
		strings.Contains(strings.ToLower(d.DeviceUID), q) ||
		strings.Contains(strings.ToLower(d.InternalCode), q)
}

// viewFromPlugin 把插件设备投影为统一设备视图。
// 订阅(事件) = 布防/撤防；统计(心跳) 由后台探活写入 Redis 后统一展示。
func viewFromPlugin(d *domainplugin.Device) *DeviceView {
	status := StatusOffline
	if d.Online {
		status = StatusOnline
	}
	addr := pluginAddress(d)
	return &DeviceView{
		ID:           fmt.Sprintf("plugin:%d", d.ID),
		InternalCode: d.InternalCode,
		Name:         d.Name,
		AccessMode:   AccessActive,
		Protocol:     ProtocolPlugin,
		Vendor:       d.Manufacturer,
		Model:        d.Model,
		Status:       status,
		ChannelCount: d.ChannelCount,
		AddressText:  addr,
		RawID:        strconv.Itoa(d.ID),
		DBID:         d.ID,
		// 插件能力：同步/直播/云台/布防撤防(事件)/编辑/删除
		Capabilities: []string{"sync", "play", "ptz", "event", "stats", "edit", "delete"},
		CreateTime:   d.CreateTime,
		UpdateTime:   d.UpdateTime,
	}
}

// pluginAddress 从 driverParams 的 ip/port 或 deviceUid 里的 ip-port 还原地址文本。
func pluginAddress(d *domainplugin.Device) string {
	var params map[string]any
	if strings.TrimSpace(d.DriverParams) != "" {
		_ = json.Unmarshal([]byte(d.DriverParams), &params)
	}
	if ip, ok := params["ip"].(string); ok && ip != "" {
		if port, ok := params["port"].(float64); ok && port > 0 {
			return fmt.Sprintf("%s:%d", ip, int(port))
		}
		return ip
	}
	uid := d.DeviceUID
	if i := strings.LastIndexByte(uid, '-'); i > 0 {
		if _, err := strconv.Atoi(uid[i+1:]); err == nil {
			return uid[:i] + ":" + uid[i+1:]
		}
	}
	if uid == "" {
		return "—"
	}
	return uid
}
