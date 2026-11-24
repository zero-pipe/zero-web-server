package persistence

import (
	"gorm.io/gorm"
)

// AssetProjector 曾负责把 GB/ONVIF/插件旧表投影到统一资产模型。
// 统一 schema 落地后，设备/通道/接入/扩展本就在同一组表，删除由外键级联完成；
// 投影与孤儿清理不再需要，保留入口以兼容启动/删除后的调用。
type AssetProjector struct{ db *gorm.DB }

func NewAssetProjector(db *gorm.DB) *AssetProjector { return &AssetProjector{db: db} }

// Backfill 统一后为空操作。
func (p *AssetProjector) Backfill() error { return nil }

// Prune 统一后由外键级联处理，空操作。
func (p *AssetProjector) Prune() error { return nil }

// onlineStatus 在线状态字符串（多处复用）。
func onlineStatus(online bool) string {
	if online {
		return "ON"
	}
	return "OFF"
}
