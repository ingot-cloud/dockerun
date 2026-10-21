package handler

import "github.com/gin-gonic/gin"

func ManagedTemplates(c *gin.Context) {
	c.JSON(200, []map[string]any{
		{"name": "单服务 HTTP 应用", "description": "固定镜像身份，具备可配置端口和健康检查", "defaults": map[string]string{"IMAGE": "registry.example.com/team/app:v1", "HOST_PORT": "8081", "CONTAINER_PORT": "80"}, "compose": "services:\n  app:\n    image: ${IMAGE:?请设置 IMAGE}\n    restart: unless-stopped\n    ports:\n      - '${HOST_PORT:-8081}:${CONTAINER_PORT:-80}'\n"},
		{"name": "应用与持久化依赖", "description": "独立服务与命名数据卷；更新不删除数据", "defaults": map[string]string{"APP_IMAGE": "registry.example.com/team/app:v1", "CACHE_IMAGE": "registry.example.com/library/redis:7"}, "compose": "services:\n  app:\n    image: ${APP_IMAGE:?请设置 APP_IMAGE}\n    restart: unless-stopped\n    depends_on:\n      cache:\n        condition: service_started\n  cache:\n    image: ${CACHE_IMAGE:?请设置 CACHE_IMAGE}\n    restart: unless-stopped\n    volumes:\n      - cache-data:/data\nvolumes:\n  cache-data: {}\n"},
	})
}
