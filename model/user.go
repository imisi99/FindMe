package model


import "gorm.io/gorm"


type User struct {
	gorm.Model
	FullName 	string 		`gorm:"column:fullname;not null"`
	UserName 	string 		`gorm:"column:username;unique;not null"`
	Email 		string	 	`gorm:"unique;not null"`
	GitUserName *string		`gorm:"column:gitusername;uniqueIndex"`
	GitID		*int64		`gorm:"column:gitid;uniqueIndex"`
	Password 	string
	Bio 		string
	GitUser       bool		`gorm:"column:gituser"`
	Availability  bool


	// Relations:
	Skills []*Skill 			`gorm:"many2many:user_skills"`
}
