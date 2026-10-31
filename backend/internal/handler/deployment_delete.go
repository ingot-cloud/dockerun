package handler

import (
	"dockerun/internal/builtin"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var completedDeploymentStatuses = []string{"success", "failed", "needs_attention", "cancelled", "paused"}

// DeleteDeployment 删除历史和日志，保留运行检查及回滚依赖的执行快照和配置文件。
func DeleteDeployment(c *gin.Context) {
	id, err := deleteID(c.Param("id"))
	if err != nil {
		workspaceDeleteFailed(c, err)
		return
	}
	// 与任务完成及最后一条日志串行，避免删除后再次写入日志。
	builtin.ReferenceGate.Lock()
	defer builtin.ReferenceGate.Unlock()
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var task models.Deployment
		if err := tx.Select("id").Where("history_deleted_at IS NULL").First(&task, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return workspaceDeleteError{http.StatusNotFound, "部署记录不存在或已删除"}
			}
			return err
		}
		result := tx.Model(&models.Deployment{}).
			Where("id = ? AND history_deleted_at IS NULL AND status IN ? AND (recovery_status IS NULL OR recovery_status <> ?)", id, completedDeploymentStatuses, "running").
			Update("history_deleted_at", time.Now())
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return workspaceDeleteError{http.StatusConflict, "排队、执行或恢复中的任务不能删除，请等待任务结束并刷新后重试"}
		}
		if err := tx.Where("deployment_id = ?", id).Delete(&models.TaskLog{}).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditEvent{UserID: c.GetUint("user_id"), Action: "deployment.delete", Resource: fmt.Sprint(id)}).Error
	})
	if err != nil {
		workspaceDeleteFailed(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "部署记录和日志已删除，已部署资源会保留"})
}
