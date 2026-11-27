package storagenode

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	domainstoragenode "zero-web-server/internal/domain/storagenode"
	applog "zero-web-server/pkg/log"
)

// RegisterRequest 是 zstorage on_storage_started/keepalive 上报体。
type RegisterRequest struct {
	MediaServerID string `json:"mediaServerId"`
	Type          string `json:"type"`
	Version       string `json:"version"`
	IP            string `json:"ip"`
	HookIP        string `json:"hookIp"`
	StreamIP      string `json:"streamIp"`
	Secret        string `json:"secret"`
	HTTPPort      int    `json:"httpPort"`
	RTSPPort      int    `json:"rtspPort"`
	CapacityGB    int    `json:"capacityGb"`
	UsedGB        int    `json:"usedGb"`
	Slots         int    `json:"slots"`
	FreeSlots     int    `json:"freeSlots"`
}

// Service 管理云存储节点（独立表，不参与取流节点选择）。
type Service struct {
	repo     domainstoragenode.Repository
	serverID string
	mu       sync.RWMutex
}

func NewService(repo domainstoragenode.Repository, serverID string) *Service {
	return &Service{repo: repo, serverID: serverID}
}

// Register 幂等 upsert 节点，并标记在线。返回规范化 node_id。
func (s *Service) Register(ctx context.Context, req RegisterRequest) (string, error) {
	_ = ctx
	ip := strings.TrimSpace(req.StreamIP)
	if ip == "" {
		ip = strings.TrimSpace(req.IP)
	}
	if ip == "" {
		return "", fmt.Errorf("存储节点自注册缺少 ip")
	}
	if req.HTTPPort <= 0 {
		return "", fmt.Errorf("存储节点自注册缺少 httpPort")
	}
	nodeID := strings.TrimSpace(req.MediaServerID)
	if nodeID == "" {
		nodeID = fmt.Sprintf("%s:%d", ip, req.HTTPPort)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	n := &domainstoragenode.StorageNode{
		NodeID:     nodeID,
		IP:         ip,
		HTTPPort:   req.HTTPPort,
		RTSPPort:   req.RTSPPort,
		Secret:     req.Secret,
		Version:    req.Version,
		CapacityGB: req.CapacityGB,
		UsedGB:     req.UsedGB,
		Online:     true,
		LastSeen:   time.Now().UnixMilli(),
		UpdateTime: now,
	}
	if existing, err := s.repo.GetByNodeID(nodeID); err == nil && existing != nil {
		if n.Secret == "" {
			n.Secret = existing.Secret
		}
		n.Name = existing.Name
		n.DefaultNode = existing.DefaultNode
		n.CreateTime = existing.CreateTime
	} else {
		n.CreateTime = now
	}
	if err := s.repo.Save(n); err != nil {
		return "", err
	}
	applog.Infof("[storage-node] 注册 node=%s ip=%s http=%d cap=%dGB used=%dGB", nodeID, ip,
		req.HTTPPort, req.CapacityGB, req.UsedGB)
	return nodeID, nil
}

// Touch 仅刷新在线与 LastSeen。
func (s *Service) Touch(nodeID string) {
	if strings.TrimSpace(nodeID) == "" {
		return
	}
	if err := s.repo.UpdateLastSeen(nodeID, time.Now().UnixMilli(), true); err != nil {
		applog.Warnf("[storage-node] touch %s: %v", nodeID, err)
	}
}

// List 返回全部存储节点。
func (s *Service) List() ([]domainstoragenode.StorageNode, error) {
	return s.repo.ListAll()
}

// NodeView 是给前端的安全视图（不含 secret）。
type NodeView struct {
	NodeID      string `json:"nodeId"`
	Name        string `json:"name"`
	IP          string `json:"ip"`
	HTTPPort    int    `json:"httpPort"`
	RTSPPort    int    `json:"rtspPort"`
	Online      bool   `json:"online"`
	CapacityGB  int    `json:"capacityGb"`
	UsedGB      int    `json:"usedGb"`
	DefaultNode bool   `json:"defaultNode"`
	LastSeen    int64  `json:"lastSeen"`
}

// ListView 返回脱敏后的节点列表（给「录制目标」下拉用）。
func (s *Service) ListView() ([]NodeView, error) {
	rows, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	out := make([]NodeView, 0, len(rows))
	for i := range rows {
		r := rows[i]
		out = append(out, NodeView{
			NodeID: r.NodeID, Name: r.Name, IP: r.IP, HTTPPort: r.HTTPPort, RTSPPort: r.RTSPPort,
			Online: r.Online, CapacityGB: r.CapacityGB, UsedGB: r.UsedGB, DefaultNode: r.DefaultNode, LastSeen: r.LastSeen,
		})
	}
	return out, nil
}

// Resolve 取目标存储节点：优先 preferID，否则默认节点，否则唯一在线节点。
func (s *Service) Resolve(preferID string) (*domainstoragenode.StorageNode, error) {
	preferID = strings.TrimSpace(preferID)
	if preferID != "" && !strings.EqualFold(preferID, "auto") {
		n, err := s.repo.GetByNodeID(preferID)
		if err != nil {
			return nil, fmt.Errorf("存储节点不存在: %s", preferID)
		}
		return n, nil
	}
	rows, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	var fallback *domainstoragenode.StorageNode
	var onlineCount int
	for i := range rows {
		row := rows[i]
		if !row.Online {
			continue
		}
		onlineCount++
		if row.DefaultNode {
			return &row, nil
		}
		if fallback == nil {
			cp := row
			fallback = &cp
		}
	}
	if fallback != nil && onlineCount == 1 {
		return fallback, nil
	}
	if fallback != nil {
		return fallback, nil
	}
	return nil, fmt.Errorf("无可用存储节点，请确认 zstorage 已注册并在线")
}

// MarkOffline 把长时间未心跳的节点置离线（后台任务调用）。
func (s *Service) MarkOffline(nodeID string) {
	_ = s.repo.UpdateLastSeen(nodeID, time.Now().UnixMilli(), false)
}

// ResolveHint 依据回调里的 mediaServerId 或来源 IP 反查存储节点（用于 on_record_ps）。
func (s *Service) ResolveHint(mediaServerID, clientIP string) *domainstoragenode.StorageNode {
	mediaServerID = strings.TrimSpace(mediaServerID)
	if mediaServerID != "" {
		if n, err := s.repo.GetByNodeID(mediaServerID); err == nil && n != nil {
			return n
		}
	}
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		return nil
	}
	rows, err := s.repo.ListAll()
	if err != nil {
		return nil
	}
	for i := range rows {
		if rows[i].IP == clientIP && rows[i].Online {
			n := rows[i]
			return &n
		}
	}
	return nil
}
