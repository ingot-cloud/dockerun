package release

import (
	"context"
	"dockerun/internal/models"
	"strings"
	"testing"
)

func TestRenderPreservesReadableImageReferenceWithoutChangingIdentity(t *testing.T) {
	id := "sha256:" + strings.Repeat("a", 64)
	digest := "sha256:" + strings.Repeat("b", 64)
	for _, tc := range []struct{ trigger, want string }{
		{"", "registry.test/redis:8.2.0"},
		{"registry.test/redis:stable", "registry.test/redis:stable"},
		{"registry.test/another:stable", "registry.test/redis:8.2.0"},
	} {
		r := models.Release{Compose: "services:\n  redis:\n    image: registry.test/redis:8.2.0\n", TriggerImage: tc.trigger, TriggerDigest: digest}
		project, err := Render(context.Background(), r, models.Environment{ApplicationID: 1, ID: 2}, t.TempDir(), []Image{{Reference: "registry.test/redis:8.2.0", ID: id, Digest: digest}})
		if err != nil {
			t.Fatal(err)
		}
		service := project.Services["redis"]
		if service.Image != id || service.Labels[ImageReferenceLabel] != tc.want || service.Labels[ImageIDLabel] != id {
			t.Fatal("可读名称丢失或改变了固定镜像身份", tc.trigger)
		}
	}
	for _, tc := range []struct{ name, expected, actual, want string }{
		{"redis:8.2.0", id, id, "redis:8.2.0"},
		{"redis:8.2.0", "sha256:old", id, ""},
		{"redis:8.2.0", "", "", ""},
		{id, id, id, ""},
		{"not a valid image", id, id, ""},
	} {
		if got := VerifiedImageReference(tc.name, tc.expected, tc.actual); got != tc.want {
			t.Fatal("未核对身份就展示了镜像名称", got)
		}
	}
}
