package handler

import (
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Serialize queue admission with manual read-only transitions in this process.
var maintenanceActionMu sync.Mutex

func CreateBackup(c *gin.Context) {
	maintenanceActionMu.Lock()
	defer maintenanceActionMu.Unlock()
	enqueueMaintenance(c, "backup")
}
func enqueueMaintenance(c *gin.Context, kind string) {
	var job models.MaintenanceJob
	err := database.DB.WithContext(c.Request.Context()).Where("kind = ? AND status IN ?", kind, []string{"queued", "running"}).Order("id").First(&job).Error
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"job_id": job.ID, "reused": true, "message": "已有同类任务排队或执行中，请在任务列表查看进度。"})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(500, gin.H{"error": "无法查询维护任务，请重试"})
		return
	}
	job = models.MaintenanceJob{Kind: kind, Status: "queued", UserID: c.GetUint("user_id")}
	if err := database.DB.WithContext(c.Request.Context()).Create(&job).Error; err != nil {
		c.JSON(500, gin.H{"error": "无法创建维护任务，请重试"})
		return
	}
	action := "backup.create"
	if kind == "registry_gc" {
		action = "registry.gc.create"
	}
	identity.Audit(job.UserID, action, fmt.Sprintf("job=%d", job.ID))
	c.JSON(http.StatusAccepted, gin.H{"job_id": job.ID, "message": "任务已创建，刷新或离开页面后仍会继续，可在任务列表查看结果。"})
}
func ListBackups(c *gin.Context)             { listMaintenanceJobs(c, "backup") }
func ListRegistryMaintenance(c *gin.Context) { listMaintenanceJobs(c, "registry_gc") }
func listMaintenanceJobs(c *gin.Context, kind string) {
	jobs := []models.MaintenanceJob{}
	if err := database.DB.WithContext(c.Request.Context()).Where("kind = ?", kind).Order("id DESC").Limit(100).Find(&jobs).Error; err != nil {
		c.JSON(500, gin.H{"error": "无法加载维护任务，请重试"})
		return
	}
	c.JSON(200, jobs)
}
func DownloadBackup(c *gin.Context) {
	var job models.MaintenanceJob
	if database.DB.WithContext(c.Request.Context()).First(&job, c.Param("id")).Error != nil || job.Kind != "backup" || job.Status != "ready" {
		c.JSON(404, gin.H{"error": "完整备份尚不可用"})
		return
	}
	path, err := release.SafePath(filepath.Join(config.Load().DataDir, "backups"), job.File)
	if err != nil {
		c.JSON(400, gin.H{"error": "备份文件路径无效，请重新创建备份"})
		return
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		c.JSON(404, gin.H{"error": "备份文件已不存在或不可读取，请重新创建备份"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.FileAttachment(path, job.File)
}
