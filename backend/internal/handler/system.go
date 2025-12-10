package handler

import (
	"dockerun/internal/docker"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CheckDockerStatus 检查Docker状态
func CheckDockerStatus(c *gin.Context) {
	available := docker.IsAvailable()
	c.JSON(http.StatusOK, gin.H{
		"available": available,
	})
}

// GetSystemInfo 获取系统信息和Docker版本
func GetSystemInfo(c *gin.Context) {
	info, err := docker.GetDockerInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}
