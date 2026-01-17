package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

type Git interface {
	GitHubAddUser(ctx *gin.Context)
	SelectCallback(ctx *gin.Context)
	ConnectGitHub(ctx *gin.Context)
	ConnectGitHubCallback(uid, code, state, storedState string) (string, error)
	GitHubAddUserCallback(token, code, state, storedState string) (string, string, error)
	ViewRepo(ctx *gin.Context)
}

type GitService struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
	DB           core.DB
	EmbHub       *core.EmbeddingHub
	Client       *http.Client
}

func NewGitService(id, secret, callback string, db core.DB, emb *core.EmbeddingHub, client *http.Client) *GitService {
	return &GitService{ClientID: id, ClientSecret: secret, CallbackURL: callback, DB: db, EmbHub: emb, Client: client}
}

// GitHubAddUser godoc
// @Summary Signing up user using github
// @Description A redirecting endpoint for sign-in / sign-up with github it calls a service internally to create a vector for new users
// @Tags Auth
// @Accept json
// @Produce json
// @Success 307 {string} string "Redirecting"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /github-signup [get]
func (g *GitService) GitHubAddUser(ctx *gin.Context) {
	if _, err := ctx.Cookie("git-access-token"); err == nil {
		ctx.SetCookie("auth", "login", 150, "/", "", false, true)
		ctx.Redirect(http.StatusTemporaryRedirect, g.CallbackURL)
	}
	state, err := core.GenerateState()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate user state."})
		return
	}

	ctx.SetCookie("state", state, 150, "/", "", false, true)
	ctx.SetCookie("auth", "login", 150, "/", "", false, true)

	redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=read:user user:email", g.ClientID, state)
	ctx.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// ConnectGitHub godoc
// @Summary Connecting github account to user account
// @Description An redirecting endpoint for connecting a github account to the current user account
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 307 {string} string "Redirecting"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/user/connect-github [post]
func (g *GitService) ConnectGitHub(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	state, err := core.GenerateState()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to generate state for github session."})
		return
	}

	ctx.SetCookie("state", state, 150, "/", "", false, true)
	ctx.SetCookie("auth", "connect", 150, "/", "", false, true)
	ctx.SetCookie("uid", uid, 150, "/", "", false, true)

	redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=read:user user:email", g.ClientID, state)
	ctx.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// SelectCallback godoc
// @Summary  Callback for the github sign-in / connect
// @Description An endpoint for selecting the callback for login, connect github endpoint
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} schema.DocTokenResponse "Token generated"
// @Success 202 {object} schema.DocNormalResponse "Github Connected"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 409 {object} schema.DocNormalResponse "Existing record"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Failure 502 {object} schema.DocNormalResponse "Bad Gateway"
// @Router /api/v1/auth/github/callback [get]
func (g *GitService) SelectCallback(ctx *gin.Context) {
	auth, _ := ctx.Cookie("auth")
	token, _ := ctx.Cookie("git-access-token")
	storedState, _ := ctx.Cookie("state")
	uid, _ := ctx.Cookie("uid")

	state := ctx.Query("state")
	code := ctx.Query("code")
	switch auth {
	case "login":
		jwtToken, gitToken, err := g.GitHubAddUserCallback(token, code, state, storedState)
		if err != nil {
			cm := err.(*core.CustomMessage)
			ctx.AbortWithStatusJSON(cm.Code, gin.H{"msg": cm.Message})
			return
		}
		ctx.SetCookie("git-access-token", gitToken, 60*60*24, "/", "", false, true)
		ctx.JSON(http.StatusOK, gin.H{"token": jwtToken})
		return
	case "auth":
		gitToken, err := g.ConnectGitHubCallback(uid, code, state, storedState)
		if err != nil {
			cm := err.(*core.CustomMessage)
			ctx.AbortWithStatusJSON(cm.Code, gin.H{"msg": cm.Message})
			return
		}
		ctx.SetCookie("git-access-token", gitToken, 60*60*24, "/", "", false, true)
		ctx.JSON(http.StatusAccepted, gin.H{"msg": "Github account connected successfully."})
		return
	default:
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid auth mode."})
	}
}

// GitHubAddUserCallback -> callback for the github sign-up/sign-in endpoint
func (g *GitService) GitHubAddUserCallback(token, code, state, storedState string) (string, string, error) {
	if token == "" {
		if state != storedState {
			return "", "", &core.CustomMessage{Code: http.StatusBadRequest, Message: "Invalid or expired state."}
		}

		data := url.Values{}
		data.Add("client_id", g.ClientID)
		data.Add("client_secret", g.ClientSecret)
		data.Add("code", code)

		req, _ := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", bytes.NewBufferString(data.Encode()))
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-type", "application/x-www-form-urlencoded")

		resp, err := g.Client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			return "", "", &core.CustomMessage{Code: http.StatusBadGateway, Message: "Failed to signup with github."}
		}

		defer resp.Body.Close()

		var gitToken schema.GitToken

		if err := json.NewDecoder(resp.Body).Decode(&gitToken); err != nil {
			return "", "", &core.CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to parse access token."}
		}
		token = gitToken.AccessToken
	}

	userReq, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+token)

	userResp, err := g.Client.Do(userReq)
	if err != nil || userResp.StatusCode != http.StatusOK {
		return "", "", &core.CustomMessage{Code: http.StatusBadGateway, Message: "Failed to signup with github."}
	}

	defer userResp.Body.Close()

	var user schema.GitHubUser
	if err := json.NewDecoder(userResp.Body).Decode(&user); err != nil {
		return "", "", &core.CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to parse user info."}
	}

	var existingUser model.User
	if err := g.DB.FindExistingGitID(&existingUser, user.ID); err == nil {
		premium := core.CheckSubscription(&existingUser)
		userToken, err := GenerateJWT(existingUser.ID, "login", premium, JWTExpiry)
		if err != nil {
			return "", "", &core.CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to generate jwt token for user."}
		}
		return userToken, token, nil
	}

	if user.Email == "" {
		emailReq, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user/emails", nil)
		emailReq.Header.Set("Authorization", "Bearer "+token)
		emailReq.Header.Set("Accept", "application/vnd.github+json")
		emailResp, err := g.Client.Do(emailReq)

		if err != nil || emailResp.StatusCode != http.StatusOK {
			return "", "", &core.CustomMessage{Code: http.StatusBadGateway, Message: "Failed to fetch user email."}
		}

		defer emailResp.Body.Close()

		var email []struct {
			Email   string `json:"email"`
			Primary bool   `json:"primary"`
		}

		if err := json.NewDecoder(emailResp.Body).Decode(&email); err != nil {
			return "", "", &core.CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to parse user github emails."}
		}

		for _, e := range email {
			if e.Primary {
				user.Email = e.Email
				break
			}
		}

		if user.Email == "" {
			return "", "", &core.CustomMessage{Code: http.StatusBadRequest, Message: "No email available on github."}
		}

	}

	if err := g.DB.CheckExistingEmail(user.Email); err != nil {
		return "", "", &core.CustomMessage{Code: http.StatusConflict, Message: "There's an account associated with that email already!"}
	}

	newUsername := user.UserName
	if err := g.DB.CheckExistingUsername(newUsername); err != nil {
		newUsername = core.GenerateUsername(existingUser.UserName)
	}

	newUser := model.User{
		FullName:     user.FullName,
		Email:        user.Email,
		GitUserName:  &user.UserName,
		GitID:        &user.ID,
		GitUser:      true,
		UserName:     newUsername,
		Availability: true,
		Bio:          user.Bio,
	}

	if err := g.DB.AddUser(&newUser); err != nil {
		return "", "", err
	}

	g.EmbHub.QueueUserCreate(newUser.ID, newUser.Bio, []string{""}, []string{""})

	userToken, err := GenerateJWT(newUser.ID, "login", false, JWTExpiry)
	if err != nil {
		return "", "", &core.CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to generate jwt token for user."}
	}

	return userToken, token, nil
}

// ConnectGitHubCallback -> callback for the github connect endpoint
func (g *GitService) ConnectGitHubCallback(uid, code, state, storedState string) (string, error) {
	if uid == "" {
		return "", &core.CustomMessage{Code: http.StatusUnauthorized, Message: "Unauthorized user."}
	}

	if storedState != state {
		return "", &core.CustomMessage{Code: http.StatusBadRequest, Message: "Invalid or Expired state."}
	}

	data := url.Values{}
	data.Add("client_id", g.ClientID)
	data.Add("client_secret", g.ClientSecret)
	data.Add("code", code)

	req, _ := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", bytes.NewBufferString(data.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")

	resp, err := g.Client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", &core.CustomMessage{Code: http.StatusBadGateway, Message: "Failed to connect with github."}
	}

	defer resp.Body.Close()

	var gitToken schema.GitToken

	if err := json.NewDecoder(resp.Body).Decode(&gitToken); err != nil {
		return "", &core.CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to parse access token."}
	}

	userReq, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+gitToken.AccessToken)

	userResp, err := g.Client.Do(userReq)
	if err != nil || userResp.StatusCode != http.StatusOK {
		return "", &core.CustomMessage{Code: http.StatusBadGateway, Message: "Failed to signup with github."}
	}

	defer userResp.Body.Close()

	var gitUser schema.GitHubUser
	if err := json.NewDecoder(userReq.Body).Decode(&gitUser); err != nil {
		return "", &core.CustomMessage{Code: http.StatusInternalServerError, Message: "Failed to parse user info."}
	}

	var user model.User
	if err := g.DB.FetchUser(&user, uid); err != nil {
		return "", err
	}

	user.GitID = &gitUser.ID
	user.GitUser = true
	user.GitUserName = &gitUser.UserName

	if err := g.DB.SaveUser(&user); err != nil {
		return "", err
	}

	return gitToken.AccessToken, nil
}

// ViewRepo godoc
// @Summary View Github public repo for current user
// @Description An endpoint for viewing the user public repo to tag to a project.
// @Tags Git
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocViewRepos "Repo fetched"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Failure 502 {object} schema.DocNormalResponse "Bad Gateway"
// @Router /api/user/view-repo [get]
func (g *GitService) ViewRepo(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := g.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if !user.GitUser {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "You have to connect your github account."})
		return
	}

	req, _ := http.NewRequest("GET", "https://api.github.com/users/"+*user.GitUserName+"/repos?sort=pushed", nil)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := g.Client.Do(req)

	if err != nil || resp.StatusCode != http.StatusOK {
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": "Failed to communicate with github."})
		return
	}

	defer resp.Body.Close()
	var repos []schema.ViewRepo

	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse github data."})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"repos": repos})
}
