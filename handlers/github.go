package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Git interface {
	GitHubAddUser(ctx *gin.Context)
	GitHubAddUserCallback(ctx *gin.Context)
}

type GitService struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
	DB           *gorm.DB
	Client       *http.Client
}

func NewGitService(id, secret, callback string, db *gorm.DB, client *http.Client) *GitService {
	return &GitService{ClientID: id, ClientSecret: secret, CallbackURL: callback, DB: db, Client: client}
}

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

		body, _ := io.ReadAll(resp.Body)

		var gitToken struct {
			AccessToken string `json:"access_token"`
		}

		if err := json.Unmarshal(body, &gitToken); err != nil {
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

	userBody, _ := io.ReadAll(userResp.Body)
	var user schema.GitHubUser
	if err := json.Unmarshal(userBody, &user); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse user info."})
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

		emailBody, _ := io.ReadAll(emailResp.Body)
		var email []struct {
			Email   string `json:"email"`
			Primary bool   `json:"primary"`
		}

		if err := json.Unmarshal(emailBody, &email); err != nil {
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
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Unable to signup with github."})
			return
		}

	}

	var existingUser model.User
	if err := g.DB.Where("gitid = ?", user.ID).First(&existingUser).Error; err == nil {
		userToken, err := GenerateJWT(existingUser.ID, "login", JWTExpiry)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate jwt token for user."})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"token": userToken, "message": "Logged in successfully."})
		return
	}

	if err := g.DB.Where("email = ?", user.Email).First(&existingUser).Error; err == nil {
		if !existingUser.GitUser {
			existingUser.GitID = &user.ID
			existingUser.GitUserName = &user.UserName
			existingUser.GitUser = true

			if err := g.DB.Save(&existingUser).Error; err != nil {
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to log in user."})
				return
			}

			userToken, err := GenerateJWT(existingUser.ID, "login", JWTExpiry)
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate jwt token for user."})
				return
			}

			ctx.JSON(http.StatusOK, gin.H{"token": userToken, "message": "Logged in successfully."})
			return
		} else {
			ctx.AbortWithStatusJSON(http.StatusConflict, gin.H{"message": "A github accouont associated with your email already in use."})
			return
		}
	}

	newUsername := user.UserName
	if err := g.DB.Where("username = ?", user.UserName).First(&existingUser).Error; err == nil {
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

	if err := g.DB.Create(&newUser).Error; err != nil {
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
