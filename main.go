package main

import (
	"findme/core"
	"findme/database"
	"findme/handlers"
	"log"

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
	database.Connect()
	database.ConnectRedis()
	core.CacheSkills(database.GetDB(), database.GetRDB())

	router := gin.Default()
	handlers.SetupHandler(router)
	router.Run("localhost:8080")

}
