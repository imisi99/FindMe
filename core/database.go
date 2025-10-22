package core

import (
	"errors"
	"net/http"

	"findme/model"

	"gorm.io/gorm"
)

type DB interface {
	FetchAllSkills(skills *[]model.Skill) error
	AddUser(user *model.User) error
	CheckExistingUser(user *model.User, email, username string) error
	CheckExistingUserUpdate(user *model.User, email, username string, uid uint) error
	CheckExistingEmail(user *model.User, email string) error
	CheckExistingUsername(user *model.User, username string) error
	VerifyUser(user *model.User, username string) error
	SaveUser(user *model.User) error
	FetchUser(user *model.User, uid uint) error
	FetchUserPreloadS(user *model.User, id uint) error
	FetchUserPreloadF(user *model.User, uid uint) error
	FetchUserPreloadFReq(user *model.User, uid uint) error
	SearchUser(user *model.User, username string) error
	SearchUserEmail(user *model.User, email string) error
	SearchUserPreloadSP(user *model.User, username string) error
	SearchUserGitPreloadSP(user *model.User, gitusername string) error
	SearchUsersBySKills(users *[]model.User, skills []string) error
	AddFriendReq(req *model.FriendReq) error
	ViewFriendReq(user *model.User, uid uint) error
	FetchFriendReq(req *model.FriendReq, rid uint) error
	UpdateFriendReqReject(req *model.FriendReq) error
	UpdateFriendReqAccept(req *model.FriendReq, user, friend *model.User) error
	DeleteFriendReq(req *model.FriendReq) error
	DeleteFriend(user, friend *model.User) error
	UpdateSkills(user *model.User, skills []*model.Skill) error
	DeleteSkills(user *model.User, skills []*model.Skill) error
	DeleteUser(user *model.User) error
	AddPost(post *model.Post) error
	FetchUserPosts(user *model.User, uid uint) error
	FetchPost(post *model.Post, pid uint) error
	FetchPostPreloadT(post *model.Post, pid uint) error
	FetchPostPreloadTU(post *model.Post, pid uint) error
	EditPost(post *model.Post, skills []*model.Skill) error
	SavePost(post *model.Post) error
	BookmarkPost(user *model.User, post *model.Post) error
	FetchUserPreloadB(user *model.User, uid uint) error
	FetchPostPreloadU(post *model.Post, pid uint) error
	SearchPostsBySKills(posts *[]model.Post, skills []string) error
	RemoveBookmarkedPost(user *model.User, post *model.Post) error
	AddPostApplicationReq(req *model.PostReq) error
	ViewPostApplications(user *model.User, uid uint) error
	FetchPostApplication(req *model.PostReq, rid uint) error
	UpdatePostAppliationReject(req *model.PostReq) error
	UpdatePostApplicationAccept(req *model.PostReq, user1, user2 *model.User, friends bool) error
	FetchPostAppPreloadFU(req *model.PostReq, rid uint) error
	DeletePostApplicationReq(req *model.PostReq) error
	DeletePost(post *model.Post) error
	AddSkills(skills *[]*model.Skill) error
	FindExistingSkills(skills *[]*model.Skill, skill []string) error
	FindExistingGitID(user *model.User, gitid int64) error
}

type GormDB struct {
	DB *gorm.DB
}

func NewGormDB(db *gorm.DB) *GormDB {
	return &GormDB{DB: db}
}

func (db *GormDB) FetchAllSkills(skills *[]model.Skill) error {
	if err := db.DB.Find(skills).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch skills."}
	}
	return nil
}

func (db *GormDB) AddUser(user *model.User) error {
	if err := db.DB.Create(user).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to create new user."}
	}
	return nil
}

func (db *GormDB) CheckExistingUser(user *model.User, email, username string) error {
	err := db.DB.Where("username = ? OR email = ?", username, email).First(user).Error
	if err == nil {
		if user.Email == email {
			return &CustomMessage{Code: http.StatusConflict, Message: "Email already in use !"}
		} else {
			return &CustomMessage{Code: http.StatusConflict, Message: "Username already in use !"}
		}
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
	}
	return nil
}

func (db *GormDB) CheckExistingUserUpdate(user *model.User, email, username string, uid uint) error {
	err := db.DB.Where("username = ? OR email = ?", username, email).First(user).Error
	if err == nil && user.ID != uid {
		if user.Email == email {
			return &CustomMessage{Code: http.StatusConflict, Message: "Email already in use !"}
		} else {
			return &CustomMessage{Code: http.StatusConflict, Message: "Username already in use !"}
		}
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
	}
	return nil
}

func (db *GormDB) VerifyUser(user *model.User, username string) error {
	if err := db.DB.Where("username = ? OR email = ?", username, username).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Invalid Credentials !"}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) SaveUser(user *model.User) error {
	if err := db.DB.Save(user).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to save user."}
	}
	return nil
}

func (db *GormDB) FetchUser(user *model.User, uid uint) error {
	if err := db.DB.Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) FetchUserPreloadS(user *model.User, id uint) error {
	if err := db.DB.Preload("Skills").Where("id = ?", id).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) FetchUserPreloadF(user *model.User, uid uint) error {
	if err := db.DB.Preload("Friends").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) FetchUserPreloadFReq(user *model.User, uid uint) error {
	if err := db.DB.Preload("Friends").Preload("FriendReq.Friend").Preload("RecFriendReq.User").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) SearchUser(user *model.User, username string) error {
	if err := db.DB.Where("username = ?", username).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) SearchUserEmail(user *model.User, email string) error {
	if err := db.DB.Where("email = ?", email).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) SearchUserPreloadSP(user *model.User, username string) error {
	if err := db.DB.Preload("Skills").Preload("Posts").Where("username = ?", username).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) SearchUserGitPreloadSP(user *model.User, gitusername string) error {
	if err := db.DB.Preload("Skills").Preload("Posts").Where("gitusername = ?", gitusername).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) SearchUsersBySKills(users *[]model.User, skills []string) error {
	subquery := db.DB.Select("user_id").
		Table("user_skills").
		Joins("JOIN skills s ON user_skills.skill_id = s.id").
		Where("s.name IN ?", skills)

	if err := db.DB.Preload("Skills").Where("id IN (?)", subquery).Find(users).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to find users."}
	}
	return nil
}

// TODO: I can use the users friend req list instead of the DB this makes it more efficient

func (db *GormDB) CheckExistingFriendReq(req *model.FriendReq, uid, fid uint) error {
	if err := db.DB.Where("user_id = ?", uid).Where("friend_id = ?", fid).First(req).Error; err == nil {
		return &CustomMessage{Code: http.StatusConflict, Message: "Request to this user already exists !"}
	}
	return nil
}

func (db *GormDB) AddFriendReq(req *model.FriendReq) error {
	if err := db.DB.Create(req).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to create request."}
	}
	return nil
}

func (db *GormDB) ViewFriendReq(user *model.User, uid uint) error {
	if err := db.DB.Preload("FriendReq.Friend").Preload("RecFriendReq.User").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) FetchFriendReq(req *model.FriendReq, rid uint) error {
	if err := db.DB.Where("id = ?", rid).First(req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Request not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve request."}
		}
	}
	return nil
}

func (db *GormDB) UpdateFriendReqReject(req *model.FriendReq) error {
	if err := db.DB.Model(req).Update("Status", model.StatusRejected).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to update request status."}
	}
	return nil
}

func (db *GormDB) UpdateFriendReqAccept(req *model.FriendReq, user, friend *model.User) error {
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
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
	}); err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to update request status."}
	}
	return nil
}

func (db *GormDB) DeleteFriendReq(req *model.FriendReq) error {
	if err := db.DB.Unscoped().Delete(req).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to delete friend request."}
	}
	return nil
}

func (db *GormDB) DeleteFriend(user, friend *model.User) error {
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(user).Association("Friends").Delete(friend); err != nil {
			return err
		}
		if err := tx.Model(friend).Association("Friends").Delete(user); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to delete friend."}
	}
	return nil
}

func (db *GormDB) UpdateSkills(user *model.User, skills []*model.Skill) error {
	if err := db.DB.Model(user).Association("Skills").Replace(skills); err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to update skills."}
	}
	return nil
}

func (db *GormDB) DeleteSkills(user *model.User, skills []*model.Skill) error {
	if err := db.DB.Model(user).Association("Skills").Delete(skills); err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to delete skills."}
	}
	return nil
}

func (db *GormDB) DeleteUser(user *model.User) error {
	if err := db.DB.Delete(user).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to delete user"}
	}
	return nil
}

func (db *GormDB) AddPost(post *model.Post) error {
	err := db.DB.Create(post).Error
	return err
}

func (db *GormDB) FetchUserPosts(user *model.User, uid uint) error {
	if err := db.DB.Preload("Posts.Tags").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch user posts."}
		}
	}
	return nil
}

func (db *GormDB) FetchPost(post *model.Post, pid uint) error {
	if err := db.DB.Where("id = ?", pid).First(post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Post not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch post."}
		}
	}
	return nil
}

func (db *GormDB) FetchPostPreloadT(post *model.Post, pid uint) error {
	if err := db.DB.Preload("Tags").Where("id = ?", pid).First(post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Post not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch post."}
		}
	}
	return nil
}

func (db *GormDB) FetchPostPreloadTU(post *model.Post, pid uint) error {
	if err := db.DB.Preload("Tags").Preload("User").Where("id = ?", pid).First(post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Post not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch post."}
		}
	}
	return nil
}

func (db *GormDB) EditPost(post *model.Post, skills []*model.Skill) error {
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(post).Association("Tags").Replace(skills); err != nil {
			return err
		}
		if err := tx.Save(post).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to edit post."}
	}
	return nil
}

func (db *GormDB) SavePost(post *model.Post) error {
	if err := db.DB.Save(post).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to save post."}
	}
	return nil
}

func (db *GormDB) BookmarkPost(user *model.User, post *model.Post) error {
	if err := db.DB.Model(user).Association("SavedPosts").Append(post); err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to bookmark post."}
	}
	return nil
}

func (db *GormDB) FetchUserPreloadB(user *model.User, uid uint) error {
	if err := db.DB.Preload("SavedPosts").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch user bookmarks."}
		}
	}
	return nil
}

func (db *GormDB) FetchPostPreloadU(post *model.Post, pid uint) error {
	if err := db.DB.Preload("User").Where("id = ?", pid).First(post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Post not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch post."}
		}
	}
	return nil
}

func (db *GormDB) SearchPostsBySKills(posts *[]model.Post, skills []string) error {
	subquery := db.DB.Select("post_id").
		Table("post_skills").
		Joins("JOIN skills s ON post_skills.skill_id = s.id").
		Where("s.name IN ?", skills)

	if err := db.DB.Preload("Tags").Where("id IN (?)", subquery).Find(&posts).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to search post by skills."}
	}
	return nil
}

func (db *GormDB) RemoveBookmarkedPost(user *model.User, post *model.Post) error {
	if err := db.DB.Model(user).Association("SavedPosts").Delete(post); err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to remove post from bookmark."}
	}
	return nil
}

func (db *GormDB) AddPostApplicationReq(req *model.PostReq) error {
	if err := db.DB.Save(req).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to send post application request."}
	}
	return nil
}

func (db *GormDB) ViewPostApplications(user *model.User, uid uint) error {
	if err := db.DB.Preload("RecPostReq.FromUser").Preload("SentPostReq.ToUser").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch user post applications."}
		}
	}
	return nil
}

func (db *GormDB) FetchPostApplication(req *model.PostReq, rid uint) error {
	if err := db.DB.Preload("FromUser").Preload("Post").Where("id = ?", rid).First(req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Post request not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch post request."}
		}
	}
	return nil
}

func (db *GormDB) UpdatePostAppliationReject(req *model.PostReq) error {
	if err := db.DB.Model(req).Update("Status", model.StatusRejected).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to update post application status."}
	}
	return nil
}

func (db *GormDB) UpdatePostApplicationAccept(req *model.PostReq, user1, user2 *model.User, friends bool) error {
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Delete(req).Error; err != nil {
			return err
		}
		if !friends {
			if err := tx.Model(user1).Association("Friends").Append(user2); err != nil {
				return err
			}
			if err := tx.Model(user2).Association("Friends").Append(user1); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to update post application status."}
	}
	return nil
}

func (db *GormDB) FetchPostAppPreloadFU(req *model.PostReq, rid uint) error {
	if err := db.DB.Preload("FromUser").Where("id = ?", rid).First(req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Post application request not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch post application request."}
		}
	}
	return nil
}

func (db *GormDB) DeletePostApplicationReq(req *model.PostReq) error {
	if err := db.DB.Unscoped().Delete(req).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to delete post application request."}
	}
	return nil
}

func (db *GormDB) DeletePost(post *model.Post) error {
	if err := db.DB.Delete(post).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to delete post."}
	}
	return nil
}

func (db *GormDB) AddSkills(skills *[]*model.Skill) error {
	if err := db.DB.Create(skills).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to create new skills."}
	}
	return nil
}

func (db *GormDB) FindExistingSkills(skills *[]*model.Skill, skill []string) error {
	if err := db.DB.Where("name IN ?", skill).Find(skills).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to find skills"}
	}
	return nil
}

func (db *GormDB) FindExistingGitID(user *model.User, gitid int64) error {
	err := db.DB.Where("gitid = ?", gitid).First(user).Error
	return err
}

func (db *GormDB) CheckExistingEmail(user *model.User, email string) error {
	err := db.DB.Where("email = ?", email).First(user).Error
	return err
}

func (db *GormDB) CheckExistingUsername(user *model.User, username string) error {
	err := db.DB.Where("username = ?", username).First(user).Error
	return err
}
