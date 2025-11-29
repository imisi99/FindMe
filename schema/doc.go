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
