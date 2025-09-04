package unit

import (
	"bytes"
	"encoding/json"
	"findme/core"
	"findme/database"
	"findme/handlers"
	"findme/model"
	"findme/schema"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)


var router *gin.Engine
var mock redismock.ClientMock


func getTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	handlers.SetupHandler(router)
	return router
}


func getTestDB() *gorm.DB{
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		log.Println("An error occured while trying to connect to db")
	}

	db.AutoMigrate(&model.Skill{}, &model.User{}, &model.Post{}, &model.PostSkill{}, &model.UserSkill{})
	db.SetupJoinTable(&model.Post{}, "Tags", &model.PostSkill{})
	db.SetupJoinTable(&model.User{}, "Skills", &model.UserSkill{})
	superUser(db)

	database.SetDB(db)
	return db
}


func getTestRDB() redismock.ClientMock{
	rdb, mock := redismock.NewClientMock()

	database.SetRDB(rdb)

	return mock
}


func superUser(db *gorm.DB) {
	gitusername := "imisi99"

	hashpass, _ := core.HashPassword("Password")
	super := model.User{
		FullName: "Isong Imisioluwa",
		UserName: "Imisioluwa23",
		GitUserName: &gitusername,
		Bio: "I am the super user",
		Email: "isongrichard234@gmail.com",
		Password: hashpass,
		Availability: true,
	}

	db.Create(&super)
	skill := model.Skill{
		Name: "frontend-dev",
	}
	db.Create(&skill)
	post := model.Post{
		Description: "Working on a platform for finding developers for contributive project",
		UserID: super.ID,
		Views: 4,
		Tags: []*model.Skill{&skill},
	}

	db.Create(&post)
}


func clearDB(db *gorm.DB) {
	db.Exec("DELETE FROM user_skills")
	db.Exec("DELETE FROM post_skills")
	db.Exec("DELETE FROM skills")
	db.Exec("DELETE FROM posts")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM sqlite_sequence")
}


var (
	tokenString = ""
	resetToken = ""
	defPayload  = map[string]string{
		"username": "JohnDoe23",
		"fullname": "John Doe",
		"email": "johndoe@gmail.com",
		"password": "JohnDoe234",
		"gitusername": "johndoe23",
	}
)


func TestSignup(t *testing.T) {
	mock.ExpectGet("skills").SetVal(`{"frontend-dev": 1, "ml": 2, "backend": 3}`)

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


func TestSignupInvalidPayload(t *testing.T) {
	payload := map[string]string{
		"username": "",
		"fullname": "John Doe",
		"email": "JohnDoe",
		"password": "JohnDoe234",
	}

	body, _ := json.Marshal(payload)


	req, _ := http.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)


	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to parse the payload.")
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


func TestViewUser(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/user/view/Imisioluwa23", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "isongrichard234@gmail.com")
}


func TestForgotPassword(t *testing.T) {
	otp := schema.OTPInfo{UserID: 2}
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
	mock.ExpectGet("123456").SetVal(`{"UserID": 2}`)
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
		"password": "Johndoe12.",
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
	mock.ExpectGet("skills").SetVal(`{"frontend-dev": 1, "ml": 2, "backend": 3}`)

	payload := map[string][]string{
		"skills": {"rust", "java"},
	}

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
	mock.ExpectGet("skills").SetVal(`{"frontend-dev": 1, "ml": 2, "backend": 3, "rust": 4, "java": 5}`)
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
	assert.Contains(t, w.Body.String(), "")
}


func TestDeleteUser(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/user/delete-user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Contains(t, w.Body.String(), "")
}


func TestMain(m *testing.M) {

	database.SetDB(getTestDB())
	router = getTestRouter()
	mock = getTestRDB()

	os.Setenv("Testing", "True") 			// Using this for skipping the sending of email for the the forget password test    not proper

	code := m.Run()

	clearDB(database.GetDB())
	os.Exit(code)
}
