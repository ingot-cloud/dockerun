package release

import (
	"context"
	"maps"
	"strings"
	"testing"
)

func TestParsePreservesParameters(t *testing.T) {
	for _, tc := range []struct {
		name       string
		parameters map[string]string
		content    string
		wantError  bool
	}{
		{name: "空参数", parameters: map[string]string{}},
		{name: "未提供参数"},
		{name: "模板参数", parameters: map[string]string{"PORT": "6379", "PASSWORD": "  literal$secret  "}},
		{name: "旧版模板预览参数", parameters: map[string]string{"COMPOSE_PROJECT_NAME": "dockerun-template-preview"}},
		{name: "解析失败", parameters: map[string]string{"PORT": "6379"}, content: "services:\n  app:\n    image: ${IMAGE:?请填写镜像}\n", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := maps.Clone(tc.parameters)
			content := tc.content
			if content == "" {
				content = "services:\n  app:\n    image: redis:8.2.0\n    container_name: ${COMPOSE_PROJECT_NAME}-redis\n"
			}
			project, err := Parse(context.Background(), content, t.TempDir(), "dockerun-a1-e1", tc.parameters)
			if (err != nil) != tc.wantError {
				t.Fatalf("解析错误不符合预期：%v", err)
			}
			if !maps.Equal(tc.parameters, before) {
				t.Fatal("Compose 解析改写了调用方的运行参数")
			}
			if err == nil {
				if project.Services["app"].ContainerName != "dockerun-a1-e1-redis" {
					t.Fatal("解析副本未使用当前部署项目名")
				}
				project.Environment["PASSWORD"] = "changed"
				if !maps.Equal(tc.parameters, before) {
					t.Fatal("解析结果仍与调用方共享运行参数")
				}
			}
		})
	}
}

func TestParseCustomContainerAndExistingNetwork(t *testing.T) {
	source := `services:
  redis:
    image: redis:8.2.0
    container_name: ${REDIS_CONTAINER_NAME}
    networks:
      dockerun_external_ingot_network:
        aliases: [cache]
        ipv4_address: 172.30.0.2
networks:
  dockerun_external_ingot_network:
    name: ingot-network
    external: true
    x-dockerun-managed: true
`
	project, err := Parse(context.Background(), source, t.TempDir(), "test", map[string]string{"REDIS_CONTAINER_NAME": "ingot-redis"})
	if err != nil {
		t.Fatal(err)
	}
	service := project.Services["redis"]
	connection := service.Networks["dockerun_external_ingot_network"]
	network := project.Networks["dockerun_external_ingot_network"]
	if service.ContainerName != "ingot-redis" || !network.External || network.Name != "ingot-network" || connection == nil || connection.Ipv4Address != "172.30.0.2" || len(connection.Aliases) != 1 || connection.Aliases[0] != "cache" {
		t.Fatal("自定义容器名称或已有网络的连接配置未保留")
	}
}

func TestRejectImplicitMaterials(t *testing.T) {
	for _, s := range []string{"include: https://example.test/compose.yaml\nservices: {}", "services:\n  app:\n    build: .", "services:\n  app:\n    image: nginx\n    env_file: /etc/passwd", "services:\n  app:\n    image: nginx\n    extends: {file: /etc/compose.yaml, service: app}"} {
		if _, err := Parse(context.Background(), s, t.TempDir(), "test", nil); err == nil {
			t.Fatalf("accepted implicit material: %s", s)
		}
	}
}
func TestRequiredEnvironmentAndSafePaths(t *testing.T) {
	src := "services:\n  app:\n    image: nginx:1\n    environment:\n      REQUIRED: ${REQUIRED:?missing}\n"
	if _, err := Parse(context.Background(), src, t.TempDir(), "test", nil); err == nil {
		t.Fatal("missing parameter accepted")
	}
	p, err := Parse(context.Background(), src, t.TempDir(), "test", map[string]string{"REQUIRED": "value"})
	if err != nil {
		t.Fatal(err)
	}
	out, _ := Encode(p)
	if !strings.Contains(out, `"pull_policy":"never"`) {
		t.Fatal(out)
	}
	for _, path := range []string{"../x", "/x", "a/../../x", "a\\b"} {
		if _, err := SafePath(t.TempDir(), path); err == nil {
			t.Fatalf("unsafe path %s", path)
		}
	}
}
func TestDockerErrorFrames(t *testing.T) {
	if err := Consume(strings.NewReader("{\"status\":\"loading\"}\n{\"errorDetail\":{\"message\":\"no space left\"}}"), nil); err == nil {
		t.Fatal("Docker stream error was ignored")
	}
}

func TestPromotionFreezesImageButKeepsEnvironment(t *testing.T) {
	ctx := context.Background()
	src := "services:\n  app:\n    image: ${IMAGE}\n    environment:\n      PASSWORD: ${PASSWORD}\n    command: [sh, -c, 'echo $$PASSWORD']\n"
	p, err := Parse(ctx, src, t.TempDir(), "test", map[string]string{"IMAGE": "registry.example/team/app:v1", "PASSWORD": "test"})
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := FreezeImages(src, p)
	if err != nil {
		t.Fatal(err)
	}
	promoted, err := Parse(ctx, frozen, t.TempDir(), "prod", map[string]string{"IMAGE": "registry.example/team/app:v2", "PASSWORD": "literal$secret"})
	if err != nil {
		t.Fatal(err)
	}
	if promoted.Services["app"].Image != "registry.example/team/app:v1" {
		t.Fatal("promotion changed image identity")
	}
	encoded, err := EncodeExecution(promoted)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(encoded, "literal$$secret") || !strings.Contains(encoded, "echo $$PASSWORD") {
		t.Fatal("literal dollars were not protected from Compose re-interpolation:", encoded)
	}
}
