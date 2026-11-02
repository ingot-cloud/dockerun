package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataPathMappingAndBoundary(t *testing.T) {
	root := t.TempDir()
	p := dataPaths{local: root, host: "/engine/volumes/dockerun-data/_data"}
	for _, suffix := range []string{"", "jobs/42/configs/redis.conf", "uploads/name with spaces.tar"} {
		local := filepath.Join(root, suffix)
		host := filepath.Join(p.host, suffix)
		got, err := p.hostPath(local)
		if err != nil || got != host || p.localPath(host) != local {
			t.Fatalf("路径往返转换失败: %q %q %v", local, got, err)
		}
	}
	for _, path := range []string{root + "-other/config", "/opt/business/data", "relative/file"} {
		got, err := p.hostPath(path)
		if err != nil || got != path || p.localPath(path) != path {
			t.Fatalf("转换了数据根目录以外的路径: %q %q %v", path, got, err)
		}
	}
}

func TestDataPathsResolveSymlinksInsideRootAndRejectEscape(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "real"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(root, "inside")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "outside")); err != nil {
		t.Fatal(err)
	}
	p := dataPaths{local: root, host: "/host/data"}
	got, err := p.hostPath(filepath.Join(root, "inside", "not-created-yet"))
	if err != nil || got != "/host/data/real/not-created-yet" {
		t.Fatalf("目录内链接未解析: %s %v", got, err)
	}
	if _, err := p.hostPath(filepath.Join(root, "outside", "not-created-yet")); err == nil {
		t.Fatal("接受了指向数据根目录外的链接")
	}
}
