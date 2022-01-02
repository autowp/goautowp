package goautowp

import (
	"context"
	"database/sql"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"math/rand"
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

	name := "test"
	newName := "test 2"
	password := "password"

	_, err = client.CreateUser(ctx, &APICreateUserRequest{
		Email:           userEmail,
		Name:            name,
		Password:        password,
		PasswordConfirm: password,
		Language:        "en",
		Captcha:         "",
	})
	require.NoError(t, err)

	cfg := config.LoadConfig(".")

	var userID int64

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)
	defer util.Close(db)
	err = db.QueryRow("SELECT id FROM users WHERE email_to_check = ?", userEmail).Scan(&userID)
	require.NoError(t, err)

	_, err = client.UpdateUser(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+createToken(t, userID, cfg.Auth.OAuth.Secret)),
		&APIUpdateUserRequest{UserId: userID, Name: newName},
	)
	require.NoError(t, err)

	// set avatar
	imageStorage, err := storage.NewStorage(db, cfg.ImageStorage)
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
	require.Equal(t, newName, user.Name)

	_, err = client.DeleteUser(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+createToken(t, adminUserID, cfg.Auth.OAuth.Secret)),
		&APIDeleteUserRequest{UserId: userID, Password: password},
	)
	require.NoError(t, err)
}
