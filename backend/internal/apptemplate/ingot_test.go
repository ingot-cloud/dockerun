package apptemplate

import (
	"context"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func ingotTestParameters(tmpl Template) models.StringMap {
	values := models.StringMap{}
	for _, p := range tmpl.Definition.Parameters {
		values[p.Key] = p.Default
		if p.Default == "" {
			values[p.Key] = strings.Repeat("a", 32)
		}
	}
	return values
}

func TestIngotAESKeysUseRawUTF8Bytes(t *testing.T) {
	for _, tmpl := range testTemplates(t) {
		if !strings.HasPrefix(tmpl.ID, "ingot-") {
			continue
		}
		for _, p := range tmpl.Definition.Parameters {
			if p.Key != "AUTH_JWK_MASTER_KEY" && p.Key != "CRYPTO_AES_KEY" && p.Key != "CRYPTO_AES_GCM_KEY" {
				continue
			}
			t.Run(tmpl.ID+"/"+p.Key, func(t *testing.T) {
				provided := ingotTestParameters(tmpl)
				for _, value := range []string{"short-key", strings.Repeat("a", 44), strings.Repeat("中", 16)} {
					provided[p.Key] = value
					if _, err := Values(tmpl, provided, "https://registry.test"); err == nil || strings.Contains(err.Error(), value) {
						t.Fatalf("接受了无效 AES 密钥长度，或错误泄漏凭据：%v", err)
					}
				}
				for _, value := range []string{strings.Repeat("a", 16), strings.Repeat("a", 24), strings.Repeat("a", 32), strings.Repeat("中", 10) + "ab"} {
					provided[p.Key] = value
					if _, err := Values(tmpl, provided, "https://registry.test"); err != nil {
						t.Fatal("有效 UTF-8 字节长度被拒绝", err)
					}
				}
			})
		}
	}
}

func TestIngotHealthChecksRequireApplicationReadiness(t *testing.T) {
	for _, tmpl := range testTemplates(t) {
		if !strings.HasPrefix(tmpl.ID, "ingot-") {
			continue
		}
		t.Run(tmpl.ID, func(t *testing.T) {
			dir := t.TempDir()
			if err := release.WriteConfigs(dir, tmpl.Definition.Configs); err != nil {
				t.Fatal(err)
			}
			values, err := Values(tmpl, ingotTestParameters(tmpl), "https://registry.test")
			if err != nil {
				t.Fatal(err)
			}
			project, err := release.Parse(context.Background(), tmpl.Definition.Compose, dir, "ingot-test", values)
			if err != nil {
				t.Fatal(err)
			}
			for _, svc := range project.Services {
				if svc.HealthCheck == nil || len(svc.HealthCheck.Test) != 2 || svc.HealthCheck.Test[0] != "CMD-SHELL" || svc.Environment["MANAGEMENT_PORT"] == nil {
					t.Fatal("缺少明确的管理端口或健康检查")
				}
				// 让 curl 返回 HTTP 200 的不同服务状态，验证命令的响应内容检查和端口绑定。
				stub := "#!/bin/sh\ncase \"$*\" in *\"http://127.0.0.1:$MANAGEMENT_PORT/actuator/health\"*) ;; *) exit 2 ;; esac\nprintf '%s' \"$TEST_HEALTH_RESPONSE\"\n"
				if err := os.WriteFile(filepath.Join(dir, "curl"), []byte(stub), 0700); err != nil {
					t.Fatal(err)
				}
				for _, test := range []struct {
					response string
					healthy  bool
				}{{`{"status":"UP"}`, true}, {`{"status":"DOWN"}`, false}, {`{"status":"DOWN","components":{"redis":{"status":"UP"}}}`, false}, {`{"status":"OUT_OF_SERVICE"}`, false}} {
					cmd := exec.Command("sh", "-c", svc.HealthCheck.Test[1])
					cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "MANAGEMENT_PORT="+*svc.Environment["MANAGEMENT_PORT"], "TEST_HEALTH_RESPONSE="+test.response)
					if output, err := cmd.CombinedOutput(); (err == nil) != test.healthy {
						t.Fatalf("服务状态 %s 的健康判定错误：%v，%s", test.response, err, output)
					}
				}
			}
		})
	}
}
