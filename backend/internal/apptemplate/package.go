package apptemplate

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	MaxArchiveSize      = 32 << 20
	MaxExpandedSize     = 128 << 20
	MaxPackageFiles     = 2048
	MaxPackageTemplates = 100
)

func ValidKey(key string) bool {
	return len(key) <= 100 && templateID.MatchString(key) && !strings.HasPrefix(key, "custom-")
}
func PortableKey(t Template) string {
	if t.Key != "" {
		return t.Key
	}
	return t.ID
}

func documentation(name string) bool {
	if strings.HasPrefix(name, "docs/") {
		return true
	}
	if strings.Contains(name, "/") {
		return false
	}
	n := strings.ToUpper(name)
	return n == "README" || n == "README.MD" || n == "README.TXT" || n == "COPYING" || n == "NOTICE" || n == "NOTICE.TXT" || n == "NOTICE.MD" || n == "LICENSE" || n == "LICENSE.TXT" || n == "LICENSE.MD" || strings.HasSuffix(n, "-LICENSE")
}
func safePackagePath(name string) bool {
	return fs.ValidPath(name) && name != "." && !strings.ContainsAny(name, "\\:\x00") && len(name) <= 512
}
func readDocumentation(root *os.Root) (models.StringMap, error) {
	out := models.StringMap{}
	err := fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if name != "." && name != "docs" && !strings.HasPrefix(name, "docs/") {
				return fs.SkipDir
			}
			return nil
		}
		if !documentation(name) {
			return nil
		}
		if !safePackagePath(name) {
			return fmt.Errorf("说明文件路径无效：%s", name)
		}
		content, err := readTemplateFile(root, name, 8<<20)
		if err == nil {
			out[name] = content
		}
		return err
	})
	return out, err
}

// PackageFiles 统一重建标准文件布局；从不读取环境参数、主机凭据或业务数据。
func PackageFiles(t Template) (models.StringMap, error) {
	key := PortableKey(t)
	if !ValidKey(key) {
		return nil, fmt.Errorf("模板标识无效：%s", key)
	}
	m := manifest{SchemaVersion: 1, ID: key, Name: t.Name, Version: t.Version, Description: t.Description, Category: t.Category, Order: t.Order}
	d := t.Definition
	m.Definition.ComposeFile = "compose.yaml"
	m.Definition.ConfigFiles = map[string]string{}
	m.Definition.Parameters = append([]models.ApplicationParameter{}, d.Parameters...)
	m.Definition.Stateful, m.Definition.Topology = d.Stateful, d.Topology
	m.Definition.Instructions, m.Definition.Storage, m.Definition.Initialization = d.Instructions, d.Storage, d.Initialization
	m.Definition.Endpoints = append([]models.ApplicationEndpoint{}, d.Endpoints...)
	m.Definition.Platform = d.Platform
	if d.Execution != (models.DeploymentExecution{}) {
		e := d.Execution
		m.Definition.Execution = &e
	}
	m.ScriptFiles = map[string]string{}
	files := models.StringMap{"compose.yaml": d.Compose}
	for name, content := range d.Configs {
		if !safePackagePath(name) || !strings.HasPrefix(name, "configs/") {
			return nil, fmt.Errorf("配置文件必须位于 configs/：%s", name)
		}
		files[name] = content
		m.Definition.ConfigFiles[name] = name
	}
	for stage, content := range map[string]string{"before": t.Scripts.Before, "deploy": t.Scripts.Deploy, "after": t.Scripts.After} {
		if content != "" {
			name := "scripts/" + stage + ".sh"
			files[name] = content
			m.ScriptFiles[stage] = name
		}
	}
	for name, content := range t.Attachments {
		if !safePackagePath(name) || !documentation(name) {
			return nil, fmt.Errorf("说明或许可文件路径无效：%s", name)
		}
		if _, exists := files[name]; exists {
			return nil, fmt.Errorf("模板文件路径重复：%s", name)
		}
		files[name] = content
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	files[manifestName] = string(b) + "\n"
	return files, nil
}

func Fingerprint(t Template) (string, error) {
	files, err := PackageFiles(t)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(files)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// ValidatePackage 以真实参数声明准备占位值，只做解析及 bash -n，绝不运行模板代码。
func ValidatePackage(ctx context.Context, t Template) error {
	files, err := PackageFiles(t)
	if err != nil {
		return err
	}
	// SQLite 与 MySQL 使用相同上限，为 mediumtext 及后续应用快照保留空间。
	for _, value := range []any{t.Definition, t.Attachments} {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if len(encoded) > 15<<20 {
			return fmt.Errorf("单个模板的配置或说明内容过大，请拆分模板（序列化上限 15 MiB）")
		}
	}
	dir, err := os.MkdirTemp("", "dockerun-template-validation-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	var total int
	for name, content := range files {
		total += len(content)
		if total > MaxExpandedSize || len(content) > 8<<20 || !utf8.ValidString(content) || strings.ContainsRune(content, 0) {
			return fmt.Errorf("模板文件 %s 超过限制或不是有效 UTF-8 文本", name)
		}
		filename := filepath.Join(dir, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			return err
		}
		if err = os.WriteFile(filename, []byte(content), 0600); err != nil {
			return err
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	_, _, err = loadManifest(root)
	root.Close()
	if err != nil {
		return err
	}
	if err = release.ValidateScripts(t.Scripts); err != nil {
		return err
	}
	if err = release.ValidateExecution(t.Definition.Execution); err != nil {
		return err
	}
	values := models.StringMap{}
	for i, p := range t.Definition.Parameters {
		value := p.Default
		if p.Type == "password" {
			value = strings.Repeat("a", 32)
		}
		if value == "" {
			switch p.Type {
			case "image":
				value = "example/template-check:v1"
			case "port":
				value = fmt.Sprint(20000 + i)
			case "path":
				value = "/srv/dockerun-template-check"
			default:
				value = "dockerun-template-check"
			}
		}
		values[p.Key] = value
	}
	if _, ok := values["NACOS_AUTH_TOKEN"]; ok {
		values["NACOS_AUTH_TOKEN"] = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32)))
	}
	if _, ok := values["CLUSTER_ID"]; ok {
		values["CLUSTER_ID"] = "MkU3OEVBNTcwNTJENDM2Qk"
	}
	values, err = Values(t, values, "https://registry.invalid")
	if err != nil {
		return fmt.Errorf("参数默认值无效：%w", err)
	}
	_, err = release.Parse(ctx, t.Definition.Compose, dir, "dockerun-template-check", values)
	if err != nil {
		return fmt.Errorf("Compose 校验失败：%w", err)
	}
	return nil
}

func ReadPackage(ctx context.Context, data []byte) ([]Template, error) {
	if len(data) > MaxArchiveSize {
		return nil, fmt.Errorf("模板 ZIP 不能超过 32 MiB")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("无法读取模板 ZIP：%w", err)
	}
	if len(archive.File) > MaxPackageFiles {
		return nil, fmt.Errorf("ZIP 文件及目录数量不能超过 %d", MaxPackageFiles)
	}
	files := models.StringMap{}
	seen := map[string]bool{}
	total := 0
	for _, entry := range archive.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(entry.Name, "/")
		if !safePackagePath(name) {
			return nil, fmt.Errorf("ZIP 路径无效：%s", entry.Name)
		}
		if seen[name] {
			return nil, fmt.Errorf("ZIP 包含重复路径：%s", name)
		}
		seen[name] = true
		mode := entry.Mode()
		if mode&os.ModeSymlink != 0 || !mode.IsRegular() && !mode.IsDir() {
			return nil, fmt.Errorf("ZIP 只接受普通文件及目录：%s", name)
		}
		if strings.HasPrefix(name, "__MACOSX/") || path.Base(name) == ".DS_Store" || name == "__MACOSX" {
			continue
		}
		if mode.IsDir() {
			continue
		}
		if entry.Flags&1 != 0 {
			return nil, fmt.Errorf("不支持加密 ZIP：%s", name)
		}
		if entry.UncompressedSize64 > 8<<20 {
			return nil, fmt.Errorf("ZIP 单个文件不能超过 8 MiB：%s", name)
		}
		if entry.UncompressedSize64 > uint64(MaxExpandedSize-total) {
			return nil, fmt.Errorf("ZIP 解压总量不能超过 128 MiB")
		}
		r, err := entry.Open()
		if err != nil {
			return nil, err
		}
		b, readErr := io.ReadAll(io.LimitReader(r, 8<<20+1))
		r.Close()
		if readErr != nil {
			return nil, fmt.Errorf("ZIP 文件读取失败 %s：%w", name, readErr)
		}
		total += len(b)
		if len(b) > 8<<20 || total > MaxExpandedSize || !utf8.Valid(b) || bytes.IndexByte(b, 0) >= 0 {
			return nil, fmt.Errorf("ZIP 文件 %s 超过限制或不是有效 UTF-8 文本", name)
		}
		files[name] = string(b)
	}
	// 兼容 zip 模板根目录、直接压缩 templates/，以及单个模板目录。
	for depth := 0; depth < 4; depth++ {
		direct := false
		for name := range files {
			if name == manifestName || strings.Count(name, "/") == 1 && path.Base(name) == manifestName {
				direct = true
				break
			}
		}
		if direct {
			break
		}
		prefix := ""
		valid := true
		for name := range files {
			if documentation(name) {
				continue
			}
			first, _, ok := strings.Cut(name, "/")
			if !ok || prefix != "" && prefix != first {
				valid = false
				break
			}
			prefix = first
		}
		if !valid || prefix == "" {
			break
		}
		next := models.StringMap{}
		for name, content := range files {
			normalized := strings.TrimPrefix(name, prefix+"/")
			if _, ok := next[normalized]; ok {
				return nil, fmt.Errorf("ZIP 包装目录中存在重复路径：%s", normalized)
			}
			next[normalized] = content
		}
		files = next
	}
	roots := []string{}
	if _, ok := files[manifestName]; ok {
		roots = append(roots, "")
	}
	for name := range files {
		if strings.Count(name, "/") == 1 && path.Base(name) == manifestName {
			roots = append(roots, path.Dir(name))
		}
	}
	sort.Strings(roots)
	if len(roots) == 0 || len(roots) > MaxPackageTemplates {
		return nil, fmt.Errorf("ZIP 必须包含 1 至 %d 个标准模板，每个模板都应有 template.json", MaxPackageTemplates)
	}
	if roots[0] == "" && len(roots) > 1 {
		return nil, fmt.Errorf("不能同时包含根目录单模板和子目录模板")
	}
	temp, err := os.MkdirTemp("", "dockerun-template-package-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temp)
	for name, content := range files {
		p := filepath.Join(temp, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			return nil, fmt.Errorf("ZIP 存在文件与目录冲突：%s", name)
		}
		if err = os.WriteFile(p, []byte(content), 0600); err != nil {
			return nil, err
		}
	}
	out := []Template{}
	identities := map[string]bool{}
	used := map[string]bool{}
	for name := range files {
		if documentation(name) {
			used[name] = true
		}
	}
	for _, directory := range roots {
		prefix := ""
		if directory != "" {
			prefix = directory + "/"
		}
		root, err := os.OpenRoot(filepath.Join(temp, directory))
		if err != nil {
			return nil, err
		}
		t, _, err := loadManifest(root)
		root.Close()
		if err != nil {
			return nil, fmt.Errorf("模板 %s 无效：%w", prefix+manifestName, err)
		}
		var m manifest
		json.Unmarshal([]byte(files[prefix+manifestName]), &m)
		refs := []string{manifestName, m.Definition.ComposeFile}
		for _, name := range m.Definition.ConfigFiles {
			refs = append(refs, name)
		}
		for _, name := range m.ScriptFiles {
			refs = append(refs, name)
		}
		for _, name := range refs {
			used[prefix+name] = true
		}
		for name, content := range files {
			if directory != "" && documentation(name) {
				target := name
				if local, exists := t.Attachments[target]; exists && local != content {
					target = "docs/package/" + name
				}
				if local, exists := t.Attachments[target]; exists && local != content {
					return nil, fmt.Errorf("公共说明与模板说明路径冲突：%s", target)
				}
				t.Attachments[target] = content
			}
			if strings.HasPrefix(name, prefix) && documentation(strings.TrimPrefix(name, prefix)) {
				used[name] = true
			}
		}
		identity := t.Key + "\x00" + t.Version
		if identities[identity] {
			return nil, fmt.Errorf("ZIP 中模板标识和版本重复：%s · %s", t.Key, t.Version)
		}
		identities[identity] = true
		if err = ValidatePackage(ctx, t); err != nil {
			return nil, fmt.Errorf("模板 %s · %s 无效：%w", t.Key, t.Version, err)
		}
		out = append(out, t)
	}
	for name := range files {
		if !used[name] {
			return nil, fmt.Errorf("ZIP 包含未声明的文件：%s", name)
		}
	}
	return out, nil
}

func WritePackage(ctx context.Context, items []Template) ([]byte, error) {
	if len(items) == 0 || len(items) > MaxPackageTemplates {
		return nil, fmt.Errorf("请选择 1 至 %d 个模板", MaxPackageTemplates)
	}
	counts := map[string]int{}
	for _, item := range items {
		counts[PortableKey(item)]++
	}
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	total, entries := 0, 0
	identities := map[string]bool{}
	for _, t := range items {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := ValidatePackage(ctx, t); err != nil {
			return nil, fmt.Errorf("模板 %s · %s 无法下载：%w", t.Name, t.Version, err)
		}
		files, err := PackageFiles(t)
		if err != nil {
			return nil, err
		}
		fingerprint, err := Fingerprint(t)
		if err != nil {
			return nil, err
		}
		identity := PortableKey(t) + "\x00" + t.Version
		if identities[identity] {
			return nil, fmt.Errorf("下载列表含有相同标识和版本，请分别下载：%s", PortableKey(t))
		}
		identities[identity] = true
		dir := PortableKey(t)
		if counts[dir] > 1 {
			dir += "-" + fingerprint[:12]
		}
		names := make([]string, 0, len(files))
		for name := range files {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			content := files[name]
			total += len(content)
			entries++
			if total > MaxExpandedSize || entries > MaxPackageFiles {
				return nil, fmt.Errorf("模板包内容过大，请减少本次选择的模板")
			}
			header := &zip.FileHeader{Name: dir + "/" + name, Method: zip.Deflate}
			header.SetMode(0644)
			file, err := writer.CreateHeader(header)
			if err != nil {
				return nil, err
			}
			if _, err = file.Write([]byte(content)); err != nil {
				return nil, err
			}
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	if buf.Len() > MaxArchiveSize {
		return nil, fmt.Errorf("模板 ZIP 超过 32 MiB，请减少本次选择的模板")
	}
	return buf.Bytes(), nil
}
