package mysql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"zero-web-server/internal/infrastructure/config"
	"zero-web-server/internal/infrastructure/persistence/model"
	"zero-web-server/pkg/idcode"
	applog "zero-web-server/pkg/log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/go-sql-driver/mysql"
)

// NewDB 连接 MySQL：库不存在则创建。
// 开发期权威建表：sql/init_zws_mysql.sql；AutoMigrate 仅作启动兜底补缺。
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
	if err := ensureInternalCodes(db); err != nil {
		return err
	}
	return nil
}

// ensureInternalCodes 为空内码行补生成（开发期未清库时 AutoMigrate 补列后用）。
func ensureInternalCodes(db *gorm.DB) error {
	type rowID struct {
		ID int64 `gorm:"column:id"`
	}
	backfill := func(table string, gen func() (string, error)) error {
		var rows []rowID
		if err := db.Raw(
			"SELECT id FROM `"+table+"` WHERE internal_code IS NULL OR TRIM(internal_code) = ''",
		).Scan(&rows).Error; err != nil {
			if strings.Contains(err.Error(), "doesn't exist") {
				return nil
			}
			return fmt.Errorf("list empty internal_code on %s: %w", table, err)
		}
		for _, row := range rows {
			code, err := gen()
			if err != nil {
				return err
			}
			if err := db.Exec(
				"UPDATE `"+table+"` SET internal_code = ? WHERE id = ? AND (internal_code IS NULL OR TRIM(internal_code) = '')",
				code, row.ID,
			).Error; err != nil {
				return fmt.Errorf("backfill %s id=%d: %w", table, row.ID, err)
			}
		}
		if len(rows) > 0 {
			applog.Info("backfilled internal codes", "table", table, "count", len(rows))
		}
		return nil
	}
	if err := backfill("zws_device", idcode.Device); err != nil {
		return err
	}
	if err := backfill("zws_device_channel", idcode.Channel); err != nil {
		return err
	}
	if err := backfill("zws_onvif_device", idcode.Device); err != nil {
		return err
	}
	if err := backfill("zws_onvif_channel", idcode.Channel); err != nil {
		return err
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
