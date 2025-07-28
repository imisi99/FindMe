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
)

// Sign up endpoint for user
func AddUser(ctx *gin.Context) {
	db := database.GetDB()

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

	// Expensive operation for adding new skills to database
	var existingSkills []*model.Skill
	for i := range payload.Skills {
		payload.Skills[i] = strings.ToLower(payload.Skills[i])
	}
	
	db.Where("name IN ?", payload.Skills).Find(&existingSkills)

	existingSkillSet := make(map[string]bool)
	for _, name := range existingSkills {
		existingSkillSet[name.Name] = true
	}

	var newSkill []*model.Skill
	for _, skills := range payload.Skills {
		if _, exist := existingSkillSet[skills]; !exist {
			newSkill = append(newSkill, &model.Skill{Name: skills})
		}
	}

	if len(newSkill) > 0 {
		if err := db.Create(newSkill).Error; err != nil {
			ctx.JSON((http.StatusInternalServerError), gin.H{"message": "Failed to add new skills."})
			return			
		}
	}

	// Adding the new user to the database
	hashedPassword, err := core.HashPassword(payload.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to encrypt the user password."})
		return
	}

	allskills := append(newSkill, existingSkills...)

	user := model.User{
		FullName: payload.FullName,
		UserName: payload.UserName,
		Email: payload.Email,
		Password: hashedPassword,
		Bio: payload.Bio,

		Skills: allskills,
		Availability: true,
	}

	if err = db.Create(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create user."})
		return
	}

	jwtToken, err := core.GenerateJWT(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate jwt token.", "detail": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "Signed up successfully.", "token": jwtToken})
}


// Signing up user using github
func GitHubAddUser(ctx *gin.Context) {
	state, err := core.GenerateState()
	if err != nil {
		ctx.JSON(500, gin.H{"message": "Failed to generate user state."})
		return
	}

	ctx.SetCookie("state", state, 150, "/", "", false, true)

	redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=read:user user:email", os.Getenv("GIT_CLIENT_ID"), state)
	ctx.Redirect(http.StatusTemporaryRedirect, redirectURL)
}


func GitHubAddUserCallback(ctx *gin.Context) {
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
		log.Println("Failed to fetch access token from github ->", err.Error())
		ctx.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "Failed to fetch access token from github"})
		return
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var token struct {AccessToken string		`json:"access_token"`}
	
	if err := json.Unmarshal(body, &token); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse access token."})
		return
	}

	userReq, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

	userResp, err := core.HttpClient.Do(userReq)
	if err != nil || userResp.StatusCode != http.StatusOK {
		log.Println("Failed to fetch user info from github ->", err.Error())
		ctx.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "Failed to fetch user info."})
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
		emailReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
		emailReq.Header.Set("Accept", "application/vnd.github+json")

		emailResp, err := core.HttpClient.Do(emailReq)
		if err != nil ||emailResp.StatusCode != http.StatusOK {
			log.Println("Failed to fetch user email from github ->", err.Error())
			ctx.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "Failed to fetch user email."})
			return
		}

		defer emailResp.Body.Close()

		emailBody, _ := io.ReadAll(emailResp.Body)
		var email []struct {
			Email		string  `json:"email"`
			Primary		bool 	`json:"primary"`
			Verified	bool 	`json:"verified"`
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
		}

	}

	var existingUser model.User
	db := database.GetDB()
	if err := db.Where("email = ?", user.Email).First(&existingUser).Error; err == nil {
		userToken, err := core.GenerateJWT(existingUser.ID)
		if err != nil {
			log.Println("Failed to generate jwt token for user -> ", err.Error())
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate jwt token for user."})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"token": userToken, "message": "Logged in successfully."})
		return
	}

	newUser := model.User{
		FullName : user.FullName,
		Email : user.Email,
		GitUserName : &user.UserName,
		UserName :user.UserName,
		Availability : true,
		Bio : user.Bio,
	}

	if err := db.Create(&newUser).Error; err != nil {
			log.Println("Failed to store user in db -> ", err.Error())
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to create user."})
			return
	}
	
	userToken, err := core.GenerateJWT(newUser.ID)
	if err != nil {
		log.Println("Failed to generate jwt token for user -> ", err.Error())
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

	uid := ctx.GetUint("userID")
	if uid == 0 {
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

	payload := schema.UserProfileResponse{
		UserName: user.UserName,
		FullName: user.FullName,
		Email: user.Email,
		GitUserName: user.GitUserName,
		Bio: user.Bio,
		Skills: skills,
	}
	
	ctx.JSON(http.StatusOK, payload)	
}


// Update user info endpoint 
func UpdateUserInfo(ctx *gin.Context) {
	db := database.GetDB()

	uid := ctx.GetUint("userID")
	if uid == 0 {
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
	user.GitUserName = payload.GitUserName

	if err := db.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user profile."})
		return
	}
	
	ctx.JSON(http.StatusAccepted, gin.H{"message": "User profile updated successfully."})
}


// Update user avaibility status endpoint 
func UpdateUserAvaibilityStatus(ctx *gin.Context) {
	db := database.GetDB()

	uid := ctx.GetUint("userID")
	if uid == 0{
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
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "User availability updated successfully."})
}


// Update user skills endpoint
func UpdateUserSkills(ctx *gin.Context) {
	db := database.GetDB()

	uid := ctx.GetUint("userID")
	if uid == 0 {
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

	for i := range payload.Skills {
		payload.Skills[i] = strings.ToLower(payload.Skills[i])
	}

	// Expensive operation of updating skills 
	var existingSkills []*model.Skill
	db.Where("name IN ?", payload.Skills).Find(&existingSkills)

	existingSkillSet := make(map[string]bool)
	for _, skill := range existingSkills {
		existingSkillSet[skill.Name] = true
	}

	var newSkill []*model.Skill
	for _, skill := range payload.Skills {
		if _, exists := existingSkillSet[skill]; !exists {
			newSkill = append(newSkill, &model.Skill{Name: skill})
		}
	}

	if len(newSkill) > 0 {
		if err := db.Create(newSkill).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user skills."})
			return
		}
	}

	newSkill = append(newSkill, existingSkills...)
	if err := db.Model(&user).Association("Skills").Append(newSkill); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user skills."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "User skills updated successfully."})
}


// Delete user skills endpoint
func DeleteUserSkills(ctx *gin.Context) {
	db := database.GetDB()

	uid := ctx.GetUint("userID")
	if uid == 0 {
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

	deletedSkills := make(map[string]bool)
	for _, skill := range payload.Skills {
		deletedSkills[strings.ToLower(skill)] = true
	}

	var skillsToDelete []*model.Skill
	for _, skill := range user.Skills {
		if _, exists := deletedSkills[skill.Name]; exists {
			skillsToDelete = append(skillsToDelete, skill)
		}
	}

	if err := db.Model(&user).Association("Skills").Delete(skillsToDelete); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete user skills."})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
