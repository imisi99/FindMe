package handlers

import (
	"findme/core"
	"findme/database"
	"findme/model"
	"findme/schema"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Endpoint for creating post
func CreatePost(ctx *gin.Context) {
	db := database.GetDB()
	rdb := database.GetRDB()


	skills := core.RetrieveCachedSkills(rdb)
	
	uid := ctx.GetUint("userID")
	if uid == 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var payload schema.NewPostRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload."})
		return
	}

	var newSkills, allskills []*model.Skill

	for _, skill := range payload.Tags {
		if _, exists := skills[skill]; !exists {
			newSkills = append(newSkills, &model.Skill{Name: skill})
			continue
		} 
		allskills = append(allskills, &model.Skill{Name: skill})
	}


	if len(newSkills) > 0 {
		if err := db.Create(newSkills).Error; err != nil {
			log.Printf("Failed to create new skill -> %v", err.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create new skills."})
			return
		}

	}

	allskills = append(allskills, newSkills...)

	post := model.Post{
		Description: payload.Description,
		Tags: allskills,
		UserID: uid,
	}

	if err := db.Create(&post).Error; err != nil {
		log.Printf("Failed to create new post -> %v", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create new post."})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "Post created successfully."})
}
