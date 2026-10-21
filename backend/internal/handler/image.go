package handler

import (
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"io"
	"net/http"
	"strings"

	"github.com/docker/docker/errdefs"
	"github.com/gin-gonic/gin"
)

// ListImages 列出所有镜像
func ListImages(c *gin.Context) {
	images, err := docker.ListImages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, images)
}

// InspectImage 获取镜像详情
func InspectImage(c *gin.Context) {
	imageID := c.Param("id")

	image, err := docker.InspectImage(imageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "镜像不存在"})
		return
	}

	c.JSON(http.StatusOK, image)
}

// GetImageHistory 获取镜像构建层历史。
func GetImageHistory(c *gin.Context) {
	history, err := docker.ImageHistory(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errdefs.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "镜像不存在或已被删除"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取镜像层历史：" + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, history)
}

// PullImage 拉取镜像
func PullImage(c *gin.Context) {
	var req struct {
		ImageName string `json:"image_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	reader, err := docker.PullImage(req.ImageName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	// 流式返回拉取进度
	c.Header("Content-Type", "application/json")
	io.Copy(c.Writer, reader)
}

// RemoveImage 删除镜像
func RemoveImage(c *gin.Context) {
	if !release.ExecutionGate.TryLock() {
		c.JSON(409, gin.H{"error": "发布或备份正在执行，请稍后清理本机镜像"})
		return
	}
	defer release.ExecutionGate.Unlock()
	imageID := c.Param("id")
	image, err := docker.InspectImage(imageID)
	if err != nil {
		c.JSON(404, gin.H{"error": "镜像不存在"})
		return
	}
	var n int64
	database.DB.Model(&models.Release{}).Where("retained = ? AND (images LIKE ? OR images LIKE ?)", true, "%"+imageID+"%", "%"+image.ID+"%").Count(&n)
	var snapshots int64
	database.DB.Model(&models.Deployment{}).Where("release_id IN (?) AND snapshot LIKE ?", database.DB.Model(&models.Release{}).Select("id").Where("retained = ?", true), "%"+image.ID+"%").Count(&snapshots)
	n += snapshots
	if n > 0 {
		c.JSON(409, gin.H{"error": "镜像仍被保留版本引用"})
		return
	}
	force := c.DefaultQuery("force", "false") == "true"

	_, err = docker.RemoveImage(imageID, force)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "镜像删除成功"})
}

// TagImage 给镜像打标签
func TagImage(c *gin.Context) {
	var req struct {
		Source string `json:"source" binding:"required"`
		Target string `json:"target" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	if err := docker.TagImage(req.Source, req.Target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "标签创建成功"})
}

// LoadImage 从文件导入镜像
func LoadImage(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	if err := docker.LoadImage(req.Path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "镜像导入成功"})
}

// SaveImage 导出镜像到文件
func SaveImage(c *gin.Context) {
	var req struct {
		ImageID string `json:"image_id" binding:"required"`
		Path    string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	if err := docker.SaveImage(req.ImageID, req.Path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "镜像导出成功"})
}

// ImportImage 通过文件上传导入镜像（SSE 流式返回日志）
func ImportImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}

	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取文件"})
		return
	}
	defer src.Close()

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// 发送开始事件
	c.SSEvent("log", gin.H{"message": "正在导入镜像…"})
	c.Writer.Flush()

	// 导入镜像并获取响应
	reader, err := docker.LoadImageWithProgress(src)
	if err != nil {
		c.SSEvent("error", gin.H{"message": err.Error()})
		c.Writer.Flush()
		return
	}
	defer reader.Close()

	if err := release.Consume(reader, func(message string) { c.SSEvent("log", gin.H{"message": message}); c.Writer.Flush() }); err != nil {
		c.SSEvent("error", gin.H{"message": err.Error()})
		c.Writer.Flush()
		return
	}

	// 发送完成事件
	c.SSEvent("complete", gin.H{"message": "镜像导入完成"})
	c.Writer.Flush()
}

// ExportImage 导出镜像并下载
func ExportImage(c *gin.Context) {
	exportImages(c, []string{c.Param("id")})
}

// ExportImages 将多个镜像导出为一个 tar，避免浏览器拦截连续下载。
func ExportImages(c *gin.Context) {
	var req struct {
		ImageIDs []string `json:"image_ids" binding:"required,min=1,dive,required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择需要导出的镜像"})
		return
	}
	ids := make([]string, 0, len(req.ImageIDs))
	seen := make(map[string]bool)
	for _, id := range req.ImageIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "镜像 ID 不能为空"})
			return
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	exportImages(c, ids)
}

func exportImages(c *gin.Context, imageIDs []string) {
	// 获取镜像导出流
	reader, err := docker.SaveImagesToReader(imageIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	// 设置响应头
	c.Header("Content-Type", "application/x-tar")
	c.Header("Content-Disposition", "attachment; filename=images.tar")

	// 流式传输
	io.Copy(c.Writer, reader)
}
