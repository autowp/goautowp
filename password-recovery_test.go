package goautowp

import (
	"context"
	"database/sql"
	"github.com/Nerzal/gocloak/v9"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/email"
	"github.com/autowp/goautowp/users"
	"github.com/stretchr/testify/require"
	"math/rand"
	"regexp"
	"strconv"
	"testing"
	"time"
)

func TestRestorePassword(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	userEmail := "test" + strconv.Itoa(rand.Int()) + "@example.com"
	password := "password"
	newPassword := "password2"
	name := "User, who restore password"

	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	keycloak := gocloak.NewClient(cfg.Keycloak.URL)

	emailSender := email.MockSender{}

	usersRep := users.NewRepository(
		db,
		cfg.UsersSalt,
		cfg.EmailSalt,
		cfg.Languages,
		&emailSender,
		keycloak,
		cfg.Keycloak,
	)

	userID, err := usersRep.CreateUser(users.CreateUserOptions{
		Email:           userEmail,
		Password:        password,
		PasswordConfirm: password,
		Name:            name,
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

	err = usersRep.EmailChangeFinish(context.Background(), token)
	require.NoError(t, err)

	pr := NewPasswordRecovery(db, false, cfg.Languages, &emailSender)
	require.NoError(t, err)

	// request email message
	fv, err := pr.Start(userEmail, "", "127.0.0.1")
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

	err = usersRep.SetPassword(context.Background(), userID, newPassword)
	require.NoError(t, err)
}
