package database

import (
	"dockerun/internal/models"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestTemplateKeyMigrationPreservesOldIDsAndAllowsNamedCopies(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "templates.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	connection, _ := db.DB()
	defer connection.Close()
	if err = db.Exec("CREATE TABLE application_templates (id integer PRIMARY KEY, name text, version text)").Error; err != nil {
		t.Fatal(err)
	}
	db.Exec("CREATE UNIQUE INDEX template_version ON application_templates(name, version)")
	db.Exec("INSERT INTO application_templates(id,name,version) VALUES (7,'旧模板甲','1.0.0'),(12,'旧模板乙','1.0.0')")
	if err = db.AutoMigrate(&models.ApplicationTemplate{}); err != nil {
		t.Fatal(err)
	}
	if err = migrateTemplateKeys(db); err != nil {
		t.Fatal(err)
	}
	var rows []models.ApplicationTemplate
	db.Order("id").Find(&rows)
	if len(rows) != 2 || rows[0].ID != 7 || rows[1].ID != 12 || rows[0].TemplateKey == "" || rows[0].TemplateKey == rows[1].TemplateKey {
		t.Fatal("旧模板标识迁移错误", rows)
	}
	key := rows[0].TemplateKey
	if err = migrateTemplateKeys(db); err != nil {
		t.Fatal(err)
	}
	db.First(&rows[0], 7)
	if rows[0].TemplateKey != key {
		t.Fatal("稳定标识被重新生成")
	}
	if db.Migrator().HasIndex(&models.ApplicationTemplate{}, "template_version") {
		t.Fatal("旧名称唯一索引未移除")
	}
	copy := models.ApplicationTemplate{TemplateKey: "named-copy", Name: rows[0].Name, Version: rows[0].Version, Source: "imported"}
	if err = db.Create(&copy).Error; err != nil {
		t.Fatal("同名不同标识模板无法共存", err)
	}
	duplicate := copy
	duplicate.ID = 0
	duplicate.Name = "别名"
	if err = db.Create(&duplicate).Error; err == nil {
		t.Fatal("标识与版本唯一约束失效")
	}
}
