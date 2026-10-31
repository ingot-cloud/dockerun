package handler

import (
	"context"
	"dockerun/internal/builtin"
	"dockerun/internal/bundle"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var uploadMu sync.Mutex
var uploadID = regexp.MustCompile(`^[a-f0-9]{64}$`)

func UploadBundle(c *gin.Context) {
	id := c.Param("upload")
	if !uploadID.MatchString(id) {
		fail(c, fmt.Errorf("上传 ID 应为文件 SHA-256"))
		return
	}
	path := filepath.Join(config.Load().DataDir, "uploads", id+".part")
	uploadMu.Lock()
	defer uploadMu.Unlock()
	offset := int64(0)
	if st, err := os.Stat(path); err == nil {
		offset = st.Size()
	}
	if c.Request.Method == "GET" {
		c.JSON(200, gin.H{"offset": offset})
		return
	}
	var start, end, total int64
	if _, err := fmt.Sscanf(c.GetHeader("Content-Range"), "bytes %d-%d/%d", &start, &end, &total); err != nil || start != offset || end < start || end >= total || total > bundle.MaxSize || end-start+1 > 16<<20 {
		c.JSON(409, gin.H{"error": "分块范围或偏移无效", "offset": offset})
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		fail(c, err)
		return
	}
	defer f.Close()
	f.Seek(offset, 0)
	n, err := io.Copy(f, io.LimitReader(c.Request.Body, end-start+2))
	if err != nil || n != end-start+1 {
		f.Truncate(offset)
		fail(c, fmt.Errorf("分块长度不符"))
		return
	}
	if err = f.Sync(); err != nil {
		f.Truncate(offset)
		fail(c, err)
		return
	}
	c.JSON(200, gin.H{"offset": offset + n, "total": total})
}
func FinishUpload(c *gin.Context) {
	id := c.Param("upload")
	if !uploadID.MatchString(id) {
		c.Status(400)
		return
	}
	uploadMu.Lock()
	defer uploadMu.Unlock()
	var existing models.Bundle
	if database.DB.Where("sha256 = ?", id).First(&existing).Error == nil {
		c.JSON(202, existing)
		return
	}
	path := filepath.Join(config.Load().DataDir, "uploads", id+".part")
	hash, err := bundle.HashFile(path)
	if err != nil || hash != id {
		fail(c, fmt.Errorf("上传未完成或文件 SHA-256 不符"))
		return
	}
	var q struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	}
	c.ShouldBindJSON(&q)
	if q.Kind == "" {
		q.Kind = "bundle"
	}
	if q.Kind != "bundle" && q.Kind != "image" {
		fail(c, fmt.Errorf("不支持的材料类型"))
		return
	}
	final := filepath.Join(config.Load().DataDir, "uploads", id+".tar.gz")
	if err = os.Rename(path, final); err != nil {
		fail(c, err)
		return
	}
	path = final
	b, err := stageBundle(path, id, c.GetUint("user_id"), q.Kind)
	if err != nil {
		fail(c, err)
		return
	}
	if q.Name != "" {
		b.Name = filepath.Base(q.Name)
		database.DB.Model(b).Update("name", b.Name)
	}
	c.JSON(202, b)
}
func ImportBundleDirectory(c *gin.Context) {
	var q struct {
		File string `json:"file"`
		Kind string `json:"kind"`
	}
	if c.ShouldBindJSON(&q) != nil {
		c.Status(400)
		return
	}
	path, err := release.SafePath(config.Load().ImportDir, q.File)
	if err != nil {
		fail(c, err)
		return
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		fail(c, err)
		return
	}
	root, _ := filepath.Abs(config.Load().ImportDir)
	real, _ = filepath.Abs(real)
	rel, _ := filepath.Rel(root, real)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		fail(c, fmt.Errorf("仅允许导入指定目录中的文件"))
		return
	}
	info, err := os.Stat(real)
	if err != nil || !info.Mode().IsRegular() || info.Size() > bundle.MaxSize {
		fail(c, fmt.Errorf("导入文件类型或大小无效"))
		return
	}
	if q.Kind == "" {
		q.Kind = "bundle"
	}
	if q.Kind != "bundle" && q.Kind != "image" {
		c.Status(400)
		return
	}
	hash, err := bundle.HashFile(real)
	if err != nil {
		fail(c, err)
		return
	}
	var existing models.Bundle
	if database.DB.Where("sha256 = ?", hash).First(&existing).Error == nil {
		c.JSON(202, existing)
		return
	}
	uploadMu.Lock()
	defer uploadMu.Unlock()
	dst := filepath.Join(config.Load().DataDir, "uploads", hash+".part")
	src, err := os.Open(real)
	if err != nil {
		fail(c, err)
		return
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		fail(c, err)
		return
	}
	n, copyErr := io.Copy(out, io.LimitReader(src, bundle.MaxSize+1))
	err = copyErr
	if n > bundle.MaxSize {
		err = fmt.Errorf("导入材料超过大小限制")
	}
	out.Close()
	if err != nil {
		fail(c, err)
		return
	}
	copied, err := bundle.HashFile(dst)
	if err != nil || copied != hash {
		fail(c, fmt.Errorf("导入期间源文件变化或写入校验失败"))
		return
	}
	final := filepath.Join(config.Load().DataDir, "uploads", hash+".tar.gz")
	if err = os.Rename(dst, final); err != nil {
		fail(c, err)
		return
	}
	b, err := stageBundle(final, hash, c.GetUint("user_id"), q.Kind)
	if err != nil {
		fail(c, err)
		return
	}
	b.Name = filepath.Base(q.File)
	database.DB.Model(b).Update("name", b.Name)
	c.JSON(202, b)
}
func stageBundle(path, hash string, uid uint, kind string) (*models.Bundle, error) {
	var b models.Bundle
	if database.DB.Where("sha256 = ?", hash).First(&b).Error == nil {
		return &b, nil
	}
	b = models.Bundle{Kind: kind, Name: hash[:12] + ".tar.gz", SHA256: hash, Status: "queued", Path: path, UserID: uid}
	if err := database.DB.Create(&b).Error; err != nil {
		return nil, err
	}
	identity.Audit(uid, "bundle.import", fmt.Sprint(b.ID))
	go processBundle(b)
	return &b, nil
}
func ResumeBundleImports() {
	var rows []models.Bundle
	database.DB.Where("status IN ?", []string{"queued", "validating", "loading"}).Find(&rows)
	for _, b := range rows {
		go processBundle(b)
	}
}
func processBundle(b models.Bundle) {
	b.LocalLoaded = false
	b.LocalLoadNote = ""
	release.ExecutionGate.RLock()
	defer release.ExecutionGate.RUnlock()
	if b.Kind == "image" {
		processImageArchive(b)
		return
	}
	database.DB.Model(&b).Updates(map[string]any{"status": "validating", "error": ""})
	dir := filepath.Join(config.Load().DataDir, "releases", "bundle-"+strconv.FormatUint(uint64(b.ID), 10))
	os.RemoveAll(dir)
	err := os.MkdirAll(dir, 0700)
	var m *bundle.Manifest
	if err == nil {
		m, err = bundle.Extract(b.Path, dir)
	}
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
		platform, platformErr := release.Platform(ctx)
		if platformErr == nil && platform == m.Platform {
			database.DB.Model(&b).Update("status", "loading")
			err = bundle.Load(ctx, dir, *m, func(s string) {})
			b.LocalLoaded = err == nil
		} else {
			b.LocalLoadNote = "材料已校验，须由目标平台节点加载；控制台本机状态不代表其他节点就绪"
		}
		cancel()
	}

	if err != nil {
		database.DB.Model(&b).Updates(map[string]any{"status": "failed", "error": err.Error()})
		return
	}
	data, _ := json.Marshal(m)
	database.DB.Model(&b).Updates(map[string]any{"status": "ready", "manifest": string(data), "local_loaded": b.LocalLoaded, "local_load_note": b.LocalLoadNote})
	identity.Audit(b.UserID, "bundle.ready", fmt.Sprint(b.ID))
}
func ListBundles(c *gin.Context) {
	var rows []models.Bundle
	database.DB.Order("id DESC").Find(&rows)
	c.JSON(200, rows)
}
func BundleVersion(c *gin.Context) {
	builtin.ReferenceGate.RLock()
	defer builtin.ReferenceGate.RUnlock()
	var q struct {
		ApplicationID uint `json:"application_id"`
	}
	if c.ShouldBindJSON(&q) != nil {
		c.Status(400)
		return
	}
	var b models.Bundle
	if database.DB.First(&b, c.Param("id")).Error != nil || b.Status != "ready" || b.Kind == "image" {
		fail(c, fmt.Errorf("发布包尚未准备完成"))
		return
	}
	var m bundle.Manifest
	if err := json.Unmarshal([]byte(b.Manifest), &m); err != nil {
		fail(c, err)
		return
	}
	if q.ApplicationID == 0 {
		var a models.Application
		if database.DB.Where("name = ?", m.Application).First(&a).Error != nil {
			a = models.Application{Name: m.Application, Kind: "compose"}
			if err := database.DB.Create(&a).Error; err != nil {
				fail(c, err)
				return
			}
		}
		q.ApplicationID = a.ID
	}
	var a models.Application
	if database.DB.First(&a, q.ApplicationID).Error != nil {
		fail(c, fmt.Errorf("应用不存在"))
		return
	}
	dir := filepath.Join(config.Load().DataDir, "releases", fmt.Sprintf("bundle-%d", b.ID))
	content, err := os.ReadFile(filepath.Join(dir, "compose.yaml"))
	if err != nil {
		fail(c, err)
		return
	}
	definition, err := release.Parse(c.Request.Context(), string(content), dir, "dockerun-package", m.Defaults)
	if err != nil {
		fail(c, err)
		return
	}
	frozen, err := release.FreezeImages(string(content), definition)
	if err != nil {
		fail(c, err)
		return
	}
	content = []byte(frozen)
	configs, err := bundle.Configs(dir)
	if err != nil {
		fail(c, err)
		return
	}
	images, _ := json.Marshal(m.Images)
	conf, _ := json.Marshal(configs)
	r := models.Release{ApplicationID: q.ApplicationID, Version: m.Version, Source: "bundle", BundleID: b.ID, Platform: m.Platform, Compose: string(content), Images: string(images), Configs: string(conf), PublisherID: c.GetUint("user_id"), GitCommit: m.GitCommit, Retained: true, Scripts: a.Scripts}
	// 当前请求已持有引用锁，避免重复获取读锁。
	if err := database.DB.Create(&r).Error; err != nil {
		fail(c, err)
		return
	}
	// No deployment or environment parameter updates are performed by importing a bundle.
	c.JSON(201, gin.H{"release": r, "defaults": m.Defaults, "message": "已创建待发布版本，请在目标环境确认参数后部署"})
}
func DownloadBundle(c *gin.Context) {
	var b models.Bundle
	if database.DB.First(&b, c.Param("id")).Error != nil {
		c.Status(404)
		return
	}
	c.FileAttachment(b.Path, b.Name)
}

func processImageArchive(b models.Bundle) {
	database.DB.Model(&b).Updates(map[string]any{"status": "loading", "error": ""})
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	f, err := os.Open(b.Path)
	if err != nil {
		database.DB.Model(&b).Updates(map[string]any{"status": "failed", "error": err.Error()})
		return
	}
	defer f.Close()
	resp, err := docker.GetClient().ImageLoad(ctx, f, false)
	refs := []string{}
	if err == nil {
		err = release.Consume(resp.Body, func(m string) {
			for _, line := range strings.Split(m, "\n") {
				for _, prefix := range []string{"Loaded image: ", "Loaded image ID: "} {
					if strings.HasPrefix(line, prefix) {
						refs = append(refs, strings.TrimSpace(strings.TrimPrefix(line, prefix)))
					}
				}
			}
		})
		resp.Body.Close()
	}
	var images []release.Image
	if err == nil {
		if len(refs) == 0 {
			err = fmt.Errorf("Docker 未返回可验证的镜像身份")
		}
		for _, ref := range refs {
			v, _, e := docker.GetClient().ImageInspectWithRaw(ctx, ref)
			if e != nil {
				err = e
				break
			}
			images = append(images, release.Image{Reference: ref, ID: v.ID, Platform: v.Os + "/" + v.Architecture})
		}
	}
	if err != nil {
		database.DB.Model(&b).Updates(map[string]any{"status": "failed", "error": err.Error()})
		return
	}
	b.LocalLoaded = true
	data, _ := json.Marshal(images)
	database.DB.Model(&b).Updates(map[string]any{"status": "ready", "manifest": string(data)})
	identity.Audit(b.UserID, "image.import", fmt.Sprint(b.ID))
}

func RetryBundle(c *gin.Context) {
	builtin.ReferenceGate.RLock()
	defer builtin.ReferenceGate.RUnlock()
	var b models.Bundle
	if database.DB.First(&b, c.Param("id")).Error != nil {
		c.Status(404)
		return
	}
	res := database.DB.Model(&b).Where("status = ?", "failed").Updates(map[string]any{"status": "queued", "error": ""})
	if res.RowsAffected != 1 {
		c.JSON(409, gin.H{"error": "仅失败的导入可以重试"})
		return
	}
	go processBundle(b)
	c.JSON(202, gin.H{"id": b.ID, "status": "queued"})
}
