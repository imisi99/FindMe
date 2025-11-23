package schema

import "time"

type NewProjectRequest struct {
	Description string   `json:"description" binding:"required"`
	Tags        []string `json:"tags" binding:"required"`
	Git         bool     `json:"git" binding:"omitempty"`
	GitLink     string   `json:"git_link" binding:"omitempty"`
}

type ProjectResponse struct {
	ID          string
	Description string
	Available   bool
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Views       uint
}

type DetailedProjectResponse struct {
	ProjectResponse
	Username   string
	GitProject bool
	GitLink    string
}

type ApplicationProjectResponse struct {
	Applications []ViewProjectApplication
}

type RejectApplication struct {
	Reason string `json:"msg" binding:"omitempty"`
}

type ProjectApplication struct {
	Message string `json:"msg" binding:"omitempty,max=50"`
}

type SearchProjectWithTags struct {
	Tags []string `json:"tags" binding:"required"`
}

type ViewProjectApplication struct {
	ReqID    string
	Status   string
	Message  string
	Username string
	Sent     time.Time
}
