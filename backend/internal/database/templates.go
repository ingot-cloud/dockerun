package database

import (
	"crypto/rand"
	"dockerun/internal/models"
	"strings"

	"gorm.io/gorm"
)

func NewTemplateKey() string { return "template-" + strings.ToLower(rand.Text()) }

// EnsureTemplateKeys 为旧记录补齐一次性稳定标识；条件更新防止并发请求改写已有标识。
func EnsureTemplateKeys(db *gorm.DB) error {
	var rows []models.ApplicationTemplate
	if err := db.Select("id").Where("template_key IS NULL OR template_key = ?", "").Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := db.Model(&models.ApplicationTemplate{}).Where("id = ? AND (template_key IS NULL OR template_key = ?)", row.ID, "").UpdateColumn("template_key", NewTemplateKey()).Error; err != nil {
			return err
		}
	}
	return nil
}

func migrateTemplateKeys(db *gorm.DB) error {
	if err := EnsureTemplateKeys(db); err != nil {
		return err
	}
	if db.Migrator().HasIndex(&models.ApplicationTemplate{}, "template_version") {
		return db.Migrator().DropIndex(&models.ApplicationTemplate{}, "template_version")
	}
	return nil
}
