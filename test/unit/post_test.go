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
	projectPayload = schema.NewProjectRequest{
		Description: "Testing the project creation endpoint.",
		Tags:        []string{"ml", "backend"},
		Git:         true,
	}
	defProjectDescription = "Working on a platform for finding developers for contributive project"
	reqPayload            = map[string]string{
		"msg": "hey I'm interested in this project",
	}
	project    ProjectResponse
	projectReq ProjectApplicationResponse
)

func TestGetProject(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/post/view?id="+pid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), defProjectDescription)
}

func TestCreateProject(t *testing.T) {
	payload := projectPayload

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/post/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), projectPayload.Description)
	_ = json.Unmarshal(w.Body.Bytes(), &project)
}

func TestEditProject(t *testing.T) {
	payload := projectPayload
	payload.Description = "Testing the edit project endpoint"

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPut, "/api/post/edit?id="+project.ID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), payload.Description)
}

func TestGetProjects(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/post/posts/all", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Testing the edit project endpoint")
	assert.Contains(t, w.Body.String(), defProjectDescription)
}

func TestSearchProjectTags(t *testing.T) {
	payload := map[string][]string{"tags": {"backend"}}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodGet, "/api/post/tags", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), defProjectDescription)
	assert.Contains(t, w.Body.String(), "Testing the edit project endpoint")
}

func TestEditProjectView(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/edit-view?id="+pid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "5")
}

func TestEditProjectAvailability(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/edit-status?id="+pid+"&status=false", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "false")
}

func TestSaveProjectFailed(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPut, "/api/post/save-post?id="+pid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "You can't save a project created by you.")
}

func TestSaveProject(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPut, "/api/post/save-post?id="+pid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestViewSavedProject(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/post/view/saved-post", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), defProjectDescription)
}

func TestRemoveSavedProject(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/post/remove-post?id="+pid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestApplyForProjectFailed(t *testing.T) {
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/post/apply?id="+pid, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "The owner of the project is no longer accepting applications.")
}

func TestEditProjectAvailabilityTrue(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/edit-status?id="+pid+"&status=true", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "true")
}

func TestApplyForProject(t *testing.T) {
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/post/apply?id="+pid, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Contains(t, w.Body.String(), reqPayload["msg"])

	_ = json.Unmarshal(w.Body.Bytes(), &projectReq)
}

func TestViewProjectApplications(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/post/view-applications", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), reqPayload["msg"])
	assert.Contains(t, w.Body.String(), superUserName)
}

func TestViewSingleProjectApplications(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/post/view-application?id="+pid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), reqPayload["msg"])
	assert.Contains(t, w.Body.String(), superUserName1)
	assert.Contains(t, w.Body.String(), projectReq.ReqID)
}

func TestUpdateProjectApplicationInvalidStatus(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/update-application?id="+projectReq.ReqID+"&status=invalidstatus", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid status.")
}

func TestUpdateProjectApplicationReject(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/update-application?id="+projectReq.ReqID+"&status=rejected", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Application status updated successfully.")
}

func TestCreateProjectApplicationToAccept(t *testing.T) {
	body, _ := json.Marshal(defPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/post/apply?id="+pid, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	_ = json.Unmarshal(w.Body.Bytes(), &projectReq)
}

func TestUpdateProjectApplicationAccept(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/post/update-application?id="+projectReq.ReqID+"&status=accepted", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Application status updated successfully.")
}

func TestCreateProjectApplicationToDelete(t *testing.T) {
	body, _ := json.Marshal(defPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/post/apply?id="+project.ID, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	_ = json.Unmarshal(w.Body.Bytes(), &projectReq)
}

func TestDeleteProjectApplication(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/post/delete-application?id="+projectReq.ReqID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCreateProjectApplicationToClear(t *testing.T) {
	body, _ := json.Marshal(defPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/post/apply?id="+project.ID, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	_ = json.Unmarshal(w.Body.Bytes(), &projectReq)
}

func TestClearProjectApplication(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/post/clear-application?id="+project.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteProject(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/post/delete?id="+project.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
