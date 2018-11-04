package goautowp

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSpecs(t *testing.T) {
	config := LoadConfig()

	s, err := NewService(config)
	require.NoError(t, err)
	defer s.Close()
	router := s.GetRouter()

	req, err := http.NewRequest("GET", "/go-api/spec", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	bodyBytes, err := ioutil.ReadAll(w.Body)
	require.NoError(t, err)

	var response specResult
	err = json.Unmarshal(bodyBytes, &response)
	require.NoError(t, err)

	assert.True(t, len(response.Items) > 0)
}

func TestGetPerspectives(t *testing.T) {
	config := LoadConfig()

	s, err := NewService(config)
	require.NoError(t, err)
	defer s.Close()
	router := s.GetRouter()

	req, err := http.NewRequest("GET", "/go-api/perspective", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	bodyBytes, err := ioutil.ReadAll(w.Body)
	require.NoError(t, err)

	var response perspectiveResult
	err = json.Unmarshal(bodyBytes, &response)
	require.NoError(t, err)

	assert.True(t, len(response.Items) > 0)
}
