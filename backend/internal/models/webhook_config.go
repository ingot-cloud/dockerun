package models

import (
	"time"

	"gorm.io/gorm"
)

// WebhookConfig 存储 Webhook 配置，用于自动部署
type WebhookConfig struct {
	ID                 uint           `json:"id" gorm:"primarykey"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
	Name               string         `json:"name" gorm:"size:200;not null;comment:配置名称"`
	Description        string         `json:"description" gorm:"size:500;comment:配置描述"`
	Enabled            bool           `json:"enabled" gorm:"default:true;comment:是否启用"`
	RegistryID         uint           `json:"registry_id" gorm:"index;comment:关联的 Registry ID"`
	Registry           *Registry      `json:"registry,omitempty" gorm:"-"` // 应用层处理关联，不使用外键
	ImageName          string         `json:"image_name" gorm:"size:500;not null;index;comment:镜像名称（如：namespace/repo）"`
	ImageTag           string         `json:"image_tag" gorm:"size:100;default:latest;comment:镜像标签（如：latest, v1.0.0，留空表示任意标签）"`
	DeployScriptID     uint           `json:"deploy_script_id" gorm:"not null;index;comment:关联的部署脚本 ID"`
	DeployScript       *DeployScript  `json:"deploy_script,omitempty" gorm:"-"` // 应用层处理关联，不使用外键
	RemoveOldImage     bool           `json:"remove_old_image" gorm:"default:true;comment:部署前是否删除旧镜像"`
	RemoveOldContainer bool           `json:"remove_old_container" gorm:"default:true;comment:部署前是否删除旧容器"`
	Secret             string         `json:"secret,omitempty" gorm:"size:200;comment:Webhook 密钥（用于验证）"`
	LastTriggeredAt    *time.Time     `json:"last_triggered_at,omitempty" gorm:"comment:最后触发时间"`
	TriggerCount       int            `json:"trigger_count" gorm:"default:0;comment:触发次数"`
}

// WebhookLog 存储 Webhook 触发日志
type WebhookLog struct {
	ID              uint           `json:"id" gorm:"primarykey"`
	CreatedAt       time.Time      `json:"created_at"`
	WebhookConfigID uint           `json:"webhook_config_id" gorm:"index;not null"`
	WebhookConfig   *WebhookConfig `json:"webhook_config,omitempty" gorm:"-"` // 应用层处理关联，不使用外键
	ImageName       string         `json:"image_name" gorm:"size:500;comment:触发的镜像名称"`
	ImageTag        string         `json:"image_tag" gorm:"size:100;comment:触发的镜像标签"`
	Status          string         `json:"status" gorm:"size:50;default:pending;comment:状态：pending, processing, success, failed"`
	Message         string         `json:"message" gorm:"type:text;comment:执行消息"`
	StartedAt       *time.Time     `json:"started_at,omitempty" gorm:"comment:开始执行时间"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty" gorm:"comment:完成时间"`
	ExecutionTime   int            `json:"execution_time" gorm:"default:0;comment:执行耗时（秒）"`
}

// TableName 指定表名
func (WebhookConfig) TableName() string {
	return "webhook_configs"
}

// TableName 指定表名
func (WebhookLog) TableName() string {
	return "webhook_logs"
}
