package handler

import (
	"context"
	"dockerun/internal/apptemplate"
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"github.com/distribution/reference"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func fail(c *gin.Context, err error) { c.JSON(400, gin.H{"error": err.Error()}) }
func ListApplications(c *gin.Context) {
	var rows []models.Application
	database.DB.Select("id", "created_at", "updated_at", "name", "description", "kind", "category", "template_id", "template_version").Order("id DESC").Find(&rows)
	c.JSON(200, rows)
}
func CreateApplication(c *gin.Context) {
	var q struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if c.ShouldBindJSON(&q) != nil || q.Name == "" || len(q.Name) > 100 {
		fail(c, fmt.Errorf("请填写应用名称"))
		return
	}
	a := models.Application{Name: q.Name, Description: q.Description, Kind: "compose"}
	if err := database.DB.Create(&a).Error; err != nil {
		fail(c, err)
		return
	}
	c.JSON(201, a)
}
func ApplicationDetail(c *gin.Context) {
	var a models.Application
	if err := database.DB.First(&a, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"error": "应用不存在"})
		return
	}
	envs := []environmentSummary{}
	versions := []releaseSummary{}
	if err := database.DB.Model(&models.Environment{}).Where("application_id = ?", a.ID).Find(&envs).Error; err != nil {
		fail(c, err)
		return
	}
	if err := database.DB.Model(&models.Release{}).Where("application_id = ?", a.ID).Order("id DESC").Find(&versions).Error; err != nil {
		fail(c, err)
		return
	}
	var states []models.TargetState
	if err := database.DB.Where("environment_id IN (?)", database.DB.Model(&models.Environment{}).Select("id").Where("application_id = ?", a.ID)).Find(&states).Error; err != nil {
		fail(c, err)
		return
	}
	ids := make([]uint, 0, len(versions))
	for _, version := range versions {
		ids = append(ids, version.ID)
	}
	reasons, err := releaseDeletionReasons(database.DB, ids)
	if err != nil {
		fail(c, err)
		return
	}
	for i := range versions {
		versions[i].DeletionBlockedReason = reasons[versions[i].ID]
	}
	environmentIDs := make([]uint, 0, len(envs))
	for _, environment := range envs {
		environmentIDs = append(environmentIDs, environment.ID)
	}
	environmentReasons, err := environmentDeletionReasons(database.DB, environmentIDs)
	if err != nil {
		fail(c, err)
		return
	}
	for i := range envs {
		envs[i].DeletionBlockedReason = environmentReasons[envs[i].ID]
	}
	c.JSON(200, gin.H{"application": a, "environments": envs, "releases": versions, "targets": states})
}
func SaveApplicationScripts(c *gin.Context) {
	builtin.ReferenceGate.RLock()
	defer builtin.ReferenceGate.RUnlock()
	var q struct {
		Scripts models.DeploymentScripts `json:"scripts"`
	}
	if err := c.ShouldBindJSON(&q); err != nil {
		fail(c, err)
		return
	}
	if err := release.ValidateScripts(q.Scripts); err != nil {
		fail(c, err)
		return
	}
	var app models.Application
	if database.DB.First(&app, c.Param("id")).Error != nil {
		c.JSON(404, gin.H{"error": "应用不存在"})
		return
	}
	app.Scripts = q.Scripts
	if err := database.DB.Save(&app).Error; err != nil {
		fail(c, err)
		return
	}
	identity.Audit(c.GetUint("user_id"), "application.scripts.update", fmt.Sprint(app.ID))
	c.JSON(200, app)
}
func SaveEnvironment(c *gin.Context) {
	builtin.ReferenceGate.Lock()
	defer builtin.ReferenceGate.Unlock()
	var q models.Environment
	if err := c.ShouldBindJSON(&q); err != nil {
		fail(c, err)
		return
	}
	var a models.Application
	if err := database.DB.First(&a, c.Param("id")).Error; err != nil {
		fail(c, err)
		return
	}
	if len(q.TargetIDs) > 10000 {
		fail(c, fmt.Errorf("目标列表过长"))
		return
	}
	q.ApplicationID = a.ID
	if a.Definition.Topology == "single_host" && q.TargetIDs != "" && q.TargetIDs != "[]" {
		fail(c, fmt.Errorf("此模板仅支持单台目标主机，固定主机组会重复部署整套服务，不能用于集群编排"))
		return
	}
	if a.Definition.Stateful && (q.Stateless || q.AutoRollback) {
		fail(c, fmt.Errorf("此应用包含持久数据，不能启用无状态自动恢复"))
		return
	}
	if q.ID > 0 {
		var old models.Environment
		if database.DB.Where("id = ? AND application_id = ?", q.ID, a.ID).First(&old).Error != nil {
			c.JSON(404, gin.H{"error": "环境不存在"})
			return
		}
		q.CurrentReleaseID = old.CurrentReleaseID
	} else {
		q.CurrentReleaseID = 0
	}
	if q.ScriptMode == "" {
		q.ScriptMode = "inherit"
	}
	if q.ScriptMode != "inherit" && q.ScriptMode != "override" && q.ScriptMode != "disabled" {
		fail(c, fmt.Errorf("请选择有效的脚本配置方式"))
		return
	}
	if q.ScriptMode != "override" {
		q.Scripts = models.DeploymentScripts{}
	}
	if err := release.ValidateScripts(q.Scripts); err != nil {
		fail(c, err)
		return
	}
	reg, err := release.DefaultRegistry(q.RegistryID)
	if err != nil {
		fail(c, err)
		return
	}
	q.RegistryID = reg.ID
	if len(a.Definition.Parameters) > 0 {
		values, err := apptemplate.Values(apptemplate.Template{Definition: a.Definition}, q.Parameters, reg.URL)
		if err != nil {
			fail(c, err)
			return
		}
		if q.Parameters == nil {
			q.Parameters = models.StringMap{}
		}
		for key, value := range values {
			q.Parameters[key] = value
		}
	}
	q.Repository = strings.TrimSpace(q.Repository)
	if q.TagPattern == "" {
		q.TagPattern = "*"
	}
	if _, err := filepath.Match(q.TagPattern, "test"); err != nil {
		fail(c, fmt.Errorf("标签规则格式无效，请使用 latest、test-* 或 * 等规则"))
		return
	}
	if q.Name == "" || len(q.Name) > 100 {
		fail(c, fmt.Errorf("请填写环境名称"))
		return
	}
	if q.AllowedPublishers == "" {
		q.AllowedPublishers = "[]"
	}
	var ids []uint
	if json.Unmarshal([]byte(q.AllowedPublishers), &ids) != nil {
		fail(c, fmt.Errorf("推送人列表格式错误"))
		return
	}
	if q.TargetIDs != "" && q.TargetIDs != "[]" {
		var ids []uint
		if json.Unmarshal([]byte(q.TargetIDs), &ids) != nil {
			fail(c, fmt.Errorf("目标主机组格式无效"))
			return
		}
		for _, id := range ids {
			var node models.Target
			if database.DB.First(&node, id).Error != nil || !node.Enabled {
				fail(c, fmt.Errorf("主机不存在或未启用"))
				return
			}
		}
	}
	if q.AutoDeploy {
		name, err := reference.WithName("registry.invalid/" + q.Repository)
		if err != nil || reference.Path(name) != q.Repository {
			fail(c, fmt.Errorf("请填写镜像名称，不含仓库域名和标签，例如 team/gateway"))
			return
		}
		if err := release.ValidateContainerBindings(q); err != nil {
			fail(c, err)
			return
		}
		if _, err := release.AutomationBase(q); err != nil {
			fail(c, err)
			return
		}
	}
	if q.AutoRollback && !q.Stateless {
		fail(c, fmt.Errorf("仅无状态应用可启用自动恢复"))
		return
	}
	if q.ID == 0 {
		q.CurrentReleaseID = 0
		q.Health = "unknown"
		q.RuntimeState = "unknown"
		if err := database.DB.Create(&q).Error; err != nil {
			fail(c, err)
			return
		}
	} else {
		scripts, _ := json.Marshal(q.Scripts)
		names, _ := json.Marshal(q.ContainerNames)
		updates := map[string]any{"name": q.Name, "target_id": q.TargetID, "target_ids": q.TargetIDs, "parameters": q.Parameters, "stateless": q.Stateless, "auto_rollback": q.AutoRollback, "auto_deploy": q.AutoDeploy, "registry_id": q.RegistryID, "repository": q.Repository, "tag_pattern": q.TagPattern, "allowed_publishers": q.AllowedPublishers, "script_mode": q.ScriptMode, "scripts": string(scripts)}
		updates["container_names"] = string(names)
		updates["replace_containers"] = q.ReplaceContainers
		res := database.DB.Model(&models.Environment{}).Where("id = ? AND application_id = ?", q.ID, a.ID).Updates(updates)
		if res.Error != nil {
			fail(c, res.Error)
			return
		}
		if res.RowsAffected == 0 {
			c.JSON(404, gin.H{"error": "环境不存在"})
			return
		}
	}
	identity.Audit(c.GetUint("user_id"), "environment.update", fmt.Sprint(q.ID))
	database.DB.First(&q, q.ID)
	c.JSON(200, q)
}
func CreateRelease(c *gin.Context) {
	builtin.ReferenceGate.RLock()
	defer builtin.ReferenceGate.RUnlock()
	var request struct {
		models.Release
		SaveDefinition bool `json:"save_definition"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, err)
		return
	}
	q := request.Release
	definitionCompose := q.Compose
	var a models.Application
	if err := database.DB.First(&a, c.Param("id")).Error; err != nil {
		fail(c, err)
		return
	}
	if q.Source == "local" {
		q.RegistryID = 0
	}
	q.ID = 0
	q.ApplicationID = a.ID
	q.PublisherID = c.GetUint("user_id")
	q.Retained = true
	q.Scripts = a.Scripts
	if err := release.ValidateExecution(q.Execution); err != nil {
		fail(c, err)
		return
	}
	if release.IsCommand(q) {
		q.Source = "command"
		q.RegistryID = 0
	}
	if q.Source == "command" && !release.IsCommand(q) {
		fail(c, fmt.Errorf("命令来源必须使用命令部署"))
		return
	}
	if q.Source != "registry" && q.Source != "local" && q.Source != "direct" && q.Source != "command" {
		fail(c, fmt.Errorf("请选择仓库、本机镜像，或通过发布包创建版本"))
		return
	}
	if q.Version == "" {
		q.Version = time.Now().Format("20060102-150405")
	}
	var params map[string]string
	if v := c.Query("environment_id"); v != "" {
		var env models.Environment
		if database.DB.Where("id = ? AND application_id = ?", v, a.ID).First(&env).Error != nil {
			fail(c, fmt.Errorf("环境不存在"))
			return
		}
		params = env.Parameters
	}
	dir, err := os.MkdirTemp(filepath.Join(config.Load().DataDir, "releases"), "validate-")
	if err != nil {
		fail(c, err)
		return
	}
	defer os.RemoveAll(dir)
	var configs map[string]string
	if q.Configs != "" && json.Unmarshal([]byte(q.Configs), &configs) != nil {
		fail(c, fmt.Errorf("配置文件格式错误"))
		return
	}
	if err = release.WriteConfigs(dir, configs); err != nil {
		fail(c, err)
		return
	}
	if release.IsCommand(q) {
		q.Images = "[]"
		q.BundleID = 0
		if err = database.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&q).Error; err != nil {
				return err
			}
			if request.SaveDefinition {
				a.Definition.Execution = q.Execution
				a.Kind = "command"
				a.Definition.Compose = definitionCompose
				a.Definition.Configs = configs
				return tx.Save(&a).Error
			}
			return nil
		}); err != nil {
			fail(c, err)
			return
		}
		c.JSON(201, q)
		return
	}
	p, err := release.Parse(c.Request.Context(), q.Compose, dir, "dockerun-preview", params)
	if err != nil {
		fail(c, err)
		return
	}
	q.Compose, err = release.FreezeImages(q.Compose, p)
	if err != nil {
		fail(c, err)
		return
	}
	var images []release.Image
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()
	var reg models.Registry
	if q.Source == "registry" {
		reg, err = release.DefaultRegistry(q.RegistryID)
		if err != nil {
			fail(c, err)
			return
		}
		q.RegistryID = reg.ID
	}
	for _, ref := range release.References(p) {
		var im release.Image
		if q.Source == "registry" {
			im, err = release.Resolve(ctx, reg, ref)
		} else if q.Source == "direct" {
			im = release.Image{Reference: ref}
		} else {
			v, _, e := docker.GetClient().ImageInspectWithRaw(ctx, ref)
			err = e
			im = release.Image{Reference: ref, ID: v.ID, Platform: v.Os + "/" + v.Architecture}
		}
		if err != nil {
			fail(c, err)
			return
		}
		images = append(images, im)
	}
	b, _ := json.Marshal(images)
	q.Images = string(b)
	q.BundleID = 0
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&q).Error; err != nil {
			return err
		}
		if request.SaveDefinition || a.Definition.Compose == "" {
			a.Definition.Compose = definitionCompose
			a.Definition.Execution = q.Execution
			a.Kind = "compose"
			a.Definition.Source = q.Source
			a.Definition.RegistryID = q.RegistryID
			a.Definition.Platform = q.Platform
			a.Definition.Configs = configs
			a.Definition.Parameters = apptemplate.Parameters(definitionCompose, a.Definition.Parameters)
			if a.Category == "" {
				a.Category = "business"
			}
			return tx.Save(&a).Error
		}
		return nil
	}); err != nil {
		fail(c, err)
		return
	}
	c.JSON(201, q)
}
func EnqueueDeployment(c *gin.Context) {
	var q struct {
		EnvironmentID        uint                      `json:"environment_id"`
		ReleaseID            uint                      `json:"release_id"`
		Key                  string                    `json:"idempotency_key"`
		RestoreConfiguration bool                      `json:"restore_configuration"`
		ExpectedParameters   *models.StringMap         `json:"expected_parameters"`
		ExpectedScripts      *models.DeploymentScripts `json:"expected_scripts"`
		ExpectedContainers   *release.ContainerPolicy  `json:"expected_containers"`
	}
	if c.ShouldBindJSON(&q) != nil {
		fail(c, fmt.Errorf("请求无效"))
		return
	}
	var env models.Environment
	var r models.Release
	if database.DB.First(&env, q.EnvironmentID).Error != nil || database.DB.First(&r, q.ReleaseID).Error != nil {
		c.JSON(404, gin.H{"error": "环境或版本不存在"})
		return
	}
	if !checkExpectedEnvironmentParameters(c, q.ExpectedParameters, env.Parameters) {
		return
	}
	if err := restoreParameters(&env, r, q.RestoreConfiguration); err != nil {
		fail(c, err)
		return
	}
	if !checkExpectedContainerPolicy(c, q.ExpectedContainers, env) {
		return
	}
	if !checkExpectedScripts(c, q.ExpectedScripts, r, env) {
		return
	}
	d, err := release.Enqueue(env, r, c.GetUint("user_id"), q.Key, false, 0)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(202, d)
}
func PreflightDeployment(c *gin.Context) {
	var q struct {
		EnvironmentID        uint                      `json:"environment_id"`
		ReleaseID            uint                      `json:"release_id"`
		RestoreConfiguration bool                      `json:"restore_configuration"`
		ExpectedParameters   *models.StringMap         `json:"expected_parameters"`
		ExpectedScripts      *models.DeploymentScripts `json:"expected_scripts"`
		ExpectedContainers   *release.ContainerPolicy  `json:"expected_containers"`
	}
	if c.ShouldBindJSON(&q) != nil {
		fail(c, fmt.Errorf("请求无效"))
		return
	}
	var env models.Environment
	var r models.Release
	if database.DB.First(&env, q.EnvironmentID).Error != nil || database.DB.First(&r, q.ReleaseID).Error != nil || r.ApplicationID != env.ApplicationID {
		fail(c, fmt.Errorf("版本和环境不匹配"))
		return
	}
	if !checkExpectedEnvironmentParameters(c, q.ExpectedParameters, env.Parameters) {
		return
	}
	if err := restoreParameters(&env, r, q.RestoreConfiguration); err != nil {
		fail(c, err)
		return
	}
	if !checkExpectedContainerPolicy(c, q.ExpectedContainers, env) {
		return
	}
	if !checkExpectedScripts(c, q.ExpectedScripts, r, env) {
		return
	}
	if release.IsCommand(r) {
		var application models.Application
		if database.DB.First(&application, r.ApplicationID).Error != nil {
			fail(c, fmt.Errorf("应用不存在"))
			return
		}
		if err := release.ValidateExecution(r.Execution); err != nil {
			fail(c, err)
			return
		}
		var current models.Release
		database.DB.First(&current, env.CurrentReleaseID)
		c.JSON(200, gin.H{"valid": true, "environment": env.Name, "target_id": env.TargetID, "images": []release.Image{}, "parameters": apptemplate.RedactParameters(env.Parameters, application.Definition.Parameters), "execution": r.Execution, "previous_execution": current.Execution, "scripts": release.EffectiveScripts(r, env), "container_policy": containerPolicy(env), "message": commandDeploymentNotice(env)})
		return
	}
	var images []release.Image
	json.Unmarshal([]byte(r.Images), &images)
	dir, err := os.MkdirTemp(filepath.Join(config.Load().DataDir, "releases"), "preview-")
	if err != nil {
		fail(c, err)
		return
	}
	defer os.RemoveAll(dir)
	// Preview validates definition and shows immutable identities. Real image and port checks run again in the task.
	var configs map[string]string
	json.Unmarshal([]byte(r.Configs), &configs)
	release.WriteConfigs(dir, configs)
	p, err := release.Parse(c.Request.Context(), r.Compose, dir, release.ProjectName(env), env.Parameters)
	if err != nil {
		fail(c, err)
		return
	}
	var app models.Application
	if database.DB.First(&app, r.ApplicationID).Error != nil {
		fail(c, fmt.Errorf("应用不存在"))
		return
	}
	p, err = release.Parse(c.Request.Context(), r.Compose, dir, release.ProjectName(env), apptemplate.RedactParameters(env.Parameters, app.Definition.Parameters))
	if err != nil {
		fail(c, err)
		return
	}
	content, _ := release.Encode(p)
	var current models.Release
	database.DB.First(&current, env.CurrentReleaseID)
	c.JSON(200, gin.H{"valid": true, "environment": env.Name, "target_id": env.TargetID, "parameters": apptemplate.RedactParameters(env.Parameters, app.Definition.Parameters), "images": images, "compose": content, "previous_compose": current.Compose, "execution": r.Execution, "previous_execution": current.Execution, "scripts": release.EffectiveScripts(r, env), "container_policy": containerPolicy(env), "message": "定义检查通过；脚本只在确认部署后执行，执行前还将检查全部镜像、平台、端口、挂载和网络"})
}
func ListDeployments(c *gin.Context) {
	rows := []deploymentSummary{}
	q := deploymentSummaries().Order("deployments.id DESC").Limit(200)
	if v := c.Query("environment_id"); v != "" {
		q = q.Where("deployments.environment_id = ?", v)
	}
	if err := q.Find(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	for i := range rows {
		describeDeployment(&rows[i])
	}
	c.JSON(200, rows)
}
func DeploymentDetail(c *gin.Context) {
	var d deploymentSummary
	if deploymentSummaries().Where("deployments.id = ?", c.Param("id")).First(&d).Error != nil {
		c.JSON(404, gin.H{"error": "任务不存在"})
		return
	}
	describeDeployment(&d)
	var logs []models.TaskLog
	after, _ := strconv.Atoi(c.DefaultQuery("after", "0"))
	database.DB.Where("deployment_id = ? AND id > ?", d.ID, after).Order("id").Find(&logs)
	nodes := []deploymentSummary{}
	if d.Batch != "" {
		deploymentSummaries().Where("deployments.batch = ?", d.Batch).Order("deployments.sequence").Find(&nodes)
		for i := range nodes {
			describeDeployment(&nodes[i])
		}
	}
	var snap release.Snapshot
	json.Unmarshal([]byte(d.Snapshot), &snap)
	c.JSON(200, gin.H{"deployment": d, "logs": logs, "nodes": nodes, "scripts": snap.Scripts})
}
func DeploymentEvents(c *gin.Context) {
	var d models.Deployment
	if database.DB.Where("history_deleted_at IS NULL").First(&d, c.Param("id")).Error != nil {
		c.Status(404)
		return
	}
	after, _ := strconv.Atoi(c.GetHeader("Last-Event-ID"))
	if v := c.Query("after"); v != "" {
		after, _ = strconv.Atoi(v)
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		var logs []models.TaskLog
		database.DB.Where("deployment_id = ? AND id > ?", d.ID, after).Order("id").Find(&logs)
		for _, l := range logs {
			fmt.Fprintf(c.Writer, "id: %d\nevent: log\ndata: ", l.ID)
			b, _ := json.Marshal(l)
			c.Writer.Write(b)
			c.Writer.WriteString("\n\n")
			after = int(l.ID)
		}
		if database.DB.Where("history_deleted_at IS NULL").First(&d, d.ID).Error != nil {
			return
		}
		c.SSEvent("status", d)
		c.Writer.Flush()
		if d.Status != "queued" && d.Status != "running" {
			return
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
		}
	}
}
func Promote(c *gin.Context) {
	var q struct {
		EnvironmentID uint   `json:"environment_id"`
		Key           string `json:"idempotency_key"`
	}
	if c.ShouldBindJSON(&q) != nil {
		fail(c, fmt.Errorf("请求无效"))
		return
	}
	var r models.Release
	var env models.Environment
	if database.DB.First(&r, c.Param("id")).Error != nil || database.DB.First(&env, q.EnvironmentID).Error != nil {
		fail(c, fmt.Errorf("版本或环境不存在"))
		return
	}
	d, err := release.Enqueue(env, r, c.GetUint("user_id"), q.Key, false, 0)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(202, d)
}
func Overview(c *gin.Context) {
	counts := map[string]int64{}
	for k, m := range map[string]any{"applications": &models.Application{}, "deployments": &models.Deployment{}, "bundles": &models.Bundle{}, "image_events": &models.ImageEvent{}} {
		var n int64
		query := database.DB.Model(m)
		if k == "deployments" {
			query = query.Where("history_deleted_at IS NULL")
		}
		query.Count(&n)
		counts[k] = n
	}
	var envs []models.Environment
	database.DB.Find(&envs)
	c.JSON(200, gin.H{"counts": counts, "environments": envs})
}
func ReleaseRetention(c *gin.Context) {
	var r models.Release
	if database.DB.First(&r, c.Param("id")).Error != nil {
		c.Status(404)
		return
	}
	if reason := release.RetentionProtection(r); reason != "" {
		fail(c, fmt.Errorf("%s", reason))
		return
	}

	database.DB.Model(&r).Update("retained", false)
	c.JSON(200, r)
}

func ApplicationSources(c *gin.Context) {
	var regs []models.Registry
	database.DB.Select("id", "name", "url", "kind", "is_default").Find(&regs)
	var targets []models.Target
	database.DB.Find(&targets)
	c.JSON(200, gin.H{"registries": regs, "targets": targets})
}

func restoreParameters(env *models.Environment, r models.Release, restore bool) error {
	if !restore {
		return nil
	}
	var d models.Deployment
	if database.DB.Where("environment_id = ? AND release_id = ? AND status = ?", env.ID, r.ID, "success").Order("id DESC").First(&d).Error != nil {
		return fmt.Errorf("该环境没有所选版本的成功配置快照，可取消恢复配置后按当前参数部署")
	}
	var snap release.Snapshot
	if err := json.Unmarshal([]byte(d.Snapshot), &snap); err != nil {
		return err
	}
	env.Parameters = snap.Environment.Parameters
	env.ScriptMode = "override"
	env.Scripts = snap.Scripts
	env.RestoreConfiguration = true
	return nil
}
