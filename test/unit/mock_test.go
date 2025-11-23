package unit

import (
	"errors"
	"net/http"

	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

type CacheMock struct {
	Store map[string]map[string]string
	Otp   map[string]string
}

func (mock *CacheMock) CacheSkills(skills []model.Skill) {
	cache := make(map[string]string, 0)

	for _, skill := range skills {
		cache[skill.Name] = skill.ID
	}

	mock.Store["skills"] = cache
}

func (mock *CacheMock) RetrieveCachedSkills(skills []string) (map[string]string, error) {
	foundskills := make(map[string]string, 0)
	for _, val := range skills {
		if id, exists := mock.Store["skills"][val]; exists {
			foundskills[val] = id
		}
	}

	return foundskills, nil
}

func (mock *CacheMock) AddNewSkillToCache(skills []*model.Skill) {
	for _, skill := range skills {
		mock.Store["skills"][skill.Name] = skill.ID
	}
}

func (mock *CacheMock) SetOTP(otp string, uid string) error {
	mock.Otp["123456"] = uid
	return nil
}

func (mock *CacheMock) GetOTP(otp string) (string, error) {
	if val, exists := mock.Otp[otp]; exists {
		return val, nil
	}
	return "", errors.New("missing")
}

func NewCacheMock() *CacheMock {
	return &CacheMock{
		Store: make(map[string]map[string]string, 0),
		Otp:   make(map[string]string, 0),
	}
}

type EmailMock struct{}

func (mock *EmailMock) SendForgotPassEmail(_, _, _ string) error                { return nil }
func (mock *EmailMock) SendFriendReqEmail(_, _, _, _, _ string) error           { return nil }
func (mock *EmailMock) SendProjectApplicationEmail(_, _, _, _, _ string) error  { return nil }
func (mock *EmailMock) SendProjectApplicationAccept(_, _, _, _, _ string) error { return nil }
func (mock *EmailMock) SendProjectApplicationReject(_, _, _, _, _ string) error { return nil }
func NewEmailMock() *EmailMock {
	return &EmailMock{}
}

type GitMock struct{}

func (mock *GitMock) GitHubAddUser(_ *gin.Context) {}
func (mock *GitMock) ConnectGitHub(_ *gin.Context) {}
func (mock *GitMock) SelectCallback(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"token": "1234", "msg": "Logged in successfully."})
}

func (mock *GitMock) GitHubAddUserCallback(_, _, _, _ string) (string, string, error) {
	return "", "", nil
}

func (mock *GitMock) ConnectGitHubCallback(_, _, _, _ string) (string, error) {
	return "", nil
}

func (mock *GitMock) ViewRepo(ctx *gin.Context) {
	repo := schema.ViewRepo{
		Name:     "FindMe",
		HTMLURL:  "https://github.com/imisi99/FindMe",
		Language: "Go",
	}

	ctx.JSON(http.StatusOK, gin.H{"repos": repo})
}

func NewGitMock() *GitMock {
	return &GitMock{}
}
