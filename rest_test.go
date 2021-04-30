package goautowp

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func testRequest(t *testing.T, req *http.Request) *httptest.ResponseRecorder {
	config := LoadConfig()

	container := NewContainer(config)

	router, err := container.GetPublicRouter()
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func testRequestURL(t *testing.T, url string) *httptest.ResponseRecorder {
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)

	return testRequest(t, req)
}

func testRequestBody(t *testing.T, url string) []byte {

	w := testRequestURL(t, url)

	require.Equal(t, http.StatusOK, w.Code)

	bodyBytes, err := ioutil.ReadAll(w.Body)
	require.NoError(t, err)

	return bodyBytes
}

func TestGetVehicleTypesInaccessibleAnonymously(t *testing.T) {
	response := testRequestURL(t, "/api/vehicle-types")

	require.Equal(t, http.StatusForbidden, response.Code)
}

func TestGetVehicleTypesInaccessibleWithEmptyToken(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/vehicle-types", nil)
	require.NoError(t, err)

	req.Header.Add("Authorization", "")

	response := testRequest(t, req)

	require.Equal(t, http.StatusForbidden, response.Code)
}

func TestGetVehicleTypesInaccessibleWithInvalidToken(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/vehicle-types", nil)
	require.NoError(t, err)

	req.Header.Add("Authorization", "abc")

	response := testRequest(t, req)

	require.Equal(t, http.StatusForbidden, response.Code)
}

func TestGetVehicleTypesInaccessibleWithWronglySignedToken(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/vehicle-types", nil)
	require.NoError(t, err)

	req.Header.Add("Authorization", "Bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJkZWZhdWx0Iiwic3ViIjoiMSJ9.yuzUurjlDfEKchYseIrHQ1D5_RWnSuMxM-iK9FDNlQBBw8kCz3H-94xHvyd9pAA6Ry2-YkGi1v6Y3AHIpkDpcQ")

	response := testRequest(t, req)

	require.Equal(t, http.StatusForbidden, response.Code)
}

func TestGetVehicleTypesInaccessibleWithoutModeratePrivilege(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/vehicle-types", nil)
	require.NoError(t, err)

	req.Header.Add("Authorization", "Bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJkZWZhdWx0Iiwic3ViIjoiMSJ9.yuzUurjlDfEKchYseIrHQ1D5_RWnSuMxM-iK9FDNlQBBw8kCz3H-94xHvyd9pAA6Ry2-YkGi1v6Y3AHIpkDpcQ")

	response := testRequest(t, req)

	require.Equal(t, http.StatusForbidden, response.Code)
}

func TestGetVehicleTypes(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/vehicle-types", nil)
	require.NoError(t, err)

	req.Header.Add("Authorization", "Bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJkZWZhdWx0Iiwic3ViIjoiMyJ9.tI-wPZ4BSqmpsZN0-SgWXaokzvB8T-uYWLR9OQurxPFNoPC56U3op1gSE5n2H02GYfDGig0Eyp6U0NbDpsQaAg")

	response := testRequest(t, req)

	require.Equal(t, http.StatusOK, response.Code)

	bodyBytes, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)

	var result VehicleTypeResult
	err = json.Unmarshal(bodyBytes, &result)
	require.NoError(t, err)

	require.Greater(t, len(result.Items), 0)
}
