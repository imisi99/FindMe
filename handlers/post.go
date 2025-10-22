package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

// GetPosts -> Endpoint for getting all user posts
func (p *Service) GetPosts(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := p.DB.FetchUserPosts(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	var reuslt []schema.PostResponse
	for _, post := range user.Posts {
		var tags []string
		for _, tag := range post.Tags {
			tags = append(tags, tag.Name)
		}
		reuslt = append(reuslt, schema.PostResponse{
			ID:          post.ID,
			Description: post.Description,
			Tags:        tags,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
			Views:       post.Views,
			Available:   post.Availability,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"post": reuslt})
}

// ViewPost -> Endpoint for viewing a single post
func (p *Service) ViewPost(ctx *gin.Context) {
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
	if err := p.DB.FetchPostPreloadTU(&post, uint(pid)); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	var tags []string
	for _, tag := range post.Tags {
		tags = append(tags, tag.Name)
	}
	result := schema.DetailedPostResponse{
		PostResponse: schema.PostResponse{
			ID:          post.ID,
			Description: post.Description,
			Tags:        tags,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
			Views:       post.Views,
			Available:   post.Availability,
		},
		Username: post.User.UserName,
	}

	ctx.JSON(http.StatusOK, gin.H{"post": result})
}

// SearchPost -> Endpoint for searching post with tags
func (p *Service) SearchPost(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var payload schema.SearchPostWithTags
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload."})
		return
	}

	for i := range payload.Tags {
		payload.Tags[i] = strings.ToLower(payload.Tags[i])
	}

	var posts []model.Post
	if err := p.DB.SearchPostsBySKills(&posts, payload.Tags); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	var postResponse []schema.PostResponse
	for _, post := range posts {
		var tags []string
		for _, tag := range post.Tags {
			tags = append(tags, tag.Name)
		}
		postResponse = append(postResponse, schema.PostResponse{
			ID:          post.ID,
			Description: post.Description,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
			Available:   post.Availability,
			Views:       post.Views,
			Tags:        tags,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"post": postResponse})
}

// CreatePost -> Endpoint for creating post
func (p *Service) CreatePost(ctx *gin.Context) {
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

	for i := range payload.Tags {
		payload.Tags[i] = strings.ToLower(payload.Tags[i])
	}
	allskills, err := p.CheckAndUpdateSkills(payload.Tags)
	if err != nil {
		log.Printf("An error occured while trying to add a new skill to db -> %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create new post."})
		return
	}

	post := model.Post{
		Description:  payload.Description,
		Tags:         allskills,
		UserID:       uid,
		Views:        0,
		Availability: true,
	}
	if err := p.DB.AddPost(&post); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	result := schema.PostResponse{
		ID:          post.ID,
		Description: post.Description,
		Tags:        payload.Tags,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		Views:       post.Views,
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "Post created successfully.", "post": result})
}

// EditPost -> Endpoint for editing post
func (p *Service) EditPost(ctx *gin.Context) {
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
	if err := p.DB.FetchPostPreloadT(&post, uint(id)); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	if post.UserID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You aren't authorized to edit this post."})
		return
	}

	for i := range payload.Tags {
		payload.Tags[i] = strings.ToLower(payload.Tags[i])
	}
	allskills, err := p.CheckAndUpdateSkills(payload.Tags)
	if err != nil {
		log.Printf("An error occured while trying to add a new skill to db %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update post."})
		return
	}

	post.Description = payload.Description

	if err := p.DB.EditPost(&post, allskills); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	result := schema.PostResponse{
		ID:          post.ID,
		Description: post.Description,
		Tags:        payload.Tags,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		Views:       post.Views,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "Post updated successfully.", "post": result})
}

// EditPostView -> Ednpoint for updating a post view
func (p *Service) EditPostView(ctx *gin.Context) {
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
	if err := p.DB.FetchPostPreloadT(&post, uint(id)); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	if post.UserID != uid {
		post.Views++
		if err := p.DB.SavePost(&post); err != nil {
			cm := err.(*core.CustomMessage)
			ctx.JSON(cm.Code, gin.H{"message": cm.Message})
			return
		}
	}

	var tags []string
	for _, tag := range post.Tags {
		tags = append(tags, tag.Name)
	}
	result := schema.PostResponse{
		ID:          post.ID,
		Description: post.Description,
		Tags:        tags,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		Views:       post.Views,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"post": result})
}

// EditPostAvailability -> Endpoint for updating the post availability status
func (p *Service) EditPostAvailability(ctx *gin.Context) {
	uid, tp, pidStr, status := ctx.GetUint("userID"), ctx.GetString("purpose"), ctx.Query("id"), ctx.Query("status")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	pid, err := strconv.ParseUint(pidStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Invalid post id."})
		return
	}

	stat, err := strconv.ParseBool(status)
	if err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Invalid status"})
		return
	}

	var post model.Post
	if err := p.DB.FetchPost(&post, uint(pid)); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	post.Availability = stat
	if err := p.DB.SavePost(&post); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	var tags []string
	for _, tag := range post.Tags {
		tags = append(tags, tag.Name)
	}
	result := schema.PostResponse{
		ID:          post.ID,
		Description: post.Description,
		Tags:        tags,
		Views:       post.Views,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		Available:   post.Availability,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"post": result})
}

// SavePost -> Endpoint for saving a post
func (p *Service) SavePost(ctx *gin.Context) {
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
	if err := p.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	var post model.Post
	if err := p.DB.FetchPost(&post, uint(id)); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	if post.UserID == user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You can't save a post created by you."})
		return
	}

	if err := p.DB.BookmarkPost(&user, &post); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "Post saved successfully"})
}

// ViewSavedPost -> Endpoint for viewing saved post
func (p *Service) ViewSavedPost(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := p.DB.FetchUserPreloadB(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	var savedPosts []schema.PostResponse
	for _, post := range user.SavedPosts {
		var tags []string
		for _, tag := range post.Tags {
			tags = append(tags, tag.Name)
		}
		savedPosts = append(savedPosts, schema.PostResponse{
			ID:          post.ID,
			Description: post.Description,
			Tags:        tags,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
		})
	}
	ctx.JSON(http.StatusOK, gin.H{"post": savedPosts})
}

// ApplyForPost -> Endpoint for applying for a post
func (p *Service) ApplyForPost(ctx *gin.Context) {
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
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload"})
		return
	}

	var post model.Post
	if err := p.DB.FetchPostPreloadU(&post, uint(id)); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	var user model.User
	if err := p.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	if post.User.ID == user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You can't apply for a post owned by you."})
		return
	}

	if !post.Availability {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "The owner of the post is no longer accepting applications."})
		return
	}

	req := model.PostReq{
		PostID: post.ID,
		FromID: user.ID,
		ToID:   post.User.ID,
	}
	if len(payload.Message) > 0 {
		req.Message = payload.Message
	}

	if err := p.DB.AddPostApplicationReq(&req); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	p.Email.SendPostApplicationEmail(post.User.Email, user.UserName, post.User.UserName, post.Description, "nil")

	ctx.JSON(http.StatusOK, gin.H{"message": "Application sent successfully."})
}

// ViewPostApplications -> Endpoint for Viewing post applications
func (p *Service) ViewPostApplications(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := p.DB.ViewPostApplications(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	var recReq, sentReq []schema.ViewPostApplication
	for _, rq := range user.SentPostReq {
		sentReq = append(sentReq, schema.ViewPostApplication{
			ReqID:    rq.ID,
			Username: rq.ToUser.UserName,
			Message:  rq.Message,
			Status:   rq.Status,
		})
	}

	for _, rq := range user.RecPostReq {
		recReq = append(recReq, schema.ViewPostApplication{
			ReqID:    rq.ID,
			Username: rq.FromUser.UserName,
			Message:  rq.Message,
			Status:   rq.Status,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"post": gin.H{"rec_req": recReq, "sent_req": sentReq}})
}

// UpdatePostApplication -> Endpoint for Updating post applications
func (p *Service) UpdatePostApplication(ctx *gin.Context) {
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

	var req model.PostReq
	if err := p.DB.FetchPostApplication(&req, uint(rid)); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	var user, friend model.User
	if err := p.DB.FetchUserPreloadF(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	if err := p.DB.FetchUser(&user, req.FromID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
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
		if err := p.DB.UpdatePostAppliationReject(&req); err != nil {
			cm := err.(*core.CustomMessage)
			ctx.JSON(cm.Code, gin.H{"message": cm.Message})
			return
		}
		p.Email.SendPostApplicationReject(friend.Email, user.UserName, friend.UserName, req.Post.Description, "reason")
	case model.StatusAccepted:
		if err := p.DB.UpdatePostApplicationAccept(&req, &user, &friend, friends); err != nil {
			cm := err.(*core.CustomMessage)
			ctx.JSON(cm.Code, gin.H{"message": cm.Message})
			return
		}
		p.Email.SendPostApplicationAccept(friend.Email, user.UserName, friend.UserName, req.Post.Description, "")
	default:
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid status."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "Application status updated successfully."})
}

// DeletePostApplication -> Endpoint for deleting sent post application
func (p *Service) DeletePostApplication(ctx *gin.Context) {
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
	if err := p.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	var req model.PostReq
	if err := p.DB.FetchPostAppPreloadFU(&req, uint(rid)); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	if req.FromUser.ID != user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You don't have permission to delete this application."})
		return
	}

	if err := p.DB.DeletePostApplicationReq(&req); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// DeletePost -> Endpoint for deleting a post
func (p *Service) DeletePost(ctx *gin.Context) {
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
	if err := p.DB.FetchPost(&post, uint(id)); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	if post.UserID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You don't have permission to delete this post."})
		return
	}

	if err := p.DB.DeletePost(&post); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"message": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
