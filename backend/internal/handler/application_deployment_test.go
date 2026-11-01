package handler

import (
	"dockerun/internal/config"
	"dockerun/internal/models"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveCommandConfigurationAndPreviewDoNotExecute(t *testing.T) {
	db := registryTestDB(t)
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Application{}, &models.Environment{}, &models.Release{}, &models.Deployment{}, &models.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	app := models.Application{Name: "command"}
	db.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "test", TargetID: 1, Parameters: models.StringMap{"VALUE": "literal $value"}, ScriptMode: "override", AutoRollback: true, RegistryID: 42}
	db.Create(&env)
	marker := filepath.Join(t.TempDir(), "must-not-exist")
	execution := models.DeploymentExecution{Mode: "command", Command: "touch '" + marker + "'", TimeoutSeconds: 10}
	scripts := models.DeploymentScripts{Before: "echo before", After: "echo after"}
	response := registryRequest(t, SaveApplicationDeployment, app.ID, map[string]any{"environment_id": env.ID, "execution": execution, "scripts": scripts, "configs": map[string]string{"configs/empty.env": ""}})
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	db.First(&app, app.ID)
	db.First(&env, env.ID)
	if app.Definition.Execution != execution || env.Scripts != scripts || env.RegistryID != 42 || !env.AutoRollback || env.Parameters["VALUE"] != "literal $value" {
		t.Fatalf("save lost unrelated state: %+v %+v", app, env)
	}
	if app.Scripts != (models.DeploymentScripts{}) {
		t.Fatal("environment override overwrote application scripts")
	}
	response = registryRequest(t, CreateRelease, app.ID, models.Release{Execution: execution, Configs: `{"configs/empty.env":""}`})
	if response.Code != http.StatusCreated {
		t.Fatal(response.Body.String())
	}
	var version models.Release
	if err := json.Unmarshal(response.Body.Bytes(), &version); err != nil {
		t.Fatal(err)
	}
	if version.Source != "command" || version.Execution != execution || version.Version == "" {
		t.Fatal("release did not preserve command", response.Body.String())
	}
	changed := execution
	changed.Command = "echo next-deployment"
	response = registryRequest(t, SaveApplicationDeployment, app.ID, map[string]any{"environment_id": env.ID, "execution": changed, "scripts": scripts, "configs": map[string]string{"configs/empty.env": "next"}})
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	var historical models.Release
	db.First(&historical, version.ID)
	if historical.Execution != execution || historical.Configs != version.Configs {
		t.Fatal("saving new configuration mutated a historical release")
	}
	response = registryRequest(t, PreflightDeployment, 0, map[string]any{"environment_id": env.ID, "release_id": version.ID, "expected_scripts": scripts})
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("save or preview executed the deployment")
	}
	var count int64
	db.Model(&models.Deployment{}).Count(&count)
	if count != 0 {
		t.Fatal("save or preview created task")
	}
	response = registryRequest(t, EnqueueDeployment, 0, map[string]any{"environment_id": env.ID, "release_id": version.ID, "expected_scripts": models.DeploymentScripts{Before: "stale"}})
	if response.Code != http.StatusConflict {
		t.Fatal("stale script confirmation accepted", response.Body.String())
	}
}

func TestSaveDeploymentValidatesFilesAndScripts(t *testing.T) {
	db := registryTestDB(t)
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&models.Application{}, &models.Environment{}, &models.AuditEvent{})
	app := models.Application{Name: "validate"}
	db.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "test"}
	db.Create(&env)
	for _, payload := range []map[string]any{
		{"environment_id": env.ID, "execution": models.DeploymentExecution{Mode: "command", Command: "echo ok"}, "configs": map[string]string{"../escape": "bad"}},
		{"environment_id": env.ID, "execution": models.DeploymentExecution{Mode: "command", Command: "echo ok"}, "scripts": models.DeploymentScripts{After: "if then"}},
		{"environment_id": env.ID + 1, "execution": models.DeploymentExecution{Mode: "command", Command: "echo ok"}},
	} {
		response := registryRequest(t, SaveApplicationDeployment, app.ID, payload)
		if response.Code < 400 {
			t.Fatal("invalid deployment saved", response.Body.String())
		}
	}
	db.First(&app, app.ID)
	if app.Definition.Execution.Mode != "" {
		t.Fatal("failed validation changed application")
	}
}
