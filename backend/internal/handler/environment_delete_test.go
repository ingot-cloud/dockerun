package handler

import (
	"dockerun/internal/config"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func environmentDeleteRequest(t *testing.T, applicationID, environmentID string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Params = gin.Params{{Key: "id", Value: applicationID}, {Key: "environment_id", Value: environmentID}}
	ctx.Request = httptest.NewRequest("DELETE", "/applications/"+url.PathEscape(applicationID)+"/environments/"+url.PathEscape(environmentID), nil)
	ctx.Set("user_id", uint(7))
	DeleteEnvironment(ctx)
	return response
}

func TestDeleteEnvironmentTaskProtection(t *testing.T) {
	db := workspaceDeleteDB(t)
	app := models.Application{Name: "环境删除状态"}
	workspaceCreate(t, db, &app)
	for i, state := range []struct {
		status, recovery string
		blocked          bool
	}{
		{"queued", "", true}, {"running", "", true}, {"blocked", "", true},
		{"failed", "running", true}, {"success", "running", true},
		{"success", "", false}, {"failed", "failed", false}, {"needs_attention", "", false},
		{"cancelled", "", false}, {"paused", "", false},
	} {
		t.Run(state.status+"/"+state.recovery, func(t *testing.T) {
			env := models.Environment{ApplicationID: app.ID, Name: fmt.Sprintf("环境-%d", i)}
			workspaceCreate(t, db, &env)
			workspaceCreate(t, db, &models.Deployment{EnvironmentID: env.ID, Status: state.status, RecoveryStatus: state.recovery, IdempotencyKey: fmt.Sprint(i)})
			reasons, err := environmentDeletionReasons(db, []uint{env.ID})
			if err != nil || (reasons[env.ID] != "") != state.blocked {
				t.Fatal("环境删除提示与任务状态不符", reasons, err)
			}
			response := environmentDeleteRequest(t, fmt.Sprint(app.ID), fmt.Sprint(env.ID))
			want := 200
			if state.blocked {
				want = 409
			}
			if response.Code != want {
				t.Fatalf("状态 %d，预期 %d：%s", response.Code, want, response.Body.String())
			}
			var count int64
			db.Model(&models.Environment{}).Where("id = ?", env.ID).Count(&count)
			if (count == 1) != state.blocked {
				t.Fatal("环境删除结果与保护状态不符")
			}
		})
	}
}

func TestDeleteEnvironmentPreservesHistoryVersionsAndRuntime(t *testing.T) {
	db := workspaceDeleteDB(t)
	app := models.Application{Name: "保留应用"}
	workspaceCreate(t, db, &app)
	version := models.Release{ApplicationID: app.ID, Version: "v1"}
	otherVersion := models.Release{ApplicationID: app.ID, Version: "v2"}
	workspaceCreate(t, db, &version)
	workspaceCreate(t, db, &otherVersion)
	env := models.Environment{ApplicationID: app.ID, Name: "生产", CurrentReleaseID: version.ID, AutoDeploy: true}
	other := models.Environment{ApplicationID: app.ID, Name: "测试", CurrentReleaseID: otherVersion.ID}
	workspaceCreate(t, db, &env)
	workspaceCreate(t, db, &other)
	for _, item := range []models.Environment{env, other} {
		workspaceCreate(t, db, &models.TargetState{EnvironmentID: item.ID, TargetID: 1, CurrentReleaseID: item.CurrentReleaseID})
	}
	snapshot, _ := json.Marshal(release.Snapshot{Release: version, Environment: env})
	task := models.Deployment{EnvironmentID: env.ID, ReleaseID: version.ID, Status: "success", Snapshot: string(snapshot), IdempotencyKey: "deleted-environment-history"}
	queued := models.Deployment{EnvironmentID: other.ID, ReleaseID: otherVersion.ID, Status: "queued", IdempotencyKey: "other-environment-queued"}
	workspaceCreate(t, db, &task)
	workspaceCreate(t, db, &queued)
	for _, item := range []models.Deployment{task, queued} {
		workspaceCreate(t, db, &models.TaskLog{DeploymentID: item.ID, Message: "保留的日志"})
	}
	file := filepath.Join(config.Load().DataDir, "jobs", fmt.Sprint(task.ID), "service.conf")
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("保留运行配置"), 0600); err != nil {
		t.Fatal(err)
	}
	response := environmentDeleteRequest(t, fmt.Sprint(app.ID), fmt.Sprint(env.ID))
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	for _, model := range []any{&models.Application{}, &models.Environment{}, &models.TargetState{}, &models.AuditEvent{}} {
		workspaceCount(t, db, model, 1)
	}
	for _, model := range []any{&models.Release{}, &models.Deployment{}, &models.TaskLog{}} {
		workspaceCount(t, db, model, 2)
	}
	var remaining models.Environment
	db.First(&remaining)
	if remaining.ID != other.ID || remaining.CurrentReleaseID != otherVersion.ID {
		t.Fatal("删除影响了其他环境")
	}
	var history models.Deployment
	db.First(&history, task.ID)
	if history.Snapshot != string(snapshot) {
		t.Fatal("历史配置快照被修改")
	}
	if content, err := os.ReadFile(file); err != nil || string(content) != "保留运行配置" {
		t.Fatal("运行配置文件被删除", err)
	}
	var audit models.AuditEvent
	db.First(&audit)
	if audit.Action != "environment.delete" || audit.UserID != 7 || audit.Resource != fmt.Sprintf("%d/%d", app.ID, env.ID) {
		t.Fatal("删除审计错误", audit)
	}
	reasons, err := releaseDeletionReasons(db, []uint{version.ID, otherVersion.ID})
	if err != nil || reasons[version.ID] != "" || reasons[otherVersion.ID] == "" {
		t.Fatal("环境删除后的版本引用状态错误", reasons, err)
	}
	if _, err := release.Enqueue(env, version, 7, "stale-environment", false, 0); err == nil {
		t.Fatal("已删除环境仍可被旧请求排队")
	}
	response = registryRequest(t, SaveEnvironment, app.ID, map[string]any{"id": env.ID, "name": "旧配置", "target_id": 1})
	if response.Code != 404 {
		t.Fatal("旧配置重新创建了已删除环境", response.Code, response.Body.String())
	}
}

func TestDeleteEnvironmentValidatesOwnershipAndIDs(t *testing.T) {
	db := workspaceDeleteDB(t)
	app := models.Application{Name: "环境所属应用"}
	workspaceCreate(t, db, &app)
	env := models.Environment{ApplicationID: app.ID, Name: "生产"}
	workspaceCreate(t, db, &env)
	for _, pair := range [][2]string{{"0", "1"}, {"bad", "1"}, {"1", "0"}, {"1", "-1"}, {"1", "bad"}, {"1", "4294967296"}} {
		if response := environmentDeleteRequest(t, pair[0], pair[1]); response.Code != 400 {
			t.Fatal("无效编号未被拒绝", pair, response.Code)
		}
	}
	for _, pair := range [][2]string{{fmt.Sprint(app.ID + 1), fmt.Sprint(env.ID)}, {fmt.Sprint(app.ID), "99999"}} {
		if response := environmentDeleteRequest(t, pair[0], pair[1]); response.Code != 404 {
			t.Fatal("跨应用或不存在的环境未被拒绝", pair, response.Code)
		}
	}
	workspaceCount(t, db, &models.Environment{}, 1)
	workspaceCount(t, db, &models.AuditEvent{}, 0)
}

func TestApplicationDetailIncludesEnvironmentDeletionReasons(t *testing.T) {
	db := workspaceDeleteDB(t)
	app := models.Application{Name: "环境任务提示"}
	workspaceCreate(t, db, &app)
	environments := []models.Environment{{ApplicationID: app.ID, Name: "排队环境"}, {ApplicationID: app.ID, Name: "可删环境"}}
	workspaceCreate(t, db, &environments)
	workspaceCreate(t, db, &models.Deployment{EnvironmentID: environments[0].ID, Status: "blocked", IdempotencyKey: "summary"})
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest("GET", "/applications/1", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(app.ID)}}
	ApplicationDetail(ctx)
	var detail struct{ Environments []environmentSummary }
	if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil || response.Code != 200 || len(detail.Environments) != 2 {
		t.Fatal("环境列表错误", response.Body.String(), err)
	}
	for _, env := range detail.Environments {
		if (env.DeletionBlockedReason != "") != (env.ID == environments[0].ID) {
			t.Fatal("环境任务保护提示错误", env.Name, env.DeletionBlockedReason)
		}
	}
}

func TestDeleteEnvironmentRollsBackOnFailure(t *testing.T) {
	for _, statement := range []string{
		`CREATE TRIGGER reject_environment_delete BEFORE DELETE ON environments BEGIN SELECT RAISE(ABORT, 'fixture failure'); END`,
		`CREATE TRIGGER reject_environment_audit BEFORE INSERT ON audit_events BEGIN SELECT RAISE(ABORT, 'fixture failure'); END`,
	} {
		t.Run(statement, func(t *testing.T) {
			db := workspaceDeleteDB(t)
			env := models.Environment{ApplicationID: 7, Name: "回滚删除"}
			workspaceCreate(t, db, &env)
			workspaceCreate(t, db, &models.TargetState{EnvironmentID: env.ID, TargetID: 1, CurrentReleaseID: 3})
			if err := db.Exec(statement).Error; err != nil {
				t.Fatal(err)
			}
			if response := environmentDeleteRequest(t, "7", fmt.Sprint(env.ID)); response.Code != 500 {
				t.Fatal("失败未返回错误", response.Code)
			}
			workspaceCount(t, db, &models.Environment{}, 1)
			workspaceCount(t, db, &models.TargetState{}, 1)
			workspaceCount(t, db, &models.AuditEvent{}, 0)
		})
	}
}

func TestDeleteEnvironmentWaitsForConcurrentEnqueue(t *testing.T) {
	db := workspaceDeleteDB(t)
	if err := db.AutoMigrate(&models.Target{}); err != nil {
		t.Fatal(err)
	}
	workspaceCreate(t, db, &models.Target{ID: 1, Name: "本机", Enabled: true})
	app := models.Application{Name: "并发环境删除"}
	workspaceCreate(t, db, &app)
	env := models.Environment{ApplicationID: app.ID, TargetID: 1, Name: "测试"}
	workspaceCreate(t, db, &env)
	version := models.Release{ApplicationID: app.ID, Version: "v1"}
	workspaceCreate(t, db, &version)
	created, resume := make(chan struct{}), make(chan struct{})
	if err := db.Callback().Create().After("gorm:create").Register("fixture_enqueue_environment", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "deployments" {
			close(created)
			<-resume
		}
	}); err != nil {
		t.Fatal(err)
	}
	queued := make(chan error, 1)
	go func() {
		_, err := release.Enqueue(env, version, 7, "concurrent-environment-delete", false, 0)
		queued <- err
	}()
	select {
	case <-created:
	case <-time.After(time.Second):
		close(resume)
		t.Fatal("任务未进入排队事务")
	}
	deleted := make(chan *httptest.ResponseRecorder, 1)
	go func() { deleted <- environmentDeleteRequest(t, fmt.Sprint(app.ID), fmt.Sprint(env.ID)) }()
	select {
	case <-deleted:
		close(resume)
		<-queued
		t.Fatal("排队期间删除检查未等待")
	case <-time.After(20 * time.Millisecond):
	}
	close(resume)
	if err := <-queued; err != nil {
		t.Fatal(err)
	}
	select {
	case response := <-deleted:
		if response.Code != 409 {
			t.Fatal("排队环境未受保护", response.Code, response.Body.String())
		}
	case <-time.After(time.Second):
		t.Fatal("删除检查未完成")
	}
	workspaceCount(t, db, &models.Environment{}, 1)
	workspaceCount(t, db, &models.Deployment{}, 1)
}
