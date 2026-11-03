package apptemplate

import (
	"bytes"
	"dockerun/internal/models"
	"dockerun/internal/utils"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const manifestName = "template.json"

type manifest struct {
	SchemaVersion int    `json:"schema_version"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	Order         int    `json:"order"`
	Definition    struct {
		ComposeFile    string                        `json:"compose_file"`
		ConfigFiles    map[string]string             `json:"config_files"`
		Parameters     []models.ApplicationParameter `json:"parameters"`
		Stateful       bool                          `json:"stateful"`
		Topology       string                        `json:"topology"`
		Instructions   string                        `json:"instructions"`
		Storage        string                        `json:"storage"`
		Initialization string                        `json:"initialization"`
		Endpoints      []models.ApplicationEndpoint  `json:"endpoints"`
		Platform       string                        `json:"platform,omitempty"`
		Execution      *models.DeploymentExecution   `json:"execution,omitempty"`
	} `json:"definition"`
	ScriptFiles map[string]string `json:"script_files"`
}

var templateID = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// LoadDir 每次重新读取目录；错误包含模板目录，避免静默丢失模板。
// 只有 template.json 参与应用模板加载，meta.json 继续供历史脚本使用。
func LoadDir(dir string) ([]Template, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("无法读取应用模板目录 %s：%w", dir, err)
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return nil, fmt.Errorf("无法列出应用模板目录 %s：%w", dir, err)
	}
	items := []Template{}
	orders, ids := map[string]int{}, map[string]string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		templateRoot, err := root.OpenRoot(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("无法打开模板 %s：%w", entry.Name(), err)
		}
		if _, err := templateRoot.Lstat(manifestName); os.IsNotExist(err) {
			templateRoot.Close()
			continue
		}
		t, order, err := loadManifest(templateRoot)
		templateRoot.Close()
		if err != nil {
			return nil, fmt.Errorf("应用模板 %s/%s 无效：%w", entry.Name(), manifestName, err)
		}
		if previous, ok := ids[t.ID]; ok {
			return nil, fmt.Errorf("应用模板标识 %s 重复：%s、%s", t.ID, previous, entry.Name())
		}
		// Nacos SQL 的完整许可位于模板根目录，下载时随对应模板一起传递。
		if _, ok := t.Definition.Configs["configs/nacos-mysql.sql"]; ok {
			if _, err := root.Stat("NACOS-LICENSE"); err == nil {
				license, err := readTemplateFile(root, "NACOS-LICENSE", 8<<20)
				if err != nil {
					return nil, err
				}
				t.Attachments["NACOS-LICENSE"] = license
			}
		}
		ids[t.ID], orders[t.ID] = entry.Name(), order
		items = append(items, t)
	}
	sort.Slice(items, func(i, j int) bool {
		if orders[items[i].ID] != orders[items[j].ID] {
			return orders[items[i].ID] < orders[items[j].ID]
		}
		return items[i].ID < items[j].ID
	})
	return items, nil
}

func readTemplateFile(root *os.Root, name string, limit int64) (string, error) {
	if !fs.ValidPath(name) || name == "." || strings.ContainsAny(name, "\\\x00") {
		return "", fmt.Errorf("文件路径必须是模板目录内的相对路径：%s", name)
	}
	f, err := root.Open(name)
	if err != nil {
		return "", fmt.Errorf("无法读取文件 %s：%w", name, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("模板文件必须为普通文件：%s", name)
	}
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return "", err
	}
	if int64(len(b)) > limit || bytes.IndexByte(b, 0) >= 0 {
		return "", fmt.Errorf("模板文件 %s 超过 %d 字节或包含空字符", name, limit)
	}
	return string(b), nil
}

func loadManifest(root *os.Root) (Template, int, error) {
	var m manifest
	content, err := readTemplateFile(root, manifestName, 1<<20)
	if err != nil {
		return Template{}, 0, err
	}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&m); err != nil {
		return Template{}, 0, fmt.Errorf("元数据解析失败：%w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Template{}, 0, fmt.Errorf("元数据只能包含一个 JSON 对象")
	}
	if m.SchemaVersion != 1 {
		return Template{}, 0, fmt.Errorf("schema_version 必须为 1")
	}
	if !templateID.MatchString(m.ID) || strings.HasPrefix(m.ID, "custom-") || len(m.ID) > 100 {
		return Template{}, 0, fmt.Errorf("id 必须为小写字母、数字及连字符，不能以 custom- 开头，最多 100 字符")
	}
	if strings.TrimSpace(m.Name) == "" || strings.TrimSpace(m.Version) == "" || len(m.Name) > 100 || len(m.Version) > 100 {
		return Template{}, 0, fmt.Errorf("必须填写名称和版本，且各不超过 100 字符")
	}
	if m.Category != "business" && m.Category != "middleware" {
		return Template{}, 0, fmt.Errorf("category 必须为 business 或 middleware")
	}
	d := m.Definition
	if d.Topology != "single_host" && d.Topology != "replicated" {
		return Template{}, 0, fmt.Errorf("topology 必须为 single_host 或 replicated")
	}
	keys := map[string]bool{}
	for _, p := range d.Parameters {
		if !variable.MatchString(p.Key) || keys[p.Key] || strings.TrimSpace(p.Label) == "" {
			return Template{}, 0, fmt.Errorf("参数名称无效、重复或缺少中文说明：%s", p.Key)
		}
		keys[p.Key] = true
		switch p.Type {
		case "string", "image", "port", "path", "password":
		default:
			return Template{}, 0, fmt.Errorf("参数 %s 的类型不受支持：%s", p.Key, p.Type)
		}
		if utils.SensitiveParameterName(p.Key) && p.Type != "password" || p.Type == "password" && p.Default != "" {
			return Template{}, 0, fmt.Errorf("敏感参数 %s 必须为 password 类型且不设默认值", p.Key)
		}
	}
	compose, err := readTemplateFile(root, d.ComposeFile, 2<<20)
	if err != nil {
		return Template{}, 0, err
	}
	var raw struct {
		Services map[string]any `yaml:"services"`
	}
	if err := yaml.Unmarshal([]byte(compose), &raw); err != nil || len(raw.Services) == 0 {
		return Template{}, 0, fmt.Errorf("Compose 必须包含有效的 services 定义")
	}
	for _, match := range placeholders.FindAllStringSubmatch(compose, -1) {
		key := match[2]
		if key == "NACOS_GRPC_PORT" && keys["NACOS_PORT"] {
			continue
		}
		if !keys[key] {
			return Template{}, 0, fmt.Errorf("Compose 参数 %s 未在 parameters 中定义", key)
		}
	}
	configs := models.StringMap{}
	for target, source := range d.ConfigFiles {
		if !fs.ValidPath(target) || !strings.HasPrefix(target, "configs/") || strings.ContainsAny(target, "\\\x00") {
			return Template{}, 0, fmt.Errorf("配置目标路径必须位于 configs/：%s", target)
		}
		for other := range d.ConfigFiles {
			if strings.HasPrefix(target, other+"/") {
				return Template{}, 0, fmt.Errorf("配置目标路径存在文件与目录冲突：%s、%s", target, other)
			}
		}
		configs[target], err = readTemplateFile(root, source, 8<<20)
		if err != nil {
			return Template{}, 0, err
		}
	}
	scripts := models.DeploymentScripts{}
	for stage, source := range m.ScriptFiles {
		var target *string
		switch stage {
		case "before":
			target = &scripts.Before
		case "deploy":
			target = &scripts.Deploy
		case "after":
			target = &scripts.After
		default:
			return Template{}, 0, fmt.Errorf("未知附加脚本阶段：%s", stage)
		}
		*target, err = readTemplateFile(root, source, 256<<10)
		if err != nil {
			return Template{}, 0, err
		}
	}
	for _, endpoint := range d.Endpoints {
		if !keys[endpoint.Port] || endpoint.Name == "" || endpoint.Protocol == "" {
			return Template{}, 0, fmt.Errorf("访问入口必须填写名称、协议并引用已定义的端口参数")
		}
	}
	attachments, err := readDocumentation(root)
	if err != nil {
		return Template{}, 0, err
	}
	execution := models.DeploymentExecution{}
	if d.Execution != nil {
		execution = *d.Execution
	}
	return Template{ID: m.ID, Key: m.ID, Order: m.Order, Name: m.Name, Version: m.Version, Description: m.Description, Category: m.Category, Source: "builtin", Scripts: scripts, Attachments: attachments,
		Definition: models.ApplicationDefinition{Compose: compose, Configs: configs, Parameters: d.Parameters, Stateful: d.Stateful, Topology: d.Topology, Instructions: d.Instructions, Storage: d.Storage, Initialization: d.Initialization, Endpoints: d.Endpoints, Platform: d.Platform, Execution: execution}}, m.Order, nil
}
