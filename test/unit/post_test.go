package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"findme/schema"

	"github.com/stretchr/testify/assert"
)

var (
	postPayload = map[string]any{
		"description": "Testing the post creation endpoint.",
		"tags":        []string{"ml", "backend"},
	}
	defPostDescription = "Working on a platform for finding developers for contributive project"
	reqPayload         = map[string]string{
		"msg": "hey I'm interested in this projec",
	}
	post *schema.PostResponse
)

func TestGetPost(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/post?id="+pid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), defPostDescription)
}

func TestCreatePost(t *testing.T) {
	payload := postPayload

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/post/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Post created successfully.")
	_ = json.Unmarshal(w.Body.Bytes(), post)
}

func TestEditPost(t *testing.T) {
	payload := postPayload
	payload["description"] = "Testing the edit post endpoint"

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPut, "/api/post/edit?id="+post.ID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Post updated successfully.")
}

func TestGetPosts(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/post/posts/all", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Testing the edit post endpoint")
	assert.Contains(t, w.Body.String(), defPostDescription)
}

func TestSearchPostTags(t *testing.T) {
	payload := map[string][]string{"tags": {"backend"}}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodGet, "/api/post/tags", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), defPostDescription)
	assert.Contains(t, w.Body.String(), "Testing the edit post endpoint")
}

func TestEditPostView(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/edit-view?id="+pid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "5")
}

func TestEditPostAvailability(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/edit-status?id="+pid+"&status=false", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "false")
}

func TestSavePostFailed(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPut, "/api/post/save-post?id="+pid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "You can't save a post created by you.")
}

func TestSavePost(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPut, "/api/post/save-post?id="+pid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Post saved successfully")
}

func TestViewSavedPost(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/post/view/saved-post", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), defPostDescription)
}

func TestRemoveSavedPost(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/post/remove-post?id="+pid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestApplyForPostFailed(t *testing.T) {
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/post/apply?id="+pid, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "The owner of the post is no longer accepting applications.")
}

func TestEditPostAvailabilityTrue(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/edit-status?id="+pid+"&status=true", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "true")
}

func TestApplyForPost(t *testing.T) {
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/post/apply?id="+pid, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Contains(t, w.Body.String(), "Application sent successfully.")
}

func TestViewPostApplicationsApplicant(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/post/view-applications", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), reqPayload["msg"])
	assert.Contains(t, w.Body.String(), superUserName)
}

func TestViewPostApplicationsPostOwner(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/post/view-applications", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), reqPayload["msg"])
	assert.Contains(t, w.Body.String(), superUserName1)
}

func TestUpdatePostApplicationReject(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/update-application?id=1&status=rejected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Application status updated successfully.")
}

func TestUpdatePostApplicationInvalidStatus(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/update-application?id=1&status=invalidstatus", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid status.")
}

func TestUpdatePostApplicationAccept(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/update-application?id=1&status=accepted", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Application status updated successfully.")
}

func TestCreatePostApplicationToDelete(t *testing.T) {
	body, _ := json.Marshal(defPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/post/apply?id=1", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Application sent successfully.")
}

func TestDeletePostApplication(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/post/delete-application?id=2", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeletePost(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/post/delete/2", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
