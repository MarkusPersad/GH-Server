package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/plugin/optimisticlock"
)

type Group struct {
	UUID string `gorm:"primaryKey;" json:"uuid,omitempty"`
	OwnerID string `gorm:"not null" json:"owner_id"`
	Name string `gorm:"type:text;uniqueIndex" json:"name"`
	Notice string `gorm:"type:text" json:"notice,omitempty"`
	CreatedAt time.Time `gorm:"type:timestamp with time zone;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp with time zone;" json:"updated_at"`

	// 软删除字段，使用gorm.DeletedAt替代自定义的time.Time
	DeletedAt gorm.DeletedAt `gorm:"type:timestamp with time zone;index" json:"deleted_at"`

	// 乐观锁版本控制
	Version optimisticlock.Version
}