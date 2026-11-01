package handler

import (
	"bytes"
	"context"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestApplicationScriptsAndEnvironmentDefaults(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Application{}, &models.Environment{}, &models.Release{}, &models.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	app := models.Application{Name: "scripted"}
	db.Create(&app)
	external := models.Registry{Name: "external", URL: "https://external.test", Kind: "external", IsDefault: true}
	builtin := models.Registry{Name: "builtin", URL: "https://builtin.test", Kind: "builtin"}
	db.Create(&external)
	db.Create(&builtin)
	scripts := models.DeploymentScripts{Before: "echo before", Deploy: "echo deploy", After: "echo after"}
	response := registryRequest(t, SaveApplicationScripts, app.ID, map[string]any{"scripts": scripts})
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	db.First(&app, app.ID)
	if app.Scripts != scripts {
		t.Fatal("application scripts were not stored")
	}
	response = registryRequest(t, SaveApplicationScripts, app.ID, map[string]any{"scripts": models.DeploymentScripts{Deploy: "if then"}})
	if response.Code != http.StatusBadRequest {
		t.Fatal("invalid shell syntax was accepted")
	}
	env := models.Environment{Name: "test", ScriptMode: "override", Scripts: models.DeploymentScripts{After: "echo environment"}}
	response = registryRequest(t, SaveEnvironment, app.ID, env)
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.RegistryID != builtin.ID || env.Scripts.After != "echo environment" {
		t.Fatal("environment defaults or scripts were lost")
	}
	env.ScriptMode = "disabled"
	response = registryRequest(t, SaveEnvironment, app.ID, env)
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	db.First(&env, env.ID)
	if env.ScriptMode != "disabled" || env.Scripts != (models.DeploymentScripts{}) {
		t.Fatal("clearing environment scripts failed")
	}
	// A caller cannot fabricate a current version to skip the first deployment.
	env.AutoDeploy = true
	env.CurrentReleaseID = 999
	env.Repository = "team/app"
	response = registryRequest(t, SaveEnvironment, app.ID, env)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "首次手动发布") {
		t.Fatal(response.Body.String())
	}
	base := models.Release{ApplicationID: app.ID, Source: "registry", RegistryID: builtin.ID, Images: `[{"reference":"builtin.test/team/app:v1"}]`}
	db.Create(&base)
	db.Model(&env).Update("current_release_id", base.ID)
	response = registryRequest(t, SaveEnvironment, app.ID, env)
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	env.RegistryID = external.ID
	response = registryRequest(t, SaveEnvironment, app.ID, env)
	if response.Code != http.StatusBadRequest {
		t.Fatal("unrelated registry accepted as an automatic source")
	}
	env.RegistryID = builtin.ID
	env.Repository = "team/another"
	response = registryRequest(t, SaveEnvironment, app.ID, env)
	if response.Code != http.StatusBadRequest {
		t.Fatal("unrelated repository accepted as an automatic source")
	}
}

func TestOldAgentCannotSkipDeploymentScripts(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Target{}, &models.Deployment{}, &models.TaskLog{}, &models.Environment{}); err != nil {
		t.Fatal(err)
	}
	target := models.Target{ID: 2, Name: "old-agent", Enabled: true}
	db.Create(&target)
	snapshot, _ := json.Marshal(release.Snapshot{Environment: models.Environment{ID: 1, TargetID: target.ID}, Scripts: models.DeploymentScripts{Before: "echo must-run"}})
	task := models.Deployment{Status: "queued", TargetID: target.ID, Snapshot: string(snapshot), IdempotencyKey: "old-agent"}
	db.Create(&task)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set("target", target)
	requestContext, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	ctx.Request = httptest.NewRequest("GET", "/agent/poll?platform=linux/amd64", bytes.NewReader(nil)).WithContext(requestContext)
	AgentPoll(ctx)
	db.First(&task, task.ID)
	if task.Status != "failed" || !strings.Contains(task.Result, "请更新 Agent") {
		t.Fatal("old Agent was allowed to silently skip deployment scripts")
	}
}

func TestOldAgentCannotIgnoreCommandExecution(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Target{}, &models.Deployment{}, &models.TaskLog{}, &models.Environment{}); err != nil {
		t.Fatal(err)
	}
	target := models.Target{ID: 2, Name: "script-only-agent", Enabled: true}
	db.Create(&target)
	snapshot, _ := json.Marshal(release.Snapshot{Release: models.Release{Execution: models.DeploymentExecution{Mode: "command", Command: "echo required"}}, Environment: models.Environment{ID: 1, TargetID: target.ID}})
	task := models.Deployment{Status: "queued", TargetID: target.ID, Snapshot: string(snapshot), IdempotencyKey: "old-execution-agent"}
	db.Create(&task)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set("target", target)
	requestContext, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	ctx.Request = httptest.NewRequest("GET", "/agent/poll?platform=linux/amd64&scripts=1", nil).WithContext(requestContext)
	AgentPoll(ctx)
	db.First(&task, task.ID)
	if task.Status != "failed" || !strings.Contains(task.Result, "请更新 Agent") {
		t.Fatal("old Agent could ignore command execution", task.Result)
	}
}

func TestOldAgentCannotIgnoreCommandTriggerImage(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Target{}, &models.Deployment{}, &models.TaskLog{}, &models.Environment{}); err != nil {
		t.Fatal(err)
	}
	target := models.Target{ID: 2, Name: "agent-without-trigger-images", Enabled: true}
	db.Create(&target)
	snapshot, _ := json.Marshal(release.Snapshot{Release: models.Release{Source: "registry", Execution: models.DeploymentExecution{Mode: "command", Command: `docker run "$DOCKERUN_IMAGE"`}}, Environment: models.Environment{ID: 1, TargetID: target.ID}})
	task := models.Deployment{Status: "queued", TargetID: target.ID, Snapshot: string(snapshot), IdempotencyKey: "old-command-image-agent"}
	db.Create(&task)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set("target", target)
	requestContext, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	ctx.Request = httptest.NewRequest("GET", "/agent/poll?platform=linux/amd64&scripts=1&execution=1", nil).WithContext(requestContext)
	AgentPoll(ctx)
	db.First(&task, task.ID)
	if task.Status != "failed" || !strings.Contains(task.Result, "触发镜像") {
		t.Fatal("old Agent could ignore the pinned command image", task.Result)
	}
}

func TestOldAgentCannotSkipAssociatedContainerReplacement(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Target{}, &models.Deployment{}, &models.TaskLog{}, &models.Environment{}); err != nil {
		t.Fatal(err)
	}
	target := models.Target{ID: 2, Name: "agent-without-container-policy", Enabled: true}
	db.Create(&target)
	snapshot, _ := json.Marshal(release.Snapshot{
		Release:     models.Release{Source: "registry", Execution: models.DeploymentExecution{Mode: "command", Command: `docker run --name test-nginx "$DOCKERUN_IMAGE"`}},
		Environment: models.Environment{ID: 1, TargetID: target.ID, ContainerNames: []string{"test-nginx"}, ReplaceContainers: true},
	})
	task := models.Deployment{Status: "queued", TargetID: target.ID, Snapshot: string(snapshot), IdempotencyKey: "old-container-policy-agent"}
	db.Create(&task)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set("target", target)
	requestContext, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	ctx.Request = httptest.NewRequest("GET", "/agent/poll?platform=linux/amd64&scripts=1&execution=1&command_images=1", nil).WithContext(requestContext)
	AgentPoll(ctx)
	db.First(&task, task.ID)
	if task.Status != "failed" || !strings.Contains(task.Result, "关联容器") || !strings.Contains(task.Result, "请更新 Agent") {
		t.Fatal("old Agent could skip container replacement", task.Result)
	}
}
