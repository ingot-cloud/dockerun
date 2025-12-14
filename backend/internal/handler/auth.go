package handler

import (
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/middleware"
	"dockerun/internal/models"
	"dockerun/internal/utils"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type SetPasswordRequest struct {
	NewPassword        string `json:"new_password" binding:"required,min=6"`
	DeployWorkDir      string `json:"deploy_work_dir"`                      // 部署脚本工作目录
	MaxScriptExecutors int    `json:"max_script_executors"`                 // 最大并发脚本执行数
	ScriptTimeout      int    `json:"script_timeout"`                       // 脚本超时时间（秒）
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// Login 登录
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	var user models.User
	if err := database.GetDB().Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 检查密码是否为空（首次登录）
	if user.Password == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":          "需要设置初始密码",
			"require_setup":  true,
		})
		return
	}

	// 解密前端传来的加密密码
	decryptedPassword, err := utils.DecryptPassword(req.Password)
	if err != nil {
		log.Printf("解密密码失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码格式错误"})
		return
	}

	// 验证解密后的密码
	if !user.CheckPassword(decryptedPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 生成JWT token
	token, err := generateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"is_admin": user.IsAdmin,
		},
	})
}

// SetInitialPassword 设置初始密码和系统配置（首次运行）
func SetInitialPassword(c *gin.Context) {
	var req SetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 检查是否已有管理员设置了密码
	var admin models.User
	if err := database.GetDB().Where("username = ? AND is_admin = ?", "admin", true).First(&admin).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "管理员账户不存在"})
		return
	}

	if admin.Password != "" {
		// 密码已经设置过，需要使用修改密码接口
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码已设置，请使用修改密码功能"})
		return
	}

	// 解密前端传来的加密密码
	decryptedPassword, err := utils.DecryptPassword(req.NewPassword)
	if err != nil {
		log.Printf("解密密码失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码格式错误"})
		return
	}

	// 设置新密码（使用解密后的密码）
	if err := admin.HashPassword(decryptedPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	if err := database.GetDB().Save(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存密码失败"})
		return
	}

	// 更新系统配置
	var systemConfig models.SystemConfig
	if err := database.GetDB().First(&systemConfig).Error; err == nil {
		// 如果提供了配置参数，则更新
		if req.DeployWorkDir != "" {
			systemConfig.DeployWorkDir = req.DeployWorkDir
		}
		if req.MaxScriptExecutors > 0 {
			systemConfig.MaxScriptExecutors = req.MaxScriptExecutors
		}
		if req.ScriptTimeout > 0 {
			systemConfig.ScriptTimeout = req.ScriptTimeout
		}
		database.GetDB().Save(&systemConfig)
	}

	c.JSON(http.StatusOK, gin.H{"message": "初始化设置成功"})
}

// CheckInitialSetup 检查是否需要初始设置
func CheckInitialSetup(c *gin.Context) {
	var admin models.User
	if err := database.GetDB().Where("username = ? AND is_admin = ?", "admin", true).First(&admin).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"require_setup": true})
		return
	}

	requireSetup := admin.Password == ""
	c.JSON(http.StatusOK, gin.H{"require_setup": requireSetup})
}

// ChangePassword 修改密码
func ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	userID := c.GetUint("user_id")
	var user models.User
	if err := database.GetDB().First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 解密旧密码
	decryptedOldPassword, err := utils.DecryptPassword(req.OldPassword)
	if err != nil {
		log.Printf("解密旧密码失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码格式错误"})
		return
	}

	// 验证旧密码
	if !user.CheckPassword(decryptedOldPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "原密码错误"})
		return
	}

	// 解密新密码
	decryptedNewPassword, err := utils.DecryptPassword(req.NewPassword)
	if err != nil {
		log.Printf("解密新密码失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码格式错误"})
		return
	}

	// 设置新密码
	if err := user.HashPassword(decryptedNewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	if err := database.GetDB().Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存密码失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}

// GetCurrentUser 获取当前用户信息
func GetCurrentUser(c *gin.Context) {
	userID := c.GetUint("user_id")
	var user models.User
	if err := database.GetDB().First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"is_admin": user.IsAdmin,
	})
}

// generateToken 生成JWT token
func generateToken(user *models.User) (string, error) {
	cfg := config.Load()
	claims := &middleware.Claims{
		UserID:   user.ID,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

