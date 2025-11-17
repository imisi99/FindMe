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
		"msg":     "Yo i need your help with the frontend or mobile dev.",
		"chat_id": "",
	}
	msg ViewMsg
)

func TestCreateMessage(t *testing.T) {
	payload := msgDefPayload
	payload["chat_id"] = cid
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/msg/send-message", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), payload["msg"])

	_ = json.Unmarshal(w.Body.Bytes(), &msg)
}

func TestCreateMessageInvalidChatID(t *testing.T) {
	payload := msgDefPayload
	payload["chat_id"] = "nil"
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/msg/send-message", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Chat not found.")
}

func TestViewHist(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/msg/view-hist?id="+cid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), msgDefPayload["msg"])
}

func TestViewChats(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/msg/view-chats", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), msgDefPayload["msg"])
	assert.Contains(t, w.Body.String(), cid)
}

func TestEditMessage(t *testing.T) {
	payload := map[string]string{
		"msg":    "Yo i really need your help i'm almost done with the project but i don't do mobile dev",
		"msg_id": msg.ID,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPatch, "/api/msg/edit-message", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), payload["msg"])
}

func TestDeleteMessage(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/msg/delete-message?id="+msg.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestLeaveChat(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/msg/leave-chat?id="+cid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
