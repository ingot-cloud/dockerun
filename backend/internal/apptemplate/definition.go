package apptemplate

import (
	"dockerun/internal/models"
	"dockerun/internal/utils"
	"github.com/distribution/reference"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var placeholders = regexp.MustCompile(`(^|[^$])\$\{([A-Za-z_][A-Za-z0-9_]*)([^}]*)\}`)
var identifier = regexp.MustCompile(`[^A-Za-z0-9_]`)

func Parameters(content string, existing []models.ApplicationParameter) []models.ApplicationParameter {
	out := append([]models.ApplicationParameter{}, existing...)
	known := map[string]bool{}
	for _, p := range out {
		known[p.Key] = true
	}
	for _, match := range placeholders.FindAllStringSubmatch(content, -1) {
		key := match[2]
		if known[key] || key == "NACOS_GRPC_PORT" {
			continue
		}
		known[key] = true
		p := models.ApplicationParameter{Key: key, Label: key, Type: "string", Required: true}
		if strings.HasPrefix(match[3], ":-") {
			p.Default = strings.TrimPrefix(match[3], ":-")
			p.Required = false
		}
		if utils.SensitiveParameterName(key) {
			p.Type = "password"
			p.Default = ""
			p.Required = true
		}
		if strings.HasSuffix(key, "IMAGE") {
			p.Type = "image"
		}
		out = append(out, p)
	}
	return out
}

// ReusableDefinition 保留占位符，移除实例专属的固定凭据、容器名称和数据路径。
func ReusableDefinition(def models.ApplicationDefinition) (models.ApplicationDefinition, error) {
	def.Parameters = append([]models.ApplicationParameter{}, def.Parameters...)
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(def.Compose), &doc); err != nil {
		return def, err
	}
	add := func(key, label, kind, value string) string {
		for _, p := range def.Parameters {
			if p.Key == key {
				return "${" + key + "}"
			}
		}
		def.Parameters = append(def.Parameters, models.ApplicationParameter{Key: key, Label: label, Type: kind, Default: value, Required: true})
		return "${" + key + "}"
	}
	services, _ := doc["services"].(map[string]any)
	for name, value := range services {
		svc, ok := value.(map[string]any)
		if !ok {
			continue
		}
		prefix := strings.ToUpper(identifier.ReplaceAllString(name, "_"))
		delete(svc, "container_name")
		if image, ok := svc["image"].(string); ok && !strings.Contains(image, "${") {
			if n, err := reference.ParseNormalizedNamed(image); err == nil {
				image = strings.TrimPrefix(n.String(), reference.Domain(n)+"/")
			}
			svc["image"] = add(prefix+"_IMAGE", name+" 镜像", "image", image)
		}
		if env, ok := svc["environment"].(map[string]any); ok {
			for key, v := range env {
				if utils.SensitiveParameterName(key) {
					if text, ok := v.(string); !ok || !strings.Contains(text, "${") {
						env[key] = add(prefix+"_"+identifier.ReplaceAllString(key, "_"), key, "password", "")
					}
				}
			}
		} else if env, ok := svc["environment"].([]any); ok {
			for i, v := range env {
				if text, ok := v.(string); ok {
					key, value, hasValue := strings.Cut(text, "=")
					if hasValue && utils.SensitiveParameterName(key) && !strings.Contains(value, "${") {
						env[i] = key + "=" + add(prefix+"_"+identifier.ReplaceAllString(key, "_"), key, "password", "")
					}
				}
			}
		}
		if volumes, ok := svc["volumes"].([]any); ok {
			for i, v := range volumes {
				key := prefix + "_DATA_DIR_" + strconv.Itoa(i+1)
				if text, ok := v.(string); ok && strings.HasPrefix(text, "/") {
					_, target, _ := strings.Cut(text, ":")
					volumes[i] = add(key, name+" 数据目录", "path", "") + ":" + target
				} else if mount, ok := v.(map[string]any); ok && mount["type"] == "bind" {
					if source, ok := mount["source"].(string); ok && strings.HasPrefix(source, "/") {
						mount["source"] = add(key, name+" 数据目录", "path", "")
					}
				}
			}
		}
	}
	if volumes, ok := doc["volumes"].(map[string]any); ok {
		for name, value := range volumes {
			if volume, ok := value.(map[string]any); ok {
				if external, ok := volume["external"].(bool); ok && external {
					volume["name"] = add("EXTERNAL_VOLUME_"+strings.ToUpper(identifier.ReplaceAllString(name, "_")), name+" 外部数据卷", "string", "")
				} else {
					delete(volume, "name")
				}
			}
		}
	}
	def.Compose = encodeYAML(doc)
	def.Compose = placeholders.ReplaceAllStringFunc(def.Compose, func(match string) string {
		parts := placeholders.FindStringSubmatch(match)
		if utils.SensitiveParameterName(parts[2]) {
			return parts[1] + "${" + parts[2] + ":?请填写此环境的凭据}"
		}
		return match
	})
	def.Parameters = Parameters(def.Compose, def.Parameters)
	for i := range def.Parameters {
		if def.Parameters[i].Type == "password" || utils.SensitiveParameterName(def.Parameters[i].Key) {
			def.Parameters[i].Default = ""
			def.Parameters[i].Type = "password"
		}
		if def.Parameters[i].Type == "path" {
			def.Parameters[i].Default = ""
		}
	}
	return def, nil
}

func RedactParameters(values models.StringMap, schema []models.ApplicationParameter) models.StringMap {
	copy := models.StringMap{}
	for k, v := range values {
		copy[k] = v
		if utils.SensitiveParameterName(k) {
			copy[k] = "******"
		}
	}
	for _, p := range schema {
		if p.Type == "password" {
			copy[p.Key] = "******"
		}
	}
	return copy
}
