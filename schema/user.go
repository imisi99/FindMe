package schema


type SignupRequest struct {
	FullName 	string 		`json:"fullname" binding:"required"`
	UserName 	string 		`json:"username" binding:"required"`
	Email 	string 			`json:"email" binding:"required,email"`
	Password 	string 		`json:"password" binding:"required,min=8"`
	Bio 	string 			`json:"bio" binding:"omitempty,max=50"`
	Skills 	[]string 		`json:"skills"`
}


type LoginRequest struct {
	UserName string		`json:"username" binding:"required"`
	Password string		`json:"password" binding:"required"`
}
