package docker

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

// IsContainerRunning 检查容器是否正在运行
func IsContainerRunning(containerID string) bool {
	if Client == nil {
		return false
	}

	ctx := context.Background()
	inspect, err := Client.ContainerInspect(ctx, containerID)
	if err != nil {
		return false
	}

	return inspect.State.Running
}

// GetContainerIDByName 根据容器名称获取容器ID
func GetContainerIDByName(name string) string {
	if Client == nil {
		return ""
	}

	ctx := context.Background()
	
	// 清理名称（去掉可能的前缀）
	name = strings.TrimPrefix(name, "/")
	
	// 创建过滤器
	filterArgs := filters.NewArgs()
	filterArgs.Add("name", name)
	
	containers, err := Client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	
	if err != nil || len(containers) == 0 {
		return ""
	}
	
	// 返回第一个匹配的容器ID
	return containers[0].ID
}

// GetContainerStatus 获取容器状态信息
func GetContainerStatus(containerID string) (string, error) {
	if Client == nil {
		return "", nil
	}

	ctx := context.Background()
	inspect, err := Client.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}

	return inspect.State.Status, nil
}

// GetContainerName 获取容器名称
func GetContainerName(containerID string) string {
	if Client == nil {
		return ""
	}

	ctx := context.Background()
	inspect, err := Client.ContainerInspect(ctx, containerID)
	if err != nil {
		return ""
	}

	// 容器名称是数组，取第一个，并去掉前导的 "/"
	if len(inspect.Name) > 0 {
		return strings.TrimPrefix(inspect.Name, "/")
	}

	return ""
}

