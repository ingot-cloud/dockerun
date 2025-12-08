package handler

import (
	"dockerun/internal/database"
	"dockerun/internal/models"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// ListDeployScripts 列出所有部署脚本
func ListDeployScripts(c *gin.Context) {
	var scripts []models.DeployScript
	if err := database.GetDB().Find(&scripts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, scripts)
}

// GetDeployScript 获取部署脚本详情
func GetDeployScript(c *gin.Context) {
	id := c.Param("id")
	
	var script models.DeployScript
	if err := database.GetDB().First(&script, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "脚本不存在"})
		return
	}

	c.JSON(http.StatusOK, script)
}

// CreateDeployScript 创建部署脚本
func CreateDeployScript(c *gin.Context) {
	var req models.DeployScript
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	userID := c.GetUint("user_id")
	req.UserID = userID
	req.Status = "idle"

	if err := database.GetDB().Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, req)
}

// UpdateDeployScript 更新部署脚本
func UpdateDeployScript(c *gin.Context) {
	id := c.Param("id")
	
	var script models.DeployScript
	if err := database.GetDB().First(&script, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "脚本不存在"})
		return
	}

	var req models.DeployScript
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 更新字段
	script.Name = req.Name
	script.Description = req.Description
	script.ScriptType = req.ScriptType
	script.Content = req.Content
	script.ComposeFile = req.ComposeFile

	if err := database.GetDB().Save(&script).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, script)
}

// DeleteDeployScript 删除部署脚本
func DeleteDeployScript(c *gin.Context) {
	id := c.Param("id")
	
	var script models.DeployScript
	if err := database.GetDB().First(&script, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "脚本不存在"})
		return
	}

	if err := database.GetDB().Delete(&script).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ExecuteDeployScript 执行部署脚本
func ExecuteDeployScript(c *gin.Context) {
	id := c.Param("id")
	
	var script models.DeployScript
	if err := database.GetDB().First(&script, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "脚本不存在"})
		return
	}

	// 创建临时目录
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("dockerun-deploy-%d", script.ID))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时目录失败"})
		return
	}
	defer os.RemoveAll(tempDir)

	// 写入脚本文件
	scriptPath := filepath.Join(tempDir, "deploy.sh")
	if err := os.WriteFile(scriptPath, []byte(script.Content), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入脚本失败"})
		return
	}

	// 如果是docker-compose类型，写入compose文件
	if script.ScriptType == "docker-compose" && script.ComposeFile != "" {
		composePath := filepath.Join(tempDir, "docker-compose.yml")
		if err := os.WriteFile(composePath, []byte(script.ComposeFile), 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入compose文件失败"})
			return
		}
	}

	// 更新脚本状态
	script.Status = "running"
	database.GetDB().Save(&script)

	// 执行脚本
	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		script.Status = "failed"
		database.GetDB().Save(&script)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "脚本执行失败",
			"output": string(output),
		})
		return
	}

	// 尝试获取容器ID（从脚本输出中提取）
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if len(line) == 64 || len(line) == 12 { // Docker容器ID长度
			script.ContainerID = line
			break
		}
	}

	script.Status = "stopped"
	database.GetDB().Save(&script)

	c.JSON(http.StatusOK, gin.H{
		"message": "脚本执行成功",
		"output":  outputStr,
		"script":  script,
	})
}

