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

type releaseSummary struct {
	models.Release
	DeletionBlockedReason string `json:"deletion_blocked_reason" gorm:"-"`
}

// releaseDeletionReasons 一次查询所有版本的引用，列表提示与删除校验使用同一规则。
func releaseDeletionReasons(db *gorm.DB, ids []uint) (map[uint]string, error) {
	reasons := make(map[uint]string, len(ids))
	if len(ids) == 0 {
		return reasons, nil
	}
	var environments, targets []uint
	if err := db.Model(&models.Environment{}).Where("current_release_id IN ?", ids).Distinct().Pluck("current_release_id", &environments).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&models.TargetState{}).Where("current_release_id IN ?", ids).Distinct().Pluck("current_release_id", &targets).Error; err != nil {
		return nil, err
	}
	for _, id := range targets {
		reasons[id] = "版本正在被目标主机使用，不能删除"
	}
	for _, id := range environments {
		reasons[id] = "版本正在被部署环境使用，不能删除"
	}
	var tasks []models.Deployment
	if err := db.Select("release_id", "previous_release_id").Where("(release_id IN ? OR previous_release_id IN ?) AND (status IN ? OR recovery_status = ?)", ids, ids, []string{"queued", "running", "blocked"}, "running").Find(&tasks).Error; err != nil {
		return nil, err
	}
	for _, task := range tasks {
		for _, id := range []uint{task.ReleaseID, task.PreviousReleaseID} {
			if id != 0 && reasons[id] == "" {
				reasons[id] = "版本正被排队、执行或恢复中的部署任务引用，不能删除"
			}
		}
	}
	return reasons, nil
}

// DeleteRelease 只删除未使用的版本定义，保留部署历史、实际资源和历史执行快照。
func DeleteRelease(c *gin.Context) {
	id, err := deleteID(c.Param("id"))
	if err != nil {
		workspaceDeleteFailed(c, err)
		return
	}
	// 与排队发布、环境配置和任务完成串行，防止检查后新增版本引用。
	builtin.ReferenceGate.Lock()
	defer builtin.ReferenceGate.Unlock()
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var version models.Release
		if err := tx.Select("id").First(&version, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return workspaceDeleteError{http.StatusNotFound, "发布版本不存在或已删除"}
			}
			return err
		}
		reasons, err := releaseDeletionReasons(tx, []uint{id})
		if err != nil {
			return err
		}
		if reason := reasons[id]; reason != "" {
			return workspaceDeleteError{http.StatusConflict, reason}
		}
		if err := tx.Delete(&version).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditEvent{UserID: c.GetUint("user_id"), Action: "release.delete", Resource: fmt.Sprint(id)}).Error
	})
	if err != nil {
		workspaceDeleteFailed(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "发布版本已删除，部署历史和已部署资源会保留"})
}
