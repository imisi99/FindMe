package model

import "gorm.io/gorm"


type Post struct {
	gorm.Model
	Description		string		`gorm:"not null"`
	Views 			uint 		`gorm:"not null"`

	// Relations:
	UserID 			uint		`gorm:"not null"`
	User			User		`gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Tags 			[]*Skill 	`gorm:"many2many:post_skills"`
}


type PostSkill struct {
	PostID 		uint		`gorm:"primaryKey"`
	SkillID 	uint		`gorm:"primaryKey"`


	// Relations:
	Post		Post		`gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}


type PostReq struct {
	gorm.Model
	Status 			string		`gorm:"not null;defualt:'pending'"`
	Message 		string		`gorm:"default:'Hey I can work on this'"`
	PostID  		uint		`gorm:"not null"`
	FromID 			uint		`gorm:"not null"`
	ToID 			uint 		`gorm:"not null"`

	// Relations:
	Post		Post			`gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	FromUser	User			`gorm:"foreignKey:FromID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ToUser 		User			`gorm:"foreignKey:ToID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
