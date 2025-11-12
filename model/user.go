package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	GormModel
	FullName     string  `gorm:"column:fullname;not null"`
	UserName     string  `gorm:"column:username;unique;not null;index"`
	Email        string  `gorm:"unique;not null"`
	GitUserName  *string `gorm:"column:gitusername;uniqueIndex"`
	GitID        *int64  `gorm:"column:gitid;uniqueIndex"`
	Password     string
	Bio          string
	GitUser      bool `gorm:"column:gituser"`
	Availability bool

	// Relations:
	Skills       []*Skill     `gorm:"many2many:user_skills"`
	Posts        []*Post      `gorm:"foreignKey:UserID"`
	SavedPosts   []*Post      `gorm:"many2many:user_saved_posts"`
	Friends      []*User      `gorm:"many2many:user_friends"`
	FriendReq    []*FriendReq `gorm:"foreignKey:UserID"`
	RecFriendReq []*FriendReq `gorm:"foreignKey:FriendID"`
	Chats        []*Chat      `gorm:"many2many:chat_users"`
	SentPostReq  []*PostReq   `gorm:"foreignKey:FromID"`
	RecPostReq   []*PostReq   `gorm:"foreignKey:ToID"`
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

type UserSavedPost struct {
	UserID string `gorm:"primaryKey"`
	PostID string `gorm:"primaryKey"`

	// Relations:
	User *User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Post *Post `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type FriendReq struct {
	GormModel
	UserFriend
	Status  string `gorm:"not null;default:'pending'"`
	Message string `gorm:"default:'Hey let's be friends'"`
}

const (
	StatusAccepted = "accepted"
	StatusRejected = "rejected"
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
	if err := tx.Model(&Post{}).Where("user_id = ?", u.ID).Delete(&Post{}).Error; err != nil {
		return err
	}
	return nil
}

type GormModel struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
