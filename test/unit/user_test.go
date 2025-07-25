package unit

import (
	"bytes"
	"encoding/json"
	"findme/database"
	"findme/handlers"
	"findme/model"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)


func getTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	handlers.SetupHandler(router)
	return router
}


func getTestDB() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		log.Println("An error occured while trying to connect to db")
	}

	db.AutoMigrate(&model.Skill{}, &model.User{})
	superUser(db)

	database.DB = db
}


func superUser(db *gorm.DB) {
	var count int64
	db.Where("username = ?", "Imisioluwa23").Count(&count)
	if count == 0 {
			super := model.User{
			FullName: "Isong Imisioluwa",
			Username: "Imisioluwa23",
			Bio: "I am the super user",
			Email: "isongrichard234@gmail.com",
			Password: ".",
			Availability: true, // for a limited time only
		}

		db.Create(&super)
	}

}


func clearDB(db *gorm.DB) {
	db.Exec("DELETE FROM user_skills")
	db.Exec("DELETE FROM skills")
	db.Exec("DELETE FROM users")
}


var (
	tokenString = ""
	defPayload  = map[string]string{
		"username": "JohnDoe23",
		"fullname": "John Doe",
		"email": "johndoe@gmail.com",
		"password": "JohnDoe234",
	}
)


func TestSignup(t *testing.T) {
	getTestDB()
	router := getTestRouter()


	payload := defPayload

	body, _ := json.Marshal(payload)


	req, _ := http.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)


	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Signed up successfully")

}


func TestSignupDuplicate(t *testing.T) {
	getTestDB()
	router := getTestRouter()


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
	getTestDB()
	router := getTestRouter()

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
	assert.Contains(t, w.Body.String(), "An error occured while tyring to parse the payload")
}


func TestLogin(t *testing.T) {
	getTestDB()
	router := getTestRouter()

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
	getTestDB()
	router := getTestRouter()

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
	assert.Contains(t, w.Body.String(), "Invalid Credentials")
}


func TestGetUserProfile(t *testing.T) {
	getTestDB()
	router := getTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "johndoe@gmail.com")
}


func TestUpdateuserProfile(t *testing.T) {
	getTestDB()
	router := getTestRouter()
	
	payload := defPayload
	payload["bio"] = "Just a chill guy building stuff"

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPut, "/user/update-profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "User profile updated successfully")
}


func TestUpdateuserProfileDuplicate(t *testing.T){
	getTestDB()
	router := getTestRouter()

	payload := map[string]string{
		"username": "Imisioluwa23",
		"fullname": "John Doe",
		"email": "JohnDoe@gmail.com",
		"password": "JohnDoe234",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPut, "/user/update-profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "Username already in use")
}

func TestUpdateAvailabilityStatus(t *testing.T) {
	getTestDB()
	router := getTestRouter()

	req, _ := http.NewRequest(http.MethodPatch, "/user/update-availability/false", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "availability status updated successfully")
}


func TestFailedUpdateAvailibilityStatus(t *testing.T) {
	getTestDB()
	router := getTestRouter()

	req, _ := http.NewRequest(http.MethodPatch, "/user/update-availability/nothing", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Availability status can only be true or false")	
}


func TestUpdateSkills(t *testing.T) {
	defer clearDB(database.DB)
	getTestDB()
	router := getTestRouter()

	payload := map[string][]string{
		"skills": {"rust", "java"},
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPatch, "/user/update-skills", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "User skills updated successfully")
}
