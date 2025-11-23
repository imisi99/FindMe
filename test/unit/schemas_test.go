package unit

import "findme/schema"

type ProjectResponse struct {
	Message                string `json:"msg"`
	schema.ProjectResponse `json:"project"`
}

type ProjectApplicationResponse struct {
	schema.ViewProjectApplication `json:"project_req"`
}

type ViewMsg struct {
	schema.ViewMessage `json:"msg"`
}

type ViewChats struct {
	schema.ViewChat `json:"msg"`
}

type Token struct {
	Token string `json:"token"`
}

type ViewFriendReq struct {
	schema.FriendReqStatus `json:"req"`
}

type ViewAllFriendReq struct {
	SentReq []schema.FriendReqStatus `json:"sent_req"`
	RecReq  []schema.FriendReqStatus `json:"rec_req"`
}

type GetChatID struct {
	Msg    string `json:"msg"`
	ChatID string `json:"chat_id"`
}
