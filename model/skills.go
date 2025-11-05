package model

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Skill struct {
	GormModel
	Name string `gorm:"unique;not null"`
}

func (s *Skill) BeforeCreate(tx *gorm.DB) (err error) {
	s.Name = strings.ToLower(s.Name)
	if s.ID == "" {
		s.ID = uuid.NewString()
	}

	return err
}
