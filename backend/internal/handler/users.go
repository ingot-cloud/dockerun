package handler

import (
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"strings"
	"sync"
	"time"
)

var userMu sync.Mutex

func ListUsers(c *gin.Context) {
	var users []models.User
	if err := database.DB.Order("id").Find(&users).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, users)
}
func CreateUser(c *gin.Context) {
	var q struct {
		Username       string `json:"username"`
		Password       string `json:"password"`
		IsAdmin        bool   `json:"is_admin"`
		RegistryAccess string `json:"registry_access"`
	}
	if c.ShouldBindJSON(&q) != nil || len(q.Username) < 1 || len(q.Username) > 100 || len(q.Password) < 6 || len(q.Password) > 72 {
		c.JSON(400, gin.H{"error": "请提供用户名及 6–72 字节密码"})
		return
	}
	if q.RegistryAccess == "" {
		q.RegistryAccess = "none"
	}
	if !validAccess(q.RegistryAccess) {
		c.JSON(400, gin.H{"error": "仓库权限无效"})
		return
	}
	u := models.User{Username: q.Username, IsAdmin: q.IsAdmin, RegistryAccess: q.RegistryAccess, Enabled: true, AuthVersion: 1}
	if err := u.HashPassword(q.Password); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := database.DB.Create(&u).Error; err != nil {
		c.JSON(409, gin.H{"error": "用户名已存在或无法保存"})
		return
	}
	identity.Audit(c.GetUint("user_id"), "user.create", fmt.Sprint(u.ID))
	c.JSON(201, u)
}
func validAccess(s string) bool { return s == "none" || s == "read_only" || s == "read_write" }
func UpdateUser(c *gin.Context) {
	var q struct {
		Enabled        *bool   `json:"enabled"`
		IsAdmin        *bool   `json:"is_admin"`
		RegistryAccess *string `json:"registry_access"`
		Password       string  `json:"password"`
	}
	if c.ShouldBindJSON(&q) != nil {
		c.JSON(400, gin.H{"error": "参数无效"})
		return
	}
	if q.RegistryAccess != nil && !validAccess(*q.RegistryAccess) || q.Password != "" && (len(q.Password) < 6 || len(q.Password) > 72) {
		c.JSON(400, gin.H{"error": "权限或密码无效"})
		return
	}
	userMu.Lock()
	defer userMu.Unlock()
	var u models.User
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&u, c.Param("id")).Error; err != nil {
			return err
		}
		enabled, admin := u.Enabled, u.IsAdmin
		if q.Enabled != nil {
			enabled = *q.Enabled
		}
		if q.IsAdmin != nil {
			admin = *q.IsAdmin
		}
		if u.Enabled && u.IsAdmin && (!enabled || !admin) {
			var n int64
			if err := tx.Model(&models.User{}).Where("enabled = ? AND is_admin = ?", true, true).Count(&n).Error; err != nil {
				return err
			}
			if n <= 1 {
				return errors.New("不能停用或降级最后一个可用管理员")
			}
		}
		updates := map[string]any{"enabled": enabled, "is_admin": admin, "auth_version": gorm.Expr("auth_version + 1")}
		if q.RegistryAccess != nil {
			updates["registry_access"] = *q.RegistryAccess
		}
		if q.Password != "" {
			if err := u.HashPassword(q.Password); err != nil {
				return err
			}
			updates["password"] = u.Password
		}
		return tx.Model(&u).Updates(updates).Error
	})
	if err != nil {
		c.JSON(409, gin.H{"error": err.Error()})
		return
	}
	identity.Audit(c.GetUint("user_id"), "user.authorization_or_password", fmt.Sprint(u.ID))
	database.DB.First(&u, u.ID)
	c.JSON(200, u)
}
func ListAudit(c *gin.Context) {
	var rows []models.AuditEvent
	q := database.DB.Order("id DESC").Limit(200)
	if a := c.Query("action"); a != "" {
		q = q.Where("action = ?", a)
	}
	q.Find(&rows)
	c.JSON(200, rows)
}
func ListTokens(c *gin.Context) {
	var rows []models.AutomationToken
	database.DB.Where("user_id = ?", c.GetUint("user_id")).Find(&rows)
	c.JSON(200, rows)
}
func CreateToken(c *gin.Context) {
	var q struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
		Days   int      `json:"days"`
	}
	if c.ShouldBindJSON(&q) != nil || q.Name == "" {
		c.JSON(400, gin.H{"error": "请填写令牌名称"})
		return
	}
	v, _ := c.Get("user")
	u := v.(*models.User)
	for _, s := range q.Scopes {
		if s != "pull" && s != "push" && s != "deploy" || s == "deploy" && !u.IsAdmin || s != "deploy" && !u.CanRegistry(s) {
			c.JSON(403, gin.H{"error": "令牌权限超出账号授权"})
			return
		}
	}
	if q.Days < 1 || q.Days > 90 {
		q.Days = 30
	}
	raw := "dku_" + identity.Random()
	t := models.AutomationToken{UserID: u.ID, Name: q.Name, Hash: identity.Hash(raw), Scopes: strings.Join(q.Scopes, " "), AuthVersion: u.AuthVersion, ExpiresAt: time.Now().Add(time.Duration(q.Days) * 24 * time.Hour)}
	if err := database.DB.Create(&t).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	identity.Audit(u.ID, "token.create", fmt.Sprint(t.ID))
	c.JSON(201, gin.H{"token": raw, "credential": t})
}
func RevokeToken(c *gin.Context) {
	database.DB.Model(&models.AutomationToken{}).Where("id = ? AND user_id = ?", c.Param("id"), c.GetUint("user_id")).Update("revoked", true)
	identity.Audit(c.GetUint("user_id"), "token.revoke", c.Param("id"))
	c.JSON(200, gin.H{"message": "已撤销"})
}
