package apptemplate

import (
	"archive/zip"
	"bytes"
	"context"
	"dockerun/internal/models"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func packageFixture() Template {
	return Template{ID: "example", Key: "example", Name: "示例服务", Version: "1.0.0", Category: "business", Definition: models.ApplicationDefinition{Compose: "services:\n  app:\n    image: ${IMAGE}\n", Parameters: []models.ApplicationParameter{{Key: "IMAGE", Label: "应用镜像", Type: "image", Default: "nginx:1.28", Required: true}}, Topology: "replicated", Configs: models.StringMap{"configs/empty.conf": ""}}, Attachments: models.StringMap{"README.md": "使用说明\n", "LICENSE": "示例许可\n"}}
}
func zipFixture(t *testing.T, files map[string]string, extra ...*zip.FileHeader) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	for _, header := range extra {
		f, err := w.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		f.Write([]byte("x"))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestShippedPackageRoundTripPreservesEveryDefinition(t *testing.T) {
	items := testTemplates(t)
	data, err := WritePackage(context.Background(), items)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ReadPackage(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 13 {
		t.Fatal("模板丢失", len(decoded))
	}
	originals := map[string]Template{}
	for _, item := range items {
		originals[item.ID] = item
	}
	for _, item := range decoded {
		before, _ := Fingerprint(originals[item.Key])
		after, _ := Fingerprint(item)
		if before != after {
			t.Fatal("模板往返后内容变化", item.Key)
		}
	}
	var nacos Template
	for _, item := range decoded {
		if item.Key == "nacos-cluster" {
			nacos = item
		}
	}
	if !strings.Contains(nacos.Attachments["NACOS-LICENSE"], "Apache") {
		t.Fatal("Nacos 许可丢失")
	}
}
func TestPackageSupportsFlatWrappedAndMultipleVersions(t *testing.T) {
	fixture := packageFixture()
	files, err := PackageFiles(fixture)
	if err != nil {
		t.Fatal(err)
	}
	for _, prefix := range []string{"", "example/", "templates/example/", "wrapper/templates/example/"} {
		t.Run(prefix, func(t *testing.T) {
			wrapped := map[string]string{}
			for name, content := range files {
				wrapped[prefix+name] = content
			}
			got, err := ReadPackage(context.Background(), zipFixture(t, wrapped))
			if err != nil || len(got) != 1 {
				t.Fatal(err)
			}
			if got[0].Definition.Configs["configs/empty.conf"] != "" || got[0].Attachments["LICENSE"] != "示例许可\n" {
				t.Fatal("空文件或许可丢失")
			}
		})
	}
	other := fixture
	other.Version = "2.0.0"
	data, err := WritePackage(context.Background(), []Template{fixture, other})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadPackage(context.Background(), data)
	if err != nil || len(got) != 2 {
		t.Fatal("多版本包失败", err)
	}
}
func TestPackagePreservesSharedAndLocalDocumentation(t *testing.T) {
	files, err := PackageFiles(packageFixture())
	if err != nil {
		t.Fatal(err)
	}
	wrapped := map[string]string{"LICENSE": "公共许可\n", "README.md": "公共使用说明\n"}
	for name, content := range files {
		wrapped["example/"+name] = content
	}
	got, err := ReadPackage(context.Background(), zipFixture(t, wrapped))
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Attachments["LICENSE"] != "示例许可\n" || got[0].Attachments["docs/package/LICENSE"] != "公共许可\n" || got[0].Attachments["docs/package/README.md"] != "公共使用说明\n" {
		t.Fatal("公共说明覆盖了模板说明", got[0].Attachments)
	}
	data, err := WritePackage(context.Background(), got)
	if err != nil {
		t.Fatal(err)
	}
	again, err := ReadPackage(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := Fingerprint(got[0])
	after, _ := Fingerprint(again[0])
	if before != after {
		t.Fatal("说明文件往返后变化")
	}
}

func TestPackageChecksScriptsWithoutExecutingThem(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "executed")
	fixture := packageFixture()
	fixture.Scripts.Before = "touch '" + marker + "'\n"
	data, err := WritePackage(context.Background(), []Template{fixture})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ReadPackage(context.Background(), data); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("校验执行了脚本")
	}
	fixture.Scripts.Before = "if then broken"
	if _, err = WritePackage(context.Background(), []Template{fixture}); err == nil {
		t.Fatal("错误脚本未拒绝")
	}
}
func TestPackageRejectsUnsafeAndIncompleteArchives(t *testing.T) {
	files, _ := PackageFiles(packageFixture())
	cases := []struct {
		name   string
		change func(map[string]string)
		extra  *zip.FileHeader
	}{
		{"父目录", func(m map[string]string) { m["../escape"] = "bad" }, nil},
		{"绝对路径", func(m map[string]string) { m["/tmp/escape"] = "bad" }, nil},
		{"反斜杠", func(m map[string]string) { m[`configs\bad`] = "bad" }, nil},
		{"盘符", func(m map[string]string) { m["C:/escape"] = "bad" }, nil},
		{"重复文件", nil, &zip.FileHeader{Name: "compose.yaml"}},
		{"符号链接", nil, func() *zip.FileHeader { h := &zip.FileHeader{Name: "link"}; h.SetMode(os.ModeSymlink | 0777); return h }()},
		{"未声明文件", func(m map[string]string) { m["unused.env"] = "bad" }, nil},
		{"文件目录冲突", func(m map[string]string) { m["compose.yaml/child"] = "bad" }, nil},
		{"缺少引用", func(m map[string]string) { delete(m, "configs/empty.conf") }, nil},
		{"未知字段", func(m map[string]string) {
			var d map[string]any
			json.Unmarshal([]byte(m[manifestName]), &d)
			d["typo"] = true
			b, _ := json.Marshal(d)
			m[manifestName] = string(b)
		}, nil},
		{"非法文本", func(m map[string]string) { m["LICENSE"] = "bad\x00text" }, nil},
		{"无效编码", func(m map[string]string) { m["LICENSE"] = string([]byte{0xff}) }, nil},
		{"构建指令", func(m map[string]string) {
			m["compose.yaml"] = "services:\n  app:\n    image: ${IMAGE}\n    build: /private\n"
		}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			copy := map[string]string{}
			for k, v := range files {
				copy[k] = v
			}
			if tc.change != nil {
				tc.change(copy)
			}
			var extra []*zip.FileHeader
			if tc.extra != nil {
				extra = append(extra, tc.extra)
			}
			if _, err := ReadPackage(context.Background(), zipFixture(t, copy, extra...)); err == nil {
				t.Fatal("无效 ZIP 未被拒绝")
			}
		})
	}
	oversized := zipFixture(t, map[string]string{"LICENSE": strings.Repeat("a", 8<<20+1)})
	if _, err := ReadPackage(context.Background(), oversized); err == nil {
		t.Fatal("过大文件未拒绝")
	}
	if _, err := ReadPackage(context.Background(), []byte("not-a-zip")); err == nil {
		t.Fatal("非 ZIP 未拒绝")
	}
}
func TestImportedAndRenamedTemplatesKeepParameterValidation(t *testing.T) {
	for _, item := range testTemplates(t) {
		if item.ID != "mysql" && item.ID != "ingot-auth" {
			continue
		}
		item.ID = "custom-123"
		item.Key = "renamed-template"
		values := models.StringMap{}
		for _, p := range item.Definition.Parameters {
			values[p.Key] = p.Default
			if values[p.Key] == "" {
				values[p.Key] = strings.Repeat("a", 32)
			}
		}
		if item.Definition.Parameters[0].Key == "MYSQL_IMAGE" {
			values["MYSQL_ROOT_PASSWORD"] = "bad'password"
		} else {
			values["AUTH_JWK_MASTER_KEY"] = "short-key"
		}
		if _, err := Values(item, values, "https://registry.test"); err == nil {
			t.Fatal("导入或另存后参数校验丢失", item.Key)
		}
	}
}
func TestExecutionAndAttachmentsSurviveExport(t *testing.T) {
	item := packageFixture()
	item.Definition.Platform = "linux/amd64"
	item.Definition.Execution = models.DeploymentExecution{Mode: "command", Command: "printf '部署\\n'", CheckCommand: "true", WorkDir: "${WORK_DIR}", TimeoutSeconds: 300}
	data, err := WritePackage(context.Background(), []Template{item})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadPackage(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Definition.Execution != item.Definition.Execution || got[0].Definition.Platform != item.Definition.Platform {
		t.Fatal("执行方式或架构丢失")
	}
}
