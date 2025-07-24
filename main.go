package main

import (
	"findme/database"
	"findme/handlers"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("[WARNING] Error loading .env file ->", err, "Ignore if in production")
	}
	database.Connect()

	router := gin.Default()
	handlers.SetupHandler(router)
	router.Run("localhost:8080")

}
