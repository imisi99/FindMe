package unit

import "findme/schema"

type PostResponse struct {
	Message             string `json:"msg"`
	schema.PostResponse `json:"post"`
}

type PostApplicationResponse struct {
	schema.ViewPostApplication `json:"post_req"`
}

type ViewMsg struct {
	schema.ViewMessage `json:"msg"`
}

type ViewChats struct {
	schema.ViewChat `json:"msg"`
}
