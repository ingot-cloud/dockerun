package release

import (
	"context"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/errdefs"
)

type ContainerPolicy struct {
	Names   []string `json:"names"`
	Replace bool     `json:"replace"`
}

var containerName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

func ContainerSettings(env models.Environment) (ContainerPolicy, error) {
	p := ContainerPolicy{Names: []string{}, Replace: env.ReplaceContainers}
	if len(env.ContainerNames) > 100 {
		return p, fmt.Errorf("关联容器不能超过 100 个")
	}
	for _, value := range env.ContainerNames {
		name := os.Expand(strings.TrimSpace(value), func(key string) string { return env.Parameters[key] })
		if !containerName.MatchString(name) || len(name) > 255 {
			return p, fmt.Errorf("关联容器名称无效：%s；可引用已保存的环境参数", value)
		}
		if slices.Contains(p.Names, name) {
			return p, fmt.Errorf("关联容器名称重复：%s", name)
		}
		p.Names = append(p.Names, name)
	}
	if p.Replace && len(p.Names) == 0 {
		return p, fmt.Errorf("开启自动替换前，请指定此环境关联的容器名称")
	}
	slices.Sort(p.Names)
	return p, nil
}

func environmentTargets(env models.Environment) ([]uint, error) {
	var ids []uint
	if env.TargetIDs != "" {
		if err := json.Unmarshal([]byte(env.TargetIDs), &ids); err != nil {
			return nil, err
		}
	}
	if len(ids) == 0 {
		id := env.TargetID
		if id == 0 {
			id = 1
		}
		ids = []uint{id}
	}
	for i, id := range ids {
		if id == 0 {
			ids[i] = 1
		}
	}
	return ids, nil
}

// ValidateContainerBindings 防止同一主机的容器被多个环境同时认领。
func ValidateContainerBindings(env models.Environment) error {
	p, err := ContainerSettings(env)
	if err != nil {
		return err
	}
	if len(p.Names) == 0 {
		return nil
	}
	targets, err := environmentTargets(env)
	if err != nil {
		return fmt.Errorf("部署目标格式无效")
	}
	var others []models.Environment
	if err := database.DB.Where("id <> ?", env.ID).Find(&others).Error; err != nil {
		return err
	}
	for _, other := range others {
		op, e := ContainerSettings(other)
		if e != nil || len(op.Names) == 0 {
			continue
		}
		ot, e := environmentTargets(other)
		if e != nil {
			continue
		}
		for _, target := range targets {
			if slices.Contains(ot, target) {
				for _, name := range p.Names {
					if slices.Contains(op.Names, name) {
						return fmt.Errorf("容器 %s 已关联环境「%s」，同一主机不能重复关联", name, other.Name)
					}
				}
			}
		}
	}
	var pending []models.Deployment
	if err := database.DB.Where("environment_id <> ? AND target_id IN ? AND status IN ?", env.ID, targets, []string{"queued", "running", "blocked"}).Find(&pending).Error; err != nil {
		return err
	}
	for _, task := range pending {
		var snapshot Snapshot
		if json.Unmarshal([]byte(task.Snapshot), &snapshot) != nil {
			continue
		}
		policy, e := ContainerSettings(snapshot.Environment)
		if e != nil {
			continue
		}
		for _, name := range p.Names {
			if slices.Contains(policy.Names, name) {
				return fmt.Errorf("容器 %s 仍被其他环境的未完成任务使用", name)
			}
		}
	}
	return nil
}

func RequiresContainerAgent(s Snapshot) bool {
	return IsCommand(s.Release) && (s.Environment.ReplaceContainers || len(s.Environment.ContainerNames) > 0)
}

func containerBelongsToEnvironment(detail dockerTypes.ContainerJSON, env models.Environment) error {
	if detail.Config == nil {
		return fmt.Errorf("容器配置不可用：%s", detail.Name)
	}
	labels := detail.Config.Labels
	if owner := labels["io.dockerun.environment"]; owner != "" && owner != fmt.Sprint(env.ID) {
		return fmt.Errorf("容器 %s 属于其他环境，已停止替换", detail.Name)
	}
	if owner := labels["io.dockerun.application"]; owner != "" && owner != fmt.Sprint(env.ApplicationID) {
		return fmt.Errorf("容器 %s 属于其他应用，已停止替换", detail.Name)
	}
	if project := labels["com.docker.compose.project"]; strings.HasPrefix(project, "dockerun-a") && project != ProjectName(env) {
		return fmt.Errorf("容器 %s 属于其他应用环境，已停止替换", detail.Name)
	}
	return nil
}

func removeAssociatedContainers(ctx context.Context, s Snapshot, log func(string, string)) (bool, error) {
	p, err := ContainerSettings(s.Environment)
	if err != nil {
		return false, err
	}
	// 先核对全部归属，再修改任何容器；后续始终使用完整 ID，避免名称变化误操作。
	var old []dockerTypes.ContainerJSON
	for _, name := range p.Names {
		detail, err := docker.GetClient().ContainerInspect(ctx, name)
		if errdefs.IsNotFound(err) {
			continue
		}
		if err != nil {
			return false, err
		}
		if detail.ContainerJSONBase == nil || strings.TrimPrefix(detail.Name, "/") != name {
			return false, fmt.Errorf("容器 %s 的名称与关联配置不一致", name)
		}
		if err := containerBelongsToEnvironment(detail, s.Environment); err != nil {
			return false, err
		}
		old = append(old, detail)
	}
	changed := false
	for _, detail := range old {
		changed = true
		if detail.State != nil && detail.State.Paused {
			if err := docker.GetClient().ContainerUnpause(ctx, detail.ID); err != nil {
				return changed, err
			}
		}
		if detail.State != nil && detail.State.Running {
			log("containers", "停止关联容器 "+detail.Name)
			timeout := 30
			if err := docker.GetClient().ContainerStop(ctx, detail.ID, container.StopOptions{Timeout: &timeout}); err != nil && !errdefs.IsNotFound(err) {
				return changed, err
			}
		}
		log("containers", "删除关联容器 "+detail.Name+"，保留数据卷")
		if err := docker.GetClient().ContainerRemove(ctx, detail.ID, container.RemoveOptions{}); err != nil && !errdefs.IsNotFound(err) {
			return changed, err
		}
	}
	return changed, nil
}

func refreshImageReferences(ctx context.Context, r models.Release, log func(string, string)) error {
	var images []Image
	if r.Images != "" {
		if err := json.Unmarshal([]byte(r.Images), &images); err != nil {
			return err
		}
	}
	for _, image := range images {
		if image.Reference == "" || strings.HasPrefix(image.Reference, "sha256:") || strings.Contains(image.Reference, "@") {
			continue
		}
		old, _, err := docker.GetClient().ImageInspectWithRaw(ctx, image.Reference)
		if errdefs.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		containers, err := docker.GetClient().ContainerList(ctx, container.ListOptions{All: true})
		if err != nil {
			return err
		}
		if slices.ContainsFunc(containers, func(c dockerTypes.Container) bool { return c.ImageID == old.ID }) {
			log("images", "本地镜像仍被其他容器使用，保留镜像并按本次摘要拉取："+image.Reference)
			continue
		}
		log("images", "清理同 tag 的本地镜像引用："+image.Reference)
		if _, err := docker.GetClient().ImageRemove(ctx, image.Reference, dockerTypes.ImageRemoveOptions{}); err != nil && !errdefs.IsNotFound(err) {
			if errdefs.IsConflict(err) {
				log("images", "镜像引用存在共享使用，保留并继续按摘要更新："+image.Reference)
				continue
			}
			return err
		}
	}
	return nil
}

// RunDeployment 统一本机与 Agent 的执行顺序；受管命令在拉取镜像前替换容器。
func RunDeployment(ctx context.Context, s *Snapshot, dir, auth string, log func(string, string), checkpoint func() error) (actual, health string, updated bool, err error) {
	managed := IsCommand(s.Release) && s.Environment.ReplaceContainers
	if _, err = ContainerSettings(s.Environment); err != nil {
		return
	}
	if managed {
		if err = ValidateExecution(s.Release.Execution); err != nil {
			return
		}
		if err = ValidateScripts(s.Scripts); err != nil {
			return
		}
		if _, err = commandImageEnvironment(s.Release); err != nil {
			return
		}
		if err = writeCommandFiles(s.Release, dir); err != nil {
			return
		}
		info, e := os.Stat(scriptWorkingDirectory(dir, *s))
		if e != nil {
			err = fmt.Errorf("脚本工作目录不可用：%w", e)
			return
		}
		if !info.IsDir() {
			err = fmt.Errorf("脚本工作目录必须是目录")
			return
		}
		if checkpoint != nil {
			if err = checkpoint(); err != nil {
				return
			}
		}
	}
	// 即便拉取或脚本失败，也核对关联容器的实际状态，不能继续显示已删除的旧服务。
	if IsCommand(s.Release) && len(s.Environment.ContainerNames) > 0 {
		defer func() {
			readCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			rows, e := RuntimeContainers(readCtx, *s)
			if e != nil {
				log("runtime", "无法核对关联容器："+e.Error())
				if err == nil {
					err = e
				}
				return
			}
			s.Containers = rows
			if err == nil {
				for _, row := range rows {
					if row.State != "running" || row.Health == "unhealthy" {
						err = fmt.Errorf("关联容器 %s 未正常运行", row.Name)
						return
					}
				}
			}
			if err == nil {
				if len(s.Images) > 0 && !slices.ContainsFunc(rows, func(row models.RuntimeContainer) bool {
					return slices.ContainsFunc(s.Images, func(image Image) bool { return image.ID == row.ImageID })
				}) {
					err = fmt.Errorf("关联容器没有使用本次发布的镜像")
					return
				}
				a := map[string]string{}
				for _, row := range rows {
					a[row.ID] = row.ImageID
				}
				actual = stringJSON(a)
			}
		}()
	}
	if managed {
		updated, err = removeAssociatedContainers(ctx, *s, log)
		if err != nil {
			return
		}
		if err = refreshImageReferences(ctx, s.Release, log); err != nil {
			return
		}
	}
	s.Images, err = PrepareWithAuth(ctx, &s.Release, auth, func(m string) { log("images", m) })
	if err != nil {
		return
	}
	if managed {
		for _, image := range s.Images {
			if image.ID != "" && image.Reference != "" && !strings.Contains(image.Reference, "@") {
				if err = docker.GetClient().ImageTag(ctx, image.ID, image.Reference); err != nil {
					return
				}
				log("images", "已刷新本地 tag："+image.Reference+" → "+image.ID)
			}
		}
	} else if checkpoint != nil {
		if err = checkpoint(); err != nil {
			return
		}
	}
	var applied bool
	actual, health, applied, err = Apply(ctx, *s, dir, log, nil)
	updated = updated || applied
	if !IsCommand(s.Release) {
		if rows, e := RuntimeContainers(ctx, *s); e == nil {
			s.Containers = rows
		}
	}
	return
}

func RuntimeContainers(ctx context.Context, s Snapshot) ([]models.RuntimeContainer, error) {
	var names []string
	if IsCommand(s.Release) {
		p, err := ContainerSettings(s.Environment)
		if err != nil {
			return nil, err
		}
		names = p.Names
	} else {
		list, err := docker.GetClient().ContainerList(ctx, container.ListOptions{All: true, Filters: filters.NewArgs(filters.Arg("label", "com.docker.compose.project="+ProjectName(s.Environment)), filters.Arg("label", "com.docker.compose.oneoff=False"))})
		if err != nil {
			return nil, err
		}
		for _, c := range list {
			names = append(names, c.ID)
		}
	}
	rows := make([]models.RuntimeContainer, 0, len(names))
	for _, name := range names {
		detail, err := docker.GetClient().ContainerInspect(ctx, name)
		if errdefs.IsNotFound(err) {
			rows = append(rows, models.RuntimeContainer{Name: name, State: "missing", CheckedAt: time.Now()})
			continue
		}
		if err != nil {
			return nil, err
		}
		if detail.ContainerJSONBase == nil || detail.State == nil || detail.Config == nil {
			return nil, fmt.Errorf("容器 %s 信息不完整", name)
		}
		if IsCommand(s.Release) {
			if strings.TrimPrefix(detail.Name, "/") != name {
				return nil, fmt.Errorf("关联容器名称不匹配：%s", name)
			}
			if err := containerBelongsToEnvironment(detail, s.Environment); err != nil {
				return nil, err
			}
		}
		image := detail.Config.Image
		for _, prepared := range s.Images {
			if prepared.ID == detail.Image && prepared.Reference != "" {
				// Compose 用 ID、命令用固定摘要启动时补充对应名称；字面 tag 保持实际配置原值。
				if image != detail.Image {
					actualRef, actualErr := reference.ParseNormalizedNamed(image)
					sourceRef, sourceErr := reference.ParseNormalizedNamed(prepared.Reference)
					canonical, pinned := actualRef.(reference.Canonical)
					if actualErr != nil || sourceErr != nil || !pinned || canonical.Name() != sourceRef.Name() || canonical.Digest().String() != prepared.Digest {
						continue
					}
				}
				image = prepared.Reference
				if s.Release.TriggerImage != "" && prepared.Digest == s.Release.TriggerDigest {
					image = s.Release.TriggerImage
				}
				break
			}
		}
		health := "unverified"
		if detail.State.Health != nil {
			health = detail.State.Health.Status
		}
		rows = append(rows, models.RuntimeContainer{ID: detail.ID, Name: strings.TrimPrefix(detail.Name, "/"), Image: image, ImageID: detail.Image, State: detail.State.Status, Health: health, CheckedAt: time.Now()})
	}
	return rows, nil
}
