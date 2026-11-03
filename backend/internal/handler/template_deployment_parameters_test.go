package handler

import (
	"bytes"
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTemplateDeploymentPreservesConfirmedParameters(t *testing.T) {
	db := registryTestDB(t)
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("PUBLIC_URL", "https://registry.test")
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Application{}, &models.ApplicationTemplate{}, &models.Environment{}, &models.Target{}, &models.TargetState{}, &models.Release{}, &models.Deployment{}, &models.AuditEvent{}, &models.User{}, &models.SystemConfig{}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []any{
		&models.Registry{Name: "内置仓库", Kind: "builtin", URL: "https://registry.test"},
		&models.Target{ID: 1, Name: "本机", Enabled: true},
		&models.User{Username: "admin", Password: "initialized-fixture", Enabled: true, IsAdmin: true, AuthVersion: 1},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	previousEngine := builtin.Engine
	builtin.Engine = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
		w.Write([]byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{},"layers":[]}`))
	})
	t.Cleanup(func() { builtin.Engine = previousEngine })
	request := func(t *testing.T, handler gin.HandlerFunc, id, path string, payload any) *httptest.ResponseRecorder {
		t.Helper()
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(response)
		ctx.Params = gin.Params{{Key: "id", Value: id}}
		ctx.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
		ctx.Request.Header.Set("Content-Type", "application/json")
		handler(ctx)
		return response
	}
	decode := func(t *testing.T, response *httptest.ResponseRecorder, status int, result any) {
		t.Helper()
		if response.Code != status {
			t.Fatalf("状态 %d，预期 %d：%s", response.Code, status, response.Body.String())
		}
		if err := json.Unmarshal(response.Body.Bytes(), result); err != nil {
			t.Fatal(err)
		}
	}
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("旧版参数=%t", legacy), func(t *testing.T) {
			var created struct {
				Application models.Application `json:"application"`
				Environment models.Environment `json:"environment"`
			}
			response := request(t, CreateApplicationFromTemplate, "redis", "/", map[string]any{
				"name": fmt.Sprintf("Redis-%t", legacy), "environment": "默认环境", "parameters": models.StringMap{"REDIS_PASSWORD": "  literal$secret  "},
			})
			decode(t, response, http.StatusCreated, &created)
			parameters := created.Environment.Parameters
			if _, exists := parameters["COMPOSE_PROJECT_NAME"]; exists {
				t.Error("模板创建将 Compose 内部项目名写入环境参数")
			}
			if legacy {
				// 兼容修复前已经创建、保存了预览项目名的应用。
				parameters["COMPOSE_PROJECT_NAME"] = "dockerun-template-preview"
				if err := db.Model(&models.Environment{}).Where("id = ?", created.Environment.ID).Update("parameters", parameters).Error; err != nil {
					t.Fatal(err)
				}
			}
			confirmed := maps.Clone(parameters)
			appID := fmt.Sprint(created.Application.ID)
			response = request(t, SaveApplicationDeployment, appID, "/", map[string]any{
				"environment_id": created.Environment.ID, "source": "registry", "compose": created.Application.Definition.Compose,
				"configs": created.Application.Definition.Configs, "execution": models.DeploymentExecution{Mode: "compose"},
			})
			var saved struct{ Environment models.Environment }
			decode(t, response, http.StatusOK, &saved)
			if !maps.Equal(saved.Environment.Parameters, confirmed) {
				t.Error("保存部署配置返回了与已确认值不同的运行参数")
			}
			configs, err := json.Marshal(created.Application.Definition.Configs)
			if err != nil {
				t.Fatal(err)
			}
			response = request(t, CreateRelease, appID, fmt.Sprintf("/?environment_id=%d", created.Environment.ID), map[string]any{
				"source": "registry", "compose": created.Application.Definition.Compose, "configs": string(configs), "save_definition": true,
			})
			var version models.Release
			decode(t, response, http.StatusCreated, &version)
			payload := map[string]any{"environment_id": created.Environment.ID, "release_id": version.ID, "expected_parameters": saved.Environment.Parameters}
			var preview struct{ Parameters models.StringMap }
			decode(t, request(t, PreflightDeployment, "", "/", payload), http.StatusOK, &preview)
			if preview.Parameters["REDIS_PASSWORD"] != "******" {
				t.Fatal("预览未隐藏敏感参数")
			}
			payload["idempotency_key"] = fmt.Sprintf("template-parameters-%t", legacy)
			var task models.Deployment
			decode(t, request(t, EnqueueDeployment, "", "/", payload), http.StatusAccepted, &task)
			if err := db.First(&task, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			var snapshot release.Snapshot
			if err := json.Unmarshal([]byte(task.Snapshot), &snapshot); err != nil {
				t.Fatal(err)
			}
			var stored models.Environment
			if err := db.First(&stored, created.Environment.ID).Error; err != nil {
				t.Fatal(err)
			}
			if !maps.Equal(stored.Parameters, confirmed) || !maps.Equal(snapshot.Environment.Parameters, confirmed) {
				t.Fatal("发布预览或任务快照改写了已确认的运行参数")
			}
		})
	}
}
