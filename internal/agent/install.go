package agent

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// currentFile 记录某插件当前生效版本（相对 plugin_dir/{pluginId}）。
const currentFile = ".current"

// removedDir 存放「已删除内置插件」的墓碑（相对 plugin_dir）。
// 重建目录（如 dev 每次构建内置插件）后，agent 依据墓碑跳过、不再自动发现。
const removedDir = ".removed"

// isRemoved 判断插件是否已被墓碑标记删除。
func (a *Agent) isRemoved(pluginID string) bool {
	_, err := os.Stat(filepath.Join(a.cfg.PluginDir, removedDir, pluginID))
	return err == nil
}

// markRemoved 写入墓碑；清除墓碑见 clearRemoved。
func (a *Agent) markRemoved(pluginID string) {
	dir := filepath.Join(a.cfg.PluginDir, removedDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, pluginID), []byte("removed by platform\n"), 0o644)
}

// clearRemoved 清除墓碑（重新安装该插件时调用）。
func (a *Agent) clearRemoved(pluginID string) {
	_ = os.Remove(filepath.Join(a.cfg.PluginDir, removedDir, pluginID))
}

// installRequest 是平台下发的安装指令。
type installRequest struct {
	PackageID string `json:"packageId"`
	PluginID  string `json:"pluginId"`
	Version   string `json:"version"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"sizeBytes"`
}

// install 执行"下载 → 校验 → 解压 → 切换 current → 重扫并上报"。
// 全程失败不触碰现网版本：所有写入都发生在临时目录，只有最后 rename 生效。
func (a *Agent) install(ctx context.Context, req installRequest) error {
	req.PluginID = strings.TrimSpace(req.PluginID)
	req.Version = strings.TrimSpace(req.Version)
	if req.PluginID == "" || req.Version == "" {
		return errors.New("pluginId/version 不能为空")
	}
	if !validID(req.PluginID) || !validID(req.Version) {
		return fmt.Errorf("pluginId/version 含非法字符")
	}
	if req.URL == "" {
		return errors.New("下载地址为空")
	}

	pluginRoot := filepath.Join(a.cfg.PluginDir, req.PluginID)
	finalDir := filepath.Join(pluginRoot, req.Version)
	// 幂等：同名版本已存在且 current 指向它，直接返回。
	if cur, _ := readCurrent(pluginRoot); cur == req.Version {
		if _, err := os.Stat(filepath.Join(finalDir, "plugin.yaml")); err == nil {
			a.rescan()
			return nil
		}
	}

	if err := os.MkdirAll(a.cfg.PluginDir, 0o755); err != nil {
		return fmt.Errorf("创建插件目录失败: %w", err)
	}
	tmpDir, err := os.MkdirTemp(a.cfg.PluginDir, ".download-"+req.PluginID+"-")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "pkg.zwplugin")
	if err := a.download(ctx, req.URL, zipPath, req.SHA256, req.SizeBytes); err != nil {
		return err
	}
	stageDir := filepath.Join(tmpDir, "stage")
	if err := extractZip(zipPath, stageDir); err != nil {
		return err
	}
	if err := verifyManifest(stageDir, req.PluginID, req.Version); err != nil {
		return err
	}
	if err := os.MkdirAll(pluginRoot, 0o755); err != nil {
		return err
	}
	// 目标目录存在则先清理（可能是上次中断的残留）。
	if err := os.RemoveAll(finalDir); err != nil {
		return err
	}
	if err := os.Rename(stageDir, finalDir); err != nil {
		return fmt.Errorf("提交版本目录失败: %w", err)
	}
	if err := writeCurrent(pluginRoot, req.Version); err != nil {
		return err
	}
	// 保留最近两个版本，其余清理。
	pruneVersions(pluginRoot, req.Version, 2)
	// 平台重新安装时清除墓碑，允许再次被发现。
	a.clearRemoved(req.PluginID)

	a.rescan()
	a.saveState()
	// 安装后立即上报一次，平台可马上看到清单。
	if err := a.register(ctx); err != nil {
		log.Printf("post-install register failed: %v", err)
	}
	log.Printf("[install] %s@%s installed", req.PluginID, req.Version)
	return nil
}

// uninstall 卸载插件：先停止进程，再删除版本目录；删除当前版本时连带清除 current。
func (a *Agent) uninstall(pluginID, version string) error {
	pluginID = strings.TrimSpace(pluginID)
	version = strings.TrimSpace(version)
	if !validID(pluginID) || !validID(version) {
		return fmt.Errorf("pluginId/version 非法")
	}
	pluginRoot := filepath.Join(a.cfg.PluginDir, pluginID)
	target := filepath.Join(pluginRoot, version)
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("版本目录不存在: %s", version)
	}
	a.mu.Lock()
	p := a.plugins[pluginID]
	a.mu.Unlock()
	if p != nil {
		p.mu.Lock()
		running := p.manifest.Version == version
		p.mu.Unlock()
		if running {
			a.stop(p, "uninstalled by platform")
			p.mu.Lock()
			p.desired = "disabled"
			p.mu.Unlock()
			delete(a.plugins, pluginID)
		}
	}
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("删除版本目录失败: %w", err)
	}
	if cur, _ := readCurrent(pluginRoot); cur == version {
		_ = os.Remove(filepath.Join(pluginRoot, currentFile))
	}
	a.rescan()
	a.saveState()
	log.Printf("[uninstall] %s@%s removed", pluginID, version)
	return nil
}

// download 下载包体并校验 sha256 与体积。
func (a *Agent) download(ctx context.Context, url, dst, wantSHA string, wantSize int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if a.cfg.Token != "" {
		req.Header.Set("X-Agent-Token", a.cfg.Token)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("下载失败 %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), resp.Body)
	if err != nil {
		return fmt.Errorf("下载中断: %w", err)
	}
	if wantSize > 0 && n != wantSize {
		return fmt.Errorf("包体大小不符: 期望 %d 实际 %d", wantSize, n)
	}
	if wantSHA != "" && !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), wantSHA) {
		return fmt.Errorf("sha256 校验失败")
	}
	return nil
}

// extractZip 解压到目标目录，拒绝路径穿越与符号链接。
func extractZip(zipPath, dst string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("不是有效的 zip 包: %w", err)
	}
	defer zr.Close()
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	base := filepath.Clean(dst) + string(os.PathSeparator)
	for _, f := range zr.File {
		name := strings.TrimSpace(f.Name)
		if name == "" {
			continue
		}
		clean := path.Clean(strings.ReplaceAll(name, "\\", "/"))
		if strings.HasPrefix(clean, "../") || clean == ".." || path.IsAbs(clean) || hasDrivePrefix(clean) {
			return fmt.Errorf("条目路径非法: %s", name)
		}
		target := filepath.Join(dst, filepath.FromSlash(clean))
		if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), base) &&
			filepath.Clean(target) != filepath.Clean(dst) {
			return fmt.Errorf("条目越出安装目录: %s", name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if f.FileInfo().Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("不允许符号链接: %s", name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		mode := f.Mode()
		if mode&0o111 != 0 {
			mode = 0o755
		} else {
			mode = 0o644
		}
		w, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			_ = rc.Close()
			return err
		}
		if _, err := io.Copy(w, rc); err != nil {
			_ = rc.Close()
			_ = w.Close()
			return err
		}
		_ = rc.Close()
		if err := w.Close(); err != nil {
			return err
		}
	}
	return nil
}

func hasDrivePrefix(p string) bool {
	return len(p) >= 2 && p[1] == ':'
}

// verifyManifest 校验解压后的清单与平台声明一致。
func verifyManifest(dir, pluginID, version string) error {
	raw, err := os.ReadFile(filepath.Join(dir, "plugin.yaml"))
	if err != nil {
		raw, err = os.ReadFile(filepath.Join(dir, "plugin.yml"))
		if err != nil {
			return fmt.Errorf("包根目录缺少 plugin.yaml")
		}
	}
	var m struct {
		ID      string `yaml:"id"`
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("plugin.yaml 解析失败: %w", err)
	}
	if m.ID != pluginID {
		return fmt.Errorf("清单 id=%s 与安装指令 %s 不一致", m.ID, pluginID)
	}
	if m.Version != version {
		return fmt.Errorf("清单 version=%s 与安装指令 %s 不一致", m.Version, version)
	}
	return nil
}

func validID(s string) bool {
	if s == "" || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if !(r == '-' || r == '.' || r == '_' ||
			(r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func readCurrent(pluginRoot string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(pluginRoot, currentFile))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func writeCurrent(pluginRoot, version string) error {
	if err := os.MkdirAll(pluginRoot, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(pluginRoot, currentFile+".tmp")
	if err := os.WriteFile(tmp, []byte(version), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(pluginRoot, currentFile))
}

// pruneVersions 只保留当前版本与最近的 keep-1 个旧版本。
func pruneVersions(pluginRoot, current string, keep int) {
	entries, err := os.ReadDir(pluginRoot)
	if err != nil {
		return
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			versions = append(versions, e.Name())
		}
	}
	sort.Strings(versions)
	removeCount := len(versions) - keep
	for _, v := range versions {
		if removeCount <= 0 {
			break
		}
		if v == current {
			continue
		}
		if err := os.RemoveAll(filepath.Join(pluginRoot, v)); err == nil {
			removeCount--
		}
	}
}

// source 标记插件来源：平台安装（版本目录 + .current）或手工部署。
func (p *plugin) source() string {
	parent := filepath.Dir(p.dir)
	if _, err := readCurrent(parent); err == nil {
		return "platform"
	}
	return "manual"
}

// removePlugin 删除内置/手工部署插件：先停进程，再删除整个插件目录并重扫。
// 仅用于平台判定「无设备且已禁用」后的清理；平台安装的版本插件走 uninstall。
func (a *Agent) removePlugin(pluginID string) error {
	pluginID = strings.TrimSpace(pluginID)
	if !validID(pluginID) {
		return fmt.Errorf("pluginId 非法")
	}
	root := filepath.Join(a.cfg.PluginDir, pluginID)
	if _, err := os.Stat(root); err != nil {
		return fmt.Errorf("插件目录不存在: %s", pluginID)
	}
	a.mu.Lock()
	p := a.plugins[pluginID]
	a.mu.Unlock()
	if p != nil {
		a.stop(p, "removed by platform")
		p.mu.Lock()
		p.desired = "disabled"
		p.mu.Unlock()
	}
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("删除插件目录失败: %w", err)
	}
	a.markRemoved(pluginID)
	a.rescan()
	a.saveState()
	log.Printf("[remove] %s deleted", pluginID)
	return nil
}

// rescan 重新加载插件目录，并让既有进程状态保持不变。
func (a *Agent) rescan() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.discoverLocked(); err != nil {
		log.Printf("rescan: %v", err)
	}
}
