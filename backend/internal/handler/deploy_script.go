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
	// Preload Files 关联
	if err := database.GetDB().Preload("Files").Find(&scripts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, scripts)
}

// GetDeployScript 获取部署脚本详情
func GetDeployScript(c *gin.Context) {
	id := c.Param("id")
	
	var script models.DeployScript
	// Preload Files 关联
	if err := database.GetDB().Preload("Files").First(&script, id).Error; err != nil {
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

	// 如果没有设置工作目录，使用默认值
	if req.WorkDir == "" {
		req.WorkDir = "/tmp/deploy"
	}

	// 创建脚本（包括关联的文件）
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
	if err := database.GetDB().Preload("Files").First(&script, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "脚本不存在"})
		return
	}

	var req models.DeployScript
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 删除旧的文件记录
	database.GetDB().Where("deploy_script_id = ?", script.ID).Delete(&models.DeployScriptFile{})

	// 更新字段
	script.Name = req.Name
	script.Description = req.Description
	script.ScriptType = req.ScriptType
	script.MainScript = req.MainScript
	script.WorkDir = req.WorkDir
	script.EnvVars = req.EnvVars
	script.Files = req.Files

	if err := database.GetDB().Save(&script).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	// 重新加载以获取关联数据
	database.GetDB().Preload("Files").First(&script, script.ID)

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

	// 删除关联的文件记录（由于设置了外键，会自动级联删除）
	database.GetDB().Where("deploy_script_id = ?", script.ID).Delete(&models.DeployScriptFile{})

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
	if err := database.GetDB().Preload("Files").First(&script, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "脚本不存在"})
		return
	}

	// 使用配置的工作目录或创建临时目录
	var workDir string
	if script.WorkDir != "" && script.WorkDir != "/tmp/deploy" {
		// 使用用户指定的工作目录
		workDir = script.WorkDir
		if err := os.MkdirAll(workDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建工作目录失败: " + err.Error()})
			return
		}
	} else {
		// 创建临时目录
		workDir = filepath.Join(os.TempDir(), fmt.Sprintf("dockerun-deploy-%d", script.ID))
		if err := os.MkdirAll(workDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时目录失败"})
			return
		}
		defer os.RemoveAll(workDir)
	}

	// 写入所有关联文件
	for _, file := range script.Files {
		// 创建文件所在的目录
		fileFullPath := filepath.Join(workDir, file.FilePath)
		fileDir := filepath.Dir(fileFullPath)
		if err := os.MkdirAll(fileDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("创建目录失败: %s", fileDir),
			})
			return
		}

		// 写入文件内容
		perm := os.FileMode(0644)
		if file.FileType == "script" {
			perm = 0755
		}
		if err := os.WriteFile(fileFullPath, []byte(file.Content), perm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("写入文件失败: %s", file.FileName),
			})
			return
		}
	}

	// 写入主脚本文件
	mainScriptPath := filepath.Join(workDir, "deploy.sh")
	if err := os.WriteFile(mainScriptPath, []byte(script.MainScript), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入主脚本失败"})
		return
	}

	// 更新脚本状态
	script.Status = "running"
	database.GetDB().Save(&script)

	// 准备环境变量
	env := os.Environ()
	if script.EnvVars != nil {
		for key, value := range script.EnvVars {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
	}
	// 添加工作目录环境变量
	env = append(env, fmt.Sprintf("WORK_DIR=%s", workDir))

	// 执行脚本
	cmd := exec.Command("/bin/bash", mainScriptPath)
	cmd.Dir = workDir
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	outputStr := string(output)

	if err != nil {
		script.Status = "failed"
		database.GetDB().Save(&script)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "脚本执行失败: " + err.Error(),
			"output": outputStr,
		})
		return
	}

	// 尝试获取容器ID（从脚本输出中提取）
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 64 || len(trimmed) == 12 { // Docker容器ID长度
			script.ContainerID = trimmed
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

