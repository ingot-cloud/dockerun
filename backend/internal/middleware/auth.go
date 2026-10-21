package middleware

import (
	"dockerun/internal/identity"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		u, _, err := identity.Validate(raw, "web")
		if strings.HasPrefix(raw, "dku_") {
			var scopes []string
			u, scopes, err = identity.Automation(raw)
			c.Set("token_scopes", scopes)
			c.Set("automation", true)
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "登录已失效"})
			return
		}
		if c.GetBool("automation") {
			p := c.Request.URL.Path
			if strings.HasPrefix(p, "/api/auth/") && p != "/api/auth/me" {
				c.AbortWithStatusJSON(403, gin.H{"error": "自动化令牌不能管理账号或签发其他令牌"})
				return
			}
			if strings.HasPrefix(p, "/api/builtin/") {
				scopes, _ := c.Get("token_scopes")
				v, _ := scopes.([]string)
				if !identity.Has(v, "pull") {
					c.AbortWithStatus(403)
					return
				}
			}
		}
		c.Set("user_id", u.ID)
		c.Set("username", u.Username)
		c.Set("is_admin", u.IsAdmin)
		c.Set("user", u)
		c.Next()
	}
}
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !c.GetBool("is_admin") {
			c.AbortWithStatusJSON(403, gin.H{"error": "此操作需要管理员权限"})
			return
		}
		if c.GetBool("automation") {
			scopes, _ := c.Get("token_scopes")
			s, _ := scopes.([]string)
			if !identity.Has(s, "deploy") {
				c.AbortWithStatusJSON(403, gin.H{"error": "令牌未授权部署操作"})
				return
			}
			p := c.Request.URL.Path
			if !(strings.HasPrefix(p, "/api/applications") || strings.HasPrefix(p, "/api/environments") || strings.HasPrefix(p, "/api/releases") || strings.HasPrefix(p, "/api/deployments") || strings.HasPrefix(p, "/api/bundles")) {
				c.AbortWithStatusJSON(403, gin.H{"error": "请使用网页登录执行系统管理"})
				return
			}
		}
		c.Next()
	}
}
