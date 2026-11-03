package apptemplate

import (
	"context"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"
)

func testTemplates(t *testing.T) []Template {
	t.Helper()
	items, err := LoadDir(filepath.Join("..", "..", "..", "templates"))
	if err != nil {
		t.Fatal(err)
	}
	return items
}

func TestBuiltinTemplatesParseWithPersistentScopedVolumes(t *testing.T) {
	for _, tmpl := range testTemplates(t) {
		t.Run(tmpl.ID, func(t *testing.T) {
			provided := models.StringMap{}
			for _, p := range tmpl.Definition.Parameters {
				if p.Required && p.Default == "" {
					provided[p.Key] = "test-value"
					if p.Type == "path" {
						provided[p.Key] = "/srv/template-test"
					}
					if strings.HasSuffix(p.Key, "_HOST") || strings.HasSuffix(p.Key, "_ADDR") {
						provided[p.Key] = "host.test"
					}
				}
				if p.Type == "password" {
					provided[p.Key] = "test-'$password"
					if strings.HasPrefix(tmpl.ID, "ingot-") && (p.Key == "AUTH_JWK_MASTER_KEY" || p.Key == "CRYPTO_AES_KEY" || p.Key == "CRYPTO_AES_GCM_KEY") {
						provided[p.Key] = strings.Repeat("a", 32)
					}
					if (tmpl.ID == "mysql" || tmpl.ID == "nacos-cluster") && strings.HasPrefix(p.Key, "MYSQL_") {
						provided[p.Key] = "test-$password:42"
					}
					if p.Default != "" {
						t.Fatal("template contains a default credential")
					}
				}
			}
			provided["NACOS_AUTH_TOKEN"] = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32)))
			values, err := Values(tmpl, provided, "https://registry.test:5443")
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if err := release.WriteConfigs(dir, tmpl.Definition.Configs); err != nil {
				t.Fatal(err)
			}
			p, err := release.Parse(context.Background(), tmpl.Definition.Compose, dir, "dockerun-a1-e2", values)
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Services) == 0 {
				t.Fatal("empty template")
			}
			for _, svc := range p.Services {
				if !strings.HasPrefix(svc.Image, "registry.test:5443/") {
					t.Fatal("template did not use the selected registry")
				}
				if svc.ContainerName != "" {
					t.Fatal("fixed container names collide across apps")
				}
			}
			for _, v := range p.Volumes {
				if bool(v.External) || !strings.HasPrefix(v.Name, "dockerun-a1-e2_") {
					t.Fatal("data volume is not scoped to its application environment")
				}
			}
			if len(p.Volumes) > 0 && !tmpl.Definition.Stateful {
				t.Fatal("含持久化数据卷的预设不能标记为无状态")
			}
			if strings.HasPrefix(tmpl.ID, "kafka-") {
				for name, svc := range p.Services {
					if name == "akhq" {
						continue
					}
					if svc.Environment["CLUSTER_ID"] == nil || *svc.Environment["CLUSTER_ID"] != values["CLUSTER_ID"] {
						t.Fatal("Kafka cluster ID was not passed to the official image")
					}
					if len(svc.Ports) != 1 {
						t.Fatal("Kafka controller port must stay internal")
					}
				}
			}
			if tmpl.ID == "registry" && p.Services["registry"].Volumes[0].Target != "/var/lib/registry" {
				t.Fatal("registry images are not persisted")
			}
			if tmpl.ID == "nacos-cluster" {
				if len(p.Services) != 4 {
					t.Fatal("Nacos cluster must contain three nodes and MySQL")
				}
				if len(p.Services["mysql"].Ports) != 0 {
					t.Fatal("internal MySQL was exposed on the host")
				}
				if len(p.Services["mysql"].Configs) != 1 || !strings.Contains(tmpl.Definition.Configs["configs/nacos-mysql.sql"], "CREATE TABLE") {
					t.Fatal("missing first startup schema")
				}
				if p.Services["nacos2"].Environment["NACOS_APPLICATION_PORT"] == nil || *p.Services["nacos2"].Environment["NACOS_APPLICATION_PORT"] != "8848" {
					t.Fatal("container service port incorrectly followed host mapping")
				}
			}
		})
	}
}

func TestTemplateValuesRejectInvalidParameters(t *testing.T) {
	for _, tmpl := range testTemplates(t) {
		if tmpl.ID != "nacos-cluster" {
			continue
		}
		if _, err := Values(tmpl, nil, "https://registry.test"); err == nil {
			t.Fatal("missing secrets accepted")
		}
		values := models.StringMap{"NACOS_AUTH_TOKEN": "invalid", "NACOS_IDENTITY_VALUE": "secret", "MYSQL_PASSWORD": "secret", "MYSQL_ROOT_PASSWORD": "secret"}
		if _, err := Values(tmpl, values, "https://registry.test"); err == nil {
			t.Fatal("invalid token accepted")
		}
		values["NACOS_AUTH_TOKEN"] = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32)))
		values["NACOS_PORT"] = "65000"
		if _, err := Values(tmpl, values, "https://registry.test"); err == nil {
			t.Fatal("overflowing derived gRPC port accepted")
		}
	}
}

func TestMySQLInitializationPasswordsAreValidatedBeforeCreation(t *testing.T) {
	for _, tmpl := range testTemplates(t) {
		if tmpl.ID != "mysql" && tmpl.ID != "nacos-cluster" {
			continue
		}
		for _, key := range []string{"MYSQL_ROOT_PASSWORD", "MYSQL_PASSWORD"} {
			if tmpl.ID == "mysql" && key == "MYSQL_PASSWORD" {
				continue
			}
			t.Run(tmpl.ID+"/"+key, func(t *testing.T) {
				provided := models.StringMap{
					"MYSQL_ROOT_PASSWORD":  "safe-$password:42",
					"MYSQL_PASSWORD":       "safe-$password:42",
					"NACOS_AUTH_TOKEN":     base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32))),
					"NACOS_IDENTITY_VALUE": "identity",
				}
				for _, password := range []string{"bad'password", "bad\"password", "bad\\password", "bad\npassword", "bad\rpassword", "bad\tpassword"} {
					provided[key] = password
					if _, err := Values(tmpl, provided, "https://registry.test"); err == nil || !strings.Contains(err.Error(), "MySQL 8.0.44") || strings.Contains(err.Error(), password) {
						t.Fatalf("未提前拒绝不支持的初始密码，或错误包含凭据：%v", err)
					}
				}
				provided[key] = "safe-$password:42"
				if _, err := Values(tmpl, provided, "https://registry.test"); err != nil {
					t.Fatal("有效密码被拒绝", err)
				}
			})
		}
	}
}

func TestReusableDefinitionDoesNotExportInstanceCredentialsOrPaths(t *testing.T) {
	def := models.ApplicationDefinition{Compose: `services:
  app:
    image: old.registry.test/team/app:v2
    container_name: instance-name
    environment:
      PASSWORD: literal-instance-password
      TOKEN: ${API_TOKEN:-default-instance-token}
    volumes: ['/old/instance/data:/data']
`}
	result, err := ReusableDefinition(def)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"literal-instance-password", "default-instance-token", "/old/instance/data", "instance-name", "old.registry.test"} {
		if strings.Contains(result.Compose, value) {
			t.Fatalf("exported instance-specific value: %s", value)
		}
	}
	for _, p := range result.Parameters {
		if p.Type == "password" && p.Default != "" {
			t.Fatal("exported secret default")
		}
	}
	masked := RedactParameters(models.StringMap{"PASSWORD": "secret", "NACOS_IDENTITY_VALUE": "identity", "PORT": "8080"}, []models.ApplicationParameter{{Key: "NACOS_IDENTITY_VALUE", Type: "password"}})
	if masked["PASSWORD"] != "******" || masked["NACOS_IDENTITY_VALUE"] != "******" || masked["PORT"] != "8080" {
		t.Fatal("preview did not mask template secrets")
	}
}
