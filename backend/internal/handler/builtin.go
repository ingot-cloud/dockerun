package handler

import (
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"github.com/distribution/reference"
	"github.com/gin-gonic/gin"
	digest "github.com/opencontainers/go-digest"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	repositories := []string{}
	last := ""
	for {
		path := fmt.Sprintf("/v2/_catalog?n=%d&last=%s", builtin.CatalogPageSize, url.QueryEscape(last))
		w, err := builtinRequest(c, u, "registry:catalog:*", http.MethodGet, path)
		if err != nil {
			fail(c, err)
			return
		}
		if w.Code != http.StatusOK {
			builtinError(c, w)
			return
		}
		var page struct {
			Repositories []string `json:"repositories"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
			c.JSON(502, gin.H{"error": "仓库返回了无效的镜像列表"})
			return
		}
		repositories = append(repositories, page.Repositories...)
		if w.Header().Get("Link") == "" {
			break
		}
		if len(page.Repositories) == 0 || page.Repositories[len(page.Repositories)-1] <= last {
			c.JSON(502, gin.H{"error": "仓库目录分页未能继续，请重试"})
			return
		}
		last = page.Repositories[len(page.Repositories)-1]
	}
	events := []models.ImageEvent{}
	if err := database.DB.WithContext(c.Request.Context()).Where("committed = ?", true).Order("id DESC").Limit(500).Find(&events).Error; err != nil {
		fail(c, err)
		return
	}
	c.JSON(200, gin.H{"catalog": gin.H{"repositories": repositories}, "events": events, "public_url": config.Load().PublicURL, "read_only": builtin.ReadOnly(), "permission": u.RegistryAccess})
}
func BuiltinTags(c *gin.Context) {
	v, _ := c.Get("user")
	u := v.(*models.User)
	if !u.CanRegistry("pull") {
		c.JSON(403, gin.H{"error": "此账号尚未获得仓库权限，请联系管理员"})
		return
	}
	name := c.Query("name")
	named, err := reference.WithName(name)
	if err != nil {
		c.JSON(400, gin.H{"error": "请提供有效的镜像名称"})
		return
	}
	path := "/v2/" + name + "/tags/list"
	if ref := c.Query("reference"); ref != "" {
		if _, err := reference.WithTag(named, ref); err != nil {
			if _, err := digest.Parse(ref); err != nil {
				c.JSON(400, gin.H{"error": "请提供有效的镜像标签或 digest"})
				return
			}
		}
		path = "/v2/" + name + "/manifests/" + url.PathEscape(ref)
	}
	w, err := builtinRequest(c, u, "repository:"+name+":pull", http.MethodGet, path)
	if err != nil {
		fail(c, err)
		return
	}
	if w.Code != http.StatusOK {
		builtinError(c, w)
		return
	}
	data, headers := w.Body.Bytes(), w.Header()
	if c.Query("reference") != "" {
		versions := []uint{}
		if err := database.DB.WithContext(c.Request.Context()).Model(&models.Release{}).Where("images LIKE ?", "%"+headers.Get("Docker-Content-Digest")+"%").Pluck("id", &versions).Error; err != nil {
			fail(c, err)
			return
		}
		c.JSON(200, gin.H{"digest": headers.Get("Docker-Content-Digest"), "manifest": json.RawMessage(data), "release_ids": versions})
		return
	}
	c.Header("Docker-Content-Digest", headers.Get("Docker-Content-Digest"))
	c.Data(200, "application/json", data)
}
func RegistryMaintenance(c *gin.Context) {
	var request struct {
		ReadOnly *bool `json:"read_only"`
		GC       bool  `json:"gc"`
	}
	if c.ShouldBindJSON(&request) != nil || (request.ReadOnly == nil && !request.GC) || (request.GC && request.ReadOnly != nil && !*request.ReadOnly) {
		c.JSON(400, gin.H{"error": "请指定只读维护状态；清理任务不能同时关闭只读维护"})
		return
	}
	maintenanceActionMu.Lock()
	defer maintenanceActionMu.Unlock()
	if request.GC {
		if builtin.Engine == nil {
			c.JSON(503, gin.H{"error": "内置仓库未启用"})
			return
		}
		if !builtin.MaintenanceState().ManualReadOnly {
			c.JSON(409, gin.H{"error": "请先开启只读维护，再创建镜像层清理任务"})
			return
		}
		enqueueMaintenance(c, "registry_gc")
		return
	}
	if !*request.ReadOnly {
		var pending int64
		if err := database.DB.WithContext(c.Request.Context()).Model(&models.MaintenanceJob{}).Where("kind = ? AND status IN ?", "registry_gc", []string{"queued", "running"}).Count(&pending).Error; err != nil {
			c.JSON(500, gin.H{"error": "无法确认维护任务状态，请重试"})
			return
		}
		if pending > 0 {
			c.JSON(409, gin.H{"error": "清理任务正在排队或执行，请等待完成后再结束只读维护"})
			return
		}
	}
	if builtin.Engine == nil {
		c.JSON(503, gin.H{"error": "内置仓库未启用"})
		return
	}
	if err := builtin.SetReadOnlyContext(c.Request.Context(), *request.ReadOnly); err != nil {
		c.JSON(409, gin.H{"error": err.Error()})
		return
	}
	identity.Audit(c.GetUint("user_id"), "registry.maintenance", fmt.Sprintf("readonly=%t", *request.ReadOnly))
	c.JSON(200, builtin.MaintenanceState())
}
func BuiltinStatus(c *gin.Context) {
	var bytes int64
	cfg := config.Load()
	state := builtin.MaintenanceState()
	err := filepath.WalkDir(filepath.Join(cfg.DataDir, "registry"), func(path string, d os.DirEntry, err error) error {
		if c.Request.Context().Err() != nil {
			return c.Request.Context().Err()
		}
		// GC can remove a file between WalkDir and Info. That file no longer counts.
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if os.IsNotExist(err) {
				return nil
			}
			if err != nil {
				return err
			}
			bytes += info.Size()
		}
		return nil
	})
	result := gin.H{"enabled": builtin.Engine != nil, "read_only": state.ReadOnly, "manual_read_only": state.ManualReadOnly, "operation": state.Operation, "public_url": cfg.PublicURL, "engine": "Distribution v3.1.2", "storage_bytes": bytes, "tls": cfg.TLSCert != ""}
	if err != nil {
		result["storage_bytes"] = nil
		result["storage_error"] = "无法读取仓库存储占用，请检查数据目录访问权限后重试"
	}
	c.JSON(200, result)
}

func BuiltinDelete(c *gin.Context) {
	var q struct {
		Name   string `json:"name"`
		Digest string `json:"digest"`
	}
	if c.ShouldBindJSON(&q) != nil || q.Name == "" {
		c.JSON(400, gin.H{"error": "请提供镜像名称和 digest"})
		return
	}
	v, _ := c.Get("user")
	u := v.(*models.User)
	if !u.IsAdmin {
		c.JSON(403, gin.H{"error": "只有管理员可以删除镜像版本"})
		return
	}
	if _, err := reference.WithName(q.Name); err != nil {
		c.JSON(400, gin.H{"error": "请提供有效的镜像名称"})
		return
	}
	if _, err := digest.Parse(q.Digest); err != nil {
		c.JSON(400, gin.H{"error": "请提供有效的镜像 digest"})
		return
	}
	w, err := builtinRequest(c, u, "repository:"+q.Name+":delete", http.MethodDelete, "/v2/"+q.Name+"/manifests/"+q.Digest)
	if err != nil {
		fail(c, err)
		return
	}
	if w.Code != 202 {
		builtinError(c, w)
		return
	}
	c.JSON(200, gin.H{"message": "已删除未受保护的 manifest"})
}

func builtinRequest(c *gin.Context, u *models.User, scope, method, path string) (*httptest.ResponseRecorder, error) {
	token, err := identity.Issue(u, "registry", []string{scope}, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), method, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", release.AcceptManifest)
	w := httptest.NewRecorder()
	builtin.Serve(w, req)
	return w, nil
}

func builtinError(c *gin.Context, w *httptest.ResponseRecorder) {
	message := strings.TrimSpace(w.Body.String())
	var response struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(w.Body.Bytes(), &response) == nil && len(response.Errors) > 0 {
		messages := []string{}
		for _, err := range response.Errors {
			messages = append(messages, err.Message)
		}
		message = strings.Join(messages, "；")
	}
	if message == "" {
		message = fmt.Sprintf("仓库请求失败（%d），请重试", w.Code)
	}
	c.JSON(w.Code, gin.H{"error": message})
}
