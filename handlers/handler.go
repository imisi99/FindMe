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
	Transc Transc
	Emb    core.Embedding
	Rec    core.Recommendation
	Chat   *core.ChatHub
	Client *http.Client
}

func NewService(db core.DB, rdb core.Cache, email core.Email, git Git, transc Transc, embHub core.Embedding, recHub core.Recommendation, client *http.Client, chat *core.ChatHub) *Service {
	return &Service{DB: db, RDB: rdb, Email: email, Git: git, Transc: transc, Emb: embHub, Rec: recHub, Client: client, Chat: chat}
}

func SetupHandler(router *gin.Engine, service *Service) {
	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "APP is up and running"})
	})

	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/health/detailed", service.DetailedHealth)

	router.POST("/signup", service.AddUser)
	router.POST("/login", service.VerifyUser)

	router.GET("/github-signup", service.Git.GitHubAddUser)
	router.GET("/api/v1/auth/github/callback", service.Git.SelectCallback)

	router.POST("/forgot-password", service.ForgotPassword)
	router.GET("/verify-otp", service.VerifyOTP)

	router.POST("/api/transc/webhook", service.Transc.VerifyTranscWebhook)

	protectedUserRoutes := router.Group("/api/user")
	protectedProjectRoutes := router.Group("/api/post")
	protectedMsgRoutes := router.Group("/api/msg")
	protectedTranscRoutes := router.Group("/api/transc")
	protectedUserRoutes.Use(service.Authentication())
	protectedProjectRoutes.Use(service.Authentication())
	protectedMsgRoutes.Use(service.Authentication())
	protectedTranscRoutes.Use(service.Authentication())

	protectedUserRoutes.GET("/profile", service.GetUserInfo)
	protectedUserRoutes.POST("/search", service.ViewUserbySkills)
	protectedUserRoutes.GET("/view", service.ViewUser)
	protectedUserRoutes.GET("/get-user", service.GetUser)
	protectedUserRoutes.GET("/view-git", service.ViewGitUser)
	protectedUserRoutes.GET("/view-user-req", service.ViewFriendReq)
	protectedUserRoutes.GET("/view-friend", service.ViewUserFriends)
	protectedUserRoutes.GET("/view-user-friend", service.ViewUserFriendsByID)
	protectedUserRoutes.GET("/view-repo", service.Git.ViewRepo)
	protectedUserRoutes.GET("/recommend", service.RecommendProjects)
	protectedUserRoutes.GET("/view-subs", service.ViewSubscriptions)
	protectedUserRoutes.POST("/send-user-req", service.SendFriendReq)
	protectedUserRoutes.POST("/connect-github", service.Git.ConnectGitHub)
	protectedUserRoutes.PUT("/update-profile", service.UpdateUserInfo)
	protectedUserRoutes.PATCH("/update-user-req", service.UpdateFriendReqStatus)
	protectedUserRoutes.PATCH("/update-password", service.UpdateUserPassword)
	protectedUserRoutes.PATCH("/update-availability/:status", service.UpdateUserAvaibilityStatus)
	protectedUserRoutes.PATCH("/update-bio", service.UpdateUserBio)
	protectedUserRoutes.PATCH("/update-interest", service.UpdateUserInterests)
	protectedUserRoutes.PATCH("/update-skills", service.UpdateUserSkills)
	protectedUserRoutes.PATCH("/reset-password", service.ResetPassword)
	protectedUserRoutes.DELETE("/delete-skills", service.DeleteUserSkills)
	protectedUserRoutes.DELETE("/delete-user", service.DeleteUserAccount)
	protectedUserRoutes.DELETE("/delete-friend-req", service.DeleteSentReq)
	protectedUserRoutes.DELETE("/delete-user-friend", service.DeleteUserFriend)

	protectedTranscRoutes.GET("/view", service.Transc.GetTransactions)
	protectedTranscRoutes.GET("/initialize", service.Transc.InitializeTransaction)
	protectedTranscRoutes.GET("/view/plans", service.Transc.ViewPlans)
	protectedTranscRoutes.GET("/update-card", service.Transc.UpdateSubscriptionCard)
	protectedTranscRoutes.PATCH("/cancel-sub", service.Transc.CancelSubscription)
	protectedTranscRoutes.PATCH("/enable-sub", service.Transc.EnableSubscription)

	protectedMsgRoutes.GET("/ws/chat", service.WSChat)

	protectedMsgRoutes.GET("/view-hist", service.ViewMessages)
	protectedMsgRoutes.GET("/view-chats", service.FetchUserChats)
	protectedMsgRoutes.GET("/open-chat", service.OpenChat)
	protectedMsgRoutes.POST("/send-message", service.CreateMessage)
	protectedMsgRoutes.PUT("/add-user", service.AddUserToChat)
	protectedMsgRoutes.PATCH("/edit-message", service.EditMessage)
	protectedMsgRoutes.PATCH("/rename-chat", service.RenameChat)
	protectedMsgRoutes.PATCH("/transfer-owner", service.TransferOwner)
	protectedMsgRoutes.DELETE("/delete-message", service.DeleteMessage)
	protectedMsgRoutes.DELETE("/remove-user", service.RemoveUserChat)
	protectedMsgRoutes.DELETE("/leave-chat", service.LeaveChat)
	protectedMsgRoutes.DELETE("/delete-chat", service.DeleteChat)

	protectedProjectRoutes.GET("/posts/all", service.GetProjects)
	protectedProjectRoutes.GET("/view", service.ViewProject)
	protectedProjectRoutes.GET("/view-applications", service.ViewProjectApplications)
	protectedProjectRoutes.GET("/view-application", service.ViewSingleProjectApplication)
	protectedProjectRoutes.GET("/view/saved-post", service.ViewSavedProject)
	protectedProjectRoutes.GET("/recommend", service.RecommendUsers)
	protectedProjectRoutes.POST("/tags", service.SearchProject)
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
