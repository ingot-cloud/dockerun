package handler

import (
	"dockerun/internal/builtin"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type environmentSummary struct {
	models.Environment
	DeletionBlockedReason string `json:"deletion_blocked_reason" gorm:"-"`
}

// environmentDeletionReasons 批量查询任务状态，列表和删除接口使用同一保护规则。
func environmentDeletionReasons(db *gorm.DB, ids []uint) (map[uint]string, error) {
	reasons := make(map[uint]string, len(ids))
	if len(ids) == 0 {
		return reasons, nil
	}
	var active []uint
	if err := db.Model(&models.Deployment{}).
		Where("environment_id IN ? AND (status IN ? OR recovery_status = ?)", ids, []string{"queued", "running", "blocked"}, "running").
		Distinct().Pluck("environment_id", &active).Error; err != nil {
		return nil, err
	}
	for _, id := range active {
		reasons[id] = "环境存在排队、执行或恢复中的部署任务，结束后才可删除"
	}
	return reasons, nil
}

// DeleteEnvironment 删除环境及节点状态，不删除部署历史、版本或实际运行资源。
func DeleteEnvironment(c *gin.Context) {
	applicationID, err := deleteID(c.Param("id"))
	if err != nil {
		workspaceDeleteFailed(c, err)
		return
	}
	id, err := deleteID(c.Param("environment_id"))
	if err != nil {
		workspaceDeleteFailed(c, err)
		return
	}
	// 与排队发布、保存环境及任务完成串行，防止检查后出现新的执行任务。
	builtin.ReferenceGate.Lock()
	defer builtin.ReferenceGate.Unlock()
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var environment models.Environment
		if err := tx.Select("id").Where("id = ? AND application_id = ?", id, applicationID).First(&environment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return workspaceDeleteError{http.StatusNotFound, "部署环境不存在或不属于此应用"}
			}
			return err
		}
		reasons, err := environmentDeletionReasons(tx, []uint{id})
		if err != nil {
			return err
		}
		if reason := reasons[id]; reason != "" {
			return workspaceDeleteError{http.StatusConflict, reason}
		}
		if err := tx.Where("environment_id = ?", id).Delete(&models.TargetState{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&environment).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditEvent{UserID: c.GetUint("user_id"), Action: "environment.delete", Resource: fmt.Sprintf("%d/%d", applicationID, id)}).Error
	})
	if err != nil {
		workspaceDeleteFailed(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "部署环境已删除，部署历史及已部署资源会保留"})
}
