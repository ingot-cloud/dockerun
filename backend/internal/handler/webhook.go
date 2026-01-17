package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"dockerun/internal/webhook"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// WebhookPayload Docker Registry Webhook 的通用 Payload 结构
type WebhookPayload struct {
	Events []WebhookEvent `json:"events"`
}

// WebhookEvent Webhook 事件
type WebhookEvent struct {
	ID        string                 `json:"id"`
	Timestamp string                 `json:"timestamp"`
	Action    string                 `json:"action"` // push, pull, delete
	Target    WebhookEventTarget     `json:"target"`
	Request   map[string]interface{} `json:"request,omitempty"`
	Actor     map[string]interface{} `json:"actor,omitempty"`
}

// WebhookEventTarget 事件目标（镜像信息）
type WebhookEventTarget struct {
	MediaType  string `json:"mediaType"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest"`
	Length     int64  `json:"length"`
	Repository string `json:"repository"` // 镜像名称，如 "namespace/repo"
	URL        string `json:"url"`
	Tag        string `json:"tag"` // 标签，如 "latest", "v1.0.0"
}

// ReceiveWebhook 接收 Webhook 推送
// @Summary 接收 Docker Registry Webhook
// @Description 接收来自 Docker Registry 的 Webhook 推送事件
// @Tags Webhook
// @Accept json
// @Produce json
// @Param id path int true "Webhook Config ID"
// @Success 200 {object} map[string]interface{}
// @Router /webhooks/{id} [post]
func ReceiveWebhook(c *gin.Context) {
	webhookID := c.Param("id")
	log.Printf("[Webhook] ReceiveWebhook: webhookId=%s", webhookID)

	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[Webhook] Failed to read request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 快速响应（避免 Registry 超时）
	c.JSON(http.StatusOK, gin.H{"status": "received"})

	// 异步处理 Webhook
	go processWebhook(webhookID, body, c.Request.Header)
}

// processWebhook 异步处理 Webhook
func processWebhook(webhookID string, body []byte, headers http.Header) {
	db := database.GetDB()

	// 加载 Webhook 配置
	var webhookConfig models.WebhookConfig
	if err := db.First(&webhookConfig, webhookID).Error; err != nil {
		log.Printf("[Webhook] Webhook config not found: id=%s, error=%v", webhookID, err)
		return
	}

	// 加载关联的 Registry
	var registry models.Registry
	if err := db.First(&registry, webhookConfig.RegistryID).Error; err != nil {
		log.Printf("[Webhook] Registry not found: id=%d, error=%v", webhookConfig.RegistryID, err)
		return
	}

	log.Printf("[Webhook] Processing webhook: %s (id=%s) for registry: %s", webhookConfig.Name, webhookID, registry.Name)

	// 验证签名（如果配置了 Secret）
	if webhookConfig.Secret != "" {
		signature := headers.Get("X-Registry-Signature")
		if signature == "" {
			signature = headers.Get("X-Hub-Signature-256")
		}
		if signature == "" {
			signature = headers.Get("X-Gitlab-Token")
		}

		if !verifySignature(body, webhookConfig.Secret, signature) {
			log.Printf("[Webhook] Signature verification failed for webhook: %s", webhookID)
			return
		}
	}

	// 解析 Payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[Webhook] Failed to parse payload: %v", err)
		return
	}

	log.Printf("[Webhook] Received %d events", len(payload.Events))

	// 处理每个事件
	for _, event := range payload.Events {
		// 只处理 push 事件
		if event.Action != "push" {
			log.Printf("[Webhook] Skipping non-push event: action=%s", event.Action)
			continue
		}

		imageName := event.Target.Repository
		imageTag := event.Target.Tag

		log.Printf("[Webhook] Processing push event: image=%s, tag=%s", imageName, imageTag)

		// 查找所有匹配的部署脚本
		matchedScripts := findMatchingDeployScripts(db, imageName, imageTag)

		if len(matchedScripts) == 0 {
			log.Printf("[Webhook] No matching deploy scripts found for %s:%s", imageName, imageTag)
			continue
		}

		log.Printf("[Webhook] Found %d matching scripts for %s:%s", len(matchedScripts), imageName, imageTag)

		// 为每个匹配的脚本创建部署任务
		for _, script := range matchedScripts {
			log.Printf("[Webhook] Creating task for script: %s (id=%d)", script.Name, script.ID)

			// 创建 Webhook 日志
			webhookLog := &models.WebhookLog{
				WebhookConfigID: webhookConfig.ID,
				ImageName:       imageName,
				ImageTag:        imageTag,
				Status:          "pending",
				Message:         fmt.Sprintf("Auto-deploy triggered for script: %s", script.Name),
			}

			if err := db.Create(webhookLog).Error; err != nil {
				log.Printf("[Webhook] Failed to create webhook log: %v", err)
				continue
			}

			// 创建部署任务
			task := &webhook.DeployTask{
				WebhookLogID:       webhookLog.ID,
				DeployScriptID:     script.ID,
				RegistryID:         fmt.Sprintf("%d", webhookConfig.RegistryID),
				ImageName:          imageName,
				ImageTag:           imageTag,
				RemoveOldImage:     true, // 默认删除旧镜像
				RemoveOldContainer: true, // 默认删除旧容器
			}

			if err := webhook.Enqueue(task); err != nil {
				log.Printf("[Webhook] Failed to enqueue task: %v", err)
				// 更新日志状态为失败
				db.Model(webhookLog).Updates(map[string]interface{}{
					"status":  "failed",
					"message": fmt.Sprintf("Failed to enqueue task: %v", err),
				})
			}
		}
	}
}

func loadWebhookConfigRelations(db *gorm.DB, config *models.WebhookConfig) error {
	// 加载Registry
	if config.RegistryID > 0 {
		var registry models.Registry
		if err := db.First(&registry, config.RegistryID).Error; err == nil {
			config.Registry = &registry
		}
	}

	return nil
}

// verifySignature 验证签名
func verifySignature(body []byte, secret, signature string) bool {
	if signature == "" {
		return false
	}

	// 支持多种签名格式
	// Docker Hub: sha256=xxxxx
	// Harbor: sha256:xxxxx
	var expectedHash string
	if strings.HasPrefix(signature, "sha256=") {
		expectedHash = strings.TrimPrefix(signature, "sha256=")
	} else if strings.HasPrefix(signature, "sha256:") {
		expectedHash = strings.TrimPrefix(signature, "sha256:")
	} else {
		expectedHash = signature
	}

	// 计算 HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	calculatedHash := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedHash), []byte(calculatedHash))
}

// matchImageName 匹配镜像名称（支持通配符和 registry 地址）
func matchImageName(pattern, name string) bool {
	// 1. 直接匹配
	if pattern == name {
		return true
	}

	// 2. 如果 pattern 包含 registry 地址（如 192.168.0.170:5000/indohit/gateway）
	// 而 name 不包含（如 indohit/gateway），则提取 pattern 的镜像名部分进行匹配
	if strings.Contains(pattern, "/") {
		// 尝试提取镜像名（去掉 registry 地址部分）
		// registry 地址通常是 host:port/ 或 host/ 的形式
		parts := strings.SplitN(pattern, "/", 2)
		if len(parts) == 2 {
			// 检查第一部分是否像是 registry 地址（包含 . 或 : 或 localhost）
			firstPart := parts[0]
			if strings.Contains(firstPart, ".") || strings.Contains(firstPart, ":") || firstPart == "localhost" {
				// 使用镜像名部分匹配
				if parts[1] == name {
					return true
				}
			}
		}
	}

	// 3. 支持 * 通配符
	if strings.Contains(pattern, "*") {
		// 简单的前缀/后缀匹配
		if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
			return strings.Contains(name, strings.Trim(pattern, "*"))
		} else if strings.HasPrefix(pattern, "*") {
			return strings.HasSuffix(name, strings.TrimPrefix(pattern, "*"))
		} else if strings.HasSuffix(pattern, "*") {
			return strings.HasPrefix(name, strings.TrimSuffix(pattern, "*"))
		}
	}

	return false
}

// ListWebhookConfigs 列出所有 Webhook 配置
func ListWebhookConfigs(c *gin.Context) {
	db := database.GetDB()

	var configs []models.WebhookConfig
	if err := db.Order("created_at DESC").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 手动加载每个配置的关联数据
	for i := range configs {
		loadWebhookConfigRelations(db, &configs[i])
	}

	c.JSON(http.StatusOK, configs)
}

// GetWebhookConfig 获取单个 Webhook 配置
func GetWebhookConfig(c *gin.Context) {
	db := database.GetDB()
	id := c.Param("id")

	var config models.WebhookConfig
	if err := db.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook config not found"})
		return
	}

	// 手动加载关联数据
	loadWebhookConfigRelations(db, &config)

	c.JSON(http.StatusOK, config)
}

// CreateWebhookConfig 创建 Webhook 配置
func CreateWebhookConfig(c *gin.Context) {
	db := database.GetDB()

	var req models.WebhookConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证 Registry 存在
	var registry models.Registry
	if err := db.First(&registry, req.RegistryID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Registry not found"})
		return
	}

	// 创建配置
	if err := db.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 手动加载关联数据
	loadWebhookConfigRelations(db, &req)

	c.JSON(http.StatusOK, req)
}

// UpdateWebhookConfig 更新 Webhook 配置
func UpdateWebhookConfig(c *gin.Context) {
	db := database.GetDB()
	id := c.Param("id")

	var config models.WebhookConfig
	if err := db.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook config not found"})
		return
	}

	var req models.WebhookConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证 Registry 存在
	if req.RegistryID > 0 {
		var registry models.Registry
		if err := db.First(&registry, req.RegistryID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Registry not found"})
			return
		}
	}

	// 更新配置
	req.ID = config.ID
	if err := db.Model(&config).Updates(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新加载完整数据
	db.First(&config, id)
	loadWebhookConfigRelations(db, &config)

	c.JSON(http.StatusOK, config)
}

// DeleteWebhookConfig 删除 Webhook 配置（级联删除相关日志）
func DeleteWebhookConfig(c *gin.Context) {
	db := database.GetDB()
	id := c.Param("id")

	// 使用事务处理级联删除
	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. 删除关联的 WebhookLog
		if err := tx.Where("webhook_config_id = ?", id).Delete(&models.WebhookLog{}).Error; err != nil {
			return err
		}

		// 2. 删除 WebhookConfig
		if err := tx.Delete(&models.WebhookConfig{}, id).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook config deleted"})
}

// ListWebhookLogs 列出 Webhook 日志
func ListWebhookLogs(c *gin.Context) {
	db := database.GetDB()
	configID := c.Query("config_id")

	query := db.Order("created_at DESC")
	if configID != "" {
		query = query.Where("webhook_config_id = ?", configID)
	}

	var logs []models.WebhookLog
	if err := query.Limit(100).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 手动加载每个日志的关联配置
	for i := range logs {
		if logs[i].WebhookConfigID > 0 {
			var config models.WebhookConfig
			if err := db.First(&config, logs[i].WebhookConfigID).Error; err == nil {
				logs[i].WebhookConfig = &config
			}
		}
	}

	c.JSON(http.StatusOK, logs)
}

// GetWebhookURL 获取 Webhook URL
func GetWebhookURL(c *gin.Context) {
	id := c.Param("id")

	// 获取当前请求的 Host
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	host := c.Request.Host

	webhookURL := fmt.Sprintf("%s://%s/webhooks/%s", scheme, host, id)

	c.JSON(http.StatusOK, gin.H{
		"url":  webhookURL,
		"note": "Configure this URL in your Docker Registry webhook settings",
	})
}

// findMatchingDeployScripts 查找匹配的部署脚本
func findMatchingDeployScripts(db *gorm.DB, imageName, imageTag string) []models.DeployScript {
	var scripts []models.DeployScript

	// 构建查询条件
	query := db.Where("auto_deploy = ?", true).Where("image_name = ?", imageName)

	// 标签匹配逻辑：
	// 1. 精确匹配：image_tag = imageTag
	// 2. 通配标签：image_tag 为空或 "*"，表示匹配任意标签
	query = query.Where(db.Where("image_tag = ?", imageTag).
		Or("image_tag = ?", "").
		Or("image_tag = ?", "*"))

	if err := query.Find(&scripts).Error; err != nil {
		log.Printf("[Webhook] Failed to query deploy scripts: %v", err)
		return nil
	}

	return scripts
}
