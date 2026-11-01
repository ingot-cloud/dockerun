package handler

import (
	"dockerun/internal/builtin"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/distribution/reference"
	"github.com/gin-gonic/gin"
)

// SaveEnvironmentAutomation 独立保存触发规则，保留主机、参数、脚本和运行状态。
func SaveEnvironmentAutomation(c *gin.Context) {
	var q struct {
		AutoDeploy        bool      `json:"auto_deploy"`
		RegistryID        uint      `json:"registry_id"`
		Repository        string    `json:"repository"`
		TagPattern        string    `json:"tag_pattern"`
		AllowedPublishers string    `json:"allowed_publishers"`
		ContainerNames    *[]string `json:"container_names"`
		ReplaceContainers *bool     `json:"replace_containers"`
	}
	if err := c.ShouldBindJSON(&q); err != nil {
		fail(c, err)
		return
	}
	builtin.ReferenceGate.Lock()
	defer builtin.ReferenceGate.Unlock()
	var env models.Environment
	if err := database.DB.Where("id = ? AND application_id = ?", c.Param("environment_id"), c.Param("id")).First(&env).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "部署环境不存在或不属于此应用"})
		return
	}
	reg, err := release.DefaultRegistry(q.RegistryID)
	if err != nil {
		fail(c, err)
		return
	}
	env.AutoDeploy, env.RegistryID = q.AutoDeploy, reg.ID
	env.Repository, env.TagPattern = strings.TrimSpace(q.Repository), q.TagPattern
	if env.TagPattern == "" {
		env.TagPattern = "*"
	}
	if _, err := filepath.Match(env.TagPattern, "test"); err != nil {
		fail(c, fmt.Errorf("标签规则格式无效，请使用 latest、test-* 或 * 等规则"))
		return
	}
	if q.AllowedPublishers == "" {
		q.AllowedPublishers = "[]"
	}
	var publishers []uint
	if json.Unmarshal([]byte(q.AllowedPublishers), &publishers) != nil {
		fail(c, fmt.Errorf("推送人列表格式错误"))
		return
	}
	env.AllowedPublishers = q.AllowedPublishers
	if q.ContainerNames != nil {
		env.ContainerNames = *q.ContainerNames
	}
	if q.ReplaceContainers != nil {
		env.ReplaceContainers = *q.ReplaceContainers
	}
	if err := release.ValidateContainerBindings(env); err != nil {
		fail(c, err)
		return
	}
	if env.AutoDeploy {
		name, err := reference.WithName("registry.invalid/" + env.Repository)
		if err != nil || reference.Path(name) != env.Repository {
			fail(c, fmt.Errorf("请填写镜像名称，不含仓库域名和标签，例如 team/gateway"))
			return
		}
		if _, err := release.AutomationBase(env); err != nil {
			fail(c, err)
			return
		}
	}
	updates := map[string]any{"auto_deploy": env.AutoDeploy, "registry_id": env.RegistryID, "repository": env.Repository,
		"tag_pattern": env.TagPattern, "allowed_publishers": env.AllowedPublishers}
	if q.ContainerNames != nil {
		encoded, _ := json.Marshal(env.ContainerNames)
		updates["container_names"] = string(encoded)
	}
	if q.ReplaceContainers != nil {
		updates["replace_containers"] = env.ReplaceContainers
	}
	if err := database.DB.Model(&models.Environment{}).Where("id = ? AND application_id = ?", env.ID, env.ApplicationID).Updates(updates).Error; err != nil {
		fail(c, err)
		return
	}
	identity.Audit(c.GetUint("user_id"), "environment.automation.update", fmt.Sprint(env.ID))
	c.JSON(http.StatusOK, env)
}
