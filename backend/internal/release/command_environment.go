package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// commandPath 保留服务进程的命令优先级，并补齐非交互启动时可能缺失的安装目录。
// 不加载用户的 Shell 配置，避免部署时执行其中的交互命令或导入服务端凭据。
func commandPath() string {
	paths := filepath.SplitList(os.Getenv("PATH"))
	paths = append(paths, "/usr/local/bin", "/usr/bin", "/bin", "/usr/local/sbin", "/usr/sbin", "/sbin")
	if home := os.Getenv("HOME"); filepath.IsAbs(home) {
		paths = append(paths, filepath.Join(home, ".docker", "bin"))
	}
	if runtime.GOOS == "darwin" {
		paths = append(paths, "/opt/homebrew/bin", "/opt/homebrew/sbin", "/Applications/Docker.app/Contents/Resources/bin")
	}
	seen := make(map[string]bool, len(paths))
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		if path != "" && !seen[path] {
			seen[path] = true
			unique = append(unique, path)
		}
	}
	return strings.Join(unique, string(os.PathListSeparator))
}

// commandExecutable 与脚本使用同一搜索路径；exec.Command 不使用 cmd.Env 查找命令。
func commandExecutable(name string) string {
	for _, dir := range filepath.SplitList(commandPath()) {
		if path, err := exec.LookPath(filepath.Join(dir, name)); err == nil && filepath.IsAbs(path) {
			return path
		}
	}
	return name
}
