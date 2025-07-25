package handlers

import (
	"findme/core"
	"findme/database"
	"findme/model"
	"findme/schema"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Sign up endpoint for user
func AddUser(ctx *gin.Context) {
	db := database.GetDB()
	var payload schema.SignupRequest

	err := ctx.ShouldBindJSON(&payload)
	if err != nil{
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "An error occured while tyring to parse the payload."})
		return
	}


	// Checking for existing username | email 
	var existingUser model.User

	if err = db.Where("username = ? OR email = ?", payload.UserName, payload.Email).First(&existingUser).Error; err == nil {
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
		err = db.Create(newSkill).Error
		if err != nil {
			ctx.JSON((http.StatusBadRequest), gin.H{"message": "An error occured while trying to add skills"})
			return
		}
	}


	// Adding the new user to the database
	hashedPassword, err := core.HashPassword(payload.Password)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "An error occured while trying to encrypt the password."})
		return
	}
	allskills := append(newSkill, existingSkills...)

	user := model.User{
		FullName: payload.FullName,
		Username: payload.UserName,
		Email: payload.Email,
		Password: hashedPassword,
		Bio: payload.Bio,

		Skills: allskills,
		Availability: true,
	}

	err = db.Create(&user).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "An error occured while trying to create user"})
		return
	}

	jwtToken, err := core.GenerateJWT(user.ID)
	if err != nil {
		ctx.JSON(500, gin.H{"message": "An error occured while generating jwt token", "detail": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"message": "Signed up successfully.", "token": jwtToken})

}


// Log in endpoint for user
func VerifyUser(ctx *gin.Context) {
	db := database.GetDB()
	var payload schema.LoginRequest

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "An error occured while trying to parse the payload"})
		return
	}

	jwtToken, err := core.Authorization(db, payload.UserName, payload.Password)
	if err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message, "detail": cm.Detail})
		return
	}

	ctx.JSON(200, gin.H{"message": "Logged in successfully.", "token": jwtToken})

}


// Get user info enpoint 
func GetUserInfo(ctx *gin.Context) {
	db := database.GetDB()

	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "User ID not found in context"})
		return
	}

	uid, ok :=  userID.(uint)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "User ID is not in the valid format"})
		return
	}

	var user model.User

	err := db.Preload("Skills").Where("id = ?", uid).First(&user).Error
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	var skills []string

	for _, skill := range user.Skills{
		skills = append(skills, skill.Name)
	}

	payload := schema.UserProfileResponse{
		UserName: user.Username,
		FullName: user.FullName,
		Email: user.Email,
		Bio: user.Bio,
		Skills: skills,
	}

	
	ctx.JSON(http.StatusOK, payload)
	
}


// Update user info endpoint 
func UpdateUserInfo(ctx *gin.Context) {
	db := database.GetDB()

	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "User ID not found in context"})
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "User ID is not in the valid format"})
		return
	}

	var payload schema.UserProfileRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "An error occured while trying to parse the payload"})
		return
	}

	var user, existingUser model.User
	err := db.Where("id = ?", uid).First(&user).Error
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
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
	user.Username = payload.UserName

	if err = db.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Unable to update user profile"})
		return
	}
	
	ctx.JSON(http.StatusAccepted, gin.H{"message": "User profile updated successfully."})
	
}


// Update user avaibility status endpoint 
func UpdateUserAvaibilityStatus(ctx *gin.Context) {
	db := database.GetDB()

	userID, exists := ctx.Get("userID")
	if !exists{
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "User ID not found in context"})
		return
	}

	uid := userID.(uint)

	status := ctx.Param("status")
	statusbool, err := strconv.ParseBool(status)
	if  err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Availability status can only be true or false"})
		return
	}

	var user model.User
	if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	user.Availability = statusbool

	if err := db.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "An error occured while updating availability"})
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "User availability status updated successfully."})
}


// Update user skills status endpoint
func UpdateUserSkills(ctx *gin.Context) {
	db := database.GetDB()

	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "User ID not found in context"})
		return
	}

	uid := userID.(uint)

	var user model.User
	if err := db.Preload("Skills").Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	var payload schema.UpdateUserSkillsRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "An error occured while tyring to parse the payload"})
		return
	}

	for i := range payload.Skills {
		payload.Skills[i] = strings.ToLower(payload.Skills[i])
	}

	// Expensive operation of updating skills 

	var existingSkills []*model.Skill
	db.Where("name IN ?", payload.Skills).Find(&existingSkills)

	existingSkillSet := make(map[string]bool)
	for i := range existingSkills {
		existingSkillSet[existingSkills[i].Name] = true
	}

	var newSkill []*model.Skill
	for i := range payload.Skills {
		if _, exists := existingSkillSet[payload.Skills[i]]; !exists {
			newSkill = append(newSkill, &model.Skill{Name: payload.Skills[i]})
		}
	}

	if len(newSkill) > 0 {
		if err := db.Create(newSkill).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "An error occured while trying to update user skills"})
			return
		}
	}

	allskills := append(newSkill, existingSkills...)

	user.Skills = allskills

	if err := db.Save(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "An error occured while trying to update user skills"})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "User skills updated successfully"})
}
