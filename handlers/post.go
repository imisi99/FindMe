package handlers

import (
	"findme/core"
	"findme/database"
	"findme/model"
	"findme/schema"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)


// Helper func for checking and updating skills 
func checkSkills(db *gorm.DB, rdb *redis.Client, payload *schema.NewPostRequest) ([]*model.Skill, error) {
	skills := core.RetrieveCachedSkills(rdb)
	
	var newSkills, allskills []*model.Skill
	for _, skill := range payload.Tags {
		if _, exists := skills[skill]; !exists {
			newSkills = append(newSkills, &model.Skill{Name: skill})
			continue
		} 
		allskills = append(allskills, &model.Skill{Name: skill})
	}


	if len(newSkills) > 0 {
		if err := db.Create(newSkills).Error; err != nil {     // Add a way to also keep track of new skills after startup in redis
			return nil, err
		}

	}

	allskills = append(allskills, newSkills...)

	return allskills, nil
} 


// Endpoint for creating post
func CreatePost(ctx *gin.Context) {
	db := database.GetDB()
	rdb := database.GetRDB()

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

	allskills, err := checkSkills(db, rdb, &payload)
	fmt.Println(allskills)
	if err != nil {
		log.Printf("An error occured while trying to add a new skill to db %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Error while trying to add new skills to db"})
	}

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


// Endpoint for editing post
func EditPost(ctx *gin.Context) {
	db := database.GetDB()
	rdb := database.GetRDB()

	uid := ctx.GetUint("userID")
	idStr := ctx.Param("id")
	if uid == 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var payload schema.NewPostRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload."})
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid id."})
		return
	}

	postID := uint(id)

	var post model.Post
	if err := db.Where("id = ?", postID).First(&post).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		return
	}

	if post.User.ID != uid {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user cannot edit post."})
		return
	}

	allskills, err := checkSkills(db, rdb, &payload)
	if err != nil {
		log.Printf("An error occured while trying to add a new skill to db %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Error while trying to add new skills to db"})
		return
	}

	post.Description = payload.Description
	post.Tags = allskills

	if err := db.Save(&post).Error; err != nil {
		log.Printf("An error occured while trying to update post %v -> %v", post.ID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update post."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "Post updated successfully."})
}
