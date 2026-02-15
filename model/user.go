package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type User struct {
	GormModel
	FullName     string `gorm:"column:fullname;not null"`
	UserName     string `gorm:"column:username;unique;not null;index"`
	Email        string `gorm:"unique;not null"`
	Country      string
	GitUserName  *string `gorm:"column:gitusername;uniqueIndex"`
	GitID        *int64  `gorm:"column:gitid;uniqueIndex"`
	Password     string
	Bio          string
	Interests    pq.StringArray `gorm:"type:text[]"`
	GitUser      bool           `gorm:"column:gituser"`
	Availability bool

	// Sub Details
	FreeTrial       time.Time `gorm:"column:freetrial"`
	TrialReminder   bool      `gorm:"column:trialreminder"`
	NextPaymentDate *time.Time
	LastSub         *time.Time

	// Paystack Details
	PaystackSubCode    *string
	PaystackEmailToken *string
	PaystackCusCode    *string
	PaystackAuthCode   *string

	// Card Details
	Last4    *string
	CardType *string
	ExpMonth *string
	ExpYear  *string

	// Relations:
	Skills         []*Skill         `gorm:"many2many:user_skills"`
	Projects       []*Project       `gorm:"foreignKey:UserID"`
	SavedProjects  []*Project       `gorm:"many2many:user_saved_projects"`
	Friends        []*User          `gorm:"many2many:user_friends"`
	FriendReq      []*FriendReq     `gorm:"foreignKey:UserID"`
	RecFriendReq   []*FriendReq     `gorm:"foreignKey:FriendID"`
	Chats          []*Chat          `gorm:"many2many:chat_users"`
	SentProjectReq []*ProjectReq    `gorm:"foreignKey:FromID"`
	RecProjectReq  []*ProjectReq    `gorm:"foreignKey:ToID"`
	Subscriptions  []*Subscriptions `gorm:"foreignKey:UserID"`
	Transactions   []*Transactions  `gorm:"foreignKey:UserID"`
}

type UserFriend struct {
	UserID   string `gorm:"primaryKey"`
	FriendID string `gorm:"primaryKey"`

	// Relations:
	User   *User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Friend *User `gorm:"foreignKey:FriendID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type UserSkill struct {
	UserID  string `gorm:"primaryKey"`
	SkillID string `gorm:"primaryKey"`

	// Relations:
	User *User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type UserSavedProject struct {
	UserID    string `gorm:"primaryKey"`
	ProjectID string `gorm:"primaryKey"`

	// Relations:
	User    *User    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Project *Project `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type FriendReq struct {
	GormModel
	UserFriend
	Status  string `gorm:"not null;default:'pending'"`
	Message string `gorm:"default:'Hey let's be friends'"`
}

const (
	StatusAccepted  = "accepted"
	StatusRejected  = "rejected"
	StatusAttention = "attention"
	StatusPending   = "pending"
	StatusActive    = "active"
	StatusFailed    = "failed"
	StatusSuccess   = "success"
	StatusExpired   = "expired"
	StatusLogin     = "login"
)

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return err
}

func (f *FriendReq) BeforeCreate(tx *gorm.DB) (err error) {
	if f.ID == "" {
		f.ID = uuid.NewString()
	}

	return err
}

func (u *User) BeforeDelete(tx *gorm.DB) (err error) {
	if err := tx.Model(&Project{}).Where("user_id = ?", u.ID).Delete(&Project{}).Error; err != nil {
		return err
	}
	return nil
}

func IsValidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

type GormModel struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
