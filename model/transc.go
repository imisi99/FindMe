package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Subscriptions struct {
	GormModel
	UserID        string
	TransactionID string
	Status        string
	PlanName      string
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
	PaystackChargeSuccess        = "charge.success"
	PaystackInvoiceUpdate        = "invoice.update"
	PaystackSubscriptionCreate   = "subscription.create"
	PaystackSubscriptionNotRenew = "subscription.not_renew"
)

func (s *Subscriptions) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}

	return err
}

func (t *Transactions) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}

	return err
}
