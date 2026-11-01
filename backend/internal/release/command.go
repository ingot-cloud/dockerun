package release

import (
	"context"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"github.com/distribution/reference"
	digest "github.com/opencontainers/go-digest"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func IsCommand(r models.Release) bool { return r.Execution.Mode == "command" }

func ValidateExecution(e models.DeploymentExecution) error {
	if e.Mode != "" && e.Mode != "compose" && e.Mode != "command" {
		return fmt.Errorf("部署方式仅支持命令或 Compose")
	}
	if e.Mode == "command" && strings.TrimSpace(e.Command) == "" {
		return fmt.Errorf("请填写部署命令")
	}
	if e.TimeoutSeconds < 0 || e.TimeoutSeconds > 86400 {
		return fmt.Errorf("任务超时必须为 0 至 86400 秒，0 表示使用系统设置")
	}
	if len(e.WorkDir) > 4096 || strings.ContainsRune(e.WorkDir, 0) {
		return fmt.Errorf("工作目录无效")
	}
	if err := ValidateScripts(models.DeploymentScripts{Deploy: e.Command}); err != nil {
		return fmt.Errorf("部署命令校验失败：%w", err)
	}
	if err := ValidateScripts(models.DeploymentScripts{Deploy: e.CheckCommand}); err != nil {
		return fmt.Errorf("部署检查命令校验失败：%w", err)
	}
	return nil
}

func ExecutionTimeout(r models.Release, fallback time.Duration) time.Duration {
	if r.Execution.TimeoutSeconds > 0 {
		return time.Duration(r.Execution.TimeoutSeconds) * time.Second
	}
	return fallback
}

// RequiresExecutionAgent 防止旧 Agent 忽略新的启动方式、检查和超时配置。
func RequiresExecutionAgent(r models.Release) bool {
	return r.Execution != (models.DeploymentExecution{})
}

func RequiresCommandImageAgent(r models.Release) bool { return IsCommand(r) && r.Source == "registry" }

func commandImageEnvironment(r models.Release) ([]string, error) {
	if !RequiresCommandImageAgent(r) {
		return nil, nil
	}
	var images []Image
	if err := json.Unmarshal([]byte(r.Images), &images); err != nil || len(images) != 1 || images[0].Digest == "" {
		return nil, fmt.Errorf("命令发布版本缺少唯一的触发镜像及摘要")
	}
	image := images[0]
	named, err := reference.ParseNormalizedNamed(image.Reference)
	if err != nil {
		return nil, err
	}
	d, err := digest.Parse(image.Digest)
	if err != nil {
		return nil, err
	}
	pinned, err := reference.WithDigest(reference.TrimNamed(named), d)
	if err != nil {
		return nil, err
	}
	tag := ""
	if tagged, ok := named.(reference.Tagged); ok {
		tag = tagged.Tag()
	}
	return []string{"DOCKERUN_IMAGE=" + pinned.String(), "DOCKERUN_IMAGE_TAG=" + tag, "DOCKERUN_IMAGE_DIGEST=" + image.Digest}, nil
}

func writeCommandFiles(r models.Release, dir string) error {
	var configs map[string]string
	if r.Configs != "" {
		if err := json.Unmarshal([]byte(r.Configs), &configs); err != nil {
			return err
		}
	}
	if err := WriteConfigs(dir, configs); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "deploy.sh"), []byte(r.Execution.Command), 0600)
}

func applyCommand(ctx context.Context, s Snapshot, dir string, log func(string, string), checkpoint func() error) (actual, health string, updated bool, err error) {
	if err = ValidateExecution(s.Release.Execution); err != nil {
		return
	}
	if err = writeCommandFiles(s.Release, dir); err != nil {
		return
	}
	if checkpoint != nil {
		if err = checkpoint(); err != nil {
			return
		}
	}
	if err = RunScript(ctx, s.Scripts.Before, "before_script", dir, s, log); err != nil {
		return
	}
	updated = true
	if err = RunScript(ctx, s.Release.Execution.Command, "deploy_script", dir, s, log); err != nil {
		return
	}
	health = "unchecked"
	if strings.TrimSpace(s.Release.Execution.CheckCommand) != "" {
		if err = RunScript(ctx, s.Release.Execution.CheckCommand, "check_script", dir, s, log); err != nil {
			return
		}
		health = "healthy"
	}
	if err = RunScript(ctx, s.Scripts.After, "after_script", dir, s, log); err != nil {
		err = fmt.Errorf("部署命令已执行，%w", err)
	}
	return
}
