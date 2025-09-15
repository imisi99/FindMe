package handlers

import (
	"findme/core"
	"findme/database"
	"findme/model"
	"findme/schema"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Helper func for checking and updating skills
func CheckAndUpdateSkills(db *gorm.DB, rdb *redis.Client, payload []string) ([]*model.Skill, error) {
	skills, err := core.RetrieveCachedSkills(rdb, payload)
	
	if err != nil { 																		// Falling back to the db if the redis fails 
		var existingSkills []*model.Skill

		if err := db.Where("name IN ?", payload).Find(&existingSkills).Error; err != nil{
			return nil, err
		}

		existingSkillSet := make(map[string]bool)
		for _, name := range existingSkills {existingSkillSet[name.Name] = true}

		var newSkill []*model.Skill
		for _, skill := range payload {
			if _, exists := existingSkillSet[skill]; !exists {
				newSkill = append(newSkill, &model.Skill{Name: skill})
			}
		}

		if len(newSkill) > 0 {
			if err := db.Create(&newSkill).Error; err != nil {
				return nil, err
			}
		}
		newSkill = append(newSkill, existingSkills...)
		return newSkill, nil
	}

	var newskills, allskills []*model.Skill
	for _, skill := range payload {
		if id, exists := skills[skill]; exists {
			allskills = append(allskills, &model.Skill{Name: skill, Model: gorm.Model{ID: id}})
			continue
		}
		newskills = append(newskills, &model.Skill{Name: skill})
	}

	if len(newskills) > 0 {
		if err := db.Create(&newskills).Error; err != nil {
			return nil, err
		}
		core.AddNewSkillToCache(rdb, newskills)
	}
	allskills = append(allskills, newskills...)
	return allskills, nil 
} 


// Endpoint for getting all user posts
func GetPosts(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	var reuslt []schema.PostResponse

	if err := db.Preload("Posts.Tags").Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch user posts"})
		return
	}

	for _, post := range user.Posts {
		var tags []string
		for _, tag := range post.Tags {tags = append(tags, tag.Name)}
		reuslt = append(reuslt, schema.PostResponse{
			ID: post.ID,
			Description: post.Description,
			Tags: tags,
			CreatedAt: post.CreatedAt,
			UpdatedAt: post.UpdatedAt,
			Views: post.Views,
		})
	}
	ctx.JSON(http.StatusOK, reuslt)	 
}


// Endpoint for viewing a single post
func ViewPost(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	pidStr := ctx.Param("id")
	pid, err := strconv.ParseUint(pidStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid post id."})
		return
	}

	id := uint(pid)
	var post model.Post
	var result schema.DetailedPostResponse

	if err := db.Preload("Tags").Preload("User").Where("id = ?", id).First(&post).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		return
	}

	var tags []string

	for _, tag := range post.Tags {tags = append(tags, tag.Name)}

	result.CreatedAt = post.CreatedAt
	result.UpdatedAt = post.UpdatedAt
	result.Description = post.Description
	result.Tags = tags
	result.Username = post.User.UserName
	result.Views = post.Views

	ctx.JSON(http.StatusOK, result)
}


// Endpoint for creating post
func CreatePost(ctx *gin.Context) {
	db := database.GetDB()
	rdb := database.GetRDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var payload schema.NewPostRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload."})
		return
	}

	for i := range payload.Tags {payload.Tags[i] = strings.ToLower(payload.Tags[i])}
	allskills, err := CheckAndUpdateSkills(db, rdb, payload.Tags)
	if err != nil {
		log.Printf("An error occured while trying to add a new skill to db %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create new post."})
		return
	}

	post := model.Post{
		Description: payload.Description,
		Tags: allskills,
		UserID: uid,
		Views: 0,
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

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	idStr := ctx.Param("id")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid id."})
		return
	}

	postID := uint(id)

	var payload schema.NewPostRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload."})
		return
	}

	var post model.Post
	if err := db.Where("id = ?", postID).First(&post).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		return
	}

	if post.UserID != uid {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user cannot edit post."})
		return
	}

	for i := range payload.Tags {payload.Tags[i] = strings.ToLower(payload.Tags[i])}
	allskills, err := CheckAndUpdateSkills(db, rdb, payload.Tags)
	if err != nil {
		log.Printf("An error occured while trying to add a new skill to db %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update post."})
		return
	}

	post.Description = payload.Description

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&post).Association("Tags").Replace(allskills); err != nil {return err}

		if err := tx.Save(&post).Error; err != nil {return err}

		return nil
	}); err != nil {
		log.Printf("An error occured during the editing of the post with id -> %v, -> %v", post.ID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"messasge": "Failed to update post."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "Post updated successfully."})
}


// Ednpoint for updating a post view
func EditPostView(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	id := ctx.Param("id")
	
	idStr, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid post id"})
		return
	}
	pid := uint(idStr)


	var post model.Post

	if err := db.Where("id = ?", pid).First(&post).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		return
	}

	if post.UserID != uid {
		post.Views++
		if err := db.Save(&post).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update views."})
			return
		}
	}
	ctx.JSON(http.StatusNoContent, nil)
}


// Endpoint for saving a post
func SavePost(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp, idStr := ctx.GetUint("userID"), ctx.GetString("purpose"), ctx.Query("id")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid post id."})
		return
	}
	pid := uint(id)

	var user model.User
	if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	var post model.Post
	if err := db.Where("id = ?", pid).First(&post).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found"})
		return
	}

	if post.UserID == user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You can't save a post created by you."})
		return
	}

	if err := db.Model(&user).Association("SavedPosts").Append(&post); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to save post."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "Post saved successfully"})
}


// Endpoint for viewing saved post
func ViewSavedPost(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unautorized user."})
		return
	}

	var user model.User
	if err := db.Preload("SavedPosts").Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "user not found."})
		return
	}

	var savedPosts []schema.PostResponse
	for _, post := range user.SavedPosts {
		var tags []string
		for _, tag := range post.Tags {tags = append(tags, tag.Name)}
		savedPosts = append(savedPosts, schema.PostResponse{
			ID: post.ID,
			Description: post.Description,
			Tags: tags,
			CreatedAt: post.CreatedAt,
			UpdatedAt: post.UpdatedAt,
		})
	}
	ctx.JSON(http.StatusOK, savedPosts)
}


// Endpoint for deleting a post
func DeletePost(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	id := ctx.Param("id")
	
	idStr, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid post id"})
		return
	}
	pid := uint(idStr)


	var post model.Post

	if err := db.Where("id = ?", pid).First(&post).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		return
	}

	if post.UserID != uid {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	if err := db.Delete(&post).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete post."})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
