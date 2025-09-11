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
	router.GET("/forgot-password", ForgotPassword)
	router.GET("/verify-otp", VerifyOTP)

	protectedUserRoutes := router.Group("/api/v1/user")
	protectedPostRoutes := router.Group("/api/v1/post")
	protectedUserRoutes.Use(core.Authentication())
	protectedPostRoutes.Use(core.Authentication())

	protectedUserRoutes.GET("/profile", GetUserInfo)
	protectedUserRoutes.GET("/view/:name", ViewUser)
	protectedUserRoutes.GET("/view-user-req", ViewFriendReq)
	protectedUserRoutes.GET("/view-user-friend", ViewUserFriends)
	protectedUserRoutes.POST("/send-user-req", SendFriendReq)
	protectedUserRoutes.PUT("/update-profile", UpdateUserInfo)
	protectedUserRoutes.PATCH("/update-user-req", UpdateFriendReqStatus)
	protectedUserRoutes.PATCH("/update-password", UpdateUserPassword)
	protectedUserRoutes.PATCH("/update-availability/:status", UpdateUserAvaibilityStatus)
	protectedUserRoutes.PATCH("/update-skills", UpdateUserSkills)
	protectedUserRoutes.PATCH("/reset-password", ResetPassword)
	protectedUserRoutes.DELETE("/delete-skills", DeleteUserSkills)
	protectedUserRoutes.DELETE("/delete-user", DeleteUserAccount)
	protectedUserRoutes.DELETE("/delete-friend-req", DeleteSentReq)
	protectedUserRoutes.DELETE("/delete-user-friend", DeleteUserFriend)

	protectedUserRoutes.GET("/view-message", ViewMessages)
	protectedUserRoutes.POST("/send-message", CreateMessage)
	protectedUserRoutes.PATCH("/edit-message", EditMessage)
	protectedUserRoutes.DELETE("/delete-message", DeleteMessage)

	protectedPostRoutes.GET("/posts/all", GetPosts)
	protectedPostRoutes.GET("/:id", ViewPost)
	protectedPostRoutes.POST("/create", CreatePost)
	protectedPostRoutes.PUT("/edit/:id", EditPost)
	protectedPostRoutes.PATCH("/edit-view/:id", EditPostView)
	protectedPostRoutes.DELETE("/delete/:id", DeletePost)
}
