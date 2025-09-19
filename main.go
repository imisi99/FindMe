package main

import (
	"findme/core"
	"findme/database"
	"findme/handlers"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	// Load environment variables from .env file
	log.SetPrefix("[FindMe]")
	log.SetFlags(log.Lshortfile)

	err := godotenv.Load()
	if err != nil {
		log.Println("[WARNING] Error loading .env file ->", err, "Ignore if in production")
	}

	// Setup db and redis
	db := database.Connect()
	rdbClient := database.ConnectRedis()
	rdb := core.NewRDB(rdbClient, db)
	client := &http.Client{Timeout: 10*time.Minute}
	email := core.NewEmail("smtp.gmail.com", os.Getenv("EMAIL"), os.Getenv("EMAIL_PASSWORD"), 587)
	git := handlers.NewGitService(os.Getenv("GIT_CLIENT_ID"), os.Getenv("GIT_CLIENT_SECRET"), os.Getenv("GIT_CALLBACK_URL"), db, client)
	service := handlers.NewService(db, rdb, email, git, client)
	router := gin.Default()
	handlers.SetupHandler(router, service)
	router.Run("localhost:8080")

}
