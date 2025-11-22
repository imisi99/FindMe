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
	postPayload = schema.NewPostRequest{
		Description: "Testing the post creation endpoint.",
		Tags:        []string{"ml", "backend"},
		Git:         true,
	}
	defPostDescription = "Working on a platform for finding developers for contributive project"
	reqPayload         = map[string]string{
		"msg": "hey I'm interested in this project",
	}
	post    PostResponse
	postReq PostApplicationResponse
)

func TestGetPost(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/post/view?id="+pid, nil)
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
	assert.Contains(t, w.Body.String(), postPayload.Description)
	_ = json.Unmarshal(w.Body.Bytes(), &post)
}

func TestEditPost(t *testing.T) {
	payload := postPayload
	payload.Description = "Testing the edit post endpoint"

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPut, "/api/post/edit?id="+post.ID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), payload.Description)
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
	req.Header.Set("Authorization", "Bearer "+tokenString1)

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
	assert.Contains(t, w.Body.String(), reqPayload["msg"])

	_ = json.Unmarshal(w.Body.Bytes(), &postReq)
}

func TestViewPostApplications(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/post/view-applications", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), reqPayload["msg"])
	assert.Contains(t, w.Body.String(), superUserName)
}

func TestViewSinglePostApplications(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/post/view-application?id="+pid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), reqPayload["msg"])
	assert.Contains(t, w.Body.String(), superUserName1)
	assert.Contains(t, w.Body.String(), postReq.ReqID)
}

func TestUpdatePostApplicationInvalidStatus(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/update-application?id="+postReq.ReqID+"&status=invalidstatus", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid status.")
}

func TestUpdatePostApplicationReject(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/update-application?id="+postReq.ReqID+"&status=rejected", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Application status updated successfully.")
}

func TestCreatePostApplicationToAccept(t *testing.T) {
	body, _ := json.Marshal(defPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/post/apply?id="+pid, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	_ = json.Unmarshal(w.Body.Bytes(), &postReq)
}

func TestUpdatePostApplicationAccept(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/update-application?id="+postReq.ReqID+"&status=accepted", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Application status updated successfully.")
}

func TestCreatePostApplicationToDelete(t *testing.T) {
	body, _ := json.Marshal(defPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/post/apply?id="+post.ID, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	_ = json.Unmarshal(w.Body.Bytes(), &postReq)
}

func TestDeletePostApplication(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/post/delete-application?id="+postReq.ReqID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCreatePostApplicationToClear(t *testing.T) {
	body, _ := json.Marshal(defPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/post/apply?id="+post.ID, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	_ = json.Unmarshal(w.Body.Bytes(), &postReq)
}

func TestClearPostApplication(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/post/clear-application?id="+post.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeletePost(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/post/delete?id="+post.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
