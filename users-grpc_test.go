package goautowp

import (
	"context"
	"database/sql"
	"github.com/Nerzal/gocloak/v8"
	"github.com/autowp/goautowp/util"
	"github.com/casbin/casbin"
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

const adminAuthToken = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJkZWZhdWx0IiwiZXhwIjoxODgwMDAwMDAwLCJzdWIiOiIzIn0.5l0HxtAvH9kmfpJXC85lpcEf2EzPucxFCLmXl1oatPwKEDb__YTIdEDaaINplD4oWg10HbOc0-vDJVoQngKn9g"

var lis *bufconn.Listener

func init() {

	config := LoadConfig()

	db, err := sql.Open("mysql", config.AutowpDSN)
	if err != nil {
		panic(err)
	}

	enforcer := casbin.NewEnforcer("model.conf", "policy.csv")

	contactsRepository := NewContactsRepository(db)
	userRepository := NewUserRepository(
		db,
		config.UsersSalt,
		config.EmailSalt,
		config.Languages,
		&MockEmailSender{},
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

func TestDeleteUser(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	require.NoError(t, err)
	defer util.Close(conn)
	client := NewUsersClient(conn)

	rand.Seed(time.Now().UnixNano())
	email := "test" + strconv.Itoa(rand.Int()) + "@example.com"

	password := "password"

	_, err = client.CreateUser(ctx, &APICreateUserRequest{
		Email:           email,
		Name:            "test",
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

	resp, err := client.DeleteUser(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminAuthToken),
		&APIDeleteUserRequest{UserId: userID, Password: password},
	)
	require.NoError(t, err)

	log.Printf("Response: %+v", resp)
	// Test for output here.
}
