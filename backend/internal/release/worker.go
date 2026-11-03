package release

import (
	"context"
	"dockerun/internal/builtin"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

type Snapshot struct {
	Automatic           bool                      `json:"automatic,omitempty"`
	Containers          []models.RuntimeContainer `json:"containers,omitempty"`
	SensitiveParameters []string                  `json:"sensitive_parameters,omitempty"`
	Release             models.Release            `json:"release"`
	Environment         models.Environment        `json:"environment"`
	Images              []Image                   `json:"images"`
	Scripts             models.DeploymentScripts  `json:"scripts"`
	ScriptsCompleted    bool                      `json:"scripts_completed"`
}

var ExecutionGate sync.RWMutex
var stop context.CancelFunc
var workerWG sync.WaitGroup

func Start() {
	ctx, cancel := context.WithCancel(context.Background())
	stop = cancel
	reconcile(ctx)
	workerWG.Add(1)
	go func() {
		defer workerWG.Done()
		t := time.NewTicker(time.Second)
		defer t.Stop()
		ticks := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				ticks++
				if ticks%15 == 0 {
					reconcileRuntime(ctx)
				}
				processEvents(ctx)
				var d models.Deployment
				if database.DB.Where("status = ? AND target_id <= ?", "queued", 1).Order("id").First(&d).Error == nil {
					execute(ctx, &d)
				}
			}
		}
	}()
}
func Stop() {
	if stop != nil {
		stop()
	}
	workerWG.Wait()
}
func Enqueue(env models.Environment, r models.Release, uid uint, key string, automatic bool, event uint) (*models.Deployment, error) {
	builtin.ReferenceGate.RLock()
	defer builtin.ReferenceGate.RUnlock()
	var application models.Application
	var storedEnvironment models.Environment
	var storedRelease models.Release
	if database.DB.First(&application, r.ApplicationID).Error != nil || database.DB.Where("id = ? AND application_id = ?", env.ID, application.ID).First(&storedEnvironment).Error != nil || database.DB.Where("id = ? AND application_id = ?", r.ID, application.ID).First(&storedRelease).Error != nil {
		return nil, fmt.Errorf("应用、环境或发布版本已删除，请刷新后重试")
	}
	if key == "" {
		key = identity.Random()
	}
	var old models.Deployment
	if database.DB.Where("idempotency_key = ?", key).First(&old).Error == nil {
		if old.HistoryDeletedAt != nil {
			return nil, fmt.Errorf("该请求对应的部署记录已删除，请重新发起部署")
		}
		return &old, nil
	}
	if r.ApplicationID != env.ApplicationID {
		return nil, fmt.Errorf("版本不属于此应用")
	}
	if ProjectName(env) != ProjectName(storedEnvironment) {
		return nil, fmt.Errorf("Compose 项目名称已变化，请重新预览后发布")
	}
	if env.ComposeProjectName != "" && (env.TargetID != storedEnvironment.TargetID || env.TargetIDs != storedEnvironment.TargetIDs) {
		return nil, fmt.Errorf("环境目标主机已变化，请重新预览后发布")
	}
	policy, err := ContainerSettings(env)
	if err != nil {
		return nil, err
	}
	currentPolicy, err := ContainerSettings(storedEnvironment)
	if err != nil || policy.Replace != currentPolicy.Replace || !slices.Equal(policy.Names, currentPolicy.Names) {
		return nil, fmt.Errorf("环境关联容器配置已变化，请刷新并重新预览")
	}
	if err := ValidateContainerBindings(env); err != nil {
		return nil, err
	}
	if err := database.DB.Model(&r).Update("retained", true).Error; err != nil {
		return nil, err
	}
	var targets []uint
	if env.TargetIDs != "" && env.TargetIDs != "[]" {
		if err := json.Unmarshal([]byte(env.TargetIDs), &targets); err != nil {
			return nil, err
		}
	}
	if len(targets) == 0 {
		targets = []uint{env.TargetID}
	}
	seen := map[uint]bool{}
	var sensitiveParameters []string
	for _, parameter := range application.Definition.Parameters {
		if parameter.Type == "password" {
			sensitiveParameters = append(sensitiveParameters, parameter.Key)
		}
	}
	var first *models.Deployment
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		for i, target := range targets {
			if target == 0 {
				target = 1
			}
			if seen[target] {
				return fmt.Errorf("目标主机重复")
			}
			seen[target] = true
			var node models.Target
			if tx.First(&node, target).Error != nil || !node.Enabled {
				return fmt.Errorf("目标主机不可用")
			}
			var state models.TargetState
			tx.Where("environment_id = ? AND target_id = ?", env.ID, target).First(&state)
			previous := state.CurrentReleaseID
			if previous == 0 && len(targets) == 1 {
				previous = env.CurrentReleaseID
			}
			copyEnv := env
			copyEnv.TargetID = target
			copyEnv.CurrentReleaseID = previous
			snap, _ := json.Marshal(Snapshot{Automatic: automatic, Release: r, Environment: copyEnv, Scripts: EffectiveScripts(r, copyEnv), SensitiveParameters: sensitiveParameters})
			status := "queued"
			if i > 0 {
				status = "blocked"
			}
			taskKey := key
			if i > 0 {
				taskKey = fmt.Sprintf("%s-node-%d", key, target)
			}
			d := models.Deployment{EnvironmentID: env.ID, ReleaseID: r.ID, PreviousReleaseID: previous, UserID: uid, TargetID: target, Batch: key, Sequence: i, IdempotencyKey: taskKey, Status: status, Stage: status, Automatic: automatic, EventID: event, Snapshot: string(snap)}
			if err := tx.Create(&d).Error; err != nil {
				return err
			}
			if i == 0 {
				first = &d
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	identity.Audit(uid, "deployment.create", key)
	return first, nil
}
func Log(d *models.Deployment, stage, message string) {
	database.DB.Create(&models.TaskLog{DeploymentID: d.ID, Stage: stage, Message: message})
	database.DB.Model(&models.Deployment{}).Where("id = ?", d.ID).Update("stage", stage)
}
func eligible(env models.Environment, uid uint) bool {
	var u models.User
	if database.DB.First(&u, uid).Error != nil || !u.Enabled || !u.CanRegistry("push") {
		return false
	}
	if u.IsAdmin {
		return true
	}
	var ids []uint
	if json.Unmarshal([]byte(env.AllowedPublishers), &ids) != nil {
		return false
	}
	for _, id := range ids {
		if id == uid {
			return true
		}
	}
	return false
}
func finish(d *models.Deployment, status, result string) {
	builtin.ReferenceGate.RLock()
	defer builtin.ReferenceGate.RUnlock()
	now := time.Now()
	resultUpdate := database.DB.Model(d).Where("history_deleted_at IS NULL").Updates(map[string]any{"status": status, "stage": "complete", "result": result, "completed_at": &now})
	if resultUpdate.Error != nil || resultUpdate.RowsAffected == 0 {
		return
	}
	Log(d, "complete", result)
	if status == "success" {
		refreshRetention(d.EnvironmentID)
	}
	if d.Batch != "" {
		if status == "success" {
			database.DB.Model(&models.Deployment{}).Where("batch = ? AND sequence = ? AND status = ?", d.Batch, d.Sequence+1, "blocked").Updates(map[string]any{"status": "queued", "stage": "queued"})
		} else {
			database.DB.Model(&models.Deployment{}).Where("batch = ? AND sequence > ? AND status = ?", d.Batch, d.Sequence, "blocked").Updates(map[string]any{"status": "paused", "result": "上一目标未成功，暂停后续发布"})
		}
	}
}
func execute(parent context.Context, d *models.Deployment) {
	ExecutionGate.RLock()
	defer ExecutionGate.RUnlock()
	if d.LegacyScriptID > 0 {
		executeLegacy(parent, d)
		return
	}
	var env models.Environment
	if database.DB.First(&env, d.EnvironmentID).Error != nil {
		finish(d, "failed", "部署环境不存在")
		return
	}
	if d.TargetID > 1 {
		return
	} // Remote work is leased by its bound Agent.
	var u models.User
	if database.DB.First(&u, d.UserID).Error != nil || !u.Enabled || !d.Automatic && !u.IsAdmin {
		finish(d, "cancelled", "发布身份已失效")
		return
	}
	if d.Automatic && (!env.AutoDeploy || !eligible(env, d.UserID)) {
		finish(d, "cancelled", "自动发布规则或推送人授权已撤销")
		return
	}
	now := time.Now()
	res := database.DB.Model(d).Where("status = ?", "queued").Updates(map[string]any{"status": "running", "started_at": &now})
	if res.RowsAffected != 1 {
		return
	}
	var snap Snapshot
	if json.Unmarshal([]byte(d.Snapshot), &snap) != nil {
		finish(d, "failed", "快照损坏")
		return
	}
	snap.Automatic = d.Automatic
	ctx, cancel := context.WithTimeout(parent, ExecutionTimeout(snap.Release, Timeout()))
	defer cancel()
	dir := JobDir(d.ID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		finish(d, "failed", err.Error())
		return
	}
	if IsCommand(snap.Release) {
		Log(d, "prepare", "准备部署命令与配置文件")
	} else {
		Log(d, "images", "准备全部镜像；完成前保留当前服务")
	}
	if err := LoadLocalMaterials(ctx, snap.Release, func(message string) { Log(d, "images", message) }); err != nil {
		finish(d, "failed", err.Error())
		return
	}
	actual, health, updated, err := RunDeployment(ctx, &snap, dir, "", func(stage, message string) { Log(d, stage, message) }, func() error {
		encoded, e := json.Marshal(snap)
		if e != nil {
			return e
		}
		return database.DB.Model(d).Update("snapshot", string(encoded)).Error
	})
	b, _ := json.Marshal(snap)
	d.Snapshot = string(b)
	if saveErr := database.DB.Model(d).Update("snapshot", d.Snapshot).Error; saveErr != nil {
		finish(d, "needs_attention", "执行状态保存失败，请核对实际服务："+saveErr.Error())
		return
	}

	if err != nil {
		Log(d, "failed", err.Error())
		if updated {
			RuntimeFailure(d)
		}
		if updated && !IsCommand(snap.Release) && snap.Environment.Stateless && snap.Environment.AutoRollback && d.PreviousReleaseID > 0 {
			recovery(parent, d)
		}
		if parent.Err() != nil {
			return
		} // Leave running for startup reconciliation after shutdown.
		finish(d, "failed", err.Error())
		return
	}
	snap.ScriptsCompleted = true
	b, _ = json.Marshal(snap)
	if err := database.DB.Model(d).Update("snapshot", string(b)).Error; err != nil {
		finish(d, "needs_attention", "服务已更新但脚本完成状态保存失败，请核对发布结果")
		return
	}
	success(d, actual, health)
}
func success(d *models.Deployment, actual, health string) {
	var snapshot Snapshot
	json.Unmarshal([]byte(d.Snapshot), &snapshot)
	runtime := "running"
	if IsCommand(snapshot.Release) && len(snapshot.Containers) == 0 {
		runtime = "unknown"
	}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var state models.TargetState
		tx.Where("environment_id = ? AND target_id = ?", d.EnvironmentID, d.TargetID).First(&state)
		state.EnvironmentID = d.EnvironmentID
		state.TargetID = d.TargetID
		state.CurrentReleaseID = d.ReleaseID
		state.Health = health
		state.Containers = snapshot.Containers
		state.RuntimeState = runtime
		if err := tx.Save(&state).Error; err != nil {
			return err
		}
		var remaining int64
		tx.Model(&models.Deployment{}).Where("batch = ? AND sequence > ?", d.Batch, d.Sequence).Count(&remaining)
		if remaining == 0 {
			if err := tx.Model(&models.Environment{}).Where("id = ?", d.EnvironmentID).Updates(map[string]any{"current_release_id": d.ReleaseID, "runtime_state": runtime, "health": health}).Error; err != nil {
				return err
			}
		} else {
			tx.Model(&models.Environment{}).Where("id = ?", d.EnvironmentID).Update("runtime_state", "partial")
		}

		var snap Snapshot
		if json.Unmarshal([]byte(d.Snapshot), &snap) == nil && snap.Environment.RestoreConfiguration {
			scripts, _ := json.Marshal(snap.Scripts)
			if err := tx.Model(&models.Environment{}).Where("id = ?", d.EnvironmentID).Updates(map[string]any{"parameters": snap.Environment.Parameters, "script_mode": "override", "scripts": string(scripts)}).Error; err != nil {
				return err
			}
		}
		return tx.Model(d).Update("actual_images", actual).Error
	})
	if err != nil {
		finish(d, "needs_attention", "服务已更新但元数据保存失败: "+err.Error())
		return
	}
	result := "发布成功，健康检查通过"
	if health == "unverified" {
		result = "已启动，未验证健康"
	}
	if IsCommand(snapshot.Release) {
		result = "部署命令执行完成，未验证服务状态"
		if len(snapshot.Containers) > 0 {
			result = "关联容器已启动，未验证健康"
		}
		if health == "healthy" {
			result = "部署命令执行完成，部署检查通过"
		}
	}
	finish(d, "success", result)
}
func recovery(parent context.Context, d *models.Deployment) {
	database.DB.Model(d).Update("recovery_status", "running")
	Log(d, "recovery", "恢复上次成功版本和配置")
	var previous models.Deployment
	err := database.DB.Where("environment_id = ? AND target_id = ? AND release_id = ? AND status = ?", d.EnvironmentID, d.TargetID, d.PreviousReleaseID, "success").Order("id DESC").First(&previous).Error
	var s Snapshot
	if err == nil {
		err = json.Unmarshal([]byte(previous.Snapshot), &s)
	}
	if err == nil && IsCommand(s.Release) {
		err = fmt.Errorf("上次为命令部署，无法自动恢复，请人工核对")
	}
	ctx, cancel := context.WithTimeout(parent, Timeout())
	defer cancel()
	if err == nil {
		dir := filepath.Join(JobDir(d.ID), "recovery")
		err = os.MkdirAll(dir, 0700)
		if err == nil {
			var images []Image
			cached := s.Release
			cached.Source = "bundle"
			cached.RegistryID = 0
			im, _ := json.Marshal(s.Images)
			cached.Images = string(im)
			images, err = Prepare(ctx, &cached, func(m string) { Log(d, "recovery", m) })
			if err == nil {
				var pErr error
				p, pErr := Render(ctx, s.Release, s.Environment, dir, images)
				err = pErr
				if err == nil {
					err = Up(ctx, p, dir, func(m string) { Log(d, "recovery", m) })
					if err == nil {
						_, _, err = Actual(ctx, p)
					}
				}
			}
		}
	}
	state := "success"
	if err != nil {
		state = "failed"
		Log(d, "recovery", err.Error())
	}
	database.DB.Model(d).Update("recovery_status", state)
	if state == "success" {
		Recovered(d, s.Release.ID)
	}
}
func reconcile(ctx context.Context) {
	var running []models.Deployment
	database.DB.Where("status = ?", "running").Find(&running)
	for i := range running {
		d := &running[i]
		var s Snapshot
		if json.Unmarshal([]byte(d.Snapshot), &s) != nil {
			finish(d, "needs_attention", "快照损坏，需核对实际服务")
			continue
		}
		if s.Environment.TargetID > 1 {
			continue
		}
		if (HasScripts(s.Scripts) || RequiresExecutionAgent(s.Release)) && !s.ScriptsCompleted {
			finish(d, "needs_attention", "脚本执行被中断；不会自动重跑，请核对脚本和服务状态后重新发布")
			continue
		}
		if IsCommand(s.Release) {
			success(d, "", "unchecked")
			continue
		}
		if len(s.Images) == 0 {
			finish(d, "needs_attention", "进程中断；需核对目标状态后重新发布")
			continue
		}
		p, err := Render(ctx, s.Release, s.Environment, JobDir(d.ID), s.Images)
		if err == nil {
			var a, h string
			a, h, err = Actual(ctx, p)
			if err == nil {
				success(d, a, h)
				continue
			}
		}
		finish(d, "needs_attention", "执行被中断，未自动重复替换；请检查实际服务状态")
	}
}
func processEvents(ctx context.Context) {
	var events []models.ImageEvent
	database.DB.Where("processed = ?", false).Order("id").Limit(50).Find(&events)
	for _, e := range events {
		if !e.Committed {
			if builtin.Engine == nil {
				continue
			}
			token, err := builtin.InternalToken([]string{e.Repository})
			if err != nil {
				continue
			}
			req := httptest.NewRequest(http.MethodHead, "/v2/"+e.Repository+"/manifests/"+e.Reference, nil).WithContext(ctx)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Accept", AcceptManifest)
			w := httptest.NewRecorder()
			builtin.Serve(w, req)
			if w.Code != 200 || w.Header().Get("Docker-Content-Digest") != e.Digest {
				continue
			}
			database.DB.Model(&e).Update("committed", true)
		}
		// Buildx pushes child manifests by digest; only the final named tag is deployable.
		if e.Imported || strings.HasPrefix(e.Reference, "sha256:") {
			database.DB.Model(&e).Update("processed", true)
			continue
		}
		var envs []models.Environment
		database.DB.Where("registry_id = ? AND repository = ?", e.RegistryID, e.Repository).Find(&envs)
		complete := true
		for _, env := range envs {
			matched, _ := filepath.Match(env.TagPattern, e.Reference)
			if env.TagPattern != "" && !matched {
				continue
			}
			unchanged, equalityErr := SameAutomaticImage(env, e)
			if equalityErr != nil {
				complete = false
				continue
			}
			if unchanged {
				continue
			}
			base, err := AutomationBase(env)
			if err != nil {
				continue
			}
			key := fmt.Sprintf("image-event-%d-env-%d", e.ID, env.ID)
			var r models.Release
			if database.DB.Where("application_id = ? AND version = ?", env.ApplicationID, key).First(&r).Error != nil {
				r, err = automaticRelease(ctx, base, env, e)
				if err != nil {
					complete = false
					continue
				}
				r.CreatedAt = time.Time{}
				var app models.Application
				if database.DB.First(&app, env.ApplicationID).Error != nil {
					complete = false
					continue
				}
				r.Scripts = app.Scripts
				if err := SaveRelease(&r); err != nil {
					complete = false
					continue
				}
			}
			if env.AutoDeploy && eligible(env, e.UserID) {
				if _, err := Enqueue(env, r, e.UserID, key, true, e.ID); err != nil {
					complete = false
				}
			}
		}
		if complete {
			database.DB.Model(&e).Update("processed", true)
		}
	}
}
func stringJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func PublisherAllowed(e models.Environment, uid uint) bool      { return eligible(e, uid) }
func RemoteSuccess(d *models.Deployment, actual, health string) { success(d, actual, health) }
func RemoteFailure(d *models.Deployment, status, result string) { finish(d, status, result) }

func SaveRelease(r *models.Release) error {
	builtin.ReferenceGate.RLock()
	defer builtin.ReferenceGate.RUnlock()
	var application models.Application
	if err := database.DB.First(&application, r.ApplicationID).Error; err != nil {
		return fmt.Errorf("应用已删除或不可用，请刷新后重试：%w", err)
	}
	if r.BundleID != 0 {
		var bundle models.Bundle
		if err := database.DB.First(&bundle, r.BundleID).Error; err != nil {
			return fmt.Errorf("发布包已删除或不可用，请刷新后重试：%w", err)
		}
	}
	return database.DB.Create(r).Error
}
func refreshRetention(environmentID uint) {
	var env models.Environment
	if database.DB.First(&env, environmentID).Error != nil {
		return
	}
	var envs []uint
	database.DB.Model(&models.Environment{}).Where("application_id = ?", env.ApplicationID).Pluck("id", &envs)
	var history []models.Deployment
	database.DB.Where("environment_id IN ? AND status = ?", envs, "success").Where("release_id IN (?)", database.DB.Model(&models.Release{}).Select("id")).Order("id DESC").Find(&history)
	keep := map[uint]bool{}
	successful := map[uint]bool{}
	perTarget := map[string]map[uint]bool{}
	for _, d := range history {
		k := fmt.Sprintf("%d:%d", d.EnvironmentID, d.TargetID)
		if perTarget[k] == nil {
			perTarget[k] = map[uint]bool{}
		}
		if len(perTarget[k]) < 3 || perTarget[k][d.ReleaseID] {
			perTarget[k][d.ReleaseID] = true
			keep[d.ReleaseID] = true
		}
		successful[d.ReleaseID] = true
	}
	var states []models.TargetState
	database.DB.Where("environment_id IN ?", envs).Find(&states)
	for _, s := range states {
		keep[s.CurrentReleaseID] = true
	}
	var active []models.Deployment
	database.DB.Where("environment_id IN ? AND status IN ?", envs, []string{"queued", "running", "blocked"}).Find(&active)
	for _, d := range active {
		keep[d.ReleaseID] = true
		keep[d.PreviousReleaseID] = true
	}
	for id := range successful {
		database.DB.Model(&models.Release{}).Where("id = ?", id).Update("retained", keep[id])
	}
}

func RuntimeFailure(d *models.Deployment) {
	var snapshot Snapshot
	if json.Unmarshal([]byte(d.Snapshot), &snapshot) == nil && snapshot.Containers != nil {
		var state models.TargetState
		database.DB.Where("environment_id = ? AND target_id = ?", d.EnvironmentID, d.TargetID).First(&state)
		state.EnvironmentID = d.EnvironmentID
		state.TargetID = d.TargetID
		state.Containers = snapshot.Containers
		database.DB.Save(&state)
	}

	database.DB.Model(&models.TargetState{}).Where("environment_id = ? AND target_id = ?", d.EnvironmentID, d.TargetID).Updates(map[string]any{"runtime_state": "degraded", "health": "unknown"})
	database.DB.Model(&models.Environment{}).Where("id = ?", d.EnvironmentID).Updates(map[string]any{"runtime_state": "partial", "health": "unknown"})
}
func Recovered(d *models.Deployment, releaseID uint) {
	database.DB.Model(&models.TargetState{}).Where("environment_id = ? AND target_id = ?", d.EnvironmentID, d.TargetID).Updates(map[string]any{"current_release_id": releaseID, "runtime_state": "running", "health": "unknown"})
	database.DB.Model(&models.Environment{}).Where("id = ?", d.EnvironmentID).Updates(map[string]any{"runtime_state": "running", "health": "unknown"})
}

func reconcileRuntime(parent context.Context) {
	var states []models.TargetState
	database.DB.Where("target_id = ?", 1).Find(&states)
	for _, state := range states {
		var active int64
		database.DB.Model(&models.Deployment{}).Where("environment_id = ? AND target_id = ? AND status = ?", state.EnvironmentID, 1, "running").Count(&active)
		if active > 0 {
			continue
		}
		var last models.Deployment
		if database.DB.Where("environment_id = ? AND target_id = ? AND status NOT IN ?", state.EnvironmentID, 1, []string{"queued", "running", "blocked"}).Order("id DESC").First(&last).Error != nil {
			continue
		}
		var s Snapshot
		if json.Unmarshal([]byte(last.Snapshot), &s) != nil {
			continue
		}
		var env models.Environment
		if database.DB.First(&env, state.EnvironmentID).Error != nil {
			continue
		}
		if IsCommand(s.Release) {
			s.Environment.ContainerNames = env.ContainerNames
			s.Environment.ReplaceContainers = env.ReplaceContainers
			s.Environment.Parameters = env.Parameters
		}
		ctx, cancel := context.WithTimeout(parent, 10*time.Second)
		rows, observeErr := RuntimeContainers(ctx, s)
		health, runtime := "unknown", "degraded"
		if IsCommand(s.Release) {
			if len(env.ContainerNames) == 0 {
				runtime = "unknown"
			} else if observeErr == nil {
				health, runtime = "unverified", "running"
				for _, row := range rows {
					if row.State != "running" {
						runtime = "stopped"
						health = "unknown"
					}
					if row.Health == "unhealthy" {
						health = "unhealthy"
						runtime = "degraded"
					}
				}
			}
		} else {
			p, err := Render(ctx, s.Release, s.Environment, JobDir(last.ID), s.Images)
			if err == nil {
				_, health, err = Actual(ctx, p)
				if err == nil {
					runtime = "running"
				}
			}
		}
		cancel()
		updates := map[string]any{"health": health, "runtime_state": runtime}
		if observeErr == nil {
			encoded, _ := json.Marshal(rows)
			updates["containers"] = string(encoded)
		}
		database.DB.Model(&state).Updates(updates)
		if env.TargetIDs == "" || env.TargetIDs == "[]" {
			database.DB.Model(&env).Updates(map[string]any{"health": health, "runtime_state": runtime})
		}

	}
}

func RetentionProtection(r models.Release) string {
	var n int64
	database.DB.Model(&models.Environment{}).Where("current_release_id = ?", r.ID).Count(&n)
	if n > 0 {
		return "当前版本受保护"
	}
	database.DB.Model(&models.TargetState{}).Where("current_release_id = ?", r.ID).Count(&n)
	if n > 0 {
		return "当前节点版本受保护"
	}
	database.DB.Model(&models.Deployment{}).Where("(release_id = ? OR previous_release_id = ?) AND status IN ?", r.ID, r.ID, []string{"queued", "running", "blocked"}).Count(&n)
	if n > 0 {
		return "执行与恢复材料受保护"
	}
	var history []models.Deployment
	database.DB.Where("environment_id IN (?) AND status = ?", database.DB.Model(&models.Environment{}).Select("id").Where("application_id = ?", r.ApplicationID), "success").Where("release_id IN (?)", database.DB.Model(&models.Release{}).Select("id")).Order("id DESC").Find(&history)
	seen := map[string]map[uint]bool{}
	for _, d := range history {
		k := fmt.Sprintf("%d:%d", d.EnvironmentID, d.TargetID)
		if seen[k] == nil {
			seen[k] = map[uint]bool{}
		}
		if len(seen[k]) >= 3 && !seen[k][d.ReleaseID] {
			continue
		}
		seen[k][d.ReleaseID] = true
		if d.ReleaseID == r.ID {
			return "最近成功版本受保护"
		}
	}
	return ""
}
