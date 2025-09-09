package schema

import "time"

type NewPostRequest struct {
	Description string				`json:"description" binding:"required"`
	Tags 		[]string			`json:"tags" binding:"required"`
}


type PostResponse struct {
	Description string
	Tags 		[]string
	CreatedAt 	time.Time
	UpdatedAt   time.Time
	Views 		uint
}


type DetailedPostResponse struct {
	PostResponse
	Username		string
}
