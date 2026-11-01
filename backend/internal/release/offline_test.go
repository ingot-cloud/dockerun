package release

import (
	"context"
	"crypto/sha256"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"encoding/hex"
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

func TestOfflineCommandReplacementAndHistoricalImage(t *testing.T) {
	for _, scenario := range []string{"first-release", "new-release", "historical-version", "missing-image"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			marker := filepath.Join(dir, "started-image")
			steps := []string{}
			id := "sha256:" + strings.Repeat("a", 64)
			ref := "registry.test/app:latest"
			removed := scenario == "first-release"
			transport := scriptDockerTransport(func(r *http.Request) (*http.Response, error) {
				w := httptest.NewRecorder()
				w.Header().Set("Content-Type", "application/json")
				switch path := r.URL.Path; {
				case strings.HasSuffix(path, "/info"):
					fmt.Fprint(w, `{"OSType":"linux","Architecture":"amd64"}`)
				case strings.Contains(path, "/images/") && strings.HasSuffix(path, "/json"):
					steps = append(steps, "verify-image")
					if scenario == "missing-image" {
						w.WriteHeader(404)
						fmt.Fprint(w, `{"message":"missing offline image"}`)
					} else {
						fmt.Fprintf(w, `{"Id":%q,"Os":"linux","Architecture":"amd64"}`, id)
					}
				case strings.HasSuffix(path, "/containers/app/json"):
					if _, err := os.Stat(marker); err == nil {
						json.NewEncoder(w).Encode(map[string]any{"Id": "new-container", "Name": "/app", "Image": id, "Config": map[string]any{"Image": id}, "State": map[string]any{"Running": true, "Status": "running"}})
					} else if !removed {
						json.NewEncoder(w).Encode(map[string]any{"Id": "old-container", "Name": "/app", "Image": "sha256:old", "Config": map[string]any{"Image": ref}, "State": map[string]any{"Running": true, "Status": "running"}})
					} else {
						w.WriteHeader(404)
						fmt.Fprint(w, `{"message":"not found"}`)
					}
				case strings.HasSuffix(path, "/containers/old-container/stop"):
					steps = append(steps, "stop")
					w.WriteHeader(204)
				case strings.HasSuffix(path, "/containers/old-container") && r.Method == "DELETE":
					steps = append(steps, "remove-container")
					removed = true
					w.WriteHeader(204)
				case strings.HasSuffix(path, "/tag"):
					steps = append(steps, "restore-version-tag")
					w.WriteHeader(201)
				default:
					t.Errorf("offline release attempted unexpected Docker operation: %s %s", r.Method, r.URL)
					w.WriteHeader(500)
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
			r := models.Release{Source: "local", BundleID: 1, Platform: "linux/amd64", TriggerImage: ref, Execution: models.DeploymentExecution{Mode: "command", Command: `printf '%s' "$DOCKERUN_IMAGE" > "$MARKER"`}, Images: stringJSON([]Image{{Reference: ref, ID: id, Platform: "linux/amd64"}})}
			if scenario == "historical-version" {
				// 历史版本仍用自己的 Image ID，环境参数不能覆盖它。
				r.Version = "old-version"
			}
			s := Snapshot{Release: r, Environment: models.Environment{ID: 1, ApplicationID: 1, ReplaceContainers: true, ContainerNames: []string{"app"}, Parameters: models.StringMap{"MARKER": marker, "DOCKERUN_IMAGE": "unrelated:latest"}}}
			_, _, updated, err := RunDeployment(context.Background(), &s, dir, "", func(stage, message string) {
				if stage == "deploy_script" && message == "开始执行脚本" {
					steps = append(steps, "script")
				}
			}, nil)
			if scenario == "missing-image" {
				if err == nil || updated || removed || !reflect.DeepEqual(steps, []string{"verify-image"}) {
					t.Fatalf("missing image interrupted old service: %v %t %v", steps, updated, err)
				}
				return
			}
			want := []string{"verify-image", "stop", "remove-container", "restore-version-tag", "script"}
			if scenario == "first-release" {
				want = []string{"verify-image", "restore-version-tag", "script"}
			}
			value, _ := os.ReadFile(marker)
			if err != nil || !updated || string(value) != id || !reflect.DeepEqual(steps, want) || len(s.Containers) != 1 || s.Containers[0].ImageID != id {
				t.Fatalf("offline command failed: %v %t %v %q %+v", steps, updated, err, value, s.Containers)
			}
		})
	}
}

func TestOfflineArchiveChecksumAndDockerErrorFrames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.tar")
	data := []byte("archive-fixture")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	loaded := 0
	cli, err := client.NewClientWithOpts(client.WithHost("http://docker.test"), client.WithVersion("1.41"), client.WithHTTPClient(&http.Client{Transport: scriptDockerTransport(func(r *http.Request) (*http.Response, error) {
		loaded++
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"errorDetail":{"message":"failed to load image"}}`)
		return w.Result(), nil
	})}))
	if err != nil {
		t.Fatal(err)
	}
	previous := docker.Client
	docker.Client = cli
	t.Cleanup(func() { docker.Client = previous; cli.Close() })
	if err := LoadImageArchive(context.Background(), path, strings.Repeat("0", 64), nil); err == nil || loaded != 0 {
		t.Fatal("corrupt archive reached Docker", err, loaded)
	}
	if err := LoadImageArchive(context.Background(), path, hex.EncodeToString(hash[:]), nil); err == nil || !strings.Contains(err.Error(), "failed to load") || loaded != 1 {
		t.Fatal("Docker error frame was ignored", err, loaded)
	}
}
