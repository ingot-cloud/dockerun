package release

import (
	"context"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"github.com/docker/docker/client"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestScriptValidationDoesNotExecute(t *testing.T) {
	file := filepath.Join(t.TempDir(), "should-not-exist")
	if err := ValidateScripts(models.DeploymentScripts{Before: "touch '" + file + "'"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("validation executed the script")
	}
	for _, script := range []string{"if then", "echo \x00", strings.Repeat("x", 256<<10+1)} {
		if ValidateScripts(models.DeploymentScripts{Deploy: script}) == nil {
			t.Fatal("invalid script accepted")
		}
	}
}

func TestScriptEnvironmentAndFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOCKERUN_TEST_SERVER_SECRET", "must-not-leak")
	s := Snapshot{Release: models.Release{ApplicationID: 3, Version: "v1"}, Environment: models.Environment{ID: 4, ApplicationID: 3, Name: "测试", Parameters: models.StringMap{"VALUE": "literal $value; $(false)", "WORK_DIR": "wrong"}}}
	var output strings.Builder
	var mu sync.Mutex
	log := func(stage, message string) { mu.Lock(); defer mu.Unlock(); output.WriteString(stage + ":" + message) }
	script := `test "$PWD" -ef "$WORK_DIR"
test "$COMPOSE_FILE" = "$WORK_DIR/resolved.json"
test "$COMPOSE_PROJECT_NAME" = dockerun-a3-e4
test "$DOCKERUN_ENVIRONMENT" = 测试
test "$DOCKERUN_VERSION" = v1
test -z "${DOCKERUN_TEST_SERVER_SECRET:-}"
printf '%s' "$VALUE" > value.txt
echo visible-output
echo visible-error >&2`
	if err := RunScript(context.Background(), script, "before_script", dir, s, log); err != nil {
		t.Fatal(err)
	}
	value, _ := os.ReadFile(filepath.Join(dir, "value.txt"))
	if string(value) != s.Environment.Parameters["VALUE"] {
		t.Fatal("environment values were interpolated or lost")
	}
	if !strings.Contains(output.String(), "visible-output") || !strings.Contains(output.String(), "visible-error") {
		t.Fatal("missing script logs")
	}
	for _, script := range []string{"false\ntouch forbidden", "false | true\ntouch forbidden"} {
		if RunScript(context.Background(), script, "deploy_script", dir, s, log) == nil {
			t.Fatal("script failure was ignored")
		}
		if _, err := os.Stat(filepath.Join(dir, "forbidden")); !os.IsNotExist(err) {
			t.Fatal("execution continued after a failed command")
		}
	}
}

func TestScriptCancellationStopsChildren(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- RunScript(ctx, "touch started\nsleep 1\ntouch forbidden", "before_script", dir, Snapshot{}, func(string, string) {})
	}()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "started")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("script did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled script succeeded")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled script hung")
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err := os.Stat(filepath.Join(dir, "forbidden")); !os.IsNotExist(err) {
		t.Fatal("script children survived cancellation")
	}
}

func TestScriptInheritanceAndSnapshot(t *testing.T) {
	r := models.Release{Scripts: models.DeploymentScripts{Before: "application"}}
	env := models.Environment{Scripts: models.DeploymentScripts{Before: "environment"}}
	if EffectiveScripts(r, env).Before != "application" {
		t.Fatal("application scripts were not inherited")
	}
	env.ScriptMode = "override"
	if EffectiveScripts(r, env).Before != "environment" {
		t.Fatal("environment override was ignored")
	}
	env.ScriptMode = "disabled"
	if HasScripts(EffectiveScripts(r, env)) {
		t.Fatal("disabled scripts were not skipped")
	}
}

type scriptDockerTransport func(*http.Request) (*http.Response, error)

func (f scriptDockerTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestApplyRunsScriptsInOrderOnTarget(t *testing.T) {
	dir := t.TempDir()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\necho deploy >> order\ntouch deployed\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	httpClient := &http.Client{Transport: scriptDockerTransport(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/containers/json") {
			if _, err := os.Stat(filepath.Join(dir, "deployed")); err == nil {
				w.Write([]byte(`[{"Id":"fixture","Labels":{"com.docker.compose.service":"app","com.docker.compose.project":"dockerun-a3-e4"}}]`))
			} else {
				w.Write([]byte(`[]`))
			}
		} else if strings.HasSuffix(r.URL.Path, "/containers/fixture/json") {
			w.Write([]byte(`{"Image":"sha256:fixture","State":{"Running":true,"Health":{"Status":"healthy"}}}`))
		} else {
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
	s := Snapshot{Release: models.Release{ApplicationID: 3, Compose: "services:\n  app:\n    image: registry.test/team/app:v1\n"}, Environment: models.Environment{ID: 4, ApplicationID: 3}, Images: []Image{{Reference: "registry.test/team/app:v1", ID: "sha256:fixture"}}, Scripts: models.DeploymentScripts{Before: "echo before >> order", After: "echo after >> order"}}
	actual, health, updated, err := Apply(context.Background(), s, dir, func(string, string) {}, nil)
	if err != nil || !updated || health != "healthy" || !strings.Contains(actual, "fixture") {
		t.Fatalf("apply: %s %s %t %v", actual, health, updated, err)
	}
	order, _ := os.ReadFile(filepath.Join(dir, "order"))
	if string(order) != "before\ndeploy\nafter\n" {
		t.Fatalf("wrong script order: %s", order)
	}
	// A custom deployment script replaces only the system startup command, while health is still verified.
	s.Scripts.Deploy = "echo custom >> order\ntouch deployed"
	s.Scripts.Before = ""
	s.Scripts.After = ""
	if _, _, _, err := Apply(context.Background(), s, dir, func(string, string) {}, nil); err != nil {
		t.Fatal(err)
	}
	order, _ = os.ReadFile(filepath.Join(dir, "order"))
	if !strings.HasSuffix(string(order), "custom\n") {
		t.Fatal("custom deployment script was ignored")
	}
	s.Release.Execution = models.DeploymentExecution{Mode: "compose", Command: "echo execution >> order\ntouch deployed", CheckCommand: "echo check >> order"}
	if _, _, _, err := Apply(context.Background(), s, dir, func(string, string) {}, nil); err != nil {
		t.Fatal(err)
	}
	order, _ = os.ReadFile(filepath.Join(dir, "order"))
	if !strings.HasSuffix(string(order), "execution\ncheck\n") {
		t.Fatal("Compose execution command or additional check was ignored")
	}
	s.Release.Execution = models.DeploymentExecution{}
	s.Scripts.Before = "exit 9"
	s.Scripts.Deploy = "touch forbidden"
	if _, _, updated, err := Apply(context.Background(), s, dir, func(string, string) {}, nil); err == nil || updated {
		t.Fatal("failed pre-deployment script did not stop deployment")
	}
	if _, err := os.Stat(filepath.Join(dir, "forbidden")); !os.IsNotExist(err) {
		t.Fatal("deployment continued after before-script failure")
	}
}
