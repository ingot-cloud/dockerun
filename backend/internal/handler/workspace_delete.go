package handler

import (
	"dockerun/internal/apptemplate"
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type workspaceDeleteError struct {
	status  int
	message string
}

func (e workspaceDeleteError) Error() string { return e.message }

func deleteID(raw string) (uint, error) {
	id, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || id == 0 {
		return 0, workspaceDeleteError{http.StatusBadRequest, "无效的资源编号"}
	}
	return uint(id), nil
}

func workspaceDeleteFailed(c *gin.Context, err error) {
	var failure workspaceDeleteError
	if errors.As(err, &failure) {
		c.JSON(failure.status, gin.H{"error": failure.message})
		return
	}
	log.Printf("删除发布工作区资源失败：%v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败，请重试"})
}

func applicationDeployments(tx *gorm.DB, id uint) *gorm.DB {
	environments := tx.Model(&models.Environment{}).Select("id").Where("application_id = ?", id)
	versions := tx.Model(&models.Release{}).Select("id").Where("application_id = ?", id)
	return tx.Model(&models.Deployment{}).Where("(environment_id IN (?) OR release_id IN (?) OR previous_release_id IN (?))", environments, versions, versions)
}

// DeleteApplication 清理应用管理记录，不向 Docker 或目标主机发送资源删除命令。
func DeleteApplication(c *gin.Context) {
	id, err := deleteID(c.Param("id"))
	if err != nil {
		workspaceDeleteFailed(c, err)
		return
	}
	// 与创建版本、排队发布及镜像引用保护串行，避免删除后又写入关联记录。
	builtin.ReferenceGate.Lock()
	defer builtin.ReferenceGate.Unlock()
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var app models.Application
		if err := tx.First(&app, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return workspaceDeleteError{http.StatusNotFound, "应用不存在或已删除"}
			}
			return err
		}
		var active int64
		if err := applicationDeployments(tx, id).Where("(status IN ? OR recovery_status = ?)", []string{"queued", "running", "blocked"}, "running").Count(&active).Error; err != nil {
			return err
		}
		if active > 0 {
			return workspaceDeleteError{http.StatusConflict, "应用存在排队、执行或恢复中的发布任务，请等待任务结束后再删除"}
		}
		if err := tx.Where("deployment_id IN (?)", applicationDeployments(tx, id).Select("id")).Delete(&models.TaskLog{}).Error; err != nil {
			return err
		}
		if err := applicationDeployments(tx, id).Delete(&models.Deployment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("environment_id IN (?)", tx.Model(&models.Environment{}).Select("id").Where("application_id = ?", id)).Delete(&models.TargetState{}).Error; err != nil {
			return err
		}
		if err := tx.Where("application_id = ?", id).Delete(&models.Environment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("application_id = ?", id).Delete(&models.Release{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&app).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditEvent{UserID: c.GetUint("user_id"), Action: "application.delete", Resource: fmt.Sprint(id)}).Error
	})
	if err != nil {
		workspaceDeleteFailed(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "应用及关联管理记录已删除，已部署资源会保留"})
}

func DeleteManagedTemplate(c *gin.Context) {
	raw := c.Param("id")
	for _, template := range apptemplate.Builtins() {
		if raw == template.ID {
			workspaceDeleteFailed(c, workspaceDeleteError{http.StatusForbidden, "内置模板不能删除，可删除自行保存的应用模板"})
			return
		}
	}
	id, err := deleteID(strings.TrimPrefix(raw, "custom-"))
	if err != nil {
		workspaceDeleteFailed(c, err)
		return
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var template models.ApplicationTemplate
		if err := tx.First(&template, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return workspaceDeleteError{http.StatusNotFound, "应用模板不存在或已删除"}
			}
			return err
		}
		if err := tx.Delete(&template).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditEvent{UserID: c.GetUint("user_id"), Action: "application.template.delete", Resource: fmt.Sprintf("custom-%d", id)}).Error
	})
	if err != nil {
		workspaceDeleteFailed(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "应用模板已删除，已创建的应用会保留"})
}

type stagedBundleFiles struct {
	dir   string
	moves [][2]string
}

func (files *stagedBundleFiles) restore() error {
	var failure error
	for i := len(files.moves) - 1; i >= 0; i-- {
		move := files.moves[i]
		failure = errors.Join(failure, os.Rename(move[1], move[0]))
	}
	if failure == nil && files.dir != "" {
		failure = os.RemoveAll(files.dir)
	}
	return failure
}

func (files *stagedBundleFiles) stage(bundle models.Bundle) error {
	cfg := config.Load()
	paths := []string{filepath.Join(cfg.DataDir, "releases", fmt.Sprintf("bundle-%d", bundle.ID))}
	if bundle.Path != "" {
		archive, err := filepath.Abs(bundle.Path)
		if err != nil {
			return err
		}
		root := filepath.Join(cfg.DataDir, "uploads")
		if !uploadID.MatchString(bundle.SHA256) || filepath.Dir(archive) != root || filepath.Base(archive) != bundle.SHA256+".tar.gz" {
			return workspaceDeleteError{http.StatusConflict, "发布包文件不在受管归档目录，无法删除，请检查存储配置"}
		}
		paths = append(paths, archive)
	}
	for _, path := range paths {
		if _, err := os.Lstat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if files.dir == "" {
			dir, err := os.MkdirTemp(cfg.DataDir, ".deleted-bundle-")
			if err != nil {
				return err
			}
			files.dir = dir
		}
		destination := filepath.Join(files.dir, strconv.Itoa(len(files.moves)))
		if err := os.Rename(path, destination); err != nil {
			return err
		}
		files.moves = append(files.moves, [2]string{path, destination})
	}
	return nil
}

func DeleteBundle(c *gin.Context) {
	id, err := deleteID(c.Param("id"))
	if err != nil {
		workspaceDeleteFailed(c, err)
		return
	}
	builtin.ReferenceGate.Lock()
	defer builtin.ReferenceGate.Unlock()
	uploadMu.Lock()
	defer uploadMu.Unlock()
	files := &stagedBundleFiles{}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var bundle models.Bundle
		if err := tx.First(&bundle, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return workspaceDeleteError{http.StatusNotFound, "发布包不存在或已删除"}
			}
			return err
		}
		if bundle.Status != "ready" && bundle.Status != "failed" {
			return workspaceDeleteError{http.StatusConflict, "材料正在导入或状态未知，请等待导入结束后再删除"}
		}
		var references int64
		if err := tx.Model(&models.Release{}).Where("bundle_id = ?", id).Count(&references).Error; err != nil {
			return err
		}
		if references > 0 {
			return workspaceDeleteError{http.StatusConflict, "发布包已被发布版本引用，仍用于部署或恢复；请先删除对应应用后再删除发布包"}
		}
		if err := files.stage(bundle); err != nil {
			return err
		}
		if err := tx.Delete(&bundle).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditEvent{UserID: c.GetUint("user_id"), Action: "bundle.delete", Resource: fmt.Sprint(id)}).Error
	})
	if err != nil {
		if restoreErr := files.restore(); restoreErr != nil {
			err = errors.Join(err, restoreErr)
		}
		workspaceDeleteFailed(c, err)
		return
	}
	if files.dir != "" {
		if err := os.RemoveAll(files.dir); err != nil {
			log.Printf("发布包 %d 已删除，暂存材料清理失败：%v", id, err)
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "发布包、导入记录及归档材料已删除，已加载的镜像会保留"})
}
