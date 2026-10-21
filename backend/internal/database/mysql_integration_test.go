package database_test

import (
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"os"
	"testing"
)

func TestMySQLMigrationAndParameters(t *testing.T) {
	if os.Getenv("DOCKERUN_MYSQL_TESTS") != "1" {
		t.Skip("requires dedicated MySQL fixture")
	}
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("DATABASE_TYPE", "mysql")
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := database.Init(""); err != nil {
		t.Fatal(err)
	}
	var admin models.User
	if err := database.DB.First(&admin, 1).Error; err != nil || !admin.IsAdmin || !admin.Enabled {
		t.Fatalf("admin migration: %+v %v", admin, err)
	}
	a := models.Application{Name: "mysql-fixture"}
	if err := database.DB.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	e := models.Environment{ApplicationID: a.ID, Name: "test", Parameters: models.StringMap{"VALUE": "中文 and = signs"}}
	if err := database.DB.Create(&e).Error; err != nil {
		t.Fatal(err)
	}
	var loaded models.Environment
	if err := database.DB.First(&loaded, e.ID).Error; err != nil || loaded.Parameters["VALUE"] != e.Parameters["VALUE"] {
		t.Fatalf("JSON parameters lost: %+v %v", loaded, err)
	}
	if err := database.Init(""); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
}
