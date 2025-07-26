package model


import "gorm.io/gorm"


type User struct {
	gorm.Model
	FullName 		string 	`gorm:"column:fullname;not null"`
	UserName 	string 		`gorm:"column:username;unique;not null"`
	Email 		string	 	`gorm:"unique;not null"`
	Password 	string 		`gorm:"not null"`
	GitUserName *string		`gorm:"column:gitusername;unique"`
	Bio 		string
	Availability  bool


	// Relations:
	Skills []*Skill 			`gorm:"many2many:user_skills"`
}
