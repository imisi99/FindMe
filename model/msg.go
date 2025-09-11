package model

import "gorm.io/gorm"


type UserMessage struct {
	gorm.Model
	FromID		uint 	
	ToID		uint	
	Message 	string		

	// Relations:
	FromUser 	User	`gorm:"foreignKey:FromID;constraint:OnUpdated:CASCADE,OnDelete:CASCADE"`
	ToUser		User	`gorm:"foreignKey:ToID;constraint:OnUpdated:CASCADE,OnDelete:CASCADE"`
}
