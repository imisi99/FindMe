// Package model -> The ERM of the app
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserMessage struct {
	GormModel
	ChatID  string `gorm:"not null"`
	FromID  string `gorm:"not null"`
	Message string `gorm:"not null"`

	// Relations:
	FromUser *User `gorm:"foreignKey:FromID"`
}

type Chat struct {
	GormModel
	Messages []*UserMessage `gorm:"foreignKey:ChatID"`
	Users    []*User        `gorm:"many2many:chat_users"`
}

type ChatUser struct {
	UserID string `gorm:"primaryKey"`
	ChatID string `gorm:"primaryKey"`

	User *User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Chat *Chat `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (c *Chat) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return err
}

func (u *UserMessage) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return err
}
