package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// StringMap 用于存储 map[string]string 类型的数据
type StringMap map[string]string

// Scan 实现 sql.Scanner 接口
func (m *StringMap) Scan(value interface{}) error {
	if value == nil {
		*m = make(StringMap)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		*m = make(StringMap)
		return nil
	}
	return json.Unmarshal(bytes, m)
}

// Value 实现 driver.Valuer 接口
func (m StringMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

type DeployScript struct {
	ID            uint               `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	DeletedAt     gorm.DeletedAt     `gorm:"index" json:"-"`
	Name          string             `gorm:"size:200;not null" json:"name"`
	Description   string             `gorm:"size:1000" json:"description"`
	ScriptType    string             `gorm:"size:50;not null;default:'shell'" json:"script_type"`     // shell, docker-compose
	MainScript    string             `gorm:"type:text" json:"main_script"`                            // 主执行脚本
	WorkDir       string             `gorm:"size:500;default:'/ingot-data/dockerun'" json:"work_dir"` // 工作目录
	EnvVars       StringMap          `gorm:"type:json" json:"env_vars"`                               // 环境变量
	Files         []DeployScriptFile `gorm:"-" json:"files"`                                          // 关联文件，应用层处理
	ContainerID   string             `gorm:"size:100" json:"container_id"`                            // 关联的容器ID
	ContainerName string             `gorm:"-" json:"container_name"`                                 // 容器名称（不存储在数据库）
	Status        string             `gorm:"size:50;default:'idle'" json:"status"`                    // idle, running, stopped, failed
	UserID        uint               `json:"user_id"`
}

type DeployScriptFile struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	DeployScriptID uint           `gorm:"not null;index" json:"deploy_script_id"` // 应用层处理关联，不使用外键
	FileName       string         `gorm:"size:255;not null" json:"file_name"`     // 文件名
	FilePath       string         `gorm:"size:500;not null" json:"file_path"`     // 相对路径（相对于工作目录）
	FileType       string         `gorm:"size:50;not null" json:"file_type"`      // compose, config, script, other
	Content        string         `gorm:"type:text;not null" json:"content"`      // 文件内容
	Description    string         `gorm:"size:1000" json:"description"`           // 文件说明
}
