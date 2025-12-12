package handler

import (
	"dockerun/internal/database"
	"dockerun/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetSystemConfig 获取系统配置
func GetSystemConfig(c *gin.Context) {
	var config models.SystemConfig
	if err := database.GetDB().First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "系统配置不存在"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateSystemConfig 更新系统配置
func UpdateSystemConfig(c *gin.Context) {
	var req models.SystemConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	var config models.SystemConfig
	if err := database.GetDB().First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "系统配置不存在"})
		return
	}

	// 更新配置
	config.DeployWorkDir = req.DeployWorkDir
	config.MaxScriptExecutors = req.MaxScriptExecutors
	config.ScriptTimeout = req.ScriptTimeout
	config.EnableScriptLog = req.EnableScriptLog

	if err := database.GetDB().Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新系统配置失败"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// GetSystemConfigValue 获取指定配置项的值
func GetSystemConfigValue(key string) (interface{}, error) {
	var config models.SystemConfig
	if err := database.GetDB().First(&config).Error; err != nil {
		return nil, err
	}

	switch key {
	case "deploy_work_dir":
		return config.DeployWorkDir, nil
	case "max_script_executors":
		return config.MaxScriptExecutors, nil
	case "script_timeout":
		return config.ScriptTimeout, nil
	case "enable_script_log":
		return config.EnableScriptLog, nil
	default:
		return nil, nil
	}
}
