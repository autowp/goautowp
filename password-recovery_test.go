package goautowp

import (
	"database/sql"
	"github.com/Nerzal/gocloak/v8"
	"github.com/stretchr/testify/require"
	"math/rand"
	"regexp"
	"strconv"
	"testing"
	"time"
)

func TestRestorePassword(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	email := "test" + strconv.Itoa(rand.Int()) + "@example.com"
	password := "password"
	newPassword := "password2"
	name := "User, who restore password"

	config := LoadConfig()

	db, err := sql.Open("mysql", config.AutowpDSN)
	require.NoError(t, err)

	keycloak := gocloak.NewClient(config.KeyCloak.URL)

	emailSender := MockEmailSender{}

	users, err := NewUserRepository(
		db,
		config.UsersSalt,
		config.EmailSalt,
		config.Languages,
		&emailSender,
		keycloak,
		config.KeyCloak,
	)
	require.NoError(t, err)

	userID, err := users.CreateUser(CreateUserOptions{
		Email:           email,
		Password:        password,
		PasswordConfirm: password,
		FirstName:       name,
		Timezone:        "UTC",
		Language:        "en",
	})
	require.NoError(t, err)
	require.NotZero(t, userID)

	// parse message for url with token
	re := regexp.MustCompile(`https?://en\.localhost/account/emailcheck/([0-9a-f]+)`)
	matches := re.FindStringSubmatch(emailSender.Body)
	require.NotEmpty(t, matches)
	token := matches[1]

	err = users.EmailChangeFinish(token)
	require.NoError(t, err)

	pr := NewPasswordRecovery(db, false, config.Languages, &emailSender)
	require.NoError(t, err)

	// request email message
	fv, err := pr.Start(email, "", "127.0.0.1")
	require.NoError(t, err)
	require.Nil(t, fv)

	// parse message for url with token
	re = regexp.MustCompile(`https?://en\.localhost/restore-password/new\?code=([0-9a-fA-F]+)`)
	matches = re.FindStringSubmatch(emailSender.Body)
	require.NotEmpty(t, matches)
	token = matches[1]

	// check token availability
	userID, err = pr.GetUserID(token)
	require.NoError(t, err)
	require.NotZero(t, userID)

	// change password with token
	fv, userID, err = pr.Finish(token, newPassword, newPassword)
	require.NoError(t, err)
	require.NotZero(t, userID)
	require.Nil(t, fv)

	err = users.SetPassword(userID, newPassword)
	require.NoError(t, err)
}
