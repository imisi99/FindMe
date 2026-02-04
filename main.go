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
	"github.com/robfig/cron/v3"

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

	// Setup db, redis and cron
	dbClient := database.Connect()
	rdbClient := database.ConnectRedis()
	db := core.NewGormDB(dbClient)
	rdb := core.NewRDB(rdbClient)
	cron := &cron.Cron{}

	// setup chat, email rec and embedding hub
	client := &http.Client{Timeout: 10 * time.Minute}
	email := core.NewEmailService("smtp.gmail.com", os.Getenv("EMAIL"), os.Getenv("EMAIL_APP_PASSWORD"), 587)

	chathub := core.NewChatHub(1000)
	emailHub := core.NewEmailHub(2000, 5, email)
	embHub := core.NewEmbeddingHub(100, 10, "emb:8000")
	recHub := core.NewRecommendationHub(10, 100, "rec:8050")

	worker := core.NewCron(db, emailHub, cron)

	// set up git and transc service
	git := handlers.NewGitService(os.Getenv("GIT_CLIENT_ID"), os.Getenv("GIT_CLIENT_SECRET"), os.Getenv("GIT_CALLBACK_URL"), db, embHub, client)
	transc := handlers.NewTranscService(db, os.Getenv("PAYSTACK_API_KEY"), client)
	service := handlers.NewService(db, rdb, emailHub, git, transc, embHub, recHub, client, chathub, worker)

	go chathub.Run()
	go embHub.Run()
	go emailHub.Run()
	go recHub.Run()
	cron.Start()

	var skills []model.Skill
	if err := service.DB.FetchAllSkills(&skills); err != nil {
		log.Fatalln("Failed to Fetch skills from DB exiting...")
	}
	service.RDB.CacheSkills(skills)
	router := gin.Default()

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swagFiles.Handler))

	// Cron Worker
	err = service.Cron.TrialEndingReminders()
	if err != nil {
		log.Println("[CRON] An error occured err -> ", err.Error())
	}

	handlers.SetupHandler(router, service)

	err = router.Run("0.0.0.0:8080")
	if err != nil {
		log.Fatalln("Failed to start app -> ", err.Error())
	}
}
