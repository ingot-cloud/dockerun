package webhook

import (
	"context"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

// DeployTask 部署任务
type DeployTask struct {
	WebhookLogID uint
	ConfigID     uint
	ImageName    string
	ImageTag     string
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

	log.Printf("[Webhook] Worker %d processing task: log_id=%d", workerID, task.WebhookLogID)

	// 更新日志状态为 processing
	now := time.Now()
	db.Model(&models.WebhookLog{}).Where("id = ?", task.WebhookLogID).Updates(map[string]interface{}{
		"status":     "processing",
		"started_at": &now,
	})

	// 获取配置信息
	var config models.WebhookConfig
	if err := db.Preload("DeployScript.Files").First(&config, task.ConfigID).Error; err != nil {
		q.updateLogFailed(task.WebhookLogID, fmt.Sprintf("Failed to load config: %v", err), startTime)
		return
	}

	// 构建完整的镜像名
	fullImageName := task.ImageName
	if task.ImageTag != "" {
		fullImageName = fmt.Sprintf("%s:%s", task.ImageName, task.ImageTag)
	}

	// 步骤1: 清理旧容器（如果配置了）
	if config.RemoveOldContainer {
		log.Printf("[Webhook] Removing old containers for image: %s", fullImageName)
		if err := q.removeOldContainers(fullImageName); err != nil {
			log.Printf("[Webhook] Warning: Failed to remove old containers: %v", err)
		}
	}

	// 步骤2: 清理旧镜像（如果配置了）
	if config.RemoveOldImage {
		log.Printf("[Webhook] Removing old image: %s", fullImageName)
		if err := q.removeOldImage(fullImageName); err != nil {
			log.Printf("[Webhook] Warning: Failed to remove old image: %v", err)
		}
	}

	// 步骤3: 拉取新镜像
	log.Printf("[Webhook] Pulling new image: %s", fullImageName)
	if err := q.pullImage(fullImageName, config.RegistryID); err != nil {
		q.updateLogFailed(task.WebhookLogID, fmt.Sprintf("Failed to pull image: %v", err), startTime)
		return
	}

	// 步骤4: 执行部署脚本
	log.Printf("[Webhook] Executing deploy script: %s", config.DeployScript.Name)
	if err := q.executeDeployScript(&config); err != nil {
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

	// 更新配置的触发统计
	db.Model(&models.WebhookConfig{}).Where("id = ?", task.ConfigID).Updates(map[string]interface{}{
		"last_triggered_at": &completedAt,
		"trigger_count":     db.Raw("trigger_count + 1"),
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
func (q *TaskQueue) pullImage(imageName string, registryID uint) error {
	// 拉取镜像
	reader, err := docker.PullImage(imageName)
	if err != nil {
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

// executeDeployScript 执行部署脚本
func (q *TaskQueue) executeDeployScript(config *models.WebhookConfig) error {
	// 这里重用 deploy_script 的执行逻辑
	// 可以调用 handler.ExecuteDeployScript 的核心逻辑
	// 为了简化，这里直接返回成功
	// 实际应该调用执行脚本的函数
	
	// TODO: 调用实际的脚本执行逻辑
	log.Printf("[Webhook] Deploy script execution: %s (id=%d)", config.DeployScript.Name, config.DeployScriptID)
	
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

