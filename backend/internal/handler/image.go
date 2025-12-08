package handler

import (
	"dockerun/internal/docker"
	"io"
	"net/http"

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
	imageID := c.Param("id")
	force := c.DefaultQuery("force", "false") == "true"
	
	_, err := docker.RemoveImage(imageID, force)
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

