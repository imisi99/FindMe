package handlers

import (
	"bytes"
	"encoding/json"
	"findme/core"
	"findme/database"
	"findme/model"
	"findme/schema"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Sign up endpoint for user
func AddUser(ctx *gin.Context) {
	db := database.GetDB()
	rdb := database.GetRDB()

	var payload schema.SignupRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload."})
		return
	}

	// Checking for existing username | email 
	var existingUser model.User
	if err := db.Where("username = ? OR email = ?", payload.UserName, payload.Email).First(&existingUser).Error; err == nil {
		if existingUser.Email == payload.Email {
			ctx.JSON(http.StatusConflict, gin.H{"message": "Email already in use!"})
			return
		}else {
			ctx.JSON(http.StatusConflict, gin.H{"message": "Username already in use!"})
			return
		}
	}
	
	var allskills []*model.Skill
	var err error
	if len(payload.Skills) > 0 {
		for i := range payload.Skills {payload.Skills[i] = strings.ToLower(payload.Skills[i])}
		allskills, err = CheckAndUpdateSkills(db, rdb, payload.Skills)
	}

	if err != nil {
		log.Printf("Failed to create skills for new user -> %s", err)
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

	if err = db.Create(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create user."})
		return
	}

	jwtToken, err := core.GenerateJWT(user.ID, "login", core.JWTExpiry)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate jwt token.", "detail": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "Signed up successfully.", "token": jwtToken})
}


// Signing up user using github
func GitHubAddUser(ctx *gin.Context) {
	if _, err := ctx.Cookie("git-access-token"); err == nil {
		ctx.Redirect(http.StatusTemporaryRedirect, "http://localhost:8080/github-signup")
	}
	state, err := core.GenerateState()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate user state."})
		return
	}

	ctx.SetCookie("state", state, 150, "/", "", false, true)

	redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=read:user user:email", os.Getenv("GIT_CLIENT_ID"), state)
	ctx.Redirect(http.StatusTemporaryRedirect, redirectURL)
}


// Callback for the github signup endpoint
func GitHubAddUserCallback(ctx *gin.Context) {
	var token string
	token, err := ctx.Cookie("git-access-token")
	if err != nil {
		code := ctx.Query("code")
		state := ctx.Query("state")

		if storedState, err := ctx.Cookie("state"); err != nil || state != storedState {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid or expired state"})
			return
		}

		data := url.Values{}
		data.Add("client_id", os.Getenv("GIT_CLIENT_ID"))
		data.Add("client_secret", os.Getenv("GIT_CLIENT_SECRET"))
		data.Add("code", code)

		req, _ := http.NewRequest(http.MethodPost,"https://github.com/login/oauth/access_token", bytes.NewBufferString(data.Encode()))
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-type", "application/x-www-form-urlencoded")

		resp, err := core.HttpClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK{
			ctx.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "Failed to signup with github."})
			return
		}

		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		var gitToken struct {AccessToken string	`json:"access_token"`}
		
		if err := json.Unmarshal(body, &gitToken); err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse access token."})
			return
		}
		token = gitToken.AccessToken
	}

	userReq, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+token)

	ctx.SetCookie("git-access-token", token, 60 * 60 * 24, "/", "", false, true)

	userResp, err := core.HttpClient.Do(userReq)
	if err != nil || userResp.StatusCode != http.StatusOK {
		ctx.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "Failed to signup with github."})
		return
	}
	
	defer userResp.Body.Close()

	userBody, _ := io.ReadAll(userResp.Body)
	var user schema.GitHubUser
	if err := json.Unmarshal(userBody, &user); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse user info."})
		return
	}

	if user.Email == "" {
		emailReq, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user/emails", nil)
		emailReq.Header.Set("Authorization", "Bearer "+token)
		emailReq.Header.Set("Accept", "application/vnd.github+json")

		emailResp, err := core.HttpClient.Do(emailReq)
		if err != nil ||emailResp.StatusCode != http.StatusOK {
			ctx.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "Failed to fetch user email."})
			return
		}

		defer emailResp.Body.Close()

		emailBody, _ := io.ReadAll(emailResp.Body)
		var email []struct {
			Email		string  `json:"email"`
			Primary		bool 	`json:"primary"`
		}

		if err := json.Unmarshal(emailBody, &email); err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse user github emails."})
			return
		}
		for _, e := range email {
			if e.Primary{
				user.Email = e.Email
				break
			}
		}
		if user.Email == "" {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Unable to signup with github."})
			return
		}

	}

	var existingUser model.User
	db := database.GetDB()
	if err := db.Where("gitid = ?", user.ID).First(&existingUser).Error; err == nil {
		userToken, err := core.GenerateJWT(existingUser.ID,"login", core.JWTExpiry)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate jwt token for user."})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"token": userToken, "message": "Logged in successfully."})
		return
	}

	if err := db.Where("email = ?", user.Email).First(&existingUser).Error; err == nil {
		if !existingUser.GitUser {
			existingUser.GitID = &user.ID
			existingUser.GitUserName = &user.UserName
			existingUser.GitUser = true

			if err := db.Save(&existingUser).Error; err != nil {
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to log in user."})
				return
			}

			userToken, err := core.GenerateJWT(existingUser.ID, "login", core.JWTExpiry)
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate jwt token for user."})
				return
			}

			ctx.JSON(http.StatusOK, gin.H{"token": userToken, "message": "Logged in successfully."})
			return
		}else {
			ctx.AbortWithStatusJSON(http.StatusConflict, gin.H{"message": "Email already in use"})
			return
		}
	}

	var newUsername = user.UserName
	if err := db.Where("username = ?", user.UserName).First(&existingUser).Error; err == nil {
		newUsername = core.GenerateUsername(existingUser.UserName)
	}

	newUser := model.User{
		FullName : user.FullName,
		Email : user.Email,
		GitUserName : &user.UserName,
		GitID: &user.ID,
		GitUser: true,
		UserName :newUsername,
		Availability : true,
		Bio : user.Bio,
	}

	if err := db.Create(&newUser).Error; err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to signup with github."})
			return
	}
	
	userToken, err := core.GenerateJWT(newUser.ID, "login", core.JWTExpiry)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate jwt token for user."})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"token": userToken, "message": "Logged in successfully."})
}


// Log in endpoint for user
func VerifyUser(ctx *gin.Context) {
	db := database.GetDB()

	var payload schema.LoginRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload."})
		return
	}

	jwtToken, err := core.Authorization(db, payload.UserName, payload.Password)
	if err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message, "detail": cm.Detail})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Logged in successfully.", "token": jwtToken})
}


// Get user info enpoint 
func GetUserInfo(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")

	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := db.Preload("Skills").Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	var skills []string
	for _, skill := range user.Skills{
		skills = append(skills, skill.Name)
	}

	var gitusername *string
	if user.GitUser{
		token, err := ctx.Cookie("git-access-token")
		if err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Git access token not found in cookie."})
			return
		}
		req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := core.HttpClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK{
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "Failed to fetch user info from github."})
			return
		}

		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		var gitUser schema.GitHubUser
		if err := json.Unmarshal(body, &gitUser); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse user github info."})
			return
		}

		if gitUser.ID != *user.GitID {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "Unidentified user."})
			return
		}

		gitusername = &gitUser.UserName

		if gitUser.UserName != *user.GitUserName {
			user.GitUserName = &gitUser.UserName

			if err := db.Save(&user).Error; err != nil{
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "An error occured while trying to update user info."})
				return
			}
		}
	}

	payload := schema.UserProfileResponse{
		UserName: user.UserName,
		FullName: user.FullName,
		Email: user.Email,
		GitUserName: gitusername,
		Gituser: user.GitUser,
		Bio: user.Bio,
		Availability: user.Availability,
		Skills: skills,
	}
	
	ctx.JSON(http.StatusOK, payload)	
}


// search for user with username endpoint
func ViewUser(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	username := ctx.Param("name")
	if username == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Username not in query."})
		return
	}

	var user model.User
	if err := db.Preload("Skills").Preload("Posts").Where("username = ?", username).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "user not found."})
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
			Description: post.Description,
			Tags: tags,
			CreatedAt: post.CreatedAt,
			UpdatedAt: post.UpdatedAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"user": userprofile, "posts": posts})
}


// Send friend request endpoint 
func SendFriendReq(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var payload schema.SendFriendReq
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload"})
		return
	}

	var friend, user model.User
	if err := db.Preload("Friends").Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	if err := db.Where("username = ?", payload.UserName).First(&friend).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
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
	if err := db.Where("user_id = ?", user.ID).Where("friend_id = ?", friend.ID).First(&existingreq).Error; err == nil {
		ctx.JSON(http.StatusConflict, gin.H{"message": "You have to delete the previous request to this user to send another."})
		return
	}
	if err := db.Where("user_id = ?", friend.ID).Where("friend_id = ?", user.ID).First(&existingreq).Error; err == nil {
		ctx.JSON(http.StatusConflict, gin.H{"message": "This user has already sent you a request."})
		return
	}

	req := model.FriendReq{
		UserFriend: model.UserFriend{
			UserID: user.ID,
			FriendID: friend.ID,
		},
	}

	if len(payload.Message) > 0 {req.Message = payload.Message}

	if err := db.Create(&req).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to send request."})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Friend request sent successfully."})
}


// View friend requests endpoint 
func ViewFriendReq(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := db.Preload("FriendReq.Friend").Preload("RecFriendReq.User").Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	var sentRec, recReq []schema.FriendReqStatus
	for _, fr := range user.FriendReq {
		sentRec = append(sentRec, schema.FriendReqStatus{
			Status: fr.Status,
			Username: fr.Friend.UserName,
			Message: fr.Message,
		})
	}

	for _, fr := range user.RecFriendReq {
		recReq = append(recReq, schema.FriendReqStatus{
			Status: fr.Status,
			Username: fr.User.UserName,
			Message: fr.Message,
		})
	}
	
	ctx.JSON(http.StatusOK, gin.H{"sent_req": sentRec, "rec_req": recReq})
}


// Update friend request endpoint
func UpdateFriendReqStatus(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	username, status := ctx.Query("id"), ctx.Query("status")

	var user, friend model.User

	if err := db.Preload("RecFriendReq.User").Preload("Friends").Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message":"User not found."})
		return
	}
	if err := db.Preload("Friends").Where("username = ?", username).First(&friend).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Friend not found."})
		return
	}

	var userreq *model.FriendReq
	for _, fr := range user.RecFriendReq {
		if fr.User.ID == friend.ID {
			userreq = fr
			break
		}
	}

	if userreq == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Request not found."})
		return
	}

	switch status {
		case model.StatusRejected:
			if err := db.Model(userreq).Update("Status", model.StatusRejected).Error; err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to reject request."})
				return
			}
		case model.StatusAccepted:
			if err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Unscoped().Delete(userreq).Error; err != nil {return err}

				if err := tx.Model(&user).Association("Friends").Append(&friend); err != nil {return err}

				if err := tx.Model(&friend).Association("Friends").Append(&user); err != nil {return err}

				return nil
			}); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update request status"})
				return
			}
		default:
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid status."})
			return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "Status updated successfully"})
}


// Delete sent request endpoint
func DeleteSentReq(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	username := ctx.Query("id")

	var user, friend model.User
	if err := db.Preload("FriendReq.Friend").Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}
	if err := db.Where("username = ?", username).First(&friend).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Friend not found."})
		return
	}

	var req *model.FriendReq
	for _, fr := range user.FriendReq {
		if fr.Friend.ID == friend.ID {
			req = fr
			break
		}
	}
	if err := db.Unscoped().Delete(req).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete req."})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}


// Remove friend endpoint
func DeleteUserFriend(ctx *gin.Context) {
	db := database.GetDB()

	uid := ctx.GetUint("userID")
	tp := ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	username := ctx.Query("id")

	var user, friend model.User
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Friends").Where("id = ?", uid).First(&user).Error; err != nil {return err}

		if err := tx.Preload("Friends").Where("username = ?", username).First(&friend).Error; err != nil {return err}

		if err := tx.Model(&user).Association("Friends").Delete(&friend); err != nil {return err}

		if err := tx.Model(&friend).Association("Friends").Delete(&user); err != nil {return err}

		return nil
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to remove friend"})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}


// view all user friend endpoint 
func ViewUserFriends(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := db.Preload("Friends").Where("id = ?", uid).First(&user).Error; err != nil {
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
func ForgotPassword(ctx *gin.Context) {
	db := database.GetDB()
	rdb := database.GetRDB()

	var payload schema.ForgotPasswordEmail
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload."})
		return
	}

	var user model.User
	if err := db.Where("email = ?", payload.Email).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User record not found."})
		return
	}

	token := core.GenerateOTP()
	if err := core.SetOTP(rdb, token, user.ID); err != nil {
		log.Printf("Failed to store token in redis -> %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to store otp"})
		return
	}

	if err := core.SendForgotPassEmail(user.Email, user.UserName, token); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to send email"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Email sent successfully."})
}


// verify otp for password reset endpoint
func VerifyOTP(ctx *gin.Context) {
	rdb := database.GetRDB()

	var payload schema.VerifyOTP

	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload"})
		return
	}

	token, err := core.GetOTP(rdb, payload.Token)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Invalid token."})
		return
	}

	jwt, err := core.GenerateJWT(token.UserID, "reset", core.JWTRExpiry)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create jwt token"})
		return
	}
	
	ctx.JSON(http.StatusOK, gin.H{"message": "otp verified", "token": jwt})
}


// Actual reset password endpoint
func ResetPassword(ctx *gin.Context) {
	db := database.GetDB()

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
	if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
		return
	}

	hashed, err := core.HashPassword(payload.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Unable to hash password"})
		return
	}

	user.Password = hashed

	if err := db.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to reset password."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "password reset successfully."})	
}


// Update user info endpoint 
func UpdateUserInfo(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}


	var user, existingUser model.User
	if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	var payload schema.UserProfileRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload."})
		return
	}

	if err := db.Where("username = ? OR email = ?", payload.UserName, payload.Email).First(&existingUser).Error; err == nil && existingUser.ID != uid {
		if existingUser.Email == payload.Email {
			ctx.JSON(http.StatusConflict, gin.H{"message": "Email already in use!"})
		}else {
			ctx.JSON(http.StatusConflict, gin.H{"message": "Username already in use!"})
		}
		return

	}

	user.Email = payload.Email
	user.Bio = payload.Bio
	user.FullName = payload.FullName
	user.UserName = payload.UserName

	if err := db.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user profile."})
		return
	}
	
	ctx.JSON(http.StatusAccepted, gin.H{"message": "User profile updated successfully."})
}


// Update user password endpoint
func UpdateUserPassword(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "user not found."})
		return
	}

	var payload schema.ResetPassword
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload."})
		return
	}

	hashed, err := core.HashPassword(payload.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate hash for user password"})
		return
	}
	user.Password = hashed

	if err := db.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user password."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "User password updated successfully."})
}


// Update user avaibility status endpoint 
func UpdateUserAvaibilityStatus(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	status := ctx.Param("status")
	statusbool, err := strconv.ParseBool(status)
	if  err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Availability status can only be true or false."})
		return
	}

	user.Availability = statusbool

	if err := db.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user availability."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "User availability updated successfully."})
}


// Update user skills endpoint
func UpdateUserSkills(ctx *gin.Context) {
	db := database.GetDB()
	rdb := database.GetRDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := db.Preload("Skills").Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	var payload schema.UpdateUserSkillsRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse the payload."})
		return
	}

	for i := range payload.Skills {payload.Skills[i] = strings.ToLower(payload.Skills[i])}
	allskills, err := CheckAndUpdateSkills(db, rdb, payload.Skills)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user skills."})
		return
	}
	
	if err := db.Model(&user).Association("Skills").Replace(allskills); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user skills."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "User skills updated successfully."})
}


// Delete user skills endpoint
func DeleteUserSkills(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := db.Preload("Skills").Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
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

	if err := db.Model(&user).Association("Skills").Delete(skillsToDelete); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete user skills."})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}


// Delete user account endpoint using soft deleting 
func DeleteUserAccount(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db"})
		return
	}

	if err := db.Delete(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete user account"})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
