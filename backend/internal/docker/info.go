package docker

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type DockerInfo struct {
	OS               string `json:"os"`
	Arch             string `json:"arch"`
	DockerVersion    string `json:"docker_version"`
	DockerApiVersion string `json:"docker_api_version"`
	ComposeVersion   string `json:"compose_version"`
	ServerVersion    string `json:"server_version"`
	ServerOS         string `json:"server_os"`
	ServerArch       string `json:"server_arch"`
	TotalMemory      string `json:"total_memory"`
	CPUs             int    `json:"cpus"`
	KernelVersion    string `json:"kernel_version"`
	OperatingSystem  string `json:"operating_system"`
}

// GetDockerInfo 获取Docker相关信息
func GetDockerInfo() (*DockerInfo, error) {
	ctx := context.Background()
	info := &DockerInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// 获取Docker客户端版本
	dockerVersion, err := getDockerClientVersion()
	if err == nil {
		info.DockerVersion = dockerVersion
	}

	// 获取Docker Compose版本
	composeVersion, err := getDockerComposeVersion()
	if err == nil {
		info.ComposeVersion = composeVersion
	}

	// 获取Docker服务器信息
	if Client != nil {
		serverInfo, err := Client.Info(ctx)
		if err == nil {
			info.ServerVersion = serverInfo.ServerVersion
			info.ServerOS = serverInfo.OperatingSystem
			info.ServerArch = serverInfo.Architecture
			info.KernelVersion = serverInfo.KernelVersion
			info.OperatingSystem = serverInfo.OperatingSystem
			info.CPUs = serverInfo.NCPU
			// 转换内存为可读格式
			info.TotalMemory = fmt.Sprintf("%.2f GB", float64(serverInfo.MemTotal)/(1024*1024*1024))
		}
		
		// 获取API版本
		version, err := Client.ServerVersion(ctx)
		if err == nil {
			info.DockerApiVersion = version.APIVersion
		}
	}

	return info, nil
}

// getDockerClientVersion 获取Docker客户端版本
func getDockerClientVersion() (string, error) {
	cmd := exec.Command("docker", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	// 解析输出: Docker version 20.10.17, build 100c701
	version := strings.TrimSpace(string(output))
	if strings.Contains(version, "version") {
		parts := strings.Split(version, " ")
		if len(parts) >= 3 {
			return strings.TrimSuffix(parts[2], ","), nil
		}
	}

	return version, nil
}

// getDockerComposeVersion 获取Docker Compose版本
func getDockerComposeVersion() (string, error) {
	// 尝试 docker compose version (plugin)
	cmd := exec.Command("docker", "compose", "version")
	output, err := cmd.CombinedOutput()
	if err == nil {
		version := strings.TrimSpace(string(output))
		// 解析输出: Docker Compose version v2.10.2
		if strings.Contains(version, "version") {
			parts := strings.Split(version, " ")
			if len(parts) >= 3 {
				return parts[len(parts)-1], nil
			}
		}
		return version, nil
	}

	// 尝试 docker-compose --version (standalone)
	cmd = exec.Command("docker-compose", "--version")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return "Not Installed", nil
	}

	version := strings.TrimSpace(string(output))
	// 解析输出: docker-compose version 1.29.2, build 5becea4c
	if strings.Contains(version, "version") {
		parts := strings.Split(version, " ")
		if len(parts) >= 3 {
			return strings.TrimSuffix(parts[2], ","), nil
		}
	}

	return version, nil
}

