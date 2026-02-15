package unit

import (
	"errors"
	"net/http"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

type CacheMock struct {
	Store map[string]map[string]string
	Otp   map[string]string
}

func (mock *CacheMock) CheckHealth() error {
	return nil
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

func (mock *CacheMock) CachePlans(plans []schema.ViewPlansResp) error {
	return nil
}

func (mock *CacheMock) RetrieveCachedPlans() ([]schema.ViewPlansResp, error) {
	return nil, nil
}

func NewCacheMock() *CacheMock {
	return &CacheMock{
		Store: make(map[string]map[string]string, 0),
		Otp:   make(map[string]string, 0),
	}
}

type EmailHub struct{}

func (mock *EmailHub) QueueSubscriptionCreate(_, _, _, _, _, _ string)     {}
func (mock *EmailHub) QueueProjectApplicationReject(_, _, _, _, _ string)  {}
func (mock *EmailHub) QueueProjectApplicationAccept(_, _, _, _, _ string)  {}
func (mock *EmailHub) QueueTransactionFailedEmail(_, _, _, _, _, _ string) {}
func (mock *EmailHub) QueueProjectApplication(_, _, _, _, _ string)        {}
func (mock *EmailHub) QueueFriendReqEmail(_, _, _, _, _ string)            {}
func (mock *EmailHub) QueueForgotPassEmail(_, _, _ string)                 {}
func (mock *EmailHub) QueueSubscriptionReEnabled(_, _, _ string)           {}
func (mock *EmailHub) QueueSubscriptionCancelled(_, _, _ string)           {}
func (mock *EmailHub) QueueNotifyFreeTrialEnding(_, _, _, _ string)        {}
func (mock *EmailHub) Worker()                                             {}

func NewEmailHubMock() *EmailHub {
	return &EmailHub{}
}

type EmailMock struct{}

func (mock *EmailMock) SendForgotPassEmail(_, _ string) (string, string)               { return "", "" }
func (mock *EmailMock) SendFriendReqEmail(_, _, _, _ string) (string, string)          { return "", "" }
func (mock *EmailMock) SendProjectApplicationEmail(_, _, _, _ string) (string, string) { return "", "" }
func (mock *EmailMock) SendProjectApplicationAccept(_, _, _, _ string) (string, string) {
	return "", ""
}

func (mock *EmailMock) SendProjectApplicationReject(_, _, _, _ string) (string, string) {
	return "", ""
}

func (mock *EmailMock) SendSubscriptionCreateEmail(_, _, _, _, _ string) (string, string) {
	return "", ""
}

func (mock *EmailMock) SendTransactionFailedEmail(_, _, _, _, _ string) (string, string) {
	return "", ""
}
func (mock *EmailMock) SendSubscriptionCancelledEmail(_, _ string) (string, string) { return "", "" }
func (mock *EmailMock) SendSubscriptionReEnabledEmail(_, _ string) (string, string) { return "", "" }

func (mock *EmailMock) SendNotifyFreeTrialEnding(_, _, _ string) (string, string) { return "", "" }

func (mock *EmailMock) SendEmail(_, _, _ string) error { return nil }

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

type TranscMock struct{}

func (mock *TranscMock) InitializeTransaction(_ *gin.Context)    {}
func (mock *TranscMock) GetTransactions(_ *gin.Context)          {}
func (mock *TranscMock) UpdateSubscriptionCard(ctx *gin.Context) {}
func (mock *TranscMock) CancelSubscription(ctx *gin.Context)     {}
func (mock *TranscMock) EnableSubscription(ctx *gin.Context)     {}
func (mock *TranscMock) ViewPlans(ctx *gin.Context)              {}
func (mock *TranscMock) VerifyTranscWebhook(ctx *gin.Context)    {}
func (mock *TranscMock) RetryFailedPayment(ctx *gin.Context)     {}

func NewTranscMock() *TranscMock {
	return &TranscMock{}
}

type EmbeddingMock struct{}

func (e *EmbeddingMock) QueueUserCreate(id, bio string, skills, interests []string)             {}
func (e *EmbeddingMock) QueueUserUpdate(id, bio string, skills, interest []string)              {}
func (e *EmbeddingMock) QueueUserUpdateStatus(id string, status bool)                           {}
func (e *EmbeddingMock) QueueUserDelete(id string)                                              {}
func (e *EmbeddingMock) QueueProjectCreate(id, title, description, uid string, skills []string) {}
func (e *EmbeddingMock) QueueProjectUpdate(id, title, description string, skills []string)      {}
func (e *EmbeddingMock) QueueProjectUpdateStatus(id string, status bool)                        {}
func (e *EmbeddingMock) QueueProjectDelete(id string)                                           {}

func NewEmbeddingMock() *EmbeddingMock {
	return &EmbeddingMock{}
}

type RecommendationMock struct{}

func (r *RecommendationMock) QueueUserRecommendation(_ string)    {}
func (r *RecommendationMock) QueueProjectRecommendation(_ string) {}
func (r *RecommendationMock) GetRecommendation(ID string, jobType core.RecommendationJobType) (*schema.RecResponse, error) {
	return &schema.RecResponse{}, nil
}

func NewRecommendationMock() *RecommendationMock {
	return &RecommendationMock{}
}

type CronMock struct{}

func NewCronMock() *CronMock {
	return &CronMock{}
}

func (mock *CronMock) TrialEndingReminders() error { return nil }
