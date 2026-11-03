// Package apptemplate 提供受管应用模板，模板内容不包含实例凭据或宿主数据路径。
package apptemplate

import (
	"dockerun/internal/models"
	"encoding/base64"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type Template struct {
	ID          string                       `json:"id"`
	Key         string                       `json:"key"`
	Order       int                          `json:"order"`
	Name        string                       `json:"name"`
	Version     string                       `json:"version"`
	Description string                       `json:"description"`
	Category    string                       `json:"category"`
	Source      string                       `json:"source"`
	Definition  models.ApplicationDefinition `json:"definition"`
	Scripts     models.DeploymentScripts     `json:"scripts"`
	Attachments models.StringMap             `json:"-"`
}

var variable = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func Values(t Template, provided models.StringMap, registryURL string) (models.StringMap, error) {
	values := models.StringMap{}
	rules := parameterRules(t.Definition)
	u, err := url.Parse(registryURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("仓库地址无效")
	}
	for _, p := range t.Definition.Parameters {
		if !variable.MatchString(p.Key) {
			return nil, fmt.Errorf("模板参数名称无效")
		}
		v, ok := provided[p.Key]
		if !ok && p.Type != "password" {
			v = p.Default
		}
		if p.Required && strings.TrimSpace(v) == "" {
			return nil, fmt.Errorf("请填写%s", p.Label)
		}
		if strings.ContainsRune(v, 0) {
			return nil, fmt.Errorf("%s不能包含空字符", p.Label)
		}
		if rules[p.Key] == "aes" {
			if n := len(v); n != 16 && n != 24 && n != 32 {
				return nil, fmt.Errorf("%s必须为 16、24 或 32 个 UTF-8 字节；当前镜像使用原始文本，不会先做 Base64 解码", p.Label)
			}
		}
		// 固定版本镜像直接把初始密码写入 SQL 和客户端配置，不支持这些字符。
		// 仅限制负责初始化数据库的服务，不影响连接已有数据库的密码。
		if rules[p.Key] == "mysql_password" {
			if strings.ContainsAny(v, "'\"\\") || strings.ContainsFunc(v, unicode.IsControl) {
				return nil, fmt.Errorf("%s不能包含引号、反斜杠或控制字符，MySQL 8.0.44 首次初始化不支持这些字符", p.Label)
			}
		}
		if p.Type == "port" {
			n, e := strconv.Atoi(v)
			if e != nil || n < 1 || n > 65535 {
				return nil, fmt.Errorf("%s必须为 1–65535 的端口", p.Label)
			}
		}
		if p.Type == "path" && v != "" && !filepath.IsAbs(v) {
			return nil, fmt.Errorf("%s必须是目标主机的绝对路径", p.Label)
		}
		if p.Type == "image" && v != "" {
			first, _, hasSlash := strings.Cut(v, "/")
			if !hasSlash || !strings.ContainsAny(first, ".:") && first != "localhost" {
				v = u.Host + "/" + v
			}
		}
		values[p.Key] = v
	}
	if value, ok := values["NACOS_AUTH_TOKEN"]; ok {
		raw, err := base64.StdEncoding.DecodeString(value)
		if err != nil || len(raw) < 32 {
			return nil, fmt.Errorf("Nacos 认证令牌必须为 Base64 编码且解码后至少 32 字节")
		}
	}
	if value, ok := values["NACOS_PORT"]; ok {
		n, _ := strconv.Atoi(value)
		if n > 64535 {
			return nil, fmt.Errorf("Nacos 服务端口不能超过 64535，需预留端口 +1000 用于 gRPC")
		}
		values["NACOS_GRPC_PORT"] = strconv.Itoa(n + 1000)
	}
	return values, nil
}
