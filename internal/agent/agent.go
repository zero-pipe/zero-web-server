// Package agent implements the zero-agent host service: it discovers plugins,
// supervises their processes, reports state to the platform, and reverse-proxies
// platform RPC calls to the plugin's local HTTP endpoint.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 是 zero-agent 的本地配置。
type Config struct {
	AgentID      string `yaml:"agent_id"`
	PlatformURL  string `yaml:"platform_url"`
	Token        string `yaml:"token"`
	Endpoint     string `yaml:"endpoint"`
	Listen       string `yaml:"listen"`
	PluginDir    string `yaml:"plugin_dir"`
	StateFile    string `yaml:"state_file"`
	Heartbeat    string `yaml:"heartbeat"`
	StartTimeout string `yaml:"start_timeout"`
}

// Manifest 是插件目录中的 plugin.yaml。
type Manifest struct {
	ID               string            `yaml:"id"`
	InstanceID       string            `yaml:"instance_id"`
	Name             string            `yaml:"name"`
	Vendor           string            `yaml:"vendor"`
	Version          string            `yaml:"version"`
	Protocols        []string          `yaml:"protocols"`
	Executable       map[string]string `yaml:"executable"`
	Args             []string          `yaml:"args"`
	Listen           string            `yaml:"listen"`
	Health           string            `yaml:"health"`
	Capabilities     []string          `yaml:"capabilities"`
	StreamTransports []string          `yaml:"stream_transports"`
	EventTypes       []string          `yaml:"event_types"`
	ConfigSchema     string            `yaml:"config_schema"`
	Env              map[string]string `yaml:"env"`
	Enabled          *bool             `yaml:"enabled"`
}

type plugin struct {
	mu       sync.Mutex
	dir      string
	manifest Manifest
	cmd      *exec.Cmd
	exited   bool
	stopping bool
	desired  string
	actual   string
	lastErr  string
	restart  time.Time
}

// Agent 是 zero-agent 运行时。
type Agent struct {
	cfg       Config
	mu        sync.RWMutex
	plugins   map[string]*plugin // key: manifest.id
	client    *http.Client
	hostname  string
	localIP   string
	startedAt time.Time
	startWait time.Duration
}

// New 创建 Agent 并加载插件清单。
func New(cfg Config) (*Agent, error) {
	cfg.AgentID = strings.TrimSpace(cfg.AgentID)
	cfg.PlatformURL = strings.TrimRight(strings.TrimSpace(cfg.PlatformURL), "/")
	cfg.Listen = strings.TrimSpace(cfg.Listen)
	cfg.PluginDir = strings.TrimSpace(cfg.PluginDir)
	if cfg.AgentID == "" {
		h, _ := os.Hostname()
		cfg.AgentID = h
	}
	if cfg.PlatformURL == "" {
		return nil, fmt.Errorf("platform_url 不能为空")
	}
	if cfg.Listen == "" {
		cfg.Listen = "0.0.0.0:19000"
	}
	if cfg.Endpoint == "" {
		_, port, _ := net.SplitHostPort(cfg.Listen)
		cfg.Endpoint = "http://127.0.0.1:" + port
	}
	cfg.Endpoint = strings.TrimRight(cfg.Endpoint, "/")
	if cfg.PluginDir == "" {
		cfg.PluginDir = "plugins"
	}
	if cfg.StateFile == "" {
		cfg.StateFile = filepath.Join("data", "state.json")
	}
	abs, err := filepath.Abs(cfg.PluginDir)
	if err != nil {
		return nil, err
	}
	cfg.PluginDir = abs
	h, _ := os.Hostname()
	a := &Agent{
		cfg: cfg, plugins: map[string]*plugin{}, hostname: h,
		client: &http.Client{Timeout: 10 * time.Second}, startedAt: time.Now(),
		startWait: 15 * time.Second,
	}
	if d, err := time.ParseDuration(cfg.StartTimeout); err == nil && d > 0 {
		a.startWait = d
	}
	a.localIP = localIP(cfg.PlatformURL)
	a.loadState()
	if err := a.discover(); err != nil {
		return nil, err
	}
	return a, nil
}

// Run 启动 HTTP 服务、插件进程、注册与心跳。
func (a *Agent) Run(ctx context.Context) error {
	for _, p := range a.snapshot() {
		if p.desired == "enabled" {
			a.start(p)
		}
	}
	go a.watchdog(ctx)
	go a.registerLoop(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "agentId": a.cfg.AgentID})
	})
	mux.HandleFunc("/api/v1/status", a.auth(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"plugins": a.report()})
	}))
	mux.HandleFunc("/api/v1/plugins/", a.auth(a.handlePluginAction))
	mux.HandleFunc("/api/v1/plugins/installed", a.auth(a.handleInstalled))
	mux.HandleFunc("/plugins/", a.auth(a.handleProxy))

	srv := &http.Server{Addr: a.cfg.Listen, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	log.Printf("zero-agent %s listening on %s, endpoint=%s, plugins=%d", a.cfg.AgentID, a.cfg.Listen, a.cfg.Endpoint, len(a.plugins))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (a *Agent) discover() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.discoverLocked()
}

// pluginRoots 返回插件清单所在目录：兼容两种布局
//   - 手工部署：plugin_dir/<plugin>/plugin.yaml
//   - 平台安装：plugin_dir/<plugin>/<version>/plugin.yaml（以 .current 指向生效版本）
func (a *Agent) pluginRoots() []string {
	var roots []string
	entries, err := os.ReadDir(a.cfg.PluginDir)
	if err != nil {
		return roots
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		// 已删除的内置插件：保留墓碑，重建目录后也不再自动发现。
		if a.isRemoved(e.Name()) {
			continue
		}
		base := filepath.Join(a.cfg.PluginDir, e.Name())
		root := base
		if cur, err := readCurrent(base); err == nil && cur != "" {
			if st, err := os.Stat(filepath.Join(base, cur)); err == nil && st.IsDir() {
				root = filepath.Join(base, cur)
			}
		}
		roots = append(roots, root)
	}
	return roots
}

// discoverLocked 重新扫描插件根目录；调用方必须持有 a.mu 写锁。
// 同版本同路径的插件保留原运行态，避免重扫导致页面闪断。
func (a *Agent) discoverLocked() error {
	seen := map[string]bool{}
	roots := a.pluginRoots()
	for _, root := range roots {
		var file string
		for _, fname := range []string{"plugin.yaml", "plugin.yml"} {
			candidate := filepath.Join(root, fname)
			if _, err := os.Stat(candidate); err == nil {
				file = candidate
				break
			}
		}
		if file == "" {
			continue
		}
		dir := root
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("read manifest %s: %v", file, err)
			continue
		}
		var m Manifest
		if err := yaml.Unmarshal(data, &m); err != nil {
			log.Printf("parse manifest %s: %v", file, err)
			continue
		}
		m.ID = strings.TrimSpace(m.ID)
		if m.ID == "" {
			log.Printf("manifest %s: id is required", file)
			continue
		}
		if seen[m.ID] {
			log.Printf("manifest %s: duplicate id %s", file, m.ID)
			continue
		}
		if m.InstanceID == "" {
			m.InstanceID = a.cfg.AgentID + "-" + m.ID
		}
		if m.Listen == "" {
			m.Listen = "127.0.0.1:0"
		}
		if m.ConfigSchema != "" && !strings.HasPrefix(strings.TrimSpace(m.ConfigSchema), "{") {
			raw, err := os.ReadFile(filepath.Join(dir, m.ConfigSchema))
			if err != nil {
				log.Printf("manifest %s: read config schema: %v", file, err)
			} else {
				m.ConfigSchema = string(raw)
			}
		}
		seen[m.ID] = true
		if old, ok := a.plugins[m.ID]; ok && old.dir == dir && old.manifest.Version == m.Version {
			old.mu.Lock()
			old.manifest = m
			old.mu.Unlock()
			continue
		}
		p := &plugin{dir: dir, manifest: m, desired: "enabled", actual: "discovered"}
		if m.Enabled != nil && !*m.Enabled {
			p.desired = "disabled"
		}
		if state, ok := a.savedState()[m.ID]; ok {
			p.desired = state
		}
		if p.desired == "disabled" {
			p.actual = "stopped"
		}
		a.plugins[m.ID] = p
	}
	// 扫描不到的插件（版本切换/卸载）：停进程并移除。
	for id, old := range a.plugins {
		if seen[id] {
			continue
		}
		old.mu.Lock()
		running := old.cmd != nil && !old.exited
		old.mu.Unlock()
		if running {
			go a.stop(old, "manifest removed")
		}
		delete(a.plugins, id)
	}
	return nil
}

func (a *Agent) snapshot() []*plugin {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]*plugin, 0, len(a.plugins))
	for _, p := range a.plugins {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].manifest.ID < out[j].manifest.ID })
	return out
}

func (a *Agent) get(id string) *plugin {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.plugins[id]
}

func (a *Agent) start(p *plugin) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil && !p.exited {
		return
	}
	if time.Now().Before(p.restart) {
		return
	}
	// 规范键优先 goos-goarch；兼容旧的仅 goos 写法与 default。
	exe := p.manifest.Executable[runtime.GOOS+"-"+runtime.GOARCH]
	if exe == "" {
		exe = p.manifest.Executable[runtime.GOOS]
	}
	if exe == "" {
		exe = p.manifest.Executable["default"]
	}
	if exe == "" {
		p.actual, p.lastErr = "error", "manifest 未提供当前平台可执行文件"
		return
	}
	if !filepath.IsAbs(exe) {
		exe = filepath.Join(p.dir, exe)
	}
	cmd := exec.Command(exe, p.manifest.Args...)
	cmd.Dir = p.dir
	cmd.Env = append(os.Environ(),
		"PLUGIN_LISTEN="+p.manifest.Listen,
		"PLUGIN_ID="+p.manifest.ID,
		"PLUGIN_INSTANCE_ID="+p.manifest.InstanceID,
		"PLUGIN_TOKEN="+a.cfg.Token,
		"AGENT_RPC_URL="+a.cfg.Endpoint,
	)
	for k, v := range p.manifest.Env {
		cmd.Env = append(cmd.Env, k+"="+strings.ReplaceAll(v, "${PLUGIN_DIR}", p.dir))
	}
	logFile := a.logFile(p.manifest.ID)
	if logFile != nil {
		cmd.Stdout, cmd.Stderr = logFile, logFile
	}
	if err := cmd.Start(); err != nil {
		p.actual, p.lastErr = "error", err.Error()
		p.restart = time.Now().Add(5 * time.Second)
		return
	}
	p.cmd, p.exited, p.stopping = cmd, false, false
	p.actual, p.lastErr = "starting", ""
	log.Printf("[plugin] %s starting pid=%d", p.manifest.ID, cmd.Process.Pid)
	go a.waitReady(p)
	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		p.exited = true
		if p.stopping {
			p.actual = "stopped"
		} else if err != nil {
			p.actual, p.lastErr = "error", err.Error()
			p.restart = time.Now().Add(5 * time.Second)
			log.Printf("[plugin] %s exited: %v", p.manifest.ID, err)
		} else {
			p.actual = "stopped"
		}
		logFile := a.logFile(p.manifest.ID)
		if logFile != nil {
			_ = logFile.Close()
		}
		p.mu.Unlock()
	}()
}

func (a *Agent) waitReady(p *plugin) {
	deadline := time.Now().Add(a.startWait)
	for time.Now().Before(deadline) {
		if probe(p.manifest.Listen) {
			p.mu.Lock()
			if !p.exited && !p.stopping {
				p.actual, p.lastErr = "running", ""
				log.Printf("[plugin] %s ready", p.manifest.ID)
			}
			p.mu.Unlock()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	p.mu.Lock()
	if !p.exited && !p.stopping {
		p.actual, p.lastErr = "error", "plugin did not become ready"
	}
	p.mu.Unlock()
}

func (a *Agent) stop(p *plugin, reason string) {
	p.mu.Lock()
	p.stopping = true
	cmd := p.cmd
	p.actual = "stopped"
	p.lastErr = reason
	p.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		select {
		case <-time.After(3 * time.Second):
		case <-waitDone(p):
		}
	}
}

func waitDone(p *plugin) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		for {
			p.mu.Lock()
			done := p.exited
			p.mu.Unlock()
			if done {
				close(ch)
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()
	return ch
}

func (a *Agent) restartPlugin(p *plugin) {
	p.mu.Lock()
	p.restart = time.Time{}
	p.stopping = false
	p.actual = "starting"
	p.mu.Unlock()
	a.stop(p, "")
	p.mu.Lock()
	p.restart = time.Time{}
	p.stopping = false
	p.mu.Unlock()
	a.start(p)
}

func (a *Agent) watchdog(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, p := range a.snapshot() {
				p.mu.Lock()
				desired, exited, restartAt, listen := p.desired, p.exited, p.restart, p.manifest.Listen
				neverStarted := p.cmd == nil
				p.mu.Unlock()
				if desired != "enabled" {
					continue
				}
				if (exited || neverStarted) && !time.Now().Before(restartAt) {
					a.start(p)
					continue
				}
				if !exited && !probe(listen) {
					p.mu.Lock()
					failing := p.actual == "running"
					p.actual = "starting"
					p.mu.Unlock()
					if failing {
						a.restartPlugin(p)
					}
				}
			}
		}
	}
}

func probe(listen string) bool {
	_, port, err := net.SplitHostPort(listen)
	if err != nil {
		return false
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", port), time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (a *Agent) handlePluginAction(w http.ResponseWriter, r *http.Request) {
	// /api/v1/plugins/{id}/{action}
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/plugins/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || r.Method != http.MethodPost {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	pluginID, action := parts[0], parts[1]
	// install/uninstall 针对"尚不存在或待移除"的插件，先于查找处理。
	switch action {
	case "install":
		var req installRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.PluginID == "" {
			req.PluginID = pluginID
		}
		if err := a.install(r.Context(), req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"plugins": a.report()})
		return
	case "uninstall":
		var req struct {
			Version string `json:"version"`
		}
		_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req)
		if err := a.uninstall(pluginID, req.Version); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"plugins": a.report()})
		return
	case "remove":
		// 删除内置/手工部署插件：停进程并删除整个插件目录。
		if err := a.removePlugin(pluginID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"plugins": a.report()})
		return
	}
	p := a.get(pluginID)
	if p == nil {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return
	}
	switch action {
	case "enable":
		a.setDesired(p, "enabled")
		a.start(p)
	case "disable":
		a.setDesired(p, "disabled")
		a.stop(p, "disabled by platform")
	case "restart":
		a.setDesired(p, "enabled")
		a.restartPlugin(p)
	default:
		http.Error(w, "unsupported action", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plugins": a.report()})
}

// handleInstalled 列出平台安装的版本（供平台对账）。
func (a *Agent) handleInstalled(w http.ResponseWriter, r *http.Request) {
	type entry struct {
		PluginID string   `json:"pluginId"`
		Current  string   `json:"current"`
		Versions []string `json:"versions"`
	}
	var out []entry
	entries, _ := os.ReadDir(a.cfg.PluginDir)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		base := filepath.Join(a.cfg.PluginDir, e.Name())
		var versions []string
		subs, _ := os.ReadDir(base)
		for _, sub := range subs {
			if sub.IsDir() && !strings.HasPrefix(sub.Name(), ".") {
				versions = append(versions, sub.Name())
			}
		}
		if len(versions) == 0 {
			continue // 手工部署布局没有版本目录
		}
		sort.Strings(versions)
		cur, _ := readCurrent(base)
		out = append(out, entry{PluginID: e.Name(), Current: cur, Versions: versions})
	}
	writeJSON(w, http.StatusOK, map[string]any{"installed": out})
}

func (a *Agent) setDesired(p *plugin, state string) {
	p.mu.Lock()
	p.desired = state
	p.mu.Unlock()
	a.saveState()
}

func (a *Agent) handleProxy(w http.ResponseWriter, r *http.Request) {
	// /plugins/{id}/{path...}
	rest := strings.TrimPrefix(r.URL.Path, "/plugins/")
	id, path, ok := strings.Cut(rest, "/")
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	p := a.get(id)
	if p == nil {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return
	}
	p.mu.Lock()
	listen, actual := p.manifest.Listen, p.actual
	p.mu.Unlock()
	if actual != "running" {
		http.Error(w, "plugin not running", http.StatusServiceUnavailable)
		return
	}
	target, err := url.Parse("http://" + listen)
	if err != nil {
		http.Error(w, "bad plugin endpoint", http.StatusInternalServerError)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	original := proxy.Director
	proxy.Director = func(req *http.Request) {
		original(req)
		req.URL.Path = "/" + path
		if req.URL.RawQuery == "" && r.URL.RawQuery != "" {
			req.URL.RawQuery = r.URL.RawQuery
		}
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "plugin unavailable: "+err.Error(), http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}

func (a *Agent) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.cfg.Token != "" {
			token := r.Header.Get("X-Agent-Token")
			if token == "" {
				token = r.Header.Get("X-Plugin-Token")
			}
			if token != a.cfg.Token {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func (a *Agent) registerLoop(ctx context.Context) {
	for {
		if err := a.register(ctx); err != nil {
			log.Printf("register failed (retry in 3s): %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}
		log.Printf("registered with platform: %s", a.cfg.AgentID)
		break
	}
	heartbeat := 10 * time.Second
	if d, err := time.ParseDuration(a.cfg.Heartbeat); err == nil && d > 0 {
		heartbeat = d
	}
	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.heartbeat(ctx); err != nil {
				log.Printf("heartbeat failed: %v", err)
			}
		}
	}
}

func (a *Agent) register(ctx context.Context) error {
	raw, err := a.post(ctx, "/api/v1/agents/register", a.payload())
	if err != nil {
		return err
	}
	a.reconcile(raw)
	return nil
}

func (a *Agent) heartbeat(ctx context.Context) error {
	raw, err := a.post(ctx, "/api/v1/agents/heartbeat", a.payload())
	if err != nil {
		return err
	}
	a.reconcile(raw)
	return nil
}

// reconcile 让本地进程状态服从平台下发的 desiredState。
func (a *Agent) reconcile(raw []byte) {
	if len(raw) == 0 {
		return
	}
	var resp struct {
		Plugins []struct {
			PluginID     string `json:"pluginId"`
			DesiredState string `json:"desiredState"`
		} `json:"plugins"`
	}
	if json.Unmarshal(raw, &resp) != nil {
		return
	}
	want := map[string]string{}
	for _, p := range resp.Plugins {
		want[p.PluginID] = p.DesiredState
	}
	for _, p := range a.snapshot() {
		desired, ok := want[p.manifest.ID]
		if !ok {
			continue
		}
		p.mu.Lock()
		current := p.desired
		p.mu.Unlock()
		if desired == current {
			continue
		}
		switch desired {
		case "enabled":
			a.setDesired(p, "enabled")
			a.start(p)
		case "disabled":
			a.setDesired(p, "disabled")
			a.stop(p, "disabled by platform")
		}
	}
}

func (a *Agent) payload() map[string]any {
	return map[string]any{
		"agentId": a.cfg.AgentID, "hostname": a.hostname, "ip": a.localIP,
		"version": "0.1.0", "endpoint": a.cfg.Endpoint, "plugins": a.report(),
	}
}

func (a *Agent) report() []map[string]any {
	out := make([]map[string]any, 0, len(a.plugins))
	for _, p := range a.snapshot() {
		p.mu.Lock()
		out = append(out, map[string]any{
			"instanceId": p.manifest.InstanceID, "pluginId": p.manifest.ID,
			"name": p.manifest.Name, "vendor": p.manifest.Vendor, "version": p.manifest.Version,
			"protocols": p.manifest.Protocols, "capabilities": p.manifest.Capabilities,
			"streamTransports": p.manifest.StreamTransports, "eventTypes": p.manifest.EventTypes,
			"configSchema": p.manifest.ConfigSchema, "enabled": p.desired == "enabled",
			"desiredState": p.desired, "actualState": p.actual, "lastError": p.lastErr,
			"source": p.source(),
		})
		p.mu.Unlock()
	}
	return out
}

func (a *Agent) post(ctx context.Context, path string, body any) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.PlatformURL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.cfg.Token != "" {
		req.Header.Set("X-Agent-Token", a.cfg.Token)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rawResp, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(rawResp)))
	}
	return rawResp, nil
}

func (a *Agent) logFile(pluginID string) *os.File {
	dir := filepath.Join(filepath.Dir(a.cfg.StateFile), "..", "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil
	}
	f, err := os.OpenFile(filepath.Join(dir, pluginID+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil
	}
	return f
}

func (a *Agent) savedState() map[string]string {
	raw, err := os.ReadFile(a.cfg.StateFile)
	if err != nil {
		return map[string]string{}
	}
	var state struct {
		Plugins map[string]string `json:"plugins"`
	}
	if json.Unmarshal(raw, &state) != nil || state.Plugins == nil {
		return map[string]string{}
	}
	return state.Plugins
}

func (a *Agent) loadState() { /* 状态文件按需读取，保留方法便于后续扩展 */ }

func (a *Agent) saveState() {
	state := map[string]string{}
	for _, p := range a.snapshot() {
		p.mu.Lock()
		state[p.manifest.ID] = p.desired
		p.mu.Unlock()
	}
	if err := os.MkdirAll(filepath.Dir(a.cfg.StateFile), 0o755); err != nil {
		return
	}
	raw, _ := json.MarshalIndent(map[string]any{"plugins": state}, "", "  ")
	_ = os.WriteFile(a.cfg.StateFile, raw, 0o644)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func localIP(platformURL string) string {
	u, err := url.Parse(platformURL)
	if err != nil || u.Host == "" {
		return ""
	}
	conn, err := net.DialTimeout("udp", u.Host, time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	return ""
}
