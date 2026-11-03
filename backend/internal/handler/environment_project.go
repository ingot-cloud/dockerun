package handler

import (
	"dockerun/internal/database"
	"dockerun/internal/models"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var composeProjectPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
var generatedProjectPattern = regexp.MustCompile(`^dockerun-a[0-9]+-e[0-9]+$`)

// 包含失败、排队和已隐藏的历史任务；提交后即固定资源归属。
func environmentProjectLocks(ids []uint) (map[uint]bool, error) {
	locked := map[uint]bool{}
	if len(ids) == 0 {
		return locked, nil
	}
	var deployed []uint
	if err := database.DB.Model(&models.Deployment{}).Where("environment_id IN ?", ids).Distinct().Pluck("environment_id", &deployed).Error; err != nil {
		return nil, err
	}
	for _, id := range deployed {
		locked[id] = true
	}
	var states []uint
	if err := database.DB.Model(&models.TargetState{}).Where("environment_id IN ? AND current_release_id > 0", ids).Distinct().Pluck("environment_id", &states).Error; err != nil {
		return nil, err
	}
	for _, id := range states {
		locked[id] = true
	}
	return locked, nil
}

func projectTargets(env models.Environment) (map[uint]bool, error) {
	var ids []uint
	if env.TargetIDs != "" {
		if err := json.Unmarshal([]byte(env.TargetIDs), &ids); err != nil {
			return nil, fmt.Errorf("目标主机组格式无效")
		}
	}
	if len(ids) == 0 {
		ids = []uint{env.TargetID}
	}
	result := map[uint]bool{}
	for _, id := range ids {
		if id == 0 {
			id = 1
		}
		result[id] = true
	}
	return result, nil
}

// 调用方持有 ReferenceGate 写锁，与任务提交及其他环境的命名互斥。
func validateEnvironmentProject(env *models.Environment, previous *models.Environment) error {
	env.ComposeProjectName = strings.TrimSpace(env.ComposeProjectName)
	name := env.ComposeProjectName
	if name != "" && (len(name) > 100 || !composeProjectPattern.MatchString(name)) {
		return fmt.Errorf("Compose 项目名称须以小写字母或数字开头，仅包含小写字母、数字、连字符和下划线，最多 100 个字符")
	}
	if generatedProjectPattern.MatchString(name) && name != release.ProjectName(models.Environment{ID: env.ID, ApplicationID: env.ApplicationID}) {
		return fmt.Errorf("dockerun-a编号-e编号 为系统保留名称，请使用其他 Compose 项目名称")
	}
	if previous != nil && release.ProjectName(*previous) != release.ProjectName(*env) {
		locked := previous.CurrentReleaseID > 0
		if !locked {
			locks, err := environmentProjectLocks([]uint{previous.ID})
			if err != nil {
				return err
			}
			locked = locks[previous.ID]
		}
		if locked {
			return fmt.Errorf("此环境已提交部署任务，Compose 项目名称已固定，不能修改；请保留原名称")
		}
	}
	if name == "" {
		return nil
	}
	targets, err := projectTargets(*env)
	if err != nil {
		return err
	}
	var others []models.Environment
	if err := database.DB.Where("compose_project_name = ? AND id <> ?", name, env.ID).Find(&others).Error; err != nil {
		return err
	}
	for _, other := range others {
		otherTargets, err := projectTargets(other)
		if err != nil {
			return err
		}
		for target := range targets {
			if otherTargets[target] {
				return fmt.Errorf("Compose 项目名称 %s 已被同一目标主机上的环境 %s 使用，请更换名称", name, other.Name)
			}
		}
	}
	return nil
}

func checkExpectedProjectName(c *gin.Context, expected *string, env models.Environment) bool {
	if expected != nil && *expected != release.ProjectName(env) {
		c.JSON(409, gin.H{"error": "Compose 项目名称已变化，请重新预览后发布"})
		return false
	}
	return true
}
