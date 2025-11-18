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
	Name     string
	Messages []*UserMessage `gorm:"foreignKey:ChatID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Users    []*User        `gorm:"many2many:chat_users"`

	Group   bool `gorm:"not null"`
	OwnerID *string
	Owner   *User `gorm:"constraint:OnUpdate;CASCADE,OnDelete:CASCADE"`
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
