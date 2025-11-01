package schema

import "time"

type NewPostRequest struct {
	Description string   `json:"description" binding:"required"`
	Tags        []string `json:"tags" binding:"required"`
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
	Username string
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
}
