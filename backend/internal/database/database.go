package database

import (
	"dockerun/internal/config"
	"dockerun/internal/models"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(dbPath string) error {
	cfg := config.Load()
	var db *gorm.DB
	var err error

	// 根据配置选择数据库类型
	if cfg.DatabaseType == "mysql" && cfg.MySQLDSN != "" {
		// 使用 MySQL
		log.Printf("Using MySQL database")
		db, err = gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{Logger: logger.New(log.New(os.Stderr, "", log.LstdFlags), logger.Config{LogLevel: logger.Warn, IgnoreRecordNotFoundError: true, SlowThreshold: time.Second})})
		if err != nil {
			return fmt.Errorf("failed to connect to MySQL: %w", err)
		}
	} else {
		// 默认使用 SQLite
		log.Printf("Using SQLite database: %s", dbPath)

		// 确保数据目录存在
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		// 连接数据库
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{Logger: logger.New(log.New(os.Stderr, "", log.LstdFlags), logger.Config{LogLevel: logger.Warn, IgnoreRecordNotFoundError: true, SlowThreshold: time.Second})})
		if err != nil {
			return err
		}
	}

	if cfg.DatabaseType != "mysql" && db.Migrator().HasTable("users") && !db.Migrator().HasTable("migrations") {
		backupDir := filepath.Join(cfg.DataDir, "backups")
		if err := os.MkdirAll(backupDir, 0700); err != nil {
			return err
		}
		path := filepath.Join(backupDir, fmt.Sprintf("pre-v1-%d.db", time.Now().UnixNano()))
		if err := db.Exec("VACUUM INTO ?", path).Error; err != nil {
			return fmt.Errorf("pre-migration backup: %w", err)
		}
	}
	if cfg.DatabaseType == "mysql" && db.Migrator().HasTable("users") && !db.Migrator().HasTable("migrations") {
		if err := backupLegacyMySQL(cfg); err != nil {
			return err
		}
	}
	// 自动迁移
	if err := db.AutoMigrate(
		&models.User{},
		&models.DeployScript{},
		&models.DeployScriptFile{},
		&models.ApplicationTemplate{},
		&models.Registry{},
		&models.SystemConfig{},
		&models.WebhookConfig{},
		&models.WebhookLog{},
		&models.Migration{}, &models.AuditEvent{}, &models.Application{}, &models.Environment{}, &models.Release{}, &models.Deployment{}, &models.TaskLog{}, &models.Bundle{}, &models.ImageEvent{}, &models.AutomationToken{}, &models.Target{}, &models.TargetState{}, &models.MaintenanceJob{},
	); err != nil {
		return err
	}
	if err := migrateTemplateKeys(db); err != nil {
		return fmt.Errorf("应用模板标识迁移失败：%w", err)
	}

	DB = db
	if cfg.DatabaseType != "mysql" {
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA busy_timeout=10000")
		sqlDB, _ := db.DB()
		sqlDB.SetMaxOpenConns(1)
	}

	// 检查是否需要创建默认管理员
	var userCount int64
	DB.Model(&models.User{}).Count(&userCount)
	if userCount == 0 {
		// 创建默认管理员账号（密码为空，标记需要首次设置）
		admin := &models.User{
			Username: "admin",
			IsAdmin:  true,
			Password: "", // 密码为空，标记需要首次设置
			Enabled:  true, RegistryAccess: "read_write", AuthVersion: 1,
		}
		if err := DB.Create(admin).Error; err != nil {
			return err
		}
	}

	// 检查是否需要创建默认系统配置
	var configCount int64
	DB.Model(&models.SystemConfig{}).Count(&configCount)
	if configCount == 0 {
		// 创建默认系统配置
		defaultConfig := &models.SystemConfig{
			DeployWorkDir:      filepath.Join(cfg.DataDir, "jobs"),
			MaxScriptExecutors: 5,
			ScriptTimeout:      3600,
			EnableScriptLog:    true,
		}
		if err := DB.Create(defaultConfig).Error; err != nil {
			return err
		}
	}
	var migration models.Migration
	if err := DB.First(&migration, 1).Error; err == gorm.ErrRecordNotFound {
		if err := DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.DeployScript{}).Where("auto_deploy = ?", true).Update("auto_deploy", false).Error; err != nil {
				return err
			}
			return tx.Create(&models.Migration{ID: 1, Name: "integrated-registry-and-deployment"}).Error
		}); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	var builtin int64
	DB.Model(&models.Registry{}).Where("kind = ?", "builtin").Count(&builtin)
	DB.Model(&models.Registry{}).Where("kind = ?", "builtin").Update("url", cfg.PublicURL)
	if builtin == 0 {
		var count int64
		DB.Model(&models.Registry{}).Count(&count)
		if err := DB.Create(&models.Registry{Name: "Dockerun 内置仓库", Kind: "builtin", URL: cfg.PublicURL, IsDefault: count == 0}).Error; err != nil {
			return err
		}
	}
	var targets int64
	DB.Model(&models.Target{}).Count(&targets)
	if targets == 0 {
		if err := DB.Create(&models.Target{Name: "本机", Enabled: true}).Error; err != nil {
			return err
		}
	}
	// Bundle paths refer to persisted materials, so restored data can move directories.
	var bundles []models.Bundle
	DB.Find(&bundles)
	for _, b := range bundles {
		candidate := filepath.Join(cfg.DataDir, "uploads", filepath.Base(b.Path))
		if _, err := os.Stat(candidate); err == nil && candidate != b.Path {
			DB.Model(&b).Update("path", candidate)
		}
	}

	return nil
}

func GetDB() *gorm.DB {
	return DB
}
