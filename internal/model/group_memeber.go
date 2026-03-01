package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GroupMember struct {
	gorm.Model
	UserId     uuid.UUID                `json:"userId" gorm:"type:uuid;index;comment:'用户ID'"`
	GroupId   uuid.UUID                 `json:"groupId" gorm:"type:uuid;index;comment:'群组ID'"`
	Nickname  string                `json:"nickname" gorm:"type:varchar(350);comment:'昵称"`
	Mute      uint8                `json:"mute" gorm:"comment:'是否禁言'"`
}