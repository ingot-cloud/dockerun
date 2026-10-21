package bundle_test

import (
	"archive/tar"
	"compress/gzip"
	"dockerun/internal/bundle"
	"os"
	"path/filepath"
	"testing"
)

func TestRejectUnsafeArchives(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		kind       byte
	}{{"parent", "../escape", tar.TypeReg}, {"absolute", "/escape", tar.TypeReg}, {"symlink", "configs/link", tar.TypeSymlink}, {"unknown", "install.sh", tar.TypeReg}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "bad.tar.gz")
			f, _ := os.Create(file)
			gz := gzip.NewWriter(f)
			tw := tar.NewWriter(gz)
			tw.WriteHeader(&tar.Header{Name: tc.path, Typeflag: tc.kind, Size: 0, Linkname: "/tmp/escape"})
			tw.Close()
			gz.Close()
			f.Close()
			target := filepath.Join(dir, "out")
			os.Mkdir(target, 0700)
			if _, err := bundle.Extract(file, target); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
}
