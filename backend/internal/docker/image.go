package docker

import (
	"context"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
)

// ListImages 列出所有镜像
func ListImages() ([]image.Summary, error) {
	ctx := context.Background()
	return Client.ImageList(ctx, types.ImageListOptions{All: true})
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
	return Client.ImagePull(ctx, imageName, types.ImagePullOptions{})
}

// RemoveImage 删除镜像
func RemoveImage(imageID string, force bool) ([]image.DeleteResponse, error) {
	ctx := context.Background()
	return Client.ImageRemove(ctx, imageID, types.ImageRemoveOptions{Force: force})
}

// TagImage 给镜像打标签
func TagImage(source, target string) error {
	ctx := context.Background()
	return Client.ImageTag(ctx, source, target)
}

// LoadImage 从文件导入镜像
func LoadImage(path string) error {
	ctx := context.Background()
	
	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// 加载镜像
	resp, err := Client.ImageLoad(ctx, file, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// 读取响应（确保完成）
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}

// SaveImage 导出镜像到文件
func SaveImage(imageID, path string) error {
	ctx := context.Background()
	
	// 保存镜像
	reader, err := Client.ImageSave(ctx, []string{imageID})
	if err != nil {
		return err
	}
	defer reader.Close()
	
	// 创建文件
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// 写入文件
	_, err = io.Copy(file, reader)
	return err
}

// LoadImageFromReader 从 io.Reader 导入镜像
func LoadImageFromReader(reader io.Reader) error {
	ctx := context.Background()
	
	// 加载镜像
	resp, err := Client.ImageLoad(ctx, reader, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// 读取响应（确保完成）
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}

// LoadImageWithProgress 从 io.Reader 导入镜像并返回进度流
func LoadImageWithProgress(reader io.Reader) (io.ReadCloser, error) {
	ctx := context.Background()
	
	// 加载镜像
	resp, err := Client.ImageLoad(ctx, reader, false)
	if err != nil {
		return nil, err
	}
	
	// 返回响应体，调用者负责关闭
	return resp.Body, nil
}

// SaveImageToReader 导出镜像到 io.ReadCloser
func SaveImageToReader(imageID string) (io.ReadCloser, error) {
	ctx := context.Background()
	return Client.ImageSave(ctx, []string{imageID})
}

