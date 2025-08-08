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
	router.GET("/github-signup", GitHubAddUser)
	router.GET("/api/v1/auth/github/callback", GitHubAddUserCallback)

	protectedUserRoutes := router.Group("/api/v1/user")
	protectedUserRoutes.Use(core.Authentication())

	protectedUserRoutes.GET("/profile", GetUserInfo)
	protectedUserRoutes.PUT("/update-profile", UpdateUserInfo)
	protectedUserRoutes.PATCH("/update-availability/:status", UpdateUserAvaibilityStatus)
	protectedUserRoutes.PATCH("/update-skills", UpdateUserSkills)
	protectedUserRoutes.DELETE("/delete-skills", DeleteUserSkills)
	protectedUserRoutes.DELETE("/delete-user", DeleteUserAccount)
}
