package handler

import (
	"dockerun/internal/docker"
	"net/http"
	"strings"

	"github.com/docker/docker/errdefs"
	"github.com/gin-gonic/gin"
)

// ListVolumes 列出所有卷
func ListVolumes(c *gin.Context) {
	volumes, err := docker.ListVolumes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, volumes)
}

// InspectVolume 获取卷详情
func InspectVolume(c *gin.Context) {
	volumeName := c.Param("name")

	volume, err := docker.InspectVolume(volumeName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "卷不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, volume)
}

// CreateVolume 创建卷
func CreateVolume(c *gin.Context) {
	var req struct {
		Name    string            `json:"name"`
		Driver  string            `json:"driver"`
		Labels  map[string]string `json:"labels"`
		Options map[string]string `json:"options"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Driver = strings.TrimSpace(req.Driver)
	if req.Driver == "" {
		req.Driver = "local"
	}

	volume, err := docker.CreateVolume(req.Name, req.Driver, req.Labels, req.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, volume)
}

// RemoveVolume 删除卷
func RemoveVolume(c *gin.Context) {
	volumeName := c.Param("name")
	force := c.DefaultQuery("force", "false") == "true"

	if err := docker.RemoveVolume(volumeName, force); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "卷删除成功"})
}
