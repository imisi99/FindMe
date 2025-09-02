package unit

import (
	"bytes"
	"encoding/json"
	"findme/core"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)


var (
	postPayload = map[string]any{
		"description": "Testing the post creation endpoint.",
		"tags": []string{"ml", "backend"},
	}
)

func TestCreatePost(t *testing.T) {
	getTestDB()
	mock := getTestRDB()

	mock.ExpectGet("skills").SetVal(`{}`)
	tokenString, _ = core.GenerateJWT(1, "login", core.JWTExpiry)
	router := getTestRouter()

	payload := postPayload

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/post/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Post created successfully.")

}


func TestEditPost(t *testing.T) {
	getTestDB()
	mock := getTestRDB()
	mock.ExpectGet("skills").SetVal(`{"ml": 1, "backend": 2}`)
	router := getTestRouter()

	payload := postPayload
	payload["description"] = "Testing the edit post endpoint"

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPut, "/api/v1/post/edit/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Post updated successfully.")
}


func TestGetPosts(t *testing.T) {
	getTestDB()
	router := getTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/post/posts/all", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Testing the edit post endpoint")
}


func TestEditPostView(t *testing.T) {
	getTestDB()
	router := getTestRouter()

	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/post/edit-view/1", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Contains(t, w.Body.String(), "")
}


func TestDeletePost(t *testing.T) {
	getTestDB()
	router := getTestRouter()

	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/post/delete/1", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Contains(t, w.Body.String(), "")
}


