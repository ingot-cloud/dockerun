package models

import (
	"time"

	"gorm.io/gorm"
)

type DeployScript struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"not null" json:"name"`
	Description string         `json:"description"`
	ScriptType  string         `gorm:"not null;default:'shell'" json:"script_type"` // shell, docker-compose
	Content     string         `gorm:"type:text;not null" json:"content"`
	ComposeFile string         `gorm:"type:text" json:"compose_file"` // docker-compose文件内容
	ContainerID string         `json:"container_id"`                  // 关联的容器ID
	Status      string         `gorm:"default:'idle'" json:"status"`  // idle, running, stopped, failed
	UserID      uint           `json:"user_id"`
}

