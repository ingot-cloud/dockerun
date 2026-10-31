package handler

import (
	"dockerun/internal/config"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestDeleteReleaseProtectsCurrentAndActiveReferences(t *testing.T) {
	for _, source := range []string{"environment", "target", "queued", "running", "blocked", "previous_queued", "previous_running", "previous_blocked", "recovery", "previous_recovery"} {
		t.Run(source, func(t *testing.T) {
			db := workspaceDeleteDB(t)
			version := models.Release{Version: "受保护版本"}
			workspaceCreate(t, db, &version)
			switch source {
			case "environment":
				workspaceCreate(t, db, &models.Environment{Name: "生产", CurrentReleaseID: version.ID})
			case "target":
				workspaceCreate(t, db, &models.TargetState{EnvironmentID: 99, TargetID: 7, CurrentReleaseID: version.ID})
			default:
				task := models.Deployment{Status: source, ReleaseID: version.ID, IdempotencyKey: source}
				switch source {
				case "previous_queued", "previous_running", "previous_blocked":
					task.Status, task.ReleaseID, task.PreviousReleaseID = source[len("previous_"):], 0, version.ID
				case "recovery", "previous_recovery":
					task.Status, task.RecoveryStatus = "failed", "running"
					if source == "previous_recovery" {
						task.ReleaseID, task.PreviousReleaseID = 0, version.ID
					}
				}
				workspaceCreate(t, db, &task)
			}
			reasons, err := releaseDeletionReasons(db, []uint{version.ID})
			if err != nil || reasons[version.ID] == "" {
				t.Fatal("列表缺少版本保护原因", err)
			}
			response := workspaceDeleteRequest(t, DeleteRelease, fmt.Sprint(version.ID))
			if response.Code != 409 {
				t.Fatal("受保护版本未被拒绝", response.Code, response.Body.String())
			}
			workspaceCount(t, db, &models.Release{}, 1)
			workspaceCount(t, db, &models.AuditEvent{}, 0)
		})
	}
}

func TestDeleteReleasePreservesHistoryAndResources(t *testing.T) {
	db := workspaceDeleteDB(t)
	app := models.Application{Name: "保留应用"}
	workspaceCreate(t, db, &app)
	env := models.Environment{ApplicationID: app.ID, Name: "测试"}
	workspaceCreate(t, db, &env)
	bundle := models.Bundle{Name: "原始发布包", SHA256: "release-delete", Status: "ready"}
	workspaceCreate(t, db, &bundle)
	version := models.Release{ApplicationID: app.ID, Version: "v1", BundleID: bundle.ID, Retained: true}
	other := models.Release{ApplicationID: app.ID, Version: "v2"}
	workspaceCreate(t, db, &version)
	workspaceCreate(t, db, &other)
	snapshot, _ := json.Marshal(release.Snapshot{Release: version, Environment: env})
	for _, status := range []string{"success", "failed", "needs_attention", "cancelled", "paused"} {
		task := models.Deployment{EnvironmentID: env.ID, ReleaseID: version.ID, PreviousReleaseID: version.ID, Status: status, Snapshot: string(snapshot), IdempotencyKey: status}
		workspaceCreate(t, db, &task)
		workspaceCreate(t, db, &models.TaskLog{DeploymentID: task.ID, Message: "历史日志"})
	}
	file := filepath.Join(config.Load().DataDir, "jobs", "1", "runtime.conf")
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("保留运行配置"), 0600); err != nil {
		t.Fatal(err)
	}
	response := workspaceDeleteRequest(t, DeleteRelease, fmt.Sprint(version.ID))
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	for _, model := range []any{&models.Application{}, &models.Environment{}, &models.Bundle{}, &models.Release{}, &models.AuditEvent{}} {
		workspaceCount(t, db, model, 1)
	}
	workspaceCount(t, db, &models.Deployment{}, 5)
	workspaceCount(t, db, &models.TaskLog{}, 5)
	var history models.Deployment
	db.First(&history)
	if history.Snapshot != string(snapshot) {
		t.Fatal("历史执行快照被修改")
	}
	if content, err := os.ReadFile(file); err != nil || string(content) != "保留运行配置" {
		t.Fatal("删除版本移除了实际配置文件", err)
	}
	var audit models.AuditEvent
	db.First(&audit)
	if audit.Action != "release.delete" || audit.UserID != 7 || audit.Resource != fmt.Sprint(version.ID) {
		t.Fatal("删除审计错误", audit)
	}
	if _, err := release.Enqueue(env, version, 7, "stale-version", false, 0); err == nil {
		t.Fatal("已删除版本仍可被旧请求部署")
	}
	if response := workspaceDeleteRequest(t, DeleteRelease, fmt.Sprint(version.ID)); response.Code != 404 {
		t.Fatal("删除不存在的版本未返回 404")
	}
	for _, id := range []string{"0", "-1", "bad", "4294967296"} {
		if response := workspaceDeleteRequest(t, DeleteRelease, id); response.Code != 400 {
			t.Fatal("无效编号未被拒绝", id, response.Code)
		}
	}
}

func TestApplicationDetailIncludesReleaseDeletionReasons(t *testing.T) {
	db := workspaceDeleteDB(t)
	app := models.Application{Name: "版本保护说明"}
	workspaceCreate(t, db, &app)
	versions := []models.Release{{ApplicationID: app.ID, Version: "环境使用"}, {ApplicationID: app.ID, Version: "主机使用"}, {ApplicationID: app.ID, Version: "等待部署"}, {ApplicationID: app.ID, Version: "可以删除"}}
	workspaceCreate(t, db, &versions)
	env := models.Environment{ApplicationID: app.ID, Name: "生产", CurrentReleaseID: versions[0].ID}
	workspaceCreate(t, db, &env)
	workspaceCreate(t, db, &models.TargetState{EnvironmentID: env.ID, TargetID: 3, CurrentReleaseID: versions[1].ID})
	workspaceCreate(t, db, &models.Deployment{EnvironmentID: env.ID, PreviousReleaseID: versions[2].ID, Status: "blocked", IdempotencyKey: "detail-protection"})
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest("GET", "/applications/1", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(app.ID)}}
	ApplicationDetail(ctx)
	var detail struct{ Releases []releaseSummary }
	if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil || response.Code != 200 || len(detail.Releases) != 4 {
		t.Fatal("应用详情版本列表错误", response.Body.String(), err)
	}
	for _, version := range detail.Releases {
		if (version.DeletionBlockedReason == "") != (version.ID == versions[3].ID) {
			t.Fatal("版本删除提示与实际引用不一致", version.Version, version.DeletionBlockedReason)
		}
	}
}

func TestDeleteReleaseRollsBackOnFailure(t *testing.T) {
	for _, statement := range []string{
		`CREATE TRIGGER reject_release_delete BEFORE DELETE ON releases BEGIN SELECT RAISE(ABORT, 'fixture failure'); END`,
		`CREATE TRIGGER reject_release_audit BEFORE INSERT ON audit_events BEGIN SELECT RAISE(ABORT, 'fixture failure'); END`,
	} {
		t.Run(statement, func(t *testing.T) {
			db := workspaceDeleteDB(t)
			version := models.Release{Version: "v1"}
			workspaceCreate(t, db, &version)
			if err := db.Exec(statement).Error; err != nil {
				t.Fatal(err)
			}
			if response := workspaceDeleteRequest(t, DeleteRelease, fmt.Sprint(version.ID)); response.Code != 500 {
				t.Fatal("删除失败未返回错误", response.Code)
			}
			workspaceCount(t, db, &models.Release{}, 1)
			workspaceCount(t, db, &models.AuditEvent{}, 0)
		})
	}
}

func TestDeleteReleaseWaitsForConcurrentEnqueue(t *testing.T) {
	db := workspaceDeleteDB(t)
	if err := db.AutoMigrate(&models.Target{}); err != nil {
		t.Fatal(err)
	}
	workspaceCreate(t, db, &models.Target{ID: 1, Name: "本机", Enabled: true})
	app := models.Application{Name: "排队并发"}
	workspaceCreate(t, db, &app)
	env := models.Environment{ApplicationID: app.ID, TargetID: 1, Name: "测试"}
	workspaceCreate(t, db, &env)
	version := models.Release{ApplicationID: app.ID, Version: "v1"}
	workspaceCreate(t, db, &version)
	created, resume := make(chan struct{}), make(chan struct{})
	if err := db.Callback().Create().After("gorm:create").Register("fixture_enqueue_release", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "deployments" {
			close(created)
			<-resume
		}
	}); err != nil {
		t.Fatal(err)
	}
	queued := make(chan error, 1)
	go func() {
		_, err := release.Enqueue(env, version, 7, "concurrent-release-delete", false, 0)
		queued <- err
	}()
	select {
	case <-created:
	case <-time.After(time.Second):
		close(resume)
		t.Fatal("部署未进入排队事务")
	}
	deleted := make(chan *httptest.ResponseRecorder, 1)
	go func() { deleted <- workspaceDeleteRequest(t, DeleteRelease, fmt.Sprint(version.ID)) }()
	select {
	case <-deleted:
		close(resume)
		<-queued
		t.Fatal("部署提交期间完成了版本删除")
	case <-time.After(20 * time.Millisecond):
	}
	close(resume)
	if err := <-queued; err != nil {
		t.Fatal("任务排队失败", err)
	}
	select {
	case response := <-deleted:
		if response.Code != 409 {
			t.Fatal("排队后的版本未受保护", response.Code, response.Body.String())
		}
	case <-time.After(time.Second):
		t.Fatal("删除检查未结束")
	}
	workspaceCount(t, db, &models.Release{}, 1)
	workspaceCount(t, db, &models.Deployment{}, 1)
}

func TestDeletedReleaseDoesNotConsumeRetentionSlot(t *testing.T) {
	db := workspaceDeleteDB(t)
	app := models.Application{Name: "保留版本名额"}
	workspaceCreate(t, db, &app)
	env := models.Environment{ApplicationID: app.ID, Name: "测试", TargetID: 1}
	workspaceCreate(t, db, &env)
	versions := make([]models.Release, 5)
	for i := range versions {
		versions[i] = models.Release{ApplicationID: app.ID, Version: fmt.Sprintf("v%d", i+1), Retained: true}
		workspaceCreate(t, db, &versions[i])
		if i < 4 {
			workspaceCreate(t, db, &models.Deployment{EnvironmentID: env.ID, TargetID: 1, ReleaseID: versions[i].ID, Status: "success", IdempotencyKey: fmt.Sprintf("retention-%d", i)})
		}
	}
	if response := workspaceDeleteRequest(t, DeleteRelease, fmt.Sprint(versions[3].ID)); response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	if reason := release.RetentionProtection(versions[0]); reason == "" {
		t.Fatal("已删除版本占用了最近成功版本的保护名额")
	}
	task := models.Deployment{EnvironmentID: env.ID, TargetID: 1, ReleaseID: versions[4].ID, Status: "running", IdempotencyKey: "retention-new"}
	workspaceCreate(t, db, &task)
	release.RemoteSuccess(&task, "[]", "healthy")
	for i, want := range map[int]bool{0: false, 1: true, 2: true, 4: true} {
		var version models.Release
		if err := db.First(&version, versions[i].ID).Error; err != nil || version.Retained != want {
			t.Fatalf("%s 保留状态 %v，预期 %v：%v", versions[i].Version, version.Retained, want, err)
		}
	}
}
