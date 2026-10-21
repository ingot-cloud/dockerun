package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Username       string         `gorm:"size:100;uniqueIndex;not null" json:"username"`
	Password       string         `gorm:"size:255;not null" json:"-"`
	IsAdmin        bool           `gorm:"default:false" json:"is_admin"`
	Enabled        bool           `gorm:"not null;default:true" json:"enabled"`
	RegistryAccess string         `gorm:"size:20;not null;default:none" json:"registry_access"`
	AuthVersion    uint           `gorm:"not null;default:1" json:"-"`
}

func (u User) CanRegistry(action string) bool {
	if !u.Enabled || u.Password == "" {
		return false
	}
	if u.IsAdmin {
		return true
	}
	return action == "pull" && (u.RegistryAccess == "read_only" || u.RegistryAccess == "read_write") || action == "push" && u.RegistryAccess == "read_write"
}

// HashPassword 加密密码
func (u *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(bytes)
	return nil
}

// CheckPassword 验证密码
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
