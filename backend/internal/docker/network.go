package docker

import (
	"context"

	"github.com/docker/docker/api/types"
)

// ListNetworks 列出所有网络
func ListNetworks() ([]types.NetworkResource, error) {
	ctx := context.Background()
	return Client.NetworkList(ctx, types.NetworkListOptions{})
}

// InspectNetwork 获取网络详情
func InspectNetwork(networkID string) (types.NetworkResource, error) {
	ctx := context.Background()
	return Client.NetworkInspect(ctx, networkID, types.NetworkInspectOptions{})
}

// CreateNetwork 创建网络
func CreateNetwork(name string, driver string) (types.NetworkCreateResponse, error) {
	ctx := context.Background()
	return Client.NetworkCreate(ctx, name, types.NetworkCreate{
		Driver: driver,
	})
}

// RemoveNetwork 删除网络
func RemoveNetwork(networkID string) error {
	ctx := context.Background()
	return Client.NetworkRemove(ctx, networkID)
}

// ConnectNetwork 连接容器到网络
func ConnectNetwork(networkID, containerID string) error {
	ctx := context.Background()
	return Client.NetworkConnect(ctx, networkID, containerID, nil)
}

// DisconnectNetwork 断开容器与网络的连接
func DisconnectNetwork(networkID, containerID string, force bool) error {
	ctx := context.Background()
	return Client.NetworkDisconnect(ctx, networkID, containerID, force)
}

