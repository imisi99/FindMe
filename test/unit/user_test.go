package unit

import (
	"bytes"
	"encoding/json"
	"findme/core"
	"findme/schema"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)


var (
	tokenString = ""
	tokenString1 = ""
	resetToken = ""
	superUserName = "Imisioluwa23"
	superUserName1 = "knightmares23"
	userToken = ""
	defPayload  = map[string]string{
		"username": "JohnDoe23",
		"fullname": "John Doe",
		"email": "johndoe@gmail.com",
		"password": "JohnDoe234",
		"gitusername": "johndoe23",
	}
)


func TestSignup(t *testing.T) {

	payload := defPayload

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)


	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Signed up successfully.")

}


func TestSignupDuplicate(t *testing.T) {
	payload := defPayload

	body, _ := json.Marshal(payload)


	req, _ := http.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)


	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already in use!")
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
	assert.Contains(t, w.Body.String(), "Logged in successfully")
	token := w.Body.String()
	tokenparts := strings.Split(token, "token")
	tokenString = tokenparts[1]
	tokenString = tokenString[3:len(tokenString)-2]
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
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "johndoe@gmail.com")
}


func TestViewGitUser(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/user/view-git?id=imisi99", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), superUserName)
}


func TestSearchUserbySkills(t *testing.T) {
	skills := map[string][]string{
		"skills": {"go", "backend"},
	}
	body, _ := json.Marshal(skills)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/user/search", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), superUserName)
	assert.Contains(t, w.Body.String(), superUserName1)
}


func TestViewUser(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/user/view?id="+superUserName, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "isongrichard234@gmail.com")
}


func TestSendFriendReq(t *testing.T) {
	payload := map[string]string{
		"username": superUserName,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/user/send-user-req", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Friend request sent successfully.")
}


func TestSendDuplicateFriendReq(t *testing.T) {
	payload := map[string]string{
		"username": defPayload["username"],
	}
	body, _ := json.Marshal(payload)
	userToken, _ = core.GenerateJWT(1, "login", 5 * time.Minute)             							// Super created user from the test above to test the accepting of friend request sent 
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/user/send-user-req", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+userToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "This user has already sent you a request.")
}


func TestViewFriendReq(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/user/view-user-req", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), defPayload["username"])
	assert.Contains(t, w.Body.String(), "pending")
}


func TestUpdateFriendReqReject(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/user/update-user-req?id=1&status=rejected", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Status updated successfully")
}


func TestUpdateFriendReqInvalidStatus(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/user/update-user-req?id=1&status=invalid", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid status")
}


func TestUpdateFriendReqAccept(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/user/update-user-req?id=1&status=accepted", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Status updated successfully")
}


func TestViewUserFriends(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/user/view-user-friend", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), superUserName)
}


func TestDeleteFriend(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/user/delete-user-friend?id="+superUserName, nil)
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

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/user/send-user-req", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Friend request sent successfully.")
}


func TestDeleteReq(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/user/delete-friend-req?id=2", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}


func TestForgotPassword(t *testing.T) {
	otp := schema.OTPInfo{UserID: 3}
	data, _ := json.Marshal(otp)

	mock.Regexp().ExpectGet("[0-9]{6}").SetErr(redis.Nil)
	mock.Regexp().ExpectSet("[0-9]{6}", data, 10*time.Minute).SetVal(`OK`)

	payload := map[string]string{
		"email": "johndoe@gmail.com",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodGet, "/forgot-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Email sent successfully")
}


func TestVerifyOPT(t *testing.T) {
	mock.ExpectGet("123456").SetVal(`{"UserID": 3}`)
	payload := map[string]string{
		"otp": "123456",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodGet, "/verify-otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "otp verified")
	token := w.Body.String()
	tokenparts := strings.Split(token, "token")
	resetToken = tokenparts[1]
	resetToken = resetToken[3:len(resetToken)-2]
}


func TestResetPassword(t *testing.T) {
	payload := map[string]string{
		"password": "johndoe66.",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/user/reset-password", bytes.NewBuffer(body))
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

	req, _ := http.NewRequest(http.MethodPut, "/api/v1/user/update-profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "User profile updated successfully.")
}


func TestUpdateuserProfileDuplicate(t *testing.T){
	payload := map[string]string{
		"username": "Imisioluwa23",
		"fullname": "John Doe",
		"email": "JohnDoe@gmail.com",
		"password": "JohnDoe234",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPut, "/api/v1/user/update-profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "Username already in use!")
}


func TestUpdateuserPassword(t *testing.T) {
	payload := map[string]string{
		"password": "johndoe66.",
		"new_password": "Johndoe12.",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/user/update-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "password updated successfully")
}


func TestUpdateuserPasswordFail(t *testing.T) {
	payload := map[string]string{
		"password": "wrongPassword",
		"new_password": "Johndoe12.",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/user/update-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Unauthorized user.")
}


func TestUpdateAvailabilityStatus(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/user/update-availability/false", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "User availability updated successfully.")
}


func TestFailedUpdateAvailibilityStatus(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/user/update-availability/nothing", nil)
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

	value := make(map[string]string, 0)
	for i := range payload["skills"] {
		value[payload["skills"][i]] = fmt.Sprintf("%d", i+3)
	}
	mock.ExpectHMGet("skills", payload["skills"]...).SetVal([]any{nil, nil})
	mock.ExpectHSet("skills", value).SetVal(int64(len(value)))

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/user/update-skills", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "User skills updated successfully.")
}


func TestDeleteSkills(t *testing.T) {
	payload := map[string][]string{
		"skills": {"rust"},
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/user/delete-skills", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}


func TestDeleteUser(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/user/delete-user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
