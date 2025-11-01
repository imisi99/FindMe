package unit

import "findme/schema"

type PostResponse struct {
	Message             string `json:"msg"`
	schema.PostResponse `json:"post"`
}
