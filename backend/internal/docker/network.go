package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
)

// NetworkCreateRequest 创建网络的请求参数
type NetworkCreateRequest struct {
	Name       string                       `json:"name"`
	Driver     string                       `json:"driver"`
	Options    map[string]string            `json:"options"`
	Internal   bool                         `json:"internal"`
	Attachable bool                         `json:"attachable"`
	EnableIPv6 bool                         `json:"enable_ipv6"`
	IPAM       *network.IPAM                `json:"ipam"`
	Labels     map[string]string            `json:"labels"`
}

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

// CreateNetworkWithConfig 使用完整配置创建网络
func CreateNetworkWithConfig(req NetworkCreateRequest) (types.NetworkCreateResponse, error) {
	ctx := context.Background()
	
	createOptions := types.NetworkCreate{
		Driver:     req.Driver,
		Options:    req.Options,
		Internal:   req.Internal,
		Attachable: req.Attachable,
		EnableIPv6: req.EnableIPv6,
		Labels:     req.Labels,
	}
	
	// 如果提供了 IPAM 配置
	if req.IPAM != nil {
		createOptions.IPAM = req.IPAM
	}
	
	return Client.NetworkCreate(ctx, req.Name, createOptions)
}

