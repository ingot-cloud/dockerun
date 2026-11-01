package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gorm.io/gorm/clause"
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
	var cfg models.WebhookConfig
	if database.DB.First(&cfg, c.Param("id")).Error != nil || !cfg.Enabled || cfg.Secret == "" {
		c.JSON(403, gin.H{"error": "Webhook 未启用或未配置签名密钥"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 4<<20+1))
	if err != nil || len(body) > 4<<20 {
		c.Status(413)
		return
	}
	signature := c.GetHeader("X-Registry-Signature")
	if signature == "" {
		signature = c.GetHeader("X-Hub-Signature-256")
	}
	if signature == "" {
		signature = c.GetHeader("X-Gitlab-Token")
	}
	if !verifySignature(body, cfg.Secret, signature) {
		c.Status(401)
		return
	}
	var payload WebhookPayload
	if json.Unmarshal(body, &payload) != nil {
		c.Status(400)
		return
	}
	var reg models.Registry
	if database.DB.First(&reg, cfg.RegistryID).Error != nil || reg.Kind == "builtin" {
		c.Status(400)
		return
	}
	var owner models.User
	if database.DB.First(&owner, cfg.OwnerID).Error != nil || !owner.Enabled || !owner.IsAdmin {
		c.JSON(409, gin.H{"error": "请由管理员重新保存 Webhook 来源规则"})
		return
	}
	for _, e := range payload.Events {
		if e.Action != "push" || !strings.Contains(e.Target.MediaType, "manifest") && !strings.Contains(e.Target.MediaType, "index") || e.Target.Tag == "" {
			continue
		}
		ref := strings.TrimPrefix(strings.TrimPrefix(reg.URL, "https://"), "http://") + "/" + e.Target.Repository + ":" + e.Target.Tag
		im, err := release.Resolve(c.Request.Context(), reg, ref)
		if err != nil || im.Digest != e.Target.Digest {
			c.JSON(502, gin.H{"error": "无法确认镜像提交结果"})
			return
		}
		delivery := e.ID + ":" + e.Timestamp
		if e.ID == "" && e.Timestamp == "" {
			// 无事件身份时交给环境当前及排队状态去重，不能永久屏蔽切回历史 tag 的新推送。
			delivery = identity.Random()
		}
		key := identity.Hash(fmt.Sprintf("external:%d:%s:%s:%s:%s", cfg.ID, e.Target.Repository, e.Target.Tag, im.Digest, delivery))
		ev := models.ImageEvent{RegistryID: reg.ID, Repository: e.Target.Repository, Reference: e.Target.Tag, Digest: im.Digest, MediaType: e.Target.MediaType, UserID: owner.ID, Key: key, Committed: true}
		if err := database.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&ev).Error; err != nil {
			c.Status(503)
			return
		}
	}
	c.JSON(202, gin.H{"status": "persisted"})
}

func loadWebhookConfigRelations(db *gorm.DB, config *models.WebhookConfig) error {
	// 加载Registry
	if config.RegistryID > 0 {
		var registry models.Registry
		if err := db.First(&registry, config.RegistryID).Error; err == nil {
			registry.Password = ""
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

	// 首先简单判断，如果和secret相等则直接返回true
	if hmac.Equal([]byte(signature), []byte(secret)) {
		return true
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

	req.OwnerID = c.GetUint("user_id")
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
	req.OwnerID = c.GetUint("user_id")
	if err := db.Model(&config).Select("name", "description", "enabled", "registry_id", "secret", "owner_id").Updates(&req).Error; err != nil {
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
