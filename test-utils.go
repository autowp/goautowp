package goautowp

import (
	"context"
	"github.com/Nerzal/gocloak/v9"
	"github.com/autowp/goautowp/config"
	"github.com/stretchr/testify/require"
	"testing"
)

const testUsername = "tester"
const testPassword = "123123"
const adminUsername = "admin"
const adminPassword = "123123"

func getUserToken(t *testing.T, username string, password string, keycloakClient gocloak.GoCloak, keycloakConfig config.KeycloakConfig) string {

	token, err := keycloakClient.Login(
		context.Background(),
		keycloakConfig.ClientID,
		keycloakConfig.ClientSecret,
		keycloakConfig.Realm,
		username,
		password,
	)
	require.NoError(t, err)

	return token.AccessToken
}
