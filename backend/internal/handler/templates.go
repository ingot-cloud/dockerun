package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ScriptTemplate struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Icon        string                 `json:"icon"`
	MainScript  string                 `json:"main_script"`
	WorkDir     string                 `json:"work_dir"`
	EnvVars     map[string]string      `json:"env_vars"`
	Files       []TemplateFile         `json:"files"`
}

type TemplateFile struct {
	FileName    string `json:"file_name"`
	FilePath    string `json:"file_path"`
	FileType    string `json:"file_type"`
	Content     string `json:"content"`
	Description string `json:"description"`
}

var templates = []ScriptTemplate{
	{
		ID:          "nginx",
		Name:        "Nginx Web Server",
		Description: "Deploy Nginx web server with custom configuration",
		Category:    "Web Server",
		Icon:        "🌐",
		WorkDir:     "/opt/nginx-deploy",
		MainScript: `#!/usr/bin/env bash
set -e

echo "Deploying Nginx..."

# Create necessary directories
mkdir -p ${WORK_DIR}/html ${WORK_DIR}/conf

# Stop and remove existing container if exists
docker stop nginx-web 2>/dev/null || true
docker rm nginx-web 2>/dev/null || true

# Run Nginx container
CONTAINER_ID=$(docker run -d \
  --name nginx-web \
  -p ${NGINX_PORT:-80}:80 \
  -v ${WORK_DIR}/html:/usr/share/nginx/html:ro \
  -v ${WORK_DIR}/conf/nginx.conf:/etc/nginx/nginx.conf:ro \
  --restart unless-stopped \
  nginx:${NGINX_VERSION:-latest})

echo "Nginx deployed successfully!"
echo "Container ID: $CONTAINER_ID"`,
		EnvVars: map[string]string{
			"NGINX_PORT":    "80",
			"NGINX_VERSION": "latest",
		},
		Files: []TemplateFile{
			{
				FileName:    "index.html",
				FilePath:    "html/index.html",
				FileType:    "config",
				Description: "Default index page",
				Content: `<!DOCTYPE html>
<html>
<head>
    <title>Welcome to Nginx</title>
</head>
<body>
    <h1>Welcome to Nginx!</h1>
    <p>This is a default page deployed via Dockerun.</p>
</body>
</html>`,
			},
			{
				FileName:    "nginx.conf",
				FilePath:    "conf/nginx.conf",
				FileType:    "config",
				Description: "Nginx configuration",
				Content: `user  nginx;
worker_processes  auto;
error_log  /var/log/nginx/error.log notice;
pid        /var/run/nginx.pid;

events {
    worker_connections  1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;
    
    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';
    
    access_log  /var/log/nginx/access.log  main;
    
    sendfile        on;
    keepalive_timeout  65;
    
    server {
        listen       80;
        server_name  localhost;
        
        location / {
            root   /usr/share/nginx/html;
            index  index.html index.htm;
        }
        
        error_page   500 502 503 504  /50x.html;
        location = /50x.html {
            root   /usr/share/nginx/html;
        }
    }
}`,
			},
		},
	},
	{
		ID:          "mysql",
		Name:        "MySQL Database",
		Description: "Deploy MySQL database server with persistent storage",
		Category:    "Database",
		Icon:        "🗄️",
		WorkDir:     "/opt/mysql-deploy",
		MainScript: `#!/usr/bin/env bash
set -e

echo "Deploying MySQL..."

# Create data directory
mkdir -p ${WORK_DIR}/data

# Stop and remove existing container if exists
docker stop mysql-db 2>/dev/null || true
docker rm mysql-db 2>/dev/null || true

# Run MySQL container
CONTAINER_ID=$(docker run -d \
  --name mysql-db \
  -p ${MYSQL_PORT:-3306}:3306 \
  -e MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD} \
  -e MYSQL_DATABASE=${MYSQL_DATABASE:-mydb} \
  -e MYSQL_USER=${MYSQL_USER:-user} \
  -e MYSQL_PASSWORD=${MYSQL_PASSWORD} \
  -v ${WORK_DIR}/data:/var/lib/mysql \
  --restart unless-stopped \
  mysql:${MYSQL_VERSION:-8.0})

echo "MySQL deployed successfully!"
echo "Container ID: $CONTAINER_ID"
echo "Root Password: ${MYSQL_ROOT_PASSWORD}"
echo "Database: ${MYSQL_DATABASE:-mydb}"`,
		EnvVars: map[string]string{
			"MYSQL_PORT":          "3306",
			"MYSQL_VERSION":       "8.0",
			"MYSQL_ROOT_PASSWORD": "changeme",
			"MYSQL_DATABASE":      "mydb",
			"MYSQL_USER":          "user",
			"MYSQL_PASSWORD":      "password",
		},
		Files: []TemplateFile{},
	},
	{
		ID:          "redis",
		Name:        "Redis Cache",
		Description: "Deploy Redis cache server",
		Category:    "Cache",
		Icon:        "⚡",
		WorkDir:     "/opt/redis-deploy",
		MainScript: `#!/usr/bin/env bash
set -e

echo "Deploying Redis..."

# Create data directory
mkdir -p ${WORK_DIR}/data

# Stop and remove existing container if exists
docker stop redis-cache 2>/dev/null || true
docker rm redis-cache 2>/dev/null || true

# Run Redis container
CONTAINER_ID=$(docker run -d \
  --name redis-cache \
  -p ${REDIS_PORT:-6379}:6379 \
  -v ${WORK_DIR}/data:/data \
  --restart unless-stopped \
  redis:${REDIS_VERSION:-latest} \
  redis-server --appendonly yes --requirepass "${REDIS_PASSWORD}")

echo "Redis deployed successfully!"
echo "Container ID: $CONTAINER_ID"
echo "Password: ${REDIS_PASSWORD}"`,
		EnvVars: map[string]string{
			"REDIS_PORT":     "6379",
			"REDIS_VERSION":  "latest",
			"REDIS_PASSWORD": "changeme",
		},
		Files: []TemplateFile{},
	},
	{
		ID:          "docker-compose",
		Name:        "Docker Compose App",
		Description: "Deploy multi-container application using Docker Compose",
		Category:    "Multi-Container",
		Icon:        "🐳",
		WorkDir:     "/opt/compose-deploy",
		MainScript: `#!/usr/bin/env bash
set -e

echo "Deploying Docker Compose application..."

cd ${WORK_DIR}

# Stop existing services
docker-compose -f docker-compose.yml down 2>/dev/null || true

# Start services
docker-compose -f docker-compose.yml up -d

# Get container IDs
CONTAINER_ID=$(docker-compose -f docker-compose.yml ps -q | head -n 1)

echo "Docker Compose application deployed successfully!"
echo "Container ID: $CONTAINER_ID"
docker-compose -f docker-compose.yml ps`,
		EnvVars: map[string]string{
			"APP_NAME": "myapp",
			"APP_PORT": "8080",
		},
		Files: []TemplateFile{
			{
				FileName:    "docker-compose.yml",
				FilePath:    "docker-compose.yml",
				FileType:    "compose",
				Description: "Docker Compose configuration",
				Content: `version: '3.8'

services:
  web:
    image: nginx:latest
    ports:
      - "\${APP_PORT:-8080}:80"
    volumes:
      - ./html:/usr/share/nginx/html:ro
    restart: unless-stopped
    
  app:
    image: node:lts-alpine
    working_dir: /app
    volumes:
      - ./app:/app
    command: sh -c "npm install && npm start"
    restart: unless-stopped`,
			},
		},
	},
}

// GetTemplates 获取所有模版
func GetTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, templates)
}

// GetTemplate 获取单个模版
func GetTemplate(c *gin.Context) {
	id := c.Param("id")
	
	for _, tmpl := range templates {
		if tmpl.ID == id {
			c.JSON(http.StatusOK, tmpl)
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
}

