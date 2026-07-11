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

// NewDB 连接 MySQL：库不存在则创建；表不存在则 AutoMigrate 创建；已有则复用（不删数据）。
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
	// 已有表：只补缺列；新建表：整表创建。字段均带 size，避免把带索引的 varchar 改成 longtext。
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
		&model.StreamPush{},
		&model.StreamProxy{},
		&model.CloudRecord{},
		&model.RecordPlan{},
		&model.RecordPlanItem{},
		&model.OnvifDevice{},
		&model.OnvifChannel{},
		&model.CommonGroup{},
		&model.CommonRegion{},
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
			ID: 1, Name: "admin", Authority: "0",
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
