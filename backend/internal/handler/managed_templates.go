package handler

import (
	"dockerun/internal/apptemplate"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"strings"
)

func ManagedTemplates(c *gin.Context) {
	items, err := applicationTemplates()
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(200, items)
}

func applicationTemplates() ([]apptemplate.Template, error) {
	items, err := apptemplate.LoadDir(config.Load().TemplateDir)
	if err != nil {
		return nil, err
	}
	var rows []models.ApplicationTemplate
	if err := database.EnsureTemplateKeys(database.DB); err != nil {
		return nil, err
	}
	if err := database.DB.Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		source := row.Source
		if source == "" {
			source = "custom"
		}
		items = append(items, apptemplate.Template{ID: fmt.Sprintf("custom-%d", row.ID), Key: row.TemplateKey, Order: row.SortOrder, Name: row.Name, Version: row.Version, Description: row.Description, Category: row.Category, Source: source, Definition: row.Definition, Scripts: row.Scripts, Attachments: row.Attachments})
	}
	return items, nil
}

func findApplicationTemplate(id string) (apptemplate.Template, error) {
	items, err := applicationTemplates()
	if err != nil {
		return apptemplate.Template{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return apptemplate.Template{}, fmt.Errorf("应用模板不存在")
}

func CreateApplicationFromTemplate(c *gin.Context) {
	var q struct {
		Name            string           `json:"name"`
		Description     string           `json:"description"`
		Environment     string           `json:"environment"`
		TargetID        uint             `json:"target_id"`
		RegistryID      uint             `json:"registry_id"`
		TemplateVersion string           `json:"template_version"`
		Parameters      models.StringMap `json:"parameters"`
	}
	if c.ShouldBindJSON(&q) != nil || strings.TrimSpace(q.Name) == "" || len(q.Name) > 100 || strings.TrimSpace(q.Environment) == "" || len(q.Environment) > 100 {
		fail(c, fmt.Errorf("请填写应用名称和首个环境名称（最多 100 字符）"))
		return
	}
	t, err := findApplicationTemplate(c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	if q.TemplateVersion != "" && q.TemplateVersion != t.Version {
		fail(c, fmt.Errorf("模板版本已更新，请刷新模板后重新确认配置"))
		return
	}
	reg, err := release.DefaultRegistry(q.RegistryID)
	if err != nil {
		fail(c, err)
		return
	}
	if q.TargetID == 0 {
		q.TargetID = 1
	}
	var target models.Target
	if database.DB.First(&target, q.TargetID).Error != nil || !target.Enabled {
		fail(c, fmt.Errorf("目标主机不存在或未启用"))
		return
	}
	values, err := apptemplate.Values(t, q.Parameters, reg.URL)
	if err != nil {
		fail(c, err)
		return
	}
	if err := release.ValidateScripts(t.Scripts); err != nil {
		fail(c, err)
		return
	}
	if err := release.ValidateExecution(t.Definition.Execution); err != nil {
		fail(c, err)
		return
	}
	dir, err := os.MkdirTemp(filepath.Join(config.Load().DataDir, "releases"), "template-")
	if err != nil {
		fail(c, err)
		return
	}
	defer os.RemoveAll(dir)
	if err := release.WriteConfigs(dir, t.Definition.Configs); err != nil {
		fail(c, err)
		return
	}
	if _, err := release.Parse(c.Request.Context(), t.Definition.Compose, dir, "dockerun-template-preview", values); err != nil {
		fail(c, fmt.Errorf("模板配置检查失败：%w", err))
		return
	}
	a := models.Application{Name: strings.TrimSpace(q.Name), Description: q.Description, Kind: "compose", Category: t.Category, TemplateID: t.ID, TemplateVersion: t.Version, Definition: t.Definition, Scripts: t.Scripts}
	env := models.Environment{Name: strings.TrimSpace(q.Environment), TargetID: q.TargetID, RegistryID: reg.ID, Parameters: values, Stateless: !t.Definition.Stateful, AutoDeploy: false, AutoRollback: false, ScriptMode: "inherit", AllowedPublishers: "[]", TargetIDs: "[]", TagPattern: "*", Health: "unknown", RuntimeState: "unknown"}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&a).Error; err != nil {
			return fmt.Errorf("创建应用失败，请确认应用名称未被使用：%w", err)
		}
		env.ApplicationID = a.ID
		return tx.Create(&env).Error
	})
	if err != nil {
		fail(c, err)
		return
	}
	identity.Audit(c.GetUint("user_id"), "application.template.create", fmt.Sprint(a.ID))
	c.JSON(201, gin.H{"application": a, "environment": env})
}

// SaveApplicationTemplate 复制未插值的应用定义，从不复制环境参数值、主机或数据。
func SaveApplicationTemplate(c *gin.Context) {
	var q struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
		Key         string `json:"key"`
	}
	if c.ShouldBindJSON(&q) != nil || strings.TrimSpace(q.Name) == "" || len(q.Name) > 100 || strings.TrimSpace(q.Version) == "" || len(q.Version) > 100 {
		fail(c, fmt.Errorf("请填写模板名称和版本（最多 100 字符）"))
		return
	}
	var app models.Application
	if database.DB.First(&app, c.Param("id")).Error != nil {
		c.JSON(404, gin.H{"error": "应用不存在"})
		return
	}
	if app.Definition.Compose == "" {
		fail(c, fmt.Errorf("此应用尚未保存可复用定义，请先保存应用配置再创建模板"))
		return
	}
	templatePackageMu.Lock()
	defer templatePackageMu.Unlock()
	if err := database.EnsureTemplateKeys(database.DB); err != nil {
		fail(c, err)
		return
	}
	key := strings.TrimSpace(q.Key)
	if key != "" && !apptemplate.ValidKey(key) {
		fail(c, fmt.Errorf("模板标识无效"))
		return
	}
	if key == "" {
		var previous []models.ApplicationTemplate
		if err := database.DB.Where("name = ? AND (source = ? OR source = ?)", strings.TrimSpace(q.Name), "custom", "").Find(&previous).Error; err != nil {
			fail(c, err)
			return
		}
		for _, row := range previous {
			if key != "" && key != row.TemplateKey {
				fail(c, fmt.Errorf("存在同名模板，请使用不同的模板名称"))
				return
			}
			key = row.TemplateKey
		}
		if key == "" {
			key = database.NewTemplateKey()
		}
	}
	t := models.ApplicationTemplate{TemplateKey: key, Source: "custom", Name: strings.TrimSpace(q.Name), Version: strings.TrimSpace(q.Version), Description: q.Description, Category: app.Category, Definition: app.Definition, Scripts: app.Scripts}
	definition, err := apptemplate.ReusableDefinition(app.Definition)
	if err != nil {
		fail(c, err)
		return
	}
	t.Definition = definition
	t.Definition.Source, t.Definition.RegistryID = "", 0
	if t.Category == "" {
		t.Category = "business"
	}
	if t.Definition.Topology == "" {
		t.Definition.Topology = "single_host"
	}
	for i := range t.Definition.Parameters {
		if t.Definition.Parameters[i].Type == "password" {
			t.Definition.Parameters[i].Default = ""
		}
	}
	portable := apptemplate.Template{ID: key, Key: key, Name: t.Name, Version: t.Version, Category: t.Category, Description: t.Description, Definition: t.Definition, Scripts: t.Scripts}
	if err := apptemplate.ValidatePackage(c.Request.Context(), portable); err != nil {
		fail(c, err)
		return
	}
	catalog, err := applicationTemplates()
	if err != nil {
		fail(c, err)
		return
	}
	status, _, err := templateConflict(portable, catalog)
	if err != nil {
		fail(c, err)
		return
	}
	if status == "identical" || status == "conflict" {
		fail(c, fmt.Errorf("此模板标识和版本已经存在，请使用新版本"))
		return
	}
	if err := database.DB.Create(&t).Error; err != nil {
		fail(c, fmt.Errorf("保存模板失败，请使用未被占用的模板名称和版本：%w", err))
		return
	}
	identity.Audit(c.GetUint("user_id"), "application.template.save", fmt.Sprint(app.ID))
	c.JSON(201, gin.H{"id": fmt.Sprintf("custom-%d", t.ID), "key": t.TemplateKey, "name": t.Name, "version": t.Version})
}
