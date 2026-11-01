package handler

import (
	"bytes"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func automationRequest(t *testing.T, app, environment uint, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(app)}, {Key: "environment_id", Value: fmt.Sprint(environment)}}
	ctx.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(raw))
	ctx.Request.Header.Set("Content-Type", "application/json")
	SaveEnvironmentAutomation(ctx)
	return response
}

func TestContainerBindingIsolationAndPreviewPolicy(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Application{}, &models.Environment{}, &models.Release{}, &models.Deployment{}, &models.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	reg := models.Registry{Name: "registry", Kind: "external", URL: "https://registry.test"}
	db.Create(&reg)
	app := models.Application{Name: "bindings", Definition: models.ApplicationDefinition{Execution: models.DeploymentExecution{Mode: "command", Command: `echo "$DOCKERUN_IMAGE"`}}}
	db.Create(&app)
	first := models.Environment{ApplicationID: app.ID, Name: "test", TargetID: 1, RegistryID: reg.ID, Parameters: models.StringMap{"APP_NAME": "test-nginx"}}
	db.Create(&first)
	second := models.Environment{ApplicationID: app.ID, Name: "prod", TargetID: 1, RegistryID: reg.ID}
	db.Create(&second)
	payload := map[string]any{"registry_id": reg.ID, "container_names": []string{"$APP_NAME"}, "replace_containers": true}
	response := automationRequest(t, app.ID, first.ID, payload)
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	db.First(&first, first.ID)
	if !first.ReplaceContainers || len(first.ContainerNames) != 1 || first.ContainerNames[0] != "$APP_NAME" {
		t.Fatal("binding was not preserved", first)
	}
	payload["container_names"] = []string{"test-nginx"}
	if response = automationRequest(t, app.ID, second.ID, payload); response.Code != http.StatusBadRequest {
		t.Fatal("same host container was bound twice")
	}
	db.Model(&second).Update("target_id", 2)
	if response = automationRequest(t, app.ID, second.ID, payload); response.Code != http.StatusOK {
		t.Fatal("different hosts could not reuse names", response.Body.String())
	}
	version := models.Release{ApplicationID: app.ID, Execution: models.DeploymentExecution{Mode: "command", Command: "echo deploy"}, Source: "command", Configs: "{}"}
	db.Create(&version)
	response = registryRequest(t, PreflightDeployment, 0, map[string]any{"environment_id": first.ID, "release_id": version.ID})
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	var preview struct {
		Policy release.ContainerPolicy `json:"container_policy"`
	}
	json.Unmarshal(response.Body.Bytes(), &preview)
	if !preview.Policy.Replace || len(preview.Policy.Names) != 1 || preview.Policy.Names[0] != "test-nginx" {
		t.Fatal("preview did not expose exact replacement scope")
	}
	db.Model(&models.Environment{}).Where("id = ?", first.ID).Update("container_names", `["another-nginx"]`)
	response = registryRequest(t, EnqueueDeployment, 0, map[string]any{"environment_id": first.ID, "release_id": version.ID, "expected_containers": preview.Policy})
	if response.Code != http.StatusConflict {
		t.Fatal("changed deletion scope accepted", response.Body.String())
	}
	snapshot, _ := json.Marshal(release.Snapshot{Environment: first})
	db.Create(&models.Deployment{EnvironmentID: first.ID, TargetID: 1, Status: "queued", IdempotencyKey: "bound-task", Snapshot: string(snapshot)})
	third := models.Environment{ApplicationID: app.ID, Name: "another", TargetID: 1, RegistryID: reg.ID}
	db.Create(&third)
	if response = automationRequest(t, app.ID, third.ID, payload); response.Code != http.StatusBadRequest {
		t.Fatal("container claimed while another environment still has a pending task")
	}
}

func TestCommandAutomationSettingsPreserveEnvironmentAndDoNotDeploy(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Application{}, &models.Environment{}, &models.Release{}, &models.Deployment{}, &models.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	reg := models.Registry{Name: "online", Kind: "external", URL: "https://registry.test"}
	db.Create(&reg)
	app := models.Application{Name: "command-auto", Definition: models.ApplicationDefinition{Execution: models.DeploymentExecution{Mode: "command", Command: `docker run "${DOCKERUN_IMAGE:-$IMAGE}"`}}}
	db.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "prod", RegistryID: reg.ID, TargetID: 3, TargetIDs: "[3,4]", Parameters: models.StringMap{"IMAGE": "nginx:test", "SECRET": "retained"}, ScriptMode: "override", Scripts: models.DeploymentScripts{Before: "echo before"}, Stateless: true, AutoRollback: true, Health: "healthy", RuntimeState: "running"}
	db.Create(&env)
	payload := map[string]any{"auto_deploy": true, "registry_id": reg.ID, "repository": "team/app", "tag_pattern": "test-*", "allowed_publishers": "[5]", "parameters": map[string]string{"SECRET": "overwritten"}, "target_id": 99}
	response := automationRequest(t, app.ID, env.ID, payload)
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	var stored models.Environment
	db.First(&stored, env.ID)
	want := env
	want.AutoDeploy = true
	want.Repository = "team/app"
	want.TagPattern = "test-*"
	want.AllowedPublishers = "[5]"
	if !reflect.DeepEqual(stored, want) {
		t.Fatalf("automation changed unrelated state: %+v", stored)
	}
	var versions, tasks int64
	db.Model(&models.Release{}).Count(&versions)
	db.Model(&models.Deployment{}).Count(&tasks)
	if versions != 0 || tasks != 0 {
		t.Fatal("saving automation created a release or deployment")
	}
	response = automationRequest(t, app.ID+1, env.ID, payload)
	if response.Code != http.StatusNotFound {
		t.Fatal("another application's environment was updated")
	}
	for _, change := range []map[string]any{{"repository": "team/app:tag"}, {"tag_pattern": "["}, {"allowed_publishers": "{}"}} {
		invalid := map[string]any{}
		for k, v := range payload {
			invalid[k] = v
		}
		for k, v := range change {
			invalid[k] = v
		}
		if response = automationRequest(t, app.ID, env.ID, invalid); response.Code != http.StatusBadRequest {
			t.Fatal("invalid automation accepted", response.Body.String())
		}
	}
	app.Definition.Execution.Command = "docker run nginx:test"
	db.Save(&app)
	response = automationRequest(t, app.ID, env.ID, payload)
	if response.Code != http.StatusBadRequest {
		t.Fatal("fixed command did not require the trigger image")
	}
	// 关闭自动发布不能被不再适用的部署配置阻止。
	payload["auto_deploy"] = false
	response = automationRequest(t, app.ID, env.ID, payload)
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
}
