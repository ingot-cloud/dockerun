package release_test

import (
	"context"
	"dockerun/internal/bundle"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDockerLocalAndOfflineDeployment(t *testing.T) {
	if os.Getenv("DOCKERUN_DOCKER_TESTS") != "1" {
		t.Skip("set DOCKERUN_DOCKER_TESTS=1 with an isolated Docker fixture")
	}
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := database.Init(config.Load().DatabasePath); err != nil {
		t.Fatal(err)
	}
	if err := docker.Init(); err != nil {
		t.Fatal(err)
	}
	log := func(s string) { t.Log(s) }
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	platform, err := release.Platform(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fixture := "nginx:1.28-alpine"
	inspect, _, err := docker.GetClient().ImageInspectWithRaw(ctx, fixture)
	if err != nil {
		t.Fatal(err)
	}
	compose := "services:\n  fixture:\n    image: " + fixture + "\n    command: [sh, -c, 'while true; do sleep 30; done']\n    healthcheck:\n      test: [CMD, 'true']\n      interval: 1s\n      timeout: 1s\n      retries: 5\n"
	env := models.Environment{ID: uint(time.Now().UnixNano()%1000000) + 1000, ApplicationID: 987654, Name: "isolated-test", TargetID: 1}
	project := release.ProjectName(env)
	t.Cleanup(func() {
		c := exec.Command("docker", "compose", "-p", project, "-f", filepath.Join(dir, "resolved.json"), "down", "--remove-orphans")
		c.Env = os.Environ()
		out, err := c.CombinedOutput()
		if err != nil {
			t.Logf("fixture cleanup: %s %v", out, err)
		}
	})
	images, _ := json.Marshal([]release.Image{{Reference: fixture, ID: inspect.ID, Platform: platform}})
	r := models.Release{Source: "local", Platform: platform, Images: string(images), Compose: compose}
	prepared, err := release.Prepare(ctx, &r, log)
	if err != nil {
		t.Fatal(err)
	}
	p, err := release.Render(ctx, r, env, dir, prepared)
	if err != nil {
		t.Fatal(err)
	}
	if err = release.Preflight(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err = release.Up(ctx, p, dir, log); err != nil {
		t.Fatal(err)
	}
	actual, health, err := release.Actual(ctx, p)
	if err != nil || health != "healthy" || !strings.Contains(actual, inspect.ID) {
		t.Fatalf("actual %s health %s err %v", actual, health, err)
	}
	// Failed preparation must leave the already running service untouched.
	bad := r
	bad.Images = `[{"reference":"dockerun-test/nonexistent:missing","id":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}]`
	if _, err := release.Prepare(ctx, &bad, nil); err == nil {
		t.Fatal("missing image unexpectedly passed")
	}
	if _, _, err := release.Actual(ctx, p); err != nil {
		t.Fatalf("old service was affected by image preparation failure: %v", err)
	}
	// Produce a complete release, verify checksums, load its images and render with local IDs.
	source := filepath.Join(dir, "compose.yaml")
	os.WriteFile(source, []byte(compose), 0600)
	archive := filepath.Join(dir, "fixture-release.tar.gz")
	if err = bundle.Pack(ctx, source, archive, bundle.Manifest{Application: "isolated-test", Version: "v1", Platform: platform}); err != nil {
		t.Fatal(err)
	}
	materials := filepath.Join(dir, "materials")
	os.MkdirAll(materials, 0700)
	m, err := bundle.Extract(archive, materials)
	if err != nil {
		t.Fatal(err)
	}
	if err = bundle.Load(ctx, materials, *m, log); err != nil {
		t.Fatal(err)
	}
	r.Source = "bundle"
	b, _ := json.Marshal(m.Images)
	r.Images = string(b)
	prepared, err = release.Prepare(ctx, &r, nil)
	if err != nil {
		t.Fatal(err)
	}
	p, err = release.Render(ctx, r, env, dir, prepared)
	if err != nil {
		t.Fatal(err)
	}
	if err = release.Up(ctx, p, dir, log); err != nil {
		t.Fatal(err)
	}
	if _, _, err = release.Actual(ctx, p); err != nil {
		t.Fatal(err)
	}
	t.Log(fmt.Sprintf("verified %s, offline bundle SHA and image identity", platform))
}
