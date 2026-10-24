package apptemplate

import (
	"context"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuiltinTemplatesParseWithPersistentScopedVolumes(t *testing.T) {
	for _, tmpl := range Builtins() {
		t.Run(tmpl.ID, func(t *testing.T) {
			provided := models.StringMap{}
			for _, p := range tmpl.Definition.Parameters {
				if p.Type == "password" {
					provided[p.Key] = "test-'$password"
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
				if p.Services["nacos2"].Environment["NACOS_SERVER_PORT"] == nil || *p.Services["nacos2"].Environment["NACOS_SERVER_PORT"] != "8848" {
					t.Fatal("container service port incorrectly followed host mapping")
				}
			}
		})
	}
}

func TestTemplateValuesRejectInvalidParameters(t *testing.T) {
	for _, tmpl := range Builtins() {
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
