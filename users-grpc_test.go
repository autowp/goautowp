package goautowp

import (
	"context"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const TestImageFile = "./image/storage/_files/Towers_Schiphol_small.jpg"

func TestCreateUpdateDeleteUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)

	defer util.Close(conn)
	client := NewUsersClient(conn)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	userEmail := "test" + strconv.Itoa(random.Int()) + "@example.com"

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
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)
	require.NotNil(t, me)

	db, err := cnt.GoquDB()
	require.NoError(t, err)

	// set avatar
	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	imageID, err := imageStorage.AddImageFromFile(ctx, TestImageFile, "user", storage.GenerateOptions{})
	require.NoError(t, err)

	_, err = db.Update(schema.UserTable).
		Set(goqu.Record{"img": imageID}).
		Where(schema.UserTableIDCol.Eq(me.GetId())).
		Executor().ExecContext(ctx)
	require.NoError(t, err)

	user, err := client.GetUser(ctx, &APIGetUserRequest{UserId: me.GetId()})
	require.NoError(t, err)
	require.NotEmpty(t, user)
	// require.NotEmpty(t, user.Gravatar)
	require.NotEmpty(t, user.GetAvatar().GetSrc())
	require.Equal(t, name+" "+lastName, user.GetName())

	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	_, err = client.DeleteUser(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIDeleteUserRequest{UserId: me.GetId(), Password: password},
	)
	require.NoError(t, err)
}

func TestCreateUserWithEmptyLastName(t *testing.T) {
	t.Parallel()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)

	defer util.Close(conn)
	client := NewUsersClient(conn)

	userEmail := "test" + strconv.Itoa(random.Int()) + "@example.com"

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
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)
	require.NotNil(t, me)

	user, err := client.GetUser(ctx, &APIGetUserRequest{UserId: me.GetId(), Fields: &UserFields{
		Email:                 true,
		Timezone:              true,
		Language:              true,
		VotesPerDay:           true,
		VotesLeft:             true,
		Img:                   true,
		GravatarLarge:         true,
		Photo:                 true,
		IsModer:               true,
		RegDate:               true,
		PicturesAdded:         true,
		PicturesAcceptedCount: true,
		LastIp:                true,
		LastOnline:            true,
		Login:                 true,
	}})
	require.NoError(t, err)
	require.NotEmpty(t, user)
	require.Equal(t, name, user.GetName())
}

func TestSetDisabledUserCommentsNotifications(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
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
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+testerToken.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	// disable
	_, err = client.DisableUserCommentsNotifications(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIUserPreferencesRequest{UserId: tester.GetId()},
	)
	require.NoError(t, err)

	res1, err := client.GetUserPreferences(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIUserPreferencesRequest{UserId: tester.GetId()},
	)
	require.NoError(t, err)
	require.True(t, res1.GetDisableCommentsNotifications())

	// enable
	_, err = client.EnableUserCommentsNotifications(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIUserPreferencesRequest{UserId: tester.GetId()},
	)
	require.NoError(t, err)

	res2, err := client.GetUserPreferences(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIUserPreferencesRequest{UserId: tester.GetId()},
	)
	require.NoError(t, err)
	require.False(t, res2.GetDisableCommentsNotifications())
}

func TestGetOnlineUsers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
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
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+testerToken.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	res, err := client.GetUsers(ctx, &APIUsersRequest{IsOnline: true, Fields: &UserFields{
		Email:                 true,
		Timezone:              true,
		Language:              true,
		VotesPerDay:           true,
		VotesLeft:             true,
		Img:                   true,
		GravatarLarge:         true,
		Photo:                 true,
		IsModer:               true,
		RegDate:               true,
		PicturesAdded:         true,
		PicturesAcceptedCount: true,
		LastIp:                true,
		LastOnline:            true,
		Login:                 true,
	}})
	require.NoError(t, err)
	require.NotEmpty(t, res.GetItems())
}

func TestGetUsersPagination(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)

	defer util.Close(conn)

	client := NewUsersClient(conn)

	_, err = client.GetUsers(ctx, &APIUsersRequest{Page: 1, Limit: 10})
	require.NoError(t, err)
}

func TestGetUsersSearch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
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
	me, err := client.Me(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+testerToken.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	res, err := client.GetUsers(ctx, &APIUsersRequest{Search: strings.ToLower(me.GetName())})
	require.NoError(t, err)
	require.NotEmpty(t, res.GetItems())
	require.NotEmpty(t, res.GetItems()[0])

	res, err = client.GetUsers(ctx, &APIUsersRequest{Search: strings.ToUpper(me.GetName())})
	require.NoError(t, err)
	require.NotEmpty(t, res.GetItems())
	require.NotEmpty(t, res.GetItems()[0])
}
