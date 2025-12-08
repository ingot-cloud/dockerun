package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
)

// ListImages 列出所有镜像
func ListImages() ([]image.Summary, error) {
	ctx := context.Background()
	return Client.ImageList(ctx, image.ListOptions{All: true})
}

// InspectImage 获取镜像详情
func InspectImage(imageID string) (types.ImageInspect, error) {
	ctx := context.Background()
	inspect, _, err := Client.ImageInspectWithRaw(ctx, imageID)
	return inspect, err
}

// PullImage 拉取镜像
func PullImage(imageName string) (io.ReadCloser, error) {
	ctx := context.Background()
	return Client.ImagePull(ctx, imageName, image.PullOptions{})
}

// RemoveImage 删除镜像
func RemoveImage(imageID string, force bool) ([]image.DeleteResponse, error) {
	ctx := context.Background()
	return Client.ImageRemove(ctx, imageID, image.RemoveOptions{Force: force})
}

// TagImage 给镜像打标签
func TagImage(source, target string) error {
	ctx := context.Background()
	return Client.ImageTag(ctx, source, target)
}

