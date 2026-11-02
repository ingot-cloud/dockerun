package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

type Config struct {
	ServerAddr      string
	DatabasePath    string
	JWTSecret       string
	Mode            string
	TemplateDir     string
	DataDir         string
	HostDataDir     string
	DataVolume      string
	PublicURL       string
	RegistrySecret  string
	RegistryEnabled bool
	TLSCert         string
	TLSKey          string
	ImportDir       string

	// 数据库配置
	DatabaseType string // sqlite 或 mysql
	MySQLDSN     string // MySQL 连接字符串

	// Docker Network 配置
	NetworkName    string
	NetworkDriver  string
	NetworkSubnet  string
	NetworkGateway string
}

func Load() *Config {
	dataDir, _ := filepath.Abs(getEnv("DATA_DIR", "./data"))
	return &Config{
		ServerAddr:     getEnv("SERVER_ADDR", ":8080"),
		DatabasePath:   getEnv("DATABASE_PATH", filepath.Join(dataDir, "dockerun.db")),
		JWTSecret:      secret("JWT_SECRET", filepath.Join(dataDir, "keys", "web.key")),
		RegistrySecret: secret("REGISTRY_SECRET", filepath.Join(dataDir, "keys", "registry.key")),
		DataDir:        dataDir, PublicURL: getEnv("PUBLIC_URL", "http://localhost:8080"), RegistryEnabled: getEnv("REGISTRY_ENABLED", "true") == "true",
		HostDataDir: os.Getenv("HOST_DATA_DIR"), DataVolume: os.Getenv("DATA_VOLUME"),
		TLSCert: os.Getenv("TLS_CERT"), TLSKey: os.Getenv("TLS_KEY"), ImportDir: getEnv("IMPORT_DIR", filepath.Join(dataDir, "imports")),
		Mode:        getEnv("GIN_MODE", "debug"),
		TemplateDir: getEnv("TEMPLATE_DIR", "./templates"),

		// 数据库配置
		DatabaseType: getEnv("DATABASE_TYPE", "sqlite"), // sqlite 或 mysql
		MySQLDSN:     getEnv("MYSQL_DSN", ""),           // MySQL 连接字符串，如：user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local

		// Network 配置
		NetworkName:    getEnv("DOCKER_NETWORK_NAME", "ingot-net"),
		NetworkDriver:  getEnv("DOCKER_NETWORK_DRIVER", "bridge"),
		NetworkSubnet:  getEnv("DOCKER_NETWORK_SUBNET", "172.88.0.0/16"),
		NetworkGateway: getEnv("DOCKER_NETWORK_GATEWAY", "172.88.0.1"),
	}
}

func secret(env, path string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	b, _ := os.ReadFile(path)
	return string(b)
}

// Initialize creates persistent, independent signing keys before accepting requests.
func Initialize() error {
	cfg := Load()
	if cfg.HostDataDir != "" && cfg.DataVolume != "" {
		return fmt.Errorf("HOST_DATA_DIR 与 DATA_VOLUME 不能同时设置")
	}
	if cfg.HostDataDir != "" && (!filepath.IsAbs(cfg.HostDataDir) || filepath.Clean(cfg.HostDataDir) == "/") {
		return fmt.Errorf("HOST_DATA_DIR 必须是非根目录的宿主机绝对路径")
	}
	u, err := url.Parse(cfg.PublicURL)
	if err != nil || u.Host == "" || u.Scheme != "http" && u.Scheme != "https" || u.User != nil || u.Path != "" && u.Path != "/" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("PUBLIC_URL 必须是完整 http(s) 入口地址，不含路径或凭据")
	}
	if cfg.DatabaseType != "sqlite" && cfg.DatabaseType != "mysql" {
		return fmt.Errorf("DATABASE_TYPE 仅支持 sqlite 或 mysql")
	}
	if cfg.DatabaseType == "mysql" && cfg.MySQLDSN == "" {
		return fmt.Errorf("MySQL 模式必须设置 MYSQL_DSN")
	}
	if (cfg.TLSCert == "") != (cfg.TLSKey == "") {
		return fmt.Errorf("TLS_CERT 与 TLS_KEY 必须同时设置")
	}
	for _, dir := range []string{"keys", "registry", "uploads", "releases", "jobs", "backups", "imports"} {
		if err := os.MkdirAll(filepath.Join(cfg.DataDir, dir), 0700); err != nil {
			return err
		}
	}
	for name, env := range map[string]string{"web.key": "JWT_SECRET", "registry.key": "REGISTRY_SECRET", "http.key": "REGISTRY_HTTP_SECRET", "credentials.key": "CREDENTIALS_SECRET"} {
		if v := os.Getenv(env); v != "" {
			if name == "credentials.key" {
				key, err := hex.DecodeString(v)
				if err != nil || len(key) != 32 {
					return fmt.Errorf("CREDENTIALS_SECRET 必须是 64 位十六进制密钥")
				}
			} else if len(v) < 32 {
				return fmt.Errorf("%s 至少需要 32 字节", env)
			}
			continue
		}
		path := filepath.Join(cfg.DataDir, "keys", name)
		if b, err := os.ReadFile(path); err == nil {
			if len(b) < 32 {
				return fmt.Errorf("signing key %s is too short", name)
			}
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, err = f.WriteString(hex.EncodeToString(b))
		ce := f.Close()
		if err != nil {
			return err
		}
		if ce != nil {
			return ce
		}
	}
	return nil
}

func RegistryHTTPSecret() string {
	return secret("REGISTRY_HTTP_SECRET", filepath.Join(Load().DataDir, "keys", "http.key"))
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func CredentialsKey() string {
	return secret("CREDENTIALS_SECRET", filepath.Join(Load().DataDir, "keys", "credentials.key"))
}
