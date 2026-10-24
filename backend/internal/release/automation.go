package release

import (
	"dockerun/internal/database"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"github.com/distribution/reference"
)

// DefaultRegistry 只为尚未选择来源的发布配置补充内置仓库，不改写明确的外部来源。
func DefaultRegistry(id uint) (models.Registry, error) {
	var reg models.Registry
	query := database.DB
	if id == 0 {
		query = query.Where("kind = ?", "builtin")
	} else {
		query = query.Where("id = ?", id)
	}
	if err := query.First(&reg).Error; err != nil {
		return reg, fmt.Errorf("请选择可用的镜像仓库")
	}
	return reg, nil
}

func AutomationBase(env models.Environment) (models.Release, error) {
	var base models.Release
	if env.CurrentReleaseID == 0 || database.DB.Where("id = ? AND application_id = ?", env.CurrentReleaseID, env.ApplicationID).First(&base).Error != nil {
		return base, fmt.Errorf("请先在此环境完成首次手动发布，配置镜像、端口、挂载和脚本后再开启自动发布")
	}
	if base.Source != "registry" || base.RegistryID != env.RegistryID {
		return base, fmt.Errorf("请先使用所选仓库在此环境手动发布一次，再开启自动发布")
	}
	var images []Image
	if json.Unmarshal([]byte(base.Images), &images) == nil {
		for _, image := range images {
			if n, err := reference.ParseNormalizedNamed(image.Reference); err == nil && reference.Path(n) == env.Repository {
				return base, nil
			}
		}
	}
	return base, fmt.Errorf("自动发布的镜像名称必须包含在此环境当前成功版本中")
}
