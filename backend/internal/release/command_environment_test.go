package release

import (
	"context"
	"dockerun/internal/models"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
)

func TestDeploymentCommandsWithMinimalPath(t *testing.T) {
	home, dir := t.TempDir(), t.TempDir()
	bin := filepath.Join(home, ".docker", "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	// 使用独立命令名，防止开发机已安装 Docker 掩盖路径补全失效。
	name := "dockerun-test-cli"
	if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nprintf '%s\\n' \"$*\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	for _, path := range []string{"/usr/bin:/bin", ""} {
		t.Run(path, func(t *testing.T) {
			t.Setenv("PATH", path)
			var output strings.Builder
			if err := RunScript(context.Background(), name+" run nginx:test", "deploy_script", dir, Snapshot{}, func(_, message string) { output.WriteString(message) }); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), "run nginx:test") {
				t.Fatalf("command was not executed: %s", output.String())
			}
			if got := commandExecutable(name); got != filepath.Join(bin, name) {
				t.Fatalf("executable lookup differs from script PATH: %s", got)
			}
		})
	}
}

func TestDeploymentPathPreservesOverridesAndFiltersEnvironment(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" > arguments\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+bin)
	t.Setenv("DOCKER_HOST", "unix:///test/docker.sock")
	t.Setenv("DOCKERUN_TEST_SERVER_SECRET", "must-not-leak")
	if got := commandExecutable("docker"); got != filepath.Join(bin, "docker") {
		t.Fatalf("inherited PATH priority lost: %s", got)
	}
	paths := filepath.SplitList(commandPath())
	if paths[0] != bin || strings.Count(commandPath(), bin) != 1 {
		t.Fatalf("inherited path changed or duplicated: %v", paths)
	}
	dir := t.TempDir()
	if err := Up(context.Background(), &types.Project{Name: "path-test", Services: types.Services{"app": {Image: "nginx:test"}}}, dir, func(string) {}); err != nil {
		t.Fatal(err)
	}
	arguments, err := os.ReadFile(filepath.Join(dir, "arguments"))
	if err != nil || !strings.HasPrefix(string(arguments), "compose --env-file ") {
		t.Fatalf("Compose used a different executable: %q %v", arguments, err)
	}
	script := `test "$DOCKER_HOST" = unix:///test/docker.sock
test -z "${DOCKERUN_TEST_SERVER_SECRET:-}"
test "$PATH" = /custom/bin`
	s := Snapshot{Environment: models.Environment{Parameters: models.StringMap{"PATH": "/custom/bin"}}}
	if err := RunScript(context.Background(), script, "deploy_script", dir, s, func(string, string) {}); err != nil {
		t.Fatal(err)
	}
}
