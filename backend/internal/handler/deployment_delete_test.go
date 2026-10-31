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

func TestDeleteDeploymentStatuses(t *testing.T) {
	db := workspaceDeleteDB(t)
	for _, status := range []string{"queued", "running", "blocked", "unknown", "", "success", "failed", "needs_attention", "cancelled", "paused"} {
		for _, recovery := range []string{"", "running", "success"} {
			t.Run(status+"/"+recovery, func(t *testing.T) {
				task := models.Deployment{Status: status, RecoveryStatus: recovery, IdempotencyKey: status + "/" + recovery}
				workspaceCreate(t, db, &task)
				log := models.TaskLog{DeploymentID: task.ID, Message: "保留或删除的日志"}
				workspaceCreate(t, db, &log)
				response := workspaceDeleteRequest(t, DeleteDeployment, fmt.Sprint(task.ID))
				terminal := status == "success" || status == "failed" || status == "needs_attention" || status == "cancelled" || status == "paused"
				allowed := terminal && recovery != "running"
				want := 409
				if allowed {
					want = 200
				}
				if response.Code != want {
					t.Fatalf("状态 %d，预期 %d：%s", response.Code, want, response.Body.String())
				}
				var saved models.Deployment
				if err := db.First(&saved, task.ID).Error; err != nil || (saved.HistoryDeletedAt != nil) != allowed {
					t.Fatalf("记录删除状态错误：%+v，%v", saved, err)
				}
				var logs int64
				db.Model(&models.TaskLog{}).Where("deployment_id = ?", task.ID).Count(&logs)
				if (logs == 0) != allowed {
					t.Fatal("日志未随删除状态保持一致")
				}
			})
		}
	}
	for _, id := range []string{"0", "-1", "bad", "4294967296"} {
		if response := workspaceDeleteRequest(t, DeleteDeployment, id); response.Code != 400 {
			t.Fatal("无效编号未被拒绝", id, response.Code)
		}
	}
	if response := workspaceDeleteRequest(t, DeleteDeployment, "99999"); response.Code != 404 {
		t.Fatal("不存在的记录未返回 404")
	}
}

func TestDeleteDeploymentPreservesRuntimeAndHidesHistory(t *testing.T) {
	db := workspaceDeleteDB(t)
	if err := db.AutoMigrate(&models.Target{}, &models.User{}, &models.DeployScript{}, &models.ImageEvent{}); err != nil {
		t.Fatal(err)
	}
	app := models.Application{Name: "保留的应用"}
	workspaceCreate(t, db, &app)
	version := models.Release{ApplicationID: app.ID, Version: "v1"}
	workspaceCreate(t, db, &version)
	env := models.Environment{ApplicationID: app.ID, Name: "生产", CurrentReleaseID: version.ID}
	workspaceCreate(t, db, &env)
	snapshot, _ := json.Marshal(release.Snapshot{Release: version, Environment: models.Environment{Parameters: models.StringMap{"PORT": "8080"}}, Scripts: models.DeploymentScripts{After: "echo ready"}})
	task := models.Deployment{EnvironmentID: env.ID, TargetID: 1, ReleaseID: version.ID, Status: "success", Batch: "same-batch", Snapshot: string(snapshot), IdempotencyKey: "deleted-history"}
	other := models.Deployment{EnvironmentID: env.ID, Status: "failed", Batch: "same-batch", IdempotencyKey: "remaining-history"}
	workspaceCreate(t, db, &task)
	workspaceCreate(t, db, &other)
	workspaceCreate(t, db, &models.TaskLog{DeploymentID: task.ID, Message: "删除的日志"})
	workspaceCreate(t, db, &models.TaskLog{DeploymentID: other.ID, Message: "保留的日志"})
	workspaceCreate(t, db, &models.TargetState{EnvironmentID: env.ID, TargetID: 1, CurrentReleaseID: version.ID})
	file := filepath.Join(config.Load().DataDir, "jobs", fmt.Sprint(task.ID), "service.conf")
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("运行配置"), 0600); err != nil {
		t.Fatal(err)
	}
	response := workspaceDeleteRequest(t, DeleteDeployment, fmt.Sprint(task.ID))
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	for _, model := range []any{&models.Application{}, &models.Environment{}, &models.Release{}, &models.TargetState{}, &models.TaskLog{}, &models.AuditEvent{}} {
		workspaceCount(t, db, model, 1)
	}
	var audit models.AuditEvent
	db.First(&audit)
	if audit.Action != "deployment.delete" || audit.Resource != fmt.Sprint(task.ID) || audit.UserID != 7 {
		t.Fatal("缺少正确的删除审计", audit)
	}
	if content, err := os.ReadFile(file); err != nil || string(content) != "运行配置" {
		t.Fatal("运行配置文件被删除", err)
	}
	if err := restoreParameters(&env, version, true); err != nil || env.Parameters["PORT"] != "8080" || env.Scripts.After != "echo ready" {
		t.Fatal("删除历史后不能再恢复配置", err)
	}
	if _, err := release.Enqueue(env, version, 7, "deleted-history", false, 0); err == nil {
		t.Fatal("重复请求返回了已经删除的历史记录")
	}
	var previous models.Deployment
	if err := db.Where("environment_id = ? AND target_id = ? AND release_id = ? AND status = ?", env.ID, 1, version.ID, "success").Order("id DESC").First(&previous).Error; err != nil || previous.Snapshot != string(snapshot) {
		t.Fatal("运行检查及回滚快照丢失", err)
	}
	request := func(handler gin.HandlerFunc, path string, id uint) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(response)
		ctx.Request = httptest.NewRequest("GET", path, nil)
		ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
		handler(ctx)
		ctx.Writer.WriteHeaderNow()
		return response
	}
	for _, handler := range []gin.HandlerFunc{DeploymentDetail, DeploymentEvents} {
		if response := request(handler, "/deployments/1", task.ID); response.Code != 404 {
			t.Fatal("已删除历史仍可访问", response.Code, response.Body.String())
		}
	}
	response = request(ListDeployments, fmt.Sprintf("/deployments?environment_id=%d", env.ID), 0)
	var rows []deploymentSummary
	if err := json.Unmarshal(response.Body.Bytes(), &rows); err != nil || len(rows) != 1 || rows[0].ID != other.ID {
		t.Fatal("部署列表未隐藏删除的记录", response.Body.String(), err)
	}
	response = request(DeploymentDetail, "/deployments/2", other.ID)
	var detail struct{ Nodes []deploymentSummary }
	if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil || len(detail.Nodes) != 1 || detail.Nodes[0].ID != other.ID {
		t.Fatal("批次详情仍包含已删除的记录", response.Body.String(), err)
	}
	response = request(Overview, "/overview", 0)
	var overview struct{ Counts map[string]int64 }
	if err := json.Unmarshal(response.Body.Bytes(), &overview); err != nil || overview.Counts["deployments"] != 1 {
		t.Fatal("概览未排除已删除的记录", response.Body.String(), err)
	}
	if response := workspaceDeleteRequest(t, DeleteDeployment, fmt.Sprint(task.ID)); response.Code != 404 {
		t.Fatal("已删除的记录仍可重复操作")
	}
}

func TestDeleteDeploymentRollsBackOnFailure(t *testing.T) {
	for _, statement := range []string{
		`CREATE TRIGGER reject_log_delete BEFORE DELETE ON task_logs BEGIN SELECT RAISE(ABORT, 'fixture failure'); END`,
		`CREATE TRIGGER reject_delete_audit BEFORE INSERT ON audit_events BEGIN SELECT RAISE(ABORT, 'fixture failure'); END`,
	} {
		t.Run(statement, func(t *testing.T) {
			db := workspaceDeleteDB(t)
			task := models.Deployment{Status: "failed", IdempotencyKey: "rollback-delete"}
			workspaceCreate(t, db, &task)
			workspaceCreate(t, db, &models.TaskLog{DeploymentID: task.ID, Message: "日志"})
			if err := db.Exec(statement).Error; err != nil {
				t.Fatal(err)
			}
			if response := workspaceDeleteRequest(t, DeleteDeployment, fmt.Sprint(task.ID)); response.Code != 500 {
				t.Fatal("删除失败未返回错误", response.Code)
			}
			db.First(&task, task.ID)
			if task.HistoryDeletedAt != nil {
				t.Fatal("失败后记录未恢复")
			}
			workspaceCount(t, db, &models.TaskLog{}, 1)
			workspaceCount(t, db, &models.AuditEvent{}, 0)
		})
	}
}

func TestDeleteDeploymentWaitsForFinalLog(t *testing.T) {
	db := workspaceDeleteDB(t)
	task := models.Deployment{Status: "running", IdempotencyKey: "final-log"}
	workspaceCreate(t, db, &task)
	updated, resume, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	if err := db.Callback().Update().After("gorm:commit_or_rollback_transaction").Register("fixture_deployment_finish", func(tx *gorm.DB) {
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
	go func() { deleted <- workspaceDeleteRequest(t, DeleteDeployment, fmt.Sprint(task.ID)) }()
	select {
	case <-deleted:
		close(resume)
		<-finished
		t.Fatal("最终日志保存前已删除")
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
		t.Fatal("任务完成后删除未结束")
	}
	workspaceCount(t, db, &models.TaskLog{}, 0)
	if err := db.Callback().Update().Remove("fixture_deployment_finish"); err != nil {
		t.Fatal(err)
	}
	// 晚到的完成回调不能重新生成已清理的日志。
	release.RemoteFailure(&task, "failed", "重复的完成回调")
	workspaceCount(t, db, &models.TaskLog{}, 0)
}
