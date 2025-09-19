package handlers

import (
	"errors"
	"findme/core"
	"findme/model"
	"findme/schema"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)


// Sign up endpoint for user
func (u *Service) AddUser(ctx *gin.Context) {
	var payload schema.SignupRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload."})
		return
	}

	// Checking for existing username | email 
	var existingUser model.User
	var err error
	if err = u.DB.Where("username = ? OR email = ?", payload.UserName, payload.Email).First(&existingUser).Error; err == nil {
		if existingUser.Email == payload.Email {
			ctx.JSON(http.StatusConflict, gin.H{"message": "Email already in use!"})
			return
		}else {
			ctx.JSON(http.StatusConflict, gin.H{"message": "Username already in use!"})
			return
		}
	}
	
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})
		return
	}
	
	var allskills []*model.Skill
	if len(payload.Skills) > 0 {
		for i := range payload.Skills {payload.Skills[i] = strings.ToLower(payload.Skills[i])}
		allskills, err = u.CheckAndUpdateSkills(payload.Skills)
		if err != nil {
		log.Printf("Failed to create skills for new user -> %s", err)
		}
	}

	hashedPassword, err := core.HashPassword(payload.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to encrypt the user password."})
		return
	}

	user := model.User{
		FullName: payload.FullName,
		UserName: payload.UserName,
		Email: payload.Email,
		Password: hashedPassword,
		Bio: payload.Bio,
		GitUser: false,

		Skills: allskills,
		Availability: true,
	}

	if err = u.DB.Create(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create user."})
		return
	}

	jwtToken, err := GenerateJWT(user.ID, "login", JWTExpiry)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate jwt token.", "detail": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "Signed up successfully.", "token": jwtToken})
}



// Log in endpoint for user
func (u *Service) VerifyUser(ctx *gin.Context) {
	var payload schema.LoginRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload."})
		return
	}

	jwtToken, err := u.Authorization(payload.UserName, payload.Password)
	if err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Logged in successfully.", "token": jwtToken})
}


// Get user info enpoint 
func (u *Service) GetUserInfo(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")

	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := u.DB.Preload("Skills").Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	var skills []string
	for _, skill := range user.Skills {skills = append(skills, skill.Name)}

	var gitusername *string

	profile := schema.UserProfileResponse{
		UserName: user.UserName,
		FullName: user.FullName,
		Email: user.Email,
		GitUserName: gitusername,
		Gituser: user.GitUser,
		Bio: user.Bio,
		Availability: user.Availability,
		Skills: skills,
	}

	ctx.JSON(http.StatusOK, gin.H{"user": profile})	
}


// search for user with username endpoint
func (u *Service) ViewUser(ctx *gin.Context) {
	uid, tp, username := ctx.GetUint("userID"), ctx.GetString("purpose"), ctx.Query("id")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := u.DB.Preload("Skills").Preload("Posts").Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "No user found with this username."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	var skills []string
	for _, skill := range user.Skills {skills = append(skills, skill.Name)}
	userprofile := schema.UserProfileResponse{
		UserName: user.UserName,
		FullName: user.FullName,
		GitUserName: user.GitUserName,
		Gituser: user.GitUser,
		Bio: user.Bio,
		Email: user.Email,
		Skills: skills,
		Availability: user.Availability,
	}

	var posts []schema.PostResponse
	for _, post := range user.Posts {
		var tags []string
		for _, tag := range post.Tags {tags = append(tags, tag.Name)}
		posts = append(posts, schema.PostResponse{
			ID: post.ID,
			Description: post.Description,
			Tags: tags,
			CreatedAt: post.CreatedAt,
			UpdatedAt: post.UpdatedAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"user": userprofile, "posts": posts})
}


// Search for user with github username endpoint
func (u *Service) ViewGitUser(ctx *gin.Context) {
	uid, tp, username := ctx.GetUint("userID"), ctx.GetString("purpose"), ctx.Query("id")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := u.DB.Preload("Posts").Preload("Skills").Where("gitusername = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "No user found with this git username."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	var skills []string
	for _, skill := range user.Skills {skills = append(skills, skill.Name)}

	profile := schema.UserProfileResponse{
		UserName: user.UserName,
		FullName: user.FullName,
		GitUserName: user.GitUserName,
		Gituser: user.GitUser,
		Bio: user.Bio,
		Email: user.Email,
		Skills: skills,
		Availability: user.Availability,
	}

	var posts []schema.PostResponse
	for _, post := range user.Posts {
		var tags []string
		for _, tag := range post.Tags {tags = append(tags, tag.Name)}
		posts = append(posts, schema.PostResponse{
			ID: post.ID,
			Description: post.Description,
			Tags: tags,
			CreatedAt: post.CreatedAt,
			UpdatedAt: post.UpdatedAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"user": profile, "posts": posts})
}


// Search for user by skills endpoint 
func (u *Service) ViewUserbySkills(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var payload schema.SearchUserbySkills
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload"})
		return
	}

	var users []model.User
	subquery := u.DB.Select("user_id").
		Table("user_skills").
		Joins("JOIN skills s ON user_skills.skill_id = s.id").
		Where("s.name IN ?", payload.Skills)

	if err := u.DB.Preload("Skills").Where("id IN (?)", subquery).Find(&users).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve users from db."})
		return
	}

	var profiles []schema.SearchUser
	for _, user := range users {
		var skills []string
		for _, skill := range user.Skills {skills = append(skills, skill.Name)}
		profiles = append(profiles, schema.SearchUser{
			UserName: user.UserName,
			Bio: user.Bio,
			Availability: user.Availability,
			GitUser: user.GitUser,
			GitUserName: user.GitUserName,
			Skills: skills,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"profiles": profiles})
}


// Send friend request endpoint 
func (u *Service) SendFriendReq(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var payload schema.SendFriendReq
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload."})
		return
	}

	var friend, user model.User
	if err := u.DB.Preload("Friends").Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	if err := u.DB.Where("username = ?", payload.UserName).First(&friend).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Friend not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retreive user from db."})}
		return
	}

	for _, fr := range user.Friends {
		if fr.ID == friend.ID {
			ctx.JSON(http.StatusConflict, gin.H{"message": "User is already your friend."})
			return
		}
	}

	if friend.ID == user.ID {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "You can't friend yourself."})
		return
	}

	var existingreq model.FriendReq
	var err error
	if err = u.DB.Where("user_id = ?", user.ID).Where("friend_id = ?", friend.ID).First(&existingreq).Error; err == nil {
		ctx.JSON(http.StatusConflict, gin.H{"message": "You have to delete the previous request to this user to send another."})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve request from db."})
		return
	}
	if err = u.DB.Where("user_id = ?", friend.ID).Where("friend_id = ?", user.ID).First(&existingreq).Error; err == nil {
		ctx.JSON(http.StatusConflict, gin.H{"message": "This user has already sent you a request."})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve request from db."})
		return
	}

	req := model.FriendReq{
		UserFriend: model.UserFriend{
			UserID: user.ID,
			FriendID: friend.ID,
		},
	}

	if len(payload.Message) > 0 {req.Message = payload.Message}

	if err := u.DB.Create(&req).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to send request."})
		return
	}
	
	u.Email.SendFriendReqEmail(friend.Email, user.UserName, friend.UserName, req.Message, "")

	ctx.JSON(http.StatusOK, gin.H{"message": "Friend request sent successfully."})
}


// View friend requests endpoint 
func (u *Service) ViewFriendReq(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := u.DB.Preload("FriendReq.Friend").Preload("RecFriendReq.User").Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	var sentRec, recReq []schema.FriendReqStatus
	for _, fr := range user.FriendReq {
		sentRec = append(sentRec, schema.FriendReqStatus{
			ID: fr.ID,
			Status: fr.Status,
			Username: fr.Friend.UserName,
			Message: fr.Message,
		})
	}

	for _, fr := range user.RecFriendReq {
		recReq = append(recReq, schema.FriendReqStatus{
			ID: fr.ID,
			Status: fr.Status,
			Username: fr.User.UserName,
			Message: fr.Message,
		})
	}
	
	ctx.JSON(http.StatusOK, gin.H{"sent_req": sentRec, "rec_req": recReq})
}


// Update friend request endpoint
func (u *Service) UpdateFriendReqStatus(ctx *gin.Context) {
	uid, tp, reqID, status := ctx.GetUint("userID"), ctx.GetString("purpose"), ctx.Query("id"), ctx.Query("status")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	rid, err := strconv.ParseUint(reqID, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Invalid request id."})
		return
	}

	var req model.FriendReq
	if err := u.DB.Where("id = ?", uint(rid)).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Request not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve request from db."})}
	}

	var user model.User
	if err := u.DB.Preload("Friends").Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message":"User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	if req.FriendID != user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You can't update the status of this request."})
		return
	}
	
	var friend model.User
	if err := u.DB.Preload("Friends").Where("id = ?", req.UserID).First(&friend).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Friend not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve friend from db."})}
		return
	}

	switch status {
		case model.StatusRejected:
			if err := u.DB.Model(&req).Update("Status", model.StatusRejected).Error; err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to reject request."})
				return
			}
		case model.StatusAccepted:
			if err := u.DB.Transaction(func(tx *gorm.DB) error {
				if err := tx.Unscoped().Delete(&req).Error; err != nil {return err}

				if err := tx.Model(&user).Association("Friends").Append(&friend); err != nil {return err}

				if err := tx.Model(&friend).Association("Friends").Append(&user); err != nil {return err}

				return nil
			}); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update request status."})
				return
			}
		default:
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid status."})
			return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "Status updated successfully."})
}


// Delete sent request endpoint
func (u *Service) DeleteSentReq(ctx *gin.Context) {
	uid, tp, reqID := ctx.GetUint("userID"), ctx.GetString("purpose"), ctx.Query("id")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	rid, err := strconv.ParseUint(reqID, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Invalid request id."})
		return
	}

	var req model.FriendReq
	if err := u.DB.Where("id = ?", uint(rid)).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Request not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve request from db."})}
		return
	}

	var user model.User
	if err := u.DB.Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	if req.UserID != user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You can't delete this request."})
		return
	}

	if err := u.DB.Unscoped().Delete(req).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete req."})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}


// Remove friend endpoint
func (u *Service) DeleteUserFriend(ctx *gin.Context) {
	uid, tp, username := ctx.GetUint("userID"), ctx.GetString("purpose"), ctx.Query("id")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := u.DB.Preload("Friends").Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
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
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "User is not your friend."})
		return
	}

	if err := u.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Association("Friends").Delete(friend); err != nil {return err}

		if err := tx.Model(friend).Association("Friends").Delete(&user); err != nil {return err}

		return nil
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to remove friend"})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}


// view all user friend endpoint 
func (u *Service) ViewUserFriends(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := u.DB.Preload("Friends").Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch user info"})
		return
	}

	var friends []schema.ViewFriends
	for _, fr := range user.Friends {
		friends = append(friends, schema.ViewFriends{
			Username: fr.UserName,
			Bio: fr.Bio,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"friends": friends})
}


// send otp for password reset endpoint 
func (u *Service) ForgotPassword(ctx *gin.Context) {
	var payload schema.ForgotPasswordEmail
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload."})
		return
	}

	var user model.User
	if err := u.DB.Where("email = ?", payload.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User record not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to recieve user from db."})}
		return
	}

	token := core.GenerateOTP()
	if err := u.RDB.SetOTP(token, user.ID); err != nil {
		log.Printf("Failed to store token in redis -> %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to store otp"})
		return
	}

	if err := u.Email.SendForgotPassEmail(user.Email, user.UserName, token); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to send email"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Email sent successfully."})
}


// verify otp for password reset endpoint
func (u *Service) VerifyOTP(ctx *gin.Context) {
	var payload schema.VerifyOTP

	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload"})
		return
	}

	var token schema.OTPInfo
	err := u.RDB.GetOTP(payload.Token, &token)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Invalid token."})
		return
	}

	jwt, err := GenerateJWT(token.UserID, "reset", JWTRExpiry)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create jwt token"})
		return
	}
	
	ctx.JSON(http.StatusOK, gin.H{"message": "otp verified", "token": jwt})
}


// Actual reset password endpoint
func (u *Service) ResetPassword(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "reset" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user"})
		return
	}

	var payload schema.ResetPassword
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload"})
		return
	}

	var user model.User
	if err := u.DB.Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
			ctx.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	hashed, err := core.HashPassword(payload.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Unable to hash password"})
		return
	}

	user.Password = hashed

	if err := u.DB.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to store new password."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "password reset successfully."})	
}


// Update user info endpoint 
func (u *Service) UpdateUserInfo(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user, existingUser model.User
	if err := u.DB.Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	var payload schema.UserProfileRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload."})
		return
	}

	var err error
	if err = u.DB.Where("username = ? OR email = ?", payload.UserName, payload.Email).First(&existingUser).Error; err == nil && existingUser.ID != uid {
		if existingUser.Email == payload.Email {
			ctx.JSON(http.StatusConflict, gin.H{"message": "Email already in use!"})
		}else {
			ctx.JSON(http.StatusConflict, gin.H{"message": "Username already in use!"})
		}
		return

	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) { 
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})
		return
	}

	user.Email = payload.Email
	user.Bio = payload.Bio
	user.FullName = payload.FullName
	user.UserName = payload.UserName

	if err := u.DB.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user profile."})
		return
	}

	profile := schema.UserProfileResponse{
		UserName: user.UserName,
		FullName: user.FullName,
		Email: user.Email,
		Bio: user.Bio,
		GitUserName: user.GitUserName,
		Gituser: user.GitUser,
		Availability: user.Availability,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "User profile updated successfully.", "user": profile})
}


// Update user password endpoint
func (u *Service) UpdateUserPassword(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := u.DB.Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "user not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	var payload schema.UpdatePassword
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload."})
		return
	}

	if err := core.VerifyHashedPassword(payload.FormerPassword, user.Password); err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	hashed, err := core.HashPassword(payload.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate hash for user password"})
		return
	}
	user.Password = hashed

	if err := u.DB.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user password."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "User password updated successfully."})
}


// Update user avaibility status endpoint 
func (u *Service) UpdateUserAvaibilityStatus(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := u.DB.Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	status := ctx.Param("status")
	statusbool, err := strconv.ParseBool(status)
	if  err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Availability status can only be true or false."})
		return
	}

	user.Availability = statusbool

	if err := u.DB.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user availability."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "User availability updated successfully."})
}


// Update user skills endpoint
func (u *Service) UpdateUserSkills(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := u.DB.Preload("Skills").Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	var payload schema.UpdateUserSkillsRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload."})
		return
	}

	for i := range payload.Skills {payload.Skills[i] = strings.ToLower(payload.Skills[i])}
	allskills, err := u.CheckAndUpdateSkills(payload.Skills)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user skills."})
		return
	}
	
	if err := u.DB.Model(&user).Association("Skills").Replace(allskills); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user skills."})
		return
	}
	
	ctx.JSON(http.StatusAccepted, gin.H{"message": "User skills updated successfully.", "user": payload.Skills})
}


// Delete user skills endpoint
func (u *Service) DeleteUserSkills(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := u.DB.Preload("Skills").Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	var payload schema.DeleteUserSkillsRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload."})
		return
	}

	userSkillSet := make(map[string]*model.Skill)
	for _, skill := range user.Skills {userSkillSet[skill.Name] = skill}
	var skillsToDelete []*model.Skill
	for _, skill := range payload.Skills {
		if delete, exists := userSkillSet[strings.ToLower(skill)]; exists {
			skillsToDelete = append(skillsToDelete, delete)
		}
	}

	if err := u.DB.Model(&user).Association("Skills").Delete(skillsToDelete); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete user skills."})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}


// Delete user account endpoint using soft deleting 
func (u *Service) DeleteUserAccount(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := u.DB.Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db"})}
		return
	}

	if err := u.DB.Delete(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete user account"})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
