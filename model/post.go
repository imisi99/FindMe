package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Project struct {
	GormModel
	Description  string `gorm:"not null"`
	Views        uint   `gorm:"not null"`
	Availability bool
	GitProject   bool
	GitLink      string
	ChatID       *string
	UserID       string `gorm:"not null"`

	// Relations:
	User         *User         `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Tags         []*Skill      `gorm:"many2many:post_skills"`
	Applications []*ProjectReq `gorm:"foreignKey:ProjectID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Chat         *Chat         `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type ProjectReq struct {
	GormModel
	Status    string `gorm:"not null;default:'pending'"`
	Message   string `gorm:"default:'Hey I can work on this'"`
	ProjectID string `gorm:"not null"`
	FromID    string `gorm:"not null"`
	ToID      string `gorm:"not null"`

	// Relations:
	Project  *Project `gorm:"foreignKey:ProjectID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	FromUser *User    `gorm:"foreignKey:FromID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ToUser   *User    `gorm:"foreignKey:ToID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type ProjectSkill struct {
	ProjectID string `gorm:"primaryKey"`
	SkillID   string `gorm:"primaryKey"`

	// Relations:
	Project *Project `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (p *Project) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return err
}

func (p *ProjectReq) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}
