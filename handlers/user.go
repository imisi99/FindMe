package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

// AddUser -> Sign up endpoint for user
func (s *Service) AddUser(ctx *gin.Context) {
	var payload schema.SignupRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
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
		FullName: payload.FullName,
		UserName: payload.UserName,
		Email:    payload.Email,
		Password: hashedPassword,
		Bio:      payload.Bio,
		GitUser:  false,

		Skills:       allskills,
		Availability: true,
	}

	if err := s.DB.AddUser(&user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	jwtToken, err := GenerateJWT(user.ID, "login", JWTExpiry)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to generate jwt token.", "detail": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"msg": "Signed up successfully.", "token": jwtToken})
}

// VerifyUser -> Log in endpoint for user
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

	ctx.JSON(http.StatusOK, gin.H{"msg": "Logged in successfully.", "token": jwtToken})
}

// GetUserInfo ->  user info enpoint
func (s *Service) GetUserInfo(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")

	if uid == "" || tp != "login" {
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
		UserName:     user.UserName,
		FullName:     user.FullName,
		Email:        user.Email,
		GitUserName:  user.GitUserName,
		Gituser:      user.GitUser,
		Bio:          user.Bio,
		Availability: user.Availability,
		Skills:       skills,
	}

	ctx.JSON(http.StatusOK, gin.H{"user": profile})
}

// ViewUser -> search for user with username endpoint
func (s *Service) ViewUser(ctx *gin.Context) {
	uid, tp, username := ctx.GetString("userID"), ctx.GetString("purpose"), ctx.Query("id")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
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
		UserName:     user.UserName,
		FullName:     user.FullName,
		GitUserName:  user.GitUserName,
		Gituser:      user.GitUser,
		Bio:          user.Bio,
		Email:        user.Email,
		Skills:       skills,
		Availability: user.Availability,
	}

	var posts []schema.PostResponse
	for _, post := range user.Posts {
		var tags []string
		for _, tag := range post.Tags {
			tags = append(tags, tag.Name)
		}
		posts = append(posts, schema.PostResponse{
			ID:          post.ID,
			Description: post.Description,
			Tags:        tags,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"user": userprofile, "posts": posts})
}

// ViewGitUser -> Search for user with github username endpoint
func (s *Service) ViewGitUser(ctx *gin.Context) {
	uid, tp, username := ctx.GetString("userID"), ctx.GetString("purpose"), ctx.Query("id")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
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
		UserName:     user.UserName,
		FullName:     user.FullName,
		GitUserName:  user.GitUserName,
		Gituser:      user.GitUser,
		Bio:          user.Bio,
		Email:        user.Email,
		Skills:       skills,
		Availability: user.Availability,
	}

	var posts []schema.PostResponse
	for _, post := range user.Posts {
		var tags []string
		for _, tag := range post.Tags {
			tags = append(tags, tag.Name)
		}
		posts = append(posts, schema.PostResponse{
			ID:          post.ID,
			Description: post.Description,
			Tags:        tags,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"user": profile, "posts": posts})
}

// ViewUserbySkills -> Search for user by skills endpoint
func (s *Service) ViewUserbySkills(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.SearchUserbySkills
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload"})
		return
	}

	var users []model.User
	if err := s.DB.SearchUsersBySKills(&users, payload.Skills); err != nil {
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
			UserName:     user.UserName,
			Bio:          user.Bio,
			Availability: user.Availability,
			GitUser:      user.GitUser,
			GitUserName:  user.GitUserName,
			Skills:       skills,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"profiles": profiles})
}

// SendFriendReq -> friend request endpoint
func (s *Service) SendFriendReq(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.SendFriendReq
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload."})
		return
	}

	var friend, user model.User
	if err := s.DB.FetchUserPreloadFReq(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := s.DB.SearchUser(&friend, payload.UserName); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	for _, fr := range user.Friends {
		if fr.ID == friend.ID {
			ctx.JSON(http.StatusConflict, gin.H{"msg": "User is already your friend."})
			return
		}
	}

	i, j := 0, 0
	for i < len(user.RecFriendReq) || j < len(user.FriendReq) {
		if i < len(user.RecFriendReq) {
			if user.RecFriendReq[i].UserID == friend.ID {
				ctx.JSON(http.StatusConflict, gin.H{"msg": "User has already sent you a friend reqest."})
				return
			}
			i++
		}
		if j < len(user.FriendReq) {
			if user.FriendReq[j].FriendID == friend.ID {
				ctx.JSON(http.StatusConflict, gin.H{"msg": "You have already send this user a friend request."})
				return
			}
			j++
		}
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

	_ = s.Email.SendFriendReqEmail(friend.Email, user.UserName, friend.UserName, req.Message, "")

	ctx.JSON(http.StatusOK, gin.H{"msg": "Friend request sent successfully."})
}

// ViewFriendReq -> friend requests endpoint
func (s *Service) ViewFriendReq(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
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
		})
	}

	for _, fr := range user.RecFriendReq {
		recReq = append(recReq, schema.FriendReqStatus{
			ID:       fr.ID,
			Status:   fr.Status,
			Username: fr.User.UserName,
			Message:  fr.Message,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"sent_req": sentRec, "rec_req": recReq})
}

// UpdateFriendReqStatus -> friend request endpoint
func (s *Service) UpdateFriendReqStatus(ctx *gin.Context) {
	uid, tp, reqID, status := ctx.GetString("userID"), ctx.GetString("purpose"), ctx.Query("id"), ctx.Query("status")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var req model.FriendReq
	if err := s.DB.FetchFriendReq(&req, reqID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadF(&user, uid); err != nil {
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
	if err := s.DB.FetchUserPreloadF(&friend, req.UserID); err != nil {
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

// DeleteSentReq -> sent request endpoint
func (s *Service) DeleteSentReq(ctx *gin.Context) {
	uid, tp, reqID := ctx.GetString("userID"), ctx.GetString("purpose"), ctx.Query("id")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
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

// DeleteUserFriend -> Remove friend endpoint
func (s *Service) DeleteUserFriend(ctx *gin.Context) {
	uid, tp, username := ctx.GetString("userID"), ctx.GetString("purpose"), ctx.Query("id")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadF(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var friend *model.User
	for _, fr := range user.Friends {
		if username == fr.UserName {
			friend = fr
			break
		}
	}
	if friend == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "User is not your friend."})
		return
	}

	if err := s.DB.DeleteFriend(&user, friend); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// ViewUserFriends -> view all user friend endpoint
func (s *Service) ViewUserFriends(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
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
			Username: fr.UserName,
			Bio:      fr.Bio,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"friends": friends})
}

// ForgotPassword -> send otp for password reset endpoint
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

	if err := s.Email.SendForgotPassEmail(user.Email, user.UserName, token); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to send email"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"msg": "Email sent successfully."})
}

// VerifyOTP -> verify otp for password reset endpoint
func (s *Service) VerifyOTP(ctx *gin.Context) {
	var payload schema.VerifyOTP

	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload"})
		return
	}

	uid, err := s.RDB.GetOTP(payload.Token)
	if err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	jwt, err := GenerateJWT(uid, "reset", JWTRExpiry)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to create jwt token"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"msg": "otp verified", "token": jwt})
}

// ResetPassword -> Actual reset password endpoint
func (s *Service) ResetPassword(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "reset" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user"})
		return
	}

	var payload schema.ResetPassword
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload"})
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Unable to hash password"})
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

// UpdateUserInfo -> user info endpoint
func (s *Service) UpdateUserInfo(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
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
	user.Bio = payload.Bio
	user.FullName = payload.FullName
	user.UserName = payload.UserName

	if err := s.DB.SaveUser(&user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	profile := schema.UserProfileResponse{
		UserName:     user.UserName,
		FullName:     user.FullName,
		Email:        user.Email,
		Bio:          user.Bio,
		GitUserName:  user.GitUserName,
		Gituser:      user.GitUser,
		Availability: user.Availability,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "User profile updated successfully.", "user": profile})
}

// UpdateUserPassword -> user password endpoint
func (s *Service) UpdateUserPassword(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.UpdatePassword
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload."})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := core.VerifyHashedPassword(payload.FormerPassword, user.Password); err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	hashed, err := core.HashPassword(payload.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to generate hash for user password"})
		return
	}
	user.Password = hashed

	if err := s.DB.SaveUser(&user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "User password updated successfully."})
}

// UpdateUserAvaibilityStatus -> user avaibility status endpoint
func (s *Service) UpdateUserAvaibilityStatus(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
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

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "User availability updated successfully."})
}

// UpdateUserSkills -> user skills endpoint
func (s *Service) UpdateUserSkills(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.UpdateUserSkillsRequest
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

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "User skills updated successfully.", "user": payload.Skills})
}

// DeleteUserSkills -> user skills endpoint
func (s *Service) DeleteUserSkills(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
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
	for _, skill := range payload.Skills {
		if delete, exists := userSkillSet[strings.ToLower(skill)]; exists {
			skillsToDelete = append(skillsToDelete, delete)
		}
	}

	if err := s.DB.DeleteSkills(&user, skillsToDelete); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// DeleteUserAccount -> user account endpoint using soft deleting
func (s *Service) DeleteUserAccount(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
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

	ctx.JSON(http.StatusNoContent, nil)
}
