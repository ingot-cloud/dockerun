package handler

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestShippedTemplatesDoNotOfferRetiredScripts(t *testing.T) {
	t.Setenv("TEMPLATE_DIR", filepath.Join("..", "..", "..", "templates"))
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	GetTemplates(ctx)
	if response.Code != 200 || response.Body.String() != "[]" {
		t.Fatal("已移除的旧脚本仍被提供，或空列表格式错误", response.Body.String())
	}
}

func TestExternalLegacyTemplatesRemainAvailable(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "external-script")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"meta.json": `{"id":"external-script","name":"自定义历史脚本","main_script":"deploy.sh","files":[{"file_name":"app.conf","file_path":"app.conf","source":"app.conf"}]}`,
		"deploy.sh": "printf '历史脚本\\n'\n",
		"app.conf":  "enabled=true\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("TEMPLATE_DIR", root)
	items, err := loadTemplatesFromFS()
	if err != nil || len(items) != 1 {
		t.Fatalf("外部历史模板数量 %d，错误 %v", len(items), err)
	}
	if items[0].ID != "external-script" || items[0].MainScript == "" || len(items[0].Files) != 1 || items[0].Files[0].Content != "enabled=true\n" {
		t.Fatal("自定义历史脚本兼容能力丢失", items)
	}
}
