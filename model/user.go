package model

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
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
	Skills       []*Skill       `gorm:"many2many:user_skills"`
	Posts        []*Post        `gorm:"foreignKey:UserID"`
	SavedPosts   []*Post        `gorm:"many2many:user_saved_posts"`
	Friends      []*User        `gorm:"many2many:user_friends"`
	FriendReq    []*FriendReq   `gorm:"foreignKey:UserID"`
	RecFriendReq []*FriendReq   `gorm:"foreignKey:FriendID"`
	Message      []*UserMessage `gorm:"foreignKey:FromID"`
	RecMessage   []*UserMessage `gorm:"foreignKey:ToID"`
	SentPostReq  []*PostReq     `gorm:"foreignKey:FromID"`
	RecPostReq   []*PostReq     `gorm:"foreignKey:ToID"`
}

type UserFriend struct {
	UserID   uint `gorm:"primaryKey"`
	FriendID uint `gorm:"primaryKey"`

	// Relations:
	User   User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Friend User `gorm:"foreignKey:FriendID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type UserSkill struct {
	UserID  uint `gorm:"primaryKey"`
	SkillID uint `gorm:"primaryKey"`

	// Relations:
	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type UserSavedPost struct {
	UserID uint `gorm:"primaryKey"`
	PostID uint `gorm:"primaryKey"`

	// Relations:
	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Post Post `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type FriendReq struct {
	gorm.Model
	UserFriend
	Status  string `gorm:"not null;default:'pending'"`
	Message string `gorm:"default:'Hey let's be friends'"`
}

const (
	StatusAccepted = "accepted"
	StatusRejected = "rejected"
)

func (u *User) BeforeDelete(tx *gorm.DB) (err error) {
	if err := tx.Model(&Post{}).Where("user_id = ?", u.ID).Delete(&Post{}).Error; err != nil {
		return err
	}
	return nil
}
