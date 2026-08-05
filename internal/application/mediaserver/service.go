package mediaserver

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"zero-web-server/internal/infrastructure/config"
	"zero-web-server/internal/infrastructure/media/mediakit"
	"zero-web-server/internal/infrastructure/persistence"
	"zero-web-server/internal/infrastructure/persistence/model"
)

// View 媒体节点列表/详情视图。
type View struct {
	ID            string `json:"id"`
	IP            string `json:"ip"`
	HookIP        string `json:"hookIp"`
	SDPIP         string `json:"sdpIp"`
	StreamIP      string `json:"streamIp"`
	HTTPPort      int    `json:"httpPort"`
	HTTPSSLPort   int    `json:"httpSSlPort"`
	RTMPPort      int    `json:"rtmpPort"`
	RTSPPort      int    `json:"rtspPort"`
	RTPProxyPort  int    `json:"rtpProxyPort"`
	Secret        string `json:"secret"`
	Type          string `json:"type"`
	DefaultServer bool   `json:"defaultServer"`
	Online        bool   `json:"status"`
	Load          int64  `json:"load"`
	CreateTime    string `json:"createTime"`
}

// Node 选中的媒体节点（开流/拉流用）。
type Node struct {
	Model  model.MediaServer
	Client *mediakit.Client
}

func (n *Node) ID() string { return n.Model.ID }

func (n *Node) SDPIP() string {
	if n.Model.SDPIP != "" {
		return n.Model.SDPIP
	}
	if n.Model.StreamIP != "" {
		return n.Model.StreamIP
	}
	return n.Model.IP
}

func (n *Node) StreamIP() string {
	if n.Model.StreamIP != "" {
		return n.Model.StreamIP
	}
	return n.Model.IP
}

func (n *Node) MediaConfig() config.MediaConfig {
	return ToMediaConfig(n.Model)
}

func ToMediaConfig(m model.MediaServer) config.MediaConfig {
	return config.MediaConfig{
		ID:       m.ID,
		Type:     m.Type,
		IP:       firstNonEmpty(m.StreamIP, m.IP),
		HTTPPort: m.HTTPPort,
		Secret:   m.Secret,
	}
}

type Service struct {
	repo     *persistence.MediaServerRepository
	serverID string

	mu         sync.RWMutex
	load       map[string]int64  // nodeID -> 活跃流计数（负载）
	streamNode map[string]string // app/stream -> nodeID
	onlineAt   map[string]time.Time
	onlineOK   map[string]bool
}

const onlineProbeTTL = 3 * time.Second
const onlineProbeTimeout = 1500 * time.Millisecond

func NewService(repo *persistence.MediaServerRepository, serverID string) *Service {
	s := &Service{
		repo:       repo,
		serverID:   serverID,
		load:       make(map[string]int64),
		streamNode: make(map[string]string),
		onlineAt:   make(map[string]time.Time),
		onlineOK:   make(map[string]bool),
	}
	// 历史「配置默认节点」标记清掉，全部改为可增删改的库管节点
	_ = repo.ClearAllDefault()
	return s
}

// List 返回数据库中的节点；启动时空列表是正常状态。
func (s *Service) List() ([]View, error) {
	rows, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	out := make([]View, len(rows))
	var wg sync.WaitGroup
	for i := range rows {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			row := rows[i]
			out[i] = s.toView(row, s.isOnline(row))
		}(i)
	}
	wg.Wait()
	return out, nil
}

func (s *Service) ListOnline() ([]View, error) {
	all, err := s.List()
	if err != nil {
		return nil, err
	}
	online := make([]View, 0, len(all))
	for _, v := range all {
		if v.Online {
			online = append(online, v)
		}
	}
	return online, nil
}

func (s *Service) GetOne(id string) (*View, error) {
	m, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("流媒体节点不存在")
	}
	v := s.toView(*m, s.isOnline(*m))
	return &v, nil
}

// Check 探测节点连通性（secret 可选，暂不强制）。
func (s *Service) Check(ip string, port int, secret string) (*View, error) {
	if ip == "" || port <= 0 {
		return nil, fmt.Errorf("IP 与 HTTP 端口不能为空")
	}
	client := mediakit.NewClientAddr(ip, port, secret)
	if err := client.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}
	id := fmt.Sprintf("%s:%d", ip, port)
	return &View{
		ID:           id,
		IP:           ip,
		HookIP:       ip,
		SDPIP:        ip,
		StreamIP:     ip,
		HTTPPort:     port,
		HTTPSSLPort:  443,
		RTMPPort:     1935,
		RTSPPort:     8554,
		RTPProxyPort: 10000,
		Secret:       secret,
		Type:         "zms",
		Online:       true,
	}, nil
}

func (s *Service) Save(m *model.MediaServer) error {
	if m == nil {
		return fmt.Errorf("参数为空")
	}
	m.IP = strings.TrimSpace(m.IP)
	if m.IP == "" || m.HTTPPort <= 0 {
		return fmt.Errorf("IP 与 HTTP 端口不能为空")
	}
	if m.ID == "" {
		m.ID = fmt.Sprintf("%s:%d", m.IP, m.HTTPPort)
	}
	if m.Type == "" {
		m.Type = "zms"
	}
	m.Type = normalizeType(m.Type)
	if m.HookIP == "" {
		m.HookIP = m.IP
	}
	if m.SDPIP == "" {
		m.SDPIP = m.IP
	}
	if m.StreamIP == "" {
		m.StreamIP = m.IP
	}
	if m.RTMPPort <= 0 {
		m.RTMPPort = 1935
	}
	if m.RTSPPort <= 0 {
		m.RTSPPort = 8554
	}
	if m.ServerID == "" {
		m.ServerID = s.serverID
	}
	// 不再有「配置文件默认节点」概念，全部由库管理
	m.DefaultServer = false

	client := mediakit.NewClientAddr(m.IP, m.HTTPPort, m.Secret)
	if err := client.Ping(context.Background()); err != nil {
		return fmt.Errorf("节点不可达，请先测试连通: %w", err)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	m.UpdateTime = now
	if existing, err := s.repo.GetByID(m.ID); err == nil && existing != nil {
		m.CreateTime = existing.CreateTime
	} else {
		m.CreateTime = now
	}
	return s.repo.Save(m)
}

func (s *Service) Delete(id string) error {
	if id == "" {
		return fmt.Errorf("id 不能为空")
	}
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.load, id)
	for k, nid := range s.streamNode {
		if nid == id {
			delete(s.streamNode, k)
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) MediaInfo(app, stream, mediaServerID string) (map[string]any, error) {
	node, err := s.Resolve(mediaServerID)
	if err != nil {
		return nil, err
	}
	info := node.Client.LookupStreamMediaInfo(context.Background(), app, stream)
	if info == nil {
		return nil, fmt.Errorf("流不存在或已离线")
	}
	fps := any(info.VideoFps)
	if info.Fps > 0 {
		fps = info.Fps
	}
	var audioSampleRate any
	if info.SampleRate > 0 {
		audioSampleRate = info.SampleRate
	}
	var channels any
	if info.Channels > 0 {
		channels = info.Channels
	}
	return map[string]any{
		"app":             info.App,
		"stream":          info.Stream,
		"schema":          info.Schema,
		"videoCodec":      info.VideoCodec,
		"audioCodec":      info.AudioCodec,
		"readerCount":     info.ReaderCount,
		"aliveSecond":     info.AliveSecond,
		"bytesSpeed":      info.BytesSpeed,
		"fps":             fps,
		"width":           info.Width,
		"height":          info.Height,
		"audioSampleRate": audioSampleRate,
		"channels":        channels,
		"originType":      info.OriginType,
		"originTypeStr":   info.OriginTypeStr,
		"mediaServerId":   node.ID(),
	}, nil
}

func (s *Service) Load() ([]View, error) {
	return s.ListOnline()
}

// Lookup 按 ID 取节点配置，不探测在线。空 ID 优先默认节点，否则取列表首个。
func (s *Service) Lookup(id string) (*Node, error) {
	id = strings.TrimSpace(id)
	if id != "" && !strings.EqualFold(id, "auto") {
		m, err := s.repo.GetByID(id)
		if err != nil {
			return nil, fmt.Errorf("流媒体节点不存在: %s", id)
		}
		return s.wrap(*m), nil
	}
	rows, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("未配置媒体节点，请先在「媒体管理 → 媒体节点」添加")
	}
	var fallback *model.MediaServer
	for i := range rows {
		row := rows[i]
		if row.DefaultServer {
			return s.wrap(row), nil
		}
		if fallback == nil {
			cp := row
			fallback = &cp
		}
	}
	return s.wrap(*fallback), nil
}

// Resolve 按 preferID 选节点：空/"auto" → 在线最小负载；指定 ID → 该节点（须在线）。
func (s *Service) Resolve(preferID string) (*Node, error) {
	preferID = strings.TrimSpace(preferID)
	if preferID == "" || strings.EqualFold(preferID, "auto") {
		return s.SelectMinimumLoad()
	}
	m, err := s.repo.GetByID(preferID)
	if err != nil {
		return nil, fmt.Errorf("流媒体节点不存在: %s", preferID)
	}
	if !s.isOnline(*m) {
		return nil, fmt.Errorf("流媒体节点离线: %s", preferID)
	}
	return s.wrap(*m), nil
}

// SelectMinimumLoad 在线节点中负载最低者。
func (s *Service) SelectMinimumLoad() (*Node, error) {
	rows, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	var best *model.MediaServer
	var bestLoad int64 = -1
	for i := range rows {
		row := rows[i]
		if !s.isOnline(row) {
			continue
		}
		load := s.GetLoad(row.ID)
		if best == nil || load < bestLoad {
			cp := row
			best = &cp
			bestLoad = load
		}
	}
	if best == nil {
		return nil, fmt.Errorf("无可用媒体节点，请先在「媒体管理 → 媒体节点」添加并确保在线")
	}
	return s.wrap(*best), nil
}

// FirstOnlineBaseURL WebRTC 代理等场景取任一在线节点 HTTP 根地址。
func (s *Service) FirstOnlineBaseURL() string {
	node, err := s.SelectMinimumLoad()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", node.StreamIP(), node.Model.HTTPPort)
}

func (s *Service) wrap(m model.MediaServer) *Node {
	return &Node{
		Model:  m,
		Client: mediakit.NewClientAddr(m.IP, m.HTTPPort, m.Secret),
	}
}

func (s *Service) ClientForID(id string) (*mediakit.Client, error) {
	m, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return mediakit.NewClientAddr(m.IP, m.HTTPPort, m.Secret), nil
}

func (s *Service) BindStream(app, stream, nodeID string) {
	key := streamKey(app, stream)
	s.mu.Lock()
	prev, had := s.streamNode[key]
	s.streamNode[key] = nodeID
	if !had || prev != nodeID {
		s.load[nodeID]++
		if had && prev != nodeID {
			s.decLocked(prev)
		}
	}
	s.mu.Unlock()
}

func (s *Service) UnbindStream(app, stream string) {
	key := streamKey(app, stream)
	s.mu.Lock()
	if nid, ok := s.streamNode[key]; ok {
		delete(s.streamNode, key)
		s.decLocked(nid)
	}
	s.mu.Unlock()
}

func (s *Service) StreamNodeID(app, stream string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.streamNode[streamKey(app, stream)]
	return id, ok
}

// ResolveForStream 优先复用已绑定节点，否则按 prefer 选新节点。
func (s *Service) ResolveForStream(app, stream, preferID string) (*Node, error) {
	if id, ok := s.StreamNodeID(app, stream); ok {
		if node, err := s.Resolve(id); err == nil {
			return node, nil
		}
		s.UnbindStream(app, stream)
	}
	return s.Resolve(preferID)
}

func (s *Service) GetLoad(id string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.load[id]
}

func (s *Service) IncLoad(id string) {
	s.mu.Lock()
	s.load[id]++
	s.mu.Unlock()
}

func (s *Service) DecLoad(id string) {
	s.mu.Lock()
	s.decLocked(id)
	s.mu.Unlock()
}

func (s *Service) decLocked(id string) {
	if s.load[id] > 0 {
		s.load[id]--
	} else {
		delete(s.load, id)
	}
}

func (s *Service) isOnline(m model.MediaServer) bool {
	s.mu.RLock()
	if t, ok := s.onlineAt[m.ID]; ok && time.Since(t) < onlineProbeTTL {
		v := s.onlineOK[m.ID]
		s.mu.RUnlock()
		return v
	}
	s.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), onlineProbeTimeout)
	defer cancel()
	client := mediakit.NewClientAddr(m.IP, m.HTTPPort, m.Secret)
	ok := client.Ping(ctx) == nil

	s.mu.Lock()
	s.onlineAt[m.ID] = time.Now()
	s.onlineOK[m.ID] = ok
	s.mu.Unlock()
	return ok
}

func (s *Service) toView(m model.MediaServer, online bool) View {
	return View{
		ID: m.ID, IP: m.IP, HookIP: m.HookIP, SDPIP: m.SDPIP, StreamIP: m.StreamIP,
		HTTPPort: m.HTTPPort, HTTPSSLPort: m.HTTPSSLPort,
		RTMPPort: m.RTMPPort, RTSPPort: m.RTSPPort, RTPProxyPort: m.RTPProxyPort,
		Secret: m.Secret, Type: m.Type, DefaultServer: m.DefaultServer,
		Online: online, Load: s.GetLoad(m.ID), CreateTime: m.CreateTime,
	}
}

func streamKey(app, stream string) string {
	return app + "/" + stream
}

func normalizeType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "zms", "zero-media-server", "zeromediaserver", "zeromediakit", "mediakit", "zlm":
		return "zms"
	case "abl":
		return "abl"
	default:
		return "zms"
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
