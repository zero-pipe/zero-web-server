package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"zero-web-kit/internal/infrastructure/config"
	"zero-web-kit/internal/infrastructure/persistence/model"
	applog "zero-web-kit/pkg/log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/go-sql-driver/mysql"
)

// NewDB 连接 MySQL：库不存在则创建。
// 表结构原则：新安装必须先执行 sql/init_zws_mysql.sql 全量脚本；
// AutoMigrate 仅用于已有库升级时补缺表/缺列，不是新装建表手段。
func NewDB(cfg config.MySQLConfig) (*gorm.DB, error) {
	if cfg.Charset == "" {
		cfg.Charset = "utf8mb4"
	}
	if cfg.Database == "" {
		return nil, fmt.Errorf("mysql.database 不能为空")
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 10
	}
	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = 100
	}

	if err := ensureDatabase(cfg); err != nil {
		return nil, err
	}

	gormCfg := &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Warn),
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN()), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("connect mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := autoMigrateAll(db); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	if err := seedAdminIfEmpty(db); err != nil {
		return nil, fmt.Errorf("seed admin: %w", err)
	}
	if err := seedDefaultRoles(db); err != nil {
		return nil, fmt.Errorf("seed roles: %w", err)
	}

	applog.Info("mysql ready", "database", cfg.Database)
	return db, nil
}

func ensureDatabase(cfg config.MySQLConfig) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=%s&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Charset)
	raw, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open mysql bootstrap: %w", err)
	}
	defer raw.Close()

	raw.SetConnMaxLifetime(time.Minute)
	if err := raw.Ping(); err != nil {
		return fmt.Errorf("ping mysql: %w", err)
	}

	stmt := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		cfg.Database,
	)
	if _, err := raw.Exec(stmt); err != nil {
		return fmt.Errorf("create database %s: %w", cfg.Database, err)
	}
	return nil
}

func autoMigrateAll(db *gorm.DB) error {
	// 升级路径：已有表只补缺列；若运维未跑全量 SQL 而缺整表，此处会创建（不应作为新装依赖）。
	models := []any{
		&model.UserRole{},
		&model.User{},
		&model.MediaServer{},
		&model.GBDevice{},
		&model.GBDeviceChannel{},
		&model.Alarm{},
		&model.MobilePosition{},
		&model.Platform{},
		&model.PlatformChannel{},
		&model.SubordinatePlatform{},
		&model.StreamPush{},
		&model.StreamProxy{},
		&model.CloudRecord{},
		&model.RecordPlan{},
		&model.RecordPlanItem{},
		&model.OnvifDevice{},
		&model.OnvifChannel{},
		&model.CommonGroup{},
		&model.CommonRegion{},
		&model.GbSipConfig{},
		&model.ObjectStoreConfig{},
	}
	for _, m := range models {
		exists := db.Migrator().HasTable(m)
		if err := db.AutoMigrate(m); err != nil {
			return fmt.Errorf("%T: %w", m, err)
		}
		if !exists {
			applog.Info("created table", "model", fmt.Sprintf("%T", m))
		}
	}
	// 历史库若由精简 AutoMigrate 建表，可能缺行政区划相关列；再兜底补一次
	if err := ensureDeviceChannelColumns(db); err != nil {
		return err
	}
	return nil
}

func ensureDeviceChannelColumns(db *gorm.DB) error {
	type col struct {
		name string
		ddl  string
	}
	cols := []col{
		{"channel_type", "ALTER TABLE zws_device_channel ADD COLUMN channel_type INT NOT NULL DEFAULT 0 COMMENT '通道类型'"},
		{"civil_code", "ALTER TABLE zws_device_channel ADD COLUMN civil_code VARCHAR(50) NULL COMMENT '行政区划代码'"},
		{"business_group_id", "ALTER TABLE zws_device_channel ADD COLUMN business_group_id VARCHAR(255) NULL COMMENT '业务分组ID'"},
		{"gb_name", "ALTER TABLE zws_device_channel ADD COLUMN gb_name VARCHAR(255) NULL"},
		{"gb_civil_code", "ALTER TABLE zws_device_channel ADD COLUMN gb_civil_code VARCHAR(255) NULL"},
		{"gb_parent_id", "ALTER TABLE zws_device_channel ADD COLUMN gb_parent_id VARCHAR(255) NULL"},
		{"gb_status", "ALTER TABLE zws_device_channel ADD COLUMN gb_status VARCHAR(50) NULL"},
		{"gb_business_group_id", "ALTER TABLE zws_device_channel ADD COLUMN gb_business_group_id VARCHAR(50) NULL"},
		{"gb_longitude", "ALTER TABLE zws_device_channel ADD COLUMN gb_longitude DOUBLE NULL"},
		{"gb_latitude", "ALTER TABLE zws_device_channel ADD COLUMN gb_latitude DOUBLE NULL"},
		{"gb_ptz_type", "ALTER TABLE zws_device_channel ADD COLUMN gb_ptz_type INT NULL"},
	}
	for _, c := range cols {
		if db.Migrator().HasColumn(&model.GBDeviceChannel{}, c.name) {
			continue
		}
		if err := db.Exec(c.ddl).Error; err != nil {
			return fmt.Errorf("add column %s: %w", c.name, err)
		}
		applog.Info("added missing column", "table", "zws_device_channel", "column", c.name)
	}
	// 空串会被 COALESCE 当成有效值，导致树节点无名/离线；统一洗成 NULL
	_ = db.Exec(`UPDATE zws_device_channel SET gb_name = NULL WHERE gb_name IS NOT NULL AND TRIM(gb_name) = ''`).Error
	_ = db.Exec(`UPDATE zws_device_channel SET gb_status = NULL WHERE gb_status IS NOT NULL AND TRIM(gb_status) = ''`).Error
	_ = db.Exec(`UPDATE zws_device_channel SET gb_device_id = NULL WHERE gb_device_id IS NOT NULL AND TRIM(gb_device_id) = ''`).Error
	return nil
}

func seedAdminIfEmpty(db *gorm.DB) error {
	var roleCount int64
	if err := db.Model(&model.UserRole{}).Count(&roleCount).Error; err != nil {
		return err
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	if roleCount == 0 {
		role := &model.UserRole{
			ID: 1, Name: "管理员", Authority: "*",
			CreateTime: now, UpdateTime: now,
		}
		if err := db.Create(role).Error; err != nil {
			return err
		}
	}

	var userCount int64
	if err := db.Model(&model.User{}).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount == 0 {
		user := &model.User{
			ID: 1, Username: "admin",
			Password:   "21232f297a57a5a743894a0e4a801fc3", // admin
			RoleID:     1,
			CreateTime: now, UpdateTime: now,
			PushKey: "3e80d1762a324d5b0ff636e0bd16f1e3",
		}
		if err := db.Create(user).Error; err != nil {
			return err
		}
		applog.Info("seeded default admin user", "username", "admin", "password", "admin")
	}
	return nil
}

// seedDefaultRoles 升级已有库：管理员权限归一、补齐预置角色。
func seedDefaultRoles(db *gorm.DB) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_ = db.Model(&model.UserRole{}).Where("id = ?", 1).Updates(map[string]any{
		"authority": "*", "update_time": now,
	}).Error
	_ = db.Model(&model.UserRole{}).Where("id = ? AND name = ?", 1, "admin").
		Update("name", "管理员").Error

	presets := []struct {
		PreferID  int
		Name      string
		Authority string
	}{
		{2, "运维", `["ops","system"]`},
		{3, "视频值班", `["map","live","record","alarm"]`},
	}
	for _, p := range presets {
		var n int64
		if err := db.Model(&model.UserRole{}).Where("name = ?", p.Name).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		role := &model.UserRole{
			ID: p.PreferID, Name: p.Name, Authority: p.Authority,
			CreateTime: now, UpdateTime: now,
		}
		var idTaken int64
		_ = db.Model(&model.UserRole{}).Where("id = ?", p.PreferID).Count(&idTaken).Error
		if idTaken > 0 {
			role.ID = 0
		}
		if err := db.Create(role).Error; err != nil {
			return err
		}
		applog.Info("seeded role", "name", p.Name, "id", role.ID)
	}
	return nil
}
