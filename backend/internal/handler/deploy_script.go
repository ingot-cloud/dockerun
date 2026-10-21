package handler

import (
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// loadDeployScriptFiles 加载DeployScript的关联文件
func loadDeployScriptFiles(db *gorm.DB, script *models.DeployScript) error {
	var files []models.DeployScriptFile
	if err := db.Where("deploy_script_id = ?", script.ID).Find(&files).Error; err != nil {
		return err
	}
	script.Files = files
	return nil
}

// ListDeployScripts 列出所有部署脚本
func ListDeployScripts(c *gin.Context) {
	db := database.GetDB()

	var scripts []models.DeployScript
	if err := db.Find(&scripts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 手动加载每个脚本的关联文件
	for i := range scripts {
		loadDeployScriptFiles(db, &scripts[i])

		// 填充容器名称
		if scripts[i].ContainerID != "" {
			scripts[i].ContainerName = docker.GetContainerName(scripts[i].ContainerID)
		}
	}

	c.JSON(http.StatusOK, scripts)
}

// GetDeployScript 获取部署脚本详情
func GetDeployScript(c *gin.Context) {
	db := database.GetDB()
	id := c.Param("id")

	var script models.DeployScript
	if err := db.First(&script, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "脚本不存在"})
		return
	}

	// 手动加载关联文件
	loadDeployScriptFiles(db, &script)

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
		req.WorkDir = "/ingot-data/dockerun"
	}

	db := database.GetDB()

	// 使用事务处理脚本和文件的创建
	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. 创建脚本记录（Files 字段有 gorm:"-"，不会自动保存）
		if err := tx.Omit("Files").Create(&req).Error; err != nil {
			return err
		}

		// 2. 手动保存关联文件到 DeployScriptFile 表
		if len(req.Files) > 0 {
			for i := range req.Files {
				req.Files[i].DeployScriptID = req.ID
				req.Files[i].ID = 0 // 确保是新记录
			}
			if err := tx.Create(&req.Files).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, req)
}

// UpdateDeployScript 更新部署脚本
func UpdateDeployScript(c *gin.Context) {
	db := database.GetDB()
	id := c.Param("id")

	var script models.DeployScript
	if err := db.First(&script, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "脚本不存在"})
		return
	}

	var req models.DeployScript
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 更新基本字段
	script.Name = req.Name
	script.Description = req.Description
	script.ScriptType = req.ScriptType
	script.MainScript = req.MainScript
	script.WorkDir = req.WorkDir
	script.EnvVars = req.EnvVars
	script.ImageName = req.ImageName
	script.ImageTag = req.ImageTag
	script.AutoDeploy = req.AutoDeploy

	// 使用事务来确保数据一致性
	err := db.Transaction(func(tx *gorm.DB) error {
		// 先保存基本字段
		if err := tx.Omit("Files").Save(&script).Error; err != nil {
			return err
		}

		// 删除旧的文件记录
		if err := tx.Where("deploy_script_id = ?", script.ID).Delete(&models.DeployScriptFile{}).Error; err != nil {
			return err
		}

		// 如果有新的文件，创建它们
		if len(req.Files) > 0 {
			for i := range req.Files {
				req.Files[i].DeployScriptID = script.ID
				req.Files[i].ID = 0 // 确保创建新记录
			}
			if err := tx.Create(&req.Files).Error; err != nil {
				return err
			}
			script.Files = req.Files
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	// 重新加载以获取关联数据
	db.First(&script, script.ID)
	loadDeployScriptFiles(db, &script)

	c.JSON(http.StatusOK, script)
}

// DeleteDeployScript 删除部署脚本（级联删除关联文件和Webhook配置）
func DeleteDeployScript(c *gin.Context) {
	db := database.GetDB()
	id := c.Param("id")

	var script models.DeployScript
	if err := db.First(&script, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "脚本不存在"})
		return
	}

	// 使用事务处理级联删除
	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. 删除关联的文件记录
		if err := tx.Where("deploy_script_id = ?", script.ID).Delete(&models.DeployScriptFile{}).Error; err != nil {
			return err
		}

		// 5. 最后删除脚本本身
		if err := tx.Delete(&script).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ExecuteDeployScript 执行部署脚本
func ExecuteDeployScript(c *gin.Context) {
	var script models.DeployScript
	if database.DB.First(&script, c.Param("id")).Error != nil {
		c.JSON(404, gin.H{"error": "脚本不存在"})
		return
	}
	loadDeployScriptFiles(database.DB, &script)
	task, err := release.EnqueueLegacy(script, c.GetUint("user_id"))
	if err != nil {
		fail(c, err)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	after := uint(0)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		var logs []models.TaskLog
		database.DB.Where("deployment_id = ? AND id > ?", task.ID, after).Order("id").Find(&logs)
		for _, l := range logs {
			data, _ := json.Marshal(gin.H{"type": "log", "data": l.Message, "task_id": task.ID})
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			after = l.ID
		}
		database.DB.First(task, task.ID)
		c.Writer.Flush()
		if task.Status != "queued" && task.Status != "running" {
			kind := "complete"
			if task.Status != "success" {
				kind = "error"
			}
			data, _ := json.Marshal(gin.H{"type": kind, "message": task.Result, "task_id": task.ID})
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			c.Writer.Flush()
			return
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func extractContainerID(output string) string {
	// 方法1: 查找标准的Docker容器ID（64位或12位十六进制）
	// 匹配 "Container ID:" 或类似的输出
	patterns := []string{
		`(?i)container\s+id[:\s]+([a-f0-9]{12,64})`,
		`(?i)started\s+([a-f0-9]{12,64})`,
		`(?i)created\s+([a-f0-9]{12,64})`,
		// 匹配单独一行的容器ID
		`^([a-f0-9]{64})$`,
		`^([a-f0-9]{12})$`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(output); len(matches) > 1 {
			return matches[1]
		}
	}

	// 方法2: 查找 docker-compose 的输出
	// 例如: "Creating network...", "Creating container_name...", "Container xxx Started"
	composePattern := regexp.MustCompile(`(?i)container\s+(\S+)\s+(?:started|created)`)
	if matches := composePattern.FindStringSubmatch(output); len(matches) > 1 {
		containerName := matches[1]
		// 尝试通过容器名称获取ID
		if id := docker.GetContainerIDByName(containerName); id != "" {
			return id
		}
	}

	// 方法3: 如果输出中有明确的十六进制ID行
	hexIDPattern := regexp.MustCompile(`^[a-f0-9]{12,64}$`)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 检查是否是纯十六进制的容器ID
		if hexIDPattern.MatchString(trimmed) {
			return trimmed
		}
	}

	return ""
}
