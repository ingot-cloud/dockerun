package database

import (
	"dockerun/internal/models"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(dbPath string) error {
	// 确保数据目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}

	// 自动迁移
	if err := db.AutoMigrate(
		&models.User{},
		&models.DeployScript{},
		&models.DeployScriptFile{},
		&models.Registry{},
		&models.SystemConfig{},
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
			DeployWorkDir:      "/opt/dockerun",
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
