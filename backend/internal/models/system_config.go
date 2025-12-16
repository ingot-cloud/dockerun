package models

import "time"

// SystemConfig 系统配置
type SystemConfig struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	DeployWorkDir      string    `json:"deploy_work_dir" gorm:"size:500;not null;default:'/opt/dockerun'"` // 部署脚本工作目录
	MaxScriptExecutors int       `json:"max_script_executors" gorm:"not null;default:5"`                   // 最大并发脚本执行数
	ScriptTimeout      int       `json:"script_timeout" gorm:"not null;default:3600"`                      // 脚本超时时间（秒）
	EnableScriptLog    bool      `json:"enable_script_log" gorm:"not null;default:true"`                   // 是否启用脚本日志
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "system_config"
}

