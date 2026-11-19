package upstreamapp

import (
	"fmt"
	"log"
	"sort"
	"strings"

	domainchannel "zero-web-server/internal/domain/channel"
	domainplatform "zero-web-server/internal/domain/platform"
	domaingroupregion "zero-web-server/internal/domain/groupregion"
	domaintree "zero-web-server/internal/domain/tree"
	"zero-web-server/internal/gbsip"
)

// catalogNodeRepo 提供区域/分组节点（级联目录组装用）。
type catalogNodeRepo interface {
	ListAllRegions() ([]domaintree.Node, error)
	ListAllGroups() ([]domaintree.Node, error)
	ListRegionCodeMappings() ([]*domaingroupregion.RegionCodeMapping, error)
}

// catalogBatchSize 单条目录通知最大条目（附录 N 建议触发阈值 100 条）。
const catalogBatchSize = 100
// catalogBatchBytes 单条 UDP 报文上限，超出则拆分（附录 M：大目录分多条发送）。
const catalogBatchBytes = 700

// regionNodeID 把行政区划编码转换为 20 位国标目录节点编号：
// 中心编码(8，区划码右补 0) + 行业 00 + 类型 216 + 网络标识 0 + 序号 000000。
func regionNodeID(code string) string {
	code = strings.TrimSpace(code)
	if len(code) == 20 {
		return code
	}
	if code == "" || len(code) > 8 {
		return ""
	}
	return code + strings.Repeat("0", 8-len(code)) + "00" + "216" + "0" + "000000"
}

// civilCodeOf 仅保留合法的行政区划编码（2/4/6/8 位），其余留空。
func civilCodeOf(code string) string {
	switch len(strings.TrimSpace(code)) {
	case 2, 4, 6, 8:
		return strings.TrimSpace(code)
	default:
		return ""
	}
}

func typeCodeOfID(deviceID string) string {
	if len(deviceID) != 20 {
		return ""
	}
	return deviceID[10:13]
}

func itemTypeOfID(deviceID string) string {
	t := typeCodeOfID(deviceID)
	if t != "" && t < "200" {
		return "channel"
	}
	return "node"
}

// buildCatalog 按下级平台共享通道组装目录树：
// 区域节点(216) / 分组节点(215/216) / 通道，父节点优先，通道挂到真实父节点。
func (s *ChannelService) buildCatalog(platformID int) ([]gbsip.CatalogItem, error) {
	channelIDs, err := s.platformCh.ListPlatformChannels(platformID)
	if err != nil {
		return nil, err
	}
	regions, _ := s.nodes.ListAllRegions()
	groups, _ := s.nodes.ListAllGroups()
	regionCodes := make(map[string]string)
	if maps, err := s.nodes.ListRegionCodeMappings(); err == nil {
		for _, m := range maps {
			if len(strings.TrimSpace(m.GBCode)) == 20 {
				regionCodes[strings.TrimSpace(m.RegionDeviceID)] = strings.TrimSpace(m.GBCode)
			}
		}
	}
	resolveRegionID := func(code string) string {
		if mapped, ok := regionCodes[strings.TrimSpace(code)]; ok {
			return mapped
		}
		return regionNodeID(code)
	}

	// 按业务目录（215/216 分组）共享：展开子树，把节点下所有通道纳入本次目录。
	if shares, err := s.platforms.ListGroupShares(platformID); err == nil && len(shares) > 0 {
		expanded := expandGroupSubtree(shares, groups)
		if all, _, err := s.channels.ListAll(1, 100000, ""); err == nil {
			for _, ch := range all {
				if _, ok := expanded[strings.TrimSpace(ch.ParentID)]; ok {
					channelIDs = append(channelIDs, ch.ID)
				}
			}
		}
		channelIDs = dedupeInts(channelIDs)
	}
	// 没有共享通道时返回空目录：取消共享后仍需据此向上级发送删除通知。
	if len(channelIDs) == 0 {
		return []gbsip.CatalogItem{}, nil
	}
	regionBy := make(map[string]domaintree.Node, len(regions))
	for _, r := range regions {
		regionBy[strings.TrimSpace(r.DeviceID)] = r
	}
	groupBy := make(map[string]domaintree.Node, len(groups))
	for _, g := range groups {
		groupBy[strings.TrimSpace(g.DeviceID)] = g
	}

	nodes := map[string]*gbsip.CatalogItem{} // deviceID -> item
	depth := map[string]int{}
	var addRegionChain func(code string, d int)
	addRegionChain = func(code string, d int) {
		code = strings.TrimSpace(code)
		if code == "" {
			return
		}
		region, ok := regionBy[code]
		if !ok {
			return
		}
		id := resolveRegionID(code)
		if id == "" {
			return
		}
		if _, seen := nodes[id]; seen {
			return
		}
		item := &gbsip.CatalogItem{
			DeviceID:  id,
			Name:      region.Name,
			Status:    "ON",
			CivilCode: civilCodeOf(code),
		}
		if parent := strings.TrimSpace(region.ParentDeviceID); parent != "" {
			item.ParentID = resolveRegionID(parent)
			addRegionChain(parent, d+1)
		}
		nodes[id] = item
		if cur, ok := depth[id]; !ok || d < cur {
			depth[id] = d
		}
	}
	var addGroupChain func(id string, d int)
	addGroupChain = func(id string, d int) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		g, ok := groupBy[id]
		if !ok {
			return
		}
		if _, seen := nodes[g.DeviceID]; seen {
			return
		}
		item := &gbsip.CatalogItem{
			DeviceID:  g.DeviceID,
			Name:      g.Name,
			Status:    "ON",
			CivilCode: civilCodeOf(g.CivilCode),
		}
		parent := strings.TrimSpace(g.ParentDeviceID)
		switch {
		case parent != "" && groupBy[parent].DeviceID != "":
			item.ParentID = parent
			addGroupChain(parent, d+1)
		case g.CivilCode != "":
			if rid := resolveRegionID(g.CivilCode); rid != "" {
				if _, ok := regionBy[g.CivilCode]; ok {
					item.ParentID = rid
					addRegionChain(g.CivilCode, d+1)
				}
			}
		}
		nodes[g.DeviceID] = item
		if cur, ok := depth[g.DeviceID]; !ok || d < cur {
			depth[g.DeviceID] = d
		}
	}

	channels := make([]gbsip.CatalogItem, 0, len(channelIDs))
	seenCh := make(map[string]struct{}, len(channelIDs))
	for _, cid := range channelIDs {
		ch, err := s.channels.GetByID(cid)
		if err != nil || ch == nil {
			continue
		}
		if _, dup := seenCh[ch.GBDeviceID]; dup {
			continue
		}
		seenCh[ch.GBDeviceID] = struct{}{}
		channels = append(channels, s.channelItem(platformID, ch, groupBy, regionBy, resolveRegionID, addGroupChain, addRegionChain))
	}

	// 计算是否有子节点（Parental），并按深度降序输出（祖先在前）。
	childCount := make(map[string]int, len(nodes))
	for _, it := range nodes {
		if it.ParentID != "" {
			childCount[it.ParentID]++
		}
	}
	nodeList := make([]gbsip.CatalogItem, 0, len(nodes))
	for id, it := range nodes {
		item := *it
		if childCount[id] > 0 {
			item.Parental = 1
		}
		nodeList = append(nodeList, item)
	}
	sort.Slice(nodeList, func(i, j int) bool {
		di, dj := depth[nodeList[i].DeviceID], depth[nodeList[j].DeviceID]
		if di != dj {
			return di > dj
		}
		return nodeList[i].DeviceID < nodeList[j].DeviceID
	})

	// 祖先缺失时不要把悬挂的 ParentID 发出去（接收端会把挂不上的当孤儿）。
	for i := range nodeList {
		if nodeList[i].ParentID != "" {
			if _, ok := nodes[nodeList[i].ParentID]; !ok {
				log.Printf("[cascade] 节点 %s 的父节点 %s 未纳入本次目录，按根节点处理",
					nodeList[i].DeviceID, nodeList[i].ParentID)
				nodeList[i].ParentID = ""
			}
		}
	}
	for i := range channels {
		if channels[i].ParentID != "" {
			if _, ok := nodes[channels[i].ParentID]; !ok {
				channels[i].ParentID = ""
			}
		}
	}

	out := make([]gbsip.CatalogItem, 0, len(nodeList)+len(channels))
	out = append(out, nodeList...)
	out = append(out, channels...)
	return out, nil
}

func (s *ChannelService) channelItem(
	platformID int,
	ch *domainchannel.Channel,
	groupBy, regionBy map[string]domaintree.Node,
	resolveRegionID func(string) string,
	addGroupChain func(string, int),
	addRegionChain func(string, int),
) gbsip.CatalogItem {
	status := strings.TrimSpace(ch.Status)
	if status == "" {
		status = "ON"
	}
	item := gbsip.CatalogItem{
		DeviceID:     ch.GBDeviceID,
		Name:         ch.Name,
		Manufacturer: ch.Manufacturer,
		Model:        ch.Model,
		Status:       status,
		CivilCode:    civilCodeOf(ch.CivilCode),
		Parental:     0,
		RegisterWay:  1,
		Longitude:    ch.Longitude,
		Latitude:     ch.Latitude,
	}
	// 平台级自定义编号/名称优先（共享通道可单独改名改名）。
	if row, err := s.platformCh.GetPlatformChannel(platformID, ch.ID); err == nil && row != nil {
		if row.CustomDeviceID != "" {
			item.DeviceID = row.CustomDeviceID
		}
		if row.CustomName != "" {
			item.Name = row.CustomName
		}
	}
	// 父节点优先级：虚拟组织(业务分组) > 行政区划。
	if p := strings.TrimSpace(ch.ParentID); p != "" {
		if _, ok := groupBy[p]; ok {
			item.ParentID = p
			addGroupChain(p, 1)
			return item
		}
		log.Printf("[cascade] 通道 %s 的父分组 %s 不存在，尝试按行政区划挂载", ch.GBDeviceID, p)
	}
	if code := strings.TrimSpace(ch.CivilCode); code != "" {
		if _, ok := regionBy[code]; ok {
			item.ParentID = resolveRegionID(code)
			addRegionChain(code, 1)
		}
	}
	return item
}

// hashItem 用于差分：名称/父节点/状态/类型变化即重新下发。
func hashItem(it gbsip.CatalogItem) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s", it.DeviceID, it.Name, it.ParentID, it.Status, itemTypeOfID(it.DeviceID))
}

// pushCatalog 计算与上次快照的差异并分批推送（ADD/UPDATE/DEL）。
func (s *ChannelService) pushCatalog(platformID int) error {
	platform, err := s.platforms.GetByID(platformID)
	if err != nil {
		return fmt.Errorf("平台不存在")
	}
	items, err := s.buildCatalog(platformID)
	if err != nil {
		return err
	}
	prev, err := s.platforms.ListCatalogSnapshot(platformID)
	if err != nil {
		return err
	}
	if len(items) == 0 && len(prev) == 0 {
		return fmt.Errorf("未配置共享通道")
	}
	prevMap := make(map[string]struct {
		hash string
		name string
	}, len(prev))
	for _, p := range prev {
		prevMap[p.DeviceID] = struct {
			hash string
			name string
		}{p.Hash, p.Name}
	}

	current := make(map[string]struct{}, len(items))
	events := make([]gbsip.CatalogItem, 0)
	for _, it := range items {
		current[it.DeviceID] = struct{}{}
		if p, ok := prevMap[it.DeviceID]; ok {
			if p.hash == hashItem(it) {
				continue
			}
			it.Event = "UPDATE"
		} else {
			it.Event = "ADD"
		}
		events = append(events, it)
	}
	for _, p := range prev {
		if _, ok := current[p.DeviceID]; ok {
			continue
		}
		events = append(events, gbsip.CatalogItem{DeviceID: p.DeviceID, Name: p.Name, Event: "DEL"})
	}
	if len(events) == 0 {
		log.Printf("[cascade] 目录无变化，跳过推送 platform=%s", platform.Name)
		return nil
	}
	batches := splitCatalogBatches(platform.ServerGBID, events)
	for _, batch := range batches {
		if err := s.sip.SendCatalogNotify(platform, batch); err != nil {
			return err
		}
	}
	snapshot := make([]*domainplatform.CatalogSnapshot, 0, len(items))
	for _, it := range items {
		snapshot = append(snapshot, &domainplatform.CatalogSnapshot{
			PlatformID: platformID, DeviceID: it.DeviceID, Name: it.Name,
			ParentID: it.ParentID, ItemType: itemTypeOfID(it.DeviceID), Status: it.Status,
			Hash: hashItem(it),
		})
	}
	if err := s.platforms.ReplaceCatalogSnapshot(platformID, snapshot); err != nil {
		log.Printf("[cascade] 保存目录快照失败 platform=%s err=%v", platform.Name, err)
	}
	log.Printf("[cascade] 目录推送 platform=%s 条目=%d 变更=%d", platform.Name, len(items), len(events))
	return nil
}

// splitCatalogBatches 按条目数与报文字节数拆分，避免 UDP 超 MTU（附录 M）。
func splitCatalogBatches(platformID string, events []gbsip.CatalogItem) [][]gbsip.CatalogItem {
	var batches [][]gbsip.CatalogItem
	var cur []gbsip.CatalogItem
	for _, ev := range events {
		next := append(cur, ev)
		if len(cur) > 0 && (len(next) > catalogBatchSize ||
			gbsip.CatalogNotifySize(platformID, next) > catalogBatchBytes) {
			batches = append(batches, cur)
			cur = []gbsip.CatalogItem{ev}
			continue
		}
		cur = next
	}
	if len(cur) > 0 {
		batches = append(batches, cur)
	}
	return batches
}

// expandGroupSubtree 从共享的 215/216 节点展开到所有后代分组。
func expandGroupSubtree(roots []string, groups []domaintree.Node) map[string]struct{} {
	children := make(map[string][]string, len(groups))
	exists := make(map[string]struct{}, len(groups))
	for _, g := range groups {
		exists[g.DeviceID] = struct{}{}
		if p := strings.TrimSpace(g.ParentDeviceID); p != "" {
			children[p] = append(children[p], g.DeviceID)
		}
	}
	out := make(map[string]struct{}, len(roots))
	queue := make([]string, 0, len(roots))
	for _, r := range roots {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		out[r] = struct{}{}
		queue = append(queue, r)
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, child := range children[cur] {
			if _, seen := out[child]; seen {
				continue
			}
			out[child] = struct{}{}
			queue = append(queue, child)
		}
	}
	return out
}

func dedupeInts(in []int) []int {
	seen := make(map[int]struct{}, len(in))
	out := make([]int, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
