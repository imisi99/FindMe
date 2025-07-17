package handlers

import (
	"findme/core"
	"findme/database"
	"findme/model"
	"findme/schema"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Sign up route for the user
func AddUser(ctx *gin.Context) {
	db := database.GetDB()
	var payload schema.SignupRequest

	err := ctx.ShouldBindJSON(&payload)
	if err != nil{
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "An error occured while tyring to parse the payload."})
		return
	}

	hashedPassword, err := core.HashPassword(payload.Password)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "An error occured while trying to encrypt the password."})
		return
	}

	// Expensive operation for adding new skills to database ? 
	var existingSkills []model.Skill
	db.Where("name = ?", payload.Skills).Find(&existingSkills)

	existingSkillSet := make(map[string]bool)
	for _, name := range existingSkills {
		existingSkillSet[name.Name] = true
	}

	var newSkill []model.Skill
	for _, skills := range payload.Skills {
		if _, exist := existingSkillSet[skills]; !exist {
			newSkill = append(newSkill, model.Skill{Name: skills})
		}
	}


	if len(newSkill) > 0 {
		db.Create(&newSkill)
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
		ctx.JSON(500, gin.H{"message": "An error occured while generating jwt token"})
		return
	}

	ctx.JSON(200, gin.H{"message": "Signed up successfully.", "token": jwtToken})
}


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
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
	}

	ctx.JSON(200, gin.H{"message": "Logged in successfully.", "token": jwtToken})
}
