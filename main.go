// @title FindMe API
// @version 1.0
// @description API documentation for FindMe application.
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @BasePath /
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"findme/core"
	"findme/database"
	_ "findme/docs"
	"findme/handlers"
	"findme/model"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	swagFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	log.SetPrefix("[FindMe]")
	log.SetFlags(log.Lshortfile)

	err := godotenv.Load()
	if err != nil {
		log.Println("[WARNING] Error loading .env file ->", err, "Ignore if in production")
	}

	// Setup db and redis
	dbClient := database.Connect()
	rdbClient := database.ConnectRedis()
	db := core.NewGormDB(dbClient)
	rdb := core.NewRDB(rdbClient)

	// setup git, chat, email and embedding hub
	client := &http.Client{Timeout: 10 * time.Minute}
	chathub := core.NewChatHub(100)
	emailHub := core.NewEmailHub(200, 5)
	embHub := core.NewEmbeddingHub(100, 10, "")

	go chathub.Run()
	go embHub.Run()
	email := core.NewEmail("smtp.gmail.com", os.Getenv("EMAIL"), os.Getenv("EMAIL_APP_PASSWORD"), 587)
	go emailHub.Run(email)

	git := handlers.NewGitService(os.Getenv("GIT_CLIENT_ID"), os.Getenv("GIT_CLIENT_SECRET"), os.Getenv("GIT_CALLBACK_URL"), db, embHub, client)
	service := handlers.NewService(db, rdb, email, git, client, chathub, emailHub, embHub)
	var skills []model.Skill
	if err := service.DB.FetchAllSkills(&skills); err != nil {
		log.Fatalln("Failed to Fetch skills from DB exiting...")
	}
	service.RDB.CacheSkills(skills)
	router := gin.Default()

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swagFiles.Handler))

	handlers.SetupHandler(router, service)

	err = router.Run("0.0.0.0:8080")
	if err != nil {
		log.Fatalln("Failed to start app -> ", err.Error())
	}
}
