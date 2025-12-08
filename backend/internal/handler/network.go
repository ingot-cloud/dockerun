package handler

import (
	"dockerun/internal/docker"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListNetworks 列出所有网络
func ListNetworks(c *gin.Context) {
	networks, err := docker.ListNetworks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, networks)
}

// InspectNetwork 获取网络详情
func InspectNetwork(c *gin.Context) {
	networkID := c.Param("id")
	
	network, err := docker.InspectNetwork(networkID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "网络不存在"})
		return
	}

	c.JSON(http.StatusOK, network)
}

// CreateNetwork 创建网络
func CreateNetwork(c *gin.Context) {
	var req docker.NetworkCreateRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "网络名称不能为空"})
		return
	}

	if req.Driver == "" {
		req.Driver = "bridge"
	}

	response, err := docker.CreateNetworkWithConfig(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// RemoveNetwork 删除网络
func RemoveNetwork(c *gin.Context) {
	networkID := c.Param("id")
	
	if err := docker.RemoveNetwork(networkID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "网络删除成功"})
}

// ConnectNetwork 连接容器到网络
func ConnectNetwork(c *gin.Context) {
	networkID := c.Param("id")
	
	var req struct {
		ContainerID string `json:"container_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	if err := docker.ConnectNetwork(networkID, req.ContainerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "连接成功"})
}

// DisconnectNetwork 断开容器与网络的连接
func DisconnectNetwork(c *gin.Context) {
	networkID := c.Param("id")
	
	var req struct {
		ContainerID string `json:"container_id" binding:"required"`
		Force       bool   `json:"force"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	if err := docker.DisconnectNetwork(networkID, req.ContainerID, req.Force); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "断开连接成功"})
}

