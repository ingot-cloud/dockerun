package handler

import (
	"context"
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func BuiltinCatalog(c *gin.Context) {
	v, _ := c.Get("user")
	u := v.(*models.User)
	if !u.CanRegistry("pull") {
		c.JSON(403, gin.H{"error": "此账号尚未获得仓库权限，请联系管理员"})
		return
	}
	if builtin.Engine == nil {
		c.JSON(503, gin.H{"error": "内置仓库未启用"})
		return
	}
	token, _ := identity.Issue(u, "registry", []string{"registry:catalog:*"}, 5*time.Minute)
	req := httptest.NewRequest("GET", "/v2/_catalog?n=1000", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	builtin.Serve(w, req)
	var catalog any
	json.Unmarshal(w.Body.Bytes(), &catalog)
	var events []models.ImageEvent
	database.DB.Where("committed = ?", true).Order("id DESC").Limit(500).Find(&events)
	c.JSON(w.Code, gin.H{"catalog": catalog, "events": events, "public_url": config.Load().PublicURL, "read_only": builtin.ReadOnly(), "permission": u.RegistryAccess})
}
func BuiltinTags(c *gin.Context) {
	v, _ := c.Get("user")
	u := v.(*models.User)
	if !u.CanRegistry("pull") {
		c.Status(403)
		return
	}
	var reg models.Registry
	database.DB.Where("kind = ?", "builtin").First(&reg)
	name := c.Query("name")
	path := "/v2/" + name + "/tags/list"
	if ref := c.Query("reference"); ref != "" {
		path = "/v2/" + name + "/manifests/" + url.PathEscape(ref)
	}
	data, headers, err := release.RegistryRequest(c.Request.Context(), reg, "GET", path)
	if err != nil {
		fail(c, err)
		return
	}
	if c.Query("reference") != "" {
		var versions []uint
		database.DB.Model(&models.Release{}).Where("images LIKE ?", "%"+headers.Get("Docker-Content-Digest")+"%").Pluck("id", &versions)
		c.JSON(200, gin.H{"digest": headers.Get("Docker-Content-Digest"), "manifest": json.RawMessage(data), "release_ids": versions})
		return
	}
	c.Header("Docker-Content-Digest", headers.Get("Docker-Content-Digest"))
	c.Data(200, "application/json", data)
}
func RegistryMaintenance(c *gin.Context) {
	var q struct {
		ReadOnly bool `json:"read_only"`
		GC       bool `json:"gc"`
	}
	if c.ShouldBindJSON(&q) != nil {
		c.Status(400)
		return
	}
	builtin.SetReadOnly(q.ReadOnly)
	if q.GC {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Hour)
		defer cancel()
		if err := builtin.GarbageCollect(ctx); err != nil {
			fail(c, err)
			return
		}
	}
	identity.Audit(c.GetUint("user_id"), "registry.maintenance", fmt.Sprintf("readonly=%t gc=%t", q.ReadOnly, q.GC))
	c.JSON(200, gin.H{"read_only": builtin.ReadOnly()})
}
func BuiltinStatus(c *gin.Context) {
	var bytes int64
	filepath.WalkDir(filepath.Join(config.Load().DataDir, "registry"), func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			info, e := d.Info()
			if e == nil {
				bytes += info.Size()
			}
		}
		return nil
	})
	c.JSON(200, gin.H{"enabled": builtin.Engine != nil, "read_only": builtin.ReadOnly(), "public_url": config.Load().PublicURL, "storage_bytes": bytes, "tls": config.Load().TLSCert != ""})
}

func BuiltinDelete(c *gin.Context) {
	var q struct {
		Name   string `json:"name"`
		Digest string `json:"digest"`
	}
	if c.ShouldBindJSON(&q) != nil || q.Name == "" {
		c.Status(400)
		return
	}
	v, _ := c.Get("user")
	u := v.(*models.User)
	token, err := identity.Issue(u, "registry", []string{"repository:" + q.Name + ":delete"}, 5*time.Minute)
	if err != nil {
		fail(c, err)
		return
	}
	req := httptest.NewRequest("DELETE", "/v2/"+q.Name+"/manifests/"+q.Digest, nil).WithContext(c.Request.Context())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	builtin.Serve(w, req)
	if w.Code != 202 {
		c.JSON(w.Code, gin.H{"error": w.Body.String()})
		return
	}
	c.JSON(200, gin.H{"message": "已删除未受保护的 manifest"})
}
