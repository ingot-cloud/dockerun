package docker

import (
	"context"
	"dockerun/internal/config"
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
)

// InitializeNetwork 初始化 Docker 网络
func InitializeNetwork() error {
	if Client == nil {
		return fmt.Errorf("docker client not initialized")
	}

	cfg := config.Load()
	ctx := context.Background()

	// 检查网络是否已存在
	networkList, err := Client.NetworkList(ctx, types.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", cfg.NetworkName)),
	})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	// 如果网络已存在，直接返回
	for _, net := range networkList {
		if net.Name == cfg.NetworkName {
			log.Printf("Network '%s' already exists, skipping creation\n", cfg.NetworkName)
			return nil
		}
	}

	// 创建网络
	log.Printf("Creating network '%s'...\n", cfg.NetworkName)

	// 构建 IPAM 配置
	ipamConfig := network.IPAMConfig{}
	if cfg.NetworkSubnet != "" {
		ipamConfig.Subnet = cfg.NetworkSubnet
	}
	if cfg.NetworkGateway != "" {
		ipamConfig.Gateway = cfg.NetworkGateway
	}

	createOptions := types.NetworkCreate{
		Driver: cfg.NetworkDriver,
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{ipamConfig},
		},
		CheckDuplicate: true,
	}

	resp, err := Client.NetworkCreate(ctx, cfg.NetworkName, createOptions)
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	log.Printf("Network '%s' created successfully (ID: %s)\n", cfg.NetworkName, resp.ID)
	return nil
}

