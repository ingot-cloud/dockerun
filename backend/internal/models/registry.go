package models

import "time"

type Registry struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`        // Registry 名称
	URL       string    `json:"url" gorm:"not null"`                      // Registry URL (e.g., https://registry.example.com)
	Username  string    `json:"username"`                                 // 认证用户名
	Password  string    `json:"password"`                                 // 认证密码（加密存储）
	IsDefault bool      `json:"is_default" gorm:"default:false"`          // 是否默认 Registry
	Insecure  bool      `json:"insecure" gorm:"default:false"`            // 是否允许不安全的 HTTPS
	Email     string    `json:"email"`                                    // 邮箱（可选）
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

