package goautowp

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func feedbackTestRequest(t *testing.T, req *http.Request) *httptest.ResponseRecorder {
	config := LoadConfig()

	// container := NewContainer(config)

	router := gin.New()

	config.Feedback.Captcha = false

	ctrl, err := NewFeedback(config.Feedback, config.Recaptcha, config.SMTP)
	require.NoError(t, err)

	apiGroup := router.Group("/api")
	ctrl.SetupRouter(apiGroup)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func TestFeedbackNoBody(t *testing.T) {
	// empty request
	req, err := http.NewRequest(http.MethodPost, "/api/feedback", nil)
	require.NoError(t, err)

	w := feedbackTestRequest(t, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFeedbackEmptyBody(t *testing.T) {
	// empty request
	req, err := http.NewRequest(http.MethodPost, "/api/feedback", strings.NewReader(
		`{}`,
	))
	require.NoError(t, err)

	w := feedbackTestRequest(t, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFeedbackEmptyValues(t *testing.T) {
	// empty request
	req, err := http.NewRequest(http.MethodPost, "/api/feedback", strings.NewReader(
		`{"name":"","email":"","message":""}`,
	))
	require.NoError(t, err)

	w := feedbackTestRequest(t, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFeedbackEmptyName(t *testing.T) {
	// empty request
	req, err := http.NewRequest(http.MethodPost, "/api/feedback", strings.NewReader(
		`{"name":"","email":"test@example.com","message":"message"}`,
	))
	require.NoError(t, err)

	w := feedbackTestRequest(t, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFeedbackEmptyEmail(t *testing.T) {
	// empty request
	req, err := http.NewRequest(http.MethodPost, "/api/feedback", strings.NewReader(
		`{"name":"user","email":"","message":"message"}`,
	))
	require.NoError(t, err)

	w := feedbackTestRequest(t, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFeedbackEmptyMessage(t *testing.T) {
	// empty request
	req, err := http.NewRequest(http.MethodPost, "/api/feedback", strings.NewReader(
		`{"name":"user","email":"test@example.com","message":""}`,
	))
	require.NoError(t, err)

	w := feedbackTestRequest(t, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFeedbackMessage(t *testing.T) {
	// empty request
	req, err := http.NewRequest(http.MethodPost, "/api/feedback", strings.NewReader(
		`{"name":"user","email":"test@example.com","message":"message"}`,
	))
	require.NoError(t, err)

	w := feedbackTestRequest(t, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
