package unit

import (
	"bytes"
	"encoding/json"
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
	defPostDescription = "Working on a platform for finding developers for contributive project"
	reqPayload = map[string]string {
		"msg": "hey I'm interested in this projec",
	}
)


func TestGetPost(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/post/1", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), defPostDescription)
}


func TestCreatePost(t *testing.T) {
	payload := postPayload

	body, _ := json.Marshal(payload)

	fields := payload["tags"]
	field := fields.([]string)
	mock.ExpectHMGet("skills", field...).SetVal([]any{"1", "2"})

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/post/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Post created successfully.")

}


func TestEditPost(t *testing.T) {

	payload := postPayload
	payload["description"] = "Testing the edit post endpoint"

	body, _ := json.Marshal(payload)

	fields := payload["tags"]
	field := fields.([]string)
	mock.ExpectHMGet("skills", field...).SetVal([]any{"1", "2"})
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/post/edit/2", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Post updated successfully.")
}


func TestGetPosts(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/post/posts/all", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Testing the edit post endpoint")
	assert.Contains(t, w.Body.String(), defPostDescription)
}


func TestEditPostView(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/post/edit-view/1", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "5")
}


func TestSavePostFailed(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/post/save-post?id=1", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "You can't save a post created by you.")
}


func TestSavePost(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/post/save-post?id=1", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Post saved successfully")
}


func TestViewSavedPost(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/post/view/saved-post", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), defPostDescription)
}


func TestApplyForPostFailed(t *testing.T) {
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/post/apply?id=1", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "You can't apply for a post owned by you.")
}


func TestApplyForPost(t *testing.T) {
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/post/apply?id=1", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Contains(t, w.Body.String(), "Application sent successfully.")
}


func TestViewPostApplicationsApplicant(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/post/view-applications", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), reqPayload["msg"])
	assert.Contains(t, w.Body.String(), superUserName)
}


func TestViewPostApplicationsPostOwner(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/post/view-applications", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), reqPayload["msg"])
	assert.Contains(t, w.Body.String(), superUserName1)
}


func TestUpdatePostApplicationReject(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/post/update-application?id=1&status=rejected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Application status updated successfully.")
}


func TestUpdatePostApplicationInvalidStatus(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/post/update-application?id=1&status=invalidstatus", nil) 
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid status.")
}


func TestUpdatePostApplicationAccept(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/post/update-application?id=1&status=accepted", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Application status updated successfully.")
}


func TestCreatePostApplicationToDelete(t *testing.T) {
	body, _ := json.Marshal(defPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/post/apply?id=1", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Application sent successfully.")
}


func TestDeletePostApplication(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/post/delete-application?id=2", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}


func TestDeletePost(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/post/delete/2", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
