package goautowp

import (
	"context"
	"database/sql"
	"github.com/Nerzal/gocloak/v8"
	"github.com/autowp/goautowp/util"
	"github.com/casbin/casbin"
	"github.com/dgrijalva/jwt-go/v4"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"log"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"
)

const bufSize = 1024 * 1024

const adminUserID = 3

var lis *bufconn.Listener

func init() {

	config := LoadConfig()

	db, err := sql.Open("mysql", config.AutowpDSN)
	if err != nil {
		panic(err)
	}

	enforcer := casbin.NewEnforcer("model.conf", "policy.csv")

	emailSender := &MockEmailSender{}

	contactsRepository := NewContactsRepository(db)
	userRepository := NewUserRepository(
		db,
		config.UsersSalt,
		config.EmailSalt,
		config.Languages,
		emailSender,
		gocloak.NewClient(config.KeyCloak.URL),
		config.KeyCloak,
	)

	lis = bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	usersSrv := NewUsersGRPCServer(
		config.OAuth,
		db,
		enforcer,
		contactsRepository,
		userRepository,
		NewEvents(db),
		config.Languages,
		false,
		NewPasswordRecovery(
			db,
			false,
			config.Languages,
			emailSender,
		),
	)
	RegisterUsersServer(grpcServer, usersSrv)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestCreateUpdateDeleteUser(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	require.NoError(t, err)
	defer util.Close(conn)
	client := NewUsersClient(conn)

	rand.Seed(time.Now().UnixNano())
	email := "test" + strconv.Itoa(rand.Int()) + "@example.com"

	name := "test"
	newName := "test 2"
	password := "password"

	_, err = client.CreateUser(ctx, &APICreateUserRequest{
		Email:           email,
		Name:            name,
		Password:        password,
		PasswordConfirm: password,
		Language:        "en",
		Captcha:         "",
	})
	require.NoError(t, err)

	config := LoadConfig()

	var userID int64

	db, err := sql.Open("mysql", config.AutowpDSN)
	require.NoError(t, err)
	err = db.QueryRow("SELECT id FROM users WHERE email_to_check = ?", email).Scan(&userID)
	require.NoError(t, err)

	_, err = client.UpdateUser(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+createToken(t, userID, config.OAuth.Secret)),
		&APIUpdateUserRequest{UserId: userID, Name: newName},
	)
	require.NoError(t, err)

	dbNewName := ""
	err = db.QueryRow("SELECT name FROM users WHERE id = ?", userID).Scan(&dbNewName)
	require.NoError(t, err)
	require.Equal(t, newName, dbNewName)

	resp, err := client.DeleteUser(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+createToken(t, adminUserID, config.OAuth.Secret)),
		&APIDeleteUserRequest{UserId: userID, Password: password},
	)
	require.NoError(t, err)

	log.Printf("Response: %+v", resp)
	// Test for output here.
}

func createToken(t *testing.T, userID int64, secret string) string {
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "default",
		"exp": time.Now().Add(time.Minute * 15).Unix(),
		"sub": strconv.FormatInt(userID, 10),
	}).SignedString([]byte(secret))
	require.NoError(t, err)

	return accessToken
}
