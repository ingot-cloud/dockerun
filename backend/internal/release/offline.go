package release

import (
	"context"
	"crypto/sha256"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LoadImageArchive 校验受管材料后加载，Docker 错误帧必须作为失败返回。
func LoadImageArchive(ctx context.Context, path, expectedSHA string, log func(string)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if expectedSHA != "" {
		hash := sha256.New()
		if _, err := io.Copy(hash, f); err != nil {
			return err
		}
		if hex.EncodeToString(hash.Sum(nil)) != expectedSHA {
			return fmt.Errorf("离线镜像归档校验失败")
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}
	response, err := docker.GetClient().ImageLoad(ctx, f, false)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return Consume(response.Body, log)
}

// LoadLocalMaterials 每次部署重新从保留材料准备镜像，支持历史版本在缓存被清理后重部署。
func LoadLocalMaterials(ctx context.Context, r models.Release, log func(string)) error {
	if r.BundleID == 0 {
		return nil
	}
	var material models.Bundle
	if err := database.DB.First(&material, r.BundleID).Error; err != nil || material.Status != "ready" {
		return fmt.Errorf("离线材料不可用，请重新导入")
	}
	platform, err := Platform(ctx)
	if err != nil {
		return err
	}
	if r.Platform != "" && platform != r.Platform {
		return fmt.Errorf("离线材料平台 %s 与目标主机 %s 不符", r.Platform, platform)
	}
	log("加载此版本保留的离线镜像；完成前保留当前服务")
	if material.Kind == "image" && r.Source == "local" {
		return LoadImageArchive(ctx, material.Path, material.SHA256, log)
	}
	if material.Kind == "image" || r.Source != "bundle" {
		return fmt.Errorf("离线版本与归档类型不匹配")
	}
	var images []Image
	if err := json.Unmarshal([]byte(r.Images), &images); err != nil || len(images) == 0 {
		return fmt.Errorf("离线版本缺少镜像清单")
	}
	root := filepath.Join(config.Load().DataDir, "releases", fmt.Sprintf("bundle-%d", material.ID))
	for _, image := range images {
		path, err := SafePath(root, image.Archive)
		if err != nil || image.SHA256 == "" {
			return fmt.Errorf("离线镜像归档元数据无效")
		}
		if err := LoadImageArchive(ctx, path, image.SHA256, log); err != nil {
			return err
		}
	}
	return nil
}

// OfflineCommandImages 供创建版本时校验，主镜像通过固定 ID 传入脚本。
func OfflineCommandImages(r models.Release) error {
	_, err := commandImageEnvironment(r)
	return err
}
