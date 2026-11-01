package handler

import (
	"dockerun/internal/bundle"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
)

func prepareOfflineCommandRelease(r *models.Release) error {
	var material models.Bundle
	if database.DB.First(&material, r.BundleID).Error != nil || material.Status != "ready" {
		return fmt.Errorf("请先导入离线材料并等待校验完成")
	}
	var images []release.Image
	if material.Kind == "image" {
		if err := json.Unmarshal([]byte(material.Manifest), &images); err != nil {
			return fmt.Errorf("离线镜像清单无效")
		}
		r.Source = "local"
	} else {
		var manifest bundle.Manifest
		if err := json.Unmarshal([]byte(material.Manifest), &manifest); err != nil {
			return fmt.Errorf("完整发布包清单无效")
		}
		images = manifest.Images
		for index := range images {
			if images[index].Platform == "" {
				images[index].Platform = manifest.Platform
			}
		}
		r.Source = "bundle"
		if r.Version == "" {
			r.Version = manifest.Version
		}
	}
	if len(images) == 0 {
		return fmt.Errorf("离线材料未包含镜像")
	}
	platform := images[0].Platform
	if platform != "linux/amd64" && platform != "linux/arm64" {
		return fmt.Errorf("离线镜像仅支持 Linux amd64 或 arm64")
	}
	for _, image := range images {
		if image.Platform != platform || image.ID == "" {
			return fmt.Errorf("离线材料中的镜像必须具有相同平台及完整 Image ID")
		}
	}
	if r.Platform != "" && r.Platform != platform {
		return fmt.Errorf("所选平台与离线镜像平台不一致")
	}
	r.Platform, r.RegistryID, r.TriggerDigest = platform, 0, ""
	if len(images) == 1 {
		r.TriggerImage = images[0].Reference
	}
	encoded, _ := json.Marshal(images)
	r.Images = string(encoded)
	return release.OfflineCommandImages(*r)
}
