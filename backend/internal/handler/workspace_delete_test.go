package handler

import (
	"dockerun/internal/config"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func workspaceDeleteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := registryTestDB(t)
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Application{}, &models.ApplicationTemplate{}, &models.Environment{}, &models.Release{}, &models.Deployment{}, &models.TaskLog{}, &models.TargetState{}, &models.Bundle{}, &models.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func workspaceDeleteRequest(t *testing.T, handler gin.HandlerFunc, id string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Params = gin.Params{{Key: "id", Value: id}}
	ctx.Request = httptest.NewRequest("DELETE", "/"+url.PathEscape(id), nil)
	ctx.Set("user_id", uint(7))
	handler(ctx)
	return response
}

func workspaceCreate(t *testing.T, db *gorm.DB, value any) {
	t.Helper()
	if err := db.Create(value).Error; err != nil {
		t.Fatal(err)
	}
}

func workspaceCount(t *testing.T, db *gorm.DB, model any, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(model).Count(&count).Error; err != nil || count != want {
		t.Fatalf("%T：记录数量 %d，预期 %d，错误 %v", model, count, want, err)
	}
}

func TestDeleteApplicationCascadesManagementRecordsOnly(t *testing.T) {
	db := workspaceDeleteDB(t)
	app, other := models.Application{Name: "删除的应用"}, models.Application{Name: "保留的应用"}
	workspaceCreate(t, db, &app)
	workspaceCreate(t, db, &other)
	for _, owner := range []models.Application{app, other} {
		env := models.Environment{ApplicationID: owner.ID, Name: "生产"}
		workspaceCreate(t, db, &env)
		version := models.Release{ApplicationID: owner.ID, Version: "v1", Source: "local"}
		workspaceCreate(t, db, &version)
		task := models.Deployment{EnvironmentID: env.ID, ReleaseID: version.ID, Status: "success", IdempotencyKey: owner.Name}
		workspaceCreate(t, db, &task)
		workspaceCreate(t, db, &models.TaskLog{DeploymentID: task.ID, Message: "部署日志"})
		workspaceCreate(t, db, &models.TargetState{EnvironmentID: env.ID, TargetID: 1, CurrentReleaseID: version.ID})
		db.Model(&env).Update("current_release_id", version.ID)
	}
	workspaceCreate(t, db, &models.ApplicationTemplate{Name: "独立模板", Version: "v1"})
	workspaceCreate(t, db, &models.Bundle{Name: "独立发布包", SHA256: strings.Repeat("a", 64), Status: "ready"})
	// 运行中的容器可能仍挂载任务目录中的配置文件，删除管理记录不能移除这些文件。
	runtimeConfig := filepath.Join(config.Load().DataDir, "jobs", "1", "runtime.conf")
	os.MkdirAll(filepath.Dir(runtimeConfig), 0700)
	os.WriteFile(runtimeConfig, []byte("运行中的配置"), 0600)
	response := workspaceDeleteRequest(t, DeleteApplication, fmt.Sprint(app.ID))
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	for _, model := range []any{&models.Application{}, &models.Environment{}, &models.Release{}, &models.Deployment{}, &models.TaskLog{}, &models.TargetState{}, &models.ApplicationTemplate{}, &models.Bundle{}} {
		workspaceCount(t, db, model, 1)
	}
	if _, err := os.Stat(runtimeConfig); err != nil {
		t.Fatal("删除应用移除了运行配置", err)
	}
	var audit models.AuditEvent
	if db.Where("action = ? AND resource = ? AND user_id = ?", "application.delete", fmt.Sprint(app.ID), 7).First(&audit).Error != nil {
		t.Fatal("缺少删除审计记录")
	}
	if response = workspaceDeleteRequest(t, DeleteApplication, fmt.Sprint(app.ID)); response.Code != 404 {
		t.Fatal("重复删除应返回不存在")
	}
	var staleEnvironment models.Environment
	db.Where("application_id = ?", other.ID).First(&staleEnvironment)
	if _, err := release.Enqueue(models.Environment{ID: staleEnvironment.ID, ApplicationID: app.ID}, models.Release{ID: 1, ApplicationID: app.ID}, 7, "stale-delete", false, 0); err == nil {
		t.Fatal("删除后的旧页面仍可排队发布")
	}
	if err := release.SaveRelease(&models.Release{ApplicationID: app.ID, Version: "stale"}); err == nil {
		t.Fatal("删除后的旧请求仍可创建孤立版本")
	}
}

func TestDeleteApplicationProtectsActiveTasks(t *testing.T) {
	for _, status := range []string{"queued", "running", "blocked", "recovering"} {
		t.Run(status, func(t *testing.T) {
			db := workspaceDeleteDB(t)
			app := models.Application{Name: "任务进行中"}
			workspaceCreate(t, db, &app)
			version := models.Release{ApplicationID: app.ID}
			workspaceCreate(t, db, &version)
			task := models.Deployment{ReleaseID: version.ID, Status: status, IdempotencyKey: status}
			if status == "recovering" {
				task.Status, task.RecoveryStatus = "failed", "running"
			}
			workspaceCreate(t, db, &task)
			response := workspaceDeleteRequest(t, DeleteApplication, fmt.Sprint(app.ID))
			if response.Code != 409 {
				t.Fatal(response.Code, response.Body.String())
			}
			workspaceCount(t, db, &models.Application{}, 1)
			workspaceCount(t, db, &models.Deployment{}, 1)
		})
	}
}

func TestDeleteApplicationRollsBackPartialCascade(t *testing.T) {
	db := workspaceDeleteDB(t)
	app := models.Application{Name: "事务回滚"}
	workspaceCreate(t, db, &app)
	env := models.Environment{ApplicationID: app.ID}
	workspaceCreate(t, db, &env)
	version := models.Release{ApplicationID: app.ID}
	workspaceCreate(t, db, &version)
	task := models.Deployment{EnvironmentID: env.ID, ReleaseID: version.ID, Status: "success", IdempotencyKey: "rollback"}
	workspaceCreate(t, db, &task)
	workspaceCreate(t, db, &models.TaskLog{DeploymentID: task.ID})
	if err := db.Exec("CREATE TRIGGER fail_release_delete BEFORE DELETE ON releases BEGIN SELECT RAISE(ABORT, 'fixture failure'); END").Error; err != nil {
		t.Fatal(err)
	}
	if response := workspaceDeleteRequest(t, DeleteApplication, fmt.Sprint(app.ID)); response.Code != 500 {
		t.Fatal(response.Code, response.Body.String())
	}
	for _, model := range []any{&models.Application{}, &models.Environment{}, &models.Release{}, &models.Deployment{}, &models.TaskLog{}} {
		workspaceCount(t, db, model, 1)
	}
	workspaceCount(t, db, &models.AuditEvent{}, 0)
}

func TestDeleteApplicationWaitsForFinalTaskLog(t *testing.T) {
	db := workspaceDeleteDB(t)
	app := models.Application{Name: "完成边界"}
	workspaceCreate(t, db, &app)
	env := models.Environment{ApplicationID: app.ID}
	workspaceCreate(t, db, &env)
	task := models.Deployment{EnvironmentID: env.ID, Status: "running", IdempotencyKey: "finish-boundary"}
	workspaceCreate(t, db, &task)
	updated, resume, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	if err := db.Callback().Update().After("gorm:commit_or_rollback_transaction").Register("fixture_finish_boundary", func(tx *gorm.DB) {
		values, ok := tx.Statement.Dest.(map[string]any)
		if ok && values["status"] == "failed" {
			close(updated)
			<-resume
		}
	}); err != nil {
		t.Fatal(err)
	}
	go func() { release.RemoteFailure(&task, "failed", "最终日志"); close(finished) }()
	select {
	case <-updated:
	case <-time.After(time.Second):
		close(resume)
		t.Fatal("任务未进入完成阶段")
	}
	deleted := make(chan *httptest.ResponseRecorder, 1)
	go func() { deleted <- workspaceDeleteRequest(t, DeleteApplication, fmt.Sprint(app.ID)) }()
	select {
	case <-deleted:
		close(resume)
		<-finished
		t.Fatal("最终日志尚未保存，应用已被删除")
	case <-time.After(20 * time.Millisecond):
	}
	close(resume)
	<-finished
	select {
	case response := <-deleted:
		if response.Code != 200 {
			t.Fatal(response.Code, response.Body.String())
		}
	case <-time.After(time.Second):
		t.Fatal("任务完成后删除仍未结束")
	}
	workspaceCount(t, db, &models.TaskLog{}, 0)
	workspaceCount(t, db, &models.Deployment{}, 0)
}

func TestDeleteManagedTemplatePreservesApplicationSnapshot(t *testing.T) {
	db := workspaceDeleteDB(t)
	template := models.ApplicationTemplate{Name: "团队模板", Version: "v1", Definition: models.ApplicationDefinition{Compose: "原始模板"}}
	workspaceCreate(t, db, &template)
	app := models.Application{Name: "已有应用", TemplateID: fmt.Sprintf("custom-%d", template.ID), TemplateVersion: "v1", Definition: models.ApplicationDefinition{Compose: "独立快照"}}
	workspaceCreate(t, db, &app)
	if response := workspaceDeleteRequest(t, DeleteManagedTemplate, app.TemplateID); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	workspaceCount(t, db, &models.ApplicationTemplate{}, 0)
	db.First(&app, app.ID)
	if app.Definition.Compose != "独立快照" || app.TemplateID == "" {
		t.Fatal("模板删除修改了已有应用")
	}
	if response := workspaceDeleteRequest(t, DeleteManagedTemplate, "mysql"); response.Code != 403 {
		t.Fatal("内置模板未受保护", response.Body.String())
	}
}

func bundleDeleteFixture(t *testing.T, db *gorm.DB, status string) models.Bundle {
	t.Helper()
	hash := strings.Repeat("b", 64)
	archive := filepath.Join(config.Load().DataDir, "uploads", hash+".tar.gz")
	if err := os.WriteFile(archive, []byte("完整归档"), 0600); err != nil {
		t.Fatal(err)
	}
	item := models.Bundle{Name: "测试发布包.tar.gz", SHA256: hash, Path: archive, Status: status}
	workspaceCreate(t, db, &item)
	dir := filepath.Join(config.Load().DataDir, "releases", fmt.Sprintf("bundle-%d", item.ID))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services: {}"), 0600)
	return item
}

func TestDeleteBundleRemovesOnlyManagedMaterials(t *testing.T) {
	db := workspaceDeleteDB(t)
	item := bundleDeleteFixture(t, db, "ready")
	part := filepath.Join(config.Load().DataDir, "uploads", item.SHA256+".part")
	source := filepath.Join(config.Load().ImportDir, "原始交付文件.tar.gz")
	os.WriteFile(part, []byte("新的未完成上传"), 0600)
	os.WriteFile(source, []byte("服务器原件"), 0600)
	if response := workspaceDeleteRequest(t, DeleteBundle, fmt.Sprint(item.ID)); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	workspaceCount(t, db, &models.Bundle{}, 0)
	for _, path := range []string{item.Path, filepath.Join(config.Load().DataDir, "releases", fmt.Sprintf("bundle-%d", item.ID))} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("材料未被移除", path, err)
		}
	}
	for _, path := range []string{source, part} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal("删除影响了原始文件或新的上传", err)
		}
	}
	workspaceCount(t, db, &models.AuditEvent{}, 1)
}

func TestDeleteBundleProtectsImportsAndReferences(t *testing.T) {
	for _, state := range []string{"queued", "validating", "loading", "unknown", "referenced"} {
		t.Run(state, func(t *testing.T) {
			db := workspaceDeleteDB(t)
			status := state
			if state == "referenced" {
				status = "ready"
			}
			item := bundleDeleteFixture(t, db, status)
			if state == "referenced" {
				workspaceCreate(t, db, &models.Release{ApplicationID: 1, BundleID: item.ID, Source: "bundle"})
			}
			if response := workspaceDeleteRequest(t, DeleteBundle, fmt.Sprint(item.ID)); response.Code != 409 {
				t.Fatal(response.Code, response.Body.String())
			}
			workspaceCount(t, db, &models.Bundle{}, 1)
			if _, err := os.Stat(item.Path); err != nil {
				t.Fatal("受保护材料被移除", err)
			}
		})
	}
}

func TestDeleteBundleRestoresFilesAfterDatabaseFailure(t *testing.T) {
	db := workspaceDeleteDB(t)
	item := bundleDeleteFixture(t, db, "failed")
	if err := db.Exec("CREATE TRIGGER fail_bundle_delete BEFORE DELETE ON bundles BEGIN SELECT RAISE(ABORT, 'fixture failure'); END").Error; err != nil {
		t.Fatal(err)
	}
	if response := workspaceDeleteRequest(t, DeleteBundle, fmt.Sprint(item.ID)); response.Code != 500 {
		t.Fatal(response.Code, response.Body.String())
	}
	workspaceCount(t, db, &models.Bundle{}, 1)
	archive, err := os.ReadFile(item.Path)
	if err != nil || string(archive) != "完整归档" {
		t.Fatal("事务失败后归档未恢复", err)
	}
	if _, err := os.Stat(filepath.Join(config.Load().DataDir, "releases", fmt.Sprintf("bundle-%d", item.ID), "compose.yaml")); err != nil {
		t.Fatal("事务失败后解包材料未恢复", err)
	}
	trash, _ := filepath.Glob(filepath.Join(config.Load().DataDir, ".deleted-bundle-*"))
	if len(trash) != 0 {
		t.Fatal("遗留暂存目录", trash)
	}
}

func TestDeleteBundleRejectsUnmanagedPathsAndInvalidIDs(t *testing.T) {
	db := workspaceDeleteDB(t)
	item := bundleDeleteFixture(t, db, "failed")
	outside := filepath.Join(t.TempDir(), "outside.tar.gz")
	os.WriteFile(outside, []byte("不能删除"), 0600)
	db.Model(&item).Update("path", outside)
	if response := workspaceDeleteRequest(t, DeleteBundle, fmt.Sprint(item.ID)); response.Code != 409 {
		t.Fatal(response.Code, response.Body.String())
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("删除越出了受管目录", err)
	}
	for _, handler := range []gin.HandlerFunc{DeleteApplication, DeleteBundle, DeleteManagedTemplate} {
		for _, id := range []string{"0", "-1", "1 OR 1=1", "4294967296"} {
			if response := workspaceDeleteRequest(t, handler, id); response.Code != 400 {
				t.Fatal("无效编号未被拒绝", id, response.Code)
			}
		}
	}
}
