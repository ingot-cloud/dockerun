package release

import (
	"context"
	"dockerun/internal/docker"
	"encoding/json"
	"fmt"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

type Image struct {
	Reference string `json:"reference"`
	Original  string `json:"original,omitempty"`
	Digest    string `json:"digest,omitempty"`
	ID        string `json:"id,omitempty"`
	Platform  string `json:"platform,omitempty"`
	Archive   string `json:"archive,omitempty"`
	SHA256    string `json:"sha256,omitempty"`
}

func SafePath(root, name string) (string, error) {
	if name == "" || filepath.IsAbs(name) || strings.Contains(name, "\\") || strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("不安全路径: %s", name)
	}
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("不安全路径: %s", name)
	}
	return filepath.Join(root, clean), nil
}
func WriteConfigs(dir string, configs map[string]string) error {
	for k, v := range configs {
		if !strings.HasPrefix(k, "configs/") {
			return fmt.Errorf("配置路径必须位于 configs/: %s", k)
		}
		p, err := SafePath(dir, k)
		if err != nil {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			return err
		}
		if err = os.WriteFile(p, []byte(v), 0644); err != nil {
			return err
		}
	}
	return nil
}

func Parse(ctx context.Context, content, dir, project string, parameters map[string]string) (*types.Project, error) {
	// Reject file inclusion before compose-go can read arbitrary files or remote sources.
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(content), &raw); err != nil {
		return nil, err
	}
	if raw["include"] != nil {
		return nil, fmt.Errorf("受管部署不接受 include，请提供完整 Compose")
	}
	if svcs, ok := raw["services"].(map[string]any); ok {
		for name, value := range svcs {
			svc, ok := value.(map[string]any)
			if !ok {
				continue
			}
			for _, key := range []string{"build", "extends", "env_file", "develop", "provider"} {
				if svc[key] != nil {
					return nil, fmt.Errorf("服务 %s 不接受 %s；请预先构建镜像，将环境参数放入环境配置", name, key)
				}
			}
			if svc["profiles"] != nil {
				return nil, fmt.Errorf("请在发布包中明确所需服务，不接受 profiles")
			}
		}
	}
	p, err := loader.LoadWithContext(ctx, types.ConfigDetails{WorkingDir: dir, ConfigFiles: []types.ConfigFile{{Filename: filepath.Join(dir, "compose.yaml"), Content: []byte(content)}}, Environment: types.Mapping(parameters)}, func(o *loader.Options) {
		o.SetProjectName(project, true)
		o.ResolvePaths = true
		o.SkipInclude = true
		o.SkipExtends = true
	})
	if err != nil {
		return nil, err
	}
	if len(p.Services) == 0 {
		return nil, fmt.Errorf("应用至少需要一个服务")
	}
	for n, s := range p.Services {
		for i := range s.Volumes {
			if s.Volumes[i].Type == types.VolumeTypeBind {
				s.Volumes[i].Source = docker.LocalDataPath(s.Volumes[i].Source)
			}
		}
		if s.Image == "" {
			return nil, fmt.Errorf("服务 %s 缺少 image", n)
		}
		if s.Build != nil {
			return nil, fmt.Errorf("部署阶段禁止构建")
		}
		if s.PullPolicy == "build" {
			return nil, fmt.Errorf("不允许构建策略")
		}
		s.PullPolicy = "never"
		if s.Labels == nil {
			s.Labels = types.Labels{}
		}
		s.Labels["io.dockerun.managed"] = "true"
		p.Services[n] = s
	}
	for name, c := range p.Configs {
		if c.File != "" {
			c.File = docker.LocalDataPath(c.File)
			p.Configs[name] = c
		}
		if c.File != "" && !inside(dir, c.File) {
			return nil, fmt.Errorf("配置文件须包含在发布材料中: %s", c.File)
		}
	}
	for name, volume := range p.Volumes {
		if volume.Driver == "" || volume.Driver == "local" {
			for _, flag := range strings.Split(volume.DriverOpts["o"], ",") {
				if flag == "bind" || flag == "rbind" {
					volume.DriverOpts["device"] = docker.LocalDataPath(volume.DriverOpts["device"])
					p.Volumes[name] = volume
					break
				}
			}
		}
	}
	for _, s := range p.Secrets {
		if s.File != "" {
			return nil, fmt.Errorf("敏感文件应通过外部 secret 或目标环境提供")
		}
	}
	return p, nil
}
func inside(root, path string) bool {
	a, _ := filepath.Abs(root)
	p, _ := filepath.Abs(path)
	rel, err := filepath.Rel(a, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
func Encode(p *types.Project) (string, error) { b, err := json.Marshal(p); return string(b), err }
func References(p *types.Project) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range p.Services {
		if !seen[s.Image] {
			out = append(out, s.Image)
			seen[s.Image] = true
		}
	}
	return out
}

// FreezeImages preserves runtime placeholders while fixing the release's image references.
func FreezeImages(content string, p *types.Project) (string, error) {
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return "", err
	}
	services, _ := doc["services"].(map[string]any)
	for name, value := range services {
		if svc, ok := value.(map[string]any); ok {
			if resolved, found := p.Services[name]; found {
				svc["image"] = resolved.Image
			}
		}
	}
	b, err := yaml.Marshal(doc)
	return string(b), err
}

// Compose will parse the resolved file once more. Escape literal dollars in values
// so passwords and container-side shell variables are not interpolated a second time.
func EncodeExecution(p *types.Project) (string, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	var doc any
	if err = json.Unmarshal(raw, &doc); err != nil {
		return "", err
	}
	var escape func(any) any
	escape = func(value any) any {
		switch v := value.(type) {
		case string:
			return strings.ReplaceAll(v, "$", "$$")
		case map[string]any:
			for k, item := range v {
				v[k] = escape(item)
			}
		case []any:
			for i, item := range v {
				v[i] = escape(item)
			}
		}
		return value
	}
	b, err := json.Marshal(escape(doc))
	return string(b), err
}
