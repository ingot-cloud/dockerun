package backup

import (
	"context"
	"dockerun/internal/builtin"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

var cancelWorker context.CancelFunc
var workerWG sync.WaitGroup
var jobKinds = []string{"backup", "registry_gc"}

func Start() {
	ctx, cancel := context.WithCancel(context.Background())
	cancelWorker = cancel
	database.DB.Model(&models.MaintenanceJob{}).Where("status = ? AND kind IN ?", "running", jobKinds).Updates(map[string]any{"status": "failed", "error": "维护任务被进程中断，请重新提交；未完成的备份不可下载"})
	workerWG.Add(1)
	go func() {
		defer workerWG.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runNext(ctx)
			}
		}
	}()
}

func runNext(ctx context.Context) {
	var job models.MaintenanceJob
	if database.DB.Where("status = ? AND kind IN ?", "queued", jobKinds).Order("id").First(&job).Error != nil {
		return
	}
	result := database.DB.Model(&job).Where("status = ?", "queued").Update("status", "running")
	if result.Error != nil || result.RowsAffected != 1 {
		return
	}
	status, file, err := runJob(ctx, job.Kind)
	if errors.Is(err, builtin.ErrMaintenanceBusy) {
		database.DB.Model(&job).Update("status", "queued")
		return
	}
	action := "backup.ready"
	if job.Kind == "registry_gc" {
		action = "registry.gc.complete"
	}
	if err != nil {
		database.DB.Model(&job).Updates(map[string]any{"status": "failed", "error": err.Error()})
		if job.Kind == "registry_gc" {
			action = "registry.gc.failed"
		} else {
			action = "backup.failed"
		}
	} else {
		database.DB.Model(&job).Updates(map[string]any{"status": status, "file": file, "error": ""})
	}
	identity.Audit(job.UserID, action, fmt.Sprintf("job=%d", job.ID))
}

func runJob(ctx context.Context, kind string) (status, file string, err error) {
	if kind == "registry_gc" {
		taskCtx, cancel := context.WithTimeout(ctx, time.Hour)
		defer cancel()
		return "success", "", builtin.GarbageCollect(taskCtx)
	}
	path, err := Create(ctx)
	return "ready", filepath.Base(path), err
}
func Stop() {
	if cancelWorker != nil {
		cancelWorker()
	}
	workerWG.Wait()
}
