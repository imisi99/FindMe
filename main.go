package main

import (
	"findme/database"

	"github.com/gin-gonic/gin"
)



func main() {
	database.Connect()
	
	router := gin.Default()

	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "APP is up and running"})
	})


	router.Run("localhost:8080")
}