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
	GitHubAddUserCallback(ctx *gin.Context)
	ConnectGitHub(ctx *gin.Context)
	ConnectGitHubCallback(ctx *gin.Context)
	ViewRepo(ctx *gin.Context)
}

type GitService struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
	DB           core.DB
	Client       *http.Client
}

func NewGitService(id, secret, callback string, db core.DB, client *http.Client) *GitService {
	return &GitService{ClientID: id, ClientSecret: secret, CallbackURL: callback, DB: db, Client: client}
}

// DONE:
// Add a check for already existing email when signing up and ask user to connect instead of assuming.
// Add an endpoint for connecting to github.
// On the signup Endpoint perform the existing userID check before the fetch for private email.
// Add a Get user repo endpoint.

// GitHubAddUser -> Signing up user using github
func (g *GitService) GitHubAddUser(ctx *gin.Context) {
	if _, err := ctx.Cookie("git-access-token"); err == nil {
		ctx.Redirect(http.StatusTemporaryRedirect, g.CallbackURL)
	}
	state, err := core.GenerateState()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate user state."})
		return
	}

	ctx.SetCookie("state", state, 150, "/", "", false, true)

	redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=read:user user:email", g.ClientID, state)
	ctx.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// GitHubAddUserCallback -> for the github signup endpoint
func (g *GitService) GitHubAddUserCallback(ctx *gin.Context) {
	var token string
	token, err := ctx.Cookie("git-access-token")
	if err != nil {
		code := ctx.Query("code")
		state := ctx.Query("state")

		if storedState, err := ctx.Cookie("state"); err != nil || state != storedState {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid or expired state"})
			return
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
			ctx.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "Failed to signup with github."})
			return
		}

		defer resp.Body.Close()

		var gitToken schema.GitToken

		if err := json.NewDecoder(resp.Body).Decode(&gitToken); err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse access token."})
			return
		}
		token = gitToken.AccessToken
	}

	ctx.SetCookie("git-access-token", token, 60*60*3, "/", "", false, true)

	userReq, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+token)

	userResp, err := g.Client.Do(userReq)
	if err != nil || userResp.StatusCode != http.StatusOK {
		ctx.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "Failed to signup with github."})
		return
	}

	defer userResp.Body.Close()

	var user schema.GitHubUser
	if err := json.NewDecoder(userResp.Body).Decode(&user); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse user info."})
		return
	}

	var existingUser model.User
	if err := g.DB.FindExistingGitID(&existingUser, user.ID); err == nil {
		userToken, err := GenerateJWT(existingUser.ID, "login", JWTExpiry)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate jwt token for user."})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"token": userToken, "message": "Logged in successfully."})
		return
	}

	if user.Email == "" {
		emailReq, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user/emails", nil)
		emailReq.Header.Set("Authorization", "Bearer "+token)
		emailReq.Header.Set("Accept", "application/vnd.github+json")
		emailResp, err := g.Client.Do(emailReq)

		if err != nil || emailResp.StatusCode != http.StatusOK {
			ctx.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "Failed to fetch user email."})
			return
		}

		defer emailResp.Body.Close()

		var email []struct {
			Email   string `json:"email"`
			Primary bool   `json:"primary"`
		}

		if err := json.NewDecoder(emailResp.Body).Decode(&email); err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse user github emails."})
			return
		}

		for _, e := range email {
			if e.Primary {
				user.Email = e.Email
				break
			}
		}

		if user.Email == "" {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Unable to signup with github no github email."})
			return
		}

	}

	if err := g.DB.CheckExistingEmail(user.Email); err == nil {
		ctx.AbortWithStatusJSON(http.StatusConflict, gin.H{"msg": "There's an account associated with that email already!"})
		return
	}

	newUsername := user.UserName
	if err := g.DB.CheckExistingUsername(newUsername); err == nil {
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
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to signup with github."})
		return
	}

	userToken, err := GenerateJWT(newUser.ID, "login", JWTExpiry)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate jwt token for user."})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"token": userToken, "message": "Logged in successfully."})
}

func (g *GitService) ConnectGitHub(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	state, err := core.GenerateState()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to generate state for github session."})
		return
	}

	ctx.SetCookie("state", state, 150, "/", "", false, true)
	ctx.SetCookie("state", uid, 150, "/", "", false, true)

	redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=read:user user:email", g.ClientID, state)
	ctx.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func (g *GitService) ConnectGitHubCallback(ctx *gin.Context) {
	uid, err := ctx.Cookie("uid")
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	code := ctx.Query("code")
	state := ctx.Query("state")

	if storedState, err := ctx.Cookie("state"); err != nil || storedState != state {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid or Expired state."})
		return
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
		ctx.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "Failed to connect with github."})
		return
	}

	defer resp.Body.Close()

	var gitToken schema.GitToken

	if err := json.NewDecoder(resp.Body).Decode(&gitToken); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse access token."})
		return
	}

	ctx.SetCookie("git-access-token", gitToken.AccessToken, 60*60*3, "/", "", false, true)

	userReq, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+gitToken.AccessToken)

	userResp, err := g.Client.Do(userReq)
	if err != nil || userResp.StatusCode != http.StatusOK {
		ctx.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "Failed to signup with github."})
		return
	}

	defer userResp.Body.Close()

	var gitUser schema.GitHubUser
	if err := json.NewDecoder(userReq.Body).Decode(&gitUser); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse user info."})
		return
	}

	var user model.User
	if err := g.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.AbortWithStatusJSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	user.GitID = &gitUser.ID
	user.GitUser = true
	user.GitUserName = &gitUser.UserName

	if err := g.DB.SaveUser(&user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.AbortWithStatusJSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "Github account connected successfully."})
}

// ViewRepo -> Endpoint for viewing the user public repo to tag to a project.
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
