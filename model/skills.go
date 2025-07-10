package model


import "gorm.io/gorm"


type Skill struct {
	gorm.Model
	Name string `gorm:"unique;not null"`
	
}
