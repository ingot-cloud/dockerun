package handler

import (
	"dockerun/internal/apptemplate"
	"dockerun/internal/builtin"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var environmentParameterName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// SaveEnvironmentParameters 只更新运行参数，保留环境的主机、脚本及发布策略。
func SaveEnvironmentParameters(c *gin.Context) {
	var q struct {
		Parameters         models.StringMap  `json:"parameters"`
		ExpectedParameters *models.StringMap `json:"expected_parameters"`
	}
	if c.ShouldBindJSON(&q) != nil || q.Parameters == nil {
		fail(c, fmt.Errorf("请提供运行参数对象，参数名称和值必须是字符串"))
		return
	}
	builtin.ReferenceGate.Lock()
	defer builtin.ReferenceGate.Unlock()
	var env models.Environment
	var app models.Application
	if database.DB.Where("id = ? AND application_id = ?", c.Param("environment_id"), c.Param("id")).First(&env).Error != nil || database.DB.First(&app, env.ApplicationID).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "部署环境不存在或不属于此应用"})
		return
	}
	if !checkExpectedEnvironmentParameters(c, q.ExpectedParameters, env.Parameters) {
		return
	}
	for key, value := range q.Parameters {
		if !environmentParameterName.MatchString(key) {
			fail(c, fmt.Errorf("运行参数名称「%s」无效，请以字母或下划线开头，仅使用字母、数字和下划线", key))
			return
		}
		if strings.ContainsRune(value, 0) {
			fail(c, fmt.Errorf("运行参数「%s」不能包含空字符", key))
			return
		}
	}
	if len(app.Definition.Parameters) > 0 {
		reg, err := release.DefaultRegistry(env.RegistryID)
		if err != nil {
			fail(c, err)
			return
		}
		values, err := apptemplate.Values(apptemplate.Template{Definition: app.Definition}, q.Parameters, reg.URL)
		if err != nil {
			fail(c, err)
			return
		}
		for key, value := range values {
			q.Parameters[key] = value
		}
	}
	env.Parameters = q.Parameters
	if err := release.ValidateContainerBindings(env); err != nil {
		fail(c, err)
		return
	}
	if err := database.DB.Model(&models.Environment{}).Where("id = ? AND application_id = ?", env.ID, app.ID).Update("parameters", q.Parameters).Error; err != nil {
		fail(c, err)
		return
	}
	identity.Audit(c.GetUint("user_id"), "environment.parameters.update", fmt.Sprint(env.ID))
	c.JSON(http.StatusOK, gin.H{"environment_id": env.ID, "parameters": q.Parameters})
}

func checkExpectedEnvironmentParameters(c *gin.Context, expected *models.StringMap, actual models.StringMap) bool {
	if expected != nil && !maps.Equal(*expected, actual) {
		c.JSON(http.StatusConflict, gin.H{"error": "环境运行参数已更新，请刷新后重新确认修改"})
		return false
	}
	return true
}
