package core

import (
	"errors"
	"log"
	"net/http"

	"findme/model"

	"gorm.io/gorm"
)

type DB interface {
	FetchAllSkills(skills *[]model.Skill) error
	AddUser(user *model.User) error
	CheckExistingUser(user *model.User, email, username string) error
	CheckExistingUserUpdate(user *model.User, email, username, uid string) error
	CheckExistingEmail(email string) error
	CheckExistingUsername(username string) error
	CheckExistingFriends(uid, fid string) (error, bool)
	CheckExistingFriendReq(uid, fid string) (error, bool)
	CheckExistingAppReq(pid, uid string) (error, bool)
	VerifyUser(user *model.User, username string) error
	SaveUser(user *model.User) error
	FetchUser(user *model.User, uid string) error
	FetchUserPreloadSP(user *model.User, uid string) error
	FetchUserPreloadS(user *model.User, uid string) error
	FetchUserPreloadF(user *model.User, uid string) error
	FetchUserPreloadFReq(user *model.User, uid string) error
	FetchUserPreloadPReq(user *model.User, uid string) error
	SearchUserEmail(user *model.User, email string) error
	SearchUserPreloadSP(user *model.User, username string) error
	SearchUserGitPreloadSP(user *model.User, gitusername string) error
	SearchUsersBySKills(users *[]model.User, skills []string, uid string) error
	AddFriendReq(req *model.FriendReq) error
	ViewFriendReq(user *model.User, uid string) error
	FetchFriendReq(req *model.FriendReq, rid string) error
	UpdateFriendReqReject(req *model.FriendReq) error
	UpdateFriendReqAccept(req *model.FriendReq, user, friend *model.User, chat *model.Chat) error
	DeleteFriendReq(req *model.FriendReq) error
	DeleteFriend(user, friend *model.User, chat *model.Chat) error
	UpdateSkills(user *model.User, skills []*model.Skill) error
	DeleteSkills(user *model.User, skills []*model.Skill) error
	DeleteUser(user *model.User) error
	AddProject(project *model.Project) error
	FetchUserProjects(user *model.User, uid string) error
	FetchProject(project *model.Project, pid string) error
	FetchProjectPreloadT(project *model.Project, pid string) error
	FetchProjectPreloadTU(project *model.Project, pid string) error
	FetchProjectPreloadA(project *model.Project, pid string) error
	FetchProjectPreloadC(project *model.Project, pid string) error
	EditProject(project *model.Project, skills []*model.Skill) error
	SaveProject(project *model.Project) error
	BookmarkProject(user *model.User, project *model.Project) error
	FetchUserPreloadB(user *model.User, uid string) error
	FetchProjectPreloadU(project *model.Project, pid string) error
	SearchProjectsBySKills(projects *[]model.Project, skills []string, uid string) error
	RemoveBookmarkedProject(user *model.User, project *model.Project) error
	AddProjectApplicationReq(req *model.ProjectReq) error
	ViewProjectApplications(user *model.User, uid string) error
	FetchProjectApplication(req *model.ProjectReq, rid string) error
	UpdateProjectAppliationReject(req *model.ProjectReq) error
	UpdateProjectApplicationAccept(req *model.ProjectReq, user *model.User, chat *model.Chat) error
	UpdateProjectApplicationAcceptF(req *model.ProjectReq, user1, user2 *model.User, project *model.Project, chat *model.Chat) error
	FetchProjectAppPreloadFU(req *model.ProjectReq, rid string) error
	DeleteProjectApplicationReq(req *model.ProjectReq) error
	ClearProjectApplication(req []*model.ProjectReq) error
	DeleteProject(project *model.Project) error
	AddSkills(skills *[]*model.Skill) error
	FindExistingSkills(skills *[]*model.Skill, skill []string) error
	FindExistingGitID(user *model.User, gitid int64) error
	AddMessage(msg *model.UserMessage) error
	GetChatHistory(chatID string, chat *model.Chat) error
	FetchChat(chatID string, chat *model.Chat) error
	FetchUserPreloadCM(user *model.User, uid string) error
	FetchUserPreloadC(user *model.User, uid string) error
	FetchMsg(msg *model.UserMessage, mid string) error
	SaveMsg(msg *model.UserMessage) error
	SaveChat(chat *model.Chat) error
	DeleteMsg(msg *model.UserMessage) error
	FindChat(uid, fid string, chat *model.Chat) error
	AddUserChat(chat *model.Chat, user *model.User) error
	RemoveUserChat(chat *model.Chat, user *model.User) error
	LeaveChat(chat *model.Chat, user *model.User) error
	DeleteChat(chat *model.Chat) error
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
		log.Println("Failed to create user err -> ", err.Error())
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to create new user."}
	}
	return nil
}

func (db *GormDB) CheckExistingUser(user *model.User, email, username string) error {
	err := db.DB.Where("username = ? OR email = ?", username, email).First(user).Error
	if err == nil {
		if user.Email == email {
			return &CustomMessage{Code: http.StatusConflict, Message: "Email already in use!"}
		} else {
			return &CustomMessage{Code: http.StatusConflict, Message: "Username already in use!"}
		}
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
	}
	return nil
}

func (db *GormDB) CheckExistingUserUpdate(user *model.User, email, username, uid string) error {
	err := db.DB.Where("username = ? OR email = ?", username, email).First(user).Error
	if err == nil && user.ID != uid {
		if user.Email == email {
			return &CustomMessage{Code: http.StatusConflict, Message: "Email already in use!"}
		} else {
			return &CustomMessage{Code: http.StatusConflict, Message: "Username already in use!"}
		}
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) && err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
	}
	return nil
}

func (db *GormDB) CheckExistingFriends(uid, fid string) (error, bool) {
	var count int64
	if err := db.DB.Model(&model.UserFriend{}).Where("(user_id = ? AND friend_id = ?)", uid, fid).Count(&count).Error; err != nil {
		return &CustomMessage{http.StatusInternalServerError, "Failed to check for existing friendship staus."}, false
	}
	if count > 0 {
		return &CustomMessage{Code: http.StatusConflict, Message: "You're already friends!"}, true
	}
	return nil, false
}

func (db *GormDB) CheckExistingFriendReq(uid, fid string) (error, bool) {
	var count int64
	if err := db.DB.Model(&model.FriendReq{}).Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", uid, fid, fid, uid).Count(&count).Error; err != nil {
		return &CustomMessage{http.StatusInternalServerError, "Failed to check for existing friend req."}, false
	}
	if count > 0 {
		return &CustomMessage{http.StatusConflict, "An existing request exists."}, true
	}
	return nil, false
}

func (db *GormDB) CheckExistingAppReq(pid, uid string) (error, bool) {
	var count int64
	if err := db.DB.Model(&model.ProjectReq{}).Where("(project_id = ? AND from_id = ?)", pid, uid).Count(&count).Error; err != nil {
		return &CustomMessage{http.StatusInternalServerError, "Failed to check for existing project application req."}, false
	}
	if count > 0 {
		return &CustomMessage{http.StatusConflict, "An existing request exists."}, true
	}
	return nil, false
}

func (db *GormDB) VerifyUser(user *model.User, username string) error {
	if err := db.DB.Where("username = ? OR email = ?", username, username).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Invalid Credentials!"}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) SaveUser(user *model.User) error {
	if err := db.DB.Save(user).Error; err != nil {
		log.Println("Failed to save user with id -> ", user.ID, "err -> ", err.Error())
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to save user."}
	}
	return nil
}

func (db *GormDB) FetchUser(user *model.User, uid string) error {
	if err := db.DB.Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) FetchUserPreloadSP(user *model.User, uid string) error {
	if err := db.DB.Preload("Projects.Tags").Preload("Skills").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{http.StatusNotFound, "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) FetchUserPreloadS(user *model.User, uid string) error {
	if err := db.DB.Preload("Skills").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) FetchUserPreloadF(user *model.User, uid string) error {
	if err := db.DB.Preload("Friends").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) FetchUserPreloadFReq(user *model.User, uid string) error {
	if err := db.DB.Preload("Friends").Preload("FriendReq.Friend").Preload("RecFriendReq.User").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) FetchUserPreloadPReq(user *model.User, uid string) error {
	if err := db.DB.Preload("SentProjectReq").Where("id = ?", uid).First(user).Error; err != nil {
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
	if err := db.DB.Preload("Skills").Preload("Projects.Tags").Where("username = ?", username).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) SearchUserGitPreloadSP(user *model.User, gitusername string) error {
	if err := db.DB.Preload("Skills").Preload("Projects.Tags").Where("gitusername = ?", gitusername).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) SearchUsersBySKills(users *[]model.User, skills []string, uid string) error {
	subquery := db.DB.Select("user_id").
		Table("user_skills").
		Joins("JOIN skills s ON user_skills.skill_id = s.id").
		Where("s.name IN ?", skills)

	if err := db.DB.Preload("Skills").Where("id IN (?)", subquery).Where("id != ?", uid).Find(users).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to find users."}
	}
	return nil
}

func (db *GormDB) AddFriendReq(req *model.FriendReq) error {
	if err := db.DB.Create(req).Error; err != nil {
		log.Println("Failed to create friend req err -> ", err.Error())
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to create request."}
	}
	return nil
}

func (db *GormDB) ViewFriendReq(user *model.User, uid string) error {
	if err := db.DB.Preload("FriendReq.Friend").Preload("RecFriendReq.User").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to retrieve user."}
		}
	}
	return nil
}

func (db *GormDB) FetchFriendReq(req *model.FriendReq, rid string) error {
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
	if err := db.DB.Delete(req).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to update request status."}
	}
	return nil
}

func (db *GormDB) UpdateFriendReqAccept(req *model.FriendReq, user, friend *model.User, chat *model.Chat) error {
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
		if err := tx.Create(chat).Error; err != nil {
			return err
		}
		if err := tx.Model(user).Association("Chats").Append(chat); err != nil {
			return err
		}
		if err := tx.Model(friend).Association("Chats").Append(chat); err != nil {
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
		log.Println("Failed to delete friend req with id -> ", req.ID, "err -> ", err.Error())
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to delete friend request."}
	}
	return nil
}

func (db *GormDB) DeleteFriend(user, friend *model.User, chat *model.Chat) error {
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(user).Association("Friends").Delete(friend); err != nil {
			return err
		}
		if err := tx.Model(friend).Association("Friends").Delete(user); err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(chat).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Println("Failed to delete user friend with ids -> ", user.ID, friend.ID, "err -> ", err.Error())
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
		log.Println("Failed to delete user with id -> ", user.ID, "err -> ", err.Error())
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to delete user."}
	}
	return nil
}

func (db *GormDB) AddProject(project *model.Project) error {
	if err := db.DB.Create(project).Error; err != nil {
		log.Println("Failed to create project err -> ", err.Error())
		return &CustomMessage{http.StatusInternalServerError, "Failed to create project."}
	}
	return nil
}

func (db *GormDB) FetchUserProjects(user *model.User, uid string) error {
	if err := db.DB.Preload("Projects.Tags").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch user projects."}
		}
	}
	return nil
}

func (db *GormDB) FetchProject(project *model.Project, pid string) error {
	if err := db.DB.Where("id = ?", pid).First(project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Project not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch project."}
		}
	}
	return nil
}

func (db *GormDB) FetchProjectPreloadT(project *model.Project, pid string) error {
	if err := db.DB.Preload("Tags").Where("id = ?", pid).First(project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Project not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch project."}
		}
	}
	return nil
}

func (db *GormDB) FetchProjectPreloadTU(project *model.Project, pid string) error {
	if err := db.DB.Preload("Tags").Preload("User").Where("id = ?", pid).First(project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Project not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch project."}
		}
	}
	return nil
}

func (db *GormDB) FetchProjectPreloadA(project *model.Project, pid string) error {
	if err := db.DB.Preload("Applications.FromUser").Where("id = ?", pid).First(project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Project not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch project."}
		}
	}
	return nil
}

func (db *GormDB) FetchProjectPreloadC(project *model.Project, pid string) error {
	if err := db.DB.Preload("Chat").Where("id = ?", pid).First(project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Project not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch project."}
		}
	}
	return nil
}

func (db *GormDB) EditProject(project *model.Project, skills []*model.Skill) error {
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(project).Association("Tags").Replace(skills); err != nil {
			return err
		}
		if err := tx.Save(project).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to edit project."}
	}
	return nil
}

func (db *GormDB) SaveProject(project *model.Project) error {
	if err := db.DB.Save(project).Error; err != nil {
		log.Println("Failed to save project with id -> ", project.ID, "err -> ", err.Error())
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to save project."}
	}
	return nil
}

func (db *GormDB) BookmarkProject(user *model.User, project *model.Project) error {
	if err := db.DB.Model(user).Association("SavedProjects").Append(project); err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to bookmark project."}
	}
	return nil
}

func (db *GormDB) FetchUserPreloadB(user *model.User, uid string) error {
	if err := db.DB.Preload("SavedProjects").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch user bookmarks."}
		}
	}
	return nil
}

func (db *GormDB) FetchProjectPreloadU(project *model.Project, pid string) error {
	if err := db.DB.Preload("User").Where("id = ?", pid).First(project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Project not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch project."}
		}
	}
	return nil
}

func (db *GormDB) SearchProjectsBySKills(projects *[]model.Project, skills []string, uid string) error {
	subquery := db.DB.Select("project_id").
		Table("project_skills").
		Joins("JOIN skills s ON project_skills.skill_id = s.id").
		Where("s.name IN ?", skills)

	if err := db.DB.Preload("Tags").Where("id IN (?)", subquery).Where("user_id != ?", uid).Find(&projects).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to search project by skills."}
	}
	return nil
}

func (db *GormDB) RemoveBookmarkedProject(user *model.User, project *model.Project) error {
	if err := db.DB.Model(user).Association("SavedProjects").Delete(project); err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to remove project from bookmark."}
	}
	return nil
}

func (db *GormDB) AddProjectApplicationReq(req *model.ProjectReq) error {
	if err := db.DB.Save(req).Error; err != nil {
		log.Println("Failed to save project application req with id -> ", req.ID, "err -> ", err.Error())
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to send project application request."}
	}
	return nil
}

func (db *GormDB) ViewProjectApplications(user *model.User, uid string) error {
	if err := db.DB.Preload("RecProjectReq.FromUser").Preload("SentProjectReq.ToUser").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "User not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch user project applications."}
		}
	}
	return nil
}

func (db *GormDB) FetchProjectApplication(req *model.ProjectReq, rid string) error {
	if err := db.DB.Preload("FromUser").Preload("ToUser").Preload("Project.Chat").Where("id = ?", rid).First(req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Project request not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch project request."}
		}
	}
	return nil
}

func (db *GormDB) UpdateProjectAppliationReject(req *model.ProjectReq) error {
	if err := db.DB.Delete(req).Error; err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to update project application status."}
	}
	return nil
}

func (db *GormDB) UpdateProjectApplicationAccept(req *model.ProjectReq, user *model.User, chat *model.Chat) error {
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Delete(req).Error; err != nil {
			return err
		}

		if err := tx.Model(user).Association("Chats").Append(chat); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to update project application status."}
	}
	return nil
}

func (db *GormDB) UpdateProjectApplicationAcceptF(req *model.ProjectReq, user1, user2 *model.User, project *model.Project, chat *model.Chat) error {
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Delete(req).Error; err != nil {
			return err
		}

		if err := tx.Create(chat).Error; err != nil {
			return err
		}

		if err := tx.Model(project).Update("ChatID", &chat.ID).Error; err != nil {
			return err
		}

		if err := tx.Model(user1).Association("Chats").Append(chat); err != nil {
			return err
		}

		if err := tx.Model(user2).Association("Chats").Append(chat); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Println(err)
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to update project application status."}
	}
	return nil
}

func (db *GormDB) FetchProjectAppPreloadFU(req *model.ProjectReq, rid string) error {
	if err := db.DB.Preload("FromUser").Where("id = ?", rid).First(req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{Code: http.StatusNotFound, Message: "Project application request not found."}
		} else {
			return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to fetch project application request."}
		}
	}
	return nil
}

func (db *GormDB) DeleteProjectApplicationReq(req *model.ProjectReq) error {
	if err := db.DB.Unscoped().Delete(req).Error; err != nil {
		log.Println("Failed to delete project application req with id -> ", req.ID, "err -> ", err.Error())
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to delete project application request."}
	}
	return nil
}

func (db *GormDB) ClearProjectApplication(req []*model.ProjectReq) error {
	if err := db.DB.Unscoped().Delete(req).Error; err != nil {
		log.Println("Failed to clear project applications for project with id -> ", req[0].ProjectID, "err -> ", err.Error())
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to clear project applications."}
	}
	return nil
}

func (db *GormDB) DeleteProject(project *model.Project) error {
	if err := db.DB.Delete(project).Error; err != nil {
		log.Println("Failed to delete project with id -> ", project.ID, "err -> ", err.Error())

		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to delete project."}
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
		return &CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to find skills."}
	}
	return nil
}

func (db *GormDB) FindExistingGitID(user *model.User, gitid int64) error {
	if err := db.DB.Where("gitid = ?", gitid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{http.StatusNotFound, "User with gitid not found."}
		} else {
			return &CustomMessage{http.StatusInternalServerError, "Failed to check for user by gitid."}
		}
	}
	return nil
}

func (db *GormDB) CheckExistingEmail(email string) error {
	var count int64
	if err := db.DB.Model(&model.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return &CustomMessage{http.StatusInternalServerError, "Failed to check for user by email."}
	}
	if count > 0 {
		return &CustomMessage{http.StatusConflict, "Email already in use!"}
	}
	return nil
}

func (db *GormDB) CheckExistingUsername(username string) error {
	var count int64
	if err := db.DB.Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return &CustomMessage{http.StatusInternalServerError, "Failed to check for user by email."}
	}
	if count > 0 {
		return &CustomMessage{http.StatusConflict, "Username already in use!"}
	}
	return nil
}

func (db *GormDB) AddMessage(msg *model.UserMessage) error {
	if err := db.DB.Create(msg).Error; err != nil {
		log.Println("Failed to create msg err -> ", err.Error())
		return &CustomMessage{http.StatusInternalServerError, "Failed to send message."}
	}
	return nil
}

func (db *GormDB) GetChatHistory(chatID string, chat *model.Chat) error {
	if err := db.DB.Preload("Messages").Preload("Users").Where("id = ?", chatID).First(chat).Error; err != nil {
		log.Println(chatID, err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{http.StatusNotFound, "Chat not found."}
		} else {
			return &CustomMessage{http.StatusInternalServerError, "Failed to get chat history."}
		}
	}
	return nil
}

func (db *GormDB) FetchChat(chatID string, chat *model.Chat) error {
	if err := db.DB.Where("id = ?", chatID).First(chat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{http.StatusNotFound, "Chat not found."}
		} else {
			return &CustomMessage{http.StatusInternalServerError, "Failed to get chat."}
		}
	}
	return nil
}

func (db *GormDB) FetchUserPreloadCM(user *model.User, uid string) error {
	if err := db.DB.Preload("Chats.Messages").Preload("Chats.Users").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{http.StatusNotFound, "User not found."}
		} else {
			return &CustomMessage{http.StatusInternalServerError, "Failed to fetch user chats."}
		}
	}
	return nil
}

func (db *GormDB) FetchUserPreloadC(user *model.User, uid string) error {
	if err := db.DB.Preload("Chats.Users").Where("id = ?", uid).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{http.StatusNotFound, "User not found."}
		} else {
			return &CustomMessage{http.StatusInternalServerError, "Failed to fetch user chats."}
		}
	}
	return nil
}

func (db *GormDB) FetchMsg(msg *model.UserMessage, mid string) error {
	if err := db.DB.Where("id = ?", mid).First(msg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{http.StatusNotFound, "Msg not found."}
		} else {
			return &CustomMessage{http.StatusInternalServerError, "Failed to fetch msg."}
		}
	}
	return nil
}

func (db *GormDB) SaveMsg(msg *model.UserMessage) error {
	if err := db.DB.Save(msg).Error; err != nil {
		log.Println("Failed to save msg with id -> ", msg.ID, "err -> ", err.Error())
		return &CustomMessage{http.StatusInternalServerError, "Failed to edit msg."}
	}
	return nil
}

func (db *GormDB) SaveChat(chat *model.Chat) error {
	if err := db.DB.Save(chat).Error; err != nil {
		log.Printf("Failed to save chat with id -> %v, err -> %v", chat.ID, err.Error())
		return &CustomMessage{http.StatusInternalServerError, "Failed to edit chat."}
	}
	return nil
}

func (db *GormDB) DeleteMsg(msg *model.UserMessage) error {
	if err := db.DB.Delete(msg).Error; err != nil {
		log.Println("Failed to delete msg with id -> ", msg.ID, "err -> ", err.Error())
		return &CustomMessage{http.StatusInternalServerError, "Failed to delete msg."}
	}
	return nil
}

func (db *GormDB) FindChat(uid, fid string, chat *model.Chat) error {
	if err := db.DB.
		Preload("Messages").
		Preload("Users").
		Joins("JOIN chat_users cu1 ON cu1.chat_id = chats.id AND cu1.user_id = ?", uid).
		Joins("JOIN chat_users cu2 ON cu2.chat_id = chats.id AND cu2.user_id = ?", fid).
		Where("\"group\" = ?", false).
		First(chat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CustomMessage{http.StatusNotFound, "Chat for friend not found."}
		} else {
			log.Println(err.Error())
			return &CustomMessage{http.StatusInternalServerError, "Failed to fetch chat for friend."}
		}
	}
	return nil
}

func (db *GormDB) AddUserChat(chat *model.Chat, user *model.User) error {
	if err := db.DB.Model(chat).Association("Users").Append(user); err != nil {
		log.Printf("Unable to add user with id -> %v to chat with id -> %v, Error: %v", user.ID, chat.ID, err.Error())
		return &CustomMessage{http.StatusInternalServerError, "Failed to add user to chat."}
	}
	return nil
}

func (db *GormDB) RemoveUserChat(chat *model.Chat, user *model.User) error {
	if err := db.DB.Model(chat).Association("Users").Delete(user); err != nil {
		log.Printf("Unable to remove user with id -> %v from chat with id -> %v, Error: %v", user.ID, chat.ID, err.Error())
		return &CustomMessage{http.StatusInternalServerError, "Failed to remove user from chat."}
	}
	return nil
}

func (db *GormDB) LeaveChat(chat *model.Chat, user *model.User) error {
	if err := db.DB.Model(chat).Association("Users").Delete(user); err != nil {
		log.Println("Failed to remove user from chat with ID -> ", err.Error())
		return &CustomMessage{http.StatusInternalServerError, "Failed to leave chat."}
	}
	return nil
}

func (db *GormDB) DeleteChat(chat *model.Chat) error {
	if err := db.DB.Delete(chat).Error; err != nil {
		log.Printf("Unable to delete a group chat with id -> %v , err -> %v ", chat.ID, err.Error())
		return &CustomMessage{http.StatusInternalServerError, "Failed to delete chat."}
	}
	return nil
}
