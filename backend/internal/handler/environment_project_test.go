package handler

import (
	"bytes"
	"context"
	"dockerun/internal/config"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestComposeProjectNameBeforeDeploymentAndSnapshot(t *testing.T) {
	db := registryTestDB(t)
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Application{}, &models.Environment{}, &models.Release{}, &models.Deployment{}, &models.TargetState{}, &models.Target{}, &models.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	app := models.Application{Name: "Redis"}
	db.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "测试", TargetID: 1, Parameters: models.StringMap{"COMPOSE_PROJECT_NAME": "ignored-variable"}}
	db.Create(&env)
	db.Create(&models.Target{ID: 1, Name: "本机", Enabled: true})
	compose := "name: ignored-yaml-name\nservices:\n  redis:\n    image: redis:8.2.0\n    environment:\n      PROJECT: ${COMPOSE_PROJECT_NAME}\n"
	save := func(name string) *httptest.ResponseRecorder {
		return registryRequest(t, SaveApplicationDeployment, app.ID, map[string]any{"environment_id": env.ID, "compose_project_name": name, "compose": compose, "execution": models.DeploymentExecution{Mode: "compose"}})
	}
	for _, name := range []string{"redis-test", "", "  redis-test  "} {
		response := save(name)
		if response.Code != 200 {
			t.Fatal(response.Body.String())
		}
		db.First(&env, env.ID)
		if env.ComposeProjectName != strings.TrimSpace(name) {
			t.Fatal("名称未保存或未裁剪空白")
		}
	}
	version := models.Release{ApplicationID: app.ID, Version: "v1", Compose: compose, Execution: models.DeploymentExecution{Mode: "compose"}}
	db.Create(&version)
	response := registryRequest(t, PreflightDeployment, 0, map[string]any{"environment_id": env.ID, "release_id": version.ID, "expected_project_name": "redis-test"})
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	var preview struct {
		ProjectName string `json:"project_name"`
		Compose     string `json:"compose"`
	}
	json.Unmarshal(response.Body.Bytes(), &preview)
	if preview.ProjectName != "redis-test" || !strings.Contains(preview.Compose, `"name":"redis-test"`) || !strings.Contains(preview.Compose, `"PROJECT":"redis-test"`) {
		t.Fatal("预览没有采用环境项目名称", response.Body.String())
	}
	for _, handler := range []gin.HandlerFunc{PreflightDeployment, EnqueueDeployment} {
		response := registryRequest(t, handler, 0, map[string]any{"environment_id": env.ID, "release_id": version.ID, "expected_project_name": "stale"})
		if response.Code != 409 {
			t.Fatal("接受了过期项目名称", response.Body.String())
		}
	}
	stale := env
	db.Model(&env).Update("compose_project_name", "redis-final")
	if _, err := release.Enqueue(stale, version, 1, "stale-project", false, 0); err == nil || !strings.Contains(err.Error(), "项目名称已变化") {
		t.Fatal("任务快照使用过期项目名称", err)
	}
	db.First(&env, env.ID)
	task, err := release.Enqueue(env, version, 1, "project-name", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot release.Snapshot
	json.Unmarshal([]byte(task.Snapshot), &snapshot)
	if release.ProjectName(snapshot.Environment) != "redis-final" {
		t.Fatal("任务未保存项目名称快照")
	}
	response = save("changed-after-enqueue")
	if response.Code != 400 || !strings.Contains(response.Body.String(), "已固定") {
		t.Fatal("排队后仍可改名", response.Body.String())
	}
	response = registryRequest(t, ApplicationDetail, app.ID, nil)
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"compose_project_name_locked":true`) {
		t.Fatal("界面未收到锁定状态", response.Body.String())
	}
}

func TestComposeProjectNameValidationAndHostConflicts(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Environment{}, &models.Deployment{}, &models.TargetState{}); err != nil {
		t.Fatal(err)
	}
	original := models.Environment{ID: 3, ApplicationID: 2, Name: "未部署", TargetID: 1}
	for _, name := range []string{"Redis", "-redis", "_redis", "redis.test", "有中文", strings.Repeat("a", 101), "dockerun-a2-e4"} {
		env := original
		env.ComposeProjectName = name
		if validateEnvironmentProject(&env, &original) == nil {
			t.Fatal("接受无效名称", name)
		}
	}
	for _, name := range []string{"", "redis_test-1", "1redis", "dockerun-a2-e3"} {
		env := original
		env.ComposeProjectName = name
		if err := validateEnvironmentProject(&env, &original); err != nil {
			t.Fatal(name, err)
		}
	}
	other := models.Environment{ApplicationID: 5, Name: "已有环境", TargetID: 2, TargetIDs: "[2,3]", ComposeProjectName: "redis-prod"}
	db.Create(&other)
	for _, tc := range []struct {
		target   uint
		ids      string
		conflict bool
	}{{1, "", false}, {2, "", true}, {1, "[1,3]", true}, {4, "[4,5]", false}} {
		env := original
		env.TargetID = tc.target
		env.TargetIDs = tc.ids
		env.ComposeProjectName = "redis-prod"
		if err := validateEnvironmentProject(&env, &original); (err != nil) != tc.conflict {
			t.Fatalf("主机冲突判断错误：%+v %v", tc, err)
		}
	}
}

func TestComposeProjectNameLocksIncludeFailureAndHiddenHistory(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Environment{}, &models.Deployment{}, &models.TargetState{}); err != nil {
		t.Fatal(err)
	}
	for i, status := range []string{"queued", "running", "success", "failed", "cancelled", "hidden", "legacy-environment", "legacy-target"} {
		original := models.Environment{ID: uint(i + 1), ApplicationID: 1, Name: status}
		switch status {
		case "legacy-environment":
			original.CurrentReleaseID = 5
		case "legacy-target":
			db.Create(&models.TargetState{EnvironmentID: original.ID, TargetID: 1, CurrentReleaseID: 5})
		default:
			task := models.Deployment{EnvironmentID: original.ID, Status: status, IdempotencyKey: status}
			if status == "hidden" {
				now := time.Now()
				task.HistoryDeletedAt = &now
			}
			db.Create(&task)
		}
		candidate := original
		candidate.ComposeProjectName = "renamed"
		if err := validateEnvironmentProject(&candidate, &original); err == nil || !strings.Contains(err.Error(), "已固定") {
			t.Fatal("已使用环境允许改名", status, err)
		}
		candidate.ComposeProjectName = release.ProjectName(original)
		if err := validateEnvironmentProject(&candidate, &original); err != nil {
			t.Fatal("同一实际名称误报改名", err)
		}
		candidate = original
		candidate.ComposeProjectName = "original-custom"
		original = candidate
		candidate.ComposeProjectName = ""
		if validateEnvironmentProject(&candidate, &original) == nil {
			t.Fatal("已部署的自定义名称被清空")
		}
	}
}

func TestOldAgentCannotIgnoreComposeProjectName(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Target{}, &models.Deployment{}, &models.TaskLog{}, &models.Environment{}); err != nil {
		t.Fatal(err)
	}
	target := models.Target{ID: 2, Name: "旧 Agent", Enabled: true}
	db.Create(&target)
	snapshot, _ := json.Marshal(release.Snapshot{Environment: models.Environment{ID: 1, TargetID: 2, ComposeProjectName: "custom-redis"}})
	task := models.Deployment{Status: "queued", TargetID: 2, Snapshot: string(snapshot), IdempotencyKey: "custom-project-old-agent"}
	db.Create(&task)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("target", target)
	requestContext, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	ctx.Request = httptest.NewRequest("GET", "/agent/poll?platform=linux/amd64&release_policy=2", bytes.NewReader(nil)).WithContext(requestContext)
	AgentPoll(ctx)
	db.First(&task, task.ID)
	if task.Status != "failed" || !strings.Contains(task.Result, "自定义 Compose 项目名称") {
		t.Fatal(fmt.Sprintf("旧 Agent 忽略了名称：%+v", task))
	}
}

func TestSaveEnvironmentPreservesOmittedProjectAndRejectsLockedRename(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Application{}, &models.Environment{}, &models.Deployment{}, &models.TargetState{}, &models.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&models.Registry{Name: "内置仓库", Kind: "builtin"})
	app := models.Application{Name: "环境名称配置"}
	db.Create(&app)
	response := registryRequest(t, SaveEnvironment, app.ID, map[string]any{"name": "测试", "target_id": 1, "compose_project_name": "redis-test"})
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	var env models.Environment
	json.Unmarshal(response.Body.Bytes(), &env)
	if env.ComposeProjectName != "redis-test" {
		t.Fatal("新环境名称未保存")
	}
	response = registryRequest(t, SaveEnvironment, app.ID, map[string]any{"id": env.ID, "name": env.Name, "target_id": 1})
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	db.First(&env, env.ID)
	if env.ComposeProjectName != "redis-test" {
		t.Fatal("旧客户端未提交名称时清空了配置")
	}
	db.Create(&models.Deployment{EnvironmentID: env.ID, Status: "failed", IdempotencyKey: "failed-name-lock"})
	response = registryRequest(t, SaveEnvironment, app.ID, map[string]any{"id": env.ID, "name": env.Name, "target_id": 1, "compose_project_name": ""})
	if response.Code != 400 || !strings.Contains(response.Body.String(), "已固定") {
		t.Fatal("环境接口允许清空已使用名称", response.Body.String())
	}
}
