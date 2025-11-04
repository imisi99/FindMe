package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"findme/handlers"

	"github.com/stretchr/testify/assert"
)

var (
	tokenString    = ""
	tokenString1   = ""
	resetToken     = ""
	superUserName  = "Imisioluwa23"
	superUserName1 = "knightmares23"
	userToken      = ""
	defPayload     = map[string]string{
		"username":    "JohnDoe23",
		"fullname":    "John Doe",
		"email":       "johndoe@gmail.com",
		"password":    "JohnDoe234",
		"gitusername": "johndoe23",
	}
	token     *Token
	friendreq *ViewFriendReq
)

func TestSignup(t *testing.T) {
	payload := defPayload

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestSignupDuplicate(t *testing.T) {
	payload := defPayload

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// Git Mock Test
func TestGitSignup(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/github/callback", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogin(t *testing.T) {
	payload := map[string]string{
		"username": "JohnDoe23",
		"password": "JohnDoe234",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	_ = json.Unmarshal(w.Body.Bytes(), token)
	tokenString = token.Token
}

func TestLoginInvalid(t *testing.T) {
	payload := map[string]string{
		"username": "JohnDoe",
		"password": "JohnDoe234",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid Credentials!")
}

func TestGetUserProfile(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "johndoe@gmail.com")
}

func TestViewGitUser(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/user/view-git?id=imisi99", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), superUserName)
	assert.Contains(t, w.Body.String(), defPostDescription)
}

func TestSearchUserbySkills(t *testing.T) {
	skills := map[string][]string{
		"skills": {"go", "backend"},
	}
	body, _ := json.Marshal(skills)

	req, _ := http.NewRequest(http.MethodGet, "/api/user/search", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), superUserName)
	assert.Contains(t, w.Body.String(), superUserName1)
}

func TestViewUser(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/user/view?id="+superUserName, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "isongrichard234@gmail.com")
	assert.Contains(t, w.Body.String(), defPostDescription)
}

func TestSendFriendReq(t *testing.T) {
	payload := map[string]string{
		"username": superUserName,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/user/send-user-req", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), superUserName)
	_ = json.Unmarshal(w.Body.Bytes(), friendreq)
}

func TestSendDuplicateFriendReq(t *testing.T) {
	payload := map[string]string{
		"username": defPayload["username"],
	}
	body, _ := json.Marshal(payload)
	userToken, _ = handlers.GenerateJWT(id1, "login", 5*time.Minute) // Super created user from the test above to test the accepting of friend request sent
	req, _ := http.NewRequest(http.MethodPost, "/api/user/send-user-req", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+userToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "This user has already sent you a request.")
}

func TestViewFriendReq(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/user/view-user-req", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), defPayload["username"])
}

func TestUpdateFriendReqReject(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/user/update-user-req?id="+friendreq.ID+"&status=rejected", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Status updated successfully.")
}

func TestUpdateFriendReqInvalidStatus(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/user/update-user-req?id="+friendreq.ID+"&status=invalid", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid status")
}

func TestUpdateFriendReqAccept(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/user/update-user-req?id="+friendreq.ID+"&status=accepted", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Status updated successfully.")
}

func TestViewUserFriends(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/user/view-user-friend", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), superUserName)
}

func TestDeleteFriend(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/user/delete-user-friend?id="+superUserName, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAddReqToTestDelete(t *testing.T) {
	payload := map[string]string{
		"username": superUserName,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/user/send-user-req", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), superUserName)
	_ = json.Unmarshal(w.Body.Bytes(), friendreq)
}

func TestDeleteReq(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/user/delete-friend-req?id="+friendreq.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestForgotPassword(t *testing.T) {
	payload := map[string]string{
		"email": "johndoe@gmail.com",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodGet, "/forgot-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Email sent successfully.")
}

func TestVerifyOPT(t *testing.T) {
	payload := map[string]string{
		"otp": "123456", // Using the default otp 123456 that is set in the mock
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodGet, "/verify-otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "otp verified")

	_ = json.Unmarshal(w.Body.Bytes(), token)
	resetToken = token.Token
}

func TestResetPassword(t *testing.T) {
	payload := map[string]string{
		"password": "johndoe66.",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPatch, "/api/user/reset-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+resetToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "password reset successfully")
}

func TestUpdateuserProfile(t *testing.T) {
	payload := defPayload
	payload["bio"] = "Just a chill guy building stuff"

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPut, "/api/user/update-profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), payload["bio"])
}

func TestUpdateuserProfileDuplicate(t *testing.T) {
	payload := map[string]string{
		"username": "Imisioluwa23",
		"fullname": "John Doe",
		"email":    "JohnDoe@gmail.com",
		"password": "JohnDoe234",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPut, "/api/user/update-profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "Username already in use!")
}

func TestUpdateuserPassword(t *testing.T) {
	payload := map[string]string{
		"password":     "johndoe66.",
		"new_password": "Johndoe12.",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPatch, "/api/user/update-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Password updated successfully.")
}

func TestUpdateuserPasswordFail(t *testing.T) {
	payload := map[string]string{
		"password":     "wrongPassword",
		"new_password": "Johndoe12.",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPatch, "/api/user/update-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Unauthorized user.")
}

func TestUpdateAvailabilityStatus(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/user/update-availability/false", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Availability updated successfully.")
}

func TestFailedUpdateAvailibilityStatus(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/user/update-availability/nothing", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Availability status can only be true or false.")
}

func TestUpdateSkills(t *testing.T) {
	payload := map[string][]string{
		"skills": {"rust", "java"},
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPatch, "/api/user/update-skills", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "rust")
	assert.Contains(t, w.Body.String(), "java")
}

func TestDeleteSkills(t *testing.T) {
	payload := map[string][]string{
		"skills": {"rust"},
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodDelete, "/api/user/delete-skills", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteUser(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/user/delete-user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
