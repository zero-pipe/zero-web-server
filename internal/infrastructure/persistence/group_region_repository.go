package persistence

import (
	"encoding/json"
	"fmt"
	"strings"

	domaingroupregion "zero-web-server/internal/domain/groupregion"
	domaintree "zero-web-server/internal/domain/tree"

	"gorm.io/gorm"
)

// GroupRegionRepository：行政区划 / 业务分组目录 + 目录挂载。
// 新 schema：
//   zws_region(code=deviceId, name, alias, parent_id)
//   zws_group(code=deviceId, name, business_group, civil_code, alias, parent_id)
//   zws_catalog(region_id|group_id, target_type=device|channel, target_id) ← 多挂载
// 通道隶属关系（gb_json.civilCode/parentId）沿用 channel_ext.gb_json 过渡语义。
type GroupRegionRepository struct {
	db *gorm.DB
}

func NewGroupRegionRepository(db *gorm.DB) *GroupRegionRepository {
	return &GroupRegionRepository{db: db}
}

func idToStr(p *int64) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%d", *p)
}

func intPtr(v int) *int64 { x := int64(v); return &x }

// ============================ 业务分组树 ============================

func (r *GroupRegionRepository) QueryGroupTree(query string, parentID *int) ([]domaintree.Node, error) {
	var rows []domaintree.Node
	sql := `SELECT id, code AS device_id, name, parent_id, '' AS parent_device_id,
        COALESCE(business_group,'') AS business_group, COALESCE(civil_code,'') AS civil_code,
        COALESCE(alias,'') AS alias, CONCAT('group', id) AS tree_id, 0 AS type, 0 AS is_leaf, 'ON' AS status
FROM zws_group WHERE 1=1`
	args := make([]any, 0, 3)
	if parentID != nil {
		sql += " AND parent_id = ?"
		args = append(args, *parentID)
	} else {
		sql += " AND parent_id IS NULL"
	}
	if query != "" {
		sql += " AND (code LIKE ? OR name LIKE ?)"
		like := fmt.Sprintf("%%%s%%", query)
		args = append(args, like, like)
	}
	sql += " ORDER BY code ASC"
	if err := r.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []domaintree.Node{}
	}
	return rows, nil
}

func (r *GroupRegionRepository) GetGroupByID(id int) (*domaintree.Node, error) {
	var row struct {
		ID     int64  `gorm:"column:id"`
		Code   string `gorm:"column:code"`
		Name   string `gorm:"column:name"`
	}
	if err := r.db.Raw("SELECT id, code, name FROM zws_group WHERE id = ?", id).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &domaintree.Node{ID: int(row.ID), DeviceID: row.Code, Name: row.Name}, nil
}

func (r *GroupRegionRepository) QueryGroupChannels(query, parentDeviceID string) ([]domaintree.Node, error) {
	sql := `SELECT c.id,
       CONCAT('channel', c.id) AS tree_id,
       COALESCE(NULLIF(TRIM(c.gb_code), ''), c.internal_code) AS device_id,
       COALESCE(NULLIF(TRIM(c.name), ''), NULLIF(TRIM(c.gb_code), ''), '未命名') AS name,
       JSON_UNQUOTE(JSON_EXTRACT(e.gb_json, '$.parentId')) AS parent_device_id,
       '' AS business_group,
       COALESCE(NULLIF(TRIM(c.status), ''), 'OFF') AS status,
       1 AS type, 1 AS is_leaf
FROM zws_channel c LEFT JOIN zws_channel_ext e ON e.channel_id = c.id
WHERE c.kind = 'video_sensor' AND JSON_UNQUOTE(JSON_EXTRACT(e.gb_json, '$.parentId')) = ?`
	args := []any{parentDeviceID}
	if query != "" {
		sql += " AND (c.gb_code LIKE ? OR c.name LIKE ?)"
		like := fmt.Sprintf("%%%s%%", query)
		args = append(args, like, like)
	}
	var rows []domaintree.Node
	if err := r.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].IsLeaf = true
		rows[i].Leaf = true
	}
	if rows == nil {
		rows = []domaintree.Node{}
	}
	return rows, nil
}

func (r *GroupRegionRepository) QueryRegionChannels(parentDeviceID string) ([]domaintree.Node, error) {
	sql := `SELECT c.id,
       CONCAT('channel', c.id) AS tree_id,
       COALESCE(NULLIF(TRIM(c.gb_code), ''), c.internal_code) AS device_id,
       COALESCE(NULLIF(TRIM(c.name), ''), NULLIF(TRIM(c.gb_code), ''), '未命名') AS name,
       JSON_UNQUOTE(JSON_EXTRACT(e.gb_json, '$.civilCode')) AS parent_device_id,
       COALESCE(NULLIF(TRIM(c.status), ''), 'OFF') AS status,
       1 AS type, 1 AS is_leaf
FROM zws_channel c LEFT JOIN zws_channel_ext e ON e.channel_id = c.id
WHERE c.kind = 'video_sensor' AND JSON_UNQUOTE(JSON_EXTRACT(e.gb_json, '$.civilCode')) = ?`
	var rows []domaintree.Node
	if err := r.db.Raw(sql, parentDeviceID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].IsLeaf = true
		rows[i].Leaf = true
	}
	if rows == nil {
		rows = []domaintree.Node{}
	}
	return rows, nil
}

func (r *GroupRegionRepository) GetGroupRecordByID(id int) (*domaingroupregion.GroupRecord, error) {
	var row domaingroupregion.GroupRecord
	if err := r.db.Raw(`SELECT id, code AS device_id, name, parent_id, '' AS parent_device_id,
       COALESCE(business_group,'') AS business_group, COALESCE(civil_code,'') AS civil_code, COALESCE(alias,'') AS alias
FROM zws_group WHERE id = ?`, id).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *GroupRegionRepository) GetGroupByDeviceID(deviceID string) (*domaingroupregion.GroupRecord, error) {
	var row domaingroupregion.GroupRecord
	if err := r.db.Raw(`SELECT id, code AS device_id, name, parent_id, '' AS parent_device_id,
       COALESCE(business_group,'') AS business_group, COALESCE(civil_code,'') AS civil_code, COALESCE(alias,'') AS alias
FROM zws_group WHERE code = ?`, deviceID).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *GroupRegionRepository) ResolveBusinessGroup(deviceID string) (string, error) {
	group, err := r.GetGroupByDeviceID(deviceID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(group.BusinessGroup) != "" {
		return group.BusinessGroup, nil
	}
	if gbGroupTypeCode(group.DeviceID) == "215" {
		return group.DeviceID, nil
	}
	if group.ParentID != nil {
		parent, err := r.GetGroupRecordByID(*group.ParentID)
		if err == nil {
			if strings.TrimSpace(parent.BusinessGroup) != "" {
				return parent.BusinessGroup, nil
			}
			if gbGroupTypeCode(parent.DeviceID) == "215" {
				return parent.DeviceID, nil
			}
		}
	}
	return "", fmt.Errorf("虚拟组织未关联业务分组")
}

func gbGroupTypeCode(deviceID string) string {
	if len(deviceID) < 13 {
		return ""
	}
	code := deviceID[10:13]
	if code == "215" || code == "216" {
		return code
	}
	return ""
}

func (r *GroupRegionRepository) GetGroupByDeviceAndBusiness(deviceID, businessGroup string) (*domaingroupregion.GroupRecord, error) {
	var row domaingroupregion.GroupRecord
	if err := r.db.Raw(`SELECT id, code AS device_id, name, parent_id, '' AS parent_device_id,
       COALESCE(business_group,'') AS business_group, COALESCE(civil_code,'') AS civil_code, COALESCE(alias,'') AS alias
FROM zws_group WHERE code = ? AND business_group = ?`, deviceID, businessGroup).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *GroupRegionRepository) GetBusinessGroup(businessGroup string) (*domaingroupregion.GroupRecord, error) {
	var row domaingroupregion.GroupRecord
	if err := r.db.Raw(`SELECT id, code AS device_id, name, parent_id, '' AS parent_device_id,
       COALESCE(business_group,'') AS business_group, COALESCE(civil_code,'') AS civil_code, COALESCE(alias,'') AS alias
FROM zws_group WHERE code = ? AND business_group = ?`, businessGroup, businessGroup).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *GroupRegionRepository) ExistsGroupDeviceID(deviceID string, excludeID int) (bool, error) {
	var count int64
	if excludeID > 0 {
		err := r.db.Raw(`SELECT COUNT(1) FROM zws_group WHERE code = ? AND id <> ?`, deviceID, excludeID).Scan(&count).Error
		return count > 0, err
	}
	err := r.db.Raw(`SELECT COUNT(1) FROM zws_group WHERE code = ?`, deviceID).Scan(&count).Error
	return count > 0, err
}

func (r *GroupRegionRepository) AddBusinessGroup(group *domaingroupregion.GroupRecord) error {
	now := nowTimeStr()
	return r.db.Exec(`INSERT INTO zws_group (code, name, business_group, civil_code, alias, create_time, update_time)
VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?)`,
		group.DeviceID, group.Name, group.DeviceID,
		strings.TrimSpace(group.CivilCode), strings.TrimSpace(group.Alias), now, now,
	).Error
}

func (r *GroupRegionRepository) AddVirtualGroup(group *domaingroupregion.GroupRecord) error {
	now := nowTimeStr()
	return r.db.Exec(`INSERT INTO zws_group (code, name, parent_id, business_group, civil_code, alias, create_time, update_time)
VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?)`,
		group.DeviceID, group.Name, group.ParentID, group.BusinessGroup,
		strings.TrimSpace(group.CivilCode), strings.TrimSpace(group.Alias), now, now,
	).Error
}

func (r *GroupRegionRepository) UpdateGroup(group *domaingroupregion.GroupRecord) error {
	return r.db.Exec(`UPDATE zws_group
SET code = ?, name = ?, parent_id = ?, business_group = ?, civil_code = NULLIF(?, ''),
    alias = NULLIF(?, ''), update_time = ? WHERE id = ?`,
		group.DeviceID, group.Name, group.ParentID, group.BusinessGroup,
		strings.TrimSpace(group.CivilCode), strings.TrimSpace(group.Alias), nowTimeStr(), group.ID,
	).Error
}

func (r *GroupRegionRepository) ListGroupsByBusinessGroup(businessGroup string) ([]domaingroupregion.GroupRecord, error) {
	var rows []domaingroupregion.GroupRecord
	err := r.db.Raw(`SELECT id, code AS device_id, name, parent_id, '' AS parent_device_id,
       COALESCE(business_group,'') AS business_group, COALESCE(civil_code,'') AS civil_code, COALESCE(alias,'') AS alias
FROM zws_group WHERE business_group = ? AND code <> ?`, businessGroup, businessGroup).Scan(&rows).Error
	if rows == nil {
		rows = []domaingroupregion.GroupRecord{}
	}
	return rows, err
}

func (r *GroupRegionRepository) ListGroupChildren(parentID int) ([]domaingroupregion.GroupRecord, error) {
	var rows []domaingroupregion.GroupRecord
	err := r.db.Raw(`SELECT id, code AS device_id, name, parent_id, '' AS parent_device_id,
       COALESCE(business_group,'') AS business_group, COALESCE(civil_code,'') AS civil_code, COALESCE(alias,'') AS alias
FROM zws_group WHERE parent_id = ?`, parentID).Scan(&rows).Error
	if rows == nil {
		rows = []domaingroupregion.GroupRecord{}
	}
	return rows, err
}

func (r *GroupRegionRepository) CollectGroupDescendants(rootID int) ([]domaingroupregion.GroupRecord, error) {
	root, err := r.GetGroupRecordByID(rootID)
	if err != nil {
		return nil, err
	}
	out := []domaingroupregion.GroupRecord{*root}
	queue := []int{rootID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		children, err := r.ListGroupChildren(id)
		if err != nil {
			return nil, err
		}
		for i := range children {
			out = append(out, children[i])
			queue = append(queue, children[i].ID)
		}
	}
	return out, nil
}

func (r *GroupRegionRepository) DeleteGroupsByIDs(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Exec(`DELETE FROM zws_group WHERE id IN ?`, ids).Error
}

// ClearChannelsByBusinessGroup 清空通道的 gb_json.parentId/businessGroupId。
func (r *GroupRegionRepository) ClearChannelsByBusinessGroup(businessGroup string) error {
	return r.patchChannelGBWhere(
		"JSON_UNQUOTE(JSON_EXTRACT(gb_json, '$.businessGroupId')) = ?", businessGroup,
		func(g *gbChannelJSON) { g.ParentID = ""; g.BusinessGroupID = "" })
}

func (r *GroupRegionRepository) ClearChannelsByParentDeviceIDs(deviceIDs []string) error {
	if len(deviceIDs) == 0 {
		return nil
	}
	return r.patchChannelGBWhere(
		"JSON_UNQUOTE(JSON_EXTRACT(gb_json, '$.parentId')) IN ?", deviceIDs,
		func(g *gbChannelJSON) { g.ParentID = ""; g.BusinessGroupID = "" })
}

// ============================ 行政区划树 ============================

func (r *GroupRegionRepository) QueryRegionTree(parentID *int) ([]domaintree.Node, error) {
	var rows []domaintree.Node
	sql := `SELECT id, code AS device_id, name, parent_id, COALESCE(alias,'') AS alias,
       CONCAT('region', id) AS tree_id, 0 AS type, 0 AS is_leaf, 'ON' AS status
FROM zws_region WHERE 1=1`
	args := make([]any, 0, 1)
	if parentID != nil {
		sql += " AND parent_id = ?"
		args = append(args, *parentID)
	} else {
		sql += " AND (parent_id IS NULL OR NOT EXISTS (SELECT 1 FROM zws_region p WHERE p.id = zws_region.parent_id))"
	}
	sql += " ORDER BY code ASC"
	if err := r.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []domaintree.Node{}
	}
	return rows, nil
}

func (r *GroupRegionRepository) GetRegionByID(id int) (*domaintree.Node, error) {
	var row struct {
		ID   int64  `gorm:"column:id"`
		Code string `gorm:"column:code"`
		Name string `gorm:"column:name"`
	}
	if err := r.db.Raw("SELECT id, code, name FROM zws_region WHERE id = ?", id).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &domaintree.Node{ID: int(row.ID), DeviceID: row.Code, Name: row.Name}, nil
}

func (r *GroupRegionRepository) AddRegion(region *domaingroupregion.RegionRecord) error {
	now := nowTimeStr()
	return r.db.Exec(`INSERT INTO zws_region (code, name, parent_id, create_time, update_time)
VALUES (?, ?, ?, ?, ?)`, region.DeviceID, region.Name, region.ParentID, now, now).Error
}

func (r *GroupRegionRepository) UpdateRegion(region *domaingroupregion.RegionRecord) error {
	return r.db.Exec(`UPDATE zws_region SET code = ?, name = ?, parent_id = ?, update_time = ? WHERE id = ?`,
		region.DeviceID, region.Name, region.ParentID, nowTimeStr(), region.ID).Error
}

func (r *GroupRegionRepository) ListRegionChildren(parentID int) ([]domaingroupregion.RegionRecord, error) {
	var rows []domaingroupregion.RegionRecord
	err := r.db.Raw(`SELECT id, code AS device_id, name, parent_id, '' AS parent_device_id
FROM zws_region WHERE parent_id = ?`, parentID).Scan(&rows).Error
	if rows == nil {
		rows = []domaingroupregion.RegionRecord{}
	}
	return rows, err
}

func (r *GroupRegionRepository) ListRegionChildrenByDeviceID(parentDeviceID string) ([]domaingroupregion.RegionRecord, error) {
	var rows []domaingroupregion.RegionRecord
	err := r.db.Raw(`SELECT c.id, c.code AS device_id, c.name, c.parent_id, '' AS parent_device_id
FROM zws_region c JOIN zws_region p ON p.id = c.parent_id WHERE p.code = ?`, strings.TrimSpace(parentDeviceID)).Scan(&rows).Error
	if rows == nil {
		rows = []domaingroupregion.RegionRecord{}
	}
	return rows, err
}

func (r *GroupRegionRepository) DeleteRegionsByIDs(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Exec(`DELETE FROM zws_region WHERE id IN ?`, ids).Error
}

func (r *GroupRegionRepository) ClearChannelsByCivilCodes(civilCodes []string) error {
	if len(civilCodes) == 0 {
		return nil
	}
	return r.patchChannelGBWhere(
		"JSON_UNQUOTE(JSON_EXTRACT(gb_json, '$.civilCode')) IN ?", civilCodes,
		func(g *gbChannelJSON) { g.CivilCode = "" })
}

func (r *GroupRegionRepository) ListAllRegions() ([]domaintree.Node, error) {
	var rows []domaintree.Node
	err := r.db.Raw(`SELECT id, code AS device_id, name, parent_id, COALESCE(alias,'') AS alias,
       CONCAT('region', id) AS tree_id, 0 AS type, 0 AS is_leaf, 'ON' AS status
FROM zws_region ORDER BY code ASC`).Scan(&rows).Error
	if rows == nil {
		rows = []domaintree.Node{}
	}
	return rows, err
}

func (r *GroupRegionRepository) ListAllGroups() ([]domaintree.Node, error) {
	var rows []domaintree.Node
	err := r.db.Raw(`SELECT id, code AS device_id, name, parent_id, '' AS parent_device_id,
       COALESCE(business_group,'') AS business_group, COALESCE(civil_code,'') AS civil_code, COALESCE(alias,'') AS alias,
       CONCAT('group', id) AS tree_id, SUBSTRING(code, 11, 3) AS type, 0 AS is_leaf, 'ON' AS status
FROM zws_group ORDER BY code ASC`).Scan(&rows).Error
	if rows == nil {
		rows = []domaintree.Node{}
	}
	return rows, err
}

func (r *GroupRegionRepository) GetRegionByDeviceID(deviceID string) (*domaintree.Node, error) {
	var row struct {
		ID   int64  `gorm:"column:id"`
		Code string `gorm:"column:code"`
		Name string `gorm:"column:name"`
	}
	if err := r.db.Raw(`SELECT id, code, name FROM zws_region WHERE code = ?`, deviceID).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &domaintree.Node{ID: int(row.ID), DeviceID: row.Code, Name: row.Name}, nil
}

// ============================ 区划编码映射 ============================

func (r *GroupRegionRepository) ListRegionCodeMappings() ([]*domaingroupregion.RegionCodeMapping, error) {
	var rows []struct {
		Code   string `gorm:"column:code"`
		GBCode string `gorm:"column:gb_code"`
	}
	if err := r.db.Raw(`SELECT r.code AS code, m.gb_code FROM zws_region_code_mapping m
JOIN zws_region r ON r.id = m.region_id`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domaingroupregion.RegionCodeMapping, 0, len(rows))
	for i := range rows {
		out = append(out, &domaingroupregion.RegionCodeMapping{RegionDeviceID: rows[i].Code, GBCode: rows[i].GBCode})
	}
	return out, nil
}

func (r *GroupRegionRepository) UpsertRegionCodeMapping(m *domaingroupregion.RegionCodeMapping) error {
	var regionID int64
	_ = r.db.Raw(`SELECT id FROM zws_region WHERE code = ?`, m.RegionDeviceID).Scan(&regionID).Error
	if regionID == 0 {
		return fmt.Errorf("区划节点不存在: %s", m.RegionDeviceID)
	}
	return r.db.Exec(`INSERT INTO zws_region_code_mapping (region_id, gb_code, update_time)
VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE gb_code = VALUES(gb_code), update_time = VALUES(update_time)`,
		regionID, m.GBCode, nowTimeStr()).Error
}

func (r *GroupRegionRepository) DeleteRegionCodeMapping(regionDeviceID string) error {
	var regionID int64
	_ = r.db.Raw(`SELECT id FROM zws_region WHERE code = ?`, regionDeviceID).Scan(&regionID).Error
	if regionID == 0 {
		return nil
	}
	return r.db.Exec(`DELETE FROM zws_region_code_mapping WHERE region_id = ?`, regionID).Error
}

// patchChannelGBWhere 按条件批量读改写 channel_ext.gb_json（多行）。
func (r *GroupRegionRepository) patchChannelGBWhere(cond string, arg any, fn func(g *gbChannelJSON)) error {
	var exts []struct {
		ID      int64  `gorm:"column:id"`
		GBJSON  string `gorm:"column:gb_json"`
	}
	if err := r.db.Table("zws_channel_ext").Select("id, gb_json").Where(cond, arg).Scan(&exts).Error; err != nil {
		return err
	}
	for _, e := range exts {
		var g gbChannelJSON
		if strings.TrimSpace(e.GBJSON) != "" {
			_ = json.Unmarshal([]byte(e.GBJSON), &g)
		}
		fn(&g)
		raw, _ := json.Marshal(g)
		if err := r.db.Exec("UPDATE zws_channel_ext SET gb_json = ?, update_time = ? WHERE id = ?",
			raw, nowTimeStr(), e.ID).Error; err != nil {
			return err
		}
	}
	return nil
}

var _ domaingroupregion.Repository = (*GroupRegionRepository)(nil)
