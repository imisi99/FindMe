package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Post struct {
	GormModel
	Description  string `gorm:"not null"`
	Views        uint   `gorm:"not null"`
	Availability bool

	// Relations:
	UserID string   `gorm:"not null"`
	User   User     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Tags   []*Skill `gorm:"many2many:post_skills"`
}

type PostReq struct {
	GormModel
	Status  string `gorm:"not null;defualt:'pending'"`
	Message string `gorm:"default:'Hey I can work on this'"`
	PostID  string `gorm:"not null"`
	FromID  string `gorm:"not null"`
	ToID    string `gorm:"not null"`

	// Relations:
	Post     Post `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	FromUser User `gorm:"foreignKey:FromID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ToUser   User `gorm:"foreignKey:ToID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type PostSkill struct {
	PostID  string `gorm:"primaryKey"`
	SkillID string `gorm:"primaryKey"`

	// Relations:
	Post Post `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (p *Post) BeforeCreate(tx *gorm.DB) (err error) {
	p.ID = uuid.NewString()
	return err
}

func (p *PostReq) BeforeCreate(tx *gorm.DB) (err error) {
	p.ID = uuid.NewString()
	return err
}
