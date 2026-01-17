package webhook

import (
	"bytes"
	"context"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/registry"
)

// DeployTask 部署任务
type DeployTask struct {
	WebhookLogID       uint
	ConfigID           uint   // 旧模式：使用 WebhookConfig ID
	DeployScriptID     uint   // 新模式：直接指定 DeployScript ID
	RegistryID         string // 新模式：Registry ID（用于获取认证信息）
	ImageName          string
	ImageTag           string
	RemoveOldImage     bool // 是否删除旧镜像
	RemoveOldContainer bool // 是否删除旧容器
}

// TaskQueue 任务队列
type TaskQueue struct {
	tasks   chan *DeployTask
	workers int
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
}

var globalQueue *TaskQueue

// InitQueue 初始化任务队列
func InitQueue(workers int) {
	if globalQueue != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	globalQueue = &TaskQueue{
		tasks:   make(chan *DeployTask, 100), // 缓冲 100 个任务
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
	}

	// 启动工作协程
	for i := 0; i < workers; i++ {
		go globalQueue.worker(i)
	}

	log.Printf("[Webhook] Task queue initialized with %d workers", workers)
}

// Enqueue 添加任务到队列
func Enqueue(task *DeployTask) error {
	if globalQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}

	select {
	case globalQueue.tasks <- task:
		log.Printf("[Webhook] Task enqueued: log_id=%d, image=%s:%s", task.WebhookLogID, task.ImageName, task.ImageTag)
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("task queue is full")
	}
}

// worker 工作协程
func (q *TaskQueue) worker(id int) {
	log.Printf("[Webhook] Worker %d started", id)
	for {
		select {
		case <-q.ctx.Done():
			log.Printf("[Webhook] Worker %d stopped", id)
			return
		case task := <-q.tasks:
			q.processTask(id, task)
		}
	}
}

// processTask 处理任务
func (q *TaskQueue) processTask(workerID int, task *DeployTask) {
	db := database.GetDB()
	startTime := time.Now()

	log.Printf("[Webhook] Worker %d processing task: log_id=%d, script_id=%d", workerID, task.WebhookLogID, task.DeployScriptID)

	// 更新日志状态为 processing
	now := time.Now()
	db.Model(&models.WebhookLog{}).Where("id = ?", task.WebhookLogID).Updates(map[string]interface{}{
		"status":     "processing",
		"started_at": &now,
	})

	// 加载 DeployScript
	var script models.DeployScript
	if err := db.First(&script, task.DeployScriptID).Error; err != nil {
		q.updateLogFailed(task.WebhookLogID, fmt.Sprintf("Failed to load deploy script: %v", err), startTime)
		return
	}

	// 手动加载DeployScript的Files
	var files []models.DeployScriptFile
	if err := db.Where("deploy_script_id = ?", script.ID).Find(&files).Error; err == nil {
		script.Files = files
	}

	// 加载 Registry（如果有）
	var registry *models.Registry
	if task.RegistryID != "" {
		var reg models.Registry
		if err := db.First(&reg, task.RegistryID).Error; err == nil {
			registry = &reg
		}
	}

	// 构建完整的镜像名
	fullImageName := task.ImageName
	if registry != nil {
		// 从 Registry URL 中提取地址（去掉 http:// 或 https://）
		registryAddr := registry.URL
		registryAddr = strings.TrimPrefix(registryAddr, "http://")
		registryAddr = strings.TrimPrefix(registryAddr, "https://")
		registryAddr = strings.TrimSuffix(registryAddr, "/")

		// 构建完整镜像名：registry/image
		fullImageName = fmt.Sprintf("%s/%s", registryAddr, task.ImageName)
	}

	if task.ImageTag != "" {
		fullImageName = fmt.Sprintf("%s:%s", fullImageName, task.ImageTag)
	}

	// 步骤1: 清理旧容器（如果配置了）
	if task.RemoveOldContainer {
		log.Printf("[Webhook] Removing old containers for image: %s", fullImageName)
		if err := q.removeOldContainers(fullImageName); err != nil {
			log.Printf("[Webhook] Warning: Failed to remove old containers: %v", err)
		}
	}

	// 步骤2: 清理旧镜像（如果配置了）
	if task.RemoveOldImage {
		log.Printf("[Webhook] Removing old image: %s", fullImageName)
		if err := q.removeOldImage(fullImageName); err != nil {
			log.Printf("[Webhook] Warning: Failed to remove old image: %v", err)
		}
	}

	// 步骤3: 拉取新镜像
	log.Printf("[Webhook] Pulling new image: %s", fullImageName)
	if err := q.pullImage(fullImageName, registry); err != nil {
		q.updateLogFailed(task.WebhookLogID, fmt.Sprintf("Failed to pull image: %v", err), startTime)
		return
	}

	// 步骤4: 执行部署脚本
	log.Printf("[Webhook] Executing deploy script: %s", script.Name)
	if err := q.executeDeployScript(&script, task.ImageName, task.ImageTag, registry); err != nil {
		q.updateLogFailed(task.WebhookLogID, fmt.Sprintf("Failed to execute deploy script: %v", err), startTime)
		return
	}

	// 成功完成
	duration := int(time.Since(startTime).Seconds())
	completedAt := time.Now()
	db.Model(&models.WebhookLog{}).Where("id = ?", task.WebhookLogID).Updates(map[string]interface{}{
		"status":         "success",
		"message":        fmt.Sprintf("Successfully deployed %s", fullImageName),
		"completed_at":   &completedAt,
		"execution_time": duration,
	})

	log.Printf("[Webhook] Task completed successfully: log_id=%d, duration=%ds", task.WebhookLogID, duration)
}

// removeOldContainers 删除使用指定镜像的旧容器
func (q *TaskQueue) removeOldContainers(imageName string) error {
	ctx := context.Background()
	cli := docker.GetClient()

	// 查找使用该镜像的容器
	filterArgs := filters.NewArgs()
	filterArgs.Add("ancestor", imageName)

	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return err
	}

	// 删除每个容器
	for _, cont := range containers {
		log.Printf("[Webhook] Stopping and removing container: %s (%s)", cont.Names, cont.ID[:12])

		// 停止容器
		timeout := 10
		if err := cli.ContainerStop(ctx, cont.ID, container.StopOptions{Timeout: &timeout}); err != nil {
			log.Printf("[Webhook] Warning: Failed to stop container %s: %v", cont.ID[:12], err)
		}

		// 删除容器
		if err := cli.ContainerRemove(ctx, cont.ID, container.RemoveOptions{Force: true}); err != nil {
			log.Printf("[Webhook] Warning: Failed to remove container %s: %v", cont.ID[:12], err)
		}
	}

	return nil
}

// removeOldImage 删除旧镜像
func (q *TaskQueue) removeOldImage(imageName string) error {
	ctx := context.Background()
	cli := docker.GetClient()

	_, err := cli.ImageRemove(ctx, imageName, dockerTypes.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	})

	return err
}

// pullImage 拉取镜像
func (q *TaskQueue) pullImage(imageName string, reg *models.Registry) error {
	ctx := context.Background()
	cli := docker.GetClient()

	// 构建拉取选项
	pullOptions := dockerTypes.ImagePullOptions{}

	// 如果配置了 Registry 且有认证信息，添加认证
	if reg != nil && reg.Username != "" && reg.Password != "" {
		authConfig := registry.AuthConfig{
			Username:      reg.Username,
			Password:      reg.Password,
			ServerAddress: reg.URL,
		}

		// 编码认证信息
		encodedAuth, err := encodeAuthConfig(authConfig)
		if err != nil {
			log.Printf("[Webhook] Warning: Failed to encode auth config: %v", err)
		} else {
			pullOptions.RegistryAuth = encodedAuth
		}
	}

	// 拉取镜像
	reader, err := cli.ImagePull(ctx, imageName, pullOptions)
	if err != nil {
		// 检查是否是 insecure registry 错误
		errMsg := err.Error()
		if strings.Contains(errMsg, "server gave HTTP response to HTTPS client") {
			// 提供详细的错误提示
			registryHost := extractRegistryHost(imageName)
			if reg != nil && reg.Insecure {
				return fmt.Errorf("failed to pull image: Registry is configured as insecure, but Docker daemon requires HTTPS. "+
					"Please add '%s' to Docker daemon's insecure-registries configuration. "+
					"See INSECURE_REGISTRY.md for instructions. Original error: %w", registryHost, err)
			}
			return fmt.Errorf("failed to pull image: Docker requires HTTPS but got HTTP response. "+
				"If this is an insecure registry, please: "+
				"1) Mark the registry as 'Insecure' in Dockerun settings, "+
				"2) Add '%s' to Docker daemon's insecure-registries configuration. "+
				"See INSECURE_REGISTRY.md for details. Original error: %w", registryHost, err)
		}
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// 读取输出（确保拉取完成）
	_, err = io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read pull output: %w", err)
	}

	return nil
}

// extractRegistryHost 从镜像名中提取 registry 地址
func extractRegistryHost(imageName string) string {
	// 去掉标签
	parts := strings.Split(imageName, ":")
	imageWithoutTag := parts[0]

	// 提取 registry 地址
	slashIndex := strings.Index(imageWithoutTag, "/")
	if slashIndex > 0 {
		host := imageWithoutTag[:slashIndex]
		// 检查是否包含端口号或域名
		if strings.Contains(host, ".") || strings.Contains(host, ":") || host == "localhost" {
			return host
		}
	}

	return "registry-host"
}

// encodeAuthConfig 编码认证配置
func encodeAuthConfig(authConfig registry.AuthConfig) (string, error) {
	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(authJSON), nil
}

// executeDeployScript 执行部署脚本
func (q *TaskQueue) executeDeployScript(script *models.DeployScript, imageName, imageTag string, registry *models.Registry) error {
	if script == nil {
		return fmt.Errorf("deploy script is nil")
	}

	log.Printf("[Webhook] Executing deploy script: %s (id=%d)", script.Name, script.ID)

	// 1. 确定工作目录
	var workDir string
	useTemp := false

	if script.WorkDir != "" && script.WorkDir != "/ingot-data/dockerun" {
		// 使用用户指定的工作目录
		workDir = script.WorkDir
	} else {
		// 使用系统配置的默认工作目录
		db := database.GetDB()
		var systemConfig models.SystemConfig
		if err := db.First(&systemConfig).Error; err == nil {
			workDir = filepath.Join(systemConfig.DeployWorkDir, fmt.Sprintf("script-%d", script.ID))
		} else {
			// 如果系统配置不存在，使用临时目录
			workDir = filepath.Join(os.TempDir(), fmt.Sprintf("dockerun-webhook-%d-%d", script.ID, time.Now().Unix()))
			useTemp = true
		}
	}

	// 2. 创建工作目录
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	// 如果使用临时目录，执行完后清理
	if useTemp {
		defer func() {
			if err := os.RemoveAll(workDir); err != nil {
				log.Printf("[Webhook] Warning: Failed to clean up temp directory %s: %v", workDir, err)
			}
		}()
	}

	log.Printf("[Webhook] Using work directory: %s", workDir)

	// 3. 写入所有关联文件
	for _, file := range script.Files {
		// 创建文件所在的目录
		fileFullPath := filepath.Join(workDir, file.FilePath)
		fileDir := filepath.Dir(fileFullPath)
		if err := os.MkdirAll(fileDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", fileDir, err)
		}

		// 写入文件内容
		perm := os.FileMode(0644)
		if file.FileType == "script" {
			perm = 0755
		}

		log.Printf("[Webhook] Writing file: %s", fileFullPath)
		if err := os.WriteFile(fileFullPath, []byte(file.Content), perm); err != nil {
			return fmt.Errorf("failed to write file %s: %w", file.FileName, err)
		}
	}

	// 4. 写入主脚本文件
	mainScriptPath := filepath.Join(workDir, "deploy.sh")
	if err := os.WriteFile(mainScriptPath, []byte(script.MainScript), 0755); err != nil {
		return fmt.Errorf("failed to write main script: %w", err)
	}

	// 5. 准备环境变量
	env := os.Environ()

	// 添加脚本自定义环境变量
	if script.EnvVars != nil {
		for key, value := range script.EnvVars {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// 添加工作目录环境变量
	env = append(env, fmt.Sprintf("WORK_DIR=%s", workDir))

	// 添加 Docker 网络名称
	appConfig := config.Load()
	env = append(env, fmt.Sprintf("DOCKER_NETWORK=%s", appConfig.NetworkName))

	// 添加触发信息环境变量（方便脚本使用）
	env = append(env, fmt.Sprintf("WEBHOOK_IMAGE_NAME=%s", imageName))
	env = append(env, fmt.Sprintf("WEBHOOK_IMAGE_TAG=%s", imageTag))
	if registry != nil {
		env = append(env, fmt.Sprintf("WEBHOOK_REGISTRY_URL=%s", registry.URL))
	}

	// 6. 执行脚本
	log.Printf("[Webhook] Executing script: %s", mainScriptPath)

	cmd := exec.Command("/bin/bash", mainScriptPath)
	cmd.Dir = workDir
	cmd.Env = env

	// 捕获输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 启动并等待完成
	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	// 记录输出
	if stdout.Len() > 0 {
		log.Printf("[Webhook] Script stdout:\n%s", stdout.String())
	}
	if stderr.Len() > 0 {
		log.Printf("[Webhook] Script stderr:\n%s", stderr.String())
	}

	if err != nil {
		// 构建详细的错误信息
		errorMsg := fmt.Sprintf("script execution failed after %v: %v", duration, err)
		if stderr.Len() > 0 {
			errorMsg += fmt.Sprintf("\nStderr: %s", stderr.String())
		}
		return fmt.Errorf("%s", errorMsg)
	}

	log.Printf("[Webhook] Script execution completed successfully in %v", duration)
	return nil
}

// updateLogFailed 更新日志为失败状态
func (q *TaskQueue) updateLogFailed(logID uint, message string, startTime time.Time) {
	db := database.GetDB()
	duration := int(time.Since(startTime).Seconds())
	completedAt := time.Now()

	db.Model(&models.WebhookLog{}).Where("id = ?", logID).Updates(map[string]interface{}{
		"status":         "failed",
		"message":        message,
		"completed_at":   &completedAt,
		"execution_time": duration,
	})

	log.Printf("[Webhook] Task failed: log_id=%d, message=%s", logID, message)
}

// Shutdown 关闭任务队列
func Shutdown() {
	if globalQueue != nil {
		globalQueue.cancel()
		close(globalQueue.tasks)
	}
}
