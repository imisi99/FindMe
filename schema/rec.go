package schema

type RecResponse struct {
	Res map[string]float32
}

type RecProjectResponse struct {
	Project ProjectResponse
	Score   float32
}

type RecProfileResponse struct {
	User  UserProfileResponse
	Score float32
}
