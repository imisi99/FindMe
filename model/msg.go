package model

import "gorm.io/gorm"


type UserMessage struct {
	gorm.Model
	From 		uint 	
	To			uint	
	Message 	string		

	// Relations:
	FromUser 	User	`gorm:"foreignKey:From;constraint:OnUpdated:CASCADE,OnDelete:CASCADE"`
	ToUser		User	`gorm:"foreignKey:To;constraint:OnUpdated:CASCADE,OnDelete:CASCADE"`
}

