package config

import (
	"os"
)

type Config struct {
	ServerAddr   string
	DatabasePath string
	JWTSecret    string
	Mode         string
	TemplateDir  string

	// Docker Network 配置
	NetworkName    string
	NetworkDriver  string
	NetworkSubnet  string
	NetworkGateway string
}

func Load() *Config {
	return &Config{
		ServerAddr:   getEnv("SERVER_ADDR", ":8080"),
		DatabasePath: getEnv("DATABASE_PATH", "./data/dockerun.db"),
		JWTSecret:    getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		Mode:         getEnv("GIN_MODE", "debug"),
		TemplateDir:  getEnv("TEMPLATE_DIR", "./templates"),

		// Network 配置
		NetworkName:    getEnv("DOCKER_NETWORK_NAME", "ingot-net"),
		NetworkDriver:  getEnv("DOCKER_NETWORK_DRIVER", "bridge"),
		NetworkSubnet:  getEnv("DOCKER_NETWORK_SUBNET", "172.88.0.0/16"),
		NetworkGateway: getEnv("DOCKER_NETWORK_GATEWAY", "172.88.0.1"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
