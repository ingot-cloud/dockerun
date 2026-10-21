package handler

import (
	"context"
	"dockerun/internal/builtin"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
		"items":         paginatedImages,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
		"registry_name": registry.Name,
	})
}

// GetRegistryImageTags 获取镜像的所有标签
func GetRegistryImageTags(c *gin.Context) {
	registryID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	imageName := c.Param("name")

	// 通配符参数会包含前导的 /，需要去掉
	imageName = strings.TrimSuffix(strings.TrimPrefix(imageName, "/"), "/tags")
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
	var out []RegistryImage
	last := ""
	for {
		path := "/v2/_catalog?n=1000"
		if last != "" {
			path += "&last=" + url.QueryEscape(last)
		}
		b, _, err := release.RegistryRequest(context.Background(), registry, "GET", path)
		if err != nil {
			return nil, err
		}
		var page struct {
			Repositories []string `json:"repositories"`
		}
		if err = json.Unmarshal(b, &page); err != nil {
			return nil, err
		}
		for _, repo := range page.Repositories {
			if search != "" && !strings.Contains(strings.ToLower(repo), strings.ToLower(search)) {
				continue
			}
			tags, err := fetchImageTags(registry, repo)
			if err != nil {
				return nil, err
			}
			out = append(out, RegistryImage{Name: repo, Tags: tags})
		}
		if len(page.Repositories) < 1000 {
			break
		}
		next := page.Repositories[len(page.Repositories)-1]
		if next == last {
			return nil, fmt.Errorf("Registry 分页没有前进")
		}
		last = next
	}
	return out, nil
}
func fetchImageTags(registry models.Registry, name string) ([]string, error) {
	b, _, err := release.RegistryRequest(context.Background(), registry, "GET", "/v2/"+name+"/tags/list")
	if err != nil {
		return nil, err
	}
	var result struct {
		Tags []string `json:"tags"`
	}
	err = json.Unmarshal(b, &result)
	return result.Tags, err
}
func deleteRegistryImage(registry models.Registry, name, tag string) error {
	if registry.Kind == "builtin" {
		return fmt.Errorf("请使用内置仓库受保护删除接口")
	}
	_, headers, err := release.RegistryRequest(context.Background(), registry, "GET", "/v2/"+name+"/manifests/"+tag)
	if err != nil {
		return err
	}
	d := headers.Get("Docker-Content-Digest")
	if err = builtin.Protected(d); err != nil {
		return err
	}
	_, _, err = release.RegistryRequest(context.Background(), registry, "DELETE", "/v2/"+name+"/manifests/"+d)
	return err
}
