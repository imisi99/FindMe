package model

import "time"

type Subscriptions struct {
	GormModel
	UserID        string
	TransactionID string
	Status        string
	StartDate     time.Time
	EndDate       time.Time

	User        *User         `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Transaction *Transactions `gorm:"foreignKey:TransactionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type Transactions struct {
	GormModel
	UserID      string
	Amount      int64
	Curency     string `gorm:"default:'NGN'"`
	PaystackRef string `gorm:"column:paystackref;uniqueIndex"`
	Status      string `gorm:"default:'pending'"`
	Channel     string
	PaidAt      *time.Time

	User *User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type Plan struct {
	GormModel
	PayStackRef string `gorm:"column:paystackref;uniqueIndex"`
}

const (
	PaystackChargeSuccess = "charge.success"
	PaystackInvoiceUpdate = "invoice.update"
	PaystackExpiringCards = "subscription.expiring_cards"
	PaystackCancelSub     = "subscription.not_renew"
)
