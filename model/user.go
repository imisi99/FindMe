package model

import (
	"gorm.io/gorm"
)


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


type UserSkill struct {
	UserID     uint				`gorm:"primaryKey"`
	SkillID    uint				`gorm:"primaryKey"`


	// Relations:
	User 		User			`gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}


func (u *User) BeforeDelete(tx *gorm.DB) (err error) {
	if err := tx.Model(&Post{}).Where("user_id = ?", u.ID).Delete(&Post{}).Error; err != nil {return err}
	return nil
}
