package model

import (
	"time"
)

type Account struct {
	UUID         string    `xorm:"varchar(36) pk" json:"uuid"`
	Email        string    `xorm:"varchar(255) not null unique" json:"email"`
	UserName     string    `xorm:"varchar(64) not null unique" json:"user_name"`
	Password     string    `xorm:"varchar(255) not null" json:"-"`
	Avatar       string    `xorm:"text" json:"avatar,omitempty"`
	Role         string    `xorm:"varchar(20)" json:"role"`
	Status       uint8     `xorm:"tinyint default 0" json:"status"`

	// 自定义时间戳：需要手动赋值
	LastLoginAt  time.Time `xorm:"null" json:"last_login_at"`
	LastLogoutAt time.Time `xorm:"null" json:"last_logout_at"`

	// XORM 自动管理：插入/更新时自动生效
	CreatedAt    time.Time `xorm:"created" json:"created_at"`
	UpdatedAt    time.Time `xorm:"updated" json:"updated_at"`
	DeletedAt    time.Time `xorm:"deleted" json:"deleted_at,omitempty"`

	Version      int       `xorm:"version" json:"-"` // 乐观锁
}