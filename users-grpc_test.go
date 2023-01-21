package goautowp

import (
	"context"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v11"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const TestImageFile = "./image/storage/_files/Towers_Schiphol_small.jpg"

func TestCreateUpdateDeleteUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)

	defer util.Close(conn)
	client := NewUsersClient(conn)

	rand.Seed(time.Now().UnixNano())
	userEmail := "test" + strconv.Itoa(rand.Int()) + "@example.com" //nolint: gosec

	name := "ivan"
	lastName := "ivanov"
	password := "password"

	cfg := config.LoadConfig(".")
	kc := gocloak.NewClient(cfg.Keycloak.URL)

	clientToken, err := kc.LoginClient(
		ctx,
		cfg.Keycloak.ClientID,
		cfg.Keycloak.ClientSecret,
		cfg.Keycloak.Realm,
	)
	require.NoError(t, err)

	_, err = kc.CreateUser(ctx, clientToken.AccessToken, cfg.Keycloak.Realm, gocloak.User{
		Enabled:       gocloak.BoolP(true),
		EmailVerified: gocloak.BoolP(true),
		Username:      &userEmail,
		FirstName:     &name,
		LastName:      &lastName,
		Email:         &userEmail,
		Credentials: &[]gocloak.CredentialRepresentation{{
			Type:  gocloak.StringP("password"),
			Value: &password,
		}},
	})
	require.NoError(t, err)

	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, userEmail, password)
	require.NoError(t, err)
	require.NotNil(t, token)

	me, err := client.Me(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)
	require.NotNil(t, me)

	db, err := cnt.AutowpDB()
	require.NoError(t, err)

	// set avatar
	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	imageID, err := imageStorage.AddImageFromFile(ctx, TestImageFile, "user", storage.GenerateOptions{})
	require.NoError(t, err)
	_, err = db.Exec("UPDATE users SET img = ? WHERE id = ?", imageID, me.Id)
	require.NoError(t, err)

	user, err := client.GetUser(ctx, &APIGetUserRequest{UserId: me.Id, Fields: []string{"avatar", "gravatar"}})
	require.NoError(t, err)
	require.NotEmpty(t, user)
	// require.NotEmpty(t, user.Gravatar)
	require.NotEmpty(t, user.Avatar.Src)
	require.Equal(t, name+" "+lastName, user.Name)

	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	_, err = client.DeleteUser(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIDeleteUserRequest{UserId: me.Id, Password: password},
	)
	require.NoError(t, err)
}

func TestCreateUserWithEmptyLastName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)

	defer util.Close(conn)
	client := NewUsersClient(conn)

	rand.Seed(time.Now().UnixNano())
	userEmail := "test" + strconv.Itoa(rand.Int()) + "@example.com" //nolint: gosec

	name := "ivan"
	password := "password"

	cfg := config.LoadConfig(".")
	kc := gocloak.NewClient(cfg.Keycloak.URL)

	clientToken, err := kc.LoginClient(
		ctx,
		cfg.Keycloak.ClientID,
		cfg.Keycloak.ClientSecret,
		cfg.Keycloak.Realm,
	)
	require.NoError(t, err)

	_, err = kc.CreateUser(ctx, clientToken.AccessToken, cfg.Keycloak.Realm, gocloak.User{
		Enabled:       gocloak.BoolP(true),
		EmailVerified: gocloak.BoolP(true),
		Username:      &userEmail,
		FirstName:     &name,
		Email:         &userEmail,
		Credentials: &[]gocloak.CredentialRepresentation{{
			Type:  gocloak.StringP("password"),
			Value: &password,
		}},
	})
	require.NoError(t, err)

	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, userEmail, password)
	require.NoError(t, err)
	require.NotNil(t, token)

	me, err := client.Me(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)
	require.NotNil(t, me)

	user, err := client.GetUser(ctx, &APIGetUserRequest{UserId: me.Id, Fields: []string{}})
	require.NoError(t, err)
	require.NotEmpty(t, user)
	require.Equal(t, name, user.Name)
}

func TestSetDisabledUserCommentsNotifications(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)

	defer util.Close(conn)

	cfg := config.LoadConfig(".")

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	client := NewUsersClient(conn)

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	// tester
	testerToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, testUsername, testPassword)
	require.NoError(t, err)
	require.NotNil(t, testerToken)

	// tester (me)
	tester, err := client.Me(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+testerToken.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	// disable
	_, err = client.DisableUserCommentsNotifications(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIUserPreferencesRequest{UserId: tester.Id},
	)
	require.NoError(t, err)

	res1, err := client.GetUserPreferences(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIUserPreferencesRequest{UserId: tester.Id},
	)
	require.NoError(t, err)
	require.True(t, res1.DisableCommentsNotifications)

	// enable
	_, err = client.EnableUserCommentsNotifications(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIUserPreferencesRequest{UserId: tester.Id},
	)
	require.NoError(t, err)

	res2, err := client.GetUserPreferences(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIUserPreferencesRequest{UserId: tester.Id},
	)
	require.NoError(t, err)
	require.False(t, res2.DisableCommentsNotifications)
}

func TestGetOnlineUsers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)

	defer util.Close(conn)

	cfg := config.LoadConfig(".")

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	client := NewUsersClient(conn)

	// tester
	testerToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, testUsername, testPassword)
	require.NoError(t, err)
	require.NotNil(t, testerToken)

	// touch last_online for tester
	_, err = client.Me(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+testerToken.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	r, err := client.GetUsers(ctx, &APIUsersRequest{IsOnline: true})
	require.NoError(t, err)
	require.NotEmpty(t, r.Items)
}
