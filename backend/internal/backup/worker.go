package backup

import (
	"context"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"path/filepath"
	"sync"
	"time"
)

var cancelWorker context.CancelFunc
var workerWG sync.WaitGroup

func Start() {
	ctx, cancel := context.WithCancel(context.Background())
	cancelWorker = cancel
	database.DB.Model(&models.MaintenanceJob{}).Where("status = ?", "running").Updates(map[string]any{"status": "failed", "error": "备份被进程中断；未完成文件不标记为可用"})
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
				var job models.MaintenanceJob
				if database.DB.Where("status = ? AND kind = ?", "queued", "backup").Order("id").First(&job).Error != nil {
					continue
				}
				res := database.DB.Model(&job).Where("status = ?", "queued").Update("status", "running")
				if res.RowsAffected != 1 {
					continue
				}
				path, err := Create(ctx)
				if err != nil {
					database.DB.Model(&job).Updates(map[string]any{"status": "failed", "error": err.Error()})
				} else {
					database.DB.Model(&job).Updates(map[string]any{"status": "ready", "file": filepath.Base(path)})
					identity.Audit(job.UserID, "backup.ready", filepath.Base(path))
				}
			}
		}
	}()
}
func Stop() {
	if cancelWorker != nil {
		cancelWorker()
	}
	workerWG.Wait()
}
