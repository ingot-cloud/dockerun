package release

import (
	"context"
	"dockerun/internal/models"
	"github.com/compose-spec/compose-go/v2/types"
	dockerTypes "github.com/docker/docker/api/types"
	"testing"
)

func TestComposeProjectNameAppliesToRenderAndScripts(t *testing.T) {
	for _, name := range []string{"", "redis-test"} {
		env := models.Environment{ID: 4, ApplicationID: 3, ComposeProjectName: name, Parameters: models.StringMap{"COMPOSE_PROJECT_NAME": "ignored"}}
		want := name
		if want == "" {
			want = "dockerun-a3-e4"
		}
		version := models.Release{Compose: "name: ignored\nservices:\n  redis:\n    image: redis:8.2.0\n    environment:\n      PROJECT: ${COMPOSE_PROJECT_NAME}\n"}
		project, err := Render(context.Background(), version, env, t.TempDir(), []Image{{Reference: "redis:8.2.0", ID: "sha256:fixed"}})
		if err != nil {
			t.Fatal(err)
		}
		service := project.Services["redis"]
		if project.Name != want || service.Environment["PROJECT"] == nil || *service.Environment["PROJECT"] != want || service.Labels["io.dockerun.environment"] != "4" {
			t.Fatal("未按环境名称解析或缺少资源归属")
		}
		if err := RunScript(context.Background(), `test "$COMPOSE_PROJECT_NAME" = `+want, "before_script", t.TempDir(), Snapshot{Environment: env}, func(string, string) {}); err != nil {
			t.Fatal(err)
		}
		if env.Parameters["COMPOSE_PROJECT_NAME"] != "ignored" {
			t.Fatal("解析修改了原始参数")
		}
	}
}

func TestCustomProjectCannotTakeOverExistingHostProject(t *testing.T) {
	service := types.ServiceConfig{Labels: types.Labels{"io.dockerun.application": "3", "io.dockerun.environment": "4"}}
	for _, tc := range []struct {
		name, app, env string
		allowed        bool
	}{
		{"dockerun-a3-e4", "", "", true},
		{"redis-prod", "3", "4", true},
		{"redis-prod", "", "", false},
		{"redis-prod", "3", "5", false},
		{"dockerun-a3-e4", "2", "4", false},
	} {
		project := &types.Project{Name: tc.name, Services: types.Services{"redis": service}}
		existing := []dockerTypes.Container{{Labels: map[string]string{"com.docker.compose.project": tc.name, "io.dockerun.application": tc.app, "io.dockerun.environment": tc.env}}}
		if err := checkProjectOwnership(project, existing); (err == nil) != tc.allowed {
			t.Fatalf("项目归属判断错误：%+v %v", tc, err)
		}
	}
}
