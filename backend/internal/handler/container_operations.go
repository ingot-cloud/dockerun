package handler

import (
	"context"
	"dockerun/internal/docker"
	"net/http"
	"strings"
	"time"

	"github.com/distribution/reference"
	"github.com/docker/docker/api/types/container"
	"github.com/gin-gonic/gin"
)

func containerOperation(c *gin.Context, operation func(context.Context, string) error, message string) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	if err := operation(ctx, c.Param("id")); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": message})
}
func KillContainer(c *gin.Context) {
	containerOperation(c, func(ctx context.Context, id string) error { return docker.Client.ContainerKill(ctx, id, "SIGKILL") }, "容器已强制停止")
}
func PauseContainer(c *gin.Context) {
	containerOperation(c, docker.Client.ContainerPause, "容器已暂停")
}
func ResumeContainer(c *gin.Context) {
	containerOperation(c, docker.Client.ContainerUnpause, "容器已恢复")
}
func RenameContainer(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(400, gin.H{"error": "请填写容器名称"})
		return
	}
	containerOperation(c, func(ctx context.Context, id string) error {
		return docker.Client.ContainerRename(ctx, id, strings.TrimSpace(req.Name))
	}, "容器名称已更新")
}
func UpdateContainerRestart(c *gin.Context) {
	var req struct {
		Name              string `json:"name"`
		MaximumRetryCount int    `json:"maximum_retry_count"`
	}
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(400, gin.H{"error": "重启策略参数无效"})
		return
	}
	switch req.Name {
	case "no", "always", "unless-stopped", "on-failure":
	default:
		c.JSON(400, gin.H{"error": "不支持的重启策略"})
		return
	}
	if req.MaximumRetryCount < 0 || (req.Name != "on-failure" && req.MaximumRetryCount != 0) {
		c.JSON(400, gin.H{"error": "最大重试次数无效"})
		return
	}
	containerOperation(c, func(ctx context.Context, id string) error {
		_, err := docker.Client.ContainerUpdate(ctx, id, container.UpdateConfig{RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyMode(req.Name), MaximumRetryCount: req.MaximumRetryCount}})
		return err
	}, "重启策略已更新")
}
func CommitContainer(c *gin.Context) {
	var req struct {
		Image string `json:"image" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(400, gin.H{"error": "请填写镜像名称"})
		return
	}
	ref, err := reference.ParseNormalizedNamed(strings.TrimSpace(req.Image))
	if err != nil {
		c.JSON(400, gin.H{"error": "镜像名称格式无效"})
		return
	}
	if _, digest := ref.(reference.Canonical); digest {
		c.JSON(400, gin.H{"error": "请填写镜像标签，不能使用摘要"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()
	result, err := docker.Client.ContainerCommit(ctx, c.Param("id"), container.CommitOptions{Reference: reference.FamiliarString(reference.TagNameOnly(ref)), Pause: true})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
func GetContainerProcesses(c *gin.Context) {
	result, err := docker.Client.ContainerTop(c.Request.Context(), c.Param("id"), nil)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, result)
}
