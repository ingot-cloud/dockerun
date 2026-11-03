package apptemplate

import (
	"dockerun/internal/models"
	"strings"

	"gopkg.in/yaml.v3"
)

// 参数约束取决于实际部署定义，不依赖本机记录编号或模板名称，另存及跨实例导入后仍然有效。
func parameterRules(def models.ApplicationDefinition) map[string]string {
	rules := map[string]string{}
	parameter := func(value string) string {
		matches := placeholders.FindStringSubmatch(value)
		if len(matches) > 2 {
			return matches[2]
		}
		return ""
	}
	for _, content := range def.Configs {
		var doc struct {
			Ingot struct {
				Crypto struct {
					SecretKeys map[string]string `yaml:"secretKeys"`
				} `yaml:"crypto"`
				Security struct {
					JWK struct {
						MasterKey string `yaml:"master-key"`
					} `yaml:"jwk"`
				} `yaml:"security"`
			} `yaml:"ingot"`
		}
		if yaml.Unmarshal([]byte(content), &doc) != nil {
			continue
		}
		for _, name := range []string{"aes", "aes_gcm"} {
			if key := parameter(doc.Ingot.Crypto.SecretKeys[name]); key != "" {
				rules[key] = "aes"
			}
		}
		if key := parameter(doc.Ingot.Security.JWK.MasterKey); key != "" {
			rules[key] = "aes"
		}
	}
	defaults := map[string]string{}
	for _, p := range def.Parameters {
		defaults[p.Key] = p.Default
	}
	var doc struct {
		Services map[string]struct {
			Image       string            `yaml:"image"`
			Environment map[string]string `yaml:"environment"`
		} `yaml:"services"`
	}
	if yaml.Unmarshal([]byte(def.Compose), &doc) == nil {
		for _, svc := range doc.Services {
			image := svc.Image
			if key := parameter(image); key != "" {
				image = defaults[key]
			}
			if image != "mysql:8.0.44" && !strings.HasSuffix(image, "/mysql:8.0.44") {
				continue
			}
			if svc.Environment["MYSQL_ROOT_PASSWORD"] == "" {
				continue
			}
			for _, name := range []string{"MYSQL_ROOT_PASSWORD", "MYSQL_PASSWORD"} {
				if key := parameter(svc.Environment[name]); key != "" {
					rules[key] = "mysql_password"
				}
			}
		}
	}
	return rules
}
