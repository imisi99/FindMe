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
	msgDefPayload = map[string]string{
		"msg": "Yo i need your help with the frontend or mobile dev.",
		"user": superUserName1,
	}
)

func TestCreateMessage(t *testing.T) {
	payload := msgDefPayload
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/user/send-message", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "message sent successfully")
}


func TestCreateMessageNotFriends(t *testing.T) {
	payload := msgDefPayload
	payload["user"] = defPayload["username"]
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/user/send-message", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "user is not your friend.")
}


func TestViewMessages(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/user/view-message?id="+superUserName1, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), msgDefPayload["msg"])
}


func TestEditMessage(t *testing.T) {
	payload := map[string]string{
		"msg": "Yo i really need your help i'm almost done with the project but i don't do mobile dev",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPatch, "/api/user/edit-message?id=1", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Message Edited successfully.")
}


func TestDeleteMessage(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/user/delete-message?id=1", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
