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
	"reflect"
	"strings"
	"testing"

	"github.com/docker/docker/client"
)

func TestContainerBindingNames(t *testing.T) {
	env := models.Environment{ReplaceContainers: true, ContainerNames: []string{"$APP_NAME", "sidecar"}, Parameters: models.StringMap{"APP_NAME": "test-nginx"}}
	policy, err := ContainerSettings(env)
	if err != nil || !reflect.DeepEqual(policy.Names, []string{"sidecar", "test-nginx"}) {
		t.Fatal(policy, err)
	}
	for _, names := range [][]string{nil, {""}, {"../nginx"}, {"test-nginx", "$APP_NAME"}, {"$MISSING"}} {
		env.ContainerNames = names
		if _, err := ContainerSettings(env); err == nil {
			t.Fatalf("invalid bindings accepted: %v", names)
		}
	}
}

func TestManagedContainerReplacementOrderAndIsolation(t *testing.T) {
	for _, scenario := range []string{"replace", "shared-image", "foreign-owner", "pull-failure", "checkpoint-failure", "wrong-image", "literal-tag", "absent-tag", "manual-online", "manual-pull-failure"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			marker := filepath.Join(dir, "created")
			steps := []string{}
			removed, pulled := false, false
			const ref = "registry.test/nginx:test"
			digest := "sha256:" + strings.Repeat("a", 64)
			transport := scriptDockerTransport(func(r *http.Request) (*http.Response, error) {
				w := httptest.NewRecorder()
				w.Header().Set("Content-Type", "application/json")
				path := r.URL.Path
				switch {
				case strings.HasSuffix(path, "/containers/test-nginx/json"):
					if !removed {
						labels := map[string]string{}
						if scenario == "foreign-owner" {
							labels["io.dockerun.environment"] = "999"
						}
						json.NewEncoder(w).Encode(map[string]any{"Id": "old-container", "Name": "/test-nginx", "Image": "sha256:old", "State": map[string]any{"Running": true, "Status": "running"}, "Config": map[string]any{"Image": ref, "Labels": labels}})
					} else if _, err := os.Stat(marker); err == nil {
						id := "sha256:new"
						if scenario == "wrong-image" {
							id = "sha256:unrelated"
						}
						configuredImage := "registry.test/nginx@" + digest
						if scenario == "literal-tag" {
							configuredImage = "registry.test/nginx:manual"
						}
						json.NewEncoder(w).Encode(map[string]any{"Id": "new-container", "Name": "/test-nginx", "Image": id, "State": map[string]any{"Running": true, "Status": "running"}, "Config": map[string]any{"Image": configuredImage}})
					} else {
						w.WriteHeader(404)
						fmt.Fprint(w, `{"message":"not found"}`)
					}
				case strings.HasSuffix(path, "/containers/old-container/stop"):
					steps = append(steps, "stop")
					w.WriteHeader(204)
				case strings.HasSuffix(path, "/containers/old-container") && r.Method == "DELETE":
					if r.URL.Query().Get("v") == "1" || r.URL.Query().Get("v") == "true" || r.URL.Query().Get("force") == "true" {
						t.Fatal("destructive flags enabled")
					}
					steps = append(steps, "remove-container")
					removed = true
					w.WriteHeader(204)
				case strings.Contains(path, "/images/") && strings.HasSuffix(path, "/json"):
					if scenario == "absent-tag" && !strings.Contains(path, "@sha256:") {
						w.WriteHeader(404)
						fmt.Fprint(w, `{"message":"not found"}`)
						break
					}
					id := "sha256:old"
					if strings.Contains(path, "@sha256:") {
						if !pulled {
							t.Fatal("digest inspected before pull")
						}
						id = "sha256:new"
					}
					fmt.Fprintf(w, `{"Id":%q,"Os":"linux","Architecture":"arm64"}`, id)
				case strings.HasSuffix(path, "/containers/json"):
					if scenario == "shared-image" {
						fmt.Fprint(w, `[{"Id":"other-environment","ImageID":"sha256:old"}]`)
					} else {
						fmt.Fprint(w, "[]")
					}
				case strings.Contains(path, "/images/") && r.Method == "DELETE":
					if r.URL.Query().Get("force") == "true" {
						t.Fatal("image deletion was forced")
					}
					if strings.HasSuffix(path, "/sha256:old") {
						t.Fatal("image deleted by ID rather than scoped tag")
					}
					steps = append(steps, "remove-image")
					fmt.Fprint(w, "[]")
				case strings.HasSuffix(path, "/info"):
					fmt.Fprint(w, `{"OSType":"linux","Architecture":"arm64"}`)
				case strings.HasSuffix(path, "/images/create"):
					steps = append(steps, "pull")
					if r.URL.Query().Get("tag") != digest {
						t.Fatalf("mutable tag was pulled: %s", r.URL)
					}
					if scenario == "pull-failure" || scenario == "manual-pull-failure" {
						w.WriteHeader(500)
						fmt.Fprint(w, `{"message":"pull failed"}`)
					} else {
						pulled = true
						fmt.Fprint(w, "{}")
					}
				case strings.HasSuffix(path, "/tag"):
					steps = append(steps, "tag")
					if r.URL.Query().Get("tag") != "test" {
						t.Fatalf("tag was not refreshed: %s", r.URL)
					}
					w.WriteHeader(201)
				default:
					t.Errorf("unexpected Docker request %s %s", r.Method, r.URL)
					w.WriteHeader(404)
				}
				return w.Result(), nil
			})
			cli, err := client.NewClientWithOpts(client.WithHost("http://docker.test"), client.WithVersion("1.41"), client.WithHTTPClient(&http.Client{Transport: transport}))
			if err != nil {
				t.Fatal(err)
			}
			previous := docker.Client
			docker.Client = cli
			t.Cleanup(func() { docker.Client = previous; cli.Close() })
			snapshot := Snapshot{Release: models.Release{ApplicationID: 3, Source: "registry", TriggerImage: ref, TriggerDigest: digest, Execution: models.DeploymentExecution{Mode: "command", Command: `printf '%s' "$DOCKERUN_IMAGE" > "$MARKER"`}, Images: stringJSON([]Image{{Reference: ref, Digest: digest}})}, Environment: models.Environment{ID: 4, ApplicationID: 3, ReplaceContainers: true, ContainerNames: []string{"test-nginx"}, Parameters: models.StringMap{"MARKER": marker}}}
			snapshot.Automatic = !strings.HasPrefix(scenario, "manual-")
			stored := stringJSON(snapshot.Release)
			actual, _, updated, err := RunDeployment(context.Background(), &snapshot, dir, "auth", func(stage, message string) {
				if stage == "deploy_script" && message == "开始执行脚本" {
					steps = append(steps, "script")
				}
			}, func() error {
				steps = append(steps, "checkpoint")
				if scenario == "checkpoint-failure" {
					return fmt.Errorf("journal failed")
				}
				return nil
			})
			if stringJSON(snapshot.Release) != stored {
				t.Fatal("historical release was modified")
			}
			switch scenario {
			case "replace", "shared-image", "literal-tag", "absent-tag", "manual-online":
				want := []string{"checkpoint", "stop", "remove-container", "remove-image", "pull", "tag", "script"}
				if scenario == "shared-image" || scenario == "absent-tag" {
					want = []string{"checkpoint", "stop", "remove-container", "pull", "tag", "script"}
				}
				if scenario == "manual-online" {
					want = []string{"checkpoint", "pull", "stop", "remove-container", "tag", "script"}
				}
				if err != nil || !updated || !reflect.DeepEqual(steps, want) {
					t.Fatalf("wrong execution order: %v %t %v", steps, updated, err)
				}
				imageName := ref
				if scenario == "literal-tag" {
					imageName = "registry.test/nginx:manual"
				}
				if len(snapshot.Containers) != 1 || snapshot.Containers[0].ImageID != "sha256:new" || snapshot.Containers[0].Image != imageName || !strings.Contains(actual, "sha256:new") {
					t.Fatalf("wrong actual image report: %+v %s", snapshot.Containers, actual)
				}
				value, _ := os.ReadFile(marker)
				if string(value) != "registry.test/nginx@"+digest {
					t.Fatalf("script received mutable reference: %s", value)
				}
			case "foreign-owner", "checkpoint-failure":
				if err == nil || updated || !reflect.DeepEqual(steps, []string{"checkpoint"}) {
					t.Fatalf("unsafe replacement: %v %t %v", steps, updated, err)
				}
			case "pull-failure":
				if err == nil || !updated || len(snapshot.Containers) != 1 || snapshot.Containers[0].State != "missing" {
					t.Fatalf("failure hid missing old container: %+v %v", snapshot, err)
				}
				if _, err := os.Stat(marker); !os.IsNotExist(err) {
					t.Fatal("script executed after failed pull")
				}
			case "wrong-image":
				if err == nil || !strings.Contains(err.Error(), "本次发布的镜像") {
					t.Fatalf("unrelated image was reported as deployed: %v", err)
				}
			case "manual-pull-failure":
				if err == nil || updated || removed || !reflect.DeepEqual(steps, []string{"checkpoint", "pull"}) {
					t.Fatalf("manual image failure interrupted old service: %v %t %v", steps, updated, err)
				}
			}
		})
	}
}

func TestAutomaticTagAndDigestRules(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := database.Init(config.Load().DatabasePath); err != nil {
		t.Fatal(err)
	}
	var admin models.User
	database.DB.First(&admin, 1)
	admin.HashPassword("test-password")
	database.DB.Save(&admin)
	app := models.Application{Name: "tag-rules", Definition: models.ApplicationDefinition{Execution: models.DeploymentExecution{Mode: "command", Command: `echo "$DOCKERUN_IMAGE"`}}}
	database.DB.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "test", TargetID: 1, RegistryID: 1, Repository: "nginx", TagPattern: "*", AutoDeploy: true, AllowedPublishers: "[]"}
	database.DB.Create(&env)
	digestA, digestB := "sha256:"+strings.Repeat("a", 64), "sha256:"+strings.Repeat("b", 64)
	push := func(key, tag, digest string, want int64) {
		t.Helper()
		event := models.ImageEvent{RegistryID: 1, Repository: "nginx", Reference: tag, Digest: digest, Key: key, Committed: true, UserID: 1}
		database.DB.Create(&event)
		processEvents(context.Background())
		var count int64
		database.DB.Model(&models.Deployment{}).Count(&count)
		if count != want {
			t.Fatalf("%s: got %d tasks, want %d", key, count, want)
		}
	}
	succeed := func() {
		var d models.Deployment
		database.DB.Order("id DESC").First(&d)
		database.DB.Model(&d).Update("status", "success")
		database.DB.Model(&env).Update("current_release_id", d.ReleaseID)
	}
	push("first", "test", digestA, 1)
	push("duplicate-in-flight", "test", digestA, 1)
	succeed()
	push("same-tag-same-content", "test", digestA, 1)
	push("different-tag-same-content", "prod", digestA, 2)
	succeed()
	push("same-tag-new-content", "prod", digestB, 3)
	succeed()
	push("return-to-previous-tag", "test", digestA, 4)
	push("switch-back-while-another-tag-is-queued", "prod", digestB, 5)
	push("duplicate-of-latest-queued-tag", "prod", digestB, 5)
}

func TestComposeTriggerTagChangesRenderedConfigurationEvenWithSameImageID(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	r := models.Release{Compose: "services:\n  web:\n    image: registry.test/nginx:test\n  db:\n    image: registry.test/db:v1\n", TriggerDigest: digest}
	images := []Image{{Reference: "registry.test/nginx:test", Digest: digest, ID: "sha256:web"}, {Reference: "registry.test/db:v1", ID: "sha256:db"}}
	render := func(tag string) string {
		t.Helper()
		r.TriggerImage = "registry.test/nginx:" + tag
		project, err := Render(context.Background(), r, models.Environment{ApplicationID: 1, ID: 2}, t.TempDir(), images)
		if err != nil {
			t.Fatal(err)
		}
		if project.Services["web"].Image != "sha256:web" || project.Services["db"].Labels["io.dockerun.image-trigger"] != "" {
			t.Fatal("trigger altered image identity or unrelated service")
		}
		return project.Services["web"].Labels["io.dockerun.image-trigger"]
	}
	first := render("test")
	if first == "" || first != render("test") || first == render("prod") {
		t.Fatal("different tags with identical contents must change Compose service configuration")
	}
}

func TestManagedReplacementValidatesWorkingDirectoryBeforeRemovingContainers(t *testing.T) {
	s := Snapshot{Release: models.Release{Execution: models.DeploymentExecution{Mode: "command", Command: "echo start", WorkDir: "missing"}}, Environment: models.Environment{ReplaceContainers: true, ContainerNames: []string{"test-nginx"}}}
	checkpoint := false
	_, _, updated, err := RunDeployment(context.Background(), &s, t.TempDir(), "", func(string, string) {}, func() error { checkpoint = true; return nil })
	if err == nil || updated || checkpoint {
		t.Fatalf("invalid working directory reached destructive stage: %v %t %t", err, updated, checkpoint)
	}
}

func TestEnqueueRejectsChangedContainerScopeBeforeCreatingSnapshot(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := database.Init(config.Load().DatabasePath); err != nil {
		t.Fatal(err)
	}
	app := models.Application{Name: "scope"}
	database.DB.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "test", TargetID: 1, ContainerNames: []string{"old-nginx"}, ReplaceContainers: true}
	database.DB.Create(&env)
	version := models.Release{ApplicationID: app.ID, Version: "v1", Execution: models.DeploymentExecution{Mode: "command", Command: "echo run"}}
	database.DB.Create(&version)
	updated := env
	updated.ContainerNames = []string{"new-nginx"}
	database.DB.Save(&updated)
	if _, err := Enqueue(env, version, 1, "changed-scope", false, 0); err == nil || !strings.Contains(err.Error(), "关联容器配置已变化") {
		t.Fatalf("stale container policy was accepted: %v", err)
	}
	var count int64
	database.DB.Model(&models.Deployment{}).Count(&count)
	if count != 0 {
		t.Fatal("stale request created a destructive deployment snapshot")
	}
}

func TestRuntimeImagesPersistActualIDsAndClearRemovedContainers(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := database.Init(config.Load().DatabasePath); err != nil {
		t.Fatal(err)
	}
	app := models.Application{Name: "runtime"}
	database.DB.Create(&app)
	env := models.Environment{ApplicationID: app.ID, Name: "test", TargetID: 1}
	database.DB.Create(&env)
	version := models.Release{ApplicationID: app.ID, Version: "v1", Execution: models.DeploymentExecution{Mode: "command", Command: "echo run"}}
	database.DB.Create(&version)
	snapshot := Snapshot{Release: version, Environment: env, Containers: []models.RuntimeContainer{{ID: "container-id", Name: "test-nginx", Image: "registry.test/nginx:test", ImageID: "sha256:actual", State: "running"}}}
	d := models.Deployment{EnvironmentID: env.ID, TargetID: 1, ReleaseID: version.ID, Snapshot: stringJSON(snapshot), Status: "running", IdempotencyKey: "runtime"}
	database.DB.Create(&d)
	success(&d, `{"container-id":"sha256:actual"}`, "unchecked")
	var state models.TargetState
	database.DB.Where("environment_id = ?", env.ID).First(&state)
	if len(state.Containers) != 1 || state.Containers[0].ImageID != "sha256:actual" || state.RuntimeState != "running" {
		t.Fatalf("actual image information not persisted: %+v", state)
	}
	snapshot.Containers = []models.RuntimeContainer{{Name: "test-nginx", State: "missing"}}
	d.Snapshot = stringJSON(snapshot)
	RuntimeFailure(&d)
	database.DB.First(&state, state.ID)
	if len(state.Containers) != 1 || state.Containers[0].State != "missing" || state.Containers[0].ImageID != "" || state.CurrentReleaseID != version.ID {
		t.Fatalf("removed old image still shown as running: %+v", state)
	}
}
