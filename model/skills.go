package model

import (
	"strings"

	"gorm.io/gorm"
)

type Skill struct {
	gorm.Model
	Name string `gorm:"unique;not null"`
}

func (s *Skill) BeforeCreate(tx *gorm.DB) (err error) {
	s.Name = strings.ToLower(s.Name)
	return err
}
