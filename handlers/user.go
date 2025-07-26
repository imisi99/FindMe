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
