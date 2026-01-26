package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

// TODO:
// Add a link to the transaction ID on the subscription

// AddUser godoc
// @Summary			Register a new user
// @Description  Sign up endpoint for new users it internally calls a service to create a vector for the user
// @Tags	Auth
// @Accept  json
// @Produce json
// @Param payload body schema.SignupRequest true "User signup payload"
// @Success 201 {object} schema.DocTokenResponse "jwt token generated"
// @Failure 409 {object} schema.DocNormalResponse "Existing email or username"
// @Failure 422 {object} schema.DocNormalResponse "Failed to parse payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /signup [post]
func (s *Service) AddUser(ctx *gin.Context) {
	var payload schema.SignupRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		log.Println(err)
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
		return
	}

	// Checking for existing username | email
	var existingUser model.User
	if err := s.DB.CheckExistingUser(&existingUser, payload.Email, payload.UserName); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var err error
	var allskills []*model.Skill
	if len(payload.Skills) > 0 {
		for i := range payload.Skills {
			payload.Skills[i] = strings.ToLower(payload.Skills[i])
		}
		allskills, err = s.CheckAndUpdateSkills(payload.Skills)
		if err != nil {
			log.Printf("Failed to create skills for new user -> %s", err)
		}
	}

	hashedPassword, err := core.HashPassword(payload.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to encrypt the user password."})
		return
	}

	user := model.User{
		FullName:  payload.FullName,
		UserName:  payload.UserName,
		Email:     payload.Email,
		Password:  hashedPassword,
		Bio:       payload.Bio,
		Interests: payload.Interests,
		GitUser:   false,
		FreeTrial: time.Now().Add(7 * 24 * time.Hour),

		Skills:       allskills,
		Availability: true,
	}

	if err := s.DB.AddUser(&user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	jwtToken, err := GenerateJWT(user.ID, "login", true, JWTExpiry)
	if err != nil {
		log.Println("[APP] Failed to generate jwt token -> ", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to generate jwt token."})
		return
	}

	s.Emb.QueueUserCreate(user.ID, user.Bio, payload.Skills, user.Interests)

	ctx.JSON(http.StatusCreated, gin.H{"token": jwtToken})
}

// VerifyUser godoc
// @Summary			Log in a user
// @Description  Log in endpoint for existing users
// @Tags 	Auth
// @Accept  json
// @Produce json
// @Param payload body schema.LoginRequest true "User login payload"
// @Success 200 {object} schema.DocTokenResponse "jwt token generated"
// @Failure 422 {object} schema.DocNormalResponse "Failed to parse payload"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /login [post]
func (s *Service) VerifyUser(ctx *gin.Context) {
	var payload schema.LoginRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
		return
	}

	var user model.User
	if err := s.DB.VerifyUser(&user, payload.UserName); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	jwtToken, err := Authorization(&user, payload.Password)
	if err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"token": jwtToken})
}

// GetUser godoc
// @Summary			 Get a user by their ID
// @Description  An endpoint for fetching a user by their ID
// @Tags 		User
// @Accept  json
// @Produce json
// @Param id query string true "User ID"
// @Security BearerAuth
// @Success 200 {object} schema.DocProjectUserResponse "user and projects fetched"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized user"
// @Failure 400 {object} schema.DocNormalResponse "Invalid user id"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/get-user [get]
func (s *Service) GetUser(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	userID := ctx.Query("id")
	if !model.IsValidUUID(userID) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid user id."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadSP(&user, userID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var skills []string
	for _, skill := range user.Skills {
		skills = append(skills, skill.Name)
	}

	profile := schema.UserProfileResponse{
		ID:           user.ID,
		UserName:     user.UserName,
		FullName:     user.FullName,
		Country:      user.Country,
		GitUserName:  user.GitUserName,
		Gituser:      user.GitUser,
		Bio:          user.Bio,
		Availability: user.Availability,
		Skills:       skills,
		Interests:    user.Interests,
	}

	var posts []schema.ProjectResponse
	for _, post := range user.Projects {
		var tags []string
		for _, tag := range post.Tags {
			tags = append(tags, tag.Name)
		}
		posts = append(posts, schema.ProjectResponse{
			ID:          post.ID,
			Title:       post.Title,
			Description: post.Description,
			Available:   post.Availability,
			Tags:        tags,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
			Views:       post.Views,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"user": profile, "posts": posts})
}

// GetUserInfo godoc
// @Summary			 Get the logged in user info
// @Description  An endpoint for fetching the currently logged in user profile details
// @Tags 		User
// @Accept  json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocUserResponse "user fetched"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized user"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/profile [get]
func (s *Service) GetUserInfo(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")

	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadS(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var skills []string
	for _, skill := range user.Skills {
		skills = append(skills, skill.Name)
	}

	profile := schema.UserProfileResponse{
		ID:           user.ID,
		UserName:     user.UserName,
		FullName:     user.FullName,
		Email:        user.Email,
		Country:      user.Country,
		GitUserName:  user.GitUserName,
		Gituser:      user.GitUser,
		Bio:          user.Bio,
		Availability: user.Availability,
		Skills:       skills,
		Interests:    user.Interests,
	}

	ctx.JSON(http.StatusOK, gin.H{"user": profile})
}

// RecommendProjects godoc
// @Summary  Recommends projects for a user to work on
// @Description An endpoint for recommending projects for a user to work on using ai
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocRecProjectsResponse "Projects Retreived"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 402 {object} schema.DocNormalResponse "Payment Required"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/recommend [get]
func (s *Service) RecommendProjects(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	premium := ctx.GetBool("premium")
	if !premium {
		ctx.JSON(http.StatusPaymentRequired, gin.H{"msg": "You need to pay to use this service."})
		return
	}

	rec, err := s.Rec.GetRecommendation(uid, core.ProjectRecommendation)
	if err != nil || rec == nil {
		log.Printf("[gRPC Recommendation] Failed to get recommendation for user -> %v, err -> %v", uid, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to retreive projects for the user."})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var ids []string

	for id := range rec.Res {
		ids = append(ids, id)
	}

	var projects []model.Project
	if err := s.DB.FindProjects(&projects, ids); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var result []schema.RecProjectResponse

	for _, project := range projects {
		score := rec.Res[project.ID]
		var tags []string
		for _, tag := range project.Tags {
			tags = append(tags, tag.Name)
		}
		result = append(result, schema.RecProjectResponse{
			Project: schema.ProjectResponse{
				ID:          project.ID,
				Title:       project.Title,
				Description: project.Description,
				Available:   project.Availability,
				Tags:        tags,
				CreatedAt:   project.CreatedAt,
				UpdatedAt:   project.UpdatedAt,
				Views:       project.Views,
			},
			Score: score,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"projects": result})
}

// ViewUser godoc
// @Summary			 Search for user with their username
// @Description  An endpoint for searching for a user with their username to show their projects and profile
// @Tags 		User
// @Accept  json
// @Produce json
// @Param id query string true "Username"
// @Security BearerAuth
// @Success 200 {object} schema.DocProjectUserResponse "user and projects fetched"
// @Failure 400 {object} schema.DocNormalResponse "Invalid username"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized user"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/view [get]
func (s *Service) ViewUser(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	username := ctx.Query("id")
	if username == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Username not in query."})
		return
	}
	var user model.User
	if err := s.DB.SearchUserPreloadSP(&user, username); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var skills []string
	for _, skill := range user.Skills {
		skills = append(skills, skill.Name)
	}
	userprofile := schema.UserProfileResponse{
		ID:           user.ID,
		UserName:     user.UserName,
		Country:      user.Country,
		GitUserName:  user.GitUserName,
		Gituser:      user.GitUser,
		Bio:          user.Bio,
		Skills:       skills,
		Interests:    user.Interests,
		Availability: user.Availability,
	}

	var posts []schema.ProjectResponse
	for _, post := range user.Projects {
		var tags []string
		for _, tag := range post.Tags {
			tags = append(tags, tag.Name)
		}
		posts = append(posts, schema.ProjectResponse{
			ID:          post.ID,
			Title:       post.Title,
			Description: post.Description,
			Available:   post.Availability,
			Tags:        tags,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"user": userprofile, "posts": posts})
}

// ViewGitUser godoc
// @Summary			 Search for user with their git username
// @Description  An endpoint for searching for a user with their git username to show their projects and profile
// @Tags 		User
// @Accept  json
// @Produce json
// @Param id query string true "Git Username"
// @Security BearerAuth
// @Success 200 {object} schema.DocProjectUserResponse "user and projects fetched"
// @Failure 400 {object} schema.DocNormalResponse "Invalid git username"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized user"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/view-git [get]
func (s *Service) ViewGitUser(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	username := ctx.Query("id")
	if username == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Git username not in query."})
		return
	}

	var user model.User
	if err := s.DB.SearchUserGitPreloadSP(&user, username); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var skills []string
	for _, skill := range user.Skills {
		skills = append(skills, skill.Name)
	}

	profile := schema.UserProfileResponse{
		ID:           user.ID,
		UserName:     user.UserName,
		GitUserName:  user.GitUserName,
		Gituser:      user.GitUser,
		Bio:          user.Bio,
		Email:        user.Email,
		Country:      user.Country,
		Skills:       skills,
		Interests:    user.Interests,
		Availability: user.Availability,
	}

	var posts []schema.ProjectResponse
	for _, post := range user.Projects {
		var tags []string
		for _, tag := range post.Tags {
			tags = append(tags, tag.Name)
		}
		posts = append(posts, schema.ProjectResponse{
			ID:          post.ID,
			Description: post.Description,
			Tags:        tags,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"user": profile, "posts": posts})
}

// ViewUserbySkills godoc
// @Summary			 Search for users by skills/tags
// @Description  An endpoint for searching for users with their skills to show their profiles
// @Tags 		User
// @Accept  json
// @Produce json
// @Param payload body schema.SearchUserbySkills true "Skills"
// @Security BearerAuth
// @Success 200 {object} schema.DocUsersSearch "users fetched"
// @Failure 422 {object} schema.DocNormalResponse "Failed to parse payload"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized user"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/search [post]
func (s *Service) ViewUserbySkills(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.SearchUserbySkills
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
		return
	}

	var users []model.User
	if err := s.DB.SearchUsersBySKills(&users, payload.Skills, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var profiles []schema.SearchUser
	for _, user := range users {
		var skills []string
		for _, skill := range user.Skills {
			skills = append(skills, skill.Name)
		}
		profiles = append(profiles, schema.SearchUser{
			ID:           user.ID,
			UserName:     user.UserName,
			Bio:          user.Bio,
			Availability: user.Availability,
			GitUser:      user.GitUser,
			GitUserName:  user.GitUserName,
			Skills:       skills,
			Interests:    user.Interests,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"users": profiles})
}

// SendFriendReq godoc
// @Summary			 Send a Friend req to a user
// @Description  An endpoint for sending friend requests to users for connecting
// @Tags 		User
// @Accept  json
// @Produce json
// @Param payload body schema.SendFriendReq true "Request"
// @Security BearerAuth
// @Success 200 {object} schema.DocFriendReqStatus "Request sent"
// @Failure 422 {object} schema.DocNormalResponse "Failed to parse payload"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized user"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 409 {object} schema.DocNormalResponse "Existing Friend Req / Friend"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/send-user-req [post]
func (s *Service) SendFriendReq(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.SendFriendReq
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload.."})
		return
	}

	if err, friends := s.DB.CheckExistingFriends(uid, payload.ID); err != nil || friends {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err, exists := s.DB.CheckExistingFriendReq(uid, payload.ID); err != nil || exists {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var friend, user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := s.DB.FetchUser(&friend, payload.ID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	req := model.FriendReq{
		UserFriend: model.UserFriend{
			UserID:   user.ID,
			FriendID: friend.ID,
		},
	}

	if len(payload.Message) > 0 {
		req.Message = payload.Message
	}

	if err := s.DB.AddFriendReq(&req); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	s.Email.QueueFriendReqEmail(user.UserName, friend.UserName, req.Message, "", friend.Email)

	friendReq := schema.FriendReqStatus{
		ID:       req.ID,
		Username: friend.UserName,
		Message:  req.Message,
		Status:   req.Status,
	}

	ctx.JSON(http.StatusOK, gin.H{"req": friendReq})
}

// ViewFriendReq godoc
// @Summary			 View All friend reqs
// @Description  An endpoint for viewing all the looged in users friend reqs (sent and received)
// @Tags 		User
// @Accept  json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocViewFriendReqs "Fetched requests"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized user"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/view-user-req [get]
func (s *Service) ViewFriendReq(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.ViewFriendReq(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var sentRec, recReq []schema.FriendReqStatus
	for _, fr := range user.FriendReq {
		sentRec = append(sentRec, schema.FriendReqStatus{
			ID:       fr.ID,
			Status:   fr.Status,
			Username: fr.Friend.UserName,
			Message:  fr.Message,
			Sent:     fr.CreatedAt,
		})
	}

	for _, fr := range user.RecFriendReq {
		recReq = append(recReq, schema.FriendReqStatus{
			ID:       fr.ID,
			Status:   fr.Status,
			Username: fr.User.UserName,
			Message:  fr.Message,
			Sent:     fr.CreatedAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"sent_req": sentRec, "rec_req": recReq})
}

// UpdateFriendReqStatus godoc
// @Summary			 Update a friend req status
// @Description  An endpoint for updating the status of received friend req to rejected / accepted
// @Tags 		User
// @Accept  json
// @Produce json
// @Param id query string true "Request ID"
// @Security BearerAuth
// @Success 202 {object} schema.DocFriendReqAccept "Request status updated"
// @Failure 400 {object} schema.DocNormalResponse "Invalid status / id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized user"
// @Failure 403 {object} schema.DocNormalResponse "Permission Denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/update-user-req [patch]
func (s *Service) UpdateFriendReqStatus(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	reqID, status := ctx.Query("id"), ctx.Query("status")
	if !model.IsValidUUID(reqID) || status == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid req ID or status"})
		return
	}

	var req model.FriendReq
	if err := s.DB.FetchFriendReq(&req, reqID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if req.FriendID != user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You can't update the status of this request."})
		return
	}

	var friend model.User
	var chat model.Chat
	if err := s.DB.FetchUser(&friend, req.UserID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	switch status {
	case model.StatusRejected:
		if err := s.DB.UpdateFriendReqReject(&req); err != nil {
			cm := err.(*core.CustomMessage)
			ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
			return
		}
	case model.StatusAccepted:
		if err := s.DB.UpdateFriendReqAccept(&req, &user, &friend, &chat); err != nil {
			cm := err.(*core.CustomMessage)
			ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
			return
		}
	default:
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid status."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "Status updated successfully.", "chat_id": chat.ID})
}

// DeleteSentReq godoc
// @Summary			 Delete a sent friend req
// @Description  An endpoint for deleting a sent friend request
// @Tags 		User
// @Accept  json
// @Produce json
// @Param id query string true "Request id"
// @Security BearerAuth
// @Success 204 {object} nil "Request deleted"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized user"
// @Failure 403 {object} schema.DocNormalResponse "Permission Denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/delete-friend-req [delete]
func (s *Service) DeleteSentReq(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	reqID := ctx.Query("id")
	if !model.IsValidUUID(reqID) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid request id."})
		return
	}

	var req model.FriendReq
	if err := s.DB.FetchFriendReq(&req, reqID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if req.UserID != user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You can't delete this request."})
		return
	}

	if err := s.DB.DeleteFriendReq(&req); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// ViewUserFriends godoc
// @Summary			 View all friends for the logged in user
// @Description  An endpoint for viewing all the friends for the currently logged in user
// @Tags 		User
// @Accept  json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocViewFriends "User friends fetched"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized user"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/view-friend [get]
func (s *Service) ViewUserFriends(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadF(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var friends []schema.ViewFriends
	for _, fr := range user.Friends {
		friends = append(friends, schema.ViewFriends{
			ID:       fr.ID,
			Username: fr.UserName,
			Bio:      fr.Bio,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"friends": friends})
}

// ViewUserFriendsByID godoc
// @Summary			 View all friends of a user
// @Description  An endpoint for viewing all the friends of a user
// @Tags 		User
// @Accept  json
// @Produce json
// @Param id query string true "User ID"
// @Security BearerAuth
// @Success 200 {object} schema.DocViewFriends "User friends fetched"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized user"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/view-user-friend [get]
func (s *Service) ViewUserFriendsByID(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	id := ctx.Query("id")
	if !model.IsValidUUID(id) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid user id."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadF(&user, id); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var friends []schema.ViewFriends
	for _, fr := range user.Friends {
		friends = append(friends, schema.ViewFriends{
			ID:       fr.ID,
			Username: fr.UserName,
			Bio:      fr.Bio,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"friends": friends})
}

// DeleteUserFriend godoc
// @Summary			 Delete a existing friendship
// @Description  An endpoint for deleting a friend from the users friend list
// @Tags 		User
// @Accept  json
// @Produce json
// @Param id query string true "User id"
// @Param chat_id query string true "Chat id"
// @Security BearerAuth
// @Success 204 {object} nil "Friend deleted"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized user"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/delete-user-friend [delete]
func (s *Service) DeleteUserFriend(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	id, cid := ctx.Query("id"), ctx.Query("chat_id")
	if !model.IsValidUUID(id) || !model.IsValidUUID(cid) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid friend or chat id."})
		return
	}

	var user, friend model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := s.DB.FetchUser(&friend, id); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var chat model.Chat
	if err := s.DB.FetchChat(cid, &chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := s.DB.DeleteFriend(&user, &friend, &chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// ForgotPassword godoc
// @Summary    Get a OTP for reseting user password
// @Description    An endpoint for getting an otp for reseting the user password
// @Tags  User
// @Accept json
// @Produce json
// @Param payload body schema.ForgotPasswordEmail true "User email"
// @Success 200 {object} schema.DocNormalResponse "Email sent to user"
// @Failure 422 {object} schema.DocNormalResponse "Invalid Email"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /forgot-password [post]
func (s *Service) ForgotPassword(ctx *gin.Context) {
	var payload schema.ForgotPasswordEmail
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
		return
	}

	var user model.User
	if err := s.DB.SearchUserEmail(&user, payload.Email); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	token := core.GenerateOTP()
	if err := s.RDB.SetOTP(token, user.ID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	s.Email.QueueForgotPassEmail(user.Email, user.UserName, token)

	ctx.JSON(http.StatusOK, gin.H{"msg": "Check email for otp."})
}

// VerifyOTP godoc
// @Summary     Verify Sent otp to reset password
// @Description   An endpoint to verify sent otp to create reset jwt token for reseting user password
// @Tags  User
// @Accept json
// @Produce json
// @Param id query string true "otp"
// @Success 200 {object} schema.DocTokenResponse "reset token"
// @Failure 404 {object} schema.DocNormalResponse "invalid otp"
// @Failure 400 {object} schema.DocNormalResponse "invalid otp"
// @Failure 500 {object} schema.DocNormalResponse "server error"
// @Router /verify-otp [get]
func (s *Service) VerifyOTP(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" || len(id) != 6 {
		ctx.JSON(http.StatusBadRequest, "Invalid otp.")
		return
	}

	uid, err := s.RDB.GetOTP(id)
	if err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	jwt, err := GenerateJWT(uid, "reset", false, JWTRExpiry)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to create jwt token."})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"token": jwt})
}

// ResetPassword godoc
// @Summary  Reset user password through forgot password route
// @Description  An endpoint for reseting the user password with a reset jwt token gotten from the verify-otp route
// @Tags User
// @Accept json
// @Produce json
// @Param payload body schema.ResetPassword true "new password"
// @Security BearerAuth
// @Success 202 {object} schema.DocNormalResponse "reset successful"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 422 {object} schema.DocNormalResponse "Invalid password"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "server error"
// @Router /api/user/reset-password [patch]
func (s *Service) ResetPassword(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "reset" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.ResetPassword
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	hashed, err := core.HashPassword(payload.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Unable to hash password."})
		return
	}

	user.Password = hashed

	if err := s.DB.SaveUser(&user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "password reset successfully."})
}

// UpdateUserInfo godoc
// @Summary     Update the current user profile details
// @Description An endpoint for updating the logged-in user profile details
// @Tags User
// @Accept json
// @Produce json
// @Param payload body schema.UserProfileRequest true "new details"
// @Security BearerAuth
// @Success 202 {object} schema.DocUserResponse "User updated"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 409 {object} schema.DocNormalResponse "Existing username / email"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/update-profile [put]
func (s *Service) UpdateUserInfo(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user, existingUser model.User
	var payload schema.UserProfileRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
		return
	}

	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := s.DB.CheckExistingUserUpdate(&existingUser, payload.Email, payload.UserName, user.ID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	user.Email = payload.Email
	user.FullName = payload.FullName
	user.UserName = payload.UserName
	user.Country = payload.Country

	if err := s.DB.SaveUser(&user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	profile := schema.UserProfileResponse{
		ID:           user.ID,
		UserName:     user.UserName,
		FullName:     user.FullName,
		Email:        user.Email,
		Bio:          user.Bio,
		Country:      user.Country,
		GitUserName:  user.GitUserName,
		Gituser:      user.GitUser,
		Availability: user.Availability,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"user": profile})
}

// UpdateUserBio godoc
// @Summary     Update the current user bio
// @Description An endpoint for updating the logged-in user bio information it internally calls a service to update the vector for the user.
// @Tags User
// @Accept json
// @Produce json
// @Param payload body schema.UpdateUserBio true "new details"
// @Security BearerAuth
// @Success 202 {object} schema.DocNormalResponse "User updated"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/update-bio [patch]
func (s *Service) UpdateUserBio(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.UpdateUserBio
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadS(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	user.Bio = payload.Bio
	if err := s.DB.SaveUser(&user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var skills []string
	for _, skill := range user.Skills {
		skills = append(skills, skill.Name)
	}

	s.Emb.QueueUserUpdate(user.ID, user.Bio, skills, user.Interests)

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "Bio updated successfully."})
}

// UpdateUserInterests godoc
// @Summary     Update the current user interests
// @Description An endpoint for updating the logged-in user interests it internally calls a service to update the vector for the user.
// @Tags User
// @Accept json
// @Produce json
// @Param payload body schema.UpdateUserInterests true "new details"
// @Security BearerAuth
// @Success 202 {object} schema.DocNormalResponse "User updated"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/update-interest [patch]
func (s *Service) UpdateUserInterests(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Failed to parse payload."})
		return
	}

	var payload schema.UpdateUserInterests
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadS(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	user.Interests = payload.Interest
	if err := s.DB.SaveUser(&user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var skills []string
	for _, skill := range user.Skills {
		skills = append(skills, skill.Name)
	}

	s.Emb.QueueUserUpdate(user.ID, user.Bio, skills, user.Interests)

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "Interests updated successfully."})
}

// UpdateUserPassword godoc
// @Summary   Update the current user password
// @Description An endpoint for updating the logged-in user password
// @Tags User
// @Accept json
// @Produce json
// @Param payload body schema.UpdatePassword true "new password"
// @Security BearerAuth
// @Success 202 {object} schema.DocNormalResponse "password updated"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "server error"
// @Router /api/user/update-password [patch]
func (s *Service) UpdateUserPassword(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.UpdatePassword
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if user.Password != "" {
		if err := core.VerifyHashedPassword(payload.FormerPassword, user.Password); err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Invalid Password."})
			return
		}
	}

	hashed, err := core.HashPassword(payload.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to generate hash for user password."})
		return
	}
	user.Password = hashed

	if err := s.DB.SaveUser(&user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "Password updated successfully."})
}

// UpdateUserAvaibilityStatus godoc
// @Summary    Update Availability status
// @Description An endpoint for updating the Availability status of the current user to either true or false it internally calls a service to update the user vector payload
// @Tags User
// @Accept json
// @Produce json
// @Param status path string true "Availability status"
// @Security BearerAuth
// @Success 202 {object} schema.DocNormalResponse "Status updated"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "server error"
// @Router /api/user/update-availability/{status} [patch]
func (s *Service) UpdateUserAvaibilityStatus(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	status := ctx.Param("status")
	statusbool, err := strconv.ParseBool(status)
	if err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Availability status can only be true or false."})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	user.Availability = statusbool

	if err := s.DB.SaveUser(&user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	s.Emb.QueueUserUpdateStatus(user.ID, user.Availability)

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "Availability updated successfully."})
}

// UpdateUserSkills godoc
// @Summary     Update User skills
// @Description An endpoint for updating the skills of the current user it internally calls a service to update the user vector
// @Tags User
// @Accept json
// @Produce json
// @Param payload body schema.UpdateUserSkillsRequest true "Skills"
// @Security BearerAuth
// @Success 202 {object} schema.DocSkillsResponse "Skills updated"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 404 {object} schema.DocNormalResponse "record not found"
// @Failure 500 {object} schema.DocNormalResponse "server error"
// @Router /api/user/update-skills [patch]
func (s *Service) UpdateUserSkills(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.UpdateUserSkillsRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	for i := range payload.Skills {
		payload.Skills[i] = strings.ToLower(payload.Skills[i])
	}

	allskills, err := s.CheckAndUpdateSkills(payload.Skills)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to update user skills."})
		return
	}

	if err := s.DB.UpdateSkills(&user, allskills); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	s.Emb.QueueUserUpdate(user.ID, user.Bio, payload.Skills, user.Interests)

	ctx.JSON(http.StatusAccepted, gin.H{"skills": payload.Skills})
}

// DeleteUserSkills godoc
// @Summary    Delete skills from the user skills
// @Description An endpoint to delete some skills from the current user skills it internally calls a service to update the user vector
// @Tags User
// @Accept json
// @Produce json
// @Param payload body schema.DeleteUserSkillsRequest true "Skills"
// @Security BearerAuth
// @Success 204 {object} nil "Skills Deleted"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "server error"
// @Router /api/user/delete-skills [delete]
func (s *Service) DeleteUserSkills(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.DeleteUserSkillsRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadS(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	userSkillSet := make(map[string]*model.Skill)
	for _, skill := range user.Skills {
		userSkillSet[skill.Name] = skill
	}

	var skillsToDelete []*model.Skill
	var skills []string
	for _, skill := range payload.Skills {
		if delete, exists := userSkillSet[strings.ToLower(skill)]; exists {
			skillsToDelete = append(skillsToDelete, delete)
		} else {
			skills = append(skills, skill)
		}
	}

	if err := s.DB.DeleteSkills(&user, skillsToDelete); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	s.Emb.QueueUserUpdate(user.ID, user.Bio, skills, user.Interests)

	ctx.JSON(http.StatusNoContent, nil)
}

// ViewSubscriptions godoc
// @Summary  Views a user subscription history
// @Description An endpoint for viewing the history of a user's subsciption
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocViewSubscriptions "Subscriptions"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/view-subs [get]
func (s *Service) ViewSubscriptions(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadSub(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var response []schema.ViewSubscriptions

	for _, sub := range user.Subscriptions {
		response = append(response, schema.ViewSubscriptions{
			ID:    sub.ID,
			Start: sub.StartDate,
			End:   sub.EndDate,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"subs": response})
}

// DeleteUserAccount godoc
// @Summary    Delete a user account
// @Description An endpoint for deleting the current user account it internally calls a service to delete the vector of the user
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 204 {object} nil "Account deleted"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "server error"
// @Router /api/user/delete-user [delete]
func (s *Service) DeleteUserAccount(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := s.DB.DeleteUser(&user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	s.Emb.QueueUserDelete(user.ID)

	ctx.JSON(http.StatusNoContent, nil)
}
