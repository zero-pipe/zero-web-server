package persistence

import (
	"fmt"
	"strings"

	domaintree "zero-web-kit/internal/domain/tree"

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
FROM wvp_common_group
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
	if err := r.db.Raw("SELECT id, device_id, name FROM wvp_common_group WHERE id = ?", id).Scan(&row).Error; err != nil {
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
       COALESCE(gb_device_id, device_id) AS device_id,
       COALESCE(gb_name, name) AS name,
       COALESCE(gb_parent_id, parent_id) AS parent_device_id,
       COALESCE(gb_business_group_id, business_group_id) AS business_group,
       COALESCE(gb_status, status) AS status,
       1 AS type,
       1 AS is_leaf
FROM wvp_device_channel
WHERE channel_type = 0 AND COALESCE(gb_parent_id, parent_id) = ?`
	args := []any{parentDeviceID}
	if query != "" {
		sql += " AND (COALESCE(gb_device_id, device_id) LIKE ? OR COALESCE(gb_name, name) LIKE ?)"
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
FROM wvp_common_region
WHERE 1=1`
	args := make([]any, 0, 1)
	if parentID != nil {
		sql += " AND parent_id = ?"
		args = append(args, *parentID)
	} else {
		sql += " AND parent_id IS NULL"
	}
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
	if err := r.db.Raw("SELECT id, device_id, name FROM wvp_common_region WHERE id = ?", id).Scan(&row).Error; err != nil {
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
       COALESCE(gb_device_id, device_id) AS device_id,
       COALESCE(gb_name, name) AS name,
       COALESCE(gb_parent_id, parent_id) AS parent_device_id,
       COALESCE(gb_status, status) AS status,
       1 AS type,
       1 AS is_leaf
FROM wvp_device_channel
WHERE channel_type = 0 AND COALESCE(gb_civil_code, civil_code) = ?`
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

type RegionRecord struct {
	ID             int    `json:"id"`
	DeviceID       string `json:"deviceId"`
	Name           string `json:"name"`
	ParentID       *int   `json:"parentId"`
	ParentDeviceID string `json:"parentDeviceId"`
}

func (r *GroupRegionRepository) AddRegion(region *RegionRecord) error {
	now := nowTimeStr()
	parentDeviceID := strings.TrimSpace(region.ParentDeviceID)
	return r.db.Exec(`
INSERT INTO wvp_common_region (device_id, name, parent_id, parent_device_id, create_time, update_time)
VALUES (?, ?, ?, NULLIF(?, ''), ?, ?)`,
		region.DeviceID, region.Name, region.ParentID, parentDeviceID, now, now,
	).Error
}

func (r *GroupRegionRepository) UpdateRegion(region *RegionRecord) error {
	parentDeviceID := strings.TrimSpace(region.ParentDeviceID)
	return r.db.Exec(`
UPDATE wvp_common_region
SET device_id = ?, name = ?, parent_id = ?, parent_device_id = NULLIF(?, ''), update_time = ?
WHERE id = ?`,
		region.DeviceID, region.Name, region.ParentID, parentDeviceID, nowTimeStr(), region.ID,
	).Error
}

type GroupRecord struct {
	ID             int    `json:"id"`
	DeviceID       string `json:"deviceId"`
	Name           string `json:"name"`
	ParentID       *int   `json:"parentId"`
	ParentDeviceID string `json:"parentDeviceId"`
	BusinessGroup  string `json:"businessGroup"`
	CivilCode      string `json:"civilCode"`
	Alias          string `json:"alias"`
}

func (r *GroupRegionRepository) GetGroupRecordByID(id int) (*GroupRecord, error) {
	var row GroupRecord
	if err := r.db.Raw(`
SELECT id, device_id, name, parent_id, COALESCE(parent_device_id, '') AS parent_device_id,
       business_group, COALESCE(civil_code, '') AS civil_code, COALESCE(alias, '') AS alias
FROM wvp_common_group WHERE id = ?`, id).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *GroupRegionRepository) GetGroupByDeviceID(deviceID string) (*GroupRecord, error) {
	var row GroupRecord
	if err := r.db.Raw(`
SELECT id, device_id, name, parent_id, COALESCE(parent_device_id, '') AS parent_device_id,
       business_group, COALESCE(civil_code, '') AS civil_code, COALESCE(alias, '') AS alias
FROM wvp_common_group WHERE device_id = ?`, deviceID).Scan(&row).Error; err != nil {
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

func (r *GroupRegionRepository) GetGroupByDeviceAndBusiness(deviceID, businessGroup string) (*GroupRecord, error) {
	var row GroupRecord
	if err := r.db.Raw(`
SELECT id, device_id, name, parent_id, COALESCE(parent_device_id, '') AS parent_device_id,
       business_group, COALESCE(civil_code, '') AS civil_code, COALESCE(alias, '') AS alias
FROM wvp_common_group WHERE device_id = ? AND business_group = ?`, deviceID, businessGroup).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *GroupRegionRepository) GetBusinessGroup(businessGroup string) (*GroupRecord, error) {
	var row GroupRecord
	if err := r.db.Raw(`
SELECT id, device_id, name, parent_id, COALESCE(parent_device_id, '') AS parent_device_id,
       business_group, COALESCE(civil_code, '') AS civil_code, COALESCE(alias, '') AS alias
FROM wvp_common_group WHERE device_id = ? AND business_group = ?`, businessGroup, businessGroup).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *GroupRegionRepository) ExistsGroupDeviceID(deviceID string, excludeID int) (bool, error) {
	var count int64
	q := r.db.Raw(`SELECT COUNT(1) FROM wvp_common_group WHERE device_id = ?`, deviceID)
	if excludeID > 0 {
		q = r.db.Raw(`SELECT COUNT(1) FROM wvp_common_group WHERE device_id = ? AND id <> ?`, deviceID, excludeID)
	}
	if err := q.Scan(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *GroupRegionRepository) AddBusinessGroup(group *GroupRecord) error {
	now := nowTimeStr()
	civilCode := strings.TrimSpace(group.CivilCode)
	alias := strings.TrimSpace(group.Alias)
	return r.db.Exec(`
INSERT INTO wvp_common_group (device_id, name, business_group, create_time, update_time, civil_code, alias)
VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))`,
		group.DeviceID, group.Name, group.DeviceID, now, now, civilCode, alias,
	).Error
}

func (r *GroupRegionRepository) AddVirtualGroup(group *GroupRecord) error {
	now := nowTimeStr()
	parentDeviceID := strings.TrimSpace(group.ParentDeviceID)
	civilCode := strings.TrimSpace(group.CivilCode)
	alias := strings.TrimSpace(group.Alias)
	return r.db.Exec(`
INSERT INTO wvp_common_group (device_id, name, parent_id, parent_device_id, business_group, create_time, update_time, civil_code, alias)
VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))`,
		group.DeviceID, group.Name, group.ParentID, parentDeviceID, group.BusinessGroup, now, now, civilCode, alias,
	).Error
}

func (r *GroupRegionRepository) UpdateGroup(group *GroupRecord) error {
	parentDeviceID := strings.TrimSpace(group.ParentDeviceID)
	civilCode := strings.TrimSpace(group.CivilCode)
	alias := strings.TrimSpace(group.Alias)
	return r.db.Exec(`
UPDATE wvp_common_group
SET device_id = ?, name = ?, parent_id = ?, parent_device_id = NULLIF(?, ''), business_group = ?,
    civil_code = NULLIF(?, ''), alias = NULLIF(?, ''), update_time = ?
WHERE id = ?`,
		group.DeviceID, group.Name, group.ParentID, parentDeviceID, group.BusinessGroup,
		civilCode, alias, nowTimeStr(), group.ID,
	).Error
}

func (r *GroupRegionRepository) ListGroupsByBusinessGroup(businessGroup string) ([]GroupRecord, error) {
	var rows []GroupRecord
	err := r.db.Raw(`
SELECT id, device_id, name, parent_id, COALESCE(parent_device_id, '') AS parent_device_id,
       business_group, COALESCE(civil_code, '') AS civil_code, COALESCE(alias, '') AS alias
FROM wvp_common_group WHERE business_group = ? AND device_id <> ?`, businessGroup, businessGroup).Scan(&rows).Error
	if rows == nil {
		rows = []GroupRecord{}
	}
	return rows, err
}

func (r *GroupRegionRepository) ListGroupChildren(parentID int) ([]GroupRecord, error) {
	var rows []GroupRecord
	err := r.db.Raw(`
SELECT id, device_id, name, parent_id, COALESCE(parent_device_id, '') AS parent_device_id,
       business_group, COALESCE(civil_code, '') AS civil_code, COALESCE(alias, '') AS alias
FROM wvp_common_group WHERE parent_id = ?`, parentID).Scan(&rows).Error
	if rows == nil {
		rows = []GroupRecord{}
	}
	return rows, err
}

func (r *GroupRegionRepository) CollectGroupDescendants(rootID int) ([]GroupRecord, error) {
	root, err := r.GetGroupRecordByID(rootID)
	if err != nil {
		return nil, err
	}
	out := []GroupRecord{*root}
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
	return r.db.Exec(`DELETE FROM wvp_common_group WHERE id IN ?`, ids).Error
}

func (r *GroupRegionRepository) ClearChannelsByBusinessGroup(businessGroup string) error {
	return r.db.Exec(`
UPDATE wvp_device_channel
SET gb_parent_id = NULL, parent_id = NULL, gb_business_group_id = NULL, business_group_id = NULL, update_time = ?
WHERE channel_type = 0 AND COALESCE(gb_business_group_id, business_group_id) = ?`, nowTimeStr(), businessGroup).Error
}

func (r *GroupRegionRepository) ClearChannelsByParentDeviceIDs(deviceIDs []string) error {
	if len(deviceIDs) == 0 {
		return nil
	}
	return r.db.Exec(`
UPDATE wvp_device_channel
SET gb_parent_id = NULL, parent_id = NULL, gb_business_group_id = NULL, business_group_id = NULL, update_time = ?
WHERE channel_type = 0 AND COALESCE(gb_parent_id, parent_id) IN ?`, nowTimeStr(), deviceIDs).Error
}
