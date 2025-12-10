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
}

func Load() *Config {
	return &Config{
		ServerAddr:   getEnv("SERVER_ADDR", ":8080"),
		DatabasePath: getEnv("DATABASE_PATH", "./data/dockerun.db"),
		JWTSecret:    getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		Mode:         getEnv("GIN_MODE", "debug"),
		TemplateDir:  getEnv("TEMPLATE_DIR", "./templates"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
