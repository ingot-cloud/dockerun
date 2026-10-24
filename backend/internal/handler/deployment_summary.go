package handler

import (
	"dockerun/internal/database"
	"dockerun/internal/models"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// deploymentSummary 在查询时补充可读名称，不改变任务执行时保存的快照。
type deploymentSummary struct {
	models.Deployment
	Name             string `json:"name" gorm:"-"`
	ApplicationID    uint   `json:"application_id"`
	ApplicationName  string `json:"application_name"`
	EnvironmentName  string `json:"environment_name"`
	ReleaseVersion   string `json:"release_version"`
	TargetName       string `json:"target_name"`
	Username         string `json:"username"`
	LegacyScriptName string `json:"legacy_script_name"`
}

func deploymentSummaries() *gorm.DB {
	return database.DB.Model(&models.Deployment{}).
		Select(`deployments.*, applications.id AS application_id, applications.name AS application_name,
			environments.name AS environment_name, releases.version AS release_version,
			targets.name AS target_name, users.username AS username, deploy_scripts.name AS legacy_script_name`).
		Joins("LEFT JOIN environments ON environments.id = deployments.environment_id").
		Joins("LEFT JOIN releases ON releases.id = deployments.release_id").
		Joins("LEFT JOIN applications ON applications.id = COALESCE(NULLIF(environments.application_id, 0), releases.application_id)").
		Joins("LEFT JOIN targets ON targets.id = deployments.target_id").
		Joins("LEFT JOIN users ON users.id = deployments.user_id").
		Joins("LEFT JOIN deploy_scripts ON deploy_scripts.id = deployments.legacy_script_id")
}

func describeDeployment(d *deploymentSummary) {
	if d.EnvironmentName == "" || d.ReleaseVersion == "" || d.ApplicationID == 0 {
		var snapshot struct {
			Environment struct{ Name string }
			Release     struct {
				Version       string
				ApplicationID uint `json:"application_id"`
			}
		}
		if json.Unmarshal([]byte(d.Snapshot), &snapshot) == nil {
			if d.EnvironmentName == "" {
				d.EnvironmentName = snapshot.Environment.Name
			}
			if d.ReleaseVersion == "" {
				d.ReleaseVersion = snapshot.Release.Version
			}
			if d.ApplicationID == 0 {
				d.ApplicationID = snapshot.Release.ApplicationID
			}
		}
	}
	if d.TargetName == "" && d.TargetID == 1 {
		d.TargetName = "本机"
	}
	if d.LegacyScriptName != "" {
		d.Name = "执行脚本 · " + d.LegacyScriptName
		return
	}
	var parts []string
	for _, value := range []string{d.ApplicationName, d.EnvironmentName, d.ReleaseVersion} {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, value)
		}
	}
	d.Name = strings.Join(parts, " · ")
	if d.Name == "" {
		d.Name = fmt.Sprintf("部署任务 #%d", d.ID)
	}
}
