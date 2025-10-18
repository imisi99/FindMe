package unit

import (
	"errors"
	"net/http"

	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CacheMock struct {
	Store map[string]map[string]uint
	Otp   map[string]uint
	DB    *gorm.DB
}

func (mock *CacheMock) CacheSkills() {
	var skills []model.Skill
	mock.DB.Find(&skills)

	cache := make(map[string]uint, 0)

	for _, skill := range skills {
		cache[skill.Name] = skill.ID
	}

	mock.Store["skills"] = cache
}

func (mock *CacheMock) RetrieveCachedSkills(skills []string) (map[string]uint, error) {
	foundskills := make(map[string]uint, 0)
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

func (mock *CacheMock) SetOTP(otp string, uid uint) error {
	mock.Otp["123456"] = uid
	return nil
}

func (mock *CacheMock) GetOTP(otp string, otpInfo *schema.OTPInfo) error {
	if val, exists := mock.Otp[otp]; exists {
		otpInfo.UserID = val
		return nil
	}
	return errors.New("missing")
}

func NewCacheMock(db *gorm.DB) *CacheMock {
	return &CacheMock{
		Store: make(map[string]map[string]uint, 0),
		Otp:   make(map[string]uint, 0),
		DB:    db,
	}
}

type EmailMock struct{}

func (mock *EmailMock) SendForgotPassEmail(_, _, _ string) error             { return nil }
func (mock *EmailMock) SendFriendReqEmail(_, _, _, _, _ string) error        { return nil }
func (mock *EmailMock) SendPostApplicationEmail(_, _, _, _, _ string) error  { return nil }
func (mock *EmailMock) SendPostApplicationAccept(_, _, _, _, _ string) error { return nil }
func (mock *EmailMock) SendPostApplicationReject(_, _, _, _, _ string) error { return nil }
func NewEmailMock() *EmailMock {
	return &EmailMock{}
}

type GitMock struct{}

func (mock *GitMock) GitHubAddUser(_ *gin.Context) {}

func (mock *GitMock) GitHubAddUserCallback(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"token": "1234", "message": "Logged in successfully."})
}

func NewGitMock() *GitMock {
	return &GitMock{}
}
