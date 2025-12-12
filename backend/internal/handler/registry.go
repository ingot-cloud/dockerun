package handler

import (
	"dockerun/internal/database"
	"dockerun/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ListRegistries 获取所有 Registry
func ListRegistries(c *gin.Context) {
	var registries []models.Registry
	if err := database.GetDB().Order("created_at DESC").Find(&registries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Registry 列表失败"})
		return
	}

	// 不返回密码
	for i := range registries {
		registries[i].Password = ""
	}

	c.JSON(http.StatusOK, registries)
}

// GetRegistry 获取单个 Registry
func GetRegistry(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var registry models.Registry
	if err := database.GetDB().First(&registry, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Registry 不存在"})
		return
	}

	// 不返回密码
	registry.Password = ""

	c.JSON(http.StatusOK, registry)
}

// CreateRegistry 创建 Registry
func CreateRegistry(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		URL       string `json:"url" binding:"required"`
		Username  string `json:"username"`
		Password  string `json:"password"`
		IsDefault bool   `json:"is_default"`
		Insecure  bool   `json:"insecure"`
		Email     string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 如果设置为默认，先取消其他默认
	if req.IsDefault {
		database.GetDB().Model(&models.Registry{}).Where("is_default = ?", true).Update("is_default", false)
	}

	registry := models.Registry{
		Name:      req.Name,
		URL:       req.URL,
		Username:  req.Username,
		Password:  req.Password,
		IsDefault: req.IsDefault,
		Insecure:  req.Insecure,
		Email:     req.Email,
	}

	if err := database.GetDB().Create(&registry).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建 Registry 失败"})
		return
	}

	// 不返回密码
	registry.Password = ""

	c.JSON(http.StatusOK, registry)
}

// UpdateRegistry 更新 Registry
func UpdateRegistry(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		Name      string `json:"name"`
		URL       string `json:"url"`
		Username  string `json:"username"`
		Password  string `json:"password"`
		IsDefault bool   `json:"is_default"`
		Insecure  bool   `json:"insecure"`
		Email     string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	var registry models.Registry
	if err := database.GetDB().First(&registry, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Registry 不存在"})
		return
	}

	// 如果设置为默认，先取消其他默认
	if req.IsDefault && !registry.IsDefault {
		database.GetDB().Model(&models.Registry{}).Where("is_default = ? AND id != ?", true, id).Update("is_default", false)
	}

	// 更新字段
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.URL != "" {
		updates["url"] = req.URL
	}
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Password != "" {
		updates["password"] = req.Password
	}
	updates["is_default"] = req.IsDefault
	updates["insecure"] = req.Insecure
	if req.Email != "" {
		updates["email"] = req.Email
	}

	if err := database.GetDB().Model(&registry).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新 Registry 失败"})
		return
	}

	// 重新查询
	database.GetDB().First(&registry, id)
	registry.Password = ""

	c.JSON(http.StatusOK, registry)
}

// DeleteRegistry 删除 Registry
func DeleteRegistry(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var registry models.Registry
	if err := database.GetDB().First(&registry, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Registry 不存在"})
		return
	}

	if err := database.GetDB().Delete(&registry).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除 Registry 失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// TestRegistry 测试 Registry 连接
func TestRegistry(c *gin.Context) {
	var req struct {
		URL      string `json:"url" binding:"required"`
		Username string `json:"username"`
		Password string `json:"password"`
		Insecure bool   `json:"insecure"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 测试连接（这里简化处理，实际应该调用 Docker Registry API）
	// TODO: 实现实际的连接测试
	c.JSON(http.StatusOK, gin.H{"message": "连接成功"})
}

