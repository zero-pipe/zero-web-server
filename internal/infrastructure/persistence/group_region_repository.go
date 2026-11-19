package persistence

import (
	"fmt"
	"strings"

	domaingroupregion "zero-web-server/internal/domain/groupregion"
	domaintree "zero-web-server/internal/domain/tree"

	"gorm.io/gorm"
)

type GroupRegionRepository struct {
	db *gorm.DB
}

func NewGroupRegionRepository(db *gorm.DB) *GroupRegionRepository {
	return &GroupRegionRepository{db: db}
}

func (r *GroupRegionRepository) QueryGroupTree(query string, parentID *int) ([]domaintree.Node, error) {
	sql := `
SELECT id, device_id, name, parent_id, parent_device_id, business_group, civil_code, alias,
       CONCAT('group', id) AS tree_id, 0 AS type, 0 AS is_leaf, 'ON' AS status
FROM zws_common_group
WHERE 1=1`
	args := make([]any, 0, 3)
	if parentID != nil {
		sql += " AND parent_id = ?"
		args = append(args, *parentID)
	} else {
		sql += " AND parent_id IS NULL"
	}
	if query != "" {
		sql += " AND (device_id LIKE ? OR name LIKE ?)"
		like := fmt.Sprintf("%%%s%%", query)
		args = append(args, like, like)
	}
	sql += " ORDER BY device_id ASC"
	var rows []domaintree.Node
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
		ID       int    `gorm:"column:id"`
		DeviceID string `gorm:"column:device_id"`
		Name     string `gorm:"column:name"`
	}
	if err := r.db.Raw("SELECT id, device_id, name FROM zws_common_group WHERE id = ?", id).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &domaintree.Node{ID: row.ID, DeviceID: row.DeviceID, Name: row.Name}, nil
}

func (r *GroupRegionRepository) QueryGroupChannels(query, parentDeviceID string) ([]domaintree.Node, error) {
	sql := `
SELECT id,
       CONCAT('channel', id) AS tree_id,
       COALESCE(NULLIF(TRIM(gb_device_id), ''), device_id) AS device_id,
       COALESCE(NULLIF(TRIM(gb_name), ''), NULLIF(TRIM(name), ''), NULLIF(TRIM(gb_device_id), ''), '未命名') AS name,
       gb_parent_id AS parent_device_id,
       gb_business_group_id AS business_group,
       COALESCE(NULLIF(TRIM(gb_status), ''), NULLIF(TRIM(status), ''), 'OFF') AS status,
       1 AS type,
       1 AS is_leaf
FROM zws_device_channel
WHERE channel_type = 0 AND gb_parent_id = ?`
	args := []any{parentDeviceID}
	if query != "" {
		sql += " AND (COALESCE(NULLIF(TRIM(gb_device_id), ''), device_id) LIKE ? OR COALESCE(NULLIF(TRIM(gb_name), ''), name) LIKE ?)"
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

func (r *GroupRegionRepository) QueryRegionTree(parentID *int) ([]domaintree.Node, error) {
	sql := `
SELECT id, device_id, name, parent_id, parent_device_id,
       CONCAT('region', id) AS tree_id, 0 AS type, 0 AS is_leaf, 'ON' AS status
FROM zws_common_region
WHERE 1=1`
	args := make([]any, 0, 1)
	if parentID != nil {
		sql += " AND parent_id = ?"
		args = append(args, *parentID)
	} else {
		sql += " AND parent_id IS NULL"
	}
	sql += " ORDER BY device_id ASC"
	var rows []domaintree.Node
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
		ID       int    `gorm:"column:id"`
		DeviceID string `gorm:"column:device_id"`
		Name     string `gorm:"column:name"`
	}
	if err := r.db.Raw("SELECT id, device_id, name FROM zws_common_region WHERE id = ?", id).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &domaintree.Node{ID: row.ID, DeviceID: row.DeviceID, Name: row.Name}, nil
}

func (r *GroupRegionRepository) QueryRegionChannels(parentDeviceID string) ([]domaintree.Node, error) {
	sql := `
SELECT id,
       CONCAT('channel', id) AS tree_id,
       COALESCE(NULLIF(TRIM(gb_device_id), ''), device_id) AS device_id,
       COALESCE(NULLIF(TRIM(gb_name), ''), NULLIF(TRIM(name), ''), NULLIF(TRIM(gb_device_id), ''), '未命名') AS name,
       gb_parent_id AS parent_device_id,
       COALESCE(NULLIF(TRIM(gb_status), ''), NULLIF(TRIM(status), ''), 'OFF') AS status,
       1 AS type,
       1 AS is_leaf
FROM zws_device_channel
WHERE channel_type = 0 AND gb_civil_code = ?`
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

func (r *GroupRegionRepository) AddRegion(region *domaingroupregion.RegionRecord) error {
	now := nowTimeStr()
	parentDeviceID := strings.TrimSpace(region.ParentDeviceID)
	return r.db.Exec(`
INSERT INTO zws_common_region (device_id, name, parent_id, parent_device_id, create_time, update_time)
VALUES (?, ?, ?, NULLIF(?, ''), ?, ?)`,
		region.DeviceID, region.Name, region.ParentID, parentDeviceID, now, now,
	).Error
}

func (r *GroupRegionRepository) UpdateRegion(region *domaingroupregion.RegionRecord) error {
	parentDeviceID := strings.TrimSpace(region.ParentDeviceID)
	return r.db.Exec(`
UPDATE zws_common_region
SET device_id = ?, name = ?, parent_id = ?, parent_device_id = NULLIF(?, ''), update_time = ?
WHERE id = ?`,
		region.DeviceID, region.Name, region.ParentID, parentDeviceID, nowTimeStr(), region.ID,
	).Error
}

func (r *GroupRegionRepository) GetGroupRecordByID(id int) (*domaingroupregion.GroupRecord, error) {
	var row domaingroupregion.GroupRecord
	if err := r.db.Raw(`
SELECT id, device_id, name, parent_id, COALESCE(parent_device_id, '') AS parent_device_id,
       business_group, COALESCE(civil_code, '') AS civil_code, COALESCE(alias, '') AS alias
FROM zws_common_group WHERE id = ?`, id).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *GroupRegionRepository) GetGroupByDeviceID(deviceID string) (*domaingroupregion.GroupRecord, error) {
	var row domaingroupregion.GroupRecord
	if err := r.db.Raw(`
SELECT id, device_id, name, parent_id, COALESCE(parent_device_id, '') AS parent_device_id,
       business_group, COALESCE(civil_code, '') AS civil_code, COALESCE(alias, '') AS alias
FROM zws_common_group WHERE device_id = ?`, deviceID).Scan(&row).Error; err != nil {
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
	if err := r.db.Raw(`
SELECT id, device_id, name, parent_id, COALESCE(parent_device_id, '') AS parent_device_id,
       business_group, COALESCE(civil_code, '') AS civil_code, COALESCE(alias, '') AS alias
FROM zws_common_group WHERE device_id = ? AND business_group = ?`, deviceID, businessGroup).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *GroupRegionRepository) GetBusinessGroup(businessGroup string) (*domaingroupregion.GroupRecord, error) {
	var row domaingroupregion.GroupRecord
	if err := r.db.Raw(`
SELECT id, device_id, name, parent_id, COALESCE(parent_device_id, '') AS parent_device_id,
       business_group, COALESCE(civil_code, '') AS civil_code, COALESCE(alias, '') AS alias
FROM zws_common_group WHERE device_id = ? AND business_group = ?`, businessGroup, businessGroup).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *GroupRegionRepository) ExistsGroupDeviceID(deviceID string, excludeID int) (bool, error) {
	var count int64
	q := r.db.Raw(`SELECT COUNT(1) FROM zws_common_group WHERE device_id = ?`, deviceID)
	if excludeID > 0 {
		q = r.db.Raw(`SELECT COUNT(1) FROM zws_common_group WHERE device_id = ? AND id <> ?`, deviceID, excludeID)
	}
	if err := q.Scan(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *GroupRegionRepository) AddBusinessGroup(group *domaingroupregion.GroupRecord) error {
	now := nowTimeStr()
	civilCode := strings.TrimSpace(group.CivilCode)
	alias := strings.TrimSpace(group.Alias)
	return r.db.Exec(`
INSERT INTO zws_common_group (device_id, name, business_group, create_time, update_time, civil_code, alias)
VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))`,
		group.DeviceID, group.Name, group.DeviceID, now, now, civilCode, alias,
	).Error
}

func (r *GroupRegionRepository) AddVirtualGroup(group *domaingroupregion.GroupRecord) error {
	now := nowTimeStr()
	parentDeviceID := strings.TrimSpace(group.ParentDeviceID)
	civilCode := strings.TrimSpace(group.CivilCode)
	alias := strings.TrimSpace(group.Alias)
	return r.db.Exec(`
INSERT INTO zws_common_group (device_id, name, parent_id, parent_device_id, business_group, create_time, update_time, civil_code, alias)
VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))`,
		group.DeviceID, group.Name, group.ParentID, parentDeviceID, group.BusinessGroup, now, now, civilCode, alias,
	).Error
}

func (r *GroupRegionRepository) UpdateGroup(group *domaingroupregion.GroupRecord) error {
	parentDeviceID := strings.TrimSpace(group.ParentDeviceID)
	civilCode := strings.TrimSpace(group.CivilCode)
	alias := strings.TrimSpace(group.Alias)
	return r.db.Exec(`
UPDATE zws_common_group
SET device_id = ?, name = ?, parent_id = ?, parent_device_id = NULLIF(?, ''), business_group = ?,
    civil_code = NULLIF(?, ''), alias = NULLIF(?, ''), update_time = ?
WHERE id = ?`,
		group.DeviceID, group.Name, group.ParentID, parentDeviceID, group.BusinessGroup,
		civilCode, alias, nowTimeStr(), group.ID,
	).Error
}

func (r *GroupRegionRepository) ListGroupsByBusinessGroup(businessGroup string) ([]domaingroupregion.GroupRecord, error) {
	var rows []domaingroupregion.GroupRecord
	err := r.db.Raw(`
SELECT id, device_id, name, parent_id, COALESCE(parent_device_id, '') AS parent_device_id,
       business_group, COALESCE(civil_code, '') AS civil_code, COALESCE(alias, '') AS alias
FROM zws_common_group WHERE business_group = ? AND device_id <> ?`, businessGroup, businessGroup).Scan(&rows).Error
	if rows == nil {
		rows = []domaingroupregion.GroupRecord{}
	}
	return rows, err
}

func (r *GroupRegionRepository) ListGroupChildren(parentID int) ([]domaingroupregion.GroupRecord, error) {
	var rows []domaingroupregion.GroupRecord
	err := r.db.Raw(`
SELECT id, device_id, name, parent_id, COALESCE(parent_device_id, '') AS parent_device_id,
       business_group, COALESCE(civil_code, '') AS civil_code, COALESCE(alias, '') AS alias
FROM zws_common_group WHERE parent_id = ?`, parentID).Scan(&rows).Error
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
	return r.db.Exec(`DELETE FROM zws_common_group WHERE id IN ?`, ids).Error
}

func (r *GroupRegionRepository) ClearChannelsByBusinessGroup(businessGroup string) error {
	return r.db.Exec(`
UPDATE zws_device_channel
SET gb_parent_id = NULL, gb_business_group_id = NULL, update_time = ?
WHERE channel_type = 0 AND gb_business_group_id = ?`, nowTimeStr(), businessGroup).Error
}

func (r *GroupRegionRepository) ClearChannelsByParentDeviceIDs(deviceIDs []string) error {
	if len(deviceIDs) == 0 {
		return nil
	}
	return r.db.Exec(`
UPDATE zws_device_channel
SET gb_parent_id = NULL, gb_business_group_id = NULL, update_time = ?
WHERE channel_type = 0 AND gb_parent_id IN ?`, nowTimeStr(), deviceIDs).Error
}

var _ domaingroupregion.Repository = (*GroupRegionRepository)(nil)

// ListAllRegions 返回全部行政区划节点（device_id, name, parent_device_id）。
func (r *GroupRegionRepository) ListAllRegions() ([]domaintree.Node, error) {
	sql := `
SELECT id, device_id, name, parent_id, parent_device_id,
       CONCAT('region', id) AS tree_id, 0 AS type, 0 AS is_leaf, 'ON' AS status
FROM zws_common_region
ORDER BY device_id ASC`
	var rows []domaintree.Node
	if err := r.db.Raw(sql).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []domaintree.Node{}
	}
	return rows, nil
}

// ListAllGroups 返回全部业务分组(215)与虚拟组织(216)节点。
func (r *GroupRegionRepository) ListAllGroups() ([]domaintree.Node, error) {
	sql := `
SELECT id, device_id, name, parent_id, parent_device_id, business_group, civil_code, alias,
       CONCAT('group', id) AS tree_id, SUBSTRING(device_id, 11, 3) AS type, 0 AS is_leaf, 'ON' AS status
FROM zws_common_group
ORDER BY device_id ASC`
	var rows []domaintree.Node
	if err := r.db.Raw(sql).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []domaintree.Node{}
	}
	return rows, nil
}

// GetRegionByDeviceID 按区划/国标编号查节点。
func (r *GroupRegionRepository) GetRegionByDeviceID(deviceID string) (*domaintree.Node, error) {
	var row struct {
		ID             int    `gorm:"column:id"`
		DeviceID       string `gorm:"column:device_id"`
		Name           string `gorm:"column:name"`
		ParentDM       string `gorm:"column:parent_device_id"`
	}
	if err := r.db.Raw(`SELECT id, device_id, name, COALESCE(parent_device_id,'') AS parent_dm
FROM zws_common_region WHERE device_id = ?`, deviceID).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &domaintree.Node{ID: row.ID, DeviceID: row.DeviceID, Name: row.Name, ParentDeviceID: row.ParentDM}, nil
}

// ListRegionCodeMappings 返回全部区划编码映射。
func (r *GroupRegionRepository) ListRegionCodeMappings() ([]*domaingroupregion.RegionCodeMapping, error) {
	var rows []domaingroupregion.RegionCodeMapping
	if err := r.db.Raw(`SELECT region_device_id, gb_code FROM zws_region_code_mapping`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domaingroupregion.RegionCodeMapping, 0, len(rows))
	for i := range rows {
		out = append(out, &rows[i])
	}
	return out, nil
}

// UpsertRegionCodeMapping 写入/更新映射。
func (r *GroupRegionRepository) UpsertRegionCodeMapping(m *domaingroupregion.RegionCodeMapping) error {
	return r.db.Exec(`INSERT INTO zws_region_code_mapping (region_device_id, gb_code, update_time)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE gb_code = VALUES(gb_code), update_time = VALUES(update_time)`,
		m.RegionDeviceID, m.GBCode, nowTimeStr()).Error
}

// DeleteRegionCodeMapping 删除映射（恢复默认推导）。
func (r *GroupRegionRepository) DeleteRegionCodeMapping(regionDeviceID string) error {
	return r.db.Exec(`DELETE FROM zws_region_code_mapping WHERE region_device_id = ?`, regionDeviceID).Error
}
