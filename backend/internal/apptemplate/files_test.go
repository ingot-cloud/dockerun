package apptemplate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifestFixture(t *testing.T, root, directory, id string) string {
	t.Helper()
	dir := filepath.Join(root, directory)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	fixture := `{"schema_version":1,"id":"` + id + `","name":"测试服务","version":"1.0.0","category":"business","definition":{"compose_file":"compose.yaml","topology":"replicated","parameters":[]}}`
	for name, content := range map[string]string{"template.json": fixture, "compose.yaml": "services:\n  app:\n    image: nginx:1.27\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func editManifestFixture(t *testing.T, dir string, edit func(map[string]any)) {
	t.Helper()
	name := filepath.Join(dir, manifestName)
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	edit(m)
	b, err = json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, b, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDirReadsFilesAndReloadsIndependentSnapshots(t *testing.T) {
	root := t.TempDir()
	dir := writeManifestFixture(t, root, "z", "file-service")
	writeManifestFixture(t, root, "a", "another-service")
	// 历史目录不会被误识别为应用模板。
	legacy := filepath.Join(root, "legacy")
	os.Mkdir(legacy, 0700)
	os.WriteFile(filepath.Join(legacy, "meta.json"), []byte(`{"id":"legacy"}`), 0600)
	os.WriteFile(filepath.Join(dir, "empty.conf"), nil, 0600)
	os.WriteFile(filepath.Join(dir, "before.sh"), []byte("printf '部署前检查\\n'\n"), 0600)
	editManifestFixture(t, dir, func(m map[string]any) {
		m["order"] = -1
		m["definition"].(map[string]any)["config_files"] = map[string]string{"configs/empty.conf": "empty.conf"}
		m["script_files"] = map[string]string{"before": "before.sh"}
	})
	items, err := LoadDir(root)
	if err != nil || len(items) != 2 {
		t.Fatal(items, err)
	}
	if items[0].ID != "file-service" || items[0].Source != "builtin" || items[0].Scripts.Before != "printf '部署前检查\\n'\n" {
		t.Fatal("文件内容、来源或排序丢失", items[0])
	}
	if content, ok := items[0].Definition.Configs["configs/empty.conf"]; !ok || content != "" {
		t.Fatal("空配置文件未保留")
	}
	os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services:\n  app:\n    image: nginx:1.28\n"), 0600)
	updated, err := LoadDir(root)
	if err != nil || !strings.Contains(updated[0].Definition.Compose, "nginx:1.28") || strings.Contains(items[0].Definition.Compose, "nginx:1.28") {
		t.Fatal("刷新未读取新文件，或修改了旧快照", err)
	}
	os.RemoveAll(dir)
	remaining, err := LoadDir(root)
	if err != nil || len(remaining) != 1 {
		t.Fatal("删除目录后仍返回旧模板", err)
	}
}

func TestLoadDirRejectsInvalidDefinitions(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"版本", func(m map[string]any) { m["schema_version"] = 2 }},
		{"保留标识", func(m map[string]any) { m["id"] = "custom-1" }},
		{"未知字段", func(m map[string]any) { m["typo"] = true }},
		{"缺少版本", func(m map[string]any) { delete(m, "version") }},
		{"未知分类", func(m map[string]any) { m["category"] = "Database" }},
		{"未知拓扑", func(m map[string]any) { m["definition"].(map[string]any)["topology"] = "cluster" }},
		{"越界", func(m map[string]any) { m["definition"].(map[string]any)["compose_file"] = "../compose.yaml" }},
		{"绝对路径", func(m map[string]any) { m["definition"].(map[string]any)["compose_file"] = "/tmp/compose.yaml" }},
		{"文件缺失", func(m map[string]any) { m["definition"].(map[string]any)["compose_file"] = "missing.yaml" }},
		{"配置目标越界", func(m map[string]any) {
			m["definition"].(map[string]any)["config_files"] = map[string]string{"configs/../../outside": "compose.yaml"}
		}},
		{"配置目录冲突", func(m map[string]any) {
			m["definition"].(map[string]any)["config_files"] = map[string]string{"configs/a": "compose.yaml", "configs/a/b": "compose.yaml"}
		}},
		{"未知脚本阶段", func(m map[string]any) { m["script_files"] = map[string]string{"setup": "compose.yaml"} }},
		{"重复参数", func(m map[string]any) {
			m["definition"].(map[string]any)["parameters"] = []any{map[string]any{"key": "PORT", "label": "端口", "type": "port"}, map[string]any{"key": "PORT", "label": "端口", "type": "port"}}
		}},
		{"密码默认值", func(m map[string]any) {
			m["definition"].(map[string]any)["parameters"] = []any{map[string]any{"key": "PASSWORD", "label": "密码", "type": "password", "default": "unsafe"}}
		}},
		{"敏感参数类型", func(m map[string]any) {
			m["definition"].(map[string]any)["parameters"] = []any{map[string]any{"key": "PASSWORD", "label": "密码", "type": "string"}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			dir := writeManifestFixture(t, root, "invalid", "invalid")
			editManifestFixture(t, dir, test.edit)
			if _, err := LoadDir(root); err == nil || !strings.Contains(err.Error(), "invalid/template.json") {
				t.Fatal("未报告具体模板的错误", err)
			}
		})
	}
}

func TestLoadDirRejectsDuplicatesMissingParametersAndEscapingSymlinks(t *testing.T) {
	root := t.TempDir()
	dir := writeManifestFixture(t, root, "first", "duplicate")
	other := writeManifestFixture(t, root, "second", "duplicate")
	if _, err := LoadDir(root); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatal("重复标识未被拒绝", err)
	}
	os.RemoveAll(other)
	os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services:\n  app:\n    image: ${MISSING_IMAGE}\n"), 0600)
	if _, err := LoadDir(root); err == nil || !strings.Contains(err.Error(), "MISSING_IMAGE") {
		t.Fatal("未声明参数未被拒绝", err)
	}
	outside := filepath.Join(t.TempDir(), "outside.yaml")
	os.WriteFile(outside, []byte("services: {}"), 0600)
	os.Remove(filepath.Join(dir, "compose.yaml"))
	if err := os.Symlink(outside, filepath.Join(dir, "compose.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDir(root); err == nil {
		t.Fatal("读取了模板目录外的符号链接")
	}
	if _, err := LoadDir(filepath.Join(root, "missing")); err == nil {
		t.Fatal("模板目录缺失未被报告")
	}
}

func TestShippedCatalogContainsEveryMigratedTemplate(t *testing.T) {
	items := testTemplates(t)
	want := strings.Fields("web-app mysql redis nacos-standalone nacos-cluster kafka-standalone kafka-cluster registry portainer nginx-proxy ingot-auth ingot-gateway ingot-pms")
	if len(items) != len(want) {
		t.Fatalf("模板数量 %d，预期 %d", len(items), len(want))
	}
	ids := map[string]bool{}
	for _, item := range items {
		ids[item.ID] = true
	}
	for _, id := range want {
		if !ids[id] {
			t.Fatal("模板未迁移", id)
		}
	}
}
