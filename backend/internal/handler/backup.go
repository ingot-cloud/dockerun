package handler

import (
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/identity"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"github.com/gin-gonic/gin"
	"path/filepath"
)

func CreateBackup(c *gin.Context) {
	job := models.MaintenanceJob{Kind: "backup", Status: "queued", UserID: c.GetUint("user_id")}
	if err := database.DB.Create(&job).Error; err != nil {
		fail(c, err)
		return
	}
	identity.Audit(job.UserID, "backup.create", "queued")
	c.JSON(202, gin.H{"job_id": job.ID, "message": "备份任务已创建，刷新或离开页面后仍会继续；请在下方查看结果。"})
}
func ListBackups(c *gin.Context) {
	var jobs []models.MaintenanceJob
	database.DB.Where("kind = ?", "backup").Order("id DESC").Limit(100).Find(&jobs)
	c.JSON(200, jobs)
}
func DownloadBackup(c *gin.Context) {
	var job models.MaintenanceJob
	if database.DB.First(&job, c.Param("id")).Error != nil || job.Kind != "backup" || job.Status != "ready" {
		c.JSON(404, gin.H{"error": "完整备份尚不可用"})
		return
	}
	path, err := release.SafePath(filepath.Join(config.Load().DataDir, "backups"), job.File)
	if err != nil {
		fail(c, err)
		return
	}
	c.FileAttachment(path, job.File)
}
