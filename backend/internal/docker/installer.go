package docker

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type SystemInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	Distribution string `json:"distribution"`
	Version      string `json:"version"`
	HasInternet  bool   `json:"has_internet"`
	DockerExists bool   `json:"docker_exists"`
	ComposeExists bool  `json:"compose_exists"`
}

// CheckSystemInfo 检查系统信息
func CheckSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// 检查是否为Linux
	if info.OS != "linux" {
		return info, fmt.Errorf("only Linux is supported for Docker installation")
	}

	// 检测Linux发行版
	distribution, version := detectLinuxDistribution()
	info.Distribution = distribution
	info.Version = version

	// 检查网络连接
	info.HasInternet = checkInternetConnectivity()

	// 检查Docker是否已安装
	info.DockerExists = checkCommandExists("docker")

	// 检查Docker Compose是否已安装
	info.ComposeExists = checkCommandExists("docker-compose") || checkDockerComposePlugin()

	return info, nil
}

// detectLinuxDistribution 检测Linux发行版
func detectLinuxDistribution() (string, string) {
	// 读取 /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "unknown", "unknown"
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	
	distro := "unknown"
	version := "unknown"
	
	for _, line := range lines {
		if strings.HasPrefix(line, "ID=") {
			distro = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
	}

	return distro, version
}

// checkInternetConnectivity 检查网络连接
func checkInternetConnectivity() bool {
	cmd := exec.Command("ping", "-c", "1", "-W", "2", "8.8.8.8")
	err := cmd.Run()
	return err == nil
}

// checkCommandExists 检查命令是否存在
func checkCommandExists(command string) bool {
	cmd := exec.Command("which", command)
	err := cmd.Run()
	return err == nil
}

// checkDockerComposePlugin 检查Docker Compose插件
func checkDockerComposePlugin() bool {
	cmd := exec.Command("docker", "compose", "version")
	err := cmd.Run()
	return err == nil
}

// InstallDockerOnline 在线安装Docker
func InstallDockerOnline(distribution string) error {
	var cmd *exec.Cmd

	switch distribution {
	case "ubuntu", "debian":
		// Ubuntu/Debian安装脚本
		script := `#!/bin/bash
set -e

# 更新包索引
apt-get update

# 安装依赖
apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

# 添加Docker官方GPG密钥
mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# 设置仓库
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

# 安装Docker Engine
apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 启动Docker
systemctl start docker
systemctl enable docker
`
		cmd = exec.Command("bash", "-c", script)

	case "centos", "rhel", "fedora":
		// CentOS/RHEL/Fedora安装脚本
		script := `#!/bin/bash
set -e

# 安装依赖
yum install -y yum-utils

# 添加Docker仓库
yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo

# 安装Docker Engine
yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 启动Docker
systemctl start docker
systemctl enable docker
`
		cmd = exec.Command("bash", "-c", script)

	default:
		return fmt.Errorf("unsupported distribution: %s", distribution)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("installation failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

// InstallDockerOfflineRPM 使用RPM包离线安装
func InstallDockerOfflineRPM(packagePath string) error {
	cmd := exec.Command("rpm", "-ivh", packagePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("RPM installation failed: %v\nOutput: %s", err, string(output))
	}
	return nil
}

// InstallDockerOfflineDEB 使用DEB包离线安装
func InstallDockerOfflineDEB(packagePath string) error {
	cmd := exec.Command("dpkg", "-i", packagePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("DEB installation failed: %v\nOutput: %s", err, string(output))
	}
	return nil
}

// InstallDockerOfflineBinary 使用二进制文件离线安装
func InstallDockerOfflineBinary(tarPath string) error {
	script := fmt.Sprintf(`#!/bin/bash
set -e

# 解压二进制文件
tar xzvf %s -C /tmp

# 移动到系统目录
cp /tmp/docker/* /usr/bin/

# 创建systemd服务文件
cat > /etc/systemd/system/docker.service << 'EOF'
[Unit]
Description=Docker Application Container Engine
Documentation=https://docs.docker.com
After=network-online.target firewalld.service containerd.service
Wants=network-online.target

[Service]
Type=notify
ExecStart=/usr/bin/dockerd
ExecReload=/bin/kill -s HUP $MAINPID
TimeoutSec=0
RestartSec=2
Restart=always
StartLimitBurst=3
StartLimitInterval=60s
LimitNOFILE=infinity
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
Delegate=yes
KillMode=process

[Install]
WantedBy=multi-user.target
EOF

# 重载systemd并启动Docker
systemctl daemon-reload
systemctl start docker
systemctl enable docker
`, tarPath)

	cmd := exec.Command("bash", "-c", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("binary installation failed: %v\nOutput: %s", err, string(output))
	}
	return nil
}

