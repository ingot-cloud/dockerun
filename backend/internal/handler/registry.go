package handler

import (
	"dockerun/internal/database"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"dockerun/internal/utils"
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
		PullUsername      string `json:"pull_username"`
		PullPassword      string `json:"pull_password"`
		PullOnlyConfirmed bool   `json:"pull_only_confirmed"`
		Name              string `json:"name" binding:"required"`
		URL               string `json:"url" binding:"required"`
		Username          string `json:"username"`
		Password          string `json:"password"`
		IsDefault         bool   `json:"is_default"`
		Insecure          bool   `json:"insecure"`
		Email             string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 如果设置为默认，先取消其他默认
	if req.IsDefault {
		database.GetDB().Model(&models.Registry{}).Where("is_default = ?", true).Update("is_default", false)
	}

	// 加密密码
	encryptedPassword := ""
	if req.Password != "" {
		encrypted, err := utils.SealCredential(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
			return
		}
		encryptedPassword = encrypted
	}

	pullPassword := ""
	if req.PullPassword != "" {
		var err error
		pullPassword, err = utils.SealCredential(req.PullPassword)
		if err != nil {
			fail(c, err)
			return
		}
	}
	registry := models.Registry{PullUsername: req.PullUsername, PullPassword: pullPassword, PullOnlyConfirmed: req.PullOnlyConfirmed,
		Kind:      "external",
		Name:      req.Name,
		URL:       req.URL,
		Username:  req.Username,
		Password:  encryptedPassword,
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
		PullUsername      string `json:"pull_username"`
		PullPassword      string `json:"pull_password"`
		PullOnlyConfirmed bool   `json:"pull_only_confirmed"`
		Name              string `json:"name"`
		URL               string `json:"url"`
		Username          string `json:"username"`
		Password          string `json:"password"`
		IsDefault         bool   `json:"is_default"`
		Insecure          bool   `json:"insecure"`
		Email             string `json:"email"`
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

	if registry.Kind == "builtin" {
		c.JSON(409, gin.H{"error": "内置仓库请通过系统设置管理，不能作为外部连接修改或删除"})
		return
	}

	// 如果设置为默认，先取消其他默认
	if req.IsDefault && !registry.IsDefault {
		database.GetDB().Model(&models.Registry{}).Where("is_default = ? AND id != ?", true, id).Update("is_default", false)
	}

	// 更新字段
	updates := map[string]interface{}{"pull_username": req.PullUsername, "pull_only_confirmed": req.PullOnlyConfirmed}
	if req.PullPassword != "" {
		p, err := utils.SealCredential(req.PullPassword)
		if err != nil {
			fail(c, err)
			return
		}
		updates["pull_password"] = p
	}
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
		// 加密密码
		encrypted, err := utils.SealCredential(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
			return
		}
		updates["password"] = encrypted
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

	if registry.Kind == "builtin" {
		c.JSON(409, gin.H{"error": "内置仓库请通过系统设置管理，不能作为外部连接修改或删除"})
		return
	}

	if err := database.GetDB().Delete(&models.Registry{}, id).Error; err != nil {
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

	if _, _, err := release.RegistryRequest(c.Request.Context(), models.Registry{URL: req.URL, Username: req.Username, Password: req.Password, Insecure: req.Insecure}, "GET", "/v2/"); err != nil {
		c.JSON(502, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "连接成功"})
}

// GetDecryptedPassword 获取解密后的密码
func GetDecryptedPassword(registry *models.Registry) string {
	if registry.Password == "" {
		return ""
	}

	// 尝试解密密码
	decrypted, err := utils.OpenCredential(registry.Password)
	if err != nil {
		// 如果解密失败，可能是旧数据（明文），直接返回
		return registry.Password
	}

	return decrypted
}
