package handler

import (
	"context"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
)

type containerWithImageReference struct {
	types.Container
	ImageReference string `json:"ImageReference,omitempty"`
}

// containerImageReferences 保留 Docker 原始字段，补充经过实际容器和镜像 ID 核对的名称。
func containerImageReferences(ctx context.Context, containers []types.Container) map[string]string {
	names := map[string]string{}
	unresolved := map[string]string{}
	for _, container := range containers {
		if container.Image != container.ImageID && container.Image != strings.TrimPrefix(container.ImageID, "sha256:") && !strings.HasPrefix(container.Image, "sha256:") {
			continue
		}
		if name := release.VerifiedImageReference(container.Labels[release.ImageReferenceLabel], container.Labels[release.ImageIDLabel], container.ImageID); name != "" {
			names[container.ID] = name
		} else if container.ImageID != "" {
			unresolved[container.ID] = container.ImageID
		}
	}
	if len(unresolved) == 0 {
		return names
	}
	// 修复前创建的容器没有名称标签，使用已核对的本机运行记录兼容，无需重新部署。
	if database.DB != nil {
		var states []models.TargetState
		if database.DB.Select("containers").Where("target_id = ?", 1).Find(&states).Error == nil {
			for _, state := range states {
				for _, container := range state.Containers {
					if name := release.VerifiedImageReference(container.Image, container.ImageID, unresolved[container.ID]); name != "" {
						names[container.ID] = name
						delete(unresolved, container.ID)
					}
				}
			}
		}
	}
	// 非受管容器只使用 Docker 核对到的唯一标签；多个标签时不猜测原始来源。
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	images := map[string]string{}
	for id, imageID := range unresolved {
		if ctx.Err() != nil {
			break
		}
		name, checked := images[imageID]
		if !checked {
			image, _, err := docker.GetClient().ImageInspectWithRaw(ctx, imageID)
			if err == nil && image.ID == imageID {
				var tags []string
				for _, tag := range image.RepoTags {
					if tag != "<none>:<none>" && release.VerifiedImageReference(tag, image.ID, imageID) != "" {
						tags = append(tags, tag)
					}
				}
				if len(tags) == 1 {
					name = tags[0]
				}
			}
			images[imageID] = name
		}
		if name != "" {
			names[id] = name
		}
	}
	return names
}
