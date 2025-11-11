// Package unit -> Unit tests for the app
package unit

import (
	"net/http"
	"os"
	"testing"

	"findme/core"
	"findme/handlers"
	"findme/model"

	"github.com/gin-gonic/gin"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	id1 = ""
	id2 = ""
	pid = ""
	cid = ""
)

var router *gin.Engine

func getTestDB() *core.GormDB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	db.AutoMigrate(
		&model.Skill{},
		&model.User{},
		&model.Post{},
		&model.PostSkill{},
		&model.UserSkill{},
		&model.UserFriend{},
		&model.FriendReq{},
		&model.UserMessage{},
		&model.PostReq{},
	)

	db.SetupJoinTable(&model.Post{}, "Tags", &model.PostSkill{})
	db.SetupJoinTable(&model.User{}, "Skills", &model.UserSkill{})
	db.SetupJoinTable(&model.User{}, "Friends", &model.UserFriend{})
	db.SetupJoinTable(&model.User{}, "Chats", &model.ChatUser{})

	gdb := core.NewGormDB(db)
	superUser(gdb)
	return gdb
}

func getTestRouter(service *handlers.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	handlers.SetupHandler(router, service)
	return router
}

func superUser(db *core.GormDB) {
	gitusername := "imisi99"

	be := model.Skill{Name: "backend"}
	ml := model.Skill{Name: "ml"}

	skill := []*model.Skill{&ml, &be}
	db.DB.Create(skill)

	hashpass, _ := core.HashPassword("Password")
	super := model.User{
		FullName:     "Isong Imisioluwa",
		UserName:     "Imisioluwa23",
		GitUserName:  &gitusername,
		GitUser:      true,
		Bio:          "I am the super user",
		Email:        "isongrichard234@gmail.com",
		Skills:       []*model.Skill{&be, &ml},
		Password:     hashpass,
		Availability: true,
	}

	super1 := model.User{
		FullName:     "Isong Imisioluwa",
		UserName:     "knightmares23",
		Email:        "knightmares234@gmail.com",
		Password:     hashpass,
		Availability: true,
		Skills:       []*model.Skill{&be},
		Bio:          "I'm the second super user",
	}

	users := []*model.User{&super, &super1}
	db.DB.Create(users)

	post := model.Post{
		Description:  "Working on a platform for finding developers for contributive project",
		UserID:       super.ID,
		Views:        4,
		Tags:         []*model.Skill{&be},
		Availability: true,
	}

	chat := model.Chat{}

	db.DB.Create(&post)

	db.DB.Create(&chat)

	db.DB.Model(&super).Association("Friends").Append(&super1)
	db.DB.Model(&super1).Association("Friends").Append(&super)
	db.DB.Model(&super).Association("Chats").Append(&chat)
	db.DB.Model(&super1).Association("Chats").Append(&chat)

	id1 = super.ID
	id2 = super1.ID
	pid = post.ID
	cid = chat.ID
}

func TestMain(m *testing.M) {
	db := getTestDB()
	rdb := NewCacheMock()
	email := NewEmailMock()
	git := NewGitMock()
	service := handlers.NewService(db, rdb, email, git, &http.Client{})

	var skills []model.Skill
	service.DB.FetchAllSkills(&skills)
	service.RDB.CacheSkills(skills)
	router = getTestRouter(service)
	tokenString, _ = handlers.GenerateJWT(id1, "login", handlers.JWTExpiry)  // Initially the logged in user is the super user me for the post test
	tokenString1, _ = handlers.GenerateJWT(id2, "login", handlers.JWTExpiry) // User for saving post
	code := m.Run()

	os.Exit(code)
}
