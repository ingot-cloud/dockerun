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
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RegistryImage 镜像信息
type RegistryImage struct {
	Name      string   `json:"name"`
	Tags      []string `json:"tags"`
	TagsError string   `json:"tags_error,omitempty"`
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
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()
	names, err := fetchRegistryRepositories(ctx, registry, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取镜像列表失败: %v", err)})
		return
	}

	// 计算分页
	total := len(names)
	start := total
	if page <= (total+pageSize-1)/pageSize {
		start = (page - 1) * pageSize
	}
	end := start + pageSize

	if end > total {
		end = total
	}

	paginatedImages := fetchRegistryPageTags(ctx, registry, names[start:end])

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
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	tags, err := fetchImageTags(ctx, registry, imageName)
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

// Discover repository names first; tags outside the requested page are never read.
func fetchRegistryRepositories(ctx context.Context, registry models.Registry, search string) ([]string, error) {
	out := []string{}
	seen := map[string]bool{}
	visited := map[string]bool{}
	search = strings.ToLower(strings.TrimSpace(search))
	path := "/v2/_catalog?n=1000"
	for {
		if visited[path] {
			return nil, fmt.Errorf("Registry 分页没有前进")
		}
		visited[path] = true
		b, headers, err := release.RegistryRequest(ctx, registry, "GET", path)
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
			if seen[repo] || (search != "" && !strings.Contains(strings.ToLower(repo), search)) {
				continue
			}
			seen[repo] = true
			out = append(out, repo)
		}
		next, err := nextRegistryPage(registry, path, headers)
		if err != nil {
			return nil, err
		}
		if next == "" && len(page.Repositories) == 1000 {
			next = "/v2/_catalog?n=1000&last=" + url.QueryEscape(page.Repositories[len(page.Repositories)-1])
		}
		if next == "" {
			break
		}
		path = next
	}
	return out, nil
}
func fetchRegistryPageTags(ctx context.Context, registry models.Registry, names []string) []RegistryImage {
	out := make([]RegistryImage, len(names))
	jobs := make(chan int, len(names))
	for i, name := range names {
		out[i] = RegistryImage{Name: name, Tags: []string{}}
		jobs <- i
	}
	close(jobs)
	var workers sync.WaitGroup
	for worker := 0; worker < 4 && worker < len(names); worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := range jobs {
				itemCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
				tags, err := fetchImageTags(itemCtx, registry, names[i])
				cancel()
				if err != nil {
					out[i].TagsError = err.Error()
				} else {
					out[i].Tags = tags
				}
			}
		}()
	}
	workers.Wait()
	return out
}

func fetchImageTags(ctx context.Context, registry models.Registry, name string) ([]string, error) {
	tags := []string{}
	path := "/v2/" + name + "/tags/list?n=1000"
	visited := map[string]bool{}
	seen := map[string]bool{}
	for {
		if visited[path] {
			return nil, fmt.Errorf("Registry 标签分页没有前进")
		}
		visited[path] = true
		b, headers, err := release.RegistryRequest(ctx, registry, "GET", path)
		if err != nil {
			return nil, err
		}
		var result struct {
			Tags []string `json:"tags"`
		}
		if err := json.Unmarshal(b, &result); err != nil {
			return nil, err
		}
		for _, tag := range result.Tags {
			if !seen[tag] {
				tags = append(tags, tag)
				seen[tag] = true
			}
		}
		next, err := nextRegistryPage(registry, path, headers)
		if err != nil {
			return nil, err
		}
		if next == "" {
			return tags, nil
		}
		path = next
	}
}

// Follow pagination only within the same registry and endpoint.
func nextRegistryPage(registry models.Registry, path string, headers http.Header) (string, error) {
	for _, header := range headers.Values("Link") {
		for _, link := range strings.Split(header, ",") {
			target, attributes, ok := strings.Cut(strings.TrimSpace(link), ">")
			if !ok || !strings.HasPrefix(target, "<") {
				continue
			}
			isNext := false
			for _, attribute := range strings.Split(attributes, ";") {
				key, value, _ := strings.Cut(strings.TrimSpace(attribute), "=")
				if key == "rel" {
					for _, relation := range strings.Fields(strings.Trim(value, "\"")) {
						if relation == "next" {
							isNext = true
						}
					}
				}
			}
			if !isNext {
				continue
			}
			base, err := url.Parse(strings.TrimRight(registry.URL, "/") + path)
			if err != nil {
				return "", err
			}
			reference, err := url.Parse(strings.TrimPrefix(target, "<"))
			if err != nil {
				return "", err
			}
			next := base.ResolveReference(reference)
			if next.Scheme != base.Scheme || next.Host != base.Host || next.Path != base.Path || next.User != nil || next.Fragment != "" {
				return "", fmt.Errorf("Registry 返回无效的分页地址")
			}
			return next.RequestURI(), nil
		}
	}
	return "", nil
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
