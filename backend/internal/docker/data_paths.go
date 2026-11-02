package docker

import (
	"context"
	"dockerun/internal/config"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// dataPaths 只转换 Dockerun 自身数据根目录，不改业务容器的目标路径或其他宿主路径。
type dataPaths struct {
	local string
	host  string
}

func resolveDataPaths(ctx context.Context) (*dataPaths, error) {
	cfg := config.Load()
	if cfg.HostDataDir != "" && cfg.DataVolume != "" {
		return nil, fmt.Errorf("HOST_DATA_DIR 与 DATA_VOLUME 不能同时设置")
	}
	host := cfg.HostDataDir
	if cfg.DataVolume != "" {
		volume, err := Client.VolumeInspect(ctx, cfg.DataVolume)
		if err != nil {
			return nil, fmt.Errorf("读取数据命名卷 %s 失败: %w", cfg.DataVolume, err)
		}
		host = volume.Mountpoint
		if host == "" {
			return nil, fmt.Errorf("数据命名卷 %s 没有可用的宿主挂载路径", cfg.DataVolume)
		}
	}
	if host == "" {
		// 原生进程和旧版同路径绑定沿用原行为，不启用代理。
		host = cfg.DataDir
	}
	if !filepath.IsAbs(host) || filepath.Clean(host) == "/" {
		return nil, fmt.Errorf("数据存储的宿主路径必须是非根目录的绝对路径")
	}
	return &dataPaths{local: filepath.Clean(cfg.DataDir), host: filepath.Clean(host)}, nil
}

func relativePath(root, path string) (string, bool) {
	if !filepath.IsAbs(path) {
		return "", false
	}
	rel, err := filepath.Rel(root, filepath.Clean(path))
	return rel, err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func (p dataPaths) localPath(path string) string {
	if _, ok := relativePath(p.local, path); ok {
		return filepath.Clean(path)
	}
	if rel, ok := relativePath(p.host, path); ok {
		return filepath.Join(p.local, rel)
	}
	return path
}

func (p dataPaths) hostPath(path string) (string, error) {
	rel, ok := relativePath(p.local, path)
	if !ok || p.local == p.host {
		return path, nil
	}
	// 解析已存在的父目录，也支持尚未创建的绑定目录，避免符号链接逃出数据根目录。
	resolved := filepath.Join(p.local, rel)
	var tail []string
	for {
		actual, err := filepath.EvalSymlinks(resolved)
		if err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				actual = filepath.Join(actual, tail[i])
			}
			root, rootErr := filepath.EvalSymlinks(p.local)
			if rootErr != nil {
				return "", rootErr
			}
			rel, ok = relativePath(root, actual)
			if !ok {
				return "", fmt.Errorf("数据目录内的挂载源不能通过符号链接指向目录外: %s", path)
			}
			return filepath.Join(p.host, rel), nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("检查数据挂载源失败: %w", err)
		}
		if resolved == p.local || filepath.Dir(resolved) == resolved {
			return "", fmt.Errorf("容器数据目录不存在: %s", p.local)
		}
		tail = append(tail, filepath.Base(resolved))
		resolved = filepath.Dir(resolved)
	}
}

// LocalDataPath 兼容旧安装中保存的宿主数据路径。
func LocalDataPath(path string) string {
	bridgeMu.RLock()
	defer bridgeMu.RUnlock()
	if bridge != nil {
		return bridge.paths.localPath(path)
	}
	return path
}

func HostDataPath(path string) (string, error) {
	bridgeMu.RLock()
	defer bridgeMu.RUnlock()
	if bridge != nil {
		return bridge.paths.hostPath(path)
	}
	return path, nil
}
