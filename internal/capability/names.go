package capability

// Names — 中台三大能力标识（菜单 / 日志 / 指标统一用这些常量）。
const (
	Access  = "access"  // 设备接入 + 国标级联
	Media   = "media"   // 流媒体集群与调度
	Storage = "storage" // 对象存储对接
	App     = "app"     // 应用入口：地图/分屏/报警
	Ops     = "ops"     // 运维
	IAM     = "user"    // 用户权限
	Org     = "org"     // 组织
)
