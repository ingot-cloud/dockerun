package handler

import (
	"dockerun/internal/config"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestDeploymentPreviewSeparatesCurrentAndNewImageMetadata(t *testing.T) {
	db := registryTestDB(t)
	t.Setenv("DATA_DIR", t.TempDir())
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Application{}, &models.Environment{}, &models.Release{}); err != nil {
		t.Fatal(err)
	}
	app := models.Application{Name: "预览验证"}
	if err := db.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	old := models.Release{ApplicationID: app.ID, Version: "v1", Compose: "services:\n  app:\n    image: nginx:old\n    ports: ['${PORT}:80']\n", Images: `[{"reference":"nginx:old","digest":"sha256:old"}]`}
	next := models.Release{ApplicationID: app.ID, Version: "v2", Compose: "services:\n  app:\n    image: nginx:new\n    ports: ['${PORT}:80']\n    environment:\n      PASSWORD: ${PASSWORD}\n", Images: `[{"reference":"nginx:new","digest":"sha256:new"}]`}
	for _, version := range []*models.Release{&old, &next} {
		if err := db.Create(version).Error; err != nil {
			t.Fatal(err)
		}
	}
	env := models.Environment{ApplicationID: app.ID, CurrentReleaseID: old.ID, Name: "生产", Parameters: models.StringMap{"PORT": "8081", "PASSWORD": "preview-secret-must-stay-hidden"}}
	if err := db.Create(&env).Error; err != nil {
		t.Fatal(err)
	}
	response := registryRequest(t, PreflightDeployment, 0, map[string]any{"environment_id": env.ID, "release_id": next.ID})
	var preview struct {
		Compose         string          `json:"compose"`
		PreviousCompose string          `json:"previous_compose"`
		Images          []release.Image `json:"images"`
		PreviousImages  []release.Image `json:"previous_images"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &preview) != nil {
		t.Fatal(response.Body.String())
	}
	if !json.Valid([]byte(preview.Compose)) || !strings.Contains(preview.Compose, "8081") || strings.Contains(response.Body.String(), env.Parameters["PASSWORD"]) {
		t.Fatal("解析后的配置或敏感参数展示错误")
	}
	if preview.PreviousCompose != old.Compose || len(preview.Images) != 1 || preview.Images[0].Reference != "nginx:new" || len(preview.PreviousImages) != 1 || preview.PreviousImages[0].Reference != "nginx:old" {
		t.Fatal("当前版本配置被新参数解析，或混入了本次发布的镜像身份")
	}
}
