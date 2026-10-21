package handler

import (
	"dockerun/internal/bundle"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/distribution/reference"
	dockertypes "github.com/docker/docker/api/types"
	regtypes "github.com/docker/docker/api/types/registry"
	"github.com/gin-gonic/gin"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func PublishBundle(c *gin.Context) {
	var q struct {
		ReleaseID uint   `json:"release_id"`
		Prefix    string `json:"prefix"`
	}
	if c.ShouldBindJSON(&q) != nil {
		c.Status(400)
		return
	}
	var r models.Release
	if database.DB.First(&r, q.ReleaseID).Error != nil || r.Source != "bundle" {
		fail(c, fmt.Errorf("请选择已导入发布包的版本"))
		return
	}
	var images []release.Image
	if json.Unmarshal([]byte(r.Images), &images) != nil {
		c.Status(400)
		return
	}
	var reg models.Registry
	database.DB.Where("kind = ?", "builtin").First(&reg)
	host := strings.TrimPrefix(strings.TrimPrefix(config.Load().PublicURL, "https://"), "http://")
	v, _ := c.Get("user")
	u := v.(*models.User)
	if q.Prefix == "" {
		q.Prefix = fmt.Sprintf("offline/app-%d", r.ApplicationID)
	}
	for i, im := range images {
		if _, _, err := docker.GetClient().ImageInspectWithRaw(c.Request.Context(), im.ID); err != nil {
			archive, err := release.SafePath(filepath.Join(config.Load().DataDir, "releases", fmt.Sprintf("bundle-%d", r.BundleID)), im.Archive)
			if err != nil {
				fail(c, err)
				return
			}
			hash, err := bundle.HashFile(archive)
			if err != nil || hash != im.SHA256 {
				fail(c, fmt.Errorf("包内镜像归档缺失或校验失败"))
				return
			}
			file, err := os.Open(archive)
			if err != nil {
				fail(c, err)
				return
			}
			response, err := docker.GetClient().ImageLoad(c.Request.Context(), file, false)
			if err == nil {
				err = release.Consume(response.Body, nil)
				response.Body.Close()
			}
			file.Close()
			if err != nil {
				fail(c, err)
				return
			}
			loaded, _, err := docker.GetClient().ImageInspectWithRaw(c.Request.Context(), im.ID)
			if err != nil || loaded.ID != im.ID || loaded.Os+"/"+loaded.Architecture != im.Platform {
				fail(c, fmt.Errorf("离线镜像身份或平台不匹配"))
				return
			}
		}
		repo := fmt.Sprintf("%s/image-%d", q.Prefix, i)
		target := host + "/" + repo + ":release-" + fmt.Sprint(r.ID)
		if _, err := reference.ParseNormalizedNamed(target); err != nil {
			fail(c, err)
			return
		}
		token, err := identity.Issue(u, "registry-import", []string{"repository:" + repo + ":pull", "repository:" + repo + ":push"}, 5*time.Minute)
		if err != nil {
			fail(c, err)
			return
		}
		auth, _ := json.Marshal(regtypes.AuthConfig{RegistryToken: token, ServerAddress: config.Load().PublicURL})
		if err = docker.GetClient().ImageTag(c.Request.Context(), im.ID, target); err != nil {
			fail(c, err)
			return
		}
		reader, err := docker.GetClient().ImagePush(c.Request.Context(), target, dockertypes.ImagePushOptions{RegistryAuth: base64.URLEncoding.EncodeToString(auth)})
		if err != nil {
			fail(c, err)
			return
		}
		err = release.Consume(reader, nil)
		reader.Close()
		if err != nil {
			fail(c, err)
			return
		}
		resolved, err := release.Resolve(c.Request.Context(), reg, target)
		if err != nil {
			fail(c, err)
			return
		}
		resolved.Original = im.Reference
		resolved.Platform = im.Platform
		images[i] = resolved
	}
	r.ID = 0
	r.Source = "registry"
	r.RegistryID = reg.ID
	r.BundleID = 0
	r.Version += "-registry"
	r.CreatedAt = time.Time{}
	data, _ := json.Marshal(images)
	r.Images = string(data)
	if err := release.SaveRelease(&r); err != nil {
		fail(c, err)
		return
	}
	identity.Audit(u.ID, "bundle.publish", fmt.Sprint(r.ID))
	c.JSON(201, r)
}
