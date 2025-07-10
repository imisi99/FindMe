package model


import "gorm.io/gorm"


type User struct {
	gorm.Model
	FullName 		string 		`gorm:"not null"`
	Username 	string 		`gorm:"unique:not null"`
	Email 		string	 	`gorm:"unique:not null"`
	Password 	string 		`gorm:"not null"`
	Bio 		string
	Availability  bool


	// Relations:
	Skills []Skill 			`gorm:"many2many:user_skills"`
}
