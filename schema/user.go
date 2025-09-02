package schema


type SignupRequest struct {
	FullName 	string 		`json:"fullname" binding:"required"`
	UserName 	string 		`json:"username" binding:"required"`
	Email 	string 			`json:"email" binding:"required,email"`
	Password 	string 		`json:"password" binding:"required,min=8"`
	Bio 	string 			`json:"bio" binding:"omitempty,max=50"`
	Skills 	[]string 		`json:"skills"`
}


type GitHubUser struct {
	ID 			int64		`json:"id" binding:"required"`
	FullName	string		`json:"name" binding:"required"`
	UserName 	string		`json:"login" binding:"required"`
	Email		string		`json:"email"`
	Bio			string		`json:"bio"`
}


type LoginRequest struct {
	UserName string		`json:"username" binding:"required"`
	Password string		`json:"password" binding:"required"`
}


type UserProfileResponse struct {
	UserName	string		
	FullName	string		
	Email 		string
	GitUserName *string
	Bio 		string
	Availability bool
	Skills 		[]string
}


type UserProfileRequest struct {
	UserName	string		`json:"username" binding:"required"`
	FullName	string		`json:"fullname" binding:"required"`
	Email 		string		`json:"email" binding:"required,email"`
	GitUserName *string		`json:"gitusername" binding:"omitempty"`
	Bio 		string		`json:"bio" binding:"omitempty,max=50"`
}


type UserAvailabilityStatusRequest struct {
	Availability bool		`json:"availability" binding:"required"`
}


type UpdateUserSkillsRequest struct {
	Skills []string			`json:"skills" binding:"required"`
}


type DeleteUserSkillsRequest struct {
	Skills []string 		`json:"skills" binding:"required"`
}


type ForgotPasswordEmail struct {
	Email 		string 		`json:"email" binding:"required,email"`
}


type VerifyOTP struct {
	Token 		string		`json:"otp" binding:"required"`
}


type ResetPassword struct {
	Password 	string		`json:"password" binding:"required,min=8"`
}