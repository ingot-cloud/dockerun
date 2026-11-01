package handler

import (
	"dockerun/internal/bundle"
	"dockerun/internal/config"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOfflineCommandVersionsFreezeArchiveAndReuseSavedConfiguration(t *testing.T) {
	db := registryTestDB(t)
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Application{}, &models.Environment{}, &models.Release{}, &models.Bundle{}, &models.Deployment{}, &models.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	app := models.Application{Name: "offline-app"}
	db.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "内网环境", TargetID: 1, RegistryID: 42, Parameters: models.StringMap{"APP_NAME": "offline-app"}, ScriptMode: "inherit"}
	db.Create(&env)
	execution := models.DeploymentExecution{Mode: "command", Command: `docker run --name "$APP_NAME" "$DOCKERUN_IMAGE"`}
	response := registryRequest(t, SaveApplicationDeployment, app.ID, map[string]any{"environment_id": env.ID, "source": "local", "execution": execution, "container_names": []string{"$APP_NAME"}, "replace_containers": true})
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	db.First(&env, env.ID)
	if !env.ReplaceContainers || !reflect.DeepEqual(env.ContainerNames, []string{"$APP_NAME"}) || env.RegistryID != 42 || env.Parameters["APP_NAME"] != "offline-app" {
		t.Fatalf("saving replacement changed unrelated environment configuration: %+v", env)
	}
	var versions []models.Release
	for index, kind := range []string{"image", "bundle"} {
		image := release.Image{Reference: "company/app:latest", ID: "sha256:" + strings.Repeat(fmt.Sprint(index+1), 64), Platform: "linux/amd64"}
		var manifest any = []release.Image{image}
		if kind == "bundle" {
			manifest = bundle.Manifest{Application: app.Name, Version: "v2", Platform: image.Platform, Images: []release.Image{image}}
		}
		encoded, _ := json.Marshal(manifest)
		material := models.Bundle{Kind: kind, Name: kind, Status: "ready", SHA256: fmt.Sprint(index), Manifest: string(encoded)}
		db.Create(&material)
		response = registryRequest(t, CreateRelease, app.ID, map[string]any{"source": "local", "execution": execution, "bundle_id": material.ID, "images": `[{"id":"forged"}]`, "save_definition": true})
		if response.Code != http.StatusCreated {
			t.Fatal(response.Body.String())
		}
		var version models.Release
		if err := json.Unmarshal(response.Body.Bytes(), &version); err != nil {
			t.Fatal(err)
		}
		var images []release.Image
		json.Unmarshal([]byte(version.Images), &images)
		if version.BundleID != material.ID || version.Execution != execution || version.Platform != image.Platform || len(images) != 1 || images[0].ID != image.ID || version.TriggerImage != image.Reference {
			t.Fatalf("offline version did not freeze actual archive identity: %+v", version)
		}
		if kind == "bundle" && version.Source != "bundle" || kind == "image" && version.Source != "local" {
			t.Fatal("archive source mismatch", version.Source)
		}
		versions = append(versions, version)
	}
	var first models.Release
	db.First(&first, versions[0].ID)
	if first.Images != versions[0].Images || first.Execution != versions[0].Execution || first.BundleID != versions[0].BundleID {
		t.Fatal("second release rewrote historical image or script")
	}
	db.First(&app, app.ID)
	if app.Definition.Execution != execution || app.Definition.Source != "bundle" {
		t.Fatal("offline deployment configuration was not reusable", app.Definition)
	}
	response = registryRequest(t, PreflightDeployment, 0, map[string]any{"environment_id": env.ID, "release_id": first.ID})
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "不删除镜像、不拉取仓库") || !strings.Contains(response.Body.String(), "company/app:latest") {
		t.Fatal("historical offline preview lost replacement policy or image manifest", response.Body.String())
	}
}

func TestOfflineCommandRejectsMissingMaterialsAndAmbiguousPrimaryImage(t *testing.T) {
	db := registryTestDB(t)
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Application{}, &models.Release{}, &models.Bundle{}, &models.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	app := models.Application{Name: "offline"}
	db.Create(&app)
	images := []release.Image{{Reference: "app:v1", ID: "sha256:" + strings.Repeat("a", 64), Platform: "linux/amd64"}, {Reference: "worker:v1", ID: "sha256:" + strings.Repeat("b", 64), Platform: "linux/amd64"}}
	encoded, _ := json.Marshal(images)
	material := models.Bundle{Kind: "image", Status: "ready", SHA256: "multi", Manifest: string(encoded)}
	db.Create(&material)
	request := map[string]any{"source": "local", "execution": models.DeploymentExecution{Mode: "command", Command: `echo "$DOCKERUN_IMAGE"`}}
	for _, id := range []uint{0, 999, material.ID} {
		request["bundle_id"] = id
		response := registryRequest(t, CreateRelease, app.ID, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid archive/primary image accepted: %d %s", id, response.Body.String())
		}
	}
	request["trigger_image"] = "worker:v1"
	response := registryRequest(t, CreateRelease, app.ID, request)
	if response.Code != http.StatusCreated {
		t.Fatal(response.Body.String())
	}
}

func TestAgentCanDownloadRetainedImageArchive(t *testing.T) {
	db := registryTestDB(t)
	if err := db.AutoMigrate(&models.Deployment{}, &models.Bundle{}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "app.tar")
	os.WriteFile(path, []byte("image-archive"), 0600)
	material := models.Bundle{Kind: "image", Status: "ready", Path: path, SHA256: "archive-checksum"}
	db.Create(&material)
	snapshot, _ := json.Marshal(release.Snapshot{Release: models.Release{Source: "local", BundleID: material.ID}, Environment: models.Environment{TargetID: 2}})
	task := models.Deployment{Status: "running", TargetID: 2, Snapshot: string(snapshot), IdempotencyKey: "offline-agent"}
	db.Create(&task)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/agent/tasks/1/bundle", nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(task.ID)}}
	ctx.Set("target", models.Target{ID: 2})
	AgentBundle(ctx)
	if w.Code != http.StatusOK || w.Body.String() != "image-archive" || w.Header().Get("X-Archive-Kind") != "image" || w.Header().Get("X-Archive-SHA256") != material.SHA256 {
		t.Fatal("Agent did not receive version's retained archive", w.Code, w.Body.String(), w.Header())
	}
}
