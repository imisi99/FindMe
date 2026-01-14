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

// GetProjects godoc
// @Summary    Retreive all current user projects
// @Description An endpoint for retreiving all current user projects
// @Tags  Project
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocAllProjectResponse "Fetched all projects"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/posts/all [get]
func (s *Service) GetProjects(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserProjects(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var reuslt []schema.ProjectResponse
	for _, project := range user.Projects {
		var tags []string
		for _, tag := range project.Tags {
			tags = append(tags, tag.Name)
		}
		reuslt = append(reuslt, schema.ProjectResponse{
			ID:          project.ID,
			Title:       project.Title,
			Description: project.Description,
			Tags:        tags,
			CreatedAt:   project.CreatedAt,
			UpdatedAt:   project.UpdatedAt,
			Views:       project.Views,
			Available:   project.Availability,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"project": reuslt})
}

// ViewProject godoc
// @Summary     View a single project with ID
// @Description An endpoint for viewing a single project in details by using the project ID
// @Tags   Project
// @Accept json
// @Produce json
// @Param id query string true "Project ID"
// @Security BearerAuth
// @Success 200 {object} schema.DocDetailedProjectResponse "Fetched project"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/view [get]
func (s *Service) ViewProject(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if !model.IsValidUUID(pid) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid project id."})
		return
	}

	var project model.Project
	if err := s.DB.FetchProjectPreloadTU(&project, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var tags []string
	for _, tag := range project.Tags {
		tags = append(tags, tag.Name)
	}
	result := schema.DetailedProjectResponse{
		ProjectResponse: schema.ProjectResponse{
			ID:          project.ID,
			Title:       project.Title,
			Description: project.Description,
			Tags:        tags,
			CreatedAt:   project.CreatedAt,
			UpdatedAt:   project.UpdatedAt,
			Views:       project.Views,
			Available:   project.Availability,
		},
		Username:   project.User.UserName,
		GitProject: project.GitProject,
		GitLink:    project.GitLink,
	}

	ctx.JSON(http.StatusOK, gin.H{"project": result})
}

// ViewSingleProjectApplication godoc
// @Summary   View all applications on a project
// @Description An endpoint for viewing all the applications on a single project
// @Tags   Project
// @Accept json
// @Produce json
// @Param id query string true "Project ID"
// @Security BearerAuth
// @Success 200 {object} schema.DocViewProjectApplications "Project applications"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/view-application [get]
func (s *Service) ViewSingleProjectApplication(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	id := ctx.Query("id")
	if !model.IsValidUUID(id) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid project id."})
		return
	}

	var project model.Project
	if err := s.DB.FetchProjectPreloadA(&project, id); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if project.UserID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You don't have permission to view the applicants on this project."})
		return
	}

	var applications []schema.ViewProjectApplication
	for _, req := range project.Applications {
		applications = append(applications, schema.ViewProjectApplication{
			ReqID:    req.ID,
			Status:   req.Status,
			Message:  req.Message,
			Username: req.FromUser.UserName,
		})
	}
	result := schema.ApplicationProjectResponse{
		Applications: applications,
	}

	ctx.JSON(http.StatusOK, gin.H{"req": result})
}

// SearchProject godoc
// @Summary   Search for a project with tags/skills
// @Description An endpoint for searching for project with tags associated with the project
// @Tags Project
// @Accept json
// @Produce json
// @Param payload body schema.SearchProjectWithTags true "Tags"
// @Security BearerAuth
// @Success 200 {object} schema.DocAllProjectResponse "Projects"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/tags [post]
func (s *Service) SearchProject(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.SearchProjectWithTags
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload."})
		return
	}

	for i := range payload.Tags {
		payload.Tags[i] = strings.ToLower(payload.Tags[i])
	}

	var projects []model.Project
	if err := s.DB.SearchProjectsBySKills(&projects, payload.Tags, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var projectResponse []schema.ProjectResponse
	for _, project := range projects {
		var tags []string
		for _, tag := range project.Tags {
			tags = append(tags, tag.Name)
		}
		projectResponse = append(projectResponse, schema.ProjectResponse{
			ID:          project.ID,
			Title:       project.Title,
			Description: project.Description,
			CreatedAt:   project.CreatedAt,
			UpdatedAt:   project.UpdatedAt,
			Available:   project.Availability,
			Views:       project.Views,
			Tags:        tags,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"project": projectResponse})
}

// CreateProject godoc
// @Summary     Create a new project
// @Description  An endpoint for creating a new project for the current user it internally calls a service to create a vector for the project
// @Tags   Project
// @Accept json
// @Produce json
// @Param payload body schema.NewProjectRequest true "Project payload"
// @Security BearerAuth
// @Success 201 {object} schema.DocProjectResponse "Project created"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/create [post]
func (s *Service) CreateProject(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.NewProjectRequest
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to create new project."})
		return
	}

	project := model.Project{
		Title:        payload.Title,
		Description:  payload.Description,
		Tags:         allskills,
		UserID:       uid,
		Views:        0,
		Availability: true,
	}

	if payload.Git {
		project.GitProject = true
		project.GitLink = payload.GitLink
	}

	if err := s.DB.AddProject(&project); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	result := schema.ProjectResponse{
		ID:          project.ID,
		Title:       project.Title,
		Description: project.Description,
		Tags:        payload.Tags,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
		Views:       project.Views,
	}

	s.Emb.QueueProjectCreate(project.ID, project.Title, project.Description, payload.Tags)

	ctx.JSON(http.StatusCreated, gin.H{"project": result})
}

// RecommendUsers godoc
// @Summary Recommends users to work on a project
// @Description An endpoint for recommending users for a project using ai
// @Tags Project
// @Accept json
// @Produce json
// @Param id query string true "Project ID"
// @Security BearerAuth
// @Success 200 {object} schema.DocUsersResponse "Users Retrieved"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/recommend [get]
func (s *Service) RecommendUsers(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if !model.IsValidUUID(pid) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid project id."})
		return
	}

	var project model.Project
	if err := s.DB.FetchProject(&project, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	res, err := s.Rec.GetRecommendation(project.ID, core.UserRecommendation)
	if err != nil || res == nil {
		log.Printf("[gRPC Recommendation] Failed to get recommendation for project -> %v, err -> %v", project.ID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to retrieve users for the project."})
		return
	}

	var users []model.User

	if err := s.DB.FindUsers(&users, res.IDs); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var profiles []schema.UserProfileResponse

	for _, user := range users {
		var skills []string
		for _, skill := range user.Skills {
			skills = append(skills, skill.Name)
		}
		profiles = append(profiles,
			schema.UserProfileResponse{
				ID:           user.ID,
				UserName:     user.UserName,
				FullName:     user.FullName,
				Email:        user.Email,
				GitUserName:  user.GitUserName,
				Gituser:      user.GitUser,
				Bio:          user.Bio,
				Availability: user.Availability,
				Skills:       skills,
				Interests:    user.Interests,
			})
	}

	ctx.JSON(http.StatusOK, gin.H{"users": profiles})
}

// EditProject godoc
// @Summary    Editing details of a project
// @Description An endpoint for editing major details of a project it internally calls a service to update the vector for the project
// @Tags Project
// @Accept json
// @Produce json
// @Param id query string true "Project ID"
// @Param payload body schema.NewProjectRequest true "Project payload"
// @Security BearerAuth
// @Success 202 {object} schema.DocProjectResponse "Project Edited"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/edit [put]
func (s *Service) EditProject(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if !model.IsValidUUID(pid) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid project id."})
		return
	}

	var payload schema.NewProjectRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload."})
		return
	}

	var project model.Project
	if err := s.DB.FetchProjectPreloadT(&project, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if project.UserID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You aren't authorized to edit this project."})
		return
	}

	for i := range payload.Tags {
		payload.Tags[i] = strings.ToLower(payload.Tags[i])
	}

	allskills, err := s.CheckAndUpdateSkills(payload.Tags)
	if err != nil {
		log.Printf("An error occured while trying to add a new skill to db %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to update project."})
		return
	}

	project.Title = payload.Title
	project.Description = payload.Description

	if payload.Git {
		project.GitProject = true
		project.GitLink = payload.GitLink
	}

	if err := s.DB.EditProject(&project, allskills); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	result := schema.ProjectResponse{
		ID:          project.ID,
		Title:       project.Title,
		Description: project.Description,
		Tags:        payload.Tags,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
		Views:       project.Views,
	}

	s.Emb.QueueProjectUpdate(project.ID, project.Title, project.Description, payload.Tags)

	ctx.JSON(http.StatusAccepted, gin.H{"project": result})
}

// EditProjectView godoc
// @Summary    Editing the number of views on a project
// @Description An endpoint for editing the number of views on a project
// @Tags Project
// @Accept json
// @Produce json
// @Param id query string true "Project ID"
// @Security BearerAuth
// @Success 202 {object} schema.DocProjectResponse "Project view edited"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/edit-view [patch]
func (s *Service) EditProjectView(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	id := ctx.Query("id")
	if !model.IsValidUUID(id) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid project id."})
		return
	}

	var project model.Project
	if err := s.DB.FetchProjectPreloadT(&project, id); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if project.UserID != uid {
		project.Views++
		if err := s.DB.SaveProject(&project); err != nil {
			cm := err.(*core.CustomMessage)
			ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
			return
		}
	}

	var tags []string
	for _, tag := range project.Tags {
		tags = append(tags, tag.Name)
	}
	result := schema.ProjectResponse{
		ID:          project.ID,
		Title:       project.Title,
		Description: project.Description,
		Tags:        tags,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
		Views:       project.Views,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"project": result})
}

// EditProjectAvailability godoc
// @Summary     Editing the availability of a project
// @Description An endpoint for editing the availability status of a project it internally calls a service to update the vector for the project
// @Tags Project
// @Accept json
// @Produce json
// @Param id query string true "Project ID"
// @Param staus query string true "Status"
// @Security BearerAuth
// @Success 202 {object} schema.DocProjectResponse "Project edited"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/edit-status [patch]
func (s *Service) EditProjectAvailability(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid, status := ctx.Query("id"), ctx.Query("status")
	if !model.IsValidUUID(pid) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid project id."})
		return
	}

	stat, err := strconv.ParseBool(status)
	if err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Invalid status"})
		return
	}

	var project model.Project
	if err := s.DB.FetchProject(&project, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if project.UserID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You aren't authorized to edit this project."})
		return
	}

	project.Availability = stat
	if err := s.DB.SaveProject(&project); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var tags []string
	for _, tag := range project.Tags {
		tags = append(tags, tag.Name)
	}
	result := schema.ProjectResponse{
		ID:          project.ID,
		Title:       project.Title,
		Description: project.Description,
		Tags:        tags,
		Views:       project.Views,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
		Available:   project.Availability,
	}

	s.Emb.QueueProjectUpdateStatus(project.ID, project.Availability)

	ctx.JSON(http.StatusAccepted, gin.H{"project": result})
}

// SaveProject godoc
// @Summary     Bookmark a project
// @Description An endpoint for adding a project to the current user bookmarks
// @Tags Project
// @Accept json
// @Produce json
// @Param id query string true "Project ID"
// @Security BearerAuth
// @Success 202 {object} schema.DocProjectResponse "Bookmarked project"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/save-post [put]
func (s *Service) SaveProject(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if !model.IsValidUUID(pid) {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid project id."})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var project model.Project
	if err := s.DB.FetchProject(&project, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if project.UserID == user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You can't save a project created by you."})
		return
	}

	if err := s.DB.BookmarkProject(&user, &project); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var tags []string
	for _, tag := range project.Tags {
		tags = append(tags, tag.Name)
	}

	projectRes := schema.ProjectResponse{
		ID:          project.ID,
		Title:       project.Title,
		Description: project.Description,
		Available:   project.Availability,
		Tags:        tags,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
		Views:       project.Views,
	}
	ctx.JSON(http.StatusAccepted, gin.H{"project": projectRes})
}

// ViewSavedProject godoc
// @Summary    View all current user bookmarked projects
// @Description An endpoint to view all of the bookmarked projects of the current user
// @Tags Project
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocAllProjectResponse "Bookmarked projects"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/view/saved-post [get]
func (s *Service) ViewSavedProject(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadB(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var savedProjects []schema.ProjectResponse
	for _, project := range user.SavedProjects {
		var tags []string
		for _, tag := range project.Tags {
			tags = append(tags, tag.Name)
		}
		savedProjects = append(savedProjects, schema.ProjectResponse{
			ID:          project.ID,
			Title:       project.Title,
			Description: project.Description,
			Tags:        tags,
			CreatedAt:   project.CreatedAt,
			UpdatedAt:   project.UpdatedAt,
		})
	}
	ctx.JSON(http.StatusOK, gin.H{"project": savedProjects})
}

// RemoveSavedProject godoc
// @Summary    Remove a project from bookmarked
// @Description An endpoint for removing a project from the current user bookmarked
// @Tags Project
// @Accept json
// @Produce json
// @Param id query string true "Project ID"
// @Security BearerAuth
// @Success 204 {object} nil "Project removed"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
func (s *Service) RemoveSavedProject(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if !model.IsValidUUID(pid) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid project id."})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var project model.Project
	if err := s.DB.FetchProject(&project, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := s.DB.RemoveBookmarkedProject(&user, &project); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// ApplyForProject godoc
// @Summary   Apply for a project to work on
// @Description An endpoint for applying to a project to work on
// @Tags Project
// @Accept json
// @Produce json
// @Param id query string true "Project ID"
// @Param payload body schema.ProjectApplication true "Application payload"
// @Security BearerAuth
// @Success 200 {object} schema.DocProjectApplication "Applied successfully"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 409 {object} schema.DocNormalResponse "Existing record"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/apply [post]
func (s *Service) ApplyForProject(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if !model.IsValidUUID(pid) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid project id."})
		return
	}

	var payload schema.ProjectApplication
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload"})
		return
	}

	var project model.Project
	if err := s.DB.FetchProjectPreloadU(&project, pid); err != nil {
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

	if !project.Availability {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "The owner of the project is no longer accepting applications."})
		return
	}

	if err, exists := s.DB.CheckExistingAppReq(pid, uid); err != nil || exists {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if project.UserID == uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You can't apply for your own project."})
		return
	}

	req := model.ProjectReq{
		ProjectID: project.ID,
		FromID:    user.ID,
		ToID:      project.User.ID,
	}

	if len(payload.Message) > 0 {
		req.Message = payload.Message
	}

	if err := s.DB.AddProjectApplicationReq(&req); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	application := schema.ViewProjectApplication{
		ReqID:    req.ID,
		Status:   req.Status,
		Message:  req.Message,
		Username: project.User.UserName,
	}

	s.Email.QueueProjectApplication(user.UserName, project.User.UserName, project.Description, "nil", project.User.Email)

	ctx.JSON(http.StatusOK, gin.H{"project_req": application})
}

// ViewProjectApplications godoc
// @Summary    View all project applications sent and received
// @Description An endpoint for viewing all sent and received project applications
// @Tags Project
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocViewAllProjectApplication "Fetched applications"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/view-applications [get]
func (s *Service) ViewProjectApplications(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.ViewProjectApplications(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var recReq, sentReq []schema.ViewProjectApplication
	for _, rq := range user.SentProjectReq {
		sentReq = append(sentReq, schema.ViewProjectApplication{
			ReqID:    rq.ID,
			Username: rq.ToUser.UserName,
			Message:  rq.Message,
			Status:   rq.Status,
			Sent:     rq.CreatedAt,
		})
	}

	for _, rq := range user.RecProjectReq {
		recReq = append(recReq, schema.ViewProjectApplication{
			ReqID:    rq.ID,
			Username: rq.FromUser.UserName,
			Message:  rq.Message,
			Status:   rq.Status,
			Sent:     rq.CreatedAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"project": gin.H{"rec_req": recReq, "sent_req": sentReq}})
}

// UpdateProjectApplication godoc
// @Summary     Updating a project application status
// @Description An endpoint for updating a project application status to accepted or rejected
// @Tags Project
// @Accept json
// @Produce json
// @Param id query string true "Request ID"
// @Param status query string true "Request status"
// @Param payload body schema.RejectApplication true "Payload"
// @Security BearerAuth
// @Success 202 {object} schema.DocNormalResponse "Status Updated"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/update-application [patch]
func (s *Service) UpdateProjectApplication(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}
	rid, status := ctx.Query("id"), ctx.Query("status")
	if !model.IsValidUUID(rid) || status == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid req id."})
		return
	}

	var payload schema.RejectApplication
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, "Failed to parse the payload.")
		return
	}

	if payload.Reason == "" {
		payload.Reason = "The author of the project didn't add a reason."
	}

	var req model.ProjectReq
	if err := s.DB.FetchProjectApplication(&req, rid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if req.ToID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You don't have permission to update this application."})
		return
	}

	switch status {
	case model.StatusRejected:
		if err := s.DB.UpdateProjectAppliationReject(&req); err != nil {
			cm := err.(*core.CustomMessage)
			ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
			return
		}

		s.Email.QueueProjectApplicationReject(req.ToUser.UserName, req.FromUser.UserName, req.Project.Description, payload.Reason, req.FromUser.Email)

	case model.StatusAccepted:
		var err error

		if req.Project.ChatID == nil {
			var chat model.Chat
			chat.Group = true
			chat.OwnerID = &uid
			err = s.DB.UpdateProjectApplicationAcceptF(&req, req.ToUser, req.FromUser, req.Project, &chat)
		} else {
			err = s.DB.UpdateProjectApplicationAccept(&req, req.FromUser, req.Project.Chat)
		}

		if err != nil {
			cm := err.(*core.CustomMessage)
			ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
			return
		}

		s.Email.QueueProjectApplicationAccept(req.ToUser.UserName, req.FromUser.UserName, req.Project.Description, "", req.FromUser.Email)

	default:
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid status."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "Application status updated successfully."})
}

// DeleteProjectApplication godoc
// @Summary    Delete a send project application
// @Description An endpoint for deleting a sent project application for the current user
// @Tags Project
// @Accept json
// @Produce json
// @Param id query string true "Request ID"
// @Security BearerAuth
// @Success 204 {object} nil "Request deleted"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/delete-application [delete]
func (s *Service) DeleteProjectApplication(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	rid := ctx.Query("id")
	if !model.IsValidUUID(rid) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid request id."})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var req model.ProjectReq
	if err := s.DB.FetchProjectAppPreloadFU(&req, rid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if req.FromUser.ID != user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You don't have permission to delete this application."})
		return
	}

	if err := s.DB.DeleteProjectApplicationReq(&req); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// ClearProjectApplication godoc
// @Summary     Clear all applications on a project
// @Description An endpoint for clearing all applications on a project
// @Tags  Project
// @Accept json
// @Produce json
// @Param id query string true "Project ID"
// @Security BearerAuth
// @Success 204 {object} nil "Applications cleared"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/post/clear-application [delete]
func (s *Service) ClearProjectApplication(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if !model.IsValidUUID(pid) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid project id."})
		return
	}

	var project model.Project
	if err := s.DB.FetchProjectPreloadA(&project, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if project.UserID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You don't have permission to clear this project applications"})
		return
	}

	if len(project.Applications) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "No request to clear!"})
		return
	}

	if err := s.DB.ClearProjectApplication(project.Applications); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// DeleteProject godoc
// @Summary    Delete a project
// @Description An endpoint for deleting the current user project it internally calls a service to delete the vector for the project
// @Tags   Project
// @Accept json
// @Produce json
// @Param id query string true "Project ID"
// @Success 204 {object} nil "Project deleted"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
func (s *Service) DeleteProject(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
	if !model.IsValidUUID(pid) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid project id."})
		return
	}

	var project model.Project
	if err := s.DB.FetchProject(&project, pid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if project.UserID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You don't have permission to delete this project."})
		return
	}

	if err := s.DB.DeleteProject(&project); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	s.Emb.QueueProjectDelete(project.ID)

	ctx.JSON(http.StatusNoContent, nil)
}
