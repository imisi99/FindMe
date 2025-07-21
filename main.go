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
	err := godotenv.Load()
	if err != nil {
		log.Println("[WARNING] Error loading .env file ->", err, "Ignore if in production")
	}
	database.Connect()
	
	router := gin.Default()

	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "APP is up and running"})
	})

	// User Endpoints
	router.POST("/signup", handlers.AddUser)
	router.POST("/login", handlers.VerifyUser)

	protectedUserRoutes := router.Group("/user")
	protectedUserRoutes.Use(core.Authentication())

	protectedUserRoutes.GET("/profile", handlers.GetUserInfo)
	protectedUserRoutes.PUT("/update-profile", handlers.UpdateUserInfo)

	
	router.Run("localhost:8080")
}