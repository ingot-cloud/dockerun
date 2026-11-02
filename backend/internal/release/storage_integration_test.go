package release_test

import (
	"context"
	"dockerun/internal/config"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

// 在隔离容器内运行；DATA_DIR 必须已挂载到测试专用卷或目录。
func TestDockerStoragePathDeployment(t *testing.T) {
	if os.Getenv("DOCKERUN_STORAGE_TESTS") != "1" {
		t.Skip("需使用测试专用数据挂载设置 DOCKERUN_STORAGE_TESTS=1")
	}
	if os.Getenv("DATA_VOLUME") == "" && os.Getenv("HOST_DATA_DIR") == "" {
		t.Fatal("测试需要显式配置卷或宿主路径映射")
	}
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := docker.Init(); err != nil {
		t.Fatal(err)
	}
	defer docker.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dir, err := os.MkdirTemp(filepath.Join(config.Load().DataDir, "jobs"), "storage-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	fixture := "nginx:1.28-alpine"
	inspect, _, err := docker.GetClient().ImageInspectWithRaw(ctx, fixture)
	if err != nil {
		t.Fatal(err)
	}
	env := models.Environment{ID: uint(time.Now().UnixNano()%100000000) + 1000, ApplicationID: 987655, Name: "storage-test"}
	hostDir, err := docker.HostDataPath(dir)
	if err != nil || hostDir == dir {
		t.Fatalf("未启用路径映射: %s %v", hostDir, err)
	}
	// 相对配置、旧宿主绝对路径和 local bind 驱动同时覆盖。
	compose := fmt.Sprintf(`services:
  fixture:
    image: %s
    command: [sh, -c, 'while true; do sleep 30; done']
    configs:
      - source: fixture
        target: /fixture.conf
    volumes:
      - %s/configs/fixture.conf:/legacy.conf:ro
      - files:/files:ro
    healthcheck:
      test: [CMD-SHELL, 'test "$$(cat /fixture.conf)" = storage-ok && cmp /fixture.conf /legacy.conf && cmp /fixture.conf /files/fixture.conf']
      interval: 1s
      timeout: 1s
      retries: 5
configs:
  fixture:
    file: ./configs/fixture.conf
volumes:
  files:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: %s/configs
`, fixture, hostDir, dir)
	r := models.Release{Compose: compose, Configs: `{"configs/fixture.conf":"storage-ok"}`}
	p, err := release.Render(ctx, r, env, dir, []release.Image{{Reference: fixture, ID: inspect.ID}})
	if err != nil {
		t.Fatal(err)
	}
	for _, mount := range p.Services["fixture"].Volumes {
		if mount.Target == "/legacy.conf" && mount.Source != filepath.Join(dir, "configs/fixture.conf") {
			t.Fatal("旧宿主路径未转换成容器路径")
		}
	}
	defer func() {
		cmd := exec.Command("docker", "compose", "-p", p.Name, "-f", filepath.Join(dir, "resolved.json"), "down", "--volumes", "--remove-orphans")
		cmd.Env = docker.CommandEnvironment(os.Environ())
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("清理隔离项目失败: %s %v", out, err)
		}
	}()
	if err := release.Preflight(ctx, p); err != nil {
		t.Fatal(err)
	}
	log := func(s string) { t.Log(strings.TrimSpace(s)) }
	if err := release.Up(ctx, p, dir, log); err != nil {
		t.Fatal(err)
	}
	if _, health, err := release.Actual(ctx, p); err != nil || health != "healthy" {
		t.Fatalf("真实配置挂载检查失败: %s %v", health, err)
	}
	containerID := func() string {
		items, err := docker.GetClient().ContainerList(ctx, container.ListOptions{Filters: filters.NewArgs(filters.Arg("label", "com.docker.compose.project="+p.Name))})
		if err != nil || len(items) != 1 {
			t.Fatalf("隔离项目容器异常: %v %v", items, err)
		}
		return items[0].ID
	}
	id := containerID()
	if err := release.Up(ctx, p, dir, log); err != nil {
		t.Fatal(err)
	}
	if containerID() != id {
		t.Fatal("未变更配置被重复重建")
	}
	env.Parameters = models.StringMap{"FIXTURE_IMAGE": fixture, "FIXTURE_CONTAINER": id}
	script := `test -f "$WORK_DIR/configs/fixture.conf"
test "$HOST_WORK_DIR" != "$WORK_DIR"
test "$(docker run --rm --pull never --network none --entrypoint cat -v "$WORK_DIR/configs/fixture.conf:/config:ro" "$FIXTURE_IMAGE" /config)" = storage-ok
test "$(docker run --rm --pull never --network none --entrypoint cat --mount "type=bind,source=$WORK_DIR/configs/fixture.conf,target=/config,readonly" "$FIXTURE_IMAGE" /config)" = storage-ok
test "$(docker exec "$FIXTURE_CONTAINER" cat /fixture.conf)" = storage-ok
`
	if err := release.RunScript(ctx, script, "deploy_script", dir, release.Snapshot{Release: r, Environment: env}, func(stage, msg string) { t.Log(stage, msg) }); err != nil {
		t.Fatal(err)
	}
}
