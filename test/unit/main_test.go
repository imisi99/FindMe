package unit

import (
	"findme/core"
	"findme/database"
	"findme/handlers"
	"findme/model"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redismock/v9"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)


var router *gin.Engine
var mock redismock.ClientMock


func getTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	handlers.SetupHandler(router)
	return router
}


func getTestDB() *gorm.DB{
	db, _ := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
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

	database.SetDB(db)
	return db
}


func getTestRDB() redismock.ClientMock{
	rdb, mock := redismock.NewClientMock()

	database.SetRDB(rdb)

	return mock
}


func superUser(db *gorm.DB) {
	gitusername := "imisi99"

	be := model.Skill{Name: "backend"}
	ml := model.Skill{Name: "ml"}

	skill := []*model.Skill{&ml, &be}
	db.Create(skill)

	hashpass, _ := core.HashPassword("Password")
	super := model.User{
		FullName: "Isong Imisioluwa",
		UserName: "Imisioluwa23",
		GitUserName: &gitusername,
		GitUser: true,
		Bio: "I am the super user",
		Email: "isongrichard234@gmail.com",
		Skills: []*model.Skill{&be, &ml},
		Password: hashpass,
		Availability: true,
	}

	super1 := model.User{
		FullName: "Isong Imisioluwa",
		UserName: "knightmares23",
		Email: "knightmares234@gmail.com",
		Password: hashpass,
		Availability: true,
		Skills: []*model.Skill{&be},
		Bio: "I'm the second super user",
	}

	users := []*model.User{&super, &super1}
	db.Create(users)
	post := model.Post{
		Description: "Working on a platform for finding developers for contributive project",
		UserID: super.ID,
		Views: 4,
		Tags: []*model.Skill{&be},
	}

	db.Create(&post)
	db.Model(&super).Association("Friends").Append(&super1)
	db.Model(&super1).Association("Friends").Append(&super)
}


func clearDB(db *gorm.DB) {
	db.Exec("DELETE FROM friend_reqs")
	db.Exec("DELETE FROM post_reqs")
	db.Exec("DELETE FROM user_skills")
	db.Exec("DELETE FROM post_skills")
	db.Exec("DELETE FROM user_friends")
	db.Exec("DELETE FROM user_messages")
	db.Exec("DELETE FROM skills")
	db.Exec("DELETE FROM posts")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM sqlite_sequence")
}


func TestMain(m *testing.M) {

	database.SetDB(getTestDB())
	router = getTestRouter()
	mock = getTestRDB()
	tokenString, _ = core.GenerateJWT(1, "login", core.JWTExpiry)   // Initially the logged in user is the super user me for the post test
	tokenString1, _ = core.GenerateJWT(2, "login", core.JWTExpiry)  // User for saving post


	os.Setenv("Testing", "True") 			// Using this for skipping the sending of email for the the forget password test    not proper

	code := m.Run()

	clearDB(database.GetDB())
	os.Exit(code)
}