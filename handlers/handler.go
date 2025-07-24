package handlers

import (
	"findme/core"

	"github.com/gin-gonic/gin"
)

func SetupHandler(router *gin.Engine) {

	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "APP is up and running"})
	})

	// User Endpoints
	router.POST("/signup", AddUser)
	router.POST("/login", VerifyUser)

	protectedUserRoutes := router.Group("/user")
	protectedUserRoutes.Use(core.Authentication())

	protectedUserRoutes.GET("/profile", GetUserInfo)
	protectedUserRoutes.PUT("/update-profile", UpdateUserInfo)

}

