// Package apptemplate 提供受管应用模板，模板内容不包含实例凭据或宿主数据路径。
package apptemplate

import (
	"dockerun/internal/models"
	_ "embed"
	"encoding/base64"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

//go:embed catalog/nacos-mysql.sql
var nacosSQL string

type Template struct {
	ID          string                       `json:"id"`
	Name        string                       `json:"name"`
	Version     string                       `json:"version"`
	Description string                       `json:"description"`
	Category    string                       `json:"category"`
	Source      string                       `json:"source"`
	Definition  models.ApplicationDefinition `json:"definition"`
	Scripts     models.DeploymentScripts     `json:"scripts"`
}

func parameter(key, label, kind, value, help string) models.ApplicationParameter {
	return models.ApplicationParameter{Key: key, Label: label, Type: kind, Default: value, Required: true, Help: help}
}
func image(key, label, value string) models.ApplicationParameter {
	return parameter(key, label, "image", value, "默认使用所选仓库的同名镜像，请先准备该镜像。")
}
func port(key, label, value string) models.ApplicationParameter {
	return parameter(key, label, "port", value, "填写目标主机未占用的端口。")
}
func password(key, label string) models.ApplicationParameter {
	return parameter(key, label, "password", "", "仅保存到当前环境，不写入公共模板。")
}

func Builtins() []Template {
	web := Template{ID: "web-app", Name: "单服务应用", Version: "1.0.0", Category: "business", Source: "builtin", Description: "从镜像部署自研服务，按环境选择手动或自动更新。", Definition: models.ApplicationDefinition{
		Topology: "replicated", Parameters: []models.ApplicationParameter{image("IMAGE", "应用镜像", "team/app:v1"), port("HOST_PORT", "访问端口", "8081"), port("CONTAINER_PORT", "容器端口", "80")},
		Compose:      "services:\n  app:\n    image: ${IMAGE:?请填写应用镜像}\n    restart: unless-stopped\n    ports:\n      - '${HOST_PORT}:${CONTAINER_PORT}'\n",
		Instructions: "首次手动发布并确认运行后，可在环境配置中开启镜像推送自动发布。生产环境可继续使用手动发布。", Endpoints: []models.ApplicationEndpoint{{Name: "服务入口", Port: "HOST_PORT", Protocol: "http"}},
	}}
	mysql := Template{ID: "mysql", Name: "MySQL", Version: "1.0.0", Category: "middleware", Source: "builtin", Description: "独立数据库与持久化数据卷，固定版本，默认手动升级。", Definition: models.ApplicationDefinition{
		Stateful: true, Topology: "single_host", Parameters: []models.ApplicationParameter{image("MYSQL_IMAGE", "MySQL 镜像", "mysql:8.0.44"), port("MYSQL_PORT", "数据库端口", "3306"), password("MYSQL_ROOT_PASSWORD", "管理员密码")},
		Compose: mysqlCompose("${MYSQL_PORT}:3306", false), Storage: "mysql-data → /var/lib/mysql；数据卷与应用环境绑定，重新部署和升级会复用原数据。",
		Initialization: "MySQL 仅在数据卷为空时初始化。更改环境中的初始密码不会修改已有数据库密码。",
		Instructions:   "默认手动发布。升级前备份数据库并确认数据格式兼容；切回旧镜像不会恢复业务数据。", Endpoints: []models.ApplicationEndpoint{{Name: "数据库连接", Port: "MYSQL_PORT", Protocol: "mysql"}},
	}}
	redis := Template{ID: "redis", Name: "Redis", Version: "1.0.0", Category: "middleware", Source: "builtin", Description: "启用身份认证与 AOF 持久化，使用独立数据卷。", Definition: models.ApplicationDefinition{
		Stateful: true, Topology: "single_host", Parameters: []models.ApplicationParameter{image("REDIS_IMAGE", "Redis 镜像", "redis:8.2.0"), port("REDIS_PORT", "访问端口", "6379"), password("REDIS_PASSWORD", "访问密码")},
		Compose: `services:
  redis:
    image: ${REDIS_IMAGE:?请填写 Redis 镜像}
    restart: unless-stopped
    ports: ['${REDIS_PORT}:6379']
    environment:
      REDIS_PASSWORD: ${REDIS_PASSWORD:?请填写访问密码}
    command: [sh, -c, 'exec redis-server /usr/local/etc/redis/redis.conf --requirepass "$$REDIS_PASSWORD"']
    configs:
      - source: redis-config
        target: /usr/local/etc/redis/redis.conf
    volumes: ['redis-data:/data']
    healthcheck:
      test: [CMD-SHELL, 'REDISCLI_AUTH="$$REDIS_PASSWORD" redis-cli ping | grep -q PONG']
      interval: 10s
      timeout: 5s
      retries: 12
configs:
  redis-config:
    file: ./configs/redis.conf
volumes:
  redis-data: {}
`, Configs: models.StringMap{"configs/redis.conf": "bind 0.0.0.0\nprotected-mode yes\nappendonly yes\ndir /data\n"}, Storage: "redis-data → /data；AOF 数据随独立数据卷保留。", Instructions: "默认手动发布。升级前备份数据；应用备份不包含业务数据卷。", Endpoints: []models.ApplicationEndpoint{{Name: "缓存连接", Port: "REDIS_PORT", Protocol: "redis"}},
	}}
	return []Template{web, mysql, redis, nacos(false), nacos(true)}
}

func mysqlCompose(published string, cluster bool) string {
	content := `services:
  mysql:
    image: ${MYSQL_IMAGE:?请填写 MySQL 镜像}
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:?请填写管理员密码}
    volumes: ['mysql-data:/var/lib/mysql']
    healthcheck:
      test: [CMD-SHELL, 'MYSQL_PWD="$$MYSQL_ROOT_PASSWORD" mysqladmin ping -h 127.0.0.1 --silent']
      interval: 5s
      timeout: 5s
      retries: 30
      start_period: 30s
volumes:
  mysql-data: {}
`
	if published != "" {
		content = strings.Replace(content, "    restart: unless-stopped", "    restart: unless-stopped\n    ports: ['"+published+"']", 1)
	}
	if cluster {
		content = strings.Replace(content, "      MYSQL_ROOT_PASSWORD:", "      MYSQL_DATABASE: nacos\n      MYSQL_USER: nacos\n      MYSQL_PASSWORD: ${MYSQL_PASSWORD:?请填写 Nacos 数据库密码}\n      MYSQL_ROOT_PASSWORD:", 1)
		content = strings.Replace(content, "    healthcheck:", "    configs:\n      - source: nacos-schema\n        target: /docker-entrypoint-initdb.d/01-nacos.sql\n    healthcheck:", 1)
	}
	return content
}

func nacos(cluster bool) Template {
	t := Template{ID: "nacos-standalone", Name: "Nacos 单机", Version: "1.0.0", Category: "middleware", Source: "builtin", Description: "内嵌存储的单节点 Nacos，配置与日志独立持久化。", Definition: models.ApplicationDefinition{Stateful: true, Topology: "single_host",
		Parameters:   []models.ApplicationParameter{image("NACOS_IMAGE", "Nacos 镜像", "nacos/nacos-server:v3.1.1"), port("NACOS_PORT", "服务端口", "8848"), port("NACOS_CONSOLE_PORT", "控制台端口", "8181"), password("NACOS_AUTH_TOKEN", "认证令牌（Base64）"), parameter("NACOS_IDENTITY_KEY", "节点身份名称", "string", "dockerun", "同一集群的节点必须一致。"), password("NACOS_IDENTITY_VALUE", "节点身份密钥")},
		Instructions: "默认手动发布。服务端口需与对应 gRPC 客户端端口一起开放。首次访问控制台时，按 Nacos 引导设置管理员密码。", Storage: "每个节点分别使用独立的数据卷和日志卷。", Endpoints: []models.ApplicationEndpoint{{Name: "控制台", Port: "NACOS_CONSOLE_PORT", Protocol: "http"}, {Name: "服务地址", Port: "NACOS_PORT", Protocol: "http", Path: "/nacos"}},
	}}
	t.Definition.Parameters[3].Help = "填写 Base64 编码、解码后至少 32 字节的随机密钥，各节点使用同一值。"
	services := map[string]any{}
	volumes := map[string]any{}
	count := 1
	if cluster {
		t.ID = "nacos-cluster"
		t.Name = "Nacos 单机三节点集群"
		t.Description = "同一主机运行三个 Nacos 节点，附带独立 MySQL 和首次建表配置。"
		count = 3
		t.Definition.Parameters = append(t.Definition.Parameters, image("MYSQL_IMAGE", "MySQL 镜像", "mysql:8.0.44"), password("MYSQL_ROOT_PASSWORD", "MySQL 管理员密码"), password("MYSQL_PASSWORD", "Nacos 数据库密码"))
		t.Definition.Instructions = "三个 Nacos 节点和独立 MySQL 部署在同一目标主机，不提供跨主机高可用。固定主机组会重复部署整套服务。MySQL 仅供此应用访问。健康检查验证服务端口，集群成员状态请在控制台确认；首次登录时按 Nacos 引导设置管理员密码。升级前备份数据库。"
		t.Definition.Initialization = "随模板提供 Nacos 3.1.1 官方建表文件，由 MySQL 在空数据卷首次启动时执行。更改 Nacos 大版本前须单独确认数据库迁移要求。"
		t.Definition.Configs = models.StringMap{"configs/nacos-mysql.sql": nacosSQL}
	}
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("nacos%d", i)
		env := map[string]any{"MODE": "standalone", "NACOS_SERVER_PORT": "8848", "NACOS_AUTH_ENABLE": "true", "NACOS_AUTH_TOKEN": "${NACOS_AUTH_TOKEN:?请填写认证令牌}", "NACOS_AUTH_IDENTITY_KEY": "${NACOS_IDENTITY_KEY}", "NACOS_AUTH_IDENTITY_VALUE": "${NACOS_IDENTITY_VALUE:?请填写节点身份密钥}"}
		if cluster {
			for k, v := range map[string]any{"MODE": "cluster", "PREFER_HOST_MODE": "hostname", "NACOS_SERVERS": "nacos1:8848 nacos2:8848 nacos3:8848", "SPRING_DATASOURCE_PLATFORM": "mysql", "MYSQL_SERVICE_HOST": "mysql", "MYSQL_SERVICE_PORT": "3306", "MYSQL_SERVICE_DB_NAME": "nacos", "MYSQL_SERVICE_USER": "nacos", "MYSQL_SERVICE_PASSWORD": "${MYSQL_PASSWORD:?请填写数据库密码}", "MYSQL_SERVICE_DB_PARAM": "characterEncoding=utf8&connectTimeout=1000&socketTimeout=3000&autoReconnect=true&useSSL=false&allowPublicKeyRetrieval=true"} {
				env[k] = v
			}
		}
		service := map[string]any{"image": "${NACOS_IMAGE:?请填写 Nacos 镜像}", "hostname": name, "restart": "unless-stopped", "environment": env, "volumes": []string{name + "-data:/home/nacos/data", name + "-logs:/home/nacos/logs"}, "healthcheck": map[string]any{"test": []string{"CMD-SHELL", "bash -c 'echo > /dev/tcp/127.0.0.1/8848'"}, "interval": "10s", "timeout": "5s", "retries": 18, "start_period": "30s"}}
		if i == 1 {
			service["ports"] = []string{"${NACOS_PORT}:8848", "${NACOS_GRPC_PORT}:9848", "${NACOS_CONSOLE_PORT}:8080"}
		}
		if cluster {
			service["depends_on"] = map[string]any{"mysql": map[string]any{"condition": "service_healthy"}}
		}
		services[name] = service
		volumes[name+"-data"] = map[string]any{}
		volumes[name+"-logs"] = map[string]any{}
	}
	doc := map[string]any{"services": services, "volumes": volumes}
	if cluster {
		var mysql map[string]any
		mustYAML(mysqlCompose("", true), &mysql)
		services["mysql"] = mysql["services"].(map[string]any)["mysql"]
		volumes["mysql-data"] = map[string]any{}
		doc["configs"] = map[string]any{"nacos-schema": map[string]any{"file": "./configs/nacos-mysql.sql"}}
	}
	t.Definition.Compose = encodeYAML(doc)
	return t
}

var variable = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func Values(t Template, provided models.StringMap, registryURL string) (models.StringMap, error) {
	values := models.StringMap{}
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
