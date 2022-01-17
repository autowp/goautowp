package goautowp

import (
	"context"
	"github.com/autowp/goautowp/email"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"math/rand"
	"regexp"
	"strconv"
	"testing"
	"time"
)

const TestImageFile = "./image/storage/_files/Towers_Schiphol_small.jpg"

func TestCreateUpdateDeleteUser(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer util.Close(conn)
	client := NewUsersClient(conn)

	rand.Seed(time.Now().UnixNano())
	userEmail := "test" + strconv.Itoa(rand.Int()) + "@example.com"

	name := "ivan"
	lastName := "ivanov"
	newName := "petr"
	newLastName := "petrov"
	password := "password"

	_, err = client.CreateUser(ctx, &APICreateUserRequest{
		Email:           userEmail,
		FirstName:       name,
		LastName:        lastName,
		Password:        password,
		PasswordConfirm: password,
		Language:        "en",
		Captcha:         "",
	})
	require.NoError(t, err)

	cnt := getContainer()

	emailSender := cnt.EmailSender().(*email.MockSender)

	re := regexp.MustCompile("https://en.localhost/account/emailcheck/([0-9a-z]+)")
	match := re.FindStringSubmatch(emailSender.Body)

	_, err = client.EmailChangeConfirm(ctx, &APIEmailChangeConfirmRequest{
		Code: match[1],
	})
	require.NoError(t, err)

	oauth, err := cnt.OAuth()
	require.NoError(t, err)

	token, userID, err := oauth.TokenByPassword(context.Background(), userEmail, password)
	require.NoError(t, err)
	require.NotNil(t, token)

	db, err := cnt.AutowpDB()
	require.NoError(t, err)

	_, err = client.UpdateUser(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token.AccessToken),
		&APIUpdateUserRequest{UserId: userID, FirstName: newName, LastName: newLastName},
	)
	require.NoError(t, err)

	// set avatar
	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	imageID, err := imageStorage.AddImageFromFile(TestImageFile, "user", storage.GenerateOptions{})
	require.NoError(t, err)
	_, err = db.Exec("UPDATE users SET img = ? WHERE id = ?", imageID, userID)
	require.NoError(t, err)

	user, err := client.GetUser(ctx, &APIGetUserRequest{UserId: userID, Fields: []string{"avatar", "gravatar"}})
	require.NoError(t, err)
	require.NotEmpty(t, user)
	// require.NotEmpty(t, user.Gravatar)
	require.NotEmpty(t, user.Avatar.Src)
	require.Equal(t, newName+" "+newLastName, user.Name)

	adminToken, _, err := oauth.TokenByPassword(context.Background(), adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	_, err = client.DeleteUser(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIDeleteUserRequest{UserId: userID, Password: password},
	)
	require.NoError(t, err)
}

func TestCreateUserWithEmptyLastName(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer util.Close(conn)
	client := NewUsersClient(conn)

	rand.Seed(time.Now().UnixNano())
	userEmail := "test" + strconv.Itoa(rand.Int()) + "@example.com"

	name := "ivan"
	password := "password"

	_, err = client.CreateUser(ctx, &APICreateUserRequest{
		Email:           userEmail,
		FirstName:       name,
		Password:        password,
		PasswordConfirm: password,
		Language:        "en",
		Captcha:         "",
	})
	require.NoError(t, err)

	cnt := getContainer()

	emailSender := cnt.EmailSender().(*email.MockSender)

	re := regexp.MustCompile("https://en.localhost/account/emailcheck/([0-9a-z]+)")
	match := re.FindStringSubmatch(emailSender.Body)

	_, err = client.EmailChangeConfirm(ctx, &APIEmailChangeConfirmRequest{
		Code: match[1],
	})
	require.NoError(t, err)

	oauth, err := cnt.OAuth()
	require.NoError(t, err)

	token, userID, err := oauth.TokenByPassword(context.Background(), userEmail, password)
	require.NoError(t, err)
	require.NotNil(t, token)

	user, err := client.GetUser(ctx, &APIGetUserRequest{UserId: userID, Fields: []string{}})
	require.NoError(t, err)
	require.NotEmpty(t, user)
	require.Equal(t, name, user.Name)
}
