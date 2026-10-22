package backup

import (
	"context"
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func workerTestDB(t *testing.T) {
	t.Helper()
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(sqlite.Open(filepath.Join(config.Load().DataDir, "jobs.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&models.MaintenanceJob{}, &models.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	previousEngine := builtin.Engine
	builtin.Engine = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	builtin.SetReadOnly(false)
	t.Cleanup(func() {
		database.DB = previous
		builtin.Engine = previousEngine
		builtin.SetReadOnly(false)
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
}
func TestWorkerProcessesGCAndKeepsManualReadOnly(t *testing.T) {
	workerTestDB(t)
	builtin.SetReadOnly(true)
	job := models.MaintenanceJob{Kind: "registry_gc", Status: "queued", UserID: 1}
	database.DB.Create(&job)
	runNext(context.Background())
	database.DB.First(&job, job.ID)
	if job.Status != "success" || job.Error != "" {
		t.Fatalf("GC job did not complete: %+v", job)
	}
	if !builtin.ReadOnly() || builtin.MaintenanceState().Operation != "" {
		t.Fatal("GC did not retain the manual maintenance window")
	}
	var events int64
	database.DB.Model(&models.AuditEvent{}).Where("action = ?", "registry.gc.complete").Count(&events)
	if events != 1 {
		t.Fatal("GC result was not audited")
	}
}
func TestWorkerDoesNotRunGCWithoutManualWindow(t *testing.T) {
	workerTestDB(t)
	job := models.MaintenanceJob{Kind: "registry_gc", Status: "queued"}
	database.DB.Create(&job)
	runNext(context.Background())
	database.DB.First(&job, job.ID)
	if job.Status != "failed" || job.Error == "" || builtin.ReadOnly() {
		t.Fatalf("GC incorrectly enabled writes or ran without a window: %+v", job)
	}
}
func TestWorkerRequeuesWhenOtherMaintenanceOwnsStorage(t *testing.T) {
	workerTestDB(t)
	builtin.SetReadOnly(true)
	finish, err := builtin.BeginMaintenance(context.Background(), "backup")
	if err != nil {
		t.Fatal(err)
	}
	job := models.MaintenanceJob{Kind: "registry_gc", Status: "queued"}
	database.DB.Create(&job)
	runNext(context.Background())
	finish()
	database.DB.First(&job, job.ID)
	if job.Status != "queued" {
		t.Fatalf("busy registry permanently failed a queued job: %+v", job)
	}
}
func TestBackupFailureCleansPartialArchiveAndRestoresManualState(t *testing.T) {
	workerTestDB(t)
	for _, manual := range []bool{false, true} {
		builtin.SetReadOnly(manual)
		// Fail late in archive creation, after the database snapshot was written.
		os.Remove(filepath.Join(config.Load().DataDir, "jobs"))
		if _, err := Create(context.Background()); err == nil {
			t.Fatal("backup with missing source directory unexpectedly succeeded")
		}
		partials, _ := filepath.Glob(filepath.Join(config.Load().DataDir, "backups", "*.partial"))
		if len(partials) != 0 {
			t.Fatalf("failed backup leaked partial archives: %v", partials)
		}
		if state := builtin.MaintenanceState(); state.ReadOnly != manual || state.ManualReadOnly != manual || state.Operation != "" {
			t.Fatalf("backup failure changed the manual state: %+v", state)
		}
	}
}

func TestWorkerRestartMarksInterruptedTasksOnly(t *testing.T) {
	workerTestDB(t)
	for _, kind := range []string{"backup", "registry_gc", "unrelated"} {
		database.DB.Create(&models.MaintenanceJob{Kind: kind, Status: "running"})
	}
	Start()
	defer Stop()
	var jobs []models.MaintenanceJob
	database.DB.Order("id").Find(&jobs)
	for _, job := range jobs {
		if job.Kind == "unrelated" {
			if job.Status != "running" {
				t.Fatal("unrelated worker state was modified")
			}
			continue
		}
		if job.Status != "failed" || job.Error == "" {
			t.Fatalf("interrupted maintenance task was not reported: %+v", job)
		}
	}
}
