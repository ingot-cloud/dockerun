package release

import (
	"context"
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/models"
	"dockerun/internal/utils"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	regtypes "github.com/docker/docker/api/types/registry"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func Consume(stream io.Reader, log func(string)) error {
	dec := json.NewDecoder(stream)
	for {
		var m struct {
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
			Stream string `json:"stream"`
			Status string `json:"status"`
		}
		err := dec.Decode(&m)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if m.Error != "" {
			return fmt.Errorf("%s", m.Error)
		}
		if m.ErrorDetail.Message != "" {
			return fmt.Errorf("%s", m.ErrorDetail.Message)
		}
		if log != nil {
			if m.Stream != "" {
				log(m.Stream)
			} else if m.Status != "" {
				log(m.Status)
			}
		}
	}
}
func Platform(ctx context.Context) (string, error) {
	i, err := docker.GetClient().Info(ctx)
	if err != nil {
		return "", err
	}
	arch := i.Architecture
	switch arch {
	case "aarch64":
		arch = "arm64"
	case "x86_64":
		arch = "amd64"
	}
	return i.OSType + "/" + arch, nil
}
func imageAuth(reg *models.Registry, repositories []string) (string, error) {
	if reg == nil {
		return "", nil
	}
	auth := regtypes.AuthConfig{ServerAddress: reg.URL}
	if reg.Kind == "builtin" {
		token, err := builtin.InternalToken(repositories)
		if err != nil {
			return "", err
		}
		auth.RegistryToken = token
	} else {
		auth.Username = reg.Username
		auth.Password = reg.Password
		if p, err := utils.OpenCredential(reg.Password); err == nil {
			auth.Password = p
		}
	}
	b, err := json.Marshal(auth)
	return base64.URLEncoding.EncodeToString(b), err
}
func Prepare(ctx context.Context, r *models.Release, log func(string)) ([]Image, error) {
	return PrepareWithAuth(ctx, r, "", log)
}
func PrepareWithAuth(ctx context.Context, r *models.Release, auth string, log func(string)) ([]Image, error) {
	if IsCommand(*r) {
		if err := ValidateExecution(r.Execution); err != nil {
			return nil, err
		}
		if !RequiresCommandImageAgent(*r) {
			return nil, nil
		}
	}
	var images []Image
	if err := json.Unmarshal([]byte(r.Images), &images); err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("版本未包含镜像清单")
	}
	platform, err := Platform(ctx)
	if err != nil {
		return nil, err
	}
	if r.Platform != "" && platform != r.Platform {
		return nil, fmt.Errorf("目标平台 %s 与版本 %s 不符", platform, r.Platform)
	}
	var reg *models.Registry
	if r.Source == "registry" && r.RegistryID > 0 && auth == "" {
		var v models.Registry
		if err := database.DB.First(&v, r.RegistryID).Error; err != nil {
			return nil, err
		}
		reg = &v
	}
	repos := []string{}
	for _, im := range images {
		n, e := reference.ParseNormalizedNamed(im.Reference)
		if e != nil {
			return nil, e
		}
		repos = append(repos, reference.Path(n))
	}
	if auth == "" {
		auth, err = imageAuth(reg, repos)
		if err != nil {
			return nil, err
		}
	}
	for idx, im := range images {
		ref := im.Reference
		if r.Source == "registry" || r.Source == "direct" {
			if r.Source == "registry" {
				if im.Digest == "" {
					return nil, fmt.Errorf("仓库版本缺少不可变 digest: %s", ref)
				}
				n, err := reference.ParseNormalizedNamed(ref)
				if err != nil {
					return nil, err
				}
				ref = reference.TrimNamed(n).String() + "@" + im.Digest
			}
			stream, err := docker.GetClient().ImagePull(ctx, ref, dockerTypes.ImagePullOptions{RegistryAuth: auth, Platform: platform})
			if err != nil {
				return nil, err
			}
			err = Consume(stream, log)
			stream.Close()
			if err != nil {
				return nil, err
			}
		}
		if r.Source == "bundle" || r.Source == "local" {
			ref = im.ID
			if ref == "" {
				return nil, fmt.Errorf("离线版本缺少 Image ID")
			}
		}
		inspect, _, err := docker.GetClient().ImageInspectWithRaw(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("镜像未准备就绪 %s: %w", ref, err)
		}
		if im.ID != "" && inspect.ID != im.ID {
			return nil, fmt.Errorf("镜像身份已变化 %s", im.Reference)
		}
		if inspect.Os+"/"+inspect.Architecture != platform {
			return nil, fmt.Errorf("镜像平台不匹配 %s", im.Reference)
		}
		images[idx].ID = inspect.ID
		images[idx].Platform = platform
	}
	return images, nil
}
func ProjectName(env models.Environment) string {
	return fmt.Sprintf("dockerun-a%d-e%d", env.ApplicationID, env.ID)
}
func Render(ctx context.Context, r models.Release, env models.Environment, dir string, images []Image) (*types.Project, error) {
	var configs map[string]string
	if r.Configs != "" {
		if err := json.Unmarshal([]byte(r.Configs), &configs); err != nil {
			return nil, err
		}
	}
	if err := WriteConfigs(dir, configs); err != nil {
		return nil, err
	}
	p, err := Parse(ctx, r.Compose, dir, ProjectName(env), env.Parameters)
	if err != nil {
		return nil, err
	}
	for name, s := range p.Services {
		found := false
		for _, im := range images {
			if im.Reference == s.Image || im.Original != "" && im.Original == s.Image {
				if im.ID == "" {
					return nil, fmt.Errorf("镜像尚未加载")
				}
				s.Image = im.ID
				// 标签变化也应触发服务更新，不能仅按相同 Image ID 判定配置未变。
				if trigger, e := reference.ParseNormalizedNamed(r.TriggerImage); e == nil && r.TriggerDigest != "" && im.Digest == r.TriggerDigest {
					if source, e := reference.ParseNormalizedNamed(im.Reference); e == nil && source.Name() == trigger.Name() {
						if s.Labels == nil {
							s.Labels = types.Labels{}
						}
						s.Labels["io.dockerun.image-trigger"] = r.TriggerImage + "@" + r.TriggerDigest
					}
				}
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("服务 %s 缺少镜像 %s", name, s.Image)
		}
		p.Services[name] = s
	}
	return p, nil
}
func Preflight(ctx context.Context, p *types.Project) error {
	cli := docker.GetClient()
	existing, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err
	}
	ports := map[string]string{}
	for name, s := range p.Services {
		for _, v := range s.Volumes {
			if v.Type == "bind" {
				if _, err := os.Stat(v.Source); err != nil {
					return fmt.Errorf("服务 %s 挂载不存在: %s", name, v.Source)
				}
			}
		}
		for _, port := range s.Ports {
			if port.Published == "" {
				continue
			}
			key := port.Published + "/" + port.Protocol
			if prev := ports[key]; prev != "" {
				return fmt.Errorf("服务 %s 和 %s 发布端口冲突", prev, name)
			}
			ports[key] = name
		}
		for _, c := range existing {
			own := c.Labels["com.docker.compose.project"] == p.Name
			if s.ContainerName != "" && !own {
				for _, n := range c.Names {
					if strings.TrimPrefix(n, "/") == s.ContainerName {
						return fmt.Errorf("容器名 %s 已被其他应用使用", s.ContainerName)
					}
				}
			}
			if own || c.State != "running" {
				continue
			}
			for _, ep := range c.Ports {
				if ports[fmt.Sprintf("%d/%s", ep.PublicPort, ep.Type)] != "" {
					return fmt.Errorf("宿主端口 %d 已被其他容器占用", ep.PublicPort)
				}
			}
		}
	}
	for name, c := range p.Configs {
		if c.File != "" {
			if _, err := os.Stat(c.File); err != nil {
				return fmt.Errorf("配置 %s 的材料不存在: %w", name, err)
			}
		}
	}
	for name, n := range p.Networks {
		if n.External {
			if _, err := cli.NetworkInspect(ctx, n.Name, dockerTypes.NetworkInspectOptions{}); err != nil {
				return fmt.Errorf("外部网络 %s 不可用: %w", name, err)
			}
		}
	}
	for name, v := range p.Volumes {
		if v.External {
			if _, err := cli.VolumeInspect(ctx, v.Name); err != nil {
				return fmt.Errorf("外部卷 %s 不可用: %w", name, err)
			}
		}
	}
	return nil
}

type logWriter struct{ log func(string) }

func (w logWriter) Write(b []byte) (int, error) { w.log(string(b)); return len(b), nil }
func WriteExecution(p *types.Project, dir string) error {
	content, err := EncodeExecution(p)
	if err != nil {
		return err
	}
	file := filepath.Join(dir, "resolved.json")
	if err = os.WriteFile(file, []byte(content), 0600); err != nil {
		return err
	}
	envFile := filepath.Join(dir, "empty.env")
	if err = os.WriteFile(envFile, nil, 0600); err != nil {
		return err
	}
	return nil
}
func Up(ctx context.Context, p *types.Project, dir string, log func(string), recreate ...bool) error {
	if err := WriteExecution(p, dir); err != nil {
		return err
	}
	file := filepath.Join(dir, "resolved.json")
	envFile := filepath.Join(dir, "empty.env")
	args := []string{"compose", "--env-file", envFile, "--project-name", p.Name, "--project-directory", dir, "-f", file, "up", "--detach", "--pull", "never", "--no-build", "--wait", "--wait-timeout", "120"}
	if len(recreate) > 0 && recreate[0] {
		args = append(args, "--force-recreate", "--remove-orphans")
	}
	cmd := exec.CommandContext(ctx, commandExecutable("docker"), args...)
	cmd.Dir = dir
	cmd.Env = commandEnvironment()
	cmd.Stdout = logWriter{log}
	cmd.Stderr = logWriter{log}
	return cmd.Run()
}
func Actual(ctx context.Context, p *types.Project) (string, string, error) {
	containers, err := docker.GetClient().ContainerList(ctx, container.ListOptions{All: true, Filters: filters.NewArgs(filters.Arg("label", "com.docker.compose.project="+p.Name), filters.Arg("label", "com.docker.compose.oneoff=False"))})
	if err != nil {
		return "", "", err
	}
	actual := map[string]string{}
	health := "healthy"
	seen := map[string]int{}
	for _, c := range containers {
		n := c.Labels["com.docker.compose.service"]
		s, ok := p.Services[n]
		if !ok {
			continue
		}
		detail, err := docker.GetClient().ContainerInspect(ctx, c.ID)
		if err != nil {
			return "", "", err
		}
		if detail.Image != s.Image {
			return "", "", fmt.Errorf("服务 %s 实际 Image ID 不匹配", n)
		}
		if !detail.State.Running {
			return "", "", fmt.Errorf("服务 %s 未运行", n)
		}
		if detail.State.Health == nil {
			health = "unverified"
		} else if detail.State.Health.Status != "healthy" {
			return "", "", fmt.Errorf("服务 %s 健康检查失败: %s", n, detail.State.Health.Status)
		}
		actual[c.ID] = detail.Image
		seen[n]++
	}
	for n, s := range p.Services {
		want := 1
		if s.Deploy != nil && s.Deploy.Replicas != nil {
			want = int(*s.Deploy.Replicas)
		}
		if s.Scale != nil {
			want = *s.Scale
		}
		if seen[n] < want {
			return "", "", fmt.Errorf("服务 %s 实例不足", n)
		}
	}
	b, _ := json.Marshal(actual)
	return string(b), health, nil
}
func JobDir(id uint) string { return filepath.Join(config.Load().DataDir, "jobs", fmt.Sprint(id)) }
func Timeout() time.Duration {
	var s models.SystemConfig
	if database.DB.First(&s).Error == nil && s.ScriptTimeout > 0 {
		return time.Duration(s.ScriptTimeout) * time.Second
	}
	return time.Hour
}

func RegistryCredentials(reg *models.Registry, repositories []string) (string, error) {
	return imageAuth(reg, repositories)
}
