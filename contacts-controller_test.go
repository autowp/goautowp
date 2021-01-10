package goautowp

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestCreateDeleteContact(t *testing.T) {
	// add
	req, err := http.NewRequest(http.MethodPut, "/api/contacts/1", nil)
	require.NoError(t, err)
	req.Header.Add("Authorization", adminAuthorizationHeader)

	w := testRequest(t, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// delete
	req, err = http.NewRequest(http.MethodDelete, "/api/contacts/1", nil)
	require.NoError(t, err)
	req.Header.Add("Authorization", adminAuthorizationHeader)

	w = testRequest(t, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}
