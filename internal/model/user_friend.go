package model

import (
	"gorm.io/gorm"
	"gorm.io/plugin/optimisticlock"
)

type UserFriend struct {
	gorm.Model
	UserID   string `gorm:"not null;index"`
	FriendID string `gorm:"not null;index"`
	Status   uint8 `gorm:";not null;default:1"` // 例如：(1)pending,(0) accepted, (2)blocked
	Version optimisticlock.Version
}