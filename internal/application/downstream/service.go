package downstreamapp

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"zero-web-server/internal/apperr"

	gbserver "github.com/zero-pipe/gb28181-go/server"
	domainsub "zero-web-server/internal/domain/subordinate"
	"zero-web-server/internal/gbsip"
	applog "zero-web-server/pkg/log"
)

var (
	ErrNameRequired     = apperr.New(apperr.CodeCascade+1, "名称不能为空")
	ErrGBIDRequired     = apperr.New(apperr.CodeCascade+2, "下级平台国标编号不能为空")
	ErrGBIDLength       = apperr.New(apperr.CodeCascade+3, "下级平台国标编号须为20位")
	ErrPasswordRequired = apperr.New(apperr.CodeCascade+4, "注册密码不能为空")
	ErrDisabled         = apperr.New(apperr.CodeCascade+5, "下级平台未启用")
	ErrPlatformNotFound = apperr.New(apperr.CodeCascade+6, "下级平台不存在")
)

type Service struct {
	repo     domainsub.Repository
	serverID string
}

func NewService(repo domainsub.Repository, serverID string) *Service {
	return &Service{repo: repo, serverID: serverID}
}

func (s *Service) List(page, count int, query string) ([]*domainsub.Platform, int64, error) {
	return s.repo.List(page, count, query)
}

func (s *Service) Get(id int) (*domainsub.Platform, error) {
	return s.repo.GetByID(id)
}

func (s *Service) Add(p *domainsub.Platform) error {
	if err := validateSub(p); err != nil {
		return err
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	p.CreateTime = now
	p.UpdateTime = now
	p.ServerID = s.serverID
	p.Status = false
	if p.Transport == "" {
		p.Transport = "UDP"
	}
	return s.repo.Create(p)
}

func (s *Service) Update(p *domainsub.Platform) error {
	if err := validateSub(p); err != nil {
		return err
	}
	p.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	return s.repo.Update(p)
}

func (s *Service) Delete(id int) error {
	return s.repo.Delete(id)
}

// ResolvePassword implements SIP auth lookup for subordinates.
func (s *Service) ResolvePassword(gbID string) (password string, known bool, err error) {
	p, err := s.repo.GetByGBID(gbID)
	if err != nil || p == nil {
		return "", false, nil
	}
	if !p.Enable {
		// 未启用视为未知，触发 403（强制预登记时）
		return "", false, nil
	}
	return p.Password, true, nil
}

func (s *Service) Known(gbID string) bool {
	p, err := s.repo.GetByGBID(gbID)
	return err == nil && p != nil && p.Enable
}

func (s *Service) Exists(gbID string) bool {
	p, err := s.repo.GetByGBID(gbID)
	return err == nil && p != nil
}

func (s *Service) OnRegister(ev gbserver.RegisterEvent) error {
	p, err := s.repo.GetByGBID(ev.DeviceID)
	if err != nil || p == nil {
		return errors.New("下级平台未预登记")
	}
	if !p.Enable {
		return ErrDisabled
	}
	return s.repo.UpdateOnline(ev.DeviceID, ev.IP, ev.Port, ev.Expires, ev.CallID, ev.Transport)
}

func (s *Service) OnUnregister(gbID string) error {
	return s.repo.UpdateOffline(gbID)
}

func (s *Service) OnKeepalive(gbID, ip string, port int) error {
	p, err := s.repo.GetByGBID(gbID)
	if err != nil || p == nil {
		return nil
	}
	if ip == "" {
		return s.repo.UpdateOnline(gbID, p.IP, p.Port, p.Expires, p.RegisterCall, p.Transport)
	}
	return s.repo.UpdateOnline(gbID, ip, port, p.Expires, p.RegisterCall, p.Transport)
}

func validateSub(p *domainsub.Platform) error {
	if p == nil {
		return errors.New("参数错误")
	}
	p.Name = strings.TrimSpace(p.Name)
	p.DeviceGBID = strings.TrimSpace(p.DeviceGBID)
	p.Password = strings.TrimSpace(p.Password)
	p.Transport = strings.ToUpper(strings.TrimSpace(p.Transport))
	if p.Name == "" {
		return ErrNameRequired
	}
	if p.DeviceGBID == "" {
		return ErrGBIDRequired
	}
	if len(p.DeviceGBID) != 20 {
		return ErrGBIDLength
	}
	if p.Password == "" {
		return ErrPasswordRequired
	}
	if p.Transport == "" {
		p.Transport = "UDP"
	}
	return nil
}

// HandleCatalog 接收下级平台推送的目录（NOTIFY 或查询应答）。
// 只按规范组树：ParentID 能找到才挂；找不到的标记孤儿并告警，不影响其它节点。
// OnCatalog 实现 sipinfra.SubordinateHandler。
func (s *Service) OnCatalog(gbID string, items []gbsip.CatalogItem, incremental bool) error {
	return s.HandleCatalog(gbID, items, incremental)
}

func (s *Service) HandleCatalog(upstreamGBID string, items []gbsip.CatalogItem, incremental bool) error {
	if len(items) == 0 {
		return nil
	}
	p, err := s.repo.GetByGBID(strings.TrimSpace(upstreamGBID))
	if err != nil || p == nil {
		return ErrPlatformNotFound
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	seen := make(map[string]struct{}, len(items))
	orphans := 0
	for _, it := range items {
		deviceID := strings.TrimSpace(it.DeviceID)
		if deviceID == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(it.Event), "DEL") {
			if err := s.repo.DeleteCatalog(p.ID, deviceID); err != nil {
				applog.Warnf("[cascade] 删除下级目录失败 platform=%s device=%s err=%v", upstreamGBID, deviceID, err)
			}
			continue
		}
		typeCode := typeCodeOf(deviceID)
		entry := &domainsub.CatalogEntry{
			PlatformID:   p.ID,
			DeviceID:     deviceID,
			Name:         strings.TrimSpace(it.Name),
			ParentID:     strings.TrimSpace(it.ParentID),
			TypeCode:     typeCode,
			ItemType:     classifyItem(typeCode),
			Status:       strings.TrimSpace(it.Status),
			CivilCode:    strings.TrimSpace(it.CivilCode),
			Manufacturer: strings.TrimSpace(it.Manufacturer),
			Model:        strings.TrimSpace(it.Model),
			RawJSON:      rawCatalogJSON(it),
		}
		if entry.Name == "" {
			entry.Name = deviceID
		}
		if err := s.repo.UpsertCatalog(entry); err != nil {
			applog.Warnf("[cascade] 写入下级目录失败 platform=%s device=%s err=%v", upstreamGBID, deviceID, err)
			continue
		}
		seen[deviceID] = struct{}{}
	}

	// 第二轮：解析父节点，挂不上的标记孤儿（不删除，便于后续修正后自动挂回）。
	all, err := s.repo.ListCatalog(p.ID)
	if err != nil {
		return err
	}
	known := make(map[string]struct{}, len(all)+1)
	known[p.DeviceGBID] = struct{}{}
	for _, e := range all {
		known[e.DeviceID] = struct{}{}
	}
	for _, e := range all {
		orphan := e.ParentID != "" && !isKnownParent(e.ParentID, known, p.DeviceGBID)
		if orphan != e.Orphan {
			e.Orphan = orphan
			e.UpdateTime = now
			if err := s.repo.UpsertCatalog(e); err != nil {
				continue
			}
		}
		if orphan {
			orphans++
			applog.Warnf("[cascade] 目录父节点不存在，暂不挂载 platform=%s device=%s name=%s parent=%s",
				upstreamGBID, e.DeviceID, e.Name, e.ParentID)
		}
	}
	applog.Infof("[cascade] 接收下级目录 platform=%s items=%d incremental=%v 孤儿=%d",
		upstreamGBID, len(items), incremental, orphans)
	return nil
}

// isKnownParent 父节点要么是下级平台自身，要么在已入库目录中。
func isKnownParent(parent string, known map[string]struct{}, platformGBID string) bool {
	if parent == platformGBID {
		return true
	}
	_, ok := known[parent]
	return ok
}

// typeCodeOf 取 20 位国标编号的第 11~13 位（类型编码）。
func typeCodeOf(deviceID string) string {
	if len(deviceID) != 20 {
		return ""
	}
	return deviceID[10:13]
}

// classifyItem 依据类型编码区分目录节点与通道。
func classifyItem(typeCode string) string {
	switch {
	case typeCode == "200", typeCode == "215", typeCode == "216":
		return domainsub.ItemNode
	case typeCode == "":
		return domainsub.ItemNode
	case typeCode >= "200" && typeCode <= "299":
		return domainsub.ItemNode
	default:
		return domainsub.ItemChannel
	}
}

func rawCatalogJSON(it gbsip.CatalogItem) string {
	b, err := json.Marshal(it)
	if err != nil {
		return ""
	}
	return string(b)
}

// CatalogNode 目录树节点。
type CatalogNode struct {
	domainsub.CatalogEntry
	Children []*CatalogNode `json:"children,omitempty"`
}

// CatalogList 返回下级目录平铺列表，可按名称/编号过滤，可只看孤儿。
func (s *Service) CatalogList(platformID int, query string, orphan *bool) ([]*domainsub.CatalogEntry, error) {
	all, err := s.repo.ListCatalog(platformID)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]*domainsub.CatalogEntry, 0, len(all))
	for _, e := range all {
		if orphan != nil && e.Orphan != *orphan {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(e.Name), q) &&
			!strings.Contains(strings.ToLower(e.DeviceID), q) {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// CatalogTree 按 ParentID 组装目录树；孤儿节点不挂载（仅在列表中标记）。
func (s *Service) CatalogTree(platformID int) ([]*CatalogNode, error) {
	p, err := s.repo.GetByID(platformID)
	if err != nil {
		return nil, err
	}
	all, err := s.repo.ListCatalog(platformID)
	if err != nil {
		return nil, err
	}
	nodes := make(map[string]*CatalogNode, len(all))
	for _, e := range all {
		nodes[e.DeviceID] = &CatalogNode{CatalogEntry: *e}
	}
	roots := make([]*CatalogNode, 0)
	for _, e := range all {
		n := nodes[e.DeviceID]
		switch {
		case e.Orphan:
			continue
		case e.ParentID == "" || e.ParentID == p.DeviceGBID:
			roots = append(roots, n)
		default:
			if parent, ok := nodes[e.ParentID]; ok && !parent.Orphan {
				parent.Children = append(parent.Children, n)
			}
			// 父节点不存在（孤儿）已在第一轮标记，这里不挂。
		}
	}
	return roots, nil
}
