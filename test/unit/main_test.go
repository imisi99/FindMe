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

var router *gin.Engine

func getTestDB() *gorm.DB {
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
	superUser(db)

	return db
}

func getTestRouter(service *handlers.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	handlers.SetupHandler(router, service)
	return router
}

func superUser(db *gorm.DB) {
	gitusername := "imisi99"

	be := model.Skill{Name: "backend"}
	ml := model.Skill{Name: "ml"}

	skill := []*model.Skill{&ml, &be}
	db.Create(skill)

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
	db.Create(users)
	post := model.Post{
		Description:  "Working on a platform for finding developers for contributive project",
		UserID:       super.ID,
		Views:        4,
		Tags:         []*model.Skill{&be},
		Availability: true,
	}

	db.Create(&post)
	db.Model(&super).Association("Friends").Append(&super1)
	db.Model(&super1).Association("Friends").Append(&super)
}

func TestMain(m *testing.M) {
	db := getTestDB()
	rdb := NewCacheMock(db)
	email := NewEmailMock()
	git := NewGitMock()
	service := handlers.NewService(getTestDB(), rdb, email, git, &http.Client{})
	service.RDB.CacheSkills()
	router = getTestRouter(service)
	tokenString, _ = handlers.GenerateJWT(1, "login", handlers.JWTExpiry)  // Initially the logged in user is the super user me for the post test
	tokenString1, _ = handlers.GenerateJWT(2, "login", handlers.JWTExpiry) // User for saving post

	code := m.Run()

	os.Exit(code)
}

