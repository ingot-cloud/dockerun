package handler

import (
	"dockerun/internal/docker"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListContainers 列出所有容器
func ListContainers(c *gin.Context) {
	showAll := c.DefaultQuery("all", "true") == "true"
	
	containers, err := docker.ListContainers(showAll)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, containers)
}

// InspectContainer 获取容器详情
func InspectContainer(c *gin.Context) {
	containerID := c.Param("id")
	
	container, err := docker.InspectContainer(containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "容器不存在"})
		return
	}

	c.JSON(http.StatusOK, container)
}

// StartContainer 启动容器
func StartContainer(c *gin.Context) {
	containerID := c.Param("id")
	
	if err := docker.StartContainer(containerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "容器启动成功"})
}

// StopContainer 停止容器
func StopContainer(c *gin.Context) {
	containerID := c.Param("id")
	
	if err := docker.StopContainer(containerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "容器停止成功"})
}

// RestartContainer 重启容器
func RestartContainer(c *gin.Context) {
	containerID := c.Param("id")
	
	if err := docker.RestartContainer(containerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "容器重启成功"})
}

// RemoveContainer 删除容器
func RemoveContainer(c *gin.Context) {
	containerID := c.Param("id")
	force := c.DefaultQuery("force", "false") == "true"
	
	if err := docker.RemoveContainer(containerID, force); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "容器删除成功"})
}

// GetContainerLogs 获取容器日志
func GetContainerLogs(c *gin.Context) {
	containerID := c.Param("id")
	tail := c.DefaultQuery("tail", "100")
	
	logs, err := docker.GetContainerLogs(containerID, tail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer logs.Close()

	c.Header("Content-Type", "text/plain; charset=utf-8")
	io.Copy(c.Writer, logs)
}

// GetContainerStats 获取容器统计信息
func GetContainerStats(c *gin.Context) {
	containerID := c.Param("id")
	
	stats, err := docker.GetContainerStats(containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

