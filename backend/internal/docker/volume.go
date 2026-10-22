package docker

import (
	"context"

	"github.com/docker/docker/api/types/volume"
)

// ListVolumes 列出所有卷
func ListVolumes() (volume.ListResponse, error) {
	ctx := context.Background()
	return Client.VolumeList(ctx, volume.ListOptions{})
}

// InspectVolume 获取卷详情
func InspectVolume(volumeName string) (volume.Volume, error) {
	ctx := context.Background()
	return Client.VolumeInspect(ctx, volumeName)
}

// CreateVolume 创建卷
func CreateVolume(name string, driver string, labels, options map[string]string) (volume.Volume, error) {
	ctx := context.Background()
	return Client.VolumeCreate(ctx, volume.CreateOptions{
		Name:       name,
		Driver:     driver,
		Labels:     labels,
		DriverOpts: options,
	})
}

// RemoveVolume 删除卷
func RemoveVolume(volumeName string, force bool) error {
	ctx := context.Background()
	return Client.VolumeRemove(ctx, volumeName, force)
}
