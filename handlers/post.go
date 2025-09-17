package handlers

import (
	"errors"
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

	if err := db.Preload("Posts.Tags").Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user posts."})}
		return
	}

	var reuslt []schema.PostResponse
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
	
	ctx.JSON(http.StatusOK, gin.H{"post": reuslt})	 
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

	var post model.Post
	if err := db.Preload("Tags").Preload("User").Where("id = ?", uint(pid)).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve post from db."})}
		return
	}

	var tags []string
	for _, tag := range post.Tags {tags = append(tags, tag.Name)}
	result := schema.DetailedPostResponse{
		PostResponse: schema.PostResponse{
			ID: post.ID,
			Description: post.Description,
			Tags: tags,
			CreatedAt: post.CreatedAt,
			UpdatedAt: post.UpdatedAt,
			Views: post.Views,
		},
		Username: post.User.UserName,
	}

	ctx.JSON(http.StatusOK, gin.H{"post": result})
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
		log.Printf("An error occured while trying to add a new skill to db -> %v", err)
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
		log.Printf("Failed to create new post -> %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create new post."})
		return
	}
	
	result := schema.PostResponse{
		ID: post.ID,
		Description: post.Description,
		Tags: payload.Tags,
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
		Views: post.Views,
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "Post created successfully.", "post": result})
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
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid post id."})
		return
	}

	var payload schema.NewPostRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload."})
		return
	}

	var post model.Post
	if err := db.Preload("Tags").Where("id = ?", uint(id)).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve post from db."})}
		return
	}

	if post.UserID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You aren't authorized to edit this post."})
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"messasge": "Failed to update post."})
		return
	}

	result := schema.PostResponse{
		ID: post.ID,
		Description: post.Description,
		Tags: payload.Tags,
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
		Views: post.Views,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "Post updated successfully.", "post": result})
}


// Ednpoint for updating a post view
func EditPostView(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp, idStr := ctx.GetUint("userID"), ctx.GetString("purpose"), ctx.Param("id")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}
	
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid post id."})
		return
	}

	var post model.Post
	if err := db.Where("id = ?", uint(id)).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve post from db."})}
		return
	}

	if post.UserID != uid {
		post.Views++
		if err := db.Save(&post).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update views."})
			return
		}
	}

	var tags []string
	for _, tag := range post.Tags {tags = append(tags, tag.Name)}
	result := schema.PostResponse{
		ID: post.ID,
		Description: post.Description,
		Tags: tags,
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
		Views: post.Views,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"post": result})
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

	var user model.User
	if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	var post model.Post
	if err := db.Where("id = ?", uint(id)).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve post from db."})}
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
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := db.Preload("SavedPosts").Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
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
	ctx.JSON(http.StatusOK, gin.H{"post": savedPosts})
}


// Endpoint for applying for a post 
func ApplyForPost(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp, idStr := ctx.GetUint("userID"), ctx.GetString("purpose"), ctx.Query("id")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Invalid post id."})
		return
	}

	var payload schema.PostApplication
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		log.Println(err)
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload"})
		return
	}

	var post model.Post
	if err := db.Preload("User").Where("id = ?", uint(id)).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve post from db."})}
		return
	}

	var user model.User
	if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	if post.User.ID == user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You can't apply for a post owned by you."})
		return
	}

	req := model.PostReq{
		PostID: post.ID,
		FromID: user.ID,
		ToID: post.User.ID,
	}
	if len(payload.Message) > 0 {req.Message = payload.Message}

	if err := db.Create(&req).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to send application."})
		return
	}
	core.SendPostApplicationEmail(post.User.Email, user.UserName, post.User.UserName, req.Message, "")

	ctx.JSON(http.StatusOK, gin.H{"message": "Application sent successfully."})
}


// Endpoint for Viewing post applications
func ViewPostApplications(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := db.Preload("RecPostReq.FromUser").Preload("SentPostReq.ToUser").Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "user not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	var rec_req, sent_req  []schema.ViewPostApplication
	for _, rq := range user.SentPostReq {
		sent_req = append(sent_req, schema.ViewPostApplication{
			ReqID: rq.ID,
			Username: rq.ToUser.UserName,
			Message: rq.Message,
			Status: rq.Status,
		})
	}

	for _, rq := range user.RecPostReq {
		rec_req = append(rec_req, schema.ViewPostApplication{
			ReqID: rq.ID,
			Username: rq.FromUser.UserName,
			Message: rq.Message,
			Status: rq.Status,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"post": gin.H{"rec_req": rec_req, "sent_req": sent_req}})
}


// Endpoint for Updating post applications 
func UpdatePostApplication(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp, reqID, status := ctx.GetUint("userID"),  ctx.GetString("purpose"), ctx.Query("id"), ctx.Query("status")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	rid, err := strconv.ParseUint(reqID, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Invalid request id."})
		return
	}

	var req model.PostReq
	if db.Preload("Post").Preload("FromUser").Where("id = ?", rid).First(&req).Error != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Application not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve application from db."})}
		return
	}
	
	var user, friend model.User
	if db.Preload("Friends").Where("id = ?", uid).First(&user).Error != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db"})}
		return
	}

	if db.Where("username = ?", req.FromUser.UserName).First(&friend).Error != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Applicant not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve applicant from db."})}
		return
	}

	if req.ToID != user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You don't have permission to update this application."})
		return
	}

	var friends bool
	for _, fr := range user.Friends {
		if friend.ID == fr.ID {
			friends = true
			break
		}
	}

	switch status {
		case model.StatusRejected:
			if err := db.Model(&req).Update("Status", model.StatusRejected).Error; err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to reject application."})
				return
			}
		case model.StatusAccepted:
			if err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Unscoped().Delete(&req).Error; err != nil {return err}
				
				if !friends {
					if err := tx.Model(&user).Association("Friends").Append(&friend); err != nil {return err}

					if err := tx.Model(&friend).Association("Friends").Append(&user); err != nil {return err}
				}
				return nil
			}); err != nil && core.SendPostApplicationAccept(friend.Email, user.UserName, friend.UserName, req.Post.Description, "")  != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to accept application."})
				return
			}
		default:
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid status."})
			return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "Application status updated successfully."})
}


// Endpoint for deleting sent post application
func DeletePostApplication(ctx *gin.Context) {
	db := database.GetDB()

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

	var user model.User
	if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})}
		return
	}

	var req model.PostReq
	if err := db.Preload("FromUser").Where("id = ?", uint(rid)).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Request not found."})
		}else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve application from db."})}
		return
	}

	if req.FromUser.ID != user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You don't have permission to delete this application."})
		return
	}

	if err := db.Unscoped().Delete(&req).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete application."})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}


// Endpoint for deleting a post
func DeletePost(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp, idStr := ctx.GetUint("userID"), ctx.GetString("purpose"), ctx.Param("id")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid post id."})
		return
	}

	var post model.Post
	if err := db.Where("id = ?", uint(id)).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		} else {ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve post from db."})}
		return
	}

	if post.UserID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You don't have permission to delete this post."})
		return
	}

	if err := db.Delete(&post).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete post."})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
