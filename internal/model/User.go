package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/plugin/optimisticlock"
)

type User struct {
	// 使用UUID作为主键，存储为binary(16)格式以提高性能
	UUID uuid.UUID `gorm:"type:uuid;primaryKey;<REDACTED_EMAIL>" json:"uuid,omitempty"`

	// 用户邮箱，唯一索引，不能为空
	Email string `gorm:"type:varchar(255);uniqueIndex;not null" json:"email,omitempty"`

	// 用户名，唯一索引，不能为空
	UserName string `gorm:"type:varchar(100);uniqueIndex;not null" json:"user_name,omitempty"`

	// 密码，不能为空
	Password string `gorm:"type:varchar(255);not null" json:"password,omitempty"`

	// 头像URL
	Avatar string `gorm:"type:text" json:"avatar,omitempty"`
	// 用户角色，默认为普通用户0
	Role uint8 `gorm:"default:0" json:"role"`
	
	LastLoginAt time.Time `gorm:"type:timestamp with time zone;" json:"last_login_at"`

	// GORM自动管理的时间字段
	CreatedAt time.Time `gorm:"type:timestamp with time zone;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp with time zone;" json:"updated_at"`

	// 软删除字段，使用gorm.DeletedAt替代自定义的time.Time
	DeletedAt gorm.DeletedAt `gorm:"type:timestamp with time zone;index" json:"deleted_at"`

	// 乐观锁版本控制
	Version optimisticlock.Version
}
