package handler

import (
	"dockerun/internal/apptemplate"
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateTemplateApplicationIsAtomicAndIndependent(t *testing.T) {
	db := registryTestDB(t)
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("PUBLIC_URL", "https://registry.test")
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Application{}, &models.ApplicationTemplate{}, &models.Environment{}, &models.Target{}, &models.AuditEvent{}, &models.Release{}, &models.User{}, &models.SystemConfig{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&models.Registry{Name: "builtin", Kind: "builtin", URL: "https://registry.test"})
	db.Create(&models.Target{ID: 1, Name: "本机", Enabled: true})
	db.Create(&models.User{Username: "admin", Password: "initialized-fixture", Enabled: true, IsAdmin: true, AuthVersion: 1})
	create := func(name string, password string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(response)
		ctx.Params = gin.Params{{Key: "id", Value: "mysql"}}
		body := fmt.Sprintf(`{"name":%q,"environment":"生产环境","parameters":{"MYSQL_ROOT_PASSWORD":%q}}`, name, password)
		ctx.Request = httptest.NewRequest("POST", "/managed-templates/mysql/applications", strings.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		CreateApplicationFromTemplate(ctx)
		return response
	}
	if response := create("invalid", ""); response.Code != 400 {
		t.Fatal("template accepted missing password")
	}
	var count int64
	db.Model(&models.Application{}).Count(&count)
	if count != 0 {
		t.Fatal("failed validation left an application behind")
	}
	response := create("database", "instance-secret")
	if response.Code != 201 {
		t.Fatal(response.Body.String())
	}
	var created struct {
		Application models.Application `json:"application"`
		Environment models.Environment `json:"environment"`
	}
	json.Unmarshal(response.Body.Bytes(), &created)
	if created.Application.TemplateID != "mysql" || created.Application.TemplateVersion == "" || !created.Application.Definition.Stateful || created.Environment.AutoDeploy || created.Environment.AutoRollback || created.Environment.Stateless {
		t.Fatal("middleware defaults were not preserved")
	}
	if strings.Contains(created.Application.Definition.Compose, "instance-secret") || created.Environment.Parameters["MYSQL_ROOT_PASSWORD"] != "instance-secret" {
		t.Fatal("credentials were not isolated to the environment")
	}
	if response := create("database", "another-secret"); response.Code != 400 {
		t.Fatal("duplicate app created")
	}
	db.Model(&models.Environment{}).Count(&count)
	if count != 1 {
		t.Fatal("duplicate request left an orphan environment")
	}
	// A saved template copies the definition, never the instance's parameter values or target.
	response = registryRequest(t, SaveApplicationTemplate, created.Application.ID, map[string]any{"name": "公司 MySQL", "version": "1.0.0"})
	if response.Code != 201 {
		t.Fatal(response.Body.String())
	}
	var saved models.ApplicationTemplate
	db.First(&saved)
	serialized, _ := json.Marshal(saved)
	if strings.Contains(string(serialized), "instance-secret") {
		t.Fatal("saved template leaked an environment secret")
	}
	before := created.Application.Definition.Compose
	saved.Definition.Compose = "services: {}"
	db.Save(&saved)
	db.First(&created.Application, created.Application.ID)
	if created.Application.Definition.Compose != before {
		t.Fatal("editing template changed the created app")
	}
	env := created.Environment
	env.TargetIDs = "[1,2]"
	response = registryRequest(t, SaveEnvironment, created.Application.ID, env)
	if response.Code != 400 || !strings.Contains(response.Body.String(), "集群编排") {
		t.Fatal("single-host template accepted a replicated host group")
	}
	env.TargetIDs = "[]"
	env.Stateless = true
	response = registryRequest(t, SaveEnvironment, created.Application.ID, env)
	if response.Code != 400 {
		t.Fatal("stateful template accepted stateless recovery")
	}
	// Publishing locks the image while keeping the reusable definition parameterized.
	previousEngine := builtin.Engine
	builtin.Engine = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
		w.Write([]byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{},"layers":[]}`))
	})
	t.Cleanup(func() { builtin.Engine = previousEngine })
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(created.Application.ID)}}
	request, _ := json.Marshal(map[string]any{"source": "registry", "compose": created.Application.Definition.Compose, "save_definition": true})
	ctx.Request = httptest.NewRequest("POST", fmt.Sprintf("/applications/%d/releases?environment_id=%d", created.Application.ID, created.Environment.ID), strings.NewReader(string(request)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	CreateRelease(ctx)
	if w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	var version models.Release
	json.Unmarshal(w.Body.Bytes(), &version)
	if !strings.Contains(version.Compose, "registry.test/mysql:8.0.44") || strings.Contains(version.Compose, "instance-secret") {
		t.Fatal("release did not freeze image without freezing runtime credentials")
	}
	db.First(&created.Application, created.Application.ID)
	if !strings.Contains(created.Application.Definition.Compose, "${MYSQL_IMAGE:") {
		t.Fatal("reusable application definition lost its image placeholder")
	}
	response = registryRequest(t, PreflightDeployment, 0, map[string]any{"release_id": version.ID, "environment_id": created.Environment.ID})
	if response.Code != 200 || strings.Contains(response.Body.String(), "instance-secret") || !strings.Contains(response.Body.String(), "******") {
		t.Fatal("preview exposed a template password: " + response.Body.String())
	}
}

func TestManagedCatalogIncludesSavedVersions(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.ApplicationTemplate{}); err != nil {
		t.Fatal(err)
	}
	template := apptemplate.Builtins()[0]
	db.Create(&models.ApplicationTemplate{Name: "团队服务", Version: "1.0.0", Definition: template.Definition, Category: "business"})
	items, err := applicationTemplates()
	if err != nil || len(items) != 6 {
		t.Fatalf("catalog: %d %v", len(items), err)
	}
	if items[5].Source != "custom" || !strings.HasPrefix(items[5].ID, "custom-") {
		t.Fatal("saved template was not available for reuse")
	}
}
