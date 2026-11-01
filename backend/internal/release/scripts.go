package release

import (
	"context"
	"dockerun/internal/models"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

func HasScripts(s models.DeploymentScripts) bool {
	return strings.TrimSpace(s.Before+s.Deploy+s.After) != ""
}

func EffectiveScripts(r models.Release, env models.Environment) models.DeploymentScripts {
	switch env.ScriptMode {
	case "override":
		return env.Scripts
	case "disabled":
		return models.DeploymentScripts{}
	default:
		return r.Scripts
	}
}

func ValidateScripts(s models.DeploymentScripts) error {
	for _, script := range []struct{ name, content string }{{"部署前", s.Before}, {"部署", s.Deploy}, {"部署后", s.After}} {
		if len(script.content) > 256<<10 || strings.ContainsRune(script.content, 0) {
			return fmt.Errorf("%s脚本不得超过 256 KiB 或包含空字符", script.name)
		}
		if strings.TrimSpace(script.content) == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, "/bin/bash", "-n")
		cmd.Stdin = strings.NewReader(script.content)
		output, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			return fmt.Errorf("%s脚本语法检查失败：%s", script.name, strings.TrimSpace(string(output)))
		}
	}
	return nil
}

var scriptVariable = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func commandEnvironment() []string {
	env := []string{"PATH=" + commandPath(), "HOME=" + os.Getenv("HOME")}
	for _, key := range []string{"DOCKER_CONFIG", "DOCKER_HOST", "DOCKER_CONTEXT", "DOCKER_TLS_VERIFY", "DOCKER_CERT_PATH"} {
		if v := os.Getenv(key); v != "" {
			env = append(env, key+"="+v)
		}
	}
	return env
}

func RunScript(ctx context.Context, content, stage, dir string, s Snapshot, log func(string, string)) error {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	if stage != "before_script" && stage != "deploy_script" && stage != "after_script" && stage != "check_script" {
		return fmt.Errorf("未知脚本阶段")
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, stage+".sh")
	if err = os.WriteFile(path, []byte(content), 0600); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "/bin/bash", "-e", "-o", "pipefail", path)
	cmd.Dir = scriptWorkingDirectory(dir, s)
	cmd.Env = commandEnvironment()
	for k, v := range s.Environment.Parameters {
		if scriptVariable.MatchString(k) {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	imageEnvironment, err := commandImageEnvironment(s.Release)
	if err != nil {
		return err
	}
	cmd.Env = append(cmd.Env, imageEnvironment...)
	cmd.Env = append(cmd.Env,
		"WORK_DIR="+dir, "COMPOSE_FILE="+filepath.Join(dir, "resolved.json"), "COMPOSE_PROJECT_NAME="+ProjectName(s.Environment),
		fmt.Sprintf("DOCKERUN_APPLICATION_ID=%d", s.Release.ApplicationID), fmt.Sprintf("DOCKERUN_ENVIRONMENT_ID=%d", s.Environment.ID),
		"DOCKERUN_ENVIRONMENT="+s.Environment.Name, "DOCKERUN_VERSION="+s.Release.Version,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = time.Second
	output := &scriptLogWriter{log: func(m string) { log(stage, m) }, secrets: scriptSecrets(s)}
	cmd.Stdout, cmd.Stderr = output, output
	log(stage, "开始执行脚本")
	err = cmd.Run()
	output.Flush()
	if err != nil {
		return fmt.Errorf("%s执行失败: %w", map[string]string{"before_script": "部署前脚本", "deploy_script": "部署命令", "after_script": "部署后脚本", "check_script": "部署检查"}[stage], err)
	}
	log(stage, "脚本执行完成")
	return nil
}

func scriptWorkingDirectory(dir string, s Snapshot) string {
	if workDir := s.Release.Execution.WorkDir; workDir != "" {
		workDir = os.Expand(workDir, func(key string) string {
			if key == "WORK_DIR" {
				return dir
			}
			return s.Environment.Parameters[key]
		})
		if !filepath.IsAbs(workDir) {
			workDir = filepath.Join(dir, workDir)
		}
		return workDir
	}
	return dir
}

// Apply 在实际目标主机上运行同一发布流程；预览不执行任何脚本。
func Apply(ctx context.Context, s Snapshot, dir string, log func(string, string), checkpoint func() error) (actual, health string, updated bool, err error) {
	if IsCommand(s.Release) {
		return applyCommand(ctx, s, dir, log, checkpoint)
	}
	p, err := Render(ctx, s.Release, s.Environment, dir, s.Images)
	if err != nil {
		return "", "", false, err
	}
	if err = WriteExecution(p, dir); err != nil {
		return "", "", false, err
	}
	if checkpoint != nil {
		if err = checkpoint(); err != nil {
			return "", "", false, err
		}
	}
	if err = RunScript(ctx, s.Scripts.Before, "before_script", dir, s, log); err != nil {
		return "", "", false, err
	}
	log("preflight", "检查目标主机的端口、挂载、网络和卷")
	if err = Preflight(ctx, p); err != nil {
		return "", "", false, fmt.Errorf("预检查失败: %w", err)
	}
	if strings.TrimSpace(s.Release.Execution.Command) != "" {
		err = RunScript(ctx, s.Release.Execution.Command, "deploy_script", dir, s, log)
	} else if strings.TrimSpace(s.Scripts.Deploy) != "" {
		err = RunScript(ctx, s.Scripts.Deploy, "deploy_script", dir, s, log)
	} else {
		log("deploy", "预检查通过，更新此应用的 Compose project")
		err = Up(ctx, p, dir, func(m string) { log("deploy", m) }, !s.Automatic)
	}
	if err == nil {
		actual, health, err = Actual(ctx, p)
	}
	if err == nil {
		err = RunScript(ctx, s.Release.Execution.CheckCommand, "check_script", dir, s, log)
	}
	if err == nil {
		err = RunScript(ctx, s.Scripts.After, "after_script", dir, s, log)
		if err == nil && strings.TrimSpace(s.Scripts.After) != "" {
			actual, health, err = Actual(ctx, p)
		}
	}
	return actual, health, true, err
}
