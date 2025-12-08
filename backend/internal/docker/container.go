package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

// ListContainers 列出所有容器
func ListContainers(showAll bool) ([]types.Container, error) {
	ctx := context.Background()
	return Client.ContainerList(ctx, container.ListOptions{All: showAll})
}

// InspectContainer 获取容器详情
func InspectContainer(containerID string) (types.ContainerJSON, error) {
	ctx := context.Background()
	return Client.ContainerInspect(ctx, containerID)
}

// StartContainer 启动容器
func StartContainer(containerID string) error {
	ctx := context.Background()
	return Client.ContainerStart(ctx, containerID, container.StartOptions{})
}

// StopContainer 停止容器
func StopContainer(containerID string) error {
	ctx := context.Background()
	timeout := 10 // 10秒超时
	return Client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

// RestartContainer 重启容器
func RestartContainer(containerID string) error {
	ctx := context.Background()
	timeout := 10
	return Client.ContainerRestart(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

// RemoveContainer 删除容器
func RemoveContainer(containerID string, force bool) error {
	ctx := context.Background()
	return Client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: force})
}

// GetContainerLogs 获取容器日志
func GetContainerLogs(containerID string, tail string) (io.ReadCloser, error) {
	ctx := context.Background()
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	}
	return Client.ContainerLogs(ctx, containerID, options)
}

// GetContainerStats 获取容器统计信息
func GetContainerStats(containerID string) (types.ContainerStats, error) {
	ctx := context.Background()
	stats, err := Client.ContainerStats(ctx, containerID, false)
	return stats, err
}

