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

// TODO:
// Add a better way to check for already applied post in

// Maybe an endpoint to add the project to a github project?
// DONE:
// Should there also be a applications on a post for easy tracking ? (This can also be used to check for existing req to a post)
// Possibly a chat group to be associated to the post nah (This can be used instead of enforcing a friendship)
// Remove user's post in the search for post tags ?
// An Endpoint to clear all applications on a post or rejected one ?
// Remodel requests to delete after ignored or rejected

// GetPosts -> Endpoint for getting all user posts
func (s *Service) GetPosts(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPosts(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
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
func (s *Service) ViewPost(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if pid == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid post id."})
		return
	}

	var post model.Post
	if err := s.DB.FetchPostPreloadTU(&post, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
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
		Username:   post.User.UserName,
		GitProject: post.GitProject,
		GitLink:    post.GitLink,
	}

	ctx.JSON(http.StatusOK, gin.H{"post": result})
}

// ViewSinglePostApplication -> Endpoint for viewing a post applications
func (s *Service) ViewSinglePostApplication(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	id := ctx.Query("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid post id."})
		return
	}

	var post model.Post
	if err := s.DB.FetchPostPreloadA(&post, id); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if post.UserID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You don't have permission to view the applicants on this post."})
		return
	}

	var applications []schema.ViewPostApplication
	for _, req := range post.Applications {
		applications = append(applications, schema.ViewPostApplication{
			ReqID:    req.ID,
			Status:   req.Status,
			Message:  req.Message,
			Username: req.FromUser.UserName,
		})
	}
	result := schema.ApplicationPostResponse{
		Applications: applications,
	}

	ctx.JSON(http.StatusOK, gin.H{"req": result})
} // SearchPost -> Endpoint for searching post with tags
func (s *Service) SearchPost(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.SearchPostWithTags
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload."})
		return
	}

	for i := range payload.Tags {
		payload.Tags[i] = strings.ToLower(payload.Tags[i])
	}

	var posts []model.Post
	if err := s.DB.SearchPostsBySKills(&posts, payload.Tags, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
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
func (s *Service) CreatePost(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.NewPostRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload."})
		return
	}
	log.Println(payload.Git)

	for i := range payload.Tags {
		payload.Tags[i] = strings.ToLower(payload.Tags[i])
	}
	allskills, err := s.CheckAndUpdateSkills(payload.Tags)
	if err != nil {
		log.Printf("An error occured while trying to add a new skill to db -> %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to create new post."})
		return
	}

	post := model.Post{
		Description:  payload.Description,
		Tags:         allskills,
		UserID:       uid,
		Views:        0,
		Availability: true,
	}

	if payload.Git {
		post.GitProject = true
		post.GitLink = payload.GitLink
	}

	if err := s.DB.AddPost(&post); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
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

	ctx.JSON(http.StatusCreated, gin.H{"post": result})
}

// EditPost -> Endpoint for editing post
func (s *Service) EditPost(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	var payload schema.NewPostRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload."})
		return
	}

	var post model.Post
	if err := s.DB.FetchPostPreloadT(&post, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if post.UserID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You aren't authorized to edit this post."})
		return
	}

	for i := range payload.Tags {
		payload.Tags[i] = strings.ToLower(payload.Tags[i])
	}

	allskills, err := s.CheckAndUpdateSkills(payload.Tags)
	if err != nil {
		log.Printf("An error occured while trying to add a new skill to db %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to update post."})
		return
	}

	post.Description = payload.Description

	if err := s.DB.EditPost(&post, allskills); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
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

	ctx.JSON(http.StatusAccepted, gin.H{"post": result})
}

// EditPostView -> Ednpoint for updating a post view
func (s *Service) EditPostView(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	id := ctx.Query("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid post id."})
		return
	}

	var post model.Post
	if err := s.DB.FetchPostPreloadT(&post, id); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if post.UserID != uid {
		post.Views++
		if err := s.DB.SavePost(&post); err != nil {
			cm := err.(*core.CustomMessage)
			ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
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
func (s *Service) EditPostAvailability(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid, status := ctx.Query("id"), ctx.Query("status")
	if pid == "" {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Invalid post id."})
		return
	}

	stat, err := strconv.ParseBool(status)
	if err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Invalid status"})
		return
	}

	var post model.Post
	if err := s.DB.FetchPost(&post, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if post.UserID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You aren't authorized to edit this post."})
		return
	}

	post.Availability = stat
	if err := s.DB.SavePost(&post); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
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
func (s *Service) SavePost(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if pid == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid post id."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadB(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var post model.Post
	if err := s.DB.FetchPost(&post, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if post.UserID == user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You can't save a post created by you."})
		return
	}

	if err := s.DB.BookmarkPost(&user, &post); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var tags []string
	for _, tag := range post.Tags {
		tags = append(tags, tag.Name)
	}

	postRes := schema.PostResponse{
		ID:          post.ID,
		Description: post.Description,
		Available:   post.Availability,
		Tags:        tags,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		Views:       post.Views,
	}
	ctx.JSON(http.StatusAccepted, gin.H{"post": postRes})
}

// ViewSavedPost -> Endpoint for viewing saved post
func (s *Service) ViewSavedPost(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadB(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
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

// RemoveSavedPost -> Endpoint for removing saved post
func (s *Service) RemoveSavedPost(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if pid == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid post id."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadB(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var post model.Post
	if err := s.DB.FetchPost(&post, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := s.DB.RemoveBookmarkedPost(&user, &post); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// ApplyForPost -> Endpoint for applying for a post
func (s *Service) ApplyForPost(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if pid == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid post id."})
		return
	}

	var payload schema.PostApplication
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload"})
		return
	}

	var post model.Post
	if err := s.DB.FetchPostPreloadU(&post, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	for _, req := range post.Applications {
		if req.FromID == uid {
			ctx.JSON(http.StatusConflict, gin.H{"msg": "You have already submitted a request to this post."})
			return
		}
	}

	if post.UserID == uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You can't apply for your own post."})
		return
	}

	if !post.Availability {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "The owner of the post is no longer accepting applications."})
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

	if err := s.DB.AddPostApplicationReq(&req); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	application := schema.ViewPostApplication{
		ReqID:    req.ID,
		Status:   req.Status,
		Message:  req.Message,
		Username: post.User.UserName,
	}
	_ = s.Email.SendPostApplicationEmail(post.User.Email, user.UserName, post.User.UserName, post.Description, "nil")

	ctx.JSON(http.StatusOK, gin.H{"post_req": application})
}

// ViewPostApplications -> Endpoint for Viewing post applications
func (s *Service) ViewPostApplications(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.ViewPostApplications(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
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
func (s *Service) UpdatePostApplication(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}
	rid, status := ctx.Query("id"), ctx.Query("status")
	if rid == "" || status == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid req id."})
		return
	}

	var req model.PostReq
	if err := s.DB.FetchPostApplication(&req, rid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var post model.Post
	if err := s.DB.FetchPostPreloadC(&post, req.PostID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var user, applicant model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := s.DB.FetchUser(&applicant, req.FromID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if req.ToID != user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You don't have permission to update this application."})
		return
	}

	switch status {
	case model.StatusRejected:
		if err := s.DB.UpdatePostAppliationReject(&req); err != nil {
			cm := err.(*core.CustomMessage)
			ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
			return
		}
		_ = s.Email.SendPostApplicationReject(applicant.Email, user.UserName, applicant.UserName, post.Description, "reason")
	case model.StatusAccepted:
		var err error
		var chat model.Chat
		chat.Group = true
		chat.OwnerID = &uid

		if post.ChatID == nil {
			err = s.DB.UpdatePostApplicationAcceptF(&req, &user, &applicant, &post, &chat)
		} else {
			err = s.DB.UpdatePostApplicationAccept(&req, &applicant, post.Chat)
		}

		if err != nil {
			cm := err.(*core.CustomMessage)
			ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
			return
		}

		_ = s.Email.SendPostApplicationAccept(applicant.Email, user.UserName, applicant.UserName, req.Post.Description, "")
	default:
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid status."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "Application status updated successfully."})
}

// DeletePostApplication -> Endpoint for deleting sent post application
func (s *Service) DeletePostApplication(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	rid := ctx.Query("id")
	if rid == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid request id."})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var req model.PostReq
	if err := s.DB.FetchPostAppPreloadFU(&req, rid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if req.FromUser.ID != user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You don't have permission to delete this application."})
		return
	}

	if err := s.DB.DeletePostApplicationReq(&req); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// ClearPostApplication -> Endpoint for clearing a post applications
func (s *Service) ClearPostApplication(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if pid == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid post id."})
		return
	}

	var post model.Post
	if err := s.DB.FetchPostPreloadA(&post, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := s.DB.ClearPostApplication(post.Applications); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// DeletePost -> Endpoint for deleting a post
func (s *Service) DeletePost(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if pid == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid post id."})
		return
	}

	var post model.Post
	if err := s.DB.FetchPost(&post, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if post.UserID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You don't have permission to delete this post."})
		return
	}

	if err := s.DB.DeletePost(&post); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
