package model

import (
	"gorm.io/gorm"
	"gorm.io/plugin/optimisticlock"
)

type UserFriend struct {
	gorm.Model
	UserID   string `gorm:"not null;index"`
	FriendID string `gorm:"not null;index"`
	Status   string `gorm:";not null;default:'1'"` // 例如：(1)pending,(0) accepted, (uuid)who blocked
	Version optimisticlock.Version
}