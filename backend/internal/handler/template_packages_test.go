package handler

import (
	"bytes"
	"context"
	"dockerun/internal/apptemplate"
	"dockerun/internal/models"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func templatePackageDB(t *testing.T) *gorm.DB {
	db := registryTestDB(t)
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("TEMPLATE_DIR", t.TempDir())
	if err := db.AutoMigrate(&models.ApplicationTemplate{}, &models.AuditEvent{}, &models.Application{}, &models.Environment{}, &models.Target{}, &models.Release{}); err != nil {
		t.Fatal(err)
	}
	return db
}
func portableTemplateFixture() apptemplate.Template {
	return apptemplate.Template{ID: "team-service", Key: "team-service", Name: "团队服务", Version: "1.0.0", Category: "business", Definition: models.ApplicationDefinition{Compose: "services:\n  app:\n    image: ${IMAGE}\n", Topology: "replicated", Parameters: []models.ApplicationParameter{{Key: "IMAGE", Label: "应用镜像", Type: "image", Default: "nginx:1.28", Required: true}}}, Attachments: models.StringMap{"README.md": "使用说明", "LICENSE": "许可内容"}}
}
func previewPackage(t *testing.T, items ...apptemplate.Template) map[string]any {
	t.Helper()
	data, err := apptemplate.WritePackage(context.Background(), items)
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	f, err := w.CreateFormFile("file", "templates.zip")
	if err != nil {
		t.Fatal(err)
	}
	f.Write(data)
	w.Close()
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Set("user_id", uint(1))
	ctx.Request = httptest.NewRequest("POST", "/managed-templates/imports", &body)
	ctx.Request.Header.Set("Content-Type", w.FormDataContentType())
	PreviewTemplateImport(ctx)
	if response.Code != 200 {
		t.Fatal("预览失败", response.Code, response.Body.String())
	}
	var result map[string]any
	if err = json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func packageRequest(t *testing.T, handler gin.HandlerFunc, token string, user uint, payload any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Set("user_id", user)
	ctx.Params = gin.Params{{Key: "import", Value: token}}
	ctx.Request = httptest.NewRequest("POST", "/", bytes.NewReader(b))
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler(ctx)
	ctx.Writer.WriteHeaderNow()
	return response
}
func packageChoice(index int, action string) map[string]any {
	return map[string]any{"index": index, "action": action}
}
func TestTemplateImportPreviewCommitAndReExport(t *testing.T) {
	db := templatePackageDB(t)
	item := portableTemplateFixture()
	preview := previewPackage(t, item)
	token := preview["id"].(string)
	var count int64
	db.Model(&models.ApplicationTemplate{}).Count(&count)
	if count != 0 {
		t.Fatal("预览提前写入了模板")
	}
	payload := map[string]any{"items": []any{packageChoice(0, "import")}}
	if response := packageRequest(t, CommitTemplateImport, token, 2, payload); response.Code == 200 {
		t.Fatal("其他用户提交了预览")
	}
	response := packageRequest(t, CommitTemplateImport, token, 1, payload)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	var result templateImportResult
	json.Unmarshal(response.Body.Bytes(), &result)
	if result.Imported != 1 || result.Skipped != 0 {
		t.Fatal(result)
	}
	retry := packageRequest(t, CommitTemplateImport, token, 1, payload)
	if retry.Body.String() != response.Body.String() {
		t.Fatal("重试响应不一致")
	}
	var row models.ApplicationTemplate
	db.First(&row)
	if row.TemplateKey != item.Key || row.Source != "imported" || row.Attachments["LICENSE"] != "许可内容" {
		t.Fatal("导入内容丢失", row)
	}
	db.Model(&models.AuditEvent{}).Where("action = ?", "application.template.import").Count(&count)
	if count != 1 {
		t.Fatal("导入审计重复", count)
	}
	items, err := applicationTemplates()
	if err != nil || len(items) != 1 || items[0].ID == items[0].Key || items[0].Source != "imported" {
		t.Fatal("模板管理列表不正确", items, err)
	}
	before, _ := apptemplate.Fingerprint(item)
	after, _ := apptemplate.Fingerprint(items[0])
	if before != after {
		t.Fatal("存储后模板内容发生变化")
	}
	export := packageRequest(t, ExportTemplatePackage, "", 1, map[string]any{"ids": []string{items[0].ID}})
	if export.Code != 200 || !strings.Contains(export.Header().Get("Content-Disposition"), "attachment") {
		t.Fatal(export.Body.String())
	}
	decoded, err := apptemplate.ReadPackage(context.Background(), export.Body.Bytes())
	if err != nil || len(decoded) != 1 {
		t.Fatal(err)
	}
	hash, _ := apptemplate.Fingerprint(decoded[0])
	if hash != before {
		t.Fatal("跨实例下载包内容变化")
	}
	second := previewPackage(t, item)
	entries := second["entries"].([]any)
	if entries[0].(map[string]any)["status"] != "identical" {
		t.Fatal("重复包未识别", entries)
	}
	response = packageRequest(t, CommitTemplateImport, second["id"].(string), 1, payload)
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"skipped":1`) {
		t.Fatal(response.Body.String())
	}
	db.Model(&models.ApplicationTemplate{}).Count(&count)
	if count != 1 {
		t.Fatal("重复导入产生新模板")
	}
}
func TestTemplateImportsConflictRenameAndAreAtomic(t *testing.T) {
	db := templatePackageDB(t)
	item := portableTemplateFixture()
	p := previewPackage(t, item)
	response := packageRequest(t, CommitTemplateImport, p["id"].(string), 1, map[string]any{"items": []any{packageChoice(0, "import")}})
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	fresh := item
	fresh.Key = "new-service"
	fresh.ID = fresh.Key
	changed := item
	changed.Description = "与原版本不同的内容"
	preview := previewPackage(t, fresh, changed)
	entries := preview["entries"].([]any)
	choices := []any{}
	for _, entry := range entries {
		e := entry.(map[string]any)
		choices = append(choices, packageChoice(int(e["index"].(float64)), "import"))
	}
	response = packageRequest(t, CommitTemplateImport, preview["id"].(string), 1, map[string]any{"items": choices})
	if response.Code != 409 {
		t.Fatal("内容冲突没有拒绝", response.Body.String())
	}
	var count int64
	db.Model(&models.ApplicationTemplate{}).Count(&count)
	if count != 1 {
		t.Fatal("冲突时部分模板已写入")
	}
	for _, entry := range entries {
		e := entry.(map[string]any)
		if e["key"] == item.Key {
			choices[int(e["index"].(float64))] = map[string]any{"index": int(e["index"].(float64)), "action": "rename", "key": "team-service-copy", "name": "团队服务副本"}
		}
	}
	response = packageRequest(t, CommitTemplateImport, preview["id"].(string), 1, map[string]any{"items": choices})
	if response.Code != 200 {
		t.Fatal("另存失败", response.Body.String())
	}
	db.Model(&models.ApplicationTemplate{}).Count(&count)
	if count != 3 {
		t.Fatal(count)
	}
}
func TestImportRechecksConflictsExpiryAndMaterialIntegrity(t *testing.T) {
	db := templatePackageDB(t)
	item := portableTemplateFixture()
	first, second := previewPackage(t, item), previewPackage(t, item)
	payload := map[string]any{"items": []any{packageChoice(0, "import")}}
	response := packageRequest(t, CommitTemplateImport, first["id"].(string), 1, payload)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	db.Model(&models.ApplicationTemplate{}).Where("template_key = ?", item.Key).Update("description", "并发更新")
	response = packageRequest(t, CommitTemplateImport, second["id"].(string), 1, payload)
	if response.Code != 409 {
		t.Fatal("没有重新检查冲突", response.Body.String())
	}
	preview := previewPackage(t, item)
	token := preview["id"].(string)
	dir, _ := templateImportDir(token)
	if err := os.WriteFile(filepath.Join(dir, "package.zip"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	response = packageRequest(t, CommitTemplateImport, token, 1, payload)
	if response.Code == 200 {
		t.Fatal("被替换的材料仍能提交")
	}
	_, state, err := loadTemplateImport(token, 1)
	if err != nil {
		t.Fatal(err)
	}
	state.ExpiresAt = state.ExpiresAt.Add(-2 * templateImportTTL)
	saveTemplateImport(dir, state)
	response = packageRequest(t, CommitTemplateImport, token, 1, payload)
	if response.Code == 200 {
		t.Fatal("过期预览仍能提交")
	}
}
func TestDownloadBuiltinTemplateDoesNotChangeIt(t *testing.T) {
	db := templatePackageDB(t)
	t.Setenv("TEMPLATE_DIR", filepath.Join("..", "..", "..", "templates"))
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Set("user_id", uint(1))
	ctx.Params = gin.Params{{Key: "id", Value: "redis"}}
	ctx.Request = httptest.NewRequest("GET", "/managed-templates/redis/download", nil)
	ExportTemplatePackage(ctx)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	decoded, err := apptemplate.ReadPackage(context.Background(), response.Body.Bytes())
	if err != nil || len(decoded) != 1 || decoded[0].Key != "redis" {
		t.Fatal(err)
	}
	var count int64
	db.Model(&models.ApplicationTemplate{}).Count(&count)
	if count != 0 {
		t.Fatal("下载写入了模板")
	}
}
func TestConfirmedImportSurvivesStagingCleanup(t *testing.T) {
	db := templatePackageDB(t)
	p := previewPackage(t, portableTemplateFixture())
	token := p["id"].(string)
	response := packageRequest(t, CommitTemplateImport, token, 1, map[string]any{"items": []any{packageChoice(0, "import")}})
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if response = packageRequest(t, CancelTemplateImport, token, 1, nil); response.Code != 204 {
		t.Fatal(response.Body.String())
	}
	var count int64
	db.Model(&models.ApplicationTemplate{}).Count(&count)
	if count != 1 {
		t.Fatal("清理预览删除了模板")
	}
}
