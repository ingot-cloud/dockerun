package database

import (
	"dockerun/internal/config"
	"dockerun/internal/models"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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
		db, err = gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
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
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			return err
		}
	}

	// 自动迁移
	if err := db.AutoMigrate(
		&models.User{},
		&models.DeployScript{},
		&models.DeployScriptFile{},
		&models.Registry{},
		&models.SystemConfig{},
		&models.WebhookConfig{},
		&models.WebhookLog{},
	); err != nil {
		return err
	}

	DB = db

	// 检查是否需要创建默认管理员
	var userCount int64
	DB.Model(&models.User{}).Count(&userCount)
	if userCount == 0 {
		// 创建默认管理员账号（密码为空，标记需要首次设置）
		admin := &models.User{
			Username: "admin",
			IsAdmin:  true,
			Password: "", // 密码为空，标记需要首次设置
		}
		DB.Create(admin)
	}

	// 检查是否需要创建默认系统配置
	var configCount int64
	DB.Model(&models.SystemConfig{}).Count(&configCount)
	if configCount == 0 {
		// 创建默认系统配置
		defaultConfig := &models.SystemConfig{
			DeployWorkDir:      "/ingot-data/dockerun",
			MaxScriptExecutors: 5,
			ScriptTimeout:      3600,
			EnableScriptLog:    true,
		}
		DB.Create(defaultConfig)
	}

	return nil
}

func GetDB() *gorm.DB {
	return DB
}
