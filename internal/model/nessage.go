package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/plugin/optimisticlock"
)

type Message struct {
	gorm.Model
	FromUserId  uuid.UUID `json:"fromUserId" gorm:"type:uuid;index;comment:'发送者ID'"`
	ToUserId    uuid.UUID                 `json:"toUserId" gorm:"type:uuid;index;comment:'发送给端的id,可为用户id或者群id'"`
	Content     string                `json:"content" gorm:"type:text;comment:'消息内容'"`
	MessageType  uint8                `json:"messageType" gorm:"comment:'消息类型:1单聊,2群聊'"`
	ContentType  uint8                `json:"contentType" gorm:"comment:'消息内容类型:1文字 2.普通文件 3.图片 4.音频 5.视频 6.语音聊天 7.视频聊天'"`
	Pic         string                `json:"pic" gorm:"type:text;comment:'缩略图"`
	Url         string                `json:"url" gorm:"type:text;comment:'文件或者图片地址'"`
	Version optimisticlock.Version
}