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

	assert.Equal(t, http.StatusCreated, w.Code)
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

func TestOpenChat(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/msg/open-chat?id="+id2, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), superUserName1)
}

func TestRenameChat(t *testing.T) {
	payload := map[string]string{
		"chat_id": gid,
		"name":    "Bankai",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPatch, "/api/msg/rename-chat", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "Chat name updated successfully.")
}

func TestAddUserToChat(t *testing.T) {
	payload := map[string]string{
		"chat_id": gid,
		"user_id": id2,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPut, "/api/msg/add-user", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestRemoveUserFromChat(t *testing.T) {
	payload := map[string]string{
		"chat_id": gid,
		"user_id": id2,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodDelete, "/api/msg/remove-user", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAddUserToChatToLeave(t *testing.T) {
	payload := map[string]string{
		"chat_id": gid,
		"user_id": id2,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPut, "/api/msg/add-user", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestLeaveChat(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/msg/leave-chat?id="+gid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString1)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteChat(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/api/msg/delete-chat?id="+gid, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
