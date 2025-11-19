package schema

import "time"

type NewPostRequest struct {
	Description string   `json:"description" binding:"required"`
	Tags        []string `json:"tags" binding:"required"`
	Git         bool     `json:"git" binding:"omitempty"`
	GitLink     string   `json:"git_link" binding:"omitempty"`
}

type PostResponse struct {
	ID          string
	Description string
	Available   bool
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Views       uint
}

type DetailedPostResponse struct {
	PostResponse
	Username   string
	GitProject bool
	GitLink    string
}

type ApplicationPostResponse struct {
	Applications []ViewPostApplication
}

type PostApplication struct {
	Message string `json:"msg" binding:"omitempty,max=50"`
}

type SearchPostWithTags struct {
	Tags []string `json:"tags" binding:"required"`
}

type ViewPostApplication struct {
	ReqID    string
	Status   string
	Message  string
	Username string
	Sent     time.Time
}
