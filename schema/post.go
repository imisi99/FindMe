package schema


type NewPostRequest struct {
	Description string				`json:"description" binding:"required"`
	Tags 		[]string			`json:"tags" binding:"required"`
}
