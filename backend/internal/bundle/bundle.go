package bundle

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"dockerun/internal/docker"
	"dockerun/internal/release"
	"encoding/hex"
	"encoding/json"
	"fmt"
	digest "github.com/opencontainers/go-digest"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const MaxSize int64 = 100 << 30
const MaxFiles = 10000

type Manifest struct {
	SchemaVersion        int               `json:"schema_version"`
	Application          string            `json:"application"`
	Version              string            `json:"version"`
	Platform             string            `json:"platform"`
	Images               []release.Image   `json:"images"`
	Defaults             map[string]string `json:"defaults,omitempty"`
	ExternalDependencies []string          `json:"external_dependencies,omitempty"`
	GitCommit            string            `json:"git_commit,omitempty"`
}

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	_, err = io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), err
}
func Extract(file, dir string) (*Manifest, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var size int64
	seen := map[string]bool{}
	for i := 0; ; i++ {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if i >= MaxFiles {
			return nil, fmt.Errorf("发布包文件过多")
		}
		name := strings.TrimPrefix(h.Name, "./")
		path, err := release.SafePath(dir, name)
		if err != nil {
			return nil, err
		}
		if seen[name] {
			return nil, fmt.Errorf("重复路径 %s", name)
		}
		seen[name] = true
		if h.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(path, 0700); err != nil {
				return nil, err
			}
			continue
		}
		if h.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("不允许链接、设备或特殊文件: %s", name)
		}
		if name != "release.json" && name != "compose.yaml" && name != "env.example" && name != "checksums.sha256" && !strings.HasPrefix(name, "images/") && !strings.HasPrefix(name, "configs/") {
			return nil, fmt.Errorf("不允许的发布包文件 %s", name)
		}
		size += h.Size
		if size > MaxSize || h.Size < 0 {
			return nil, fmt.Errorf("解包大小超限")
		}
		if !strings.HasPrefix(name, "images/") && h.Size > 8<<20 {
			return nil, fmt.Errorf("配置文件过大")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, err
		}
		out, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		_, err = io.CopyN(out, tr, h.Size)
		ce := out.Close()
		if err != nil {
			return nil, err
		}
		if ce != nil {
			return nil, ce
		}
	}
	// Drain gzip to verify its checksum even if tar reached its end markers.
	if _, err := io.Copy(io.Discard, gz); err != nil {
		return nil, err
	}
	sums, err := os.Open(filepath.Join(dir, "checksums.sha256"))
	if err != nil {
		return nil, err
	}
	defer sums.Close()
	checked := map[string]bool{}
	s := bufio.NewScanner(sums)
	for s.Scan() {
		line := s.Text()
		if len(line) < 67 {
			return nil, fmt.Errorf("校验清单格式错误")
		}
		hash, name := line[:64], strings.TrimPrefix(line[66:], "*")
		if checked[name] || !seen[name] || name == "checksums.sha256" {
			return nil, fmt.Errorf("校验清单路径无效: %s", name)
		}
		p, err := release.SafePath(dir, name)
		if err != nil {
			return nil, err
		}
		actual, err := HashFile(p)
		if err != nil || actual != hash {
			return nil, fmt.Errorf("文件 SHA-256 不匹配: %s", name)
		}
		checked[name] = true
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	for name := range seen {
		p, _ := release.SafePath(dir, name)
		info, _ := os.Stat(p)
		if info.IsDir() || name == "checksums.sha256" {
			continue
		}
		if !checked[name] {
			return nil, fmt.Errorf("文件缺少校验值: %s", name)
		}
	}
	b, err := os.ReadFile(filepath.Join(dir, "release.json"))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err = json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m.SchemaVersion != 1 {
		return nil, fmt.Errorf("不支持 schema_version=%d", m.SchemaVersion)
	}
	if m.Platform != "linux/amd64" && m.Platform != "linux/arm64" {
		return nil, fmt.Errorf("不支持目标平台 %s", m.Platform)
	}
	if m.Application == "" || len(m.Application) > 100 || m.Version == "" || len(m.Version) > 200 || len(m.Images) == 0 || len(m.Images) > 200 {
		return nil, fmt.Errorf("发布包元数据不完整")
	}
	names := map[string]bool{}
	for _, im := range m.Images {
		if names[im.Reference] || !strings.HasPrefix(im.Archive, "images/") || !checked[im.Archive] {
			return nil, fmt.Errorf("镜像归档无效: %s", im.Reference)
		}
		names[im.Reference] = true
		if digest.Digest(im.ID).Validate() != nil || im.Platform != m.Platform {
			return nil, fmt.Errorf("镜像身份或平台无效")
		}
		path, _ := release.SafePath(dir, im.Archive)
		actual, _ := HashFile(path)
		if actual != im.SHA256 {
			return nil, fmt.Errorf("镜像归档校验失败")
		}
	}
	compose, err := os.ReadFile(filepath.Join(dir, "compose.yaml"))
	if err != nil {
		return nil, err
	}
	p, err := release.Parse(context.Background(), string(compose), dir, "dockerun-package", m.Defaults)
	if err != nil {
		return nil, err
	}
	for _, ref := range release.References(p) {
		if !names[ref] {
			return nil, fmt.Errorf("发布包缺少服务镜像 %s", ref)
		}
	}
	if _, err := Configs(dir); err != nil {
		return nil, err
	}
	return &m, nil
}
func Load(ctx context.Context, dir string, m Manifest, log func(string)) error {
	platform, err := release.Platform(ctx)
	if err != nil {
		return err
	}
	if platform != m.Platform {
		return fmt.Errorf("发布包平台 %s 与目标主机 %s 不符", m.Platform, platform)
	}
	for _, im := range m.Images {
		f, err := os.Open(filepath.Join(dir, im.Archive))
		if err != nil {
			return err
		}
		resp, err := docker.GetClient().ImageLoad(ctx, f, false)
		if err != nil {
			f.Close()
			return err
		}
		err = release.Consume(resp.Body, log)
		resp.Body.Close()
		f.Close()
		if err != nil {
			return err
		}
		v, _, err := docker.GetClient().ImageInspectWithRaw(ctx, im.ID)
		if err != nil || v.ID != im.ID || v.Os+"/"+v.Architecture != m.Platform {
			return fmt.Errorf("加载结果与发布包镜像身份不符: %s", im.Reference)
		}
	}
	return nil
}
func Configs(dir string) (map[string]string, error) {
	out := map[string]string{}
	var total int64
	err := filepath.WalkDir(filepath.Join(dir, "configs"), func(p string, d os.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("配置不接受符号链接: %s", p)
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		if total > 8<<20 {
			return fmt.Errorf("非敏感配置合计大小超过 8 MiB")
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if !utf8.Valid(b) {
			return fmt.Errorf("配置必须是 UTF-8 文本: %s", rel)
		}
		out[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	return out, err
}
func Pack(ctx context.Context, compose, output string, m Manifest) error {
	if m.Platform != "linux/amd64" && m.Platform != "linux/arm64" {
		return fmt.Errorf("请指定 linux/amd64 或 linux/arm64")
	}
	dir, err := os.MkdirTemp("", "dockerun-package-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	os.MkdirAll(filepath.Join(dir, "images"), 0700)
	content, err := os.ReadFile(compose)
	if err != nil {
		return err
	}
	p, err := release.Parse(ctx, string(content), filepath.Dir(compose), "dockerun-package", m.Defaults)
	if err != nil {
		return err
	}
	frozen, err := release.FreezeImages(string(content), p)
	if err != nil {
		return err
	}
	content = []byte(frozen)
	m.SchemaVersion = 1
	m.Images = nil
	for idx, ref := range release.References(p) {
		v, _, err := docker.GetClient().ImageInspectWithRaw(ctx, ref)
		if err != nil {
			return err
		}
		if v.Os+"/"+v.Architecture != m.Platform {
			return fmt.Errorf("镜像 %s 平台错误", ref)
		}
		name := fmt.Sprintf("images/%d.tar", idx)
		path := filepath.Join(dir, name)
		stream, err := docker.GetClient().ImageSave(ctx, []string{v.ID})
		if err != nil {
			return err
		}
		out, err := os.Create(path)
		if err != nil {
			stream.Close()
			return err
		}
		_, err = io.Copy(out, stream)
		ce := out.Close()
		stream.Close()
		if err != nil {
			return err
		}
		if ce != nil {
			return ce
		}
		hash, err := HashFile(path)
		if err != nil {
			return err
		}
		m.Images = append(m.Images, release.Image{Reference: ref, ID: v.ID, Platform: m.Platform, Archive: name, SHA256: hash})
	}
	// Only explicitly referenced non-sensitive config files are copied.
	for _, c := range p.Configs {
		if c.File != "" {
			rel, err := filepath.Rel(filepath.Dir(compose), c.File)
			if err != nil || !strings.HasPrefix(filepath.ToSlash(rel), "configs/") {
				return fmt.Errorf("配置必须位于 Compose 同级 configs/ 中")
			}
			dst, _ := release.SafePath(dir, rel)
			os.MkdirAll(filepath.Dir(dst), 0700)
			b, err := os.ReadFile(c.File)
			if err != nil {
				return err
			}
			if err = os.WriteFile(dst, b, 0600); err != nil {
				return err
			}
		}
	}
	sourceConfigs, err := Configs(filepath.Dir(compose))
	if err != nil {
		return err
	}
	if err = release.WriteConfigs(dir, sourceConfigs); err != nil {
		return err
	}
	os.WriteFile(filepath.Join(dir, "compose.yaml"), content, 0600)
	b, _ := json.MarshalIndent(m, "", "  ")
	os.WriteFile(filepath.Join(dir, "release.json"), b, 0600)
	var env strings.Builder
	keys := []string{}
	for k := range m.Defaults {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&env, "%s=%s\n", k, m.Defaults[k])
	}
	os.WriteFile(filepath.Join(dir, "env.example"), []byte(env.String()), 0600)
	files := []string{}
	err = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			r, _ := filepath.Rel(dir, p)
			files = append(files, filepath.ToSlash(r))
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(files)
	var sums strings.Builder
	for _, f := range files {
		hash, err := HashFile(filepath.Join(dir, f))
		if err != nil {
			return err
		}
		fmt.Fprintf(&sums, "%s  %s\n", hash, f)
	}
	os.WriteFile(filepath.Join(dir, "checksums.sha256"), []byte(sums.String()), 0600)
	files = append(files, "checksums.sha256")
	out, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)
	for _, name := range files {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		st, _ := f.Stat()
		h := &tar.Header{Name: name, Mode: 0600, Size: st.Size(), Typeflag: tar.TypeReg}
		if err = tw.WriteHeader(h); err == nil {
			_, err = io.Copy(tw, f)
		}
		f.Close()
		if err != nil {
			return err
		}
	}
	if err = tw.Close(); err != nil {
		return err
	}
	if err = gz.Close(); err != nil {
		return err
	}
	return out.Close()
}
