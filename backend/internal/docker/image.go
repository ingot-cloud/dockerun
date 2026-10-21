package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

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

// ImageHistory 返回镜像的真实构建历史（包括不产生文件层的配置指令）。
func ImageHistory(ctx context.Context, imageID string) ([]image.HistoryResponseItem, error) {
	return Client.ImageHistory(ctx, imageID)
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
	return CheckStream(resp.Body)
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
	return CheckStream(resp.Body)
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
	return SaveImagesToReader([]string{imageID})
}

// SaveImagesToReader 将所选镜像保存到同一归档，并保留全部标签。
func SaveImagesToReader(imageIDs []string) (io.ReadCloser, error) {
	if len(imageIDs) == 0 {
		return nil, fmt.Errorf("请选择需要导出的镜像")
	}
	ctx := context.Background()
	var references []string
	seen := make(map[string]bool)
	for _, id := range imageIDs {
		inspect, _, err := Client.ImageInspectWithRaw(ctx, id)
		if err != nil {
			return nil, err
		}
		var tags []string
		for _, tag := range inspect.RepoTags {
			if tag != "" && !strings.HasSuffix(tag, ":<none>") {
				tags = append(tags, tag)
			}
		}
		// Docker 按 ID 保存时不携带 RepoTags；命名镜像必须传入标签。
		if len(tags) == 0 {
			tags = []string{inspect.ID}
		}
		for _, ref := range tags {
			if !seen[ref] {
				references = append(references, ref)
				seen[ref] = true
			}
		}
	}
	return Client.ImageSave(ctx, references)
}

func CheckStream(reader io.Reader) error {
	d := json.NewDecoder(reader)
	for {
		var v struct {
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
		}
		err := d.Decode(&v)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if v.Error != "" {
			return fmt.Errorf("%s", v.Error)
		}
		if v.ErrorDetail.Message != "" {
			return fmt.Errorf("%s", v.ErrorDetail.Message)
		}
	}
}
