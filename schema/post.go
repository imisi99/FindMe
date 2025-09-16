package schema

import "time"

type NewPostRequest struct {
	Description string				`json:"description" binding:"required"`
	Tags 		[]string			`json:"tags" binding:"required"`
}


type PostResponse struct {
	ID			uint
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


type PostApplication struct {
	Message 		string 			`json:"msg" binding:"omitempty,max=50"`
}


type ViewPostApplication struct {
	ReqID 			uint
	Status 			string
	Message 		string
	Username 		string
}
