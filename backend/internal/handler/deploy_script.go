package handler

import (
	"bufio"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListDeployScripts 列出所有部署脚本
func ListDeployScripts(c *gin.Context) {
	var scripts []models.DeployScript
	// Preload Files 关联
	if err := database.GetDB().Preload("Files").Find(&scripts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 填充容器名称
	for i := range scripts {
		if scripts[i].ContainerID != "" {
			scripts[i].ContainerName = docker.GetContainerName(scripts[i].ContainerID)
		}
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
		req.WorkDir = "/ingot-data/dockerun"
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

	// 更新基本字段
	script.Name = req.Name
	script.Description = req.Description
	script.ScriptType = req.ScriptType
	script.MainScript = req.MainScript
	script.WorkDir = req.WorkDir
	script.EnvVars = req.EnvVars

	// 使用事务来确保数据一致性
	err := database.GetDB().Transaction(func(tx *gorm.DB) error {
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
		}

		return nil
	})

	if err != nil {
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

	// 使用配置的工作目录或系统默认工作目录
	var workDir string
	useTemp := false
	if script.WorkDir != "" && script.WorkDir != "/ingot-data/dockerun" {
		// 使用用户指定的工作目录
		workDir = script.WorkDir
	} else {
		// 使用系统配置的默认工作目录
		var systemConfig models.SystemConfig
		if err := database.GetDB().First(&systemConfig).Error; err == nil {
			workDir = filepath.Join(systemConfig.DeployWorkDir, fmt.Sprintf("script-%d", script.ID))
		} else {
			// 如果系统配置不存在，使用临时目录
			workDir = filepath.Join(os.TempDir(), fmt.Sprintf("dockerun-deploy-%d", script.ID))
			useTemp = true
		}
	}

	// 创建工作目录
	if err := os.MkdirAll(workDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建工作目录失败: " + err.Error()})
		return
	}

	// 如果使用临时目录，执行完后清理
	if useTemp {
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
		log.Println("写入文件: ", fileFullPath)
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

	// 添加 Docker 网络名称（从配置中获取）
	cfg := config.Load()
	env = append(env, fmt.Sprintf("DOCKER_NETWORK=%s", cfg.NetworkName))

	// 执行脚本（流式输出）
	cmd := exec.Command("/bin/bash", mainScriptPath)
	cmd.Dir = workDir
	cmd.Env = env

	// 获取输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建输出管道失败"})
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建错误管道失败"})
		return
	}

	// 设置流式响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 启动命令
	if err := cmd.Start(); err != nil {
		script.Status = "failed"
		database.GetDB().Save(&script)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "启动脚本失败: " + err.Error()})
		return
	}

	// 用于收集所有输出
	var fullOutput strings.Builder
	flusher, _ := c.Writer.(http.Flusher)

	// 发送日志事件
	sendLogEvent := func(line string) {
		fullOutput.WriteString(line + "\n")
		data, _ := json.Marshal(map[string]string{
			"type": "log",
			"data": line,
		})
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		if flusher != nil {
			flusher.Flush()
		}
	}

	// 读取标准输出
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			sendLogEvent(scanner.Text())
		}
	}()

	// 读取标准错误
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			sendLogEvent("[STDERR] " + scanner.Text())
		}
	}()

	// 等待命令完成
	cmdErr := cmd.Wait()

	outputStr := fullOutput.String()

	// 发送完成事件
	if cmdErr != nil {
		script.Status = "failed"
		database.GetDB().Save(&script)

		data, _ := json.Marshal(map[string]interface{}{
			"type":    "error",
			"message": "脚本执行失败: " + cmdErr.Error(),
			"output":  outputStr,
		})
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		if flusher != nil {
			flusher.Flush()
		}
		return
	}

	// 智能提取容器ID
	containerID := extractContainerID(outputStr)
	if containerID != "" {
		script.ContainerID = containerID
		script.ContainerName = docker.GetContainerName(containerID)

		// 检查容器是否真的在运行
		if docker.IsContainerRunning(containerID) {
			script.Status = "running"
		} else {
			script.Status = "stopped"
		}
	} else {
		script.Status = "stopped"
	}

	database.GetDB().Save(&script)

	// 发送成功事件
	data, _ := json.Marshal(map[string]interface{}{
		"type":         "complete",
		"message":      "脚本执行成功",
		"output":       outputStr,
		"script":       script,
		"container_id": containerID,
	})
	fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	if flusher != nil {
		flusher.Flush()
	}
}

// extractContainerID 智能提取容器ID
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
