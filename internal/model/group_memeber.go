package model

import (
	"gorm.io/gorm"
)

type GroupMember struct {
	gorm.Model
	UserId     string                `json:"userId" gorm:"index;comment:'用户ID'"`
	GroupId   string                 `json:"groupId" gorm:"index;comment:'群组ID'"`
	Nickname  string                `json:"nickname" gorm:"type:varchar(350);comment:'昵称"`
	Mute      uint8                `json:"mute" gorm:"comment:'是否禁言'"`
}