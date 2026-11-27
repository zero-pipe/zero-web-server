package persistence

import (
	domainstoragenode "zero-web-server/internal/domain/storagenode"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type StorageNodeRepository struct{ db *gorm.DB }

func NewStorageNodeRepository(db *gorm.DB) *StorageNodeRepository {
	return &StorageNodeRepository{db: db}
}

func (r *StorageNodeRepository) ListAll() ([]domainstoragenode.StorageNode, error) {
	var rows []model.StorageNode
	if err := r.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domainstoragenode.StorageNode, 0, len(rows))
	for i := range rows {
		out = append(out, *toDomainStorageNode(&rows[i]))
	}
	return out, nil
}

func (r *StorageNodeRepository) GetByNodeID(nodeID string) (*domainstoragenode.StorageNode, error) {
	var m model.StorageNode
	if err := r.db.Where("node_id = ?", nodeID).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainStorageNode(&m), nil
}

func (r *StorageNodeRepository) Save(d *domainstoragenode.StorageNode) error {
	m := toModelStorageNode(d)
	var existing model.StorageNode
	err := r.db.Where("node_id = ?", m.NodeID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(m).Error
	}
	if err != nil {
		return err
	}
	m.ID = existing.ID
	return r.db.Save(m).Error
}

func (r *StorageNodeRepository) Delete(nodeID string) error {
	return r.db.Delete(&model.StorageNode{}, "node_id = ?", nodeID).Error
}

func (r *StorageNodeRepository) UpdateLastSeen(nodeID string, ts int64, online bool) error {
	return r.db.Model(&model.StorageNode{}).Where("node_id = ?", nodeID).
		Updates(map[string]any{"last_seen": ts, "online": online}).Error
}

func (r *StorageNodeRepository) ClearAllDefault() error {
	return r.db.Model(&model.StorageNode{}).Where("default_node = ?", true).
		Update("default_node", false).Error
}

func toModelStorageNode(d *domainstoragenode.StorageNode) *model.StorageNode {
	return &model.StorageNode{
		ID: d.ID, NodeID: d.NodeID, Name: d.Name, IP: d.IP,
		HTTPPort: d.HTTPPort, RTSPPort: d.RTSPPort, Secret: d.Secret, Version: d.Version,
		CapacityGB: d.CapacityGB, UsedGB: d.UsedGB, Online: d.Online, LastSeen: d.LastSeen,
		DefaultNode: d.DefaultNode, CreateTime: d.CreateTime, UpdateTime: d.UpdateTime,
	}
}

func toDomainStorageNode(m *model.StorageNode) *domainstoragenode.StorageNode {
	return &domainstoragenode.StorageNode{
		ID: m.ID, NodeID: m.NodeID, Name: m.Name, IP: m.IP,
		HTTPPort: m.HTTPPort, RTSPPort: m.RTSPPort, Secret: m.Secret, Version: m.Version,
		CapacityGB: m.CapacityGB, UsedGB: m.UsedGB, Online: m.Online, LastSeen: m.LastSeen,
		DefaultNode: m.DefaultNode, CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

var _ domainstoragenode.Repository = (*StorageNodeRepository)(nil)
