package release

import (
	"context"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"github.com/distribution/reference"
	"net/url"
	"os"
	"regexp"
	"strings"
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
	var app models.Application
	if err := database.DB.First(&app, env.ApplicationID).Error; err != nil {
		return models.Release{}, fmt.Errorf("应用不存在")
	}
	return AutomationDefinition(app, env)
}

var commandImageVariable = regexp.MustCompile(`\$(DOCKERUN_IMAGE\b|\{DOCKERUN_IMAGE[}:?+%-])`)

// AutomationDefinition 优先使用明确保存的应用配置，历史安装回退到环境当前成功版本。
// 此处仅校验定义；保存配置不会访问远端仓库、创建版本或执行部署。
func AutomationDefinition(app models.Application, env models.Environment) (models.Release, error) {
	definition := app.Definition
	if definition.Execution.Mode == "" && strings.TrimSpace(definition.Compose) == "" {
		return legacyAutomationBase(env)
	}
	r := models.Release{ApplicationID: app.ID, Execution: definition.Execution, Compose: definition.Compose,
		Configs: stringJSON(definition.Configs), Platform: definition.Platform, Source: "registry", RegistryID: env.RegistryID, Scripts: app.Scripts}
	if err := ValidateExecution(r.Execution); err != nil {
		return r, err
	}
	if IsCommand(r) {
		if !commandImageVariable.MatchString(r.Execution.Command) {
			return r, fmt.Errorf("命令自动发布需在已保存的部署命令中使用 $DOCKERUN_IMAGE，可用 ${DOCKERUN_IMAGE:-$IMAGE} 兼容手动发布")
		}
		return r, nil
	}
	if definition.Source == "local" || definition.Source == "bundle" {
		return r, fmt.Errorf("自动发布需使用仓库镜像，请先保存在线部署配置")
	}
	if definition.Source == "registry" && definition.RegistryID != 0 && definition.RegistryID != env.RegistryID {
		return r, fmt.Errorf("自动发布的仓库须与已保存部署配置中的仓库一致")
	}
	dir, err := os.MkdirTemp("", "dockerun-automation-")
	if err != nil {
		return r, err
	}
	defer os.RemoveAll(dir)
	if err = WriteConfigs(dir, definition.Configs); err != nil {
		return r, err
	}
	p, err := Parse(context.Background(), r.Compose, dir, ProjectName(env), env.Parameters)
	if err != nil {
		return r, fmt.Errorf("已保存的部署配置无效：%w", err)
	}
	r.Compose, err = FreezeImages(r.Compose, p)
	if err != nil {
		return r, err
	}
	reg, err := DefaultRegistry(env.RegistryID)
	if err != nil {
		return r, err
	}
	host, err := automaticRegistryHost(reg)
	if err != nil {
		return r, err
	}
	var images []Image
	matched := false
	for _, ref := range References(p) {
		n, err := reference.ParseNormalizedNamed(ref)
		if err != nil {
			return r, err
		}
		if reference.Domain(n) != host && !(reference.Domain(n) == "docker.io" && host == "registry-1.docker.io") {
			return r, fmt.Errorf("镜像 %s 必须包含监听仓库地址 %s", ref, host)
		}
		matched = matched || reference.Path(n) == env.Repository
		images = append(images, Image{Reference: ref})
	}
	if !matched {
		return r, fmt.Errorf("监听的镜像名称必须包含在已保存的部署配置中")
	}
	r.Images = stringJSON(images)
	return r, nil
}

func legacyAutomationBase(env models.Environment) (models.Release, error) {
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

// automaticRelease 固定触发镜像的摘要，其他镜像在创建版本时解析；不改写已有版本。
func automaticRelease(ctx context.Context, base models.Release, env models.Environment, event models.ImageEvent) (models.Release, error) {
	reg, err := DefaultRegistry(env.RegistryID)
	if err != nil {
		return base, err
	}
	host, err := automaticRegistryHost(reg)
	if err != nil {
		return base, err
	}
	var images []Image
	if IsCommand(base) {
		images = []Image{{Reference: host + "/" + event.Repository + ":" + event.Reference, Digest: event.Digest}}
	} else {
		if err := json.Unmarshal([]byte(base.Images), &images); err != nil {
			return base, err
		}
		matched := false
		for i, image := range images {
			n, err := reference.ParseNormalizedNamed(image.Reference)
			if err != nil {
				return base, err
			}
			if reference.Domain(n) != host && !(reference.Domain(n) == "docker.io" && host == "registry-1.docker.io") {
				return base, fmt.Errorf("镜像 %s 必须包含监听仓库地址 %s", image.Reference, host)
			}
			if reference.Path(n) == event.Repository {
				images[i].Digest, images[i].ID = event.Digest, ""
				matched = true
			} else if image.Digest == "" {
				images[i], err = Resolve(ctx, reg, image.Reference)
				if err != nil {
					return base, err
				}
			}
		}
		if !matched {
			return base, fmt.Errorf("部署配置未包含触发镜像")
		}
	}
	base.ID = 0
	base.Version = fmt.Sprintf("image-event-%d-env-%d", event.ID, env.ID)
	base.PublisherID, base.RegistryID = event.UserID, event.RegistryID
	base.Source, base.BundleID, base.Retained = "registry", 0, true
	base.Images = stringJSON(images)
	base.TriggerImage = host + "/" + event.Repository + ":" + event.Reference
	base.TriggerDigest = event.Digest
	return base, nil
}

// SameAutomaticImage 同 tag 且同摘要才跳过；不同 tag 即使内容相同也重新部署。
func SameAutomaticImage(env models.Environment, event models.ImageEvent) (bool, error) {
	same := func(r models.Release) bool {
		if r.RegistryID != event.RegistryID {
			return false
		}
		match := func(ref, d string) bool {
			n, err := reference.ParseNormalizedNamed(ref)
			if err != nil || d != event.Digest || reference.Path(n) != event.Repository {
				return false
			}
			tagged, ok := n.(reference.Tagged)
			return ok && tagged.Tag() == event.Reference
		}
		if r.TriggerImage != "" {
			return match(r.TriggerImage, r.TriggerDigest)
		}
		var images []Image
		json.Unmarshal([]byte(r.Images), &images)
		for _, im := range images {
			if match(im.Reference, im.Digest) {
				return true
			}
		}
		return false
	}
	var active []models.Deployment
	if err := database.DB.Where("environment_id = ? AND status IN ?", env.ID, []string{"queued", "running", "blocked"}).Order("id DESC").Find(&active).Error; err != nil {
		return false, err
	}
	latest := map[uint]uint{}
	for _, task := range active {
		if _, exists := latest[task.TargetID]; !exists {
			latest[task.TargetID] = task.ReleaseID
		}
	}
	targets, err := environmentTargets(env)
	if err != nil {
		return false, err
	}
	for _, target := range targets {
		current := env.CurrentReleaseID
		if pending, exists := latest[target]; exists {
			// 只比较最新排队意图；当前是 A、正在部署 B 时再次推送 A 仍需排队部署 A。
			current = pending
		} else {
			var state models.TargetState
			if database.DB.Where("environment_id = ? AND target_id = ?", env.ID, target).First(&state).Error == nil {
				current = state.CurrentReleaseID
			}
		}
		if current == 0 {
			return false, nil
		}
		var r models.Release
		if database.DB.First(&r, current).Error != nil || !same(r) {
			return false, nil
		}
	}
	return true, nil
}

func automaticRegistryHost(reg models.Registry) (string, error) {
	registryURL := reg.URL
	if reg.Kind == "builtin" {
		registryURL = config.Load().PublicURL
	}
	u, err := url.Parse(registryURL)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("镜像仓库地址无效")
	}
	return u.Host, nil
}
