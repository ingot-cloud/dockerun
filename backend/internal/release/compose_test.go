package release

import (
	"context"
	"strings"
	"testing"
)

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
