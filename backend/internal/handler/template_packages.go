package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"dockerun/internal/apptemplate"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var templatePackageMu sync.Mutex
var templateImportID = regexp.MustCompile(`^[a-f0-9]{32}$`)

const templateImportTTL = 30 * time.Minute

type templateImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	IDs      []string `json:"ids"`
}
type templateImportState struct {
	UserID      uint                  `json:"user_id"`
	SHA256      string                `json:"sha256"`
	ExpiresAt   time.Time             `json:"expires_at"`
	RequestHash string                `json:"request_hash,omitempty"`
	Result      *templateImportResult `json:"result,omitempty"`
}
type templateImportChoice struct {
	Index  int    `json:"index"`
	Action string `json:"action"`
	Key    string `json:"key,omitempty"`
	Name   string `json:"name,omitempty"`
}
type templateImportEntry struct {
	Index          int              `json:"index"`
	Key            string           `json:"key"`
	Name           string           `json:"name"`
	Version        string           `json:"version"`
	Description    string           `json:"description"`
	Category       string           `json:"category"`
	Status         string           `json:"status"`
	Message        string           `json:"message"`
	ParameterCount int              `json:"parameter_count"`
	ConfigCount    int              `json:"config_count"`
	ScriptStages   []string         `json:"script_stages"`
	Files          models.StringMap `json:"files"`
}

func templateImportRoot() string { return filepath.Join(config.Load().DataDir, "template-imports") }
func templateImportDir(id string) (string, error) {
	if !templateImportID.MatchString(id) {
		return "", fmt.Errorf("模板导入编号无效")
	}
	return filepath.Join(templateImportRoot(), id), nil
}

func CleanupTemplateImports() {
	templatePackageMu.Lock()
	defer templatePackageMu.Unlock()
	cleanupTemplateImportsLocked()
}
func cleanupTemplateImportsLocked() {
	entries, _ := os.ReadDir(templateImportRoot())
	for _, entry := range entries {
		if !entry.IsDir() || !templateImportID.MatchString(entry.Name()) {
			continue
		}
		dir := filepath.Join(templateImportRoot(), entry.Name())
		var state templateImportState
		b, err := os.ReadFile(filepath.Join(dir, "state.json"))
		info, statErr := entry.Info()
		expired := err == nil && json.Unmarshal(b, &state) == nil && !time.Now().Before(state.ExpiresAt)
		abandoned := statErr == nil && time.Since(info.ModTime()) > templateImportTTL
		if expired || abandoned {
			os.RemoveAll(dir)
		}
	}
}
func saveTemplateImport(dir string, state templateImportState) error {
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "state-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(dir, "state.json"))
}
func loadTemplateImport(id string, userID uint) (string, templateImportState, error) {
	dir, err := templateImportDir(id)
	if err != nil {
		return "", templateImportState{}, err
	}
	var state templateImportState
	b, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		return "", state, fmt.Errorf("导入预览不存在或已过期，请重新上传 ZIP")
	}
	if err = json.Unmarshal(b, &state); err != nil {
		return "", state, err
	}
	if state.UserID != userID {
		return "", state, fmt.Errorf("无权访问此导入预览")
	}
	if !time.Now().Before(state.ExpiresAt) {
		os.RemoveAll(dir)
		return "", state, fmt.Errorf("导入预览已过期，请重新上传 ZIP")
	}
	return dir, state, nil
}
func templateConflict(t apptemplate.Template, catalog []apptemplate.Template) (string, string, error) {
	hash, err := apptemplate.Fingerprint(t)
	if err != nil {
		return "", "", err
	}
	family := false
	identical := false
	for _, existing := range catalog {
		if apptemplate.PortableKey(existing) != apptemplate.PortableKey(t) {
			continue
		}
		family = true
		if existing.Version != t.Version {
			continue
		}
		other, err := apptemplate.Fingerprint(existing)
		if err != nil {
			return "", "", err
		}
		if other != hash {
			return "conflict", "标识和版本已存在，但内容不同；请另存为新模板或修改版本", nil
		}
		identical = true
	}
	if identical {
		return "identical", "相同模板版本已存在，将跳过重复导入", nil
	}
	if family {
		return "new_version", "将保存为此模板的一个新版本", nil
	}
	return "new", "将新增模板", nil
}

func PreviewTemplateImport(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, apptemplate.MaxArchiveSize+(1<<20))
	err := c.Request.ParseMultipartForm(1 << 20)
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	if err != nil {
		fail(c, fmt.Errorf("无法读取模板文件，ZIP 不能超过 32 MiB"))
		return
	}
	headers := c.Request.MultipartForm.File["file"]
	count := 0
	for _, files := range c.Request.MultipartForm.File {
		count += len(files)
	}
	if len(headers) != 1 || count != 1 || !strings.EqualFold(filepath.Ext(headers[0].Filename), ".zip") {
		fail(c, fmt.Errorf("请选择一个 .zip 格式的模板包"))
		return
	}
	f, err := headers[0].Open()
	if err != nil {
		fail(c, err)
		return
	}
	data, err := io.ReadAll(io.LimitReader(f, apptemplate.MaxArchiveSize+1))
	f.Close()
	if err != nil {
		fail(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	items, err := apptemplate.ReadPackage(ctx, data)
	if err != nil {
		fail(c, err)
		return
	}
	catalog, err := applicationTemplates()
	if err != nil {
		fail(c, err)
		return
	}
	entries := []templateImportEntry{}
	for index, item := range items {
		status, message, err := templateConflict(item, catalog)
		if err != nil {
			fail(c, err)
			return
		}
		files, err := apptemplate.PackageFiles(item)
		if err != nil {
			fail(c, err)
			return
		}
		stages := []string{}
		for _, stage := range []struct{ name, content string }{{"部署前", item.Scripts.Before}, {"部署", item.Scripts.Deploy}, {"部署后", item.Scripts.After}} {
			if stage.content != "" {
				stages = append(stages, stage.name)
			}
		}
		if item.Definition.Execution.Command != "" {
			stages = append(stages, "主部署命令")
		}
		if item.Definition.Execution.CheckCommand != "" {
			stages = append(stages, "部署检查命令")
		}
		entries = append(entries, templateImportEntry{index, item.Key, item.Name, item.Version, item.Description, item.Category, status, message, len(item.Definition.Parameters), len(item.Definition.Configs), stages, files})
	}
	templatePackageMu.Lock()
	defer templatePackageMu.Unlock()
	cleanupTemplateImportsLocked()
	existing, _ := os.ReadDir(templateImportRoot())
	userPending, totalPending := 0, 0
	for _, entry := range existing {
		if !entry.IsDir() || !templateImportID.MatchString(entry.Name()) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(templateImportRoot(), entry.Name(), "state.json"))
		var old templateImportState
		if err == nil && json.Unmarshal(b, &old) == nil && old.Result == nil {
			totalPending++
			if old.UserID == c.GetUint("user_id") {
				userPending++
			}
		}
	}
	if userPending >= 3 || totalPending >= 64 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "待确认的模板导入过多，请关闭已有预览后重试"})
		return
	}
	if err = os.MkdirAll(templateImportRoot(), 0700); err != nil {
		fail(c, err)
		return
	}
	tokenBytes := make([]byte, 16)
	if _, err = rand.Read(tokenBytes); err != nil {
		fail(c, err)
		return
	}
	id := hex.EncodeToString(tokenBytes)
	dir, _ := templateImportDir(id)
	if err = os.Mkdir(dir, 0700); err != nil {
		fail(c, err)
		return
	}
	done := false
	defer func() {
		if !done {
			os.RemoveAll(dir)
		}
	}()
	if err = os.WriteFile(filepath.Join(dir, "package.zip"), data, 0600); err != nil {
		fail(c, err)
		return
	}
	sum := sha256.Sum256(data)
	state := templateImportState{UserID: c.GetUint("user_id"), SHA256: hex.EncodeToString(sum[:]), ExpiresAt: time.Now().Add(templateImportTTL)}
	if err = saveTemplateImport(dir, state); err != nil {
		fail(c, err)
		return
	}
	done = true
	c.JSON(http.StatusOK, gin.H{"id": id, "expires_at": state.ExpiresAt, "entries": entries})
}
func CancelTemplateImport(c *gin.Context) {
	templatePackageMu.Lock()
	defer templatePackageMu.Unlock()
	dir, _, err := loadTemplateImport(c.Param("import"), c.GetUint("user_id"))
	if err != nil {
		fail(c, err)
		return
	}
	if err = os.RemoveAll(dir); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func CommitTemplateImport(c *gin.Context) {
	var q struct {
		Items []templateImportChoice `json:"items"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128<<10)
	if c.ShouldBindJSON(&q) != nil || len(q.Items) == 0 || len(q.Items) > apptemplate.MaxPackageTemplates {
		fail(c, fmt.Errorf("请选择要导入的模板"))
		return
	}
	sort.Slice(q.Items, func(i, j int) bool { return q.Items[i].Index < q.Items[j].Index })
	body, _ := json.Marshal(q)
	requestSum := sha256.Sum256(body)
	requestHash := hex.EncodeToString(requestSum[:])
	templatePackageMu.Lock()
	defer templatePackageMu.Unlock()
	dir, state, err := loadTemplateImport(c.Param("import"), c.GetUint("user_id"))
	if err != nil {
		fail(c, err)
		return
	}
	if state.Result != nil {
		if state.RequestHash != requestHash {
			c.JSON(409, gin.H{"error": "本次导入已完成，请重新上传后进行新的导入"})
			return
		}
		c.JSON(200, state.Result)
		return
	}
	data, err := os.ReadFile(filepath.Join(dir, "package.zip"))
	if err != nil {
		fail(c, err)
		return
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != state.SHA256 {
		fail(c, fmt.Errorf("导入材料已变化，请重新上传"))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	items, err := apptemplate.ReadPackage(ctx, data)
	if err != nil {
		fail(c, err)
		return
	}
	catalog, err := applicationTemplates()
	if err != nil {
		fail(c, err)
		return
	}
	originalCatalog := append([]apptemplate.Template{}, catalog...)
	selected := []apptemplate.Template{}
	seen := map[int]bool{}
	identities := map[string]bool{}
	skipped := 0
	for _, choice := range q.Items {
		if choice.Index < 0 || choice.Index >= len(items) || seen[choice.Index] {
			fail(c, fmt.Errorf("模板选择无效或重复"))
			return
		}
		seen[choice.Index] = true
		if choice.Action == "skip" {
			skipped++
			continue
		}
		item := items[choice.Index]
		switch choice.Action {
		case "import":
		case "rename":
			if !apptemplate.ValidKey(choice.Key) || strings.TrimSpace(choice.Name) == "" || len(choice.Name) > 100 {
				fail(c, fmt.Errorf("另存模板需填写有效的新标识和名称，名称最多 100 字节"))
				return
			}
			for _, old := range originalCatalog {
				if apptemplate.PortableKey(old) == choice.Key {
					c.JSON(409, gin.H{"error": "另存标识已存在，请填写新的模板标识"})
					return
				}
			}
			item.Key, item.ID, item.Name = choice.Key, choice.Key, strings.TrimSpace(choice.Name)
		default:
			fail(c, fmt.Errorf("不支持的导入方式"))
			return
		}
		identity := item.Key + "\x00" + item.Version
		if identities[identity] {
			fail(c, fmt.Errorf("本次导入中存在相同的标识和版本：%s", item.Key))
			return
		}
		identities[identity] = true
		status, _, err := templateConflict(item, catalog)
		if err != nil {
			fail(c, err)
			return
		}
		if status == "conflict" {
			c.JSON(409, gin.H{"error": fmt.Sprintf("模板 %s · %s 内容冲突，请另存或重新预览", item.Key, item.Version)})
			return
		}
		if status == "identical" {
			skipped++
			continue
		}
		selected = append(selected, item)
		catalog = append(catalog, item)
	}
	result := templateImportResult{Skipped: skipped, IDs: []string{}}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range selected {
			row := models.ApplicationTemplate{TemplateKey: item.Key, Name: item.Name, Version: item.Version, Source: "imported", SortOrder: item.Order, Description: item.Description, Category: item.Category, Definition: item.Definition, Scripts: item.Scripts, Attachments: item.Attachments}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			if err := tx.Create(&models.AuditEvent{UserID: c.GetUint("user_id"), Action: "application.template.import", Resource: item.Key + " · " + item.Version}).Error; err != nil {
				return err
			}
			result.Imported++
			result.IDs = append(result.IDs, fmt.Sprintf("custom-%d", row.ID))
		}
		return nil
	})
	if err != nil {
		c.JSON(409, gin.H{"error": "模板保存失败或版本状态已变化，整批未写入，请重新预览后重试"})
		return
	}
	state.Result = &result
	state.RequestHash = requestHash
	if err = saveTemplateImport(dir, state); err == nil {
		os.Remove(filepath.Join(dir, "package.zip"))
	}
	c.JSON(200, result)
}

func ExportTemplatePackage(c *gin.Context) {
	var q struct {
		IDs []string `json:"ids"`
	}
	if id := c.Param("id"); id != "" {
		q.IDs = []string{id}
	} else {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
		if c.ShouldBindJSON(&q) != nil {
			fail(c, fmt.Errorf("下载请求无效"))
			return
		}
	}
	if len(q.IDs) == 0 || len(q.IDs) > apptemplate.MaxPackageTemplates {
		fail(c, fmt.Errorf("请选择 1 至 %d 个模板", apptemplate.MaxPackageTemplates))
		return
	}
	catalog, err := applicationTemplates()
	if err != nil {
		fail(c, err)
		return
	}
	byID := map[string]apptemplate.Template{}
	for _, item := range catalog {
		byID[item.ID] = item
	}
	selected := []apptemplate.Template{}
	seen := map[string]bool{}
	for _, id := range q.IDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		item, ok := byID[id]
		if !ok {
			c.JSON(404, gin.H{"error": "模板不存在或已删除，请刷新后重试"})
			return
		}
		selected = append(selected, item)
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	data, err := apptemplate.WritePackage(ctx, selected)
	if err != nil {
		fail(c, err)
		return
	}
	filename := "dockerun-templates.zip"
	if len(selected) == 1 {
		filename = apptemplate.PortableKey(selected[0]) + ".zip"
	}
	identity.Audit(c.GetUint("user_id"), "application.template.export", fmt.Sprintf("%d 个模板", len(selected)))
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	c.Header("Cache-Control", "no-store")
	c.Data(200, "application/zip", data)
}
