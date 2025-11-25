package handlers

import (
	"net/http"

	"findme/core"

	"github.com/gin-gonic/gin"
)

type Service struct {
	DB     core.DB
	RDB    core.Cache
	Email  core.Email
	Git    Git
	Hub    *core.Hub
	Client *http.Client
}

func NewService(db core.DB, rdb core.Cache, email core.Email, git Git, client *http.Client, hub *core.Hub) *Service {
	return &Service{DB: db, RDB: rdb, Email: email, Git: git, Client: client, Hub: hub}
}

func SetupHandler(router *gin.Engine, service *Service) {
	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "APP is up and running"})
	})

	router.POST("/signup", service.AddUser)
	router.POST("/login", service.VerifyUser)

	router.GET("/github-signup", service.Git.GitHubAddUser)
	router.GET("/api/v1/auth/github/callback", service.Git.SelectCallback)

	router.GET("/forgot-password", service.ForgotPassword)
	router.GET("/verify-otp", service.VerifyOTP)

	protectedUserRoutes := router.Group("/api/user")
	protectedProjectRoutes := router.Group("/api/post")
	protectedMsgRoutes := router.Group("/api/msg")
	protectedUserRoutes.Use(service.Authentication())
	protectedProjectRoutes.Use(service.Authentication())
	protectedMsgRoutes.Use(service.Authentication())

	protectedUserRoutes.GET("/profile", service.GetUserInfo)
	protectedUserRoutes.GET("/search", service.ViewUserbySkills)
	protectedUserRoutes.GET("/view", service.ViewUser)
	protectedUserRoutes.GET("/get-user", service.GetUser)
	protectedUserRoutes.GET("/view-git", service.ViewGitUser)
	protectedUserRoutes.GET("/view-user-req", service.ViewFriendReq)
	protectedUserRoutes.GET("/view-user-friend", service.ViewUserFriends)
	protectedUserRoutes.GET("/view-repo", service.Git.ViewRepo)
	protectedUserRoutes.POST("/send-user-req", service.SendFriendReq)
	protectedUserRoutes.POST("/connect-github", service.Git.ConnectGitHub)
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

	protectedMsgRoutes.GET("/ws/chat", service.WSChat)

	protectedMsgRoutes.GET("/view-hist", service.ViewMessages)
	protectedMsgRoutes.GET("/view-chats", service.FetchUserChats)
	protectedMsgRoutes.GET("/open-chat", service.OpenChat)
	protectedMsgRoutes.POST("/send-message", service.CreateMessage)
	protectedMsgRoutes.PUT("/add-user", service.AddUserToChat)
	protectedMsgRoutes.PATCH("/edit-message", service.EditMessage)
	protectedMsgRoutes.PATCH("/rename-chat", service.RenameChat)
	protectedMsgRoutes.DELETE("/delete-message", service.DeleteMessage)
	protectedMsgRoutes.DELETE("/remove-user", service.RemoveUserChat)
	protectedMsgRoutes.DELETE("/leave-chat", service.LeaveChat)
	protectedMsgRoutes.DELETE("/delete-chat", service.DeleteChat)

	protectedProjectRoutes.GET("/posts/all", service.GetProjects)
	protectedProjectRoutes.GET("/view", service.ViewProject)
	protectedProjectRoutes.GET("/view-applications", service.ViewProjectApplications)
	protectedProjectRoutes.GET("/view-application", service.ViewSingleProjectApplication)
	protectedProjectRoutes.GET("/view/saved-post", service.ViewSavedProject)
	protectedProjectRoutes.GET("/tags", service.SearchProject)
	protectedProjectRoutes.POST("/create", service.CreateProject)
	protectedProjectRoutes.POST("/apply", service.ApplyForProject)
	protectedProjectRoutes.PUT("/save-post", service.SaveProject)
	protectedProjectRoutes.DELETE("/remove-post", service.RemoveSavedProject)
	protectedProjectRoutes.PUT("/edit", service.EditProject)
	protectedProjectRoutes.PATCH("/edit-view", service.EditProjectView)
	protectedProjectRoutes.PATCH("/edit-status", service.EditProjectAvailability)
	protectedProjectRoutes.PATCH("/update-application", service.UpdateProjectApplication)
	protectedProjectRoutes.DELETE("/delete-application", service.DeleteProjectApplication)
	protectedProjectRoutes.DELETE("/clear-application", service.ClearProjectApplication)
	protectedProjectRoutes.DELETE("/delete", service.DeleteProject)
}
