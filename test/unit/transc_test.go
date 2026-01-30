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
	req, _ := http.NewRequest(http.MethodPost, "/api/transc/initialize", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
