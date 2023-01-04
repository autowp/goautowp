package goautowp

import (
	"context"
	"database/sql"
	"testing"

	"google.golang.org/grpc/metadata"

	"github.com/Nerzal/gocloak/v11"
	"github.com/autowp/goautowp/config"

	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func getUserWithCleanHistory(t *testing.T, conn *grpc.ClientConn, cfg config.Config) string {
	t.Helper()

	ctx := context.Background()
	kc := gocloak.NewClient(cfg.Keycloak.URL)

	// tester
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, testUsername, testPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	usersClient := NewUsersClient(conn)

	// tester (me)
	tester, err := usersClient.Me(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	_, err = db.Exec("UPDATE users SET last_message_time = '2000-01-01' WHERE id = ?", tester.Id)
	require.NoError(t, err)

	return token.AccessToken
}

func TestAddEmptyCommentShouldReturnError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewCommentsClient(conn)
	cfg := config.LoadConfig(".")

	token := getUserWithCleanHistory(t, conn, cfg)

	_, err = client.Add(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token),
		&AddCommentRequest{
			ItemId:             1,
			TypeId:             CommentsType_ARTICLES_TYPE_ID,
			Message:            "",
			ModeratorAttention: false,
			ParentId:           0,
			Resolve:            false,
		},
	)
	require.NotNil(t, err)
}

func TestAddComment(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewCommentsClient(conn)
	cfg := config.LoadConfig(".")

	token := getUserWithCleanHistory(t, conn, cfg)

	_, err = client.Add(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token),
		&AddCommentRequest{
			ItemId:             1,
			TypeId:             CommentsType_ARTICLES_TYPE_ID,
			Message:            "Test",
			ModeratorAttention: false,
			ParentId:           0,
			Resolve:            false,
		},
	)
	require.NoError(t, err)
}
