package core

import (
	"findme/model"

	"gorm.io/gorm"
)

type GormDB struct {
	DB *gorm.DB
}

func (db *GormDB) AddUser(user *model.User) error {
	err := db.DB.Create(user).Error
	return err
}

func (db *GormDB) CheckExistingUser(user *model.User, email, username string) error {
	err := db.DB.Where("username = ? OR email = ?", username, email).First(user).Error
	return err
}

func (db *GormDB) VerifyUser(user *model.User, username, password string) error {
	err := db.DB.Where("username = ?", username).First(user).Error
	return err
}

func (db *GormDB) SaveUser(user *model.User) error {
	err := db.DB.Save(user).Error
	return err
}

func (db *GormDB) FetchUser(user *model.User, uid uint) error {
	err := db.DB.Where("id = ?", uid).First(user).Error
	return err
}

func (db *GormDB) FetchUserPreloadS(user *model.User, id uint) error {
	err := db.DB.Preload("Skills").Where("id = ?", id).First(user).Error
	return err
}

func (db *GormDB) FetchuserPreloadF(user, friend *model.User, uid uint) error {
	err := db.DB.Preload("Friends").Where("id = ?", uid).First(user).Error
	return err
}

func (db *GormDB) SearchUser(user *model.User, username string) error {
	err := db.DB.Where("username = ?", username).First(user).Error
	return err
}

func (db *GormDB) SearchUserEmail(user *model.User, email string) error {
	err := db.DB.Where("email = ?", email).First(user).Error
	return err
}

func (db *GormDB) SearchUserPreloadSP(user *model.User, username string) error {
	err := db.DB.Preload("Skills").Preload("Posts").Where("username = ?", username).First(user).Error
	return err
}

func (db *GormDB) SearchUserGitPreloadSP(user *model.User, gitusername string) error {
	err := db.DB.Preload("Skills").Preload("Posts").Where("gitusername = ?", gitusername).First(user).Error
	return err
}

func (db *GormDB) CheckExistingFriendReq(req *model.FriendReq, uid, fid uint) error {
	err := db.DB.Where("user_id = ?", uid).Where("friend_id = ?", fid).First(req).Error
	return err
}

func (db *GormDB) ViewFriendReq(user *model.User, uid uint) error {
	err := db.DB.Preload("FriendReq.Friend").Preload("RecFriendReq.User").Where("id = ?", uid).First(user).Error
	return err
}

func (db *GormDB) FetchFriendReq(req *model.FriendReq, rid uint) error {
	err := db.DB.Where("id = ?", rid).First(req).Error
	return err
}

func (db *GormDB) UpdateFriendReqReject(req *model.FriendReq) error {
	err := db.DB.Model(req).Update("Status", model.StatusRejected).Error
	return err
}

func (db *GormDB) UpdateFriendReqAccept(req *model.FriendReq, user, friend *model.User) error {
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Delete(req).Error; err != nil {
			return err
		}

		if err := tx.Model(user).Association("Friends").Append(friend); err != nil {
			return err
		}

		if err := tx.Model(friend).Association("Friends").Append(user); err != nil {
			return err
		}
		return nil
	})
	return err
}

func (db *GormDB) DeleteFriendReq(req *model.FriendReq) error {
	err := db.DB.Unscoped().Delete(req).Error
	return err
}

func (db *GormDB) DeleteFriend(user, friend *model.User) error {
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(user).Association("Friends").Delete(friend); err != nil {
			return err
		}

		if err := tx.Model(friend).Association("Friends").Delete(user); err != nil {
			return err
		}
		return nil
	})
	return err
}

func (db *GormDB) UpdateSkills(user *model.User, skills *model.Skill) error {
	err := db.DB.Model(user).Association("Skills").Replace(skills)
	return err
}

func (db *GormDB) DeleteSkills(user *model.User, skills *model.Skill) error {
	err := db.DB.Model(user).Association("Skills").Delete(skills)
	return err
}

func (db *GormDB) DeleteUser(user *model.User) error {
	err := db.DB.Delete(user).Error
	return err
}

func (db *GormDB) AddPost(post *model.Post) error {
	err := db.DB.Create(post).Error
	return err
}
