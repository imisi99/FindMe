package model

import "gorm.io/gorm"


type Post struct {
	gorm.Model
	Description		string		`gorm:"not null"`
	Views 			uint 		`gorm:"not null"`

	// Relations:
	UserID 			uint		`gorm:"not null"`
	User			User		`gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Tags 			[]*Skill 	`gorm:"many2many:post_skills"`
}


type PostSkill struct {
	PostID 		uint		`gorm:"primaryKey"`
	SkillID 	uint		`gorm:"primaryKey"`


	// Relations:
	Post		Post		`gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

