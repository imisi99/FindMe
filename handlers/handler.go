package handlers

import (
	"net/http"

	"findme/core"

	"github.com/gin-gonic/gin"
)

type Service struct {
	DB     core.GDB
	RDB    core.Cache
	Email  core.Email
	Git    Git
	Client *http.Client
}

func NewService(db core.GDB, rdb core.Cache, email core.Email, git Git, client *http.Client) *Service {
	return &Service{DB: db, RDB: rdb, Email: email, Git: git, Client: client}
}

func SetupHandler(router *gin.Engine, service *Service) {
	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "APP is up and running"})
	})

	// User Endpoints
	router.POST("/signup", service.AddUser)
	router.POST("/login", service.VerifyUser)

	router.GET("/github-signup", service.Git.GitHubAddUser)
	router.GET("/api/v1/auth/github/callback", service.Git.GitHubAddUserCallback)

	router.GET("/forgot-password", service.ForgotPassword)
	router.GET("/verify-otp", service.VerifyOTP)

	protectedUserRoutes := router.Group("/api/user")
	protectedPostRoutes := router.Group("/api/post")
	protectedUserRoutes.Use(service.Authentication())
	protectedPostRoutes.Use(service.Authentication())

	protectedUserRoutes.GET("/profile", service.GetUserInfo)
	protectedUserRoutes.GET("/search", service.ViewUserbySkills)
	protectedUserRoutes.GET("/view", service.ViewUser)
	protectedUserRoutes.GET("/view-git", service.ViewGitUser)
	protectedUserRoutes.GET("/view-user-req", service.ViewFriendReq)
	protectedUserRoutes.GET("/view-user-friend", service.ViewUserFriends)
	protectedUserRoutes.POST("/send-user-req", service.SendFriendReq)
	protectedUserRoutes.PUT("/update-profile", service.UpdateUserInfo)
	protectedUserRoutes.PATCH("/update-user-req", service.UpdateFriendReqStatus)
	protectedUserRoutes.PATCH("/update-password", service.UpdateUserPassword)
	protectedUserRoutes.PATCH("/update-availability/:status", service.UpdateUserAvaibilityStatus)
	protectedUserRoutes.PATCH("/update-skills", service.UpdateUserSkills)
	protectedUserRoutes.PATCH("/reset-password", service.ResetPassword)
	protectedUserRoutes.DELETE("/delete-skills", service.DeleteUserSkills)
	protectedUserRoutes.DELETE("/delete-user", service.DeleteUserAccount)
	protectedUserRoutes.DELETE("/delete-friend-req", service.DeleteSentReq)
	protectedUserRoutes.DELETE("/delete-user-friend", service.DeleteUserFriend)

	protectedUserRoutes.GET("/view-message", service.ViewMessages)
	protectedUserRoutes.POST("/send-message", service.CreateMessage)
	protectedUserRoutes.PATCH("/edit-message", service.EditMessage)
	protectedUserRoutes.DELETE("/delete-message", service.DeleteMessage)

	protectedPostRoutes.GET("/posts/all", service.GetPosts)
	protectedPostRoutes.GET("/:id", service.ViewPost)
	protectedPostRoutes.GET("/view-applications", service.ViewPostApplications)
	protectedPostRoutes.GET("/view/saved-post", service.ViewSavedPost)
	protectedPostRoutes.GET("/tags", service.SearchPost)
	protectedPostRoutes.POST("/create", service.CreatePost)
	protectedPostRoutes.POST("/apply", service.ApplyForPost)
	protectedPostRoutes.PUT("/save-post", service.SavePost)
	protectedPostRoutes.PUT("/edit/:id", service.EditPost)
	protectedPostRoutes.PATCH("/edit-view/:id", service.EditPostView)
	protectedPostRoutes.PATCH("/edit-status", service.EditPostAvailability)
	protectedPostRoutes.PATCH("/update-application", service.UpdatePostApplication)
	protectedPostRoutes.DELETE("/delete-application", service.DeletePostApplication)
	protectedPostRoutes.DELETE("/delete/:id", service.DeletePost)
}
