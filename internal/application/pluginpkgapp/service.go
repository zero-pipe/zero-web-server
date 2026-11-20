// Package pluginpkgapp 编排插件包的上传登记、安装下发与卸载。
//
// 边界：平台只信任上传时校验过的清单；Agent 只从平台指定 URL 下载，
// 平台不直连插件进程，安装/卸载全部通过 Agent 的 HTTP 控制面完成。
package pluginpkgapp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	domainagent "zero-web-server/internal/domain/agent"
	domainpkg "zero-web-server/internal/domain/pluginpkg"
	applog "zero-web-server/pkg/log"
)

// ErrPackageInUse 卸载/删除前检查到设备引用。
var ErrPackageInUse = errors.New("插件仍有设备接入，请先删除设备")

// ErrNotFound 资源不存在（与领域层保持一致）。
var ErrNotFound = domainpkg.ErrNotFound

// Store 是本服务依赖的对象存储（由 objectstore.Service.Store 提供）。
type Store interface {
	Provider() string
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// AgentRepo 提供 Agent 在线信息。
type AgentRepo interface {
	GetAgent(id string) (*domainagent.Agent, error)
}

// DeviceCounter 统计插件设备引用（由 ipcdevice 服务实现）。
type DeviceCounter interface {
	CountByPlugin(pluginID string) (int, error)
}

type Service struct {
	packages domainpkg.Repository
	agents   AgentRepo
	devices  DeviceCounter
	storeFn  func() Store
	token    string
	http     *http.Client
}

// NewService 构造服务；storeFn 每次取当前对象存储（配置热重载后可即时生效）。
func NewService(packages domainpkg.Repository, agents AgentRepo, devices DeviceCounter, storeFn func() Store, agentToken string) *Service {
	return &Service{
		packages: packages, agents: agents, devices: devices, storeFn: storeFn,
		token: strings.TrimSpace(agentToken),
		http:  &http.Client{Timeout: 10 * time.Minute},
	}
}

// Upload 校验并登记插件包。data 为已读入内存的包体（上传体积受 MaxPackageBytes 约束）。
func (s *Service) Upload(ctx context.Context, data []byte) (*domainpkg.Package, error) {
	res, err := domainpkg.FromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("插件包校验失败: %w", err)
	}
	existing, _ := s.packages.GetPackageByPluginVersion(res.Manifest.ID, res.Manifest.Version)
	if existing != nil {
		// 同插件同版本：哈希一致视为重复上传，直接返回旧记录（幂等）。
		if existing.SHA256 == res.SHA256 {
			return existing, nil
		}
		return nil, fmt.Errorf("插件 %s@%s 已存在且内容不同，请先删除旧包", res.Manifest.ID, res.Manifest.Version)
	}
	store := s.store()
	if store == nil {
		return nil, fmt.Errorf("对象存储未配置，无法保存插件包")
	}
	key := fmt.Sprintf("plugins/%s/%s/%s.zwplugin", res.Manifest.ID, res.Manifest.Version, res.SHA256)
	if err := store.Put(ctx, key, bytes.NewReader(data), int64(len(data)), "application/zip"); err != nil {
		return nil, fmt.Errorf("保存插件包失败: %w", err)
	}
	p := &domainpkg.Package{
		ID: newID(), PluginID: res.Manifest.ID, Version: res.Manifest.Version,
		Name: res.Manifest.Name, Vendor: res.Manifest.Vendor,
		ObjectKey: key, SHA256: res.SHA256, SizeBytes: int64(len(data)),
		ManifestJSON: res.ManifestJSON, ConfigSchema: res.ConfigSchema,
		Protocols: res.Manifest.Protocols, Capabilities: res.Manifest.Capabilities,
		StreamTransports: res.Manifest.StreamTransports, EventTypes: res.Manifest.EventTypes,
		Architectures: res.Architectures, Status: domainpkg.PackageValid,
	}
	if err := s.packages.CreatePackage(p); err != nil {
		_ = store.Delete(ctx, key)
		return nil, err
	}
	applog.Infof("[plugin-pkg] 上传插件包 %s@%s sha256=%s size=%d", p.PluginID, p.Version, p.SHA256, p.SizeBytes)
	// 回填解析后的清单（CreatePackage 未持有 Manifest 指针）。
	p.Manifest = res.Manifest
	return p, nil
}

// List 返回全部插件包。
func (s *Service) List() ([]*domainpkg.Package, error) { return s.packages.ListPackages() }

// Get 返回单个插件包。
func (s *Service) Get(id string) (*domainpkg.Package, error) {
	return s.packages.GetPackage(id)
}

// Open 返回包体流（供 Agent 下载）。
func (s *Service) Open(ctx context.Context, id string) (*domainpkg.Package, io.ReadCloser, error) {
	p, err := s.Get(id)
	if err != nil {
		return nil, nil, err
	}
	store := s.store()
	if store == nil {
		return nil, nil, fmt.Errorf("对象存储未配置")
	}
	rc, err := store.Get(ctx, p.ObjectKey)
	if err != nil {
		return nil, nil, fmt.Errorf("读取插件包失败: %w", err)
	}
	return p, rc, nil
}

// Delete 删除插件包：有安装记录时拒绝，以免 Agent 侧悬浮。
func (s *Service) Delete(ctx context.Context, id string) error {
	p, err := s.Get(id)
	if err != nil {
		return err
	}
	installs, err := s.packages.ListInstalls()
	if err != nil {
		return err
	}
	for _, in := range installs {
		if in.PackageID == p.ID && in.Status != domainpkg.InstallUninstalled {
			return fmt.Errorf("该包仍有安装记录，请先卸载")
		}
	}
	if store := s.store(); store != nil {
		_ = store.Delete(ctx, p.ObjectKey)
	}
	return s.packages.DeletePackage(id)
}

// Install 向目标 Agent 下发安装命令并登记任务。
func (s *Service) Install(ctx context.Context, packageID, agentID, downloadBase string) (*domainpkg.Install, error) {
	p, err := s.Get(packageID)
	if err != nil {
		return nil, err
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return nil, fmt.Errorf("agentId 不能为空")
	}
	agent, err := s.agents.GetAgent(agentID)
	if err != nil || agent == nil {
		return nil, fmt.Errorf("Agent 不存在")
	}
	if agent.Status != "online" {
		return nil, fmt.Errorf("Agent 离线，无法安装")
	}

	in, _ := s.packages.FindInstall(agentID, packageID)
	if in == nil {
		in = &domainpkg.Install{
			ID: newID(), AgentID: agentID, PackageID: packageID,
			PluginID: p.PluginID, Version: p.Version, Status: domainpkg.InstallPending,
		}
		if err := s.packages.CreateInstall(in); err != nil {
			return nil, err
		}
	} else if in.Status == domainpkg.InstallInstalled {
		return in, nil // 幂等：已装成功直接返回
	}

	body := map[string]any{
		"packageId": p.ID, "pluginId": p.PluginID, "version": p.Version,
		"url":    strings.TrimRight(downloadBase, "/") + "/api/v1/plugin-packages/" + p.ID + "/download",
		"sha256": p.SHA256, "sizeBytes": p.SizeBytes,
	}
	if err := s.callAgent(ctx, agent, p.PluginID, "install", body); err != nil {
		in.Status, in.LastError = domainpkg.InstallFailed, err.Error()
		_ = s.packages.UpdateInstall(in)
		return in, err
	}
	in.Status, in.LastError = domainpkg.InstallInstalled, ""
	if err := s.packages.UpdateInstall(in); err != nil {
		return in, err
	}
	applog.Infof("[plugin-pkg] Agent %s 安装 %s@%s 成功", agentID, p.PluginID, p.Version)
	return in, nil
}

// Uninstall 卸载：先禁用，再要求 Agent 删除版本目录。
func (s *Service) Uninstall(ctx context.Context, installID string) error {
	in, err := s.packages.GetInstall(installID)
	if err != nil {
		return err
	}
	if in.Status == domainpkg.InstallUninstalled {
		return nil
	}
	if s.devices != nil {
		n, err := s.devices.CountByPlugin(in.PluginID)
		if err != nil {
			return err
		}
		if n > 0 {
			return ErrPackageInUse
		}
	}
	agent, err := s.agents.GetAgent(in.AgentID)
	if err != nil || agent == nil {
		return fmt.Errorf("Agent 不存在")
	}
	if agent.Status != "online" {
		return fmt.Errorf("Agent 离线，无法卸载")
	}
	if err := s.callAgent(ctx, agent, in.PluginID, "uninstall", map[string]any{"version": in.Version}); err != nil {
		in.LastError = err.Error()
		_ = s.packages.UpdateInstall(in)
		return err
	}
	in.Status, in.LastError = domainpkg.InstallUninstalled, ""
	if err := s.packages.UpdateInstall(in); err != nil {
		return err
	}
	// 对账：以 Agent 实际版本目录为准，同机同插件的其它记录若已不存在则置为已卸载。
	s.reconcileAgent(ctx, agent, in.PluginID)
	return nil
}

// reconcileAgent 拉取 Agent 上实际存在的版本，修正安装记录状态。
func (s *Service) reconcileAgent(ctx context.Context, agent *domainagent.Agent, pluginID string) {
	versions, err := s.agentVersions(ctx, agent, pluginID)
	if err != nil {
		applog.Warnf("[plugin-pkg] 对账 Agent %s 失败: %v", agent.ID, err)
		return
	}
	list, err := s.packages.ListInstallsByAgent(agent.ID)
	if err != nil {
		return
	}
	for _, item := range list {
		if item.PluginID != pluginID || item.Status != domainpkg.InstallInstalled {
			continue
		}
		if !versions[item.Version] {
			item.Status = domainpkg.InstallUninstalled
			_ = s.packages.UpdateInstall(item)
		}
	}
}

// agentVersions 查询 Agent 上某插件已安装的版本集合。
func (s *Service) agentVersions(ctx context.Context, agent *domainagent.Agent, pluginID string) (map[string]bool, error) {
	url := strings.TrimRight(agent.Endpoint, "/") + "/api/v1/plugins/installed"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if s.token != "" {
		req.Header.Set("X-Agent-Token", s.token)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Agent 返回 %d", resp.StatusCode)
	}
	var out struct {
		Installed []struct {
			PluginID string   `json:"pluginId"`
			Versions []string `json:"versions"`
		} `json:"installed"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, it := range out.Installed {
		if it.PluginID != pluginID {
			continue
		}
		for _, v := range it.Versions {
			set[v] = true
		}
	}
	return set, nil
}

// ListInstalls 返回安装记录。
func (s *Service) ListInstalls(agentID string) ([]*domainpkg.Install, error) {
	if strings.TrimSpace(agentID) != "" {
		return s.packages.ListInstallsByAgent(agentID)
	}
	return s.packages.ListInstalls()
}

func (s *Service) callAgent(ctx context.Context, agent *domainagent.Agent, pluginID, action string, body map[string]any) error {
	raw, _ := json.Marshal(body)
	url := strings.TrimRight(agent.Endpoint, "/") + "/api/v1/plugins/" + pluginID + "/" + action
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("X-Agent-Token", s.token)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("调用 Agent 失败: %w", err)
	}
	defer resp.Body.Close()
	respRaw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("Agent 返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(respRaw)))
	}
	return nil
}

// store 返回当前对象存储；未配置时返回 nil。
func (s *Service) store() Store {
	if s.storeFn == nil {
		return nil
	}
	st := s.storeFn()
	if st == nil {
		return nil
	}
	// noop 驱动视为"未配置"，避免把插件包写进空实现或掩盖错误。
	if st.Provider() == "noop" {
		return nil
	}
	return st
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
