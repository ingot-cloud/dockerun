package handler

import (
	"dockerun/internal/docker"
	"io"
	"net/http"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
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
	if tail != "all" {
		lines, err := strconv.Atoi(tail)
		if err != nil || lines < 1 || lines > 100000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "日志行数必须为 1 到 100000"})
			return
		}
	}
	info, err := docker.Client.ContainerInspect(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取容器信息"})
		return
	}

	logs, err := docker.Client.ContainerLogs(c.Request.Context(), containerID, container.LogsOptions{
		ShowStdout: true, ShowStderr: true, Tail: tail,
		Timestamps: c.DefaultQuery("timestamps", "true") == "true",
		Since:      c.Query("since"), Until: c.Query("until"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer logs.Close()

	c.Header("Content-Type", "text/plain; charset=utf-8")
	if info.Config != nil && info.Config.Tty {
		io.Copy(c.Writer, logs)
	} else {
		stdcopy.StdCopy(c.Writer, c.Writer, logs)
	}
}

// GetContainerStats 获取容器统计信息
func GetContainerStats(c *gin.Context) {
	containerID := c.Param("id")

	stats, err := docker.GetContainerStats(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
