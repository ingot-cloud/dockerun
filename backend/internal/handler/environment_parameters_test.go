package handler

import (
	"bytes"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func parameterRequest(t *testing.T, applicationID, environmentID uint, payload any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(applicationID)}, {Key: "environment_id", Value: fmt.Sprint(environmentID)}}
	ctx.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(raw))
	ctx.Request.Header.Set("Content-Type", "application/json")
	SaveEnvironmentParameters(ctx)
	return response
}

func TestSaveEnvironmentParametersPreservesOtherConfiguration(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Application{}, &models.Environment{}, &models.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	app := models.Application{Name: "网关"}
	db.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "生产", ComposeProjectName: "gateway-prod", TargetID: 2, TargetIDs: "[2,3]", CurrentReleaseID: 9, RegistryID: 42, AutoDeploy: true, Repository: "team/app", TagPattern: "v*", AllowedPublishers: "[1]", Stateless: true, AutoRollback: true, ScriptMode: "override", Scripts: models.DeploymentScripts{Before: "echo before"}, Parameters: models.StringMap{"OLD": "remove", "PASSWORD": "old"}, Health: "healthy", RuntimeState: "running"}
	db.Create(&env)
	before := env.Parameters
	updated := models.StringMap{"PASSWORD": "  value=$secret;中文  ", "EMPTY": ""}
	response := parameterRequest(t, app.ID, env.ID, map[string]any{"parameters": updated, "expected_parameters": before})
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	var stored models.Environment
	db.First(&stored, env.ID)
	want := env
	want.Parameters = updated
	if !reflect.DeepEqual(stored, want) {
		t.Fatal("保存参数修改了其他环境配置，或丢失空值与原始参数值")
	}
	response = parameterRequest(t, app.ID, env.ID, map[string]any{"parameters": models.StringMap{"OTHER": "stale"}, "expected_parameters": before})
	if response.Code != http.StatusConflict {
		t.Fatal("过期草稿覆盖了新的参数", response.Body.String())
	}
	response = parameterRequest(t, app.ID+1, env.ID, map[string]any{"parameters": updated})
	if response.Code != http.StatusNotFound {
		t.Fatal("允许更新其他应用的环境")
	}
	for _, parameters := range []any{nil, map[string]any{"PORT": 8080}, models.StringMap{"BAD-NAME": "value"}, models.StringMap{"VALUE": "a\x00b"}} {
		response = parameterRequest(t, app.ID, env.ID, map[string]any{"parameters": parameters})
		if response.Code != http.StatusBadRequest {
			t.Fatal("接受了无效参数", response.Body.String())
		}
	}
	response = parameterRequest(t, app.ID, env.ID, map[string]any{"parameters": models.StringMap{}, "expected_parameters": updated})
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	db.First(&stored, env.ID)
	if len(stored.Parameters) != 0 || stored.Scripts != env.Scripts {
		t.Fatal("删除全部参数失败，或影响了脚本配置")
	}
}

func TestSaveEnvironmentParametersValidatesTemplateAndDerivesPorts(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Application{}, &models.Environment{}, &models.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	reg := models.Registry{Name: "内置", Kind: "builtin", URL: "https://registry.test"}
	db.Create(&reg)
	app := models.Application{Name: "模板", Definition: models.ApplicationDefinition{Parameters: []models.ApplicationParameter{
		{Key: "IMAGE", Label: "镜像", Type: "image", Required: true},
		{Key: "PASSWORD", Label: "密码", Type: "password", Required: true},
		{Key: "DATA_DIR", Label: "数据目录", Type: "path", Required: true},
		{Key: "NACOS_PORT", Label: "服务端口", Type: "port", Required: true},
	}}}
	db.Create(&app)
	initial := models.StringMap{"IMAGE": "registry.test/team/app:v1", "PASSWORD": "old", "DATA_DIR": "/srv/data", "NACOS_PORT": "8848", "NACOS_GRPC_PORT": "9848"}
	env := models.Environment{ApplicationID: app.ID, Name: "测试", RegistryID: reg.ID, Parameters: initial}
	db.Create(&env)
	for key, invalid := range map[string]string{"PASSWORD": "", "DATA_DIR": "relative", "NACOS_PORT": "65536"} {
		values := models.StringMap{}
		for name, value := range initial {
			values[name] = value
		}
		values[key] = invalid
		response := parameterRequest(t, app.ID, env.ID, map[string]any{"parameters": values})
		if response.Code != http.StatusBadRequest {
			t.Fatal("模板校验被绕过", key, response.Body.String())
		}
		var stored models.Environment
		db.First(&stored, env.ID)
		if !reflect.DeepEqual(stored.Parameters, initial) {
			t.Fatal("校验失败后仍更新了环境参数")
		}
	}
	values := models.StringMap{"IMAGE": "team/app:v2", "PASSWORD": "  unchanged whitespace  ", "DATA_DIR": "/srv/new", "NACOS_PORT": "9000", "CUSTOM": "extra"}
	response := parameterRequest(t, app.ID, env.ID, map[string]any{"parameters": values, "expected_parameters": initial})
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	var stored models.Environment
	db.First(&stored, env.ID)
	if stored.Parameters["IMAGE"] != "registry.test/team/app:v2" || stored.Parameters["NACOS_GRPC_PORT"] != "10000" || stored.Parameters["CUSTOM"] != "extra" || stored.Parameters["PASSWORD"] != values["PASSWORD"] {
		t.Fatal("镜像、派生端口、附加参数或原始密码值处理错误")
	}
}

func TestDeploymentRejectsParametersChangedAfterConfirmation(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Application{}, &models.Environment{}, &models.Release{}, &models.Deployment{}); err != nil {
		t.Fatal(err)
	}
	app := models.Application{Name: "应用"}
	db.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "生产", Parameters: models.StringMap{"PORT": "9090"}}
	db.Create(&env)
	version := models.Release{ApplicationID: app.ID, Version: "v1"}
	db.Create(&version)
	for _, handler := range []gin.HandlerFunc{PreflightDeployment, EnqueueDeployment} {
		response := registryRequest(t, handler, 0, map[string]any{"environment_id": env.ID, "release_id": version.ID, "expected_parameters": models.StringMap{"PORT": "8080"}, "idempotency_key": "stale-parameters"})
		if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "环境运行参数已更新") {
			t.Fatal("参数变化后仍允许按旧确认继续发布", response.Body.String())
		}
	}
	var count int64
	db.Model(&models.Deployment{}).Count(&count)
	if count != 0 {
		t.Fatal("参数冲突仍创建了部署任务")
	}
}
