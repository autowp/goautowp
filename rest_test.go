package goautowp

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRequest(t *testing.T, url string) []byte {
	config := LoadConfig()

	wg := &sync.WaitGroup{}
	s, err := NewService(wg, config)
	require.NoError(t, err)
	defer func() {
		s.Close()
		wg.Wait()
	}()
	router := s.GetRouter()

	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	bodyBytes, err := ioutil.ReadAll(w.Body)
	require.NoError(t, err)

	return bodyBytes
}

func TestGetSpecs(t *testing.T) {
	bodyBytes := testRequest(t, "/go-api/spec")

	var response specResult
	err := json.Unmarshal(bodyBytes, &response)
	require.NoError(t, err)

	assert.True(t, len(response.Items) > 0)
}

func TestGetPerspectives(t *testing.T) {
	bodyBytes := testRequest(t, "/go-api/perspective")

	var response perspectiveResult
	err := json.Unmarshal(bodyBytes, &response)
	require.NoError(t, err)

	assert.True(t, len(response.Items) > 0)
}

func TestGetBrandIcons(t *testing.T) {
	bodyBytes := testRequest(t, "/api/brands/icons")

	var response BrandsIconsResult
	err := json.Unmarshal(bodyBytes, &response)
	require.NoError(t, err)

	assert.Contains(t, response.Image, "png")
	assert.Contains(t, response.Css, "css")
}

func TestGetVehicleTypes(t *testing.T) {
	bodyBytes := testRequest(t, "/api/vehicle-type")

	var response VehicleTypeResult
	err := json.Unmarshal(bodyBytes, &response)
	require.NoError(t, err)

	assert.True(t, len(response.Items) > 0)
}
