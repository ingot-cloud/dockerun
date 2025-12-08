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

// GetSystemInfo 获取系统信息
func GetSystemInfo(c *gin.Context) {
	info, err := docker.CheckSystemInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

// InstallDocker 安装Docker
func InstallDocker(c *gin.Context) {
	var req struct {
		Method       string `json:"method" binding:"required"` // online, offline-rpm, offline-deb, offline-binary
		PackagePath  string `json:"package_path"`              // 离线包路径
		Distribution string `json:"distribution"`              // Linux发行版
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	var err error
	switch req.Method {
	case "online":
		err = docker.InstallDockerOnline(req.Distribution)
	case "offline-rpm":
		err = docker.InstallDockerOfflineRPM(req.PackagePath)
	case "offline-deb":
		err = docker.InstallDockerOfflineDEB(req.PackagePath)
	case "offline-binary":
		err = docker.InstallDockerOfflineBinary(req.PackagePath)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的安装方法"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Docker安装成功"})
}

