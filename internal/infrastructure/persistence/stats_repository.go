package persistence

import (
	"zero-web-server/internal/application/ops"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

// StatsRepository 提供仪表盘只读统计。
type StatsRepository struct{ db *gorm.DB }

func NewStatsRepository(db *gorm.DB) *StatsRepository { return &StatsRepository{db: db} }

func (r *StatsRepository) DeviceCount() (total, online int64, err error) {
	if err = r.db.Model(&model.Device{}).Count(&total).Error; err != nil {
		return
	}
	err = r.db.Model(&model.Device{}).Where("online = ?", true).Count(&online).Error
	return
}

func (r *StatsRepository) ChannelCount() (total, online int64, err error) {
	if err = r.db.Model(&model.Channel{}).Count(&total).Error; err != nil {
		return
	}
	err = r.db.Model(&model.Channel{}).Where("online = ?", true).Count(&online).Error
	return
}

func (r *StatsRepository) StreamPushCount(mediaServerID string) (total, online int64, err error) {
	q := r.db.Model(&model.StreamPush{})
	if mediaServerID != "" {
		q = q.Where("media_server_id = ?", mediaServerID)
	}
	if err = q.Count(&total).Error; err != nil {
		return
	}
	q2 := r.db.Model(&model.StreamPush{}).Where("pushing = ?", true)
	if mediaServerID != "" {
		q2 = q2.Where("media_server_id = ?", mediaServerID)
	}
	err = q2.Count(&online).Error
	return
}

func (r *StatsRepository) StreamProxyCount(mediaServerID string) (total, online int64, err error) {
	q := r.db.Model(&model.StreamProxy{})
	if mediaServerID != "" {
		q = q.Where("media_server_id = ?", mediaServerID)
	}
	if err = q.Count(&total).Error; err != nil {
		return
	}
	q2 := r.db.Model(&model.StreamProxy{}).Where("pulling = ?", true)
	if mediaServerID != "" {
		q2 = q2.Where("media_server_id = ?", mediaServerID)
	}
	err = q2.Count(&online).Error
	return
}

var _ ops.StatsRepository = (*StatsRepository)(nil)
