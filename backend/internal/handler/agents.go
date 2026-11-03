package handler

import (
	"dockerun/internal/builtin"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"github.com/distribution/reference"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
	"net/http"
	"strings"
	"sync"
	"time"
)

var agentMu sync.Mutex

func ListTargets(c *gin.Context) {
	var rows []models.Target
	database.DB.Find(&rows)
	c.JSON(200, rows)
}
func CreateTarget(c *gin.Context) {
	var q struct {
		Name string `json:"name"`
	}
	if c.ShouldBindJSON(&q) != nil || q.Name == "" {
		c.Status(400)
		return
	}
	raw := "dka_" + identity.Random()
	t := models.Target{Name: q.Name, TokenHash: identity.Hash(raw), Enabled: true}
	if err := database.DB.Create(&t).Error; err != nil {
		fail(c, err)
		return
	}
	identity.Audit(c.GetUint("user_id"), "target.create", fmt.Sprint(t.ID))
	c.JSON(201, gin.H{"target": t, "token": raw})
}
func UpdateTarget(c *gin.Context) {
	var q struct {
		Enabled bool `json:"enabled"`
	}
	if c.ShouldBindJSON(&q) != nil || c.Param("id") == "1" {
		c.Status(400)
		return
	}
	database.DB.Model(&models.Target{}).Where("id = ?", c.Param("id")).Update("enabled", q.Enabled)
	identity.Audit(c.GetUint("user_id"), "target.update", c.Param("id"))
	c.JSON(200, gin.H{"message": "已更新"})
}
func AgentAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		var t models.Target
		if !strings.HasPrefix(raw, "dka_") || database.DB.Where("token_hash = ? AND enabled = ?", identity.Hash(raw), true).First(&t).Error != nil {
			c.AbortWithStatus(401)
			return
		}
		instance := c.GetHeader("X-Dockerun-Agent-ID")
		if instance == "" || len(instance) > 100 {
			c.AbortWithStatus(401)
			return
		}
		if t.InstanceID == "" {
			database.DB.Model(&t).Where("instance_id = ? OR instance_id IS NULL", "").Update("instance_id", instance)
			database.DB.First(&t, t.ID)
		}
		if t.InstanceID != instance {
			c.AbortWithStatus(403)
			return
		}
		now := time.Now()
		database.DB.Model(&t).Update("last_seen_at", &now)
		c.Set("target", t)
		c.Next()
	}
}
func agentTask(c *gin.Context) (*models.Deployment, *release.Snapshot, error) {
	v, _ := c.Get("target")
	target := v.(models.Target)
	var d models.Deployment
	if database.DB.First(&d, c.Param("id")).Error != nil || d.Status != "running" {
		return nil, nil, fmt.Errorf("任务未租用")
	}
	var s release.Snapshot
	if json.Unmarshal([]byte(d.Snapshot), &s) != nil || s.Environment.TargetID != target.ID {
		return nil, nil, fmt.Errorf("此凭据不能访问其他节点任务")
	}
	return &d, &s, nil
}
func AgentPoll(c *gin.Context) {
	v, _ := c.Get("target")
	target := v.(models.Target)
	platform := c.Query("platform")
	if platform != "linux/amd64" && platform != "linux/arm64" {
		c.JSON(400, gin.H{"error": "Agent 目标仅支持 Linux amd64 或 arm64"})
		return
	}
	database.DB.Model(&target).Update("platform", platform)
	until := time.Now().Add(25 * time.Second)
	for {
		agentMu.Lock()
		var jobs []models.Deployment
		database.DB.Where("status IN ? AND legacy_script_id = ?", []string{"queued", "running"}, 0).Order("id").Find(&jobs)
		for _, d := range jobs {
			var s release.Snapshot
			if json.Unmarshal([]byte(d.Snapshot), &s) != nil || s.Environment.TargetID != target.ID {
				continue
			}
			if d.Status == "running" && d.LeaseUntil != nil && d.LeaseUntil.After(time.Now()) {
				agentMu.Unlock()
				c.Status(204)
				return
			}
			if release.HasScripts(s.Scripts) && c.Query("scripts") != "1" {
				release.RemoteFailure(&d, "failed", "目标 Agent 不支持发布脚本，请更新 Agent 后重新发布")
				continue
			}
			if release.RequiresExecutionAgent(s.Release) && c.Query("execution") != "1" {
				release.RemoteFailure(&d, "failed", "目标 Agent 不支持此部署配置，请更新 Agent 后重新发布")
				continue
			}
			if release.RequiresCommandImageAgent(s.Release) && c.Query("command_images") != "1" {
				release.RemoteFailure(&d, "failed", "目标 Agent 不支持命令部署的触发镜像，请更新 Agent 后重新发布")
				continue
			}
			if release.RequiresContainerAgent(s) && c.Query("containers") != "1" {
				release.RemoteFailure(&d, "failed", "目标 Agent 不支持关联容器替换与镜像核对，请更新 Agent 后重新发布")
				continue
			}
			if s.Environment.ComposeProjectName != "" && c.Query("compose_project_name") != "1" {
				release.RemoteFailure(&d, "failed", "目标 Agent 不支持自定义 Compose 项目名称，请更新 Agent 后重新发布")
				continue
			}
			if (s.Release.BundleID != 0 && release.IsCommand(s.Release) || release.RequiresContainerAgent(s) || !d.Automatic) && c.Query("release_policy") != "2" {
				release.RemoteFailure(&d, "failed", "目标 Agent 不支持离线镜像与手动容器替换策略，请更新 Agent 后重新发布")
				continue
			}
			s.Automatic = d.Automatic
			var env models.Environment
			var user models.User
			if database.DB.First(&env, d.EnvironmentID).Error != nil || database.DB.First(&user, d.UserID).Error != nil || !user.Enabled || !d.Automatic && !user.IsAdmin || d.Automatic && (!env.AutoDeploy || !release.PublisherAllowed(env, d.UserID)) {
				database.DB.Model(&d).Updates(map[string]any{"status": "cancelled", "result": "发布授权已撤销"})
				continue
			}
			var r models.Release
			database.DB.First(&r, d.ReleaseID)
			var reg models.Registry
			auth := ""
			if r.Source == "registry" {
				if database.DB.First(&reg, r.RegistryID).Error != nil {
					continue
				}
				var ims []release.Image
				json.Unmarshal([]byte(r.Images), &ims)
				var repos []string
				for _, im := range ims {
					if n, e := reference.ParseNormalizedNamed(im.Reference); e == nil {
						repos = append(repos, reference.Path(n))
					}
				}
				if reg.Kind == "external" {
					if !reg.PullOnlyConfirmed || reg.PullUsername == "" || reg.PullPassword == "" {
						release.RemoteFailure(&d, "failed", "外部仓库未配置独立的只读 Agent 拉取账号")
						continue
					}
					reg.Username = reg.PullUsername
					reg.Password = reg.PullPassword
				}
				var err error
				auth, err = release.RegistryCredentials(&reg, repos)
				if err != nil {
					continue
				}
			}
			var previous *release.Snapshot
			previousAuth := ""
			if s.Environment.Stateless && s.Environment.AutoRollback && d.PreviousReleaseID > 0 {
				var prior models.Deployment
				if database.DB.Where("environment_id = ? AND target_id = ? AND release_id = ? AND status = ?", d.EnvironmentID, target.ID, d.PreviousReleaseID, "success").Order("id DESC").First(&prior).Error == nil {
					var ps release.Snapshot
					if json.Unmarshal([]byte(prior.Snapshot), &ps) == nil {
						previous = &ps
					}
				}
			}
			now := time.Now()
			lease := now.Add(90 * time.Second)
			database.DB.Model(&d).Updates(map[string]any{"status": "running", "lease_until": lease, "started_at": &now})
			agentMu.Unlock()
			c.JSON(200, gin.H{"deployment": d, "snapshot": s, "registry_auth": auth, "previous_snapshot": previous, "previous_auth": previousAuth, "lease_until": lease})
			return
		}
		agentMu.Unlock()
		if time.Now().After(until) {
			c.Status(204)
			return
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-time.After(time.Second):
		}
	}
}
func AgentHeartbeat(c *gin.Context) {
	d, _, err := agentTask(c)
	if err != nil {
		c.Status(403)
		return
	}
	t := time.Now().Add(90 * time.Second)
	database.DB.Model(d).Update("lease_until", t)
	c.JSON(200, gin.H{"lease_until": t})
}
func AgentLog(c *gin.Context) {
	builtin.ReferenceGate.RLock()
	defer builtin.ReferenceGate.RUnlock()
	d, _, err := agentTask(c)
	if err != nil {
		c.Status(403)
		return
	}
	var q struct {
		Sequence int    `json:"sequence"`
		Stage    string `json:"stage"`
		Message  string `json:"message"`
	}
	if c.ShouldBindJSON(&q) != nil || len(q.Message) > 1<<20 {
		c.Status(400)
		return
	}
	key := fmt.Sprintf("agent-%d-%d", d.ID, q.Sequence)
	if err := database.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.TaskLog{DeploymentID: d.ID, EventKey: &key, Stage: q.Stage, Message: q.Message}).Error; err != nil {
		c.Status(503)
		return
	}
	database.DB.Model(d).Update("stage", q.Stage)
	c.Status(204)
}
func AgentComplete(c *gin.Context) {
	d, s, err := agentTask(c)
	if err != nil {
		c.Status(403)
		return
	}
	var q struct {
		Containers       []models.RuntimeContainer `json:"containers"`
		Status           string                    `json:"status"`
		ScriptsCompleted bool                      `json:"scripts_completed"`
		RecoveryStatus   string                    `json:"recovery_status"`
		Result           string                    `json:"result"`
		ActualImages     string                    `json:"actual_images"`
		Health           string                    `json:"health"`
		Images           []release.Image           `json:"images"`
	}
	if c.ShouldBindJSON(&q) != nil || q.Status != "success" && q.Status != "failed" && q.Status != "needs_attention" {
		c.Status(400)
		return
	}
	s.Images = q.Images
	s.Containers = q.Containers
	s.ScriptsCompleted = q.ScriptsCompleted
	if q.Status == "success" && release.IsCommand(s.Release) && release.RequiresContainerAgent(*s) {
		policy, _ := release.ContainerSettings(s.Environment)
		for _, name := range policy.Names {
			found := false
			for _, row := range q.Containers {
				if row.Name == name && row.ID != "" && row.ImageID != "" && row.State == "running" {
					found = true
					break
				}
			}
			if !found {
				q.Status = "needs_attention"
				q.Result = "节点未上报关联容器的实际运行镜像，请核对容器与 Agent 版本"
				break
			}
		}
	}
	if q.Status == "success" && (release.HasScripts(s.Scripts) || release.RequiresExecutionAgent(s.Release)) && !q.ScriptsCompleted {
		q.Status = "needs_attention"
		q.Result = "节点尚未确认脚本执行完成，请更新 Agent 并核对发布结果"
	}
	b, _ := json.Marshal(s)
	d.Snapshot = string(b)
	database.DB.Model(d).Update("snapshot", d.Snapshot)
	database.DB.Model(d).Update("recovery_status", q.RecoveryStatus)
	if q.Status == "success" {
		release.RemoteSuccess(d, q.ActualImages, q.Health)
	} else {
		if q.Status == "failed" || q.Containers != nil {
			release.RuntimeFailure(d)
			if q.Status == "failed" && q.RecoveryStatus == "success" {
				release.Recovered(d, d.PreviousReleaseID)
			}
		}
		release.RemoteFailure(d, q.Status, q.Result)
	}
	c.Status(204)
}
func AgentBundle(c *gin.Context) {
	_, s, err := agentTask(c)
	if err != nil || s.Release.BundleID == 0 {
		c.Status(http.StatusForbidden)
		return
	}
	var b models.Bundle
	if database.DB.First(&b, s.Release.BundleID).Error != nil {
		c.Status(404)
		return
	}
	c.Header("X-Archive-SHA256", b.SHA256)
	c.Header("X-Archive-Kind", b.Kind)
	c.File(b.Path)
}
