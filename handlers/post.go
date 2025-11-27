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

// GetProjects -> Endpoint for getting all user projects
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

// ViewProject -> Endpoint for viewing a single project
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

// ViewSingleProjectApplication -> Endpoint for viewing a project applications
func (s *Service) ViewSingleProjectApplication(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	id := ctx.Query("id")
	if id == "" {
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

// SearchProject -> Endpoint for searching project with tags
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

// CreateProject -> Endpoint for creating project
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
		Description: project.Description,
		Tags:        payload.Tags,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
		Views:       project.Views,
	}

	ctx.JSON(http.StatusCreated, gin.H{"project": result})
}

// EditProject -> Endpoint for editing project
func (s *Service) EditProject(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid := ctx.Query("id")
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
		Description: project.Description,
		Tags:        payload.Tags,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
		Views:       project.Views,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"project": result})
}

// EditProjectView -> Endpoint for updating a project view
func (s *Service) EditProjectView(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	id := ctx.Query("id")
	if id == "" {
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
		Description: project.Description,
		Tags:        tags,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
		Views:       project.Views,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"project": result})
}

// EditProjectAvailability -> Endpoint for updating the project availability status
func (s *Service) EditProjectAvailability(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	pid, status := ctx.Query("id"), ctx.Query("status")
	if !model.IsValidUUID(pid) {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Invalid project id."})
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
		Description: project.Description,
		Tags:        tags,
		Views:       project.Views,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
		Available:   project.Availability,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"project": result})
}

// SaveProject -> Endpoint for saving a project
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
		Description: project.Description,
		Available:   project.Availability,
		Tags:        tags,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
		Views:       project.Views,
	}
	ctx.JSON(http.StatusAccepted, gin.H{"project": projectRes})
}

// ViewSavedProject -> Endpoint for viewing saved project
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
			Description: project.Description,
			Tags:        tags,
			CreatedAt:   project.CreatedAt,
			UpdatedAt:   project.UpdatedAt,
		})
	}
	ctx.JSON(http.StatusOK, gin.H{"project": savedProjects})
}

// RemoveSavedProject -> Endpoint for removing saved project
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

// ApplyForProject -> Endpoint for applying for a project
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
	_ = s.Email.SendProjectApplicationEmail(project.User.Email, user.UserName, project.User.UserName, project.Description, "nil")

	ctx.JSON(http.StatusOK, gin.H{"project_req": application})
}

// ViewProjectApplications -> Endpoint for Viewing project applications
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

// UpdateProjectApplication -> Endpoint for Updating project applications
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
		_ = s.Email.SendProjectApplicationReject(req.FromUser.Email, req.ToUser.UserName, req.FromUser.UserName, req.Project.Description, payload.Reason)
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

		_ = s.Email.SendProjectApplicationAccept(req.FromUser.Email, req.ToUser.UserName, req.FromUser.UserName, req.Project.Description, "")
	default:
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid status."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "Application status updated successfully."})
}

// DeleteProjectApplication -> Endpoint for deleting sent project application
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

// ClearProjectApplication -> Endpoint for clearing a project applications
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

// DeleteProject -> Endpoint for deleting a project
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

	ctx.JSON(http.StatusNoContent, nil)
}
