package goautowp

import (
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
	"time"
)

const testUserID = 1
const adminUserID = 3

func createToken(t *testing.T, userID int64, secret string) string {
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "default",
		"exp": time.Now().Add(time.Minute * 15).Unix(),
		"sub": strconv.FormatInt(userID, 10),
	}).SignedString([]byte(secret))
	require.NoError(t, err)

	return accessToken
}
