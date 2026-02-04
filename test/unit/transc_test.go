package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MOCK tests for the transaction service

func TestGetTransactions(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/transc/view", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInitializeTransactions(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/transc/initialize", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVerifyTranscWebhook(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/api/transc/webhook", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateSubscriptionCard(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/transc/update-card", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCancelSubscription(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/transc/cancel-sub", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEnableSubscription(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPatch, "/api/transc/enable-sub", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestViewPlans(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/transc/view/plans", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
