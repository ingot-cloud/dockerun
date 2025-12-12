package handler

import (
	"crypto/tls"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// RegistryImage 镜像信息
type RegistryImage struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// ListRegistryImages 获取 Registry 镜像列表（支持分页）
func ListRegistryImages(c *gin.Context) {
	registryID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	search := c.Query("search") // 搜索关键词
	
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取 Registry 信息
	var registry models.Registry
	if err := database.GetDB().First(&registry, registryID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Registry 不存在"})
		return
	}

	// 调用 Docker Registry API v2 获取镜像列表
	images, err := fetchRegistryImages(registry, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取镜像列表失败: %v", err)})
		return
	}

	// 计算分页
	total := len(images)
	start := (page - 1) * pageSize
	end := start + pageSize

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedImages := images[start:end]

	c.JSON(http.StatusOK, gin.H{
		"items":     paginatedImages,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetRegistryImageTags 获取镜像的所有标签
func GetRegistryImageTags(c *gin.Context) {
	registryID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	imageName := c.Param("name")
	
	// 通配符参数会包含前导的 /，需要去掉
	imageName = strings.TrimPrefix(imageName, "/")
	// 处理 URL 编码后的路径分隔符
	imageName = strings.ReplaceAll(imageName, "%2F", "/")

	// 获取 Registry 信息
	var registry models.Registry
	if err := database.GetDB().First(&registry, registryID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Registry 不存在"})
		return
	}

	// 获取镜像标签
	tags, err := fetchImageTags(registry, imageName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取镜像标签失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"name": imageName, "tags": tags})
}

// DeleteRegistryImage 删除 Registry 镜像
func DeleteRegistryImage(c *gin.Context) {
	registryID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	imageName := c.Param("name")
	tag := c.Query("tag")
	
	// 通配符参数会包含前导的 /，需要去掉
	imageName = strings.TrimPrefix(imageName, "/")
	// 处理 URL 编码后的路径分隔符
	imageName = strings.ReplaceAll(imageName, "%2F", "/")

	if tag == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请指定要删除的标签"})
		return
	}

	// 获取 Registry 信息
	var registry models.Registry
	if err := database.GetDB().First(&registry, registryID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Registry 不存在"})
		return
	}

	// 删除镜像
	if err := deleteRegistryImage(registry, imageName, tag); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("删除镜像失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// fetchRegistryImages 从 Registry 获取镜像列表
func fetchRegistryImages(registry models.Registry, search string) ([]RegistryImage, error) {
	// 创建 HTTP 客户端
	client := createHTTPClient(registry.Insecure)

	// 调用 /v2/_catalog API
	catalogURL := fmt.Sprintf("%s/v2/_catalog", strings.TrimSuffix(registry.URL, "/"))
	req, err := http.NewRequest("GET", catalogURL, nil)
	if err != nil {
		return nil, err
	}

	// 添加认证
	if registry.Username != "" {
		req.SetBasicAuth(registry.Username, registry.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var catalogResp struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&catalogResp); err != nil {
		return nil, err
	}

	// 过滤和构建结果
	var images []RegistryImage
	for _, repo := range catalogResp.Repositories {
		// 搜索过滤
		if search != "" && !strings.Contains(strings.ToLower(repo), strings.ToLower(search)) {
			continue
		}

		// 获取标签
		tags, err := fetchImageTags(registry, repo)
		if err != nil {
			// 如果获取标签失败，仍然返回镜像，但标签为空
			tags = []string{}
		}

		images = append(images, RegistryImage{
			Name: repo,
			Tags: tags,
		})
	}

	return images, nil
}

// fetchImageTags 获取镜像的标签列表
func fetchImageTags(registry models.Registry, imageName string) ([]string, error) {
	client := createHTTPClient(registry.Insecure)

	// 调用 /v2/<name>/tags/list API
	tagsURL := fmt.Sprintf("%s/v2/%s/tags/list", strings.TrimSuffix(registry.URL, "/"), imageName)
	req, err := http.NewRequest("GET", tagsURL, nil)
	if err != nil {
		return nil, err
	}

	// 添加认证
	if registry.Username != "" {
		req.SetBasicAuth(registry.Username, registry.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var tagsResp struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, err
	}

	return tagsResp.Tags, nil
}

// deleteRegistryImage 删除 Registry 镜像
func deleteRegistryImage(registry models.Registry, imageName, tag string) error {
	client := createHTTPClient(registry.Insecure)

	// 1. 获取 manifest digest
	manifestURL := fmt.Sprintf("%s/v2/%s/manifests/%s", strings.TrimSuffix(registry.URL, "/"), imageName, tag)
	req, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		return err
	}

	// 添加认证
	if registry.Username != "" {
		req.SetBasicAuth(registry.Username, registry.Password)
	}

	// 必须指定 Accept header
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("获取 manifest 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("获取 manifest 失败，状态码: %d", resp.StatusCode)
	}

	// 获取 digest
	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return fmt.Errorf("无法获取 manifest digest")
	}

	// 2. 删除 manifest
	deleteURL := fmt.Sprintf("%s/v2/%s/manifests/%s", strings.TrimSuffix(registry.URL, "/"), imageName, digest)
	delReq, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return err
	}

	// 添加认证
	if registry.Username != "" {
		delReq.SetBasicAuth(registry.Username, registry.Password)
	}

	delResp, err := client.Do(delReq)
	if err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusAccepted && delResp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("删除失败，状态码: %d", delResp.StatusCode)
	}

	return nil
}

// createHTTPClient 创建 HTTP 客户端
func createHTTPClient(insecure bool) *http.Client {
	if insecure {
		// 允许不安全的 HTTPS
		return &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}
	return &http.Client{}
}

