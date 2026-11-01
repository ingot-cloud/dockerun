package release

import (
	"context"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/client"
)

func TestCommandAutomaticDeploymentsFreezeSavedConfigurationAndTriggerImage(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := database.Init(config.Load().DatabasePath); err != nil {
		t.Fatal(err)
	}
	var admin models.User
	database.DB.First(&admin, 1)
	if err := admin.HashPassword("test-password"); err != nil {
		t.Fatal(err)
	}
	database.DB.Save(&admin)
	app := models.Application{Name: "command-events", Scripts: models.DeploymentScripts{Before: "echo old-before"}, Definition: models.ApplicationDefinition{
		Execution: models.DeploymentExecution{Mode: "command", Command: `printf '%s\n' "$DOCKERUN_IMAGE" "$DOCKERUN_IMAGE_TAG" > observed`, CheckCommand: "echo old-check"},
		Configs:   models.StringMap{"configs/.env": "old-config"},
	}}
	database.DB.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "test", TargetID: 1, RegistryID: 1, Repository: "team/app", TagPattern: "test-*", AutoDeploy: true, AllowedPublishers: "[]"}
	database.DB.Create(&env)
	if _, err := AutomationBase(env); err != nil {
		t.Fatal("saved command should support automation before its first deployment", err)
	}
	push := func(key string) Snapshot {
		t.Helper()
		event := models.ImageEvent{RegistryID: 1, Repository: "team/app", Reference: key, Digest: "sha256:" + strings.Repeat("a", 64), Key: key, Committed: true, UserID: 1}
		database.DB.Create(&event)
		processEvents(context.Background())
		processEvents(context.Background())
		var tasks []models.Deployment
		database.DB.Where("event_id = ?", event.ID).Find(&tasks)
		if len(tasks) != 1 {
			t.Fatalf("expected one automatic task, got %d", len(tasks))
		}
		var s Snapshot
		if err := json.Unmarshal([]byte(tasks[0].Snapshot), &s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	first := push("test-first")
	if first.Release.Execution != app.Definition.Execution || first.Release.Configs != stringJSON(app.Definition.Configs) || first.Scripts != app.Scripts {
		t.Fatal("automatic task lost saved deployment configuration")
	}
	var releaseBefore models.Release
	database.DB.First(&releaseBefore, first.Release.ID)
	app.Definition.Execution.Command = `echo new > observed; echo "$DOCKERUN_IMAGE"`
	app.Definition.Execution.CheckCommand = "echo new-check"
	app.Definition.Configs["configs/.env"] = "new-config"
	app.Scripts.Before = "echo new-before"
	database.DB.Save(&app)
	second := push("test-second")
	if second.Release.Execution != app.Definition.Execution || second.Scripts != app.Scripts || second.Release.Configs == first.Release.Configs {
		t.Fatal("subsequent push reused stale deployment configuration")
	}
	var releaseAfter models.Release
	database.DB.First(&releaseAfter, first.Release.ID)
	if stringJSON(releaseBefore) != stringJSON(releaseAfter) {
		t.Fatal("saving new configuration changed a historical version")
	}
	dir := t.TempDir()
	first.Environment.Parameters = models.StringMap{"DOCKERUN_IMAGE": "spoofed"}
	if err := RunScript(context.Background(), first.Release.Execution.Command, "deploy_script", dir, first, func(string, string) {}); err != nil {
		t.Fatal(err)
	}
	observed, err := os.ReadFile(filepath.Join(dir, "observed"))
	if err != nil || !strings.Contains(string(observed), "/team/app@sha256:") || !strings.Contains(string(observed), "\ntest-first\n") || strings.Contains(string(observed), "spoofed") {
		t.Fatalf("trigger image was not frozen or was overridden: %s %v", observed, err)
	}
}

func TestAutomaticComposeUsesSavedContentAndResolvesAdditionalImages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/team/sidecar/manifests/v2" {
			t.Errorf("unexpected lookup: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"schemaVersion":2}`)
	}))
	defer server.Close()
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := database.Init(config.Load().DatabasePath); err != nil {
		t.Fatal(err)
	}
	reg := models.Registry{Name: "online", Kind: "external", URL: server.URL}
	database.DB.Create(&reg)
	host := strings.TrimPrefix(server.URL, "http://")
	app := models.Application{ID: 10, Definition: models.ApplicationDefinition{Source: "registry", RegistryID: reg.ID, Compose: "services:\n  app:\n    image: ${IMAGE}\n    environment:\n      PORT: ${PORT}\n  sidecar:\n    image: " + host + "/team/sidecar:v2\n", Configs: models.StringMap{"configs/app": "new"}, Execution: models.DeploymentExecution{Mode: "compose", Command: "echo saved-start", CheckCommand: "echo saved-check"}}}
	env := models.Environment{ID: 20, ApplicationID: 10, RegistryID: reg.ID, Repository: "team/app", Parameters: models.StringMap{"IMAGE": host + "/team/app:latest", "PORT": "9090"}}
	base, err := AutomationDefinition(app, env)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(base.Compose, "${PORT}") || strings.Contains(base.Compose, "${IMAGE}") {
		t.Fatal("runtime placeholders or image references were incorrectly frozen")
	}
	r, err := automaticRelease(context.Background(), base, env, models.ImageEvent{ID: 30, RegistryID: reg.ID, Repository: "team/app", Reference: "v3", Digest: "sha256:" + strings.Repeat("b", 64)})
	if err != nil {
		t.Fatal(err)
	}
	var images []Image
	json.Unmarshal([]byte(r.Images), &images)
	if len(images) != 2 || r.Execution != app.Definition.Execution || r.Configs != stringJSON(app.Definition.Configs) {
		t.Fatal("saved definition was not used")
	}
	for _, im := range images {
		if im.Digest == "" {
			t.Fatal("additional image was not resolved")
		}
	}
}

func TestAutomaticCommandPreparesPinnedImageBeforeRunningScript(t *testing.T) {
	pulled := false
	httpClient := &http.Client{Transport: scriptDockerTransport(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/info"):
			fmt.Fprint(w, `{"OSType":"linux","Architecture":"aarch64"}`)
		case strings.HasSuffix(r.URL.Path, "/images/create"):
			pulled = true
			if r.URL.Query().Get("tag") != "sha256:"+strings.Repeat("a", 64) {
				t.Errorf("image was not pulled by digest: %s", r.URL)
			}
			fmt.Fprint(w, `{}`)
		case strings.Contains(r.URL.Path, "/images/") && strings.HasSuffix(r.URL.Path, "/json"):
			fmt.Fprint(w, `{"Id":"sha256:fixture","Os":"linux","Architecture":"arm64"}`)
		default:
			t.Errorf("unexpected Docker request: %s", r.URL.Path)
			w.WriteHeader(404)
		}
		return w.Result(), nil
	})}
	cli, err := client.NewClientWithOpts(client.WithHost("http://docker.test"), client.WithVersion("1.41"), client.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatal(err)
	}
	previous := docker.Client
	docker.Client = cli
	t.Cleanup(func() { docker.Client = previous; cli.Close() })
	r := models.Release{Source: "registry", Execution: models.DeploymentExecution{Mode: "command", Command: `echo "$DOCKERUN_IMAGE"`}, Images: stringJSON([]Image{{Reference: "registry.test/team/app:v1", Digest: "sha256:" + strings.Repeat("a", 64)}})}
	images, err := PrepareWithAuth(context.Background(), &r, "test-auth", func(string) {})
	if err != nil || !pulled || len(images) != 1 || images[0].ID != "sha256:fixture" {
		t.Fatalf("command image was not prepared: %+v %v", images, err)
	}
}
