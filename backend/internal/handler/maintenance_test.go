package handler

import (
	"bytes"
	"context"
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

func maintenanceTestSetup(t *testing.T) {
	t.Helper()
	builtinTestUser(t)
	if err := database.DB.AutoMigrate(&models.MaintenanceJob{}, &models.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATA_DIR", t.TempDir())
	builtin.Engine = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	if err := builtin.SetReadOnly(false); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { builtin.SetReadOnly(false) })
}
func maintenanceRequest(handler gin.HandlerFunc, method, path, body string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Set("user_id", uint(1))
	ctx.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler(ctx)
	return response
}
func TestMaintenanceRequestValidationAndQueuedGuard(t *testing.T) {
	maintenanceTestSetup(t)
	for _, body := range []string{"{}", `{"read_only":null}`, `{"gc":true,"read_only":false}`, `{"read_only":"false"}`} {
		if response := maintenanceRequest(RegistryMaintenance, "POST", "/", body); response.Code != 400 {
			t.Fatalf("invalid maintenance %s: %d %s", body, response.Code, response.Body.String())
		}
	}
	response := maintenanceRequest(RegistryMaintenance, "POST", "/", `{"gc":true}`)
	if response.Code != 409 {
		t.Fatalf("GC without manual window: %d %s", response.Code, response.Body.String())
	}
	response = maintenanceRequest(RegistryMaintenance, "POST", "/", `{"read_only":true}`)
	if response.Code != 200 || !builtin.ReadOnly() {
		t.Fatalf("enable maintenance: %d", response.Code)
	}
	response = maintenanceRequest(RegistryMaintenance, "POST", "/", `{"gc":true}`)
	if response.Code != 202 {
		t.Fatalf("enqueue GC: %d %s", response.Code, response.Body.String())
	}
	response = maintenanceRequest(RegistryMaintenance, "POST", "/", `{"read_only":false}`)
	if response.Code != 409 || !builtin.ReadOnly() {
		t.Fatal("queued GC did not protect the read-only window")
	}
	response = maintenanceRequest(RegistryMaintenance, "POST", "/", `{"gc":true}`)
	var reused struct {
		Reused bool `json:"reused"`
	}
	json.Unmarshal(response.Body.Bytes(), &reused)
	if response.Code != 200 || !reused.Reused {
		t.Fatalf("duplicate GC was not reused: %d %s", response.Code, response.Body.String())
	}
	database.DB.Model(&models.MaintenanceJob{}).Where("kind = ?", "registry_gc").Update("status", "success")
	response = maintenanceRequest(RegistryMaintenance, "POST", "/", `{"read_only":false}`)
	if response.Code != 200 || builtin.ReadOnly() {
		t.Fatal("completed GC prevented restoring writes")
	}
}
func TestMaintenanceCannotReenableWritesDuringBackup(t *testing.T) {
	maintenanceTestSetup(t)
	finish, err := builtin.BeginMaintenance(context.Background(), "backup")
	if err != nil {
		t.Fatal(err)
	}
	response := maintenanceRequest(RegistryMaintenance, "POST", "/", `{"read_only":false}`)
	finish()
	if response.Code != 409 {
		t.Fatalf("backup accepted write toggle: %d %s", response.Code, response.Body.String())
	}
	if builtin.ReadOnly() {
		t.Fatal("temporary backup read-only state leaked")
	}
}
func TestBackupAdmissionDeduplicatesConcurrentRequests(t *testing.T) {
	maintenanceTestSetup(t)
	const count = 8
	responses := make(chan *httptest.ResponseRecorder, count)
	var group sync.WaitGroup
	for i := 0; i < count; i++ {
		group.Add(1)
		go func() { defer group.Done(); responses <- maintenanceRequest(CreateBackup, "POST", "/", "") }()
	}
	group.Wait()
	close(responses)
	for response := range responses {
		if response.Code != 200 && response.Code != 202 {
			t.Fatalf("backup create %d %s", response.Code, response.Body.String())
		}
	}
	var jobs int64
	database.DB.Model(&models.MaintenanceJob{}).Where("kind = ?", "backup").Count(&jobs)
	if jobs != 1 {
		t.Fatalf("duplicate active backups created: %d", jobs)
	}
	response := maintenanceRequest(ListBackups, "GET", "/", "")
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	response = maintenanceRequest(ListRegistryMaintenance, "GET", "/", "")
	if response.Code != 200 || response.Body.String() != "[]" {
		t.Fatalf("GC list mixed backup jobs: %d %s", response.Code, response.Body.String())
	}
}
func TestBackupDownloadReportsMissingFiles(t *testing.T) {
	maintenanceTestSetup(t)
	job := models.MaintenanceJob{Kind: "backup", Status: "ready", File: "missing.tar.gz"}
	database.DB.Create(&job)
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(job.ID)}}
	DownloadBackup(ctx)
	if response.Code != 404 || !bytes.Contains(response.Body.Bytes(), []byte("备份文件已不存在")) {
		t.Fatalf("missing backup: %d %s", response.Code, response.Body.String())
	}
	dir := filepath.Join(config.Load().DataDir, "backups")
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, job.File), []byte("backup-material"), 0600)
	response = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(job.ID)}}
	DownloadBackup(ctx)
	if response.Code != 200 || response.Body.String() != "backup-material" || response.Header().Get("Content-Disposition") == "" {
		t.Fatalf("download: %d %s", response.Code, response.Body.String())
	}
}
func TestRegistryMaintenanceStatusDuringBackup(t *testing.T) {
	maintenanceTestSetup(t)
	finish, err := builtin.BeginMaintenance(context.Background(), "backup")
	if err != nil {
		t.Fatal(err)
	}
	response := maintenanceRequest(BuiltinStatus, "GET", "/", "")
	finish()
	var status struct {
		Enabled   bool   `json:"enabled"`
		ReadOnly  bool   `json:"read_only"`
		Manual    bool   `json:"manual_read_only"`
		Operation string `json:"operation"`
		Bytes     int64  `json:"storage_bytes"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || !status.Enabled || !status.ReadOnly || status.Manual || status.Operation != "backup" {
		t.Fatalf("incorrect status: %+v", status)
	}
}
