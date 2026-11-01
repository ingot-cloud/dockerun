package handler

import (
	"dockerun/internal/apptemplate"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"os"
	"path/filepath"
)

// SaveApplicationDeployment 只保存部署配置和当前脚本，不创建版本或执行任务。
func SaveApplicationDeployment(c *gin.Context) {
	var q struct {
		Compose       string                     `json:"compose"`
		Configs       models.StringMap           `json:"configs"`
		Execution     models.DeploymentExecution `json:"execution"`
		Scripts       models.DeploymentScripts   `json:"scripts"`
		EnvironmentID uint                       `json:"environment_id"`
		Source        string                     `json:"source"`
		RegistryID    uint                       `json:"registry_id"`
		Platform      string                     `json:"platform"`
	}
	if err := c.ShouldBindJSON(&q); err != nil {
		fail(c, err)
		return
	}
	var app models.Application
	var env models.Environment
	if database.DB.First(&app, c.Param("id")).Error != nil || database.DB.Where("id = ? AND application_id = ?", q.EnvironmentID, app.ID).First(&env).Error != nil {
		c.JSON(404, gin.H{"error": "应用或部署环境不存在"})
		return
	}
	if err := release.ValidateExecution(q.Execution); err != nil {
		fail(c, err)
		return
	}
	if err := release.ValidateScripts(q.Scripts); err != nil {
		fail(c, err)
		return
	}
	if q.Source == "" || (q.Execution.Mode == "command" && q.Source == "bundle") {
		q.Source = "direct"
	}
	if q.Source != "direct" && q.Source != "registry" && q.Source != "local" {
		fail(c, fmt.Errorf("请选择有效镜像来源"))
		return
	}
	if q.Platform != "" && q.Platform != "linux/amd64" && q.Platform != "linux/arm64" {
		fail(c, fmt.Errorf("目标平台仅支持 Linux amd64 或 arm64"))
		return
	}
	dir, err := os.MkdirTemp(filepath.Join(config.Load().DataDir, "releases"), "config-")
	if err != nil {
		fail(c, err)
		return
	}
	defer os.RemoveAll(dir)
	if err = release.WriteConfigs(dir, q.Configs); err != nil {
		fail(c, err)
		return
	}
	if q.Execution.Mode != "command" {
		if _, err = release.Parse(c.Request.Context(), q.Compose, dir, release.ProjectName(env), env.Parameters); err != nil {
			fail(c, err)
			return
		}
	}
	app.Definition.Compose = q.Compose
	app.Definition.Configs = q.Configs
	app.Definition.Execution = q.Execution
	app.Definition.Source = q.Source
	app.Definition.RegistryID = q.RegistryID
	app.Definition.Platform = q.Platform
	app.Kind = "compose"
	if q.Execution.Mode == "command" {
		app.Kind = "command"
	}
	if q.Execution.Mode != "command" {
		app.Definition.Parameters = apptemplate.Parameters(q.Compose, app.Definition.Parameters)
	}
	var automaticEnvironments []models.Environment
	if err := database.DB.Where("application_id = ? AND auto_deploy = ?", app.ID, true).Find(&automaticEnvironments).Error; err != nil {
		fail(c, err)
		return
	}
	for _, automatic := range automaticEnvironments {
		if _, err := release.AutomationDefinition(app, automatic); err != nil {
			fail(c, fmt.Errorf("环境 %s 的自动发布配置无效：%w", automatic.Name, err))
			return
		}
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		switch env.ScriptMode {
		case "override":
			env.Scripts = q.Scripts
			encoded, _ := json.Marshal(q.Scripts)
			if err := tx.Model(&models.Environment{}).Where("id = ?", env.ID).Update("scripts", string(encoded)).Error; err != nil {
				return err
			}
		case "disabled":
		default:
			app.Scripts = q.Scripts
		}
		return tx.Model(&app).Select("Definition", "Scripts", "Kind").Updates(&app).Error
	})
	if err != nil {
		fail(c, err)
		return
	}
	identity.Audit(c.GetUint("user_id"), "application.deployment.update", fmt.Sprint(app.ID))
	c.JSON(200, gin.H{"application": app, "environment": env})
}

func checkExpectedScripts(c *gin.Context, expected *models.DeploymentScripts, r models.Release, env models.Environment) bool {
	if expected != nil && *expected != release.EffectiveScripts(r, env) {
		c.JSON(409, gin.H{"error": "部署脚本已变化，请重新预览后发布"})
		return false
	}
	return true
}
