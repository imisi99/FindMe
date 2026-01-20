package schema

type DocTokenResponse struct {
	Token string `json:"token"`
}

type DocNormalResponse struct {
	Msg string `json:"msg"`
}

type DocProjectUserResponse struct {
	User UserProfileResponse `json:"user"`
	Post []ProjectResponse   `json:"posts"`
}

type DocUserResponse struct {
	User UserProfileResponse `json:"user"`
}

type DocUsersResponse struct {
	Users []UserProfileResponse `json:"users"`
}

type DocUsersSearch struct {
	Users []SearchUser `json:"users"`
}

type DocFriendReqStatus struct {
	Req FriendReqStatus `json:"req"`
}

type DocViewFriendReqs struct {
	SentReq []FriendReqStatus `json:"sent_req"`
	RecReq  []FriendReqStatus `json:"rec_req"`
}

type DocFriendReqAccept struct {
	Msg    string `json:"msg"`
	ChatID string `json:"chat_id"`
}

type DocViewFriends struct {
	Friends []ViewFriends `json:"friends"`
}

type DocSkillsResponse struct {
	Skills []string `json:"skills"`
}

type DocProjectResponse struct {
	Project ProjectResponse `json:"project"`
}

type DocProjectsResponse struct {
	Projects []ProjectResponse `json:"projects"`
}

type DocAllProjectResponse struct {
	Project []ProjectResponse `json:"project"`
}

type DocDetailedProjectResponse struct {
	Project DetailedProjectResponse `json:"project"`
}

type DocViewProjectApplications struct {
	Req ApplicationProjectResponse `json:"req"`
}

type DocProjectApplication struct {
	ProjectReq ViewProjectApplication `json:"project_req"`
}

type DocViewAllProjectApplication struct {
	Project map[string]any `json:"project"`
}

type DocMsgResponse struct {
	Msg ViewMessage `json:"msg"`
}

type DocViewChatHistory struct {
	Msg ViewChat `json:"msg"`
}

type DocViewAllChats struct {
	Msg []ViewChat `json:"msg"`
}

type DocViewRepos struct {
	Repos []ViewRepo `json:"repos"`
}

type DocTranscResposne struct {
	Transactions []TransactionResponse
}

type DocViewSubscriptions struct {
	Subs []ViewSubscriptions
}
